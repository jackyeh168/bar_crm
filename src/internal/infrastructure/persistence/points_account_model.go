package persistence

import (
	"time"

	"gorm.io/gorm"
)

// ===========================
// GORM Model 定義
// ===========================

// PointsAccountModel GORM 積分帳戶模型
type PointsAccountModel struct {
	ID           string         `gorm:"type:uuid;primary_key"`
	MemberID     string         `gorm:"type:uuid;uniqueIndex;not null"`
	EarnedPoints int            `gorm:"not null;default:0"`
	UsedPoints   int            `gorm:"not null;default:0"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (PointsAccountModel) TableName() string {
	return "points_accounts"
}
