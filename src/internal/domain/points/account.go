package points

import (
	"fmt"
	"time"

	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"github.com/shopspring/decimal"
)

// ===========================
// PointsAccount 聚合根
// ===========================

// PointsAccount 積分帳戶聚合根
//
// 設計原則：
// 1. 輕量級聚合：不包含無界集合（交易記錄儲存在獨立表）
// 2. 不變條件：UsedPoints <= EarnedPoints（必須在每個修改方法末尾檢查）
// 3. 事件驅動：所有狀態變更都發布領域事件
// 4. Tell, Don't Ask：封裝業務邏輯，不暴露內部狀態供外部判斷
//
// 業務不變條件：
// - EarnedPoints >= 0（累積獲得的積分總數）
// - UsedPoints >= 0（累積使用的積分總數）
// - UsedPoints <= EarnedPoints（使用積分不能超過獲得積分）
// - AvailablePoints = EarnedPoints - UsedPoints（可用積分為派生值）
type PointsAccount struct {
	// 聚合根識別符
	accountID AccountID
	memberID  MemberID

	// 積分數據（使用值對象）
	earnedPoints PointsAmount // 累積獲得積分
	usedPoints   PointsAmount // 累積使用積分

	// 審計字段
	createdAt time.Time
	updatedAt time.Time

	// 待發布的領域事件
	events []shared.DomainEvent
}

// ===========================
// 建構函數（工廠方法）
// ===========================

// NewPointsAccount 創建新的積分帳戶
//
// 參數：
//   memberID - 會員 ID（必填）
//
// 返回：
//   *PointsAccount - 新創建的帳戶
//   error - 如果 memberID 無效
//
// 業務規則：
// - 新帳戶初始積分為 0
// - 自動生成唯一的 AccountID
// - 發布 AccountCreated 事件
func NewPointsAccount(memberID MemberID) (*PointsAccount, error) {
	// 驗證必填字段
	if memberID.IsEmpty() {
		return nil, ErrInvalidMemberID.WithContext(
			"reason", "memberID cannot be empty",
		)
	}

	now := time.Now()

	// 創建聚合根實例
	account := &PointsAccount{
		accountID:    NewAccountID(),
		memberID:     memberID,
		earnedPoints: newPointsAmountUnchecked(0), // 0 保證有效，使用 unchecked
		usedPoints:   newPointsAmountUnchecked(0),
		createdAt:    now,
		updatedAt:    now,
		events:       make([]shared.DomainEvent, 0),
	}

	// 發布領域事件
	account.addEvent(NewPointsAccountCreatedEvent(account.accountID, memberID))

	return account, nil
}

// ===========================
// 查詢方法（Getters）
// ===========================
//
// 設計說明：
// 雖然 "Tell, Don't Ask" 原則建議不暴露內部狀態，
// 但聚合根需要提供查詢方法供以下場景使用：
// 1. Repository 持久化（需要訪問所有字段）
// 2. DTO 轉換（Application Layer 需要構建響應）
// 3. 日誌和調試
//
// ⚠️ 警告：不應在業務邏輯中使用這些 getter 做判斷
// 正確做法：在聚合根內部提供業務方法（如 CanDeductPoints）

// AccountID 獲取帳戶 ID
func (a *PointsAccount) AccountID() AccountID {
	return a.accountID
}

// MemberID 獲取會員 ID
func (a *PointsAccount) MemberID() MemberID {
	return a.memberID
}

// EarnedPoints 獲取累積獲得積分
func (a *PointsAccount) EarnedPoints() PointsAmount {
	return a.earnedPoints
}

// UsedPoints 獲取累積使用積分
func (a *PointsAccount) UsedPoints() PointsAmount {
	return a.usedPoints
}

// CreatedAt 獲取創建時間
func (a *PointsAccount) CreatedAt() time.Time {
	return a.createdAt
}

// UpdatedAt 獲取最後更新時間
func (a *PointsAccount) UpdatedAt() time.Time {
	return a.updatedAt
}

// GetAvailablePoints 獲取可用積分（派生值）
//
// 業務規則：
// - AvailablePoints = EarnedPoints - UsedPoints
// - 此為派生值，不存儲在數據庫
//
// 不變條件保證：
// - 由於 usedPoints <= earnedPoints 不變條件，結果永遠 >= 0
// - 每次調用都重新計算，確保與當前狀態一致
//
// 使用場景：
// - 在 DeductPoints 前檢查是否有足夠積分
// - Application Layer 查詢可用積分餘額（用於 DTO 響應）
// - 不應用於外部判斷（Tell, Don't Ask 原則）
func (a *PointsAccount) GetAvailablePoints() PointsAmount {
	// 不變條件保證 earnedPoints >= usedPoints
	// 因此 Subtract() 永遠不會返回錯誤，可以安全忽略
	available, _ := a.earnedPoints.Subtract(a.usedPoints)
	return available
}

// ===========================
// 事件管理
// ===========================

// addEvent 添加領域事件到待發布列表（私有方法）
func (a *PointsAccount) addEvent(event shared.DomainEvent) {
	a.events = append(a.events, event)
}

// PullEvents 獲取所有待發布事件並清空列表
//
// 使用場景：
// - Repository.Save() 成功後，調用此方法獲取事件並發布
// - 事件發布由 Infrastructure 層的 EventPublisher 處理
//
// 設計原則：
// - Pull 模式（而非 Push）：聚合根不依賴 EventPublisher
// - 只讀取一次：獲取後清空，避免重複發布
func (a *PointsAccount) PullEvents() []shared.DomainEvent {
	events := a.events
	a.events = make([]shared.DomainEvent, 0) // 清空
	return events
}

// ===========================
// 命令方法（狀態變更）
// ===========================

// EarnPoints 獲得積分（核心業務邏輯）
//
// 參數：
//   amount - 獲得的積分數量（PointsAmount 已保證 >= 0）
//   source - 積分來源（發票、問卷等）
//   sourceID - 來源標識符（如發票號碼）
//   description - 獲得積分的描述
//
// 返回：
//   error - 如果發生溢位錯誤
//
// 前置條件（由類型系統保證）：
// - amount 已通過 NewPointsAmount 驗證，保證 >= 0
// - source 為有效的 PointsSource 枚舉值
//
// 業務邏輯：
// - 累加積分到 earnedPoints
// - 零積分也接受（業務上可能存在零金額發票）
//
// 副作用：
// - 更新 earnedPoints（累加）
// - 更新 updatedAt
// - 增加版本號
// - 發布 PointsEarnedEvent
//
// 不變條件維護：
// - 此方法只增加 earnedPoints，永遠不會違反 usedPoints <= earnedPoints
func (a *PointsAccount) EarnPoints(
	amount PointsAmount,
	source PointsSource,
	sourceID string,
	description string,
) error {
	// 狀態變更
	// PointsAmount.Add() 會檢測整數溢位並返回錯誤
	newEarnedPoints, err := a.earnedPoints.Add(amount)
	if err != nil {
		// 溢位錯誤：這應該極少發生（需要累積數十億積分）
		// 返回為業務錯誤，讓上層決定如何處理
		return err
	}

	a.earnedPoints = newEarnedPoints
	a.updatedAt = time.Now()

	// 發布領域事件
	// 事件將在 Repository.Save() 成功後通過 PullEvents() 獲取並發布
	// 這實現了 Transactional Outbox 模式
	a.addEvent(NewPointsEarnedEvent(
		a.accountID,
		amount,
		source,
		sourceID,
		description,
	))

	return nil
}

// DeductPoints 扣減積分（核心業務邏輯）
//
// 參數：
//   amount - 扣減的積分數量（PointsAmount 已保證 >= 0）
//   reason - 扣減原因（如「兌換商品」、「過期清除」）
//
// 返回：
//   error - 如果餘額不足或發生溢位錯誤
//
// 前置條件（由類型系統保證）：
// - amount 已通過 NewPointsAmount 驗證，保證 >= 0
//
// 業務規則：
// - 必須先檢查可用積分是否足夠（前置條件）
// - 零積分也接受（業務上可能存在測試場景）
//
// 副作用：
// - 更新 usedPoints（累加）
// - 更新 updatedAt
// - 增加版本號
// - 發布 PointsDeductedEvent
//
// 不變條件維護：
// - 前置條件檢查確保扣減後 usedPoints <= earnedPoints
func (a *PointsAccount) DeductPoints(
	amount PointsAmount,
	reason string,
) error {
	// 前置條件：檢查是否有足夠積分
	available := a.GetAvailablePoints()
	if amount.GreaterThan(available) {
		return ErrInsufficientPoints.WithContext(
			"requested", amount.Value(),
			"available", available.Value(),
			"reason", reason,
		)
	}

	// 狀態變更
	// PointsAmount.Add() 會檢測整數溢位並返回錯誤
	newUsedPoints, err := a.usedPoints.Add(amount)
	if err != nil {
		// 溢位錯誤：這應該極少發生（需要累積數十億積分）
		return err
	}

	a.usedPoints = newUsedPoints
	a.updatedAt = time.Now()

	// 發布領域事件
	// 事件將在 Repository.Save() 成功後通過 PullEvents() 獲取並發布
	a.addEvent(NewPointsDeductedEvent(
		a.accountID,
		amount,
		reason,
	))

	return nil
}

// ===========================
// RecalculatePoints 命令方法
// ===========================

// PointsCalculableTransaction 可計算積分的交易介面
//
// 設計原則：
// - 介面名稱表達用途（積分計算），而非資料結構
// - Application Layer 的 DTO 實作此介面
// - 避免 Domain Layer 依賴 Application Layer 的具體類型
// - 介面隔離原則（ISP）：只包含必要方法，避免強迫實作不需要的方法
//
// 設計決策（ISP 優化）：
// - 移除 GetOccurredAt()：當前 RecalculatePoints 不使用交易時間
// - 遵循 YAGNI 原則：不為未來可能需求設計
// - 若未來需要：可添加新介面或擴展現有介面
type PointsCalculableTransaction interface {
	GetAmount() int // 交易金額（TWD，整數，單位：元）
}

// calculateTotalPoints 計算交易列表的總積分（私有輔助方法）
// 職責：純計算邏輯，不涉及狀態變更或驗證
//
// 設計原則（SRP）：
// - 單一職責：只負責累加積分，不處理驗證或狀態變更
// - 可測試性：可獨立測試計算邏輯
// - 關注點分離：RecalculatePoints 負責編排，此方法負責計算
func (a *PointsAccount) calculateTotalPoints(
	transactions []PointsCalculableTransaction,
	calculator *PointsCalculationService,
	conversionRate ConversionRate,
) (int, error) {
	total := 0
	for _, tx := range transactions {
		amount := decimal.NewFromInt(int64(tx.GetAmount()))
		points, err := calculator.CalculateFromAmount(amount, conversionRate)
		if err != nil {
			return 0, err
		}
		total += points.Value()
	}
	return total, nil
}

// RecalculatePoints 重算累積積分（管理員觸發）
//
// 使用場景：
// 1. 轉換規則變更後重新計算所有帳戶
// 2. 修復數據不一致問題
// 3. 遷移舊數據
//
// 參數：
//   transactions - 該會員的所有已驗證交易
//   calculator - 積分計算服務（使用 *PointsCalculationService）
//   conversionRate - 使用的轉換率
//   reason - 重算原因（審計用途，如："rule_change"、"data_correction"、"migration"）
//
// 返回：
//   error - 如果重算後違反不變條件（earned < used）
//
// 業務規則：
// - 重算後的 earnedPoints 不能小於 usedPoints（不變條件）
// - 不發布 PointsEarned 事件（不是新增積分）
// - 發布 PointsRecalculated 事件（記錄重算操作，包含完整審計信息）
//
// 副作用：
// - 更新 earnedPoints
// - 更新 updatedAt
// - 發布 PointsRecalculatedEvent（含 reason 和 conversionRate）
func (a *PointsAccount) RecalculatePoints(
	transactions []PointsCalculableTransaction,
	calculator *PointsCalculationService,
	conversionRate ConversionRate,
	reason string,
) error {
	// 計算新的累積積分（委託給私有方法）
	newEarnedTotal, err := a.calculateTotalPoints(transactions, calculator, conversionRate)
	if err != nil {
		return err
	}

	// 業務規則檢查：創建並驗證新積分數量
	newEarnedPoints, err := NewPointsAmount(newEarnedTotal)
	if err != nil {
		return err
	}

	// 不變條件檢查：新的累積積分不能小於已使用積分
	if newEarnedPoints.LessThan(a.usedPoints) {
		return ErrInsufficientEarnedPoints.WithContext(
			"newEarned", newEarnedPoints.Value(),
			"used", a.usedPoints.Value(),
		)
	}

	// 狀態變更
	oldEarnedPoints := a.earnedPoints
	a.earnedPoints = newEarnedPoints
	a.updatedAt = time.Now()

	// 發布事件（含審計信息）
	a.addEvent(NewPointsRecalculatedEvent(
		a.accountID,
		oldEarnedPoints.Value(),
		newEarnedPoints.Value(),
		reason,                 // 重算原因
		conversionRate.Value(), // 使用的轉換率
		"",                     // triggeredBy 由 Application Layer 提供（未來實作）
	))

	return nil
}

// ===========================
// 聚合重建方法（僅供 Infrastructure Layer 使用）
// ===========================

// ReconstructPointsAccount 從持久化存儲重建聚合根
//
// 設計原則：
// - 僅供 Repository 使用，不對外暴露
// - 與 NewPointsAccount 的區別：
//   * New: 創建新聚合，執行完整驗證，發布 AccountCreated 事件
//   * Reconstruct: 重建已存在的聚合，不發布事件（事件已發生過）
//
// 參數：
//   accountID - 帳戶 ID（從資料庫讀取）
//   memberID - 會員 ID（從資料庫讀取）
//   earnedPoints - 累積獲得積分（原始 int 值）
//   usedPoints - 累積使用積分（原始 int 值）
//   createdAt - 創建時間
//   updatedAt - 最後更新時間
//
// 返回：
//   *PointsAccount - 重建的聚合根
//   error - 如果資料驗證失敗（資料損壞）
//
// 重要：即使是從資料庫重建，也必須驗證不變條件，防止損壞資料污染領域層
func ReconstructPointsAccount(
	accountID AccountID,
	memberID MemberID,
	earnedPoints int,
	usedPoints int,
	createdAt time.Time,
	updatedAt time.Time,
) (*PointsAccount, error) {
	// 1. 驗證 ID 有效性
	if accountID.IsEmpty() {
		return nil, ErrInvalidAccountID.WithContext(
			"reason", "invalid account ID in database",
		)
	}

	if memberID.IsEmpty() {
		return nil, ErrInvalidMemberID.WithContext(
			"reason", "invalid member ID in database",
		)
	}

	// 2. 驗證積分數量（防止負數）
	earnedAmount, err := NewPointsAmount(earnedPoints)
	if err != nil {
		return nil, ErrCorruptedEarnedPoints.WithContext(
			"value", earnedPoints,
			"underlying_error", err.Error(),
		)
	}

	usedAmount, err := NewPointsAmount(usedPoints)
	if err != nil {
		return nil, ErrCorruptedUsedPoints.WithContext(
			"value", usedPoints,
			"underlying_error", err.Error(),
		)
	}

	// 3. 驗證關鍵不變條件：usedPoints <= earnedPoints
	if usedAmount.GreaterThan(earnedAmount) {
		return nil, ErrInvariantViolation.WithContext(
			"usedPoints", usedPoints,
			"earnedPoints", earnedPoints,
		)
	}

	// 4. 重建聚合（使用已驗證的值對象）
	return &PointsAccount{
		accountID:    accountID,
		memberID:     memberID,
		earnedPoints: earnedAmount,
		usedPoints:   usedAmount,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		events:       make([]shared.DomainEvent, 0), // 重建時不包含事件
	}, nil
}

// ===========================
// 不變條件檢查（調試用）
// ===========================

// assertInvariants 斷言聚合根的不變條件（僅用於開發和調試）
//
// 設計原則：
// - 不變條件應該通過正確的業務邏輯來「預防」，而非「檢測」後 panic
// - 此方法僅用於開發階段的防禦性編程和調試
// - 生產環境中，不變條件違反表示代碼邏輯錯誤，應該在代碼審查時發現
//
// 不變條件：
// 1. EarnedPoints >= 0（由 PointsAmount 值對象保證）
// 2. UsedPoints >= 0（由 PointsAmount 值對象保證）
// 3. UsedPoints <= EarnedPoints（由業務方法的前置驗證保證）
//
// ⚠️ 注意：僅在開發/測試環境啟用，生產環境可通過 build tags 禁用
func (a *PointsAccount) assertInvariants() {
	// 此斷言僅用於調試，幫助發現代碼邏輯錯誤
	// 如果觸發，表示某個命令方法有 bug，需要修復該方法
	if a.usedPoints.GreaterThan(a.earnedPoints) {
		panic(fmt.Sprintf(
			"INVARIANT VIOLATION: usedPoints (%d) > earnedPoints (%d) for account %s - FIX THE BUG IN COMMAND METHOD",
			a.usedPoints.Value(),
			a.earnedPoints.Value(),
			a.accountID.String(),
		))
	}
}
