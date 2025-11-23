package persistence

import (
	"errors"
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================
// TransactionManager Integration Tests
// ===========================
//
// 這些測試驗證 TransactionManager 的核心保證：
// 1. 事務隔離：錯誤時回滾，成功時提交
// 2. Panic 處理：panic 時自動回滾
// 3. 多操作原子性：多個操作在同一事務中成功或失敗

// TestRollbackOnError 驗證事務回滾機制
//
// 場景：
// 1. 開啟事務
// 2. 執行操作（Save account）
// 3. 返回錯誤（模擬失敗）
// 4. 驗證事務已回滾（帳戶未保存）
//
// 預期結果：
// - 事務應該回滾
// - 帳戶不應該存在於資料庫中
// - 後續查詢應該返回 ErrAccountNotFound
func TestRollbackOnError_DoesNotCommit(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()
	txManager := NewGORMTransactionManager(db)
	repo := NewPointsAccountRepository(db)

	memberID := points.NewMemberID()

	// Act: 執行一個會失敗的事務
	err := txManager.InTransaction(func(ctx shared.TransactionContext) error {
		// 1. 創建並保存帳戶
		account, _ := points.NewPointsAccount(memberID)
		err := repo.Save(ctx, account)
		require.NoError(t, err, "Save should succeed within transaction")

		// 2. 模擬錯誤 - 事務應該回滾
		return errors.New("simulated error - trigger rollback")
	})

	// Assert: 驗證事務返回錯誤
	require.Error(t, err)
	assert.Equal(t, "simulated error - trigger rollback", err.Error())

	// Assert: 驗證帳戶未保存（回滾成功）
	_, err = repo.FindByMemberID(nil, memberID)
	assert.ErrorIs(t, err, points.ErrAccountNotFound, "account should not exist after rollback")
}

// TestCommitOnSuccess_SavesData 驗證事務提交機制
//
// 場景：
// 1. 開啟事務
// 2. 執行操作（Save account）
// 3. 返回 nil（成功）
// 4. 驗證事務已提交（帳戶已保存）
//
// 預期結果：
// - 事務應該提交
// - 帳戶應該存在於資料庫中
// - 後續查詢應該成功找到帳戶
func TestCommitOnSuccess_SavesData(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()
	txManager := NewGORMTransactionManager(db)
	repo := NewPointsAccountRepository(db)

	memberID := points.NewMemberID()
	var accountID points.AccountID

	// Act: 執行一個成功的事務
	err := txManager.InTransaction(func(ctx shared.TransactionContext) error {
		// 創建並保存帳戶
		account, _ := points.NewPointsAccount(memberID)
		accountID = account.AccountID()
		return repo.Save(ctx, account)
	})

	// Assert: 驗證事務成功
	require.NoError(t, err)

	// Assert: 驗證帳戶已保存（提交成功）
	account, err := repo.FindByMemberID(nil, memberID)
	require.NoError(t, err, "account should exist after commit")
	assert.Equal(t, accountID.String(), account.AccountID().String())
	assert.Equal(t, memberID.String(), account.MemberID().String())
}

// TestPanicRecovery_RollsBackAndRepanics 驗證 panic 處理
//
// 場景：
// 1. 開啟事務
// 2. 執行操作（Save account）
// 3. 觸發 panic
// 4. 驗證事務已回滾
// 5. 驗證 panic 被重新拋出
//
// 預期結果：
// - 事務應該回滾
// - 帳戶不應該存在於資料庫中
// - panic 應該被重新拋出（由調用者處理）
func TestPanicRecovery_RollsBackAndRepanics(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()
	txManager := NewGORMTransactionManager(db)
	repo := NewPointsAccountRepository(db)

	memberID := points.NewMemberID()

	// Act & Assert: 執行會 panic 的事務，並捕獲 panic
	assert.Panics(t, func() {
		_ = txManager.InTransaction(func(ctx shared.TransactionContext) error {
			// 1. 創建並保存帳戶
			account, _ := points.NewPointsAccount(memberID)
			err := repo.Save(ctx, account)
			require.NoError(t, err, "Save should succeed within transaction")

			// 2. 觸發 panic
			panic("simulated panic - should rollback")
		})
	}, "panic should be re-thrown")

	// Assert: 驗證帳戶未保存（回滾成功）
	_, err := repo.FindByMemberID(nil, memberID)
	assert.ErrorIs(t, err, points.ErrAccountNotFound, "account should not exist after panic rollback")
}

// TestMultipleOperations_AtomicCommit 驗證多操作原子性
//
// 場景：
// 1. 開啟事務
// 2. 執行多個操作（Save 兩個帳戶）
// 3. 驗證兩個操作都成功或都失敗
//
// 預期結果：
// - 兩個帳戶都應該保存成功
// - 提交後兩個帳戶都應該存在
func TestMultipleOperations_AtomicCommit(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()
	txManager := NewGORMTransactionManager(db)
	repo := NewPointsAccountRepository(db)

	memberID1 := points.NewMemberID()
	memberID2 := points.NewMemberID()

	// Act: 在同一事務中保存兩個帳戶
	err := txManager.InTransaction(func(ctx shared.TransactionContext) error {
		// 保存第一個帳戶
		account1, _ := points.NewPointsAccount(memberID1)
		if err := repo.Save(ctx, account1); err != nil {
			return err
		}

		// 保存第二個帳戶
		account2, _ := points.NewPointsAccount(memberID2)
		if err := repo.Save(ctx, account2); err != nil {
			return err
		}

		return nil
	})

	// Assert: 驗證事務成功
	require.NoError(t, err)

	// Assert: 驗證兩個帳戶都存在
	account1, err := repo.FindByMemberID(nil, memberID1)
	require.NoError(t, err, "account1 should exist")
	assert.Equal(t, memberID1.String(), account1.MemberID().String())

	account2, err := repo.FindByMemberID(nil, memberID2)
	require.NoError(t, err, "account2 should exist")
	assert.Equal(t, memberID2.String(), account2.MemberID().String())
}

// TestMultipleOperations_AtomicRollback 驗證多操作原子回滾
//
// 場景：
// 1. 開啟事務
// 2. 執行第一個操作（Save account1）成功
// 3. 執行第二個操作（Save account2）失敗
// 4. 驗證兩個操作都被回滾
//
// 預期結果：
// - 第一個帳戶不應該存在（即使 Save 成功）
// - 第二個帳戶不應該存在
// - 事務整體回滾
func TestMultipleOperations_AtomicRollback(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()
	txManager := NewGORMTransactionManager(db)
	repo := NewPointsAccountRepository(db)

	memberID1 := points.NewMemberID()
	memberID2 := points.NewMemberID()

	// Act: 在同一事務中，第一個操作成功，第二個操作失敗
	err := txManager.InTransaction(func(ctx shared.TransactionContext) error {
		// 保存第一個帳戶（成功）
		account1, _ := points.NewPointsAccount(memberID1)
		if err := repo.Save(ctx, account1); err != nil {
			return err
		}

		// 保存第二個帳戶（成功）
		account2, _ := points.NewPointsAccount(memberID2)
		if err := repo.Save(ctx, account2); err != nil {
			return err
		}

		// 模擬後續操作失敗
		return errors.New("second operation failed")
	})

	// Assert: 驗證事務失敗
	require.Error(t, err)

	// Assert: 驗證兩個帳戶都不存在（原子回滾）
	_, err = repo.FindByMemberID(nil, memberID1)
	assert.ErrorIs(t, err, points.ErrAccountNotFound, "account1 should not exist after rollback")

	_, err = repo.FindByMemberID(nil, memberID2)
	assert.ErrorIs(t, err, points.ErrAccountNotFound, "account2 should not exist after rollback")
}

// TestRepository_NilContext_AutoCommitMode 驗證 nil context 的 auto-commit 行為
//
// 場景：
// 1. 不使用 TransactionManager
// 2. 直接調用 Repository 方法，傳入 nil context
// 3. 驗證讀操作可以正常工作（auto-commit 模式）
//
// 預期結果：
// - 傳入 nil context 的讀操作應該成功
// - 驗證 auto-commit 模式下的獨立查詢行為
//
// 注意：
// - 這個測試驗證了 TransactionContext 文檔中的 "ctx == nil" 語義
// - 證明讀操作不強制要求事務參與
func TestRepository_NilContext_AutoCommitMode(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPointsAccountRepository(db)

	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// 先在事務中保存一個帳戶（為後續查詢準備數據）
	txManager := NewGORMTransactionManager(db)
	err := txManager.InTransaction(func(ctx shared.TransactionContext) error {
		return repo.Save(ctx, account)
	})
	require.NoError(t, err, "setup: save account should succeed")

	// Act: 使用 nil context 進行查詢（auto-commit 模式）
	foundAccount, err := repo.FindByMemberID(nil, memberID)

	// Assert: 驗證查詢成功
	require.NoError(t, err, "FindByMemberID with nil context should succeed")
	assert.NotNil(t, foundAccount)
	assert.Equal(t, memberID.String(), foundAccount.MemberID().String())
	assert.Equal(t, account.AccountID().String(), foundAccount.AccountID().String())
}

// TestRepository_NilContext_MultipleReads 驗證 nil context 下的多次讀取
//
// 場景：
// 1. 保存兩個帳戶
// 2. 使用 nil context 進行多次獨立查詢
// 3. 驗證每次查詢都是獨立的（不在同一事務中）
//
// 預期結果：
// - 所有查詢都應該成功
// - 每次查詢都是獨立的 auto-commit 操作
func TestRepository_NilContext_MultipleReads(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPointsAccountRepository(db)
	txManager := NewGORMTransactionManager(db)

	memberID1 := points.NewMemberID()
	memberID2 := points.NewMemberID()

	// 保存兩個帳戶
	err := txManager.InTransaction(func(ctx shared.TransactionContext) error {
		account1, _ := points.NewPointsAccount(memberID1)
		if err := repo.Save(ctx, account1); err != nil {
			return err
		}

		account2, _ := points.NewPointsAccount(memberID2)
		return repo.Save(ctx, account2)
	})
	require.NoError(t, err)

	// Act: 使用 nil context 進行多次獨立查詢
	account1, err1 := repo.FindByMemberID(nil, memberID1)
	account2, err2 := repo.FindByMemberID(nil, memberID2)

	// Assert: 驗證兩次查詢都成功
	require.NoError(t, err1, "first query should succeed")
	require.NoError(t, err2, "second query should succeed")
	assert.Equal(t, memberID1.String(), account1.MemberID().String())
	assert.Equal(t, memberID2.String(), account2.MemberID().String())
}
