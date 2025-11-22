package persistence

import (
	"errors"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"gorm.io/gorm"
)

// ===========================
// GORM PointsAccountRepository 實作
// ===========================

// GORMPointsAccountRepository GORM 實作的積分帳戶倉儲
//
// 設計原則：
// 1. 單一職責：只負責 Domain ↔ GORM 的轉換和錯誤映射
// 2. 依賴倒置：實作 Domain Layer 定義的介面
// 3. 錯誤封裝：將 GORM 錯誤映射為 DomainError
// 4. 事務支持：通過 TransactionContext 管理事務
//
// 職責：
// - 調用 Mapper (toDomain, toGORM) 進行轉換
// - 執行 GORM 操作 (Create, First, Save, Delete)
// - 映射 GORM 錯誤到 Domain 錯誤
// - 不包含業務邏輯（業務邏輯在 Domain Layer）
type GORMPointsAccountRepository struct {
	db *gorm.DB
}

// NewPointsAccountRepository 創建 GORM Repository 實例（核心介面）
func NewPointsAccountRepository(db *gorm.DB) points.PointsAccountRepository {
	return &GORMPointsAccountRepository{db: db}
}

// NewPointsAccountAdminRepository 創建 GORM Repository 實例（管理介面）
// 返回：PointsAccountAdminRepository（包含所有操作）
func NewPointsAccountAdminRepository(db *gorm.DB) points.PointsAccountAdminRepository {
	return &GORMPointsAccountRepository{db: db}
}

// ===========================
// Repository 方法實作
// ===========================

// Save 保存新的積分帳戶
//
// 實作細節：
// 1. 調用 toGORM() 轉換 Domain → GORM Model
// 2. 使用 GORM Create() 插入記錄
// 3. 映射錯誤：唯一約束違反 → ErrAccountAlreadyExists
//
// 前置條件：帳戶不存在（MemberID 唯一）
// 後置條件：帳戶已持久化
// 錯誤：ErrAccountAlreadyExists（如果 MemberID 已存在）
func (r *GORMPointsAccountRepository) Save(ctx shared.TransactionContext, account *points.PointsAccount) error {
	// 1. 獲取事務上下文中的 DB
	db := r.getDB(ctx)

	// 2. Domain → GORM 轉換（使用 Mapper）
	model := toGORM(account)

	// 3. 插入記錄
	result := db.Create(model)

	// 4. 錯誤映射
	if result.Error != nil {
		return r.mapError(result.Error)
	}

	return nil
}

// FindByID 根據帳戶 ID 查找積分帳戶
//
// 實作細節：
// 1. 使用 GORM First() 查詢單筆記錄
// 2. 調用 toDomain() 轉換 GORM Model → Domain
// 3. 映射錯誤：RecordNotFound → ErrAccountNotFound
//
// 返回：找到的帳戶，或 ErrAccountNotFound
func (r *GORMPointsAccountRepository) FindByID(ctx shared.TransactionContext, accountID points.AccountID) (*points.PointsAccount, error) {
	// 1. 獲取事務上下文中的 DB
	db := r.getDB(ctx)

	// 2. 查詢記錄
	var model PointsAccountModel
	result := db.First(&model, "id = ?", accountID.String())

	// 3. 錯誤映射
	if result.Error != nil {
		return nil, r.mapError(result.Error)
	}

	// 4. GORM Model → Domain 轉換（使用 Mapper）
	return toDomain(&model)
}

// FindByMemberID 根據會員 ID 查找積分帳戶
//
// 實作細節：
// 1. 使用 GORM Where().First() 查詢單筆記錄
// 2. 調用 toDomain() 轉換
// 3. 映射錯誤
//
// 業務規則：一個會員對應一個積分帳戶（1:1 關係）
// 返回：找到的帳戶，或 ErrAccountNotFound
func (r *GORMPointsAccountRepository) FindByMemberID(ctx shared.TransactionContext, memberID points.MemberID) (*points.PointsAccount, error) {
	// 1. 獲取事務上下文中的 DB
	db := r.getDB(ctx)

	// 2. 查詢記錄（使用 MemberID 唯一索引）
	var model PointsAccountModel
	result := db.Where("member_id = ?", memberID.String()).First(&model)

	// 3. 錯誤映射
	if result.Error != nil {
		return nil, r.mapError(result.Error)
	}

	// 4. GORM Model → Domain 轉換
	return toDomain(&model)
}

// Update 更新積分帳戶
//
// 實作細節：
// 1. 調用 toGORM() 轉換 Domain → GORM Model
// 2. 使用 GORM Updates() 更新記錄（WHERE 條件確保只更新存在的記錄）
// 3. 檢查 RowsAffected：如果為 0 表示記錄不存在
// 4. 映射錯誤
//
// 前置條件：帳戶已存在
// 後置條件：帳戶狀態已更新
// 錯誤：ErrAccountNotFound（如果帳戶不存在）
//
// 設計原則：
// - 單一職責：一個查詢完成更新和存在性檢查
// - 效能優化：避免額外的 Count 查詢
func (r *GORMPointsAccountRepository) Update(ctx shared.TransactionContext, account *points.PointsAccount) error {
	// 1. 獲取事務上下文中的 DB
	db := r.getDB(ctx)

	// 2. Domain → GORM 轉換
	model := toGORM(account)

	// 3. 更新記錄（WHERE 確保只更新存在的記錄）
	result := db.Model(&PointsAccountModel{}).
		Where("id = ?", model.ID).
		Updates(model)

	// 4. 錯誤檢查
	if result.Error != nil {
		return r.mapError(result.Error)
	}

	// 5. 檢查是否真的更新了記錄
	// RowsAffected = 0 表示記錄不存在（WHERE 條件未匹配）
	if result.RowsAffected == 0 {
		return points.ErrAccountNotFound.WithContext(
			"account_id", account.AccountID().String(),
			"reason", "account does not exist in database",
		)
	}

	return nil
}

// ===========================
// 未來實作方法（TODO）
// ===========================
//
// 以下方法計劃在未來版本實作：
//
// 1. FindPaginated - 分頁查詢，替代 FindAll
//    func (r *GORMPointsAccountRepository) FindPaginated(
//        ctx shared.TransactionContext,
//        page, pageSize int,
//    ) (PagedResult[*points.PointsAccount], error)
//
// 2. Iterate - 迭代器模式，用於大批次處理
//    func (r *GORMPointsAccountRepository) Iterate(
//        ctx shared.TransactionContext,
//        batchSize int,
//    ) (points.AccountIterator, error)
//
// 3. MarkInactive - 狀態管理，替代 Delete
//    func (r *GORMPointsAccountRepository) MarkInactive(
//        ctx shared.TransactionContext,
//        accountID points.AccountID,
//        reason string,
//    ) error
//
// 4. PermanentlyDelete - 物理刪除（需要審計）
//    func (r *GORMPointsAccountRepository) PermanentlyDelete(
//        ctx shared.TransactionContext,
//        accountID points.AccountID,
//        auditInfo AuditInfo,
//    ) error

// ===========================
// 私有輔助方法
// ===========================

// getDB 從 TransactionContext 獲取 GORM DB
//
// 設計原則：
// - TransactionContext 是介面（Domain Layer 定義）
// - gormTransactionContext 是具體實作（Infrastructure Layer）
// - 這裡進行類型斷言，獲取內部的 *gorm.DB
//
// 如果 ctx 是事務上下文，返回事務 DB
// 如果 ctx 是普通上下文，返回普通 DB
func (r *GORMPointsAccountRepository) getDB(ctx shared.TransactionContext) *gorm.DB {
	// 類型斷言：TransactionContext → gormTransactionContext
	if gormCtx, ok := ctx.(*gormTransactionContext); ok {
		return gormCtx.GetDB()
	}

	// 如果不是 gormTransactionContext，使用默認 DB
	return r.db
}

// mapError 映射 GORM 錯誤到 Domain 錯誤
//
// 映射規則：
// - gorm.ErrRecordNotFound       → points.ErrAccountNotFound
// - gorm.ErrDuplicatedKey        → points.ErrAccountAlreadyExists
// - Unique constraint violation  → points.ErrAccountAlreadyExists
// - 其他錯誤                      → points.ErrRepositoryError
//
// 設計原則：
// - 防止 GORM 錯誤洩漏到 Domain Layer
// - 統一錯誤格式（DomainError）
// - 保留原始錯誤上下文（方便排查）
//
// 支援的資料庫：
// - SQLite 3.x（主要測試環境）
// - PostgreSQL 12+（計劃支援）
//
// 已知限制：
// - 唯一約束檢測使用字符串匹配（依賴英文錯誤訊息）
// - 不支援資料庫錯誤訊息本地化
// - 未來版本將改用資料庫驅動錯誤碼（如 PostgreSQL 的 sqlstate）
//
// TODO: 重構為策略模式，支援多資料庫
// - SQLiteErrorMapper: 處理 SQLite 特定錯誤
// - PostgreSQLErrorMapper: 處理 PostgreSQL 特定錯誤（使用 pq.Error.Code）
func (r *GORMPointsAccountRepository) mapError(err error) error {
	// 1. RecordNotFound → ErrAccountNotFound
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return points.ErrAccountNotFound
	}

	// 2. DuplicatedKey → ErrAccountAlreadyExists
	// 注意：GORM 的 ErrDuplicatedKey 在某些版本可能不存在
	// 也需要檢查資料庫原生錯誤（如 SQLite 的 UNIQUE constraint failed）
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return points.ErrAccountAlreadyExists
	}

	// 3. 檢查錯誤訊息中是否包含唯一約束違反（資料庫層級錯誤）
	// SQLite: "UNIQUE constraint failed"
	// PostgreSQL: "duplicate key value violates unique constraint"
	errMsg := err.Error()
	if containsAny(errMsg, []string{"UNIQUE constraint", "duplicate key", "Duplicate entry"}) {
		return points.ErrAccountAlreadyExists.WithContext(
			"database_error", errMsg,
		)
	}

	// 4. 其他錯誤 → ErrRepositoryError
	return points.ErrRepositoryError.WithContext(
		"database_error", err.Error(),
	)
}

// containsAny 檢查字串是否包含任一子字串
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains 檢查字串是否包含子字串（簡單實作）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

// indexOf 返回子字串在字串中的位置（簡單實作）
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
