package points

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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

// ConversionRate 轉換率值對象（例如 100 元 = 1 點）
// 設計原則：值對象不可變、自我驗證
//
// 業務規則：範圍 1-1000
// - 最小值 1：表示 1 元 = 1 點（極端促銷）
// - 最大值 1000：表示 1000 元 = 1 點（極低回饋率）
// - 標準值 100：表示 100 元 = 1 點（常見設定）
type ConversionRate struct {
	value int
}

// NewConversionRate 建構函數（checked 版本）
// 對外部輸入進行完整驗證
//
// 建構約束：轉換率必須在 1-1000 範圍內
func NewConversionRate(value int) (ConversionRate, error) {
	if value < 1 || value > 1000 {
		return ConversionRate{}, fmt.Errorf(
			"%w: attempted to create ConversionRate with value %d (must be 1-1000)",
			ErrInvalidConversionRate,
			value,
		)
	}
	return ConversionRate{value: value}, nil
}

// Value 獲取轉換率值
func (r ConversionRate) Value() int {
	return r.value
}

// Equals 比較兩個 ConversionRate 是否相等
func (r ConversionRate) Equals(other ConversionRate) bool {
	return r.value == other.value
}

// CalculatePoints 計算積分（核心業務邏輯）
// 積分 = floor(金額 / 轉換率)
//
// 設計原則：
// 1. 使用向下取整（floor）：99.99 元在 100 TWD/點 的規則下為 0 點
// 2. 處理負數金額：返回 0 點（理論上不應該發生）
// 3. 使用 decimal.Decimal 確保精確計算
//
// 業務規則：積分必須為非負整數
func (r ConversionRate) CalculatePoints(amount decimal.Decimal) PointsAmount {
	// 防止除以零（理論上不會發生，因為 ConversionRate >= 1）
	if r.value == 0 {
		return newPointsAmountUnchecked(0)
	}

	// 計算：amount / conversion_rate，然後向下取整
	rate := decimal.NewFromInt(int64(r.value))
	points := amount.Div(rate).Floor().IntPart()

	// floor 結果可能為負數（如果 amount 為負）
	// 我們確保返回的積分 >= 0
	if points < 0 {
		return newPointsAmountUnchecked(0)
	}

	// 安全使用 unchecked 建構：已確保 points >= 0
	return newPointsAmountUnchecked(int(points))
}

// AccountID 帳戶 ID 值對象（UUID 封裝）
// 設計原則：值對象不可變、自我驗證
//
// 用途：唯一標識一個積分帳戶
type AccountID struct {
	value uuid.UUID
}

// NewAccountID 生成新的帳戶 ID（使用 UUID v4）
func NewAccountID() AccountID {
	return AccountID{value: uuid.New()}
}

// AccountIDFromString 從字串解析帳戶 ID
// 建構約束：必須是有效的 UUID 格式
func AccountIDFromString(s string) (AccountID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return AccountID{}, fmt.Errorf(
			"%w: attempted to parse AccountID from invalid UUID string '%s': %v",
			ErrInvalidAccountID,
			s,
			err,
		)
	}
	return AccountID{value: id}, nil
}

// String 轉換為字串表示（小寫 UUID）
func (a AccountID) String() string {
	return a.value.String()
}

// Equals 比較兩個 AccountID 是否相等
func (a AccountID) Equals(other AccountID) bool {
	return a.value == other.value
}

// IsEmpty 判斷是否為空 ID（零值）
func (a AccountID) IsEmpty() bool {
	return a.value == uuid.Nil
}

// MemberID 會員 ID 值對象（UUID 封裝）
// 設計原則：值對象不可變、自我驗證
//
// 用途：唯一標識一個會員
type MemberID struct {
	value uuid.UUID
}

// NewMemberID 生成新的會員 ID（使用 UUID v4）
func NewMemberID() MemberID {
	return MemberID{value: uuid.New()}
}

// MemberIDFromString 從字串解析會員 ID
// 建構約束：必須是有效的 UUID 格式
func MemberIDFromString(s string) (MemberID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return MemberID{}, fmt.Errorf(
			"%w: attempted to parse MemberID from invalid UUID string '%s': %v",
			ErrInvalidMemberID,
			s,
			err,
		)
	}
	return MemberID{value: id}, nil
}

// String 轉換為字串表示（小寫 UUID）
func (m MemberID) String() string {
	return m.value.String()
}

// Equals 比較兩個 MemberID 是否相等
func (m MemberID) Equals(other MemberID) bool {
	return m.value == other.value
}

// IsEmpty 判斷是否為空 ID（零值）
func (m MemberID) IsEmpty() bool {
	return m.value == uuid.Nil
}
