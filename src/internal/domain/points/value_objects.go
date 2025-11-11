package points

import "fmt"

// PointsAmount 積分數量值對象
// 設計原則：值對象不可變、自我驗證
type PointsAmount struct {
	value int
}

// NewPointsAmount 建構函數（checked 版本）
// 對外部輸入進行完整驗證
//
// 建構約束：積分數量必須 >= 0（不存在負數積分的概念）
func NewPointsAmount(value int) (PointsAmount, error) {
	if value < 0 {
		return PointsAmount{}, fmt.Errorf(
			"%w: attempted to create PointsAmount with value %d",
			ErrNegativePointsAmount,
			value,
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

// Value 獲取積分數量
func (p PointsAmount) Value() int {
	return p.value
}

// Add 相加（返回新的 PointsAmount，保持不變性）
//
// 設計假設：
// 在生產系統中，積分受業務規則限制（例如：單帳戶最多 1,000,000 點）
// 因此整數溢位在實際業務場景中不會發生
//
// 如果需要處理任意大的積分數量，應使用 big.Int 或在聚合根層面強制上限
func (p PointsAmount) Add(other PointsAmount) PointsAmount {
	return newPointsAmountUnchecked(p.value + other.value)
}

// Subtract 相減（返回新的 PointsAmount）
// 業務規則：不能扣除超過當前數量的積分
func (p PointsAmount) Subtract(other PointsAmount) (PointsAmount, error) {
	// 檢查業務規則：餘額是否足夠
	if p.value < other.value {
		// 這是業務規則違反，不是建構約束違反
		return PointsAmount{}, fmt.Errorf(
			"%w: cannot subtract %d from %d (insufficient balance)",
			ErrInsufficientPoints,
			other.value,
			p.value,
		)
	}

	// 已經保證 result >= 0，可以安全使用 unchecked 建構
	result := p.value - other.value
	return newPointsAmountUnchecked(result), nil
}

// Equals 比較兩個 PointsAmount 是否相等
func (p PointsAmount) Equals(other PointsAmount) bool {
	return p.value == other.value
}

// GreaterThan 判斷是否大於另一個 PointsAmount
func (p PointsAmount) GreaterThan(other PointsAmount) bool {
	return p.value > other.value
}

// LessThan 判斷是否小於另一個 PointsAmount
func (p PointsAmount) LessThan(other PointsAmount) bool {
	return p.value < other.value
}

// GreaterThanOrEqual 判斷是否大於等於另一個 PointsAmount
func (p PointsAmount) GreaterThanOrEqual(other PointsAmount) bool {
	return p.value >= other.value
}
