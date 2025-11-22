package persistence

import (
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
