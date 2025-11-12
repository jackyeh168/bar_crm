package points

import (
	"github.com/shopspring/decimal"
)

// ===========================
// PointsCalculationService 領域服務
// ===========================

// PointsCalculationService 積分計算領域服務
//
// 設計原則：
// 1. Domain Service 封裝不屬於任何單一實體/值對象的業務邏輯
// 2. 協調多個值對象（ConversionRate + Money → PointsAmount）
// 3. 無狀態（stateless）- 所有數據通過參數傳入
//
// 為什麼需要 Domain Service：
// - ConversionRate 不應該依賴 PointsAmount（違反 DIP）
// - PointsAmount 不應該知道如何從金額計算（不是它的職責）
// - 計算邏輯需要協調兩個值對象，屬於領域服務的範疇
type PointsCalculationService struct{}

// NewPointsCalculationService 建構函數
// Domain Service 通常是無狀態的，但保留建構函數用於未來擴展
func NewPointsCalculationService() *PointsCalculationService {
	return &PointsCalculationService{}
}

// CalculateFromAmount 根據消費金額和轉換率計算積分
//
// 業務規則：
// - 積分 = floor(金額 / 轉換率)
// - 使用向下取整（消費者不會因為 99.99 元得到 1 點）
// - 負數金額返回 0 積分（防禦性編程）
//
// 參數：
//   amount - 消費金額（使用 decimal.Decimal 確保精確計算）
//   rate - 轉換率值對象
//
// 返回：
//   PointsAmount - 計算得到的積分（保證 >= 0）
//   error - 如果計算過程出現錯誤（如整數溢出）
func (s *PointsCalculationService) CalculateFromAmount(
	amount decimal.Decimal,
	rate ConversionRate,
) (PointsAmount, error) {
	// 將轉換率轉換為 decimal 進行精確計算
	rateValue := decimal.NewFromInt(int64(rate.Value()))

	// 計算：amount / conversion_rate，然後向下取整
	pointsValue := amount.Div(rateValue).Floor().IntPart()

	// 處理邊緣情況：負數金額（理論上不應該發生，但保持防禦性）
	if pointsValue < 0 {
		pointsValue = 0
	}

	// 使用 checked 建構函數，確保積分有效性
	return NewPointsAmount(int(pointsValue))
}

// ===========================
// 設計決策說明
// ===========================

// Q: 為什麼不直接在 ConversionRate 上實現 CalculatePoints？
// A: 違反依賴倒置原則（DIP）
//    - ConversionRate（值對象）會依賴 PointsAmount（另一個值對象）
//    - 值對象應該是"葉節點"，沒有依賴
//    - 協調多個值對象的邏輯屬於 Domain Service

// Q: 為什麼 PointsCalculationService 是無狀態的？
// A: Domain Service 不持有狀態，所有數據通過參數傳入
//    - 可以安全地在多個 goroutine 中共享
//    - 易於測試（純函數，無副作用）
//    - 符合函數式編程原則

// Q: 這個 Service 應該在哪裡使用？
// A: 在 Application Layer 的 Use Case 中：
//    - EarnPointsUseCase: 計算並累加積分
//    - 或在 PointsAccount 聚合根的方法中調用

// Q: 為什麼返回 error 而不是直接返回 PointsAmount？
// A: 雖然當前實現不會失敗，但：
//    - 未來可能添加更複雜的驗證（如積分上限檢查）
//    - 保持與其他建構函數一致的錯誤處理模式
//    - 遵循 Go 的慣例（可能失敗的操作返回 error）
