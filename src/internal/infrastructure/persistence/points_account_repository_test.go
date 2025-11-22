package persistence

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ===========================
// Repository 整合測試（重構後）
// ===========================
// 測試重點：
// 1. 錯誤映射（GORM errors → Domain errors）
// 2. 事務處理（提交和回滾）
// 3. 我們的代碼邏輯，而非 GORM 的功能
// ===========================

// ===========================
// Test Group 1: 錯誤映射測試
// ===========================

// Test 1: FindByID NotFound - 驗證 GORM 錯誤映射到 Domain 錯誤
func TestGORMRepository_FindByID_NotFound_MapsToErrAccountNotFound(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPointsAccountRepository(db)
	ctx := NewGORMTransactionContext(db)

	nonExistentID := points.NewAccountID()

	// Act
	account, err := repo.FindByID(ctx, nonExistentID)

	// Assert - 驗證錯誤映射是我們的代碼，而非 GORM
	assert.Nil(t, account)
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrAccountNotFound)

	// 驗證這是 DomainError，而非 gorm.ErrRecordNotFound
	assert.NotErrorIs(t, err, gorm.ErrRecordNotFound)

	// 驗證錯誤上下文
	var domainErr *points.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, points.ErrCodeAccountNotFound, domainErr.Code)
}

// Test 2: FindByMemberID NotFound - 驗證錯誤映射
func TestGORMRepository_FindByMemberID_NotFound_MapsToErrAccountNotFound(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPointsAccountRepository(db)
	ctx := NewGORMTransactionContext(db)

	nonExistentMemberID := points.NewMemberID()

	// Act
	account, err := repo.FindByMemberID(ctx, nonExistentMemberID)

	// Assert
	assert.Nil(t, account)
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrAccountNotFound)

	// 驗證不是 GORM 的錯誤
	assert.NotErrorIs(t, err, gorm.ErrRecordNotFound)
}

// Test 3: Save Duplicate - 驗證唯一約束違反映射到 Domain 錯誤
func TestGORMRepository_Save_DuplicateMemberID_MapsToErrAccountAlreadyExists(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPointsAccountRepository(db)
	ctx := NewGORMTransactionContext(db)

	memberID := points.NewMemberID()
	account1, _ := points.NewPointsAccount(memberID)

	// 先保存一次
	err := repo.Save(ctx, account1)
	require.NoError(t, err)

	// Act - 嘗試用相同 MemberID 創建第二個帳戶
	account2, _ := points.NewPointsAccount(memberID)
	err = repo.Save(ctx, account2)

	// Assert - 驗證錯誤映射
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrAccountAlreadyExists)

	// 驗證是 DomainError
	var domainErr *points.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, points.ErrCodeAccountAlreadyExists, domainErr.Code)
}

// Test 4: Update NotFound - 驗證更新不存在的帳戶
func TestGORMRepository_Update_NotFound_MapsToErrAccountNotFound(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPointsAccountRepository(db)
	ctx := NewGORMTransactionContext(db)

	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)
	// 注意：沒有調用 Save，直接 Update

	// Act - 嘗試更新未保存的帳戶
	err := repo.Update(ctx, account)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrAccountNotFound)
}

// ===========================
// Test Group 2: 事務處理測試
// ===========================

// Test 5: 事務提交 - 驗證 TransactionContext 正確處理提交
func TestGORMRepository_WithinTransaction_CommitsChangesCorrectly(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPointsAccountRepository(db)

	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// Act - 在事務中保存並更新
	err := db.Transaction(func(tx *gorm.DB) error {
		txCtx := NewGORMTransactionContext(tx)

		// 保存帳戶
		if err := repo.Save(txCtx, account); err != nil {
			return err
		}

		// 修改帳戶
		earned, _ := points.NewPointsAmount(100)
		account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "測試")

		// 更新帳戶
		return repo.Update(txCtx, account)
	})

	// Assert - 事務應該成功提交
	require.NoError(t, err)

	// 驗證更改已持久化（在事務外讀取）
	ctx := NewGORMTransactionContext(db)
	found, err := repo.FindByID(ctx, account.AccountID())
	require.NoError(t, err)
	assert.Equal(t, 100, found.EarnedPoints().Value())
	assert.Equal(t, 100, found.GetAvailablePoints().Value())
}

// Test 6: 事務回滾 - 驗證失敗時正確回滾
func TestGORMRepository_TransactionRollback_RevertsChanges(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPointsAccountRepository(db)
	ctx := NewGORMTransactionContext(db)

	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// 先保存帳戶（事務外）
	err := repo.Save(ctx, account)
	require.NoError(t, err)

	// Act - 在事務中更新，但事務失敗回滾
	err = db.Transaction(func(tx *gorm.DB) error {
		txCtx := NewGORMTransactionContext(tx)

		// 修改帳戶
		earned, _ := points.NewPointsAmount(100)
		account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "測試")

		// 更新帳戶
		if err := repo.Update(txCtx, account); err != nil {
			return err
		}

		// 模擬業務邏輯失敗
		return fmt.Errorf("business rule violation")
	})

	// Assert - 事務應該失敗
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "business rule violation")

	// 驗證更改未持久化（事務已回滾）
	found, err := repo.FindByID(ctx, account.AccountID())
	require.NoError(t, err)
	assert.Equal(t, 0, found.EarnedPoints().Value(), "積分應該仍為 0（事務已回滾）")
	assert.Equal(t, 0, found.GetAvailablePoints().Value())
}

// ===========================
// Test Group 3: 完整流程測試（保留一個）
// ===========================

// Test 7: 完整 CRUD 流程 - 驗證 Save, Find, Update 的協同工作
func TestGORMRepository_CompleteCRUDFlow_WorksCorrectly(t *testing.T) {
	// Arrange
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewPointsAccountRepository(db)
	ctx := NewGORMTransactionContext(db)

	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// Act & Assert - Step 1: Save
	err := repo.Save(ctx, account)
	require.NoError(t, err, "Save 應該成功")

	// Act & Assert - Step 2: FindByID
	found, err := repo.FindByID(ctx, account.AccountID())
	require.NoError(t, err, "FindByID 應該成功")
	assert.Equal(t, account.AccountID().String(), found.AccountID().String())

	// Act & Assert - Step 3: FindByMemberID
	foundByMember, err := repo.FindByMemberID(ctx, memberID)
	require.NoError(t, err, "FindByMemberID 應該成功")
	assert.Equal(t, account.AccountID().String(), foundByMember.AccountID().String())

	// Act & Assert - Step 4: Update
	earned, _ := points.NewPointsAmount(200)
	found.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "測試")

	err = repo.Update(ctx, found)
	require.NoError(t, err, "Update 應該成功")

	// Act & Assert - Step 5: 驗證更新已持久化
	updated, err := repo.FindByID(ctx, account.AccountID())
	require.NoError(t, err)
	assert.Equal(t, 200, updated.EarnedPoints().Value())
}
