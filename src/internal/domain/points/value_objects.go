package points

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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

// CalculatePoints 計算積分
// 積分 = floor(金額 / 轉換率)
//
// TODO (Uncle Bob Review): 違反 SRP - 業務邏輯應該在 PointsCalculationService (Domain Service)
// 這個方法將在 Day 6 重構時移除
func (r ConversionRate) CalculatePoints(amount decimal.Decimal) PointsAmount {
	rate := decimal.NewFromInt(int64(r.value))
	points := amount.Div(rate).Floor().IntPart()

	// 確保返回的積分 >= 0（處理負數金額）
	if points < 0 {
		return newPointsAmountUnchecked(0)
	}

	return newPointsAmountUnchecked(int(points))
}

// AccountID 帳戶 ID 值對象（UUID 封裝）
// TODO (Uncle Bob Review): AccountID 和 MemberID 有 80+ 行重複代碼
// 建議在 Week 1 結束時使用 Go 泛型重構（EntityID[T]）
type AccountID struct {
	value uuid.UUID
}

func NewAccountID() AccountID {
	return AccountID{value: uuid.New()}
}

func AccountIDFromString(s string) (AccountID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return AccountID{}, ErrInvalidAccountID.WithContext(
			"input", s,
			"parse_error", err.Error(),
		)
	}
	return AccountID{value: id}, nil
}

func (a AccountID) String() string {
	return a.value.String()
}

func (a AccountID) Equals(other AccountID) bool {
	return a.value == other.value
}

func (a AccountID) IsEmpty() bool {
	return a.value == uuid.Nil
}

// MemberID 會員 ID 值對象（UUID 封裝）
type MemberID struct {
	value uuid.UUID
}

func NewMemberID() MemberID {
	return MemberID{value: uuid.New()}
}

func MemberIDFromString(s string) (MemberID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return MemberID{}, ErrInvalidMemberID.WithContext(
			"input", s,
			"parse_error", err.Error(),
		)
	}
	return MemberID{value: id}, nil
}

func (m MemberID) String() string {
	return m.value.String()
}

func (m MemberID) Equals(other MemberID) bool {
	return m.value == other.value
}

func (m MemberID) IsEmpty() bool {
	return m.value == uuid.Nil
}
