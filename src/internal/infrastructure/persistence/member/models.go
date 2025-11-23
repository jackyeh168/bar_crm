package member

import (
	"time"

	"github.com/jackyeh168/bar_crm/src/internal/domain/member"
	"gorm.io/gorm"
)

// ===========================
// GORM Models
// ===========================

// MemberGORM 會員資料表模型
//
// 設計原則：
// - 僅用於 Infrastructure Layer（不暴露給 Domain Layer）
// - 使用 GORM 標籤定義資料庫結構
// - 與 Domain Member 聚合分離（Mapper 轉換）
//
// 資料庫約束：
// - member_id: 主鍵（UUID）
// - line_user_id: 唯一索引（防止重複註冊）
// - phone_number: 唯一索引（防止重複綁定），可為空
// - display_name: 不可為空
type MemberGORM struct {
	// 識別欄位
	MemberID   string `gorm:"column:member_id;type:varchar(36);primaryKey"` // UUID 字串
	LineUserID string `gorm:"column:line_user_id;type:varchar(33);uniqueIndex;not null"`

	// 基本信息
	DisplayName string `gorm:"column:display_name;type:varchar(255);not null"`

	// 綁定信息
	PhoneNumber *string `gorm:"column:phone_number;type:varchar(10);uniqueIndex"` // Nullable

	// 審計欄位
	CreatedAt time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null"`
	Version   int            `gorm:"column:version;not null;default:1"` // 樂觀鎖
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`           // 軟刪除
}

// TableName 指定資料表名稱
func (MemberGORM) TableName() string {
	return "members"
}

// ===========================
// Mapper Functions
// ===========================

// toDomain 將 GORM 模型轉換為 Domain 模型
//
// 參數：
// - m: GORM 模型
//
// 返回：
// - *member.Member: Domain 聚合
// - error: 轉換失敗時返回錯誤
//
// 轉換邏輯：
// - MemberID: 字串 → MemberID 值對象
// - LineUserID: 字串 → LineUserID 值對象
// - PhoneNumber: *string → PhoneNumber 值對象（處理 NULL）
func (m *MemberGORM) toDomain() (*member.Member, error) {
	// 1. 轉換 MemberID
	memberID, err := member.MemberIDFromString(m.MemberID)
	if err != nil {
		return nil, err
	}

	// 2. 轉換 LineUserID
	lineUserID, err := member.NewLineUserID(m.LineUserID)
	if err != nil {
		return nil, err
	}

	// 3. 轉換 PhoneNumber（處理 NULL）
	var phoneNumber member.PhoneNumber
	if m.PhoneNumber != nil {
		phoneNumber, err = member.NewPhoneNumber(*m.PhoneNumber)
		if err != nil {
			return nil, err
		}
	}
	// 如果 m.PhoneNumber == nil，phoneNumber 維持零值

	// 4. 重建 Domain 聚合
	return member.ReconstructMember(
		memberID,
		lineUserID,
		m.DisplayName,
		phoneNumber,
		m.CreatedAt,
		m.UpdatedAt,
		m.Version,
	)
}

// toGORM 將 Domain 模型轉換為 GORM 模型
//
// 參數：
// - m: Domain 聚合
//
// 返回：
// - *MemberGORM: GORM 模型
//
// 轉換邏輯：
// - MemberID: MemberID 值對象 → 字串
// - LineUserID: LineUserID 值對象 → 字串
// - PhoneNumber: PhoneNumber 值對象 → *string（處理零值 → NULL）
func toGORM(m *member.Member) *MemberGORM {
	// 處理 PhoneNumber（零值 → NULL）
	var phoneNumber *string
	if !m.PhoneNumber().IsZero() {
		phoneStr := m.PhoneNumber().String()
		phoneNumber = &phoneStr
	}

	return &MemberGORM{
		MemberID:    m.MemberID().String(),
		LineUserID:  m.LineUserID().String(),
		DisplayName: m.DisplayName(),
		PhoneNumber: phoneNumber,
		CreatedAt:   m.CreatedAt(),
		UpdatedAt:   m.UpdatedAt(),
		Version:     m.Version(),
	}
}
