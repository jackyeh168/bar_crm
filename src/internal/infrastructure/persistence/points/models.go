package points

import (
	"time"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"gorm.io/gorm"
)

// ===========================
// GORM Models
// ===========================

// PointsAccountGORM 積分帳戶資料表模型
//
// 設計原則：
// - 僅用於 Infrastructure Layer（不暴露給 Domain Layer）
// - 使用 GORM 標籤定義資料庫結構
// - 與 Domain PointsAccount 聚合分離（Mapper 轉換）
//
// 資料庫約束：
// - account_id: 主鍵（UUID）
// - member_id: 唯一索引（一個會員對應一個積分帳戶）
// - earned_points: 累積獲得積分（>= 0）
// - used_points: 累積使用積分（>= 0）
// - 業務不變條件：used_points <= earned_points（在 Application 層保證）
type PointsAccountGORM struct {
	// 識別欄位
	AccountID string `gorm:"column:account_id;type:varchar(36);primaryKey"` // UUID 字串
	MemberID  string `gorm:"column:member_id;type:varchar(36);uniqueIndex;not null"`

	// 積分數據
	EarnedPoints int `gorm:"column:earned_points;not null;default:0;check:earned_points >= 0"`
	UsedPoints   int `gorm:"column:used_points;not null;default:0;check:used_points >= 0"`

	// 審計欄位
	CreatedAt time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"` // 軟刪除
}

// TableName 指定資料表名稱
func (PointsAccountGORM) TableName() string {
	return "points_accounts"
}

// ===========================
// Mapper Functions
// ===========================

// toDomain 將 GORM 模型轉換為 Domain 模型
//
// 參數：
//   - g: GORM 模型
//
// 返回：
//   - *points.PointsAccount: Domain 聚合
//   - error: 轉換失敗時返回錯誤
//
// 轉換邏輯：
//   - AccountID: 字串 → AccountID 值對象
//   - MemberID: 字串 → MemberID 值對象
//   - EarnedPoints: int → PointsAmount 值對象
//   - UsedPoints: int → PointsAmount 值對象
func (g *PointsAccountGORM) toDomain() (*points.PointsAccount, error) {
	// 1. 轉換 AccountID
	accountID, err := points.AccountIDFromString(g.AccountID)
	if err != nil {
		return nil, err
	}

	// 2. 轉換 MemberID
	memberID, err := points.MemberIDFromString(g.MemberID)
	if err != nil {
		return nil, err
	}

	// 3. 重建 Domain 聚合（使用 int，由 ReconstructPointsAccount 內部轉換）
	return points.ReconstructPointsAccount(
		accountID,
		memberID,
		g.EarnedPoints,
		g.UsedPoints,
		g.CreatedAt,
		g.UpdatedAt,
	)
}

// toGORM 將 Domain 模型轉換為 GORM 模型
//
// 參數：
//   - account: Domain 聚合
//
// 返回：
//   - *PointsAccountGORM: GORM 模型
//
// 轉換邏輯：
//   - AccountID: AccountID 值對象 → 字串
//   - MemberID: MemberID 值對象 → 字串
//   - EarnedPoints: PointsAmount 值對象 → int
//   - UsedPoints: PointsAmount 值對象 → int
func toGORM(account *points.PointsAccount) *PointsAccountGORM {
	return &PointsAccountGORM{
		AccountID:    account.AccountID().String(),
		MemberID:     account.MemberID().String(),
		EarnedPoints: account.EarnedPoints().Value(),
		UsedPoints:   account.UsedPoints().Value(),
		CreatedAt:    account.CreatedAt(),
		UpdatedAt:    account.UpdatedAt(),
	}
}
