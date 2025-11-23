package persistence

import (
	"fmt"
	"log"

	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"gorm.io/gorm"
)

// ===========================
// GORM TransactionContext 實作
// ===========================

// gormTransactionContext GORM 事務上下文實作
// 設計原則：
// 1. 實作 shared.TransactionContext 介面（標記介面）
// 2. 封裝 *gorm.DB，避免洩漏到 Domain Layer
// 3. 提供 GetDB() 方法供 Infrastructure Layer 內部使用
type gormTransactionContext struct {
	db *gorm.DB
}

// NewGORMTransactionContext 創建 GORM 事務上下文
// 參數：
// - db: GORM 資料庫連接
// 返回：
// - shared.TransactionContext: 事務上下文介面
func NewGORMTransactionContext(db *gorm.DB) shared.TransactionContext {
	return &gormTransactionContext{db: db}
}

// GetDB 獲取 GORM DB 連接（僅供 Infrastructure Layer 內部使用）
// 注意：這個方法不在 shared.TransactionContext 介面中
// 這樣 Domain Layer 無法訪問 GORM，保持依賴方向正確
func (ctx *gormTransactionContext) GetDB() *gorm.DB {
	return ctx.db
}

// ===========================
// GORM TransactionManager 實作
// ===========================

// gormTransactionManager GORM 事務管理器實作
// 設計原則：
// 1. 實作 shared.TransactionManager 介面
// 2. 提供事務管理能力（Begin, Commit, Rollback）
// 3. 自動處理 panic 和錯誤回滾
type gormTransactionManager struct {
	db *gorm.DB
}

// NewGORMTransactionManager 創建 GORM 事務管理器
// 參數：
// - db: GORM 資料庫連接
// 返回：
// - shared.TransactionManager: 事務管理器介面
func NewGORMTransactionManager(db *gorm.DB) shared.TransactionManager {
	return &gormTransactionManager{db: db}
}

// InTransaction 在事務中執行函數
// 設計原則：
// 1. 自動開啟事務（Begin）
// 2. 函數返回 nil → Commit
// 3. 函數返回 error → Rollback
// 4. 函數 panic → Rollback + re-panic
//
// 使用範例：
//   err := txManager.InTransaction(func(ctx shared.TransactionContext) error {
//       account, _ := repo.FindByID(ctx, accountID)
//       account.EarnPoints(amount, source, sourceID, description)
//       return repo.Update(ctx, account)
//   })
func (tm *gormTransactionManager) InTransaction(fn func(ctx shared.TransactionContext) error) error {
	// 1. 開啟事務
	tx := tm.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 2. 創建事務上下文
	ctx := NewGORMTransactionContext(tx)

	// 3. 使用 defer 處理 panic 和回滾
	defer func() {
		if r := recover(); r != nil {
			// Panic 發生，回滾事務
			if rbErr := tx.Rollback().Error; rbErr != nil {
				// 記錄回滾失敗（但仍然 panic 原始錯誤）
				// 注意：這種情況很罕見，通常表示資料庫連接問題
				log.Printf("[CRITICAL] Failed to rollback transaction after panic: %v (original panic: %v)", rbErr, r)
			}
			// 重新 panic，讓上層處理
			panic(r)
		}
	}()

	// 4. 執行函數
	err := fn(ctx)
	if err != nil {
		// 函數返回錯誤，回滾事務
		if rbErr := tx.Rollback().Error; rbErr != nil {
			// 記錄回滾失敗，並返回組合錯誤
			log.Printf("[ERROR] Failed to rollback transaction: %v (original error: %v)", rbErr, err)
			return fmt.Errorf("transaction rollback failed: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	// 5. 提交事務
	return tx.Commit().Error
}
