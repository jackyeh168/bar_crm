package points

import (
	"errors"
	"strings"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"gorm.io/gorm"
)

// gormTransactionContext GORM 事務上下文
type gormTransactionContext interface {
	shared.TransactionContext
	GetDB() *gorm.DB
}

// ===========================
// PointsAccountRepositoryImpl
// ===========================

// PointsAccountRepositoryImpl 積分帳戶倉儲實現（GORM）
//
// 設計原則：
// - 實作 points.PointsAccountRepository 接口
// - 處理 Domain 與 GORM 模型轉換
// - 封裝所有資料庫操作細節
// - 將 GORM 錯誤轉換為 Domain 錯誤
//
// 依賴：
// - *gorm.DB: GORM 資料庫實例（由 DI 容器注入）
type PointsAccountRepositoryImpl struct {
	db *gorm.DB
}

// NewPointsAccountRepository 創建新的積分帳戶倉儲實例
//
// 參數：
//   - db: GORM 資料庫實例
//
// 返回：
//   - points.PointsAccountRepository: 倉儲接口實例
func NewPointsAccountRepository(db *gorm.DB) points.PointsAccountRepository {
	return &PointsAccountRepositoryImpl{db: db}
}

// Save 保存新的積分帳戶
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 將 Domain 模型轉換為 GORM 模型
// 3. 使用 GORM Create（新增記錄）
// 4. 處理唯一約束衝突錯誤
//
// 錯誤處理：
// - UNIQUE constraint 違反（member_id 重複）→ ErrAccountAlreadyExists
// - 其他資料庫錯誤 → 原始錯誤
func (r *PointsAccountRepositoryImpl) Save(ctx shared.TransactionContext, account *points.PointsAccount) error {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	// 2. 轉換為 GORM 模型
	gormModel := toGORM(account)

	// 3. 執行 Create
	result := db.Create(gormModel)
	if result.Error != nil {
		// 4. 處理唯一約束錯誤
		if isUniqueConstraintError(result.Error) {
			return points.ErrAccountAlreadyExists.WithContext(
				"member_id", account.MemberID().String(),
			)
		}
		return result.Error
	}

	return nil
}

// FindByID 根據帳戶 ID 查找積分帳戶
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 使用 GORM Where + First 查詢
// 3. 將 GORM 模型轉換為 Domain 模型
// 4. 處理 Not Found 錯誤
//
// 錯誤處理：
// - gorm.ErrRecordNotFound → points.ErrAccountNotFound
// - 其他資料庫錯誤 → 原始錯誤
func (r *PointsAccountRepositoryImpl) FindByID(ctx shared.TransactionContext, accountID points.AccountID) (*points.PointsAccount, error) {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	var gormModel PointsAccountGORM

	// 2. 查詢資料庫
	result := db.Where("account_id = ?", accountID.String()).First(&gormModel)
	if result.Error != nil {
		// 3. 處理 Not Found
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, points.ErrAccountNotFound.WithContext(
				"account_id", accountID.String(),
			)
		}
		return nil, result.Error
	}

	// 4. 轉換為 Domain 模型
	return gormModel.toDomain()
}

// FindByMemberID 根據會員 ID 查找積分帳戶
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 使用 GORM Where + First 查詢
// 3. 將 GORM 模型轉換為 Domain 模型
// 4. 處理 Not Found 錯誤
//
// 業務規則：一個會員對應一個積分帳戶（1:1 關係，由 unique index 保證）
//
// 錯誤處理：
// - gorm.ErrRecordNotFound → points.ErrAccountNotFound
// - 其他資料庫錯誤 → 原始錯誤
func (r *PointsAccountRepositoryImpl) FindByMemberID(ctx shared.TransactionContext, memberID points.MemberID) (*points.PointsAccount, error) {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	var gormModel PointsAccountGORM

	// 2. 查詢資料庫
	result := db.Where("member_id = ?", memberID.String()).First(&gormModel)
	if result.Error != nil {
		// 3. 處理 Not Found
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, points.ErrAccountNotFound.WithContext(
				"member_id", memberID.String(),
			)
		}
		return nil, result.Error
	}

	// 4. 轉換為 Domain 模型
	return gormModel.toDomain()
}

// Update 更新積分帳戶
//
// 實作邏輯：
// 1. 從 TransactionContext 獲取 DB 實例
// 2. 將 Domain 模型轉換為 GORM 模型
// 3. 使用 GORM Save (Upsert: 存在則更新，不存在則新增)
// 4. 處理唯一約束衝突錯誤
//
// 注意：使用 Save 而非 Updates，因為：
// - Save 會更新所有字段（包括零值）
// - Updates 會忽略零值字段
// - 積分數據可能降為 0，需要正確更新
//
// 錯誤處理：
// - 帳戶不存在時也會執行 Save（變成新增）
// - UNIQUE constraint 違反 → ErrAccountAlreadyExists
// - 其他資料庫錯誤 → 原始錯誤
func (r *PointsAccountRepositoryImpl) Update(ctx shared.TransactionContext, account *points.PointsAccount) error {
	// 1. 獲取 DB 實例
	db := r.getDB(ctx)

	// 2. 轉換為 GORM 模型
	gormModel := toGORM(account)

	// 3. 執行 Save (Upsert)
	result := db.Save(gormModel)
	if result.Error != nil {
		// 4. 處理唯一約束錯誤
		if isUniqueConstraintError(result.Error) {
			return points.ErrAccountAlreadyExists.WithContext(
				"account_id", account.AccountID().String(),
			)
		}
		return result.Error
	}

	return nil
}

// ===========================
// Helper Methods
// ===========================

// getDB 獲取 GORM DB 實例
//
// 參數：
//   - ctx: 事務上下文（可為 nil）
//
// 返回：
//   - *gorm.DB: GORM 資料庫實例
//
// 行為：
//   - ctx != nil: 使用事務中的 DB（從 TransactionContext 獲取）
//   - ctx == nil: 使用預設 DB（auto-commit 模式）
func (r *PointsAccountRepositoryImpl) getDB(ctx shared.TransactionContext) *gorm.DB {
	if ctx != nil {
		if txCtx, ok := ctx.(gormTransactionContext); ok {
			return txCtx.GetDB()
		}
	}
	return r.db
}

// isUniqueConstraintError 判斷是否為唯一約束錯誤
//
// 支持的資料庫：
// - PostgreSQL: "duplicate key value violates unique constraint"
// - SQLite: "UNIQUE constraint failed"
// - MySQL: "Duplicate entry"
//
// 參數：
//   - err: GORM 錯誤
//
// 返回：
//   - bool: 是否為唯一約束錯誤
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// PostgreSQL
	if strings.Contains(errMsg, "duplicate key value violates unique constraint") {
		return true
	}

	// SQLite
	if strings.Contains(errMsg, "unique constraint failed") {
		return true
	}

	// MySQL
	if strings.Contains(errMsg, "duplicate entry") {
		return true
	}

	return false
}
