package member

import (
	"errors"

	"github.com/jackyeh168/bar_crm/src/internal/domain/member"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"gorm.io/gorm"
)

// gormTransactionContext GORM事務上下文（來自persistence package）
type gormTransactionContext interface {
	shared.TransactionContext
	GetDB() *gorm.DB
}

// ===========================
// MemberRepositoryImpl
// ===========================

// MemberRepositoryImpl 會員倉儲實現（GORM）
//
// 設計原則：
// - 實作 member.MemberRepository 接口
// - 處理 Domain 與 GORM 模型轉換
// - 封裝所有資料庫操作細節
// - 將 GORM 錯誤轉換為 Domain 錯誤
//
// 依賴：
// - *gorm.DB: GORM 資料庫實例（由 DI 容器注入）
type MemberRepositoryImpl struct {
	db *gorm.DB
}

// NewMemberRepository 創建新的會員倉儲實例
//
// 參數：
// - db: GORM 資料庫實例
//
// 返回：
// - member.MemberRepository: 倉儲接口實例
func NewMemberRepository(db *gorm.DB) member.MemberRepository {
	return &MemberRepositoryImpl{db: db}
}

// Save 保存會員（Upsert 模式）
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 將 Domain 模型轉換為 GORM 模型
// 3. 使用 GORM Save (Upsert: 存在則更新，不存在則新增)
// 4. 處理唯一約束衝突錯誤
//
// 錯誤處理：
// - UNIQUE constraint 違反 → ErrPhoneNumberAlreadyBound
// - 其他資料庫錯誤 → 原始錯誤
func (r *MemberRepositoryImpl) Save(ctx shared.TransactionContext, m *member.Member) error {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	// 2. 轉換為 GORM 模型
	gormModel := toGORM(m)

	// 3. 執行 Upsert
	result := db.Save(gormModel)
	if result.Error != nil {
		// 4. 處理唯一約束錯誤
		if isUniqueConstraintError(result.Error) {
			return member.ErrPhoneNumberAlreadyBound.WithContext(
				"phone_number", m.PhoneNumber().String(),
			)
		}
		return result.Error
	}

	return nil
}

// FindByMemberID 根據會員 ID 查找會員
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 使用 GORM Where + First 查詢
// 3. 將 GORM 模型轉換為 Domain 模型
// 4. 處理 Not Found 錯誤
//
// 錯誤處理：
// - gorm.ErrRecordNotFound → member.ErrMemberNotFound
// - 其他資料庫錯誤 → 原始錯誤
func (r *MemberRepositoryImpl) FindByMemberID(ctx shared.TransactionContext, id member.MemberID) (*member.Member, error) {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	var gormModel MemberGORM

	// 2. 查詢資料庫
	result := db.Where("member_id = ?", id.String()).First(&gormModel)
	if result.Error != nil {
		// 3. 處理 Not Found
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, member.ErrMemberNotFound.WithContext(
				"member_id", id.String(),
			)
		}
		return nil, result.Error
	}

	// 4. 轉換為 Domain 模型
	return gormModel.toDomain()
}

// FindByLineUserID 根據 LINE UserID 查找會員
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 使用 GORM Where + First 查詢
// 3. 將 GORM 模型轉換為 Domain 模型
// 4. 處理 Not Found 錯誤
//
// 錯誤處理：
// - gorm.ErrRecordNotFound → member.ErrMemberNotFound
// - 其他資料庫錯誤 → 原始錯誤
func (r *MemberRepositoryImpl) FindByLineUserID(ctx shared.TransactionContext, lineUserID member.LineUserID) (*member.Member, error) {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	var gormModel MemberGORM

	// 2. 查詢資料庫
	result := db.Where("line_user_id = ?", lineUserID.String()).First(&gormModel)
	if result.Error != nil {
		// 3. 處理 Not Found
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, member.ErrMemberNotFound.WithContext(
				"line_user_id", lineUserID.String(),
			)
		}
		return nil, result.Error
	}

	// 4. 轉換為 Domain 模型
	return gormModel.toDomain()
}

// ExistsByPhoneNumber 檢查手機號碼是否已被註冊
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 使用 COUNT 查詢（效能優化，不載入完整資料）
// 3. 返回 count > 0
//
// 注意：
// - 只檢查是否存在，不載入完整會員資料
// - 比 Find 更高效
func (r *MemberRepositoryImpl) ExistsByPhoneNumber(ctx shared.TransactionContext, phoneNumber member.PhoneNumber) (bool, error) {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	var count int64

	// 2. COUNT 查詢
	phoneStr := phoneNumber.String()
	result := db.Model(&MemberGORM{}).Where("phone_number = ?", phoneStr).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}

	// 3. 返回是否存在
	return count > 0, nil
}

// ExistsByLineUserID 檢查 LINE UserID 是否已註冊
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 使用 COUNT 查詢（效能優化）
// 3. 返回 count > 0
//
// 注意：
// - 只檢查是否存在，不載入完整會員資料
// - 比 Find 更高效
func (r *MemberRepositoryImpl) ExistsByLineUserID(ctx shared.TransactionContext, lineUserID member.LineUserID) (bool, error) {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	var count int64

	// 2. COUNT 查詢
	result := db.Model(&MemberGORM{}).Where("line_user_id = ?", lineUserID.String()).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}

	// 3. 返回是否存在
	return count > 0, nil
}

// getDB 獲取資料庫實例
//
// 邏輯：
// - 如果 ctx 是 gormTransactionContext，返回事務中的 DB
// - 否則返回預設的 DB（auto-commit 模式）
//
// 這個方法實現了可選事務參與模式：
// - 寫操作：必須在事務中（ctx != nil）
// - 讀操作：可選擇是否參與事務
func (r *MemberRepositoryImpl) getDB(ctx shared.TransactionContext) *gorm.DB {
	// 類型斷言：TransactionContext → gormTransactionContext
	if gormCtx, ok := ctx.(gormTransactionContext); ok {
		return gormCtx.GetDB()
	}

	// 如果不是 gormTransactionContext，使用預設 DB
	return r.db
}

// ===========================
// Helper Functions
// ===========================

// isUniqueConstraintError 檢查是否為唯一約束錯誤
//
// 參數：
// - err: GORM 錯誤
//
// 返回：
// - bool: true 表示唯一約束錯誤
//
// 支援的資料庫：
// - SQLite: "UNIQUE constraint failed"
// - PostgreSQL: "duplicate key value violates unique constraint"
// - MySQL: "Duplicate entry"
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return containsAny(errMsg,
		"UNIQUE constraint failed",           // SQLite
		"duplicate key value",                 // PostgreSQL
		"Duplicate entry",                     // MySQL
		"violates unique constraint",          // PostgreSQL (alternative)
	)
}

// containsAny 檢查字串是否包含任一子字串
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			// Simple substring check (not using strings.Contains to avoid import)
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
