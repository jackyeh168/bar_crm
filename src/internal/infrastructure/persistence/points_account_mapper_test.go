package persistence

import (
	"testing"
	"time"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================
// Model 轉換測試（TDD Red Phase）
// ===========================

// Test 1: toDomain 成功轉換有效的 GORM Model
func TestToDomain_ValidModel_Success(t *testing.T) {
	// Arrange
	now := time.Now()
	validAccountID := points.NewAccountID()
	validMemberID := points.NewMemberID()

	model := &PointsAccountModel{
		ID:           validAccountID.String(),
		MemberID:     validMemberID.String(),
		EarnedPoints: 100,
		UsedPoints:   30,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Act
	account, err := toDomain(model)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, validAccountID.String(), account.AccountID().String())
	assert.Equal(t, validMemberID.String(), account.MemberID().String())
	assert.Equal(t, 100, account.EarnedPoints().Value())
	assert.Equal(t, 30, account.UsedPoints().Value())
	assert.Equal(t, 70, account.GetAvailablePoints().Value())
	assert.WithinDuration(t, now, account.CreatedAt(), time.Second)
	assert.WithinDuration(t, now, account.UpdatedAt(), time.Second)
}

// Test 2: toDomain 檢測負數 EarnedPoints
func TestToDomain_NegativeEarnedPoints_ReturnsError(t *testing.T) {
	// Arrange
	validAccountID := points.NewAccountID()
	validMemberID := points.NewMemberID()

	model := &PointsAccountModel{
		ID:           validAccountID.String(),
		MemberID:     validMemberID.String(),
		EarnedPoints: -100, // 無效：負數
		UsedPoints:   0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Act
	account, err := toDomain(model)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, points.ErrCorruptedEarnedPoints)
}

// Test 3: toDomain 檢測負數 UsedPoints
func TestToDomain_NegativeUsedPoints_ReturnsError(t *testing.T) {
	// Arrange
	validAccountID := points.NewAccountID()
	validMemberID := points.NewMemberID()

	model := &PointsAccountModel{
		ID:           validAccountID.String(),
		MemberID:     validMemberID.String(),
		EarnedPoints: 100,
		UsedPoints:   -30, // 無效：負數
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Act
	account, err := toDomain(model)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, points.ErrCorruptedUsedPoints)
}

// Test 4: toDomain 檢測不變條件違反（usedPoints > earnedPoints）
func TestToDomain_InvariantViolation_ReturnsError(t *testing.T) {
	// Arrange
	validAccountID := points.NewAccountID()
	validMemberID := points.NewMemberID()

	model := &PointsAccountModel{
		ID:           validAccountID.String(),
		MemberID:     validMemberID.String(),
		EarnedPoints: 50,
		UsedPoints:   100, // 無效：usedPoints > earnedPoints
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Act
	account, err := toDomain(model)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, points.ErrInvariantViolation)
}

// Test 5: toDomain 檢測無效的 AccountID
func TestToDomain_InvalidAccountID_ReturnsError(t *testing.T) {
	// Arrange
	validMemberID := points.NewMemberID()

	model := &PointsAccountModel{
		ID:           "", // 無效：空字串
		MemberID:     validMemberID.String(),
		EarnedPoints: 100,
		UsedPoints:   30,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Act
	account, err := toDomain(model)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, points.ErrInvalidAccountID)
}

// Test 6: toDomain 檢測無效的 MemberID
func TestToDomain_InvalidMemberID_ReturnsError(t *testing.T) {
	// Arrange
	validAccountID := points.NewAccountID()

	model := &PointsAccountModel{
		ID:           validAccountID.String(),
		MemberID:     "", // 無效：空字串
		EarnedPoints: 100,
		UsedPoints:   30,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Act
	account, err := toDomain(model)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, points.ErrInvalidMemberID)
}

// ===========================
// toGORM 轉換測試
// ===========================

// Test 7: toGORM 成功轉換 Domain 聚合
func TestToGORM_ValidAggregate_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, err := points.NewPointsAccount(memberID)
	require.NoError(t, err)

	// 獲得並使用積分
	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "測試")

	used, _ := points.NewPointsAmount(30)
	account.DeductPoints(used, "兌換")

	// Act
	model := toGORM(account)

	// Assert
	assert.NotNil(t, model)
	assert.Equal(t, account.AccountID().String(), model.ID)
	assert.Equal(t, memberID.String(), model.MemberID)
	assert.Equal(t, 100, model.EarnedPoints)
	assert.Equal(t, 30, model.UsedPoints)
	assert.False(t, model.CreatedAt.IsZero())
	assert.False(t, model.UpdatedAt.IsZero())
}

// Test 8: toGORM 處理新建立的帳戶
func TestToGORM_NewAccount_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, err := points.NewPointsAccount(memberID)
	require.NoError(t, err)

	// Act
	model := toGORM(account)

	// Assert
	assert.NotNil(t, model)
	assert.NotEmpty(t, model.ID)
	assert.NotEmpty(t, model.MemberID)
	assert.Equal(t, 0, model.EarnedPoints)
	assert.Equal(t, 0, model.UsedPoints)
}
