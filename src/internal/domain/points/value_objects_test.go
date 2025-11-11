package points_test

import (
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/stretchr/testify/assert"
)

// ===== PointsAmount 測試 =====

// Test 1: 建構有效的 PointsAmount
func TestNewPointsAmount_ValidValue_ReturnsPointsAmount(t *testing.T) {
	// Arrange
	value := 100

	// Act
	amount, err := points.NewPointsAmount(value)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, amount.Value())
}

// Test 2: 建構負數 PointsAmount 失敗（建構約束違反）
func TestNewPointsAmount_NegativeValue_ReturnsError(t *testing.T) {
	// Arrange
	value := -10

	// Act
	amount, err := points.NewPointsAmount(value)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrNegativePointsAmount)
	assert.Equal(t, 0, amount.Value())
	// 驗證錯誤訊息包含嘗試的值
	assert.Contains(t, err.Error(), "value -10")
}

// Test 3: 建構零值 PointsAmount
func TestNewPointsAmount_ZeroValue_ReturnsPointsAmount(t *testing.T) {
	// Arrange
	value := 0

	// Act
	amount, err := points.NewPointsAmount(value)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0, amount.Value())
}

// Test 4: PointsAmount 相加
func TestPointsAmount_Add_ReturnsNewPointsAmount(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(50)

	// Act
	result := amount1.Add(amount2)

	// Assert
	assert.Equal(t, 150, result.Value())
	// 驗證不變性：原始值不變
	assert.Equal(t, 100, amount1.Value())
	assert.Equal(t, 50, amount2.Value())
}

// Test 5: PointsAmount 相減
func TestPointsAmount_Subtract_ReturnsNewPointsAmount(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(30)

	// Act
	result, err := amount1.Subtract(amount2)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 70, result.Value())
	// 驗證不變性
	assert.Equal(t, 100, amount1.Value())
}

// Test 6: PointsAmount 相減超過範圍失敗（業務規則違反：積分不足）
func TestPointsAmount_Subtract_ExceedsValue_ReturnsError(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(50)
	amount2, _ := points.NewPointsAmount(100)

	// Act
	result, err := amount1.Subtract(amount2)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrInsufficientPoints)
	assert.Equal(t, 0, result.Value())
	// 驗證錯誤訊息包含上下文
	assert.Contains(t, err.Error(), "cannot subtract 100 from 50")
}

// Test 7: PointsAmount 比較相等
func TestPointsAmount_Equals_SameValue_ReturnsTrue(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(100)

	// Act
	result := amount1.Equals(amount2)

	// Assert
	assert.True(t, result)
}

// Test 8: PointsAmount 比較不相等
func TestPointsAmount_Equals_DifferentValue_ReturnsFalse(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(50)

	// Act
	result := amount1.Equals(amount2)

	// Assert
	assert.False(t, result)
}

// Test 9: PointsAmount 比較大小
func TestPointsAmount_GreaterThan_ReturnsCorrectResult(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(50)
	amount3, _ := points.NewPointsAmount(100)

	// Act & Assert
	assert.True(t, amount1.GreaterThan(amount2))
	assert.False(t, amount2.GreaterThan(amount1))
	assert.False(t, amount1.GreaterThan(amount3)) // 相等不算大於
}
