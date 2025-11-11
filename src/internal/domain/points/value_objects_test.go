package points_test

import (
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/shopspring/decimal"
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

// ===== ConversionRate 測試 =====

// Test 10: 建構有效的 ConversionRate
func TestNewConversionRate_ValidRate_Success(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"最小值 1", 1},
		{"標準值 100", 100},
		{"最大值 1000", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			rate, err := points.NewConversionRate(tt.value)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.value, rate.Value())
		})
	}
}

// Test 11: 建構無效的 ConversionRate 失敗
func TestNewConversionRate_InvalidRate_ReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"低於最小值", 0},
		{"負數", -10},
		{"超過最大值", 1001},
		{"遠超最大值", 5000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			rate, err := points.NewConversionRate(tt.value)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, points.ErrInvalidConversionRate)
			assert.Equal(t, 0, rate.Value())
		})
	}
}

// Test 12: CalculatePoints 標準轉換（核心業務邏輯）
func TestConversionRate_CalculatePoints(t *testing.T) {
	tests := []struct {
		name           string
		conversionRate int
		amount         string // decimal string
		expectedPoints int
	}{
		{
			name:           "標準轉換 100 TWD = 1 點",
			conversionRate: 100,
			amount:         "350.00",
			expectedPoints: 3, // floor(350/100) = 3
		},
		{
			name:           "促銷轉換 50 TWD = 1 點",
			conversionRate: 50,
			amount:         "125.00",
			expectedPoints: 2, // floor(125/50) = 2
		},
		{
			name:           "小數金額向下取整",
			conversionRate: 100,
			amount:         "99.99",
			expectedPoints: 0, // floor(99.99/100) = 0
		},
		{
			name:           "剛好整除",
			conversionRate: 100,
			amount:         "500.00",
			expectedPoints: 5, // floor(500/100) = 5
		},
		{
			name:           "零金額",
			conversionRate: 100,
			amount:         "0.00",
			expectedPoints: 0,
		},
		{
			name:           "1 元 = 1 點（極端情況）",
			conversionRate: 1,
			amount:         "5.50",
			expectedPoints: 5, // floor(5.50/1) = 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			rate, err := points.NewConversionRate(tt.conversionRate)
			assert.NoError(t, err)

			amount, err := decimal.NewFromString(tt.amount)
			assert.NoError(t, err)

			// Act
			result := rate.CalculatePoints(amount)

			// Assert
			assert.Equal(t, tt.expectedPoints, result.Value())
		})
	}
}

// Test 13: CalculatePoints 負數金額（理論上不應該發生）
func TestConversionRate_CalculatePoints_NegativeAmount(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	negativeAmount := decimal.NewFromFloat(-50.00)

	// Act
	result := rate.CalculatePoints(negativeAmount)

	// Assert: 應該返回 0（向下取整的結果）
	assert.Equal(t, 0, result.Value())
}

// ===== AccountID 測試 =====

// Test 14: NewAccountID 生成新 UUID
func TestNewAccountID_GeneratesUUID(t *testing.T) {
	// Act
	id1 := points.NewAccountID()
	id2 := points.NewAccountID()

	// Assert
	assert.NotEqual(t, "", id1.String())
	assert.NotEqual(t, "", id2.String())
	assert.NotEqual(t, id1.String(), id2.String()) // 每次生成不同的 UUID
}

// Test 15: AccountIDFromString 有效 UUID
func TestAccountIDFromString_ValidUUID_Success(t *testing.T) {
	// Arrange
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	// Act
	id, err := points.AccountIDFromString(validUUID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, validUUID, id.String())
}

// Test 16: AccountIDFromString 無效 UUID
func TestAccountIDFromString_InvalidUUID_ReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"空字串", ""},
		{"不是 UUID 格式", "not-a-uuid"},
		{"錯誤格式", "123-456-789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			id, err := points.AccountIDFromString(tt.value)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, points.ErrInvalidAccountID)
			assert.True(t, id.IsEmpty())
		})
	}
}

// Test 17: AccountID Equals
func TestAccountID_Equals(t *testing.T) {
	// Arrange
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	id1, _ := points.AccountIDFromString(uuid)
	id2, _ := points.AccountIDFromString(uuid)
	id3 := points.NewAccountID()

	// Act & Assert
	assert.True(t, id1.Equals(id2))
	assert.False(t, id1.Equals(id3))
}

// Test 18: AccountID IsEmpty
func TestAccountID_IsEmpty(t *testing.T) {
	// Arrange
	emptyID := points.AccountID{}
	validID := points.NewAccountID()

	// Act & Assert
	assert.True(t, emptyID.IsEmpty())
	assert.False(t, validID.IsEmpty())
}

// ===== MemberID 測試 =====

// Test 19: NewMemberID 生成新 UUID
func TestNewMemberID_GeneratesUUID(t *testing.T) {
	// Act
	id1 := points.NewMemberID()
	id2 := points.NewMemberID()

	// Assert
	assert.NotEqual(t, "", id1.String())
	assert.NotEqual(t, "", id2.String())
	assert.NotEqual(t, id1.String(), id2.String()) // 每次生成不同的 UUID
}

// Test 20: MemberIDFromString 有效 UUID
func TestMemberIDFromString_ValidUUID_Success(t *testing.T) {
	// Arrange
	validUUID := "660e8400-e29b-41d4-a716-446655440000"

	// Act
	id, err := points.MemberIDFromString(validUUID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, validUUID, id.String())
}

// Test 21: MemberIDFromString 無效 UUID
func TestMemberIDFromString_InvalidUUID_ReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"空字串", ""},
		{"不是 UUID 格式", "not-a-uuid"},
		{"錯誤格式", "123-456-789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			id, err := points.MemberIDFromString(tt.value)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, points.ErrInvalidMemberID)
			assert.True(t, id.IsEmpty())
		})
	}
}

// Test 22: MemberID Equals
func TestMemberID_Equals(t *testing.T) {
	// Arrange
	uuid := "660e8400-e29b-41d4-a716-446655440000"
	id1, _ := points.MemberIDFromString(uuid)
	id2, _ := points.MemberIDFromString(uuid)
	id3 := points.NewMemberID()

	// Act & Assert
	assert.True(t, id1.Equals(id2))
	assert.False(t, id1.Equals(id3))
}

// Test 23: MemberID IsEmpty
func TestMemberID_IsEmpty(t *testing.T) {
	// Arrange
	emptyID := points.MemberID{}
	validID := points.NewMemberID()

	// Act & Assert
	assert.True(t, emptyID.IsEmpty())
	assert.False(t, validID.IsEmpty())
}
