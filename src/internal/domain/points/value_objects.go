package points

import (
	"time"
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

// NOTE: CalculatePoints 方法已移除
// 原因：違反依賴倒置原則（DIP）- ConversionRate 不應依賴 PointsAmount
// 替代方案：使用 PointsCalculationService.CalculateFromAmount()
// 見 services.go 和 Uncle Bob Code Review - Day 2 Critical Issue #1

// NOTE: AccountID 和 MemberID 實現已移至 identifiers.go
// 原因：消除 82% 代碼重複（Uncle Bob Code Review - Day 2 Critical Issue #2）
// 新實現：使用泛型 shared.EntityID[T] + 類型別名
// 代碼減少：64 行 → 20 行（包含註釋）
// 見 identifiers.go 和 shared/entity_id.go

// ===========================
// DateRange 日期範圍值對象
// ===========================

// DateRange 日期範圍值對象
// 建構約束：startDate 必須 <= endDate
//
// 用途：
// - 積分轉換規則的有效期間
// - 促銷活動的時間範圍
// - 報表查詢的日期範圍
//
// 設計原則：
// - 不可變性：time.Time 本身是值類型
// - 自我驗證：建構函數檢查日期範圍有效性
// - 業務方法：Contains、Overlaps
type DateRange struct {
	startDate time.Time
	endDate   time.Time
}

// NewDateRange 創建日期範圍
//
// 建構約束：startDate 必須 <= endDate
//
// 參數：
//   startDate - 開始日期（包含）
//   endDate - 結束日期（包含）
//
// 返回：
//   DateRange - 有效的日期範圍
//   error - 如果 startDate > endDate
func NewDateRange(startDate, endDate time.Time) (DateRange, error) {
	if startDate.After(endDate) {
		return DateRange{}, ErrInvalidDateRange.WithContext(
			"start_date", startDate,
			"end_date", endDate,
		)
	}
	return DateRange{
		startDate: startDate,
		endDate:   endDate,
	}, nil
}

// DurationDays 計算日期範圍的天數（包含起始和結束日）
//
// 業務規則：
// - 返回範圍內的總天數（包含邊界）
// - 例如：2024-01-01 到 2024-01-03 返回 3 天
//
// 使用場景：
// - 計算促銷活動持續時間
// - 報表統計時間跨度
// - 驗證規則有效期長度
func (dr DateRange) DurationDays() int {
	duration := dr.endDate.Sub(dr.startDate)
	days := int(duration.Hours()/24) + 1 // +1 因為包含起始日
	return days
}

// Contains 判斷指定日期是否在範圍內（包含邊界）
//
// 業務規則：
// - 如果 date >= startDate AND date <= endDate，返回 true
// - 邊界日期被認為在範圍內
//
// 使用場景：
// - 判斷轉換規則在特定日期是否有效
// - 檢查交易日期是否在促銷期間內
func (dr DateRange) Contains(date time.Time) bool {
	return !date.Before(dr.startDate) && !date.After(dr.endDate)
}

// Overlaps 判斷是否與另一個日期範圍重疊
//
// 業務規則：
// - 如果兩個範圍有任何交集，返回 true
// - 邊界接觸不算重疊（例如 [2024-01-01, 2024-01-31] 和 [2024-02-01, 2024-02-28]）
//
// 使用場景：
// - 檢查轉換規則是否與其他規則衝突
// - 驗證促銷活動時間不重疊
//
// 算法說明：
// 兩個範圍 [a1, a2] 和 [b1, b2] 重疊的條件：
// - a1 < b2 AND b1 < a2
// 反之則不重疊
func (dr DateRange) Overlaps(other DateRange) bool {
	return dr.startDate.Before(other.endDate) && other.startDate.Before(dr.endDate)
}

// ===========================
// PointsSource 積分來源枚舉
// ===========================

// PointsSource 積分來源枚舉
//
// 用途：標識積分的來源，用於：
// - 積分交易記錄追蹤
// - 報表分類統計
// - 業務規則判斷（如某些來源的積分不可轉讓）
//
// 設計原則：
// - 使用 iota 自動遞增
// - 提供 String() 方法用於日誌和調試
// - 提供 IsValid() 方法驗證枚舉值有效性
// - 零值（Undefined）用於檢測未初始化的枚舉
//
// 版本規劃：
// - V3.1: Invoice, Survey（核心功能）
// - V3.2: Redemption（積分兌換）
// - V3.3: Expiration（積分過期）
// - V4.0: Transfer（積分轉讓）
type PointsSource int

const (
	PointsSourceUndefined  PointsSource = iota // 未定義：檢測未初始化的枚舉（零值）
	PointsSourceInvoice                        // 發票：消費獲得積分
	PointsSourceSurvey                         // 問卷：完成問卷獎勵積分
	PointsSourceRedemption                     // 兌換：使用積分兌換商品（負積分）（V3.2+）
	PointsSourceExpiration                     // 過期：積分過期扣除（負積分）（V3.3+）
	PointsSourceTransfer                       // 轉讓：積分轉讓給他人（V4.0+）
)

// String 返回積分來源的字符串表示（僅用於調試和日誌）
//
// 設計原則：
// - 領域層不應知道 API 格式（JSON、XML 等）
// - String() 僅用於調試、日誌記錄
// - API 響應格式應由 interface adapters 層處理
//
// 用途：
// - 日誌記錄：fmt.Printf("source: %s", source)
// - 調試輸出：便於開發時查看枚舉值
// - 錯誤訊息：包含在 error context 中
//
// 返回值：
// - Go 風格的枚舉字符串："PointsSource(Invoice)"
// - "PointsSource(Undefined)" 表示未初始化
// - "PointsSource(Unknown)" 表示無效值
//
// 注意：如需 API 響應格式（如 JSON），請使用 interface adapters 層的轉換函數
func (s PointsSource) String() string {
	switch s {
	case PointsSourceUndefined:
		return "PointsSource(Undefined)"
	case PointsSourceInvoice:
		return "PointsSource(Invoice)"
	case PointsSourceSurvey:
		return "PointsSource(Survey)"
	case PointsSourceRedemption:
		return "PointsSource(Redemption)"
	case PointsSourceExpiration:
		return "PointsSource(Expiration)"
	case PointsSourceTransfer:
		return "PointsSource(Transfer)"
	default:
		return "PointsSource(Unknown)"
	}
}

// IsValid 判斷積分來源是否有效
//
// 用途：
// - 驗證從外部輸入（API、數據庫）讀取的枚舉值
// - 防禦性編程，避免使用無效枚舉值
// - 檢測未初始化的枚舉（Undefined 不是有效值）
//
// 返回值：
// - true: 枚舉值在有效範圍內（不包含 Undefined）
// - false: 枚舉值無效（包括 Undefined 和超出範圍的值）
func (s PointsSource) IsValid() bool {
	return s >= PointsSourceInvoice && s <= PointsSourceTransfer
}
