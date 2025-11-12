package points_test

import (
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// ===== PointsCalculationService 測試 =====

// Test 1: CalculateFromAmount 標準轉換測試
func TestPointsCalculationService_CalculateFromAmount_StandardConversion(t *testing.T) {
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
		{
			name:           "高轉換率 1000 TWD = 1 點",
			conversionRate: 1000,
			amount:         "2500.00",
			expectedPoints: 2, // floor(2500/1000) = 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			service := points.NewPointsCalculationService()
			rate, err := points.NewConversionRate(tt.conversionRate)
			assert.NoError(t, err)

			amount, err := decimal.NewFromString(tt.amount)
			assert.NoError(t, err)

			// Act
			result, err := service.CalculateFromAmount(amount, rate)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPoints, result.Value())
		})
	}
}

// Test 2: CalculateFromAmount 負數金額處理
func TestPointsCalculationService_CalculateFromAmount_NegativeAmount(t *testing.T) {
	// Arrange
	service := points.NewPointsCalculationService()
	rate, _ := points.NewConversionRate(100)
	negativeAmount := decimal.NewFromFloat(-50.00)

	// Act
	result, err := service.CalculateFromAmount(negativeAmount, rate)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Value(), "負數金額應該返回 0 積分")
}

// Test 3: CalculateFromAmount 大金額測試（確保無溢出）
func TestPointsCalculationService_CalculateFromAmount_LargeAmount(t *testing.T) {
	// Arrange
	service := points.NewPointsCalculationService()
	rate, _ := points.NewConversionRate(1)

	// 測試 1,000,000 TWD = 1,000,000 點（在 int 範圍內）
	largeAmount := decimal.NewFromInt(1000000)

	// Act
	result, err := service.CalculateFromAmount(largeAmount, rate)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1000000, result.Value())
}

// Test 4: CalculateFromAmount 精確度測試（小數點處理）
func TestPointsCalculationService_CalculateFromAmount_DecimalPrecision(t *testing.T) {
	tests := []struct {
		name           string
		amount         string
		expectedPoints int
	}{
		{"99.01 元", "99.01", 0},
		{"99.99 元", "99.99", 0},
		{"100.00 元", "100.00", 1},
		{"100.01 元", "100.01", 1},
		{"199.99 元", "199.99", 1},
		{"200.00 元", "200.00", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			service := points.NewPointsCalculationService()
			rate, _ := points.NewConversionRate(100)
			amount, _ := decimal.NewFromString(tt.amount)

			// Act
			result, err := service.CalculateFromAmount(amount, rate)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPoints, result.Value(),
				"金額 %s 在 100 TWD/點 規則下應得 %d 點", tt.amount, tt.expectedPoints)
		})
	}
}

// Test 5: CalculateFromAmount 服務無狀態驗證
func TestPointsCalculationService_Stateless(t *testing.T) {
	// Arrange
	service := points.NewPointsCalculationService()
	rate, _ := points.NewConversionRate(100)
	amount1, _ := decimal.NewFromString("100.00")
	amount2, _ := decimal.NewFromString("200.00")

	// Act - 連續調用
	result1, err1 := service.CalculateFromAmount(amount1, rate)
	result2, err2 := service.CalculateFromAmount(amount2, rate)

	// Assert - 每次調用獨立，互不影響
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 1, result1.Value())
	assert.Equal(t, 2, result2.Value())
}
