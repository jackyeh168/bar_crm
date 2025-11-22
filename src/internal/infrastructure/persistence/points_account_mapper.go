package persistence

import (
	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
)

// ===========================
// Domain ↔ GORM Model 轉換函數
// ===========================

// toDomain 將 GORM Model 轉換為 Domain 聚合根
//
// 設計原則：
// 1. 防禦性編程：驗證所有數據，即使來自資料庫
// 2. 使用 ReconstructPointsAccount：重建聚合，不發布事件
// 3. 錯誤傳播：數據損壞時返回 DomainError
//
// 參數：
// - model: GORM 模型
//
// 返回：
// - *points.PointsAccount: 重建的聚合根
// - error: 數據驗證錯誤（ErrCorruptedEarnedPoints, ErrInvariantViolation等）
//
// 錯誤處理：
// - 如果數據庫數據違反業務規則，返回錯誤而非 panic
// - 這允許上層決定如何處理（記錄日誌、告警、數據修復等）
func toDomain(model *PointsAccountModel) (*points.PointsAccount, error) {
	// 1. 轉換 ID（使用 FromString 驗證格式）
	accountID, err := points.AccountIDFromString(model.ID)
	if err != nil {
		return nil, points.ErrInvalidAccountID.WithContext(
			"id", model.ID,
			"reason", "invalid UUID format in database",
		)
	}

	memberID, err := points.MemberIDFromString(model.MemberID)
	if err != nil {
		return nil, points.ErrInvalidMemberID.WithContext(
			"id", model.MemberID,
			"reason", "invalid UUID format in database",
		)
	}

	// 2. 使用 ReconstructPointsAccount 重建聚合
	// 這會執行完整驗證（負數檢查、不變條件檢查）
	account, err := points.ReconstructPointsAccount(
		accountID,
		memberID,
		model.EarnedPoints,
		model.UsedPoints,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		// ReconstructPointsAccount 已經返回適當的 DomainError
		return nil, err
	}

	return account, nil
}

// toGORM 將 Domain 聚合根轉換為 GORM Model
//
// 設計原則：
// 1. 單向轉換：Domain → Infrastructure
// 2. 無驗證：Domain 聚合已保證數據有效性
// 3. 簡單映射：直接提取值對象的值
//
// 參數：
// - account: Domain 聚合根
//
// 返回：
// - *PointsAccountModel: GORM 模型（準備持久化）
//
// 注意：
// - 不處理事件：事件由 Repository 在持久化後發布
// - 時間戳：CreatedAt/UpdatedAt 由 Domain 聚合管理
func toGORM(account *points.PointsAccount) *PointsAccountModel {
	return &PointsAccountModel{
		ID:           account.AccountID().String(),
		MemberID:     account.MemberID().String(),
		EarnedPoints: account.EarnedPoints().Value(),
		UsedPoints:   account.UsedPoints().Value(),
		CreatedAt:    account.CreatedAt(),
		UpdatedAt:    account.UpdatedAt(),
		// DeletedAt 由 GORM 管理（軟刪除）
	}
}
