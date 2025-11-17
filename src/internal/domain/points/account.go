package points

import (
	"fmt"
	"time"

	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
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
