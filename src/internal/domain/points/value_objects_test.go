package points_test

import (
	"testing"
	"time"

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
	// 驗證錯誤訊息包含上下文信息
	assert.Contains(t, err.Error(), "POINTS_NEGATIVE")
	assert.Contains(t, err.Error(), "attempted_value")
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
	result, err := amount1.Add(amount2)

	// Assert
	assert.NoError(t, err)
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

// Test 6: PointsAmount 相減超過範圍失敗（建構約束違反：會產生負數）
func TestPointsAmount_Subtract_ExceedsValue_ReturnsError(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(50)
	amount2, _ := points.NewPointsAmount(100)

	// Act
	result, err := amount1.Subtract(amount2)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrNegativePointsAmount)
	assert.Equal(t, 0, result.Value())
	// 驗證錯誤訊息包含上下文信息
	assert.Contains(t, err.Error(), "POINTS_NEGATIVE")
	assert.Contains(t, err.Error(), "minuend")
	assert.Contains(t, err.Error(), "subtrahend")
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

// Test 9a: PointsAmount IsZero
func TestPointsAmount_IsZero(t *testing.T) {
	// Arrange
	zeroAmount, _ := points.NewPointsAmount(0)
	nonZeroAmount, _ := points.NewPointsAmount(100)

	// Act & Assert
	assert.True(t, zeroAmount.IsZero())
	assert.False(t, nonZeroAmount.IsZero())
}

// Test 9b: PointsAmount Add 溢出保護
func TestPointsAmount_Add_Overflow_ReturnsError(t *testing.T) {
	// Arrange - 使用實際的 maxInt（64位系統上會更大）
	const maxInt = int(^uint(0) >> 1)
	amount1, _ := points.NewPointsAmount(maxInt)
	amount2, _ := points.NewPointsAmount(1)

	// Act
	result, err := amount1.Add(amount2)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrInvalidPointsAmount)
	assert.Equal(t, 0, result.Value())
	assert.Contains(t, err.Error(), "POINTS_INVALID")
	assert.Contains(t, err.Error(), "overflow")
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

// NOTE: CalculatePoints 測試已移至 services_test.go
// 原因：CalculatePoints 違反 DIP，已移至 PointsCalculationService
// 見 Uncle Bob Code Review - Day 2 Critical Issue #1

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

// ===== DateRange 測試 =====

// Test 24: NewDateRange 有效範圍
func TestNewDateRange_ValidRange_Success(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	// Act
	dr, err := points.NewDateRange(start, end)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, start, dr.StartDate())
	assert.Equal(t, end, dr.EndDate())
}

// Test 25: NewDateRange 同一天
func TestNewDateRange_SameDay_Success(t *testing.T) {
	// Arrange
	sameDay := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	// Act
	dr, err := points.NewDateRange(sameDay, sameDay)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, sameDay, dr.StartDate())
	assert.Equal(t, sameDay, dr.EndDate())
}

// Test 26: NewDateRange 開始日期晚於結束日期
func TestNewDateRange_StartAfterEnd_ReturnsError(t *testing.T) {
	// Arrange
	start := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Act
	dr, err := points.NewDateRange(start, end)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrInvalidDateRange)
	assert.True(t, dr.StartDate().IsZero())
	assert.True(t, dr.EndDate().IsZero())
}

// Test 27: DateRange Contains - 日期在範圍內
func TestDateRange_Contains_DateInRange(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	testDate := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	// Act
	result := dr.Contains(testDate)

	// Assert
	assert.True(t, result)
}

// Test 28: DateRange Contains - 日期在範圍外（之前）
func TestDateRange_Contains_DateBeforeRange(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	testDate := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)

	// Act
	result := dr.Contains(testDate)

	// Assert
	assert.False(t, result)
}

// Test 29: DateRange Contains - 日期在範圍外（之後）
func TestDateRange_Contains_DateAfterRange(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	testDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Act
	result := dr.Contains(testDate)

	// Assert
	assert.False(t, result)
}

// Test 30: DateRange Contains - 邊界測試（開始日期）
func TestDateRange_Contains_StartDate(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	// Act
	result := dr.Contains(start)

	// Assert
	assert.True(t, result)
}

// Test 31: DateRange Contains - 邊界測試（結束日期）
func TestDateRange_Contains_EndDate(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	// Act
	result := dr.Contains(end)

	// Assert
	assert.True(t, result)
}

// Test 32: DateRange Overlaps - 完全重疊
func TestDateRange_Overlaps_CompleteOverlap(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.True(t, result)
}

// Test 33: DateRange Overlaps - 部分重疊
func TestDateRange_Overlaps_PartialOverlap(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.True(t, result)
}

// Test 34: DateRange Overlaps - 不重疊（之前）
func TestDateRange_Overlaps_NoOverlapBefore(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.False(t, result)
}

// Test 35: DateRange Overlaps - 不重疊（之後）
func TestDateRange_Overlaps_NoOverlapAfter(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.False(t, result)
}

// Test 36: DateRange Overlaps - 邊界接觸（不算重疊）
func TestDateRange_Overlaps_EdgeTouch(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.False(t, result)
}

// ===== PointsSource 測試 =====

// Test 37: PointsSource String 方法
func TestPointsSource_String(t *testing.T) {
	tests := []struct {
		name     string
		source   points.PointsSource
		expected string
	}{
		{"發票來源", points.PointsSourceInvoice, "invoice"},
		{"問卷來源", points.PointsSourceSurvey, "survey"},
		{"兌換來源", points.PointsSourceRedemption, "redemption"},
		{"過期來源", points.PointsSourceExpiration, "expiration"},
		{"轉讓來源", points.PointsSourceTransfer, "transfer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := tt.source.String()

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test 38: PointsSource 未知類型
func TestPointsSource_String_Unknown(t *testing.T) {
	// Arrange
	unknownSource := points.PointsSource(999)

	// Act
	result := unknownSource.String()

	// Assert
	assert.Equal(t, "unknown", result)
}

// Test 39: PointsSource IsValid 方法
func TestPointsSource_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		source   points.PointsSource
		expected bool
	}{
		{"Invoice 有效", points.PointsSourceInvoice, true},
		{"Survey 有效", points.PointsSourceSurvey, true},
		{"Redemption 有效", points.PointsSourceRedemption, true},
		{"Expiration 有效", points.PointsSourceExpiration, true},
		{"Transfer 有效", points.PointsSourceTransfer, true},
		{"負數無效", points.PointsSource(-1), false},
		{"超出範圍無效", points.PointsSource(999), false},
		{"零值無效", points.PointsSource(0), true}, // 0 is PointsSourceInvoice
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := tt.source.IsValid()

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}
