package points

// PointsAmount 積分數量值對象
// 建構約束：積分數量必須 >= 0（不存在負數積分的概念）
type PointsAmount struct {
	value int
}

// NewPointsAmount 建構函數（checked 版本）
// 對外部輸入進行完整驗證
func NewPointsAmount(value int) (PointsAmount, error) {
	if value < 0 {
		return PointsAmount{}, ErrNegativePointsAmount.WithContext(
			"attempted_value", value,
			"constraint", ">= 0",
		)
	}
	return PointsAmount{value: value}, nil
}

// newPointsAmountUnchecked 內部建構函數（unchecked 版本）
// 僅供內部使用，當我們確定值有效時使用（性能優化）
//
// 設計理由：
// 1. 性能優化：在內部操作中跳過重複驗證（如 Add、Subtract 的結果）
// 2. 不變性保證：如果兩個有效的 PointsAmount 相加，結果必然有效（假設無溢位）
// 3. 封裝性：小寫開頭確保外部無法繞過驗證
//
// 前提條件：調用者必須保證 value >= 0
func newPointsAmountUnchecked(value int) PointsAmount {
	return PointsAmount{value: value}
}

func (p PointsAmount) Value() int {
	return p.value
}

// IsZero 判斷積分數量是否為零
func (p PointsAmount) IsZero() bool {
	return p.value == 0
}

// Add 相加（返回新的 PointsAmount，保持不變性）
// 包含溢出檢測，確保數據完整性
func (p PointsAmount) Add(other PointsAmount) (PointsAmount, error) {
	// 檢測整數溢出：如果 sum < p.value，表示發生溢出
	// 因為兩個非負數相加，結果應該 >= 較大的操作數
	const maxInt = int(^uint(0) >> 1) // 2147483647 for 32-bit, much larger for 64-bit

	if p.value > maxInt-other.value {
		return PointsAmount{}, ErrInvalidPointsAmount.WithContext(
			"operation", "add",
			"operand1", p.value,
			"operand2", other.value,
			"error", "integer overflow",
		)
	}

	sum := p.value + other.value
	return newPointsAmountUnchecked(sum), nil
}

// Subtract 相減（返回新的 PointsAmount）
// 建構約束：結果必須 >= 0（不能創建負數積分）
func (p PointsAmount) Subtract(other PointsAmount) (PointsAmount, error) {
	if p.value < other.value {
		return PointsAmount{}, ErrNegativePointsAmount.WithContext(
			"operation", "subtract",
			"minuend", p.value,
			"subtrahend", other.value,
			"result", p.value-other.value,
		)
	}

	result := p.value - other.value
	return newPointsAmountUnchecked(result), nil
}

func (p PointsAmount) Equals(other PointsAmount) bool {
	return p.value == other.value
}

func (p PointsAmount) GreaterThan(other PointsAmount) bool {
	return p.value > other.value
}

func (p PointsAmount) LessThan(other PointsAmount) bool {
	return p.value < other.value
}

func (p PointsAmount) GreaterThanOrEqual(other PointsAmount) bool {
	return p.value >= other.value
}

// ConversionRate 轉換率值對象（例如 100 元 = 1 點）
// 建構約束：範圍 1-1000
type ConversionRate struct {
	value int
}

// NewConversionRate 建構函數
// 建構約束：轉換率必須在 1-1000 範圍內
func NewConversionRate(value int) (ConversionRate, error) {
	if value < 1 || value > 1000 {
		return ConversionRate{}, ErrInvalidConversionRate.WithContext(
			"attempted_value", value,
			"constraint", "1-1000",
		)
	}
	return ConversionRate{value: value}, nil
}

func (r ConversionRate) Value() int {
	return r.value
}

func (r ConversionRate) Equals(other ConversionRate) bool {
	return r.value == other.value
}

// NOTE: CalculatePoints 方法已移除
// 原因：違反依賴倒置原則（DIP）- ConversionRate 不應依賴 PointsAmount
// 替代方案：使用 PointsCalculationService.CalculateFromAmount()
// 見 services.go 和 Uncle Bob Code Review - Day 2 Critical Issue #1

// NOTE: AccountID 和 MemberID 實現已移至 identifiers.go
// 原因：消除 82% 代碼重複（Uncle Bob Code Review - Day 2 Critical Issue #2）
// 新實現：使用泛型 shared.EntityID[T] + 類型別名
// 代碼減少：64 行 → 20 行（包含註釋）
// 見 identifiers.go 和 shared/entity_id.go
