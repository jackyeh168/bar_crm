# Domain Layer 實現指南

> **版本**: 1.0
> **最後更新**: 2025-01-10
> **原則**: 純粹的業務邏輯，無外部依賴

---

## 1. Domain Layer 概述

### 1.1 職責定義

**Domain Layer 的核心職責**:
- ✅ 封裝業務邏輯與業務規則
- ✅ 定義聚合根、實體、值對象
- ✅ 定義領域服務（跨聚合的業務邏輯）
- ✅ 定義 Repository 接口（由 Infrastructure 實現）
- ✅ 發布領域事件（狀態變更的通知）
- ❌ 不包含技術實現細節（無 GORM, HTTP, Redis）

### 1.2 設計原則

**SOLID 原則應用**:
1. **SRP (單一職責原則)**: 每個聚合只負責一個業務概念
2. **OCP (開閉原則)**: 對擴展開放，對修改封閉（使用策略模式）
3. **LSP (里氏替換原則)**: 子類型可以替換父類型
4. **ISP (接口隔離原則)**: Repository 接口按使用場景拆分
5. **DIP (依賴反轉原則)**: 依賴接口而非實現

**充血模型 (Rich Domain Model)**:
```go
// ✅ 正確：業務邏輯封裝在聚合內部
type PointsAccount struct {
    accountID    AccountID
    earnedPoints PointsAmount
    usedPoints   PointsAmount
}

func (a *PointsAccount) EarnPoints(amount PointsAmount, ...) error {
    if amount.Value() < 0 {
        return ErrNegativePointsAmount  // 業務規則檢查
    }
    a.earnedPoints = a.earnedPoints.Add(amount)
    a.publishEvent(PointsEarned{...})  // 發布事件
    return nil
}

// ❌ 錯誤：貧血模型（業務邏輯在外部）
type PointsAccount struct {
    AccountID string
    EarnedPoints int  // 公開字段，無封裝
    UsedPoints int
}

// 業務邏輯在 Service 層
func (s *PointsService) EarnPoints(account *PointsAccount, amount int) {
    account.EarnedPoints += amount  // 無業務規則檢查
}
```

---

## 2. 聚合根實現

### 2.1 聚合根結構

**文件**: `internal/domain/points/account.go`

```go
package points

import (
    "time"
    "github.com/shopspring/decimal"
    "github.com/yourorg/bar_crm/internal/domain/shared"
)

// PointsAccount 積分帳戶聚合根
// 設計原則：輕量級聚合，不包含無界集合
type PointsAccount struct {
    // 私有字段（封裝）
    accountID    AccountID
    memberID     MemberID  // 引用其他聚合（使用 ID，非對象）
    earnedPoints PointsAmount
    usedPoints   PointsAmount
    lastUpdatedAt time.Time
    version      int  // 樂觀鎖版本號

    // 領域事件（待發布）
    events []shared.DomainEvent
}

// NewPointsAccount 構造函數（工廠方法）
// 所有聚合必須通過構造函數創建，確保初始狀態有效
func NewPointsAccount(memberID MemberID) (*PointsAccount, error) {
    // 驗證必填字段
    if memberID.IsEmpty() {
        return nil, ErrInvalidMemberID
    }

    // 生成聚合根 ID
    accountID := NewAccountID()

    // 初始狀態（使用 unchecked 版本，因為 0 保證有效）
    account := &PointsAccount{
        accountID:    accountID,
        memberID:     memberID,
        earnedPoints: newPointsAmountUnchecked(0),
        usedPoints:   newPointsAmountUnchecked(0),
        lastUpdatedAt: time.Now(),
        version:      1,
        events:       []shared.DomainEvent{},
    }

    // 發布創建事件
    account.publishEvent(PointsAccountCreated{
        AccountID: accountID,
        MemberID:  memberID,
        OccurredAt: time.Now(),
    })

    return account, nil
}

// --- 命令方法（狀態變更）---

// EarnPoints 獲得積分（核心業務邏輯）
func (a *PointsAccount) EarnPoints(
    amount PointsAmount,
    source PointsSource,
    sourceID string,
    description string,
) error {
    // 前置條件檢查（不變性保護）
    if amount.Value() < 0 {
        return ErrNegativePointsAmount
    }

    // 狀態變更
    a.earnedPoints = a.earnedPoints.Add(amount)
    a.lastUpdatedAt = time.Now()
    a.version++  // 聚合自己控制版本號（樂觀鎖）

    // 發布領域事件（業務邏輯副作用）
    event := NewPointsEarnedEvent(a.accountID, amount, source, sourceID, description)
    a.publishEvent(event)

    return nil
}

// DeductPoints 扣除積分（V3.2+ 兌換功能）
func (a *PointsAccount) DeductPoints(amount PointsAmount, reason string) error {
    // 前置條件檢查
    if amount.Value() < 0 {
        return ErrNegativePointsAmount
    }

    // 業務規則檢查
    if !a.HasSufficientPoints(amount) {
        return ErrInsufficientPoints
    }

    // 狀態變更
    a.usedPoints = a.usedPoints.Add(amount)
    a.lastUpdatedAt = time.Now()
    a.version++  // 聚合自己控制版本號（樂觀鎖）

    // 發布事件
    event := NewPointsDeductedEvent(a.accountID, amount, reason)
    a.publishEvent(event)

    return nil
}

// RecalculatePoints 重算累積積分（管理員觸發）
// 使用 Domain Service 計算，聚合負責狀態更新
func (a *PointsAccount) RecalculatePoints(
    transactions []PointsCalculableTransaction,
    calculator PointsCalculationService,
) error {
    // 計算新的累積積分（委託給 Domain Service）
    newEarnedPoints := 0
    for _, tx := range transactions {
        points := calculator.CalculateForTransaction(tx)
        newEarnedPoints += points.Value()
    }

    // 業務規則檢查：創建並驗證新積分數量
    newAmount, err := NewPointsAmount(newEarnedPoints)
    if err != nil {
        return err  // 負數積分錯誤
    }

    if newAmount.Value() < a.usedPoints.Value() {
        return ErrInsufficientEarnedPoints
    }

    // 狀態變更
    oldPoints := a.earnedPoints
    a.earnedPoints = newAmount
    a.lastUpdatedAt = time.Now()
    a.version++  // 聚合自己控制版本號（樂觀鎖）

    // 發布事件
    event := NewPointsRecalculatedEvent(a.accountID, oldPoints.Value(), newEarnedPoints)
    a.publishEvent(event)

    return nil
}

// --- 查詢方法（無狀態變更）---

// GetAvailablePoints 獲取可用積分（計算屬性）
// 使用 unchecked 版本，因為聚合不變性保證 earnedPoints >= usedPoints
// 如果不變條件被違反（數據損壞），subtractUnchecked 會 panic
func (a *PointsAccount) GetAvailablePoints() PointsAmount {
    // 防禦性檢查：在調用 subtractUnchecked 前驗證不變條件
    // 如果違反，提供更清晰的錯誤信息（包含帳戶 ID）
    if a.usedPoints.Value() > a.earnedPoints.Value() {
        panic(fmt.Sprintf("invariant violation: used points (%d) > earned points (%d) for account %s",
            a.usedPoints.Value(), a.earnedPoints.Value(), a.accountID.Value()))
    }
    return a.earnedPoints.subtractUnchecked(a.usedPoints)
}

// HasSufficientPoints 檢查積分是否足夠
func (a *PointsAccount) HasSufficientPoints(amount PointsAmount) bool {
    return a.GetAvailablePoints().Value() >= amount.Value()
}

// GetAccountID 獲取帳戶 ID
func (a *PointsAccount) GetAccountID() AccountID {
    return a.accountID
}

// GetMemberID 獲取會員 ID
func (a *PointsAccount) GetMemberID() MemberID {
    return a.memberID
}

// GetEarnedPoints 獲取累積積分
func (a *PointsAccount) GetEarnedPoints() PointsAmount {
    return a.earnedPoints
}

// GetUsedPoints 獲取已使用積分
func (a *PointsAccount) GetUsedPoints() PointsAmount {
    return a.usedPoints
}

// GetLastUpdatedAt 獲取最後更新時間
func (a *PointsAccount) GetLastUpdatedAt() time.Time {
    return a.lastUpdatedAt
}

// GetVersion 獲取版本號（樂觀鎖）
func (a *PointsAccount) GetVersion() int {
    return a.version
}

// GetPreviousVersion 獲取上一個版本號（用於樂觀鎖檢查）
// Repository 在 Update 時使用此方法獲取 WHERE 條件中的版本號
// 這避免了 Repository 需要計算 version - 1 的職責洩漏
func (a *PointsAccount) GetPreviousVersion() int {
    if a.version <= 1 {
        return 1  // 新創建的聚合，上一個版本就是 1
    }
    return a.version - 1
}

// --- 聚合重建方法（僅供 Infrastructure Layer 使用）---

// ReconstructPointsAccount 從持久化存儲重建聚合根
// 注意：此方法僅供 Repository 使用
// 重要：即使是從資料庫重建，也必須驗證不變條件，防止損壞資料污染領域層
// 重建的聚合不包含領域事件（事件已發布過）
func ReconstructPointsAccount(
    accountID AccountID,
    memberID MemberID,
    earnedPoints int,
    usedPoints int,
    version int,
    lastUpdatedAt time.Time,
) (*PointsAccount, error) {
    // 1. 驗證積分數量（防止負數）
    earnedAmount, err := NewPointsAmount(earnedPoints)
    if err != nil {
        return nil, fmt.Errorf("invalid earned points in database: %w", err)
    }

    usedAmount, err := NewPointsAmount(usedPoints)
    if err != nil {
        return nil, fmt.Errorf("invalid used points in database: %w", err)
    }

    // 2. 驗證關鍵不變條件：usedPoints <= earnedPoints
    if usedAmount.Value() > earnedAmount.Value() {
        return nil, fmt.Errorf("data corruption: used points (%d) exceeds earned points (%d)",
            usedPoints, earnedPoints)
    }

    // 3. 驗證版本號
    if version < 1 {
        return nil, fmt.Errorf("invalid version in database: %d", version)
    }

    // 4. 重建聚合（使用已驗證的值對象）
    return &PointsAccount{
        accountID:     accountID,
        memberID:      memberID,
        earnedPoints:  earnedAmount,
        usedPoints:    usedAmount,
        version:       version,
        lastUpdatedAt: lastUpdatedAt,
        events:        []shared.DomainEvent{},  // 重建時不包含事件
    }, nil
}

// --- 領域事件管理 ---

// GetEvents 獲取待發布的事件
func (a *PointsAccount) GetEvents() []shared.DomainEvent {
    return a.events
}

// ClearEvents 清空事件（發布後調用）
func (a *PointsAccount) ClearEvents() {
    a.events = []shared.DomainEvent{}
}

// publishEvent 發布事件（私有方法）
func (a *PointsAccount) publishEvent(event shared.DomainEvent) {
    a.events = append(a.events, event)
}

// --- PointsCalculableTransaction 接口定義（用於解耦）---

// PointsCalculableTransaction 可計算積分的交易接口
// 設計原則：接口名稱表達用途（積分計算），而非數據結構
// Application Layer 的 DTO 實現此接口
type PointsCalculableTransaction interface {
    GetTransactionAmount() decimal.Decimal
    GetTransactionDate() time.Time
    HasCompletedSurvey() bool
}

// --- PointsCalculationService 接口定義 ---

// PointsCalculationService 積分計算服務接口
type PointsCalculationService interface {
    CalculateForTransaction(tx PointsCalculableTransaction) PointsAmount
}
```

### 2.2 設計原則總結

**聚合根設計原則**:
1. ✅ **輕量級聚合**: 不包含無界集合（`transactions` 通過獨立 Repository 查詢）
2. ✅ **封裝性**: 所有字段私有，只通過方法訪問
3. ✅ **不變性保護**: 狀態變更前檢查業務規則
4. ✅ **Tell, Don't Ask**: 告訴聚合做什麼，而非獲取數據後在外部處理
5. ✅ **單一職責**: 只管理積分狀態，不負責計算邏輯（委託給 Domain Service）
6. ✅ **領域事件**: 所有狀態變更發布事件
7. ✅ **版本控制**: 樂觀鎖支持併發控制

---

## 3. 值對象實現

### 3.1 值對象結構

**文件**: `internal/domain/points/value_objects.go`

```go
package points

import (
    "errors"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "time"
)

// --- AccountID 值對象 ---

// AccountID 帳戶 ID
type AccountID struct {
    value string
}

// NewAccountID 生成新的帳戶 ID
func NewAccountID() AccountID {
    return AccountID{value: uuid.New().String()}
}

// AccountIDFromString 從字符串創建 AccountID
func AccountIDFromString(value string) (AccountID, error) {
    if value == "" {
        return AccountID{}, errors.New("account ID cannot be empty")
    }
    // 驗證 UUID 格式
    if _, err := uuid.Parse(value); err != nil {
        return AccountID{}, errors.New("invalid account ID format")
    }
    return AccountID{value: value}, nil
}

// String 返回字符串表示
func (id AccountID) String() string {
    return id.value
}

// Equals 判斷相等性
func (id AccountID) Equals(other AccountID) bool {
    return id.value == other.value
}

// IsEmpty 判斷是否為空
func (id AccountID) IsEmpty() bool {
    return id.value == ""
}

// --- PointsAmount 值對象 ---

// PointsAmount 積分數量
// 設計原則：不可變、包含驗證邏輯
type PointsAmount struct {
    value int
}

// NewPointsAmount 創建積分數量（帶驗證）
// 返回錯誤而非 panic，符合 Go 慣用法和錯誤處理原則
func NewPointsAmount(value int) (PointsAmount, error) {
    // 驗證：積分不可為負數
    if value < 0 {
        return PointsAmount{}, ErrNegativePointsAmount
    }
    return PointsAmount{value: value}, nil
}

// newPointsAmountUnchecked 創建積分數量（無驗證）
// 僅供內部算術操作使用（調用方已保證有效性）
func newPointsAmountUnchecked(value int) PointsAmount {
    return PointsAmount{value: value}
}

// Value 獲取值
func (p PointsAmount) Value() int {
    return p.value
}

// Add 相加（不可變操作，返回新對象）
// 內部操作保證結果有效，使用 unchecked 版本提升性能
func (p PointsAmount) Add(other PointsAmount) PointsAmount {
    return newPointsAmountUnchecked(p.value + other.value)
}

// Subtract 相減（不可變操作，返回錯誤而非靜默截斷）
// 透明的錯誤處理：調用方必須明確處理負數情況
func (p PointsAmount) Subtract(other PointsAmount) (PointsAmount, error) {
    result := p.value - other.value
    if result < 0 {
        return PointsAmount{}, ErrNegativePointsAmount
    }
    return newPointsAmountUnchecked(result), nil
}

// subtractUnchecked 相減（無驗證，假設調用方已保證有效性）
// 僅供調用方已驗證有效性的場景使用（例如 GetAvailablePoints）
// 如果結果為負數，說明不變條件被違反，直接 panic（程序錯誤，非業務錯誤）
func (p PointsAmount) subtractUnchecked(other PointsAmount) PointsAmount {
    result := p.value - other.value
    if result < 0 {
        // 不變條件違反：這是程序錯誤，必須立即暴露
        panic(fmt.Sprintf("subtractUnchecked: invariant violation (%d - %d < 0)",
            p.value, other.value))
    }
    return newPointsAmountUnchecked(result)
}

// Equals 判斷相等性
func (p PointsAmount) Equals(other PointsAmount) bool {
    return p.value == other.value
}

// IsZero 判斷是否為零
func (p PointsAmount) IsZero() bool {
    return p.value == 0
}

// --- ConversionRate 值對象 ---

// ConversionRate 轉換率（例如 100 元 = 1 點）
// 業務規則：範圍 1-1000
type ConversionRate struct {
    value int
}

// NewConversionRate 創建轉換率
func NewConversionRate(value int) (ConversionRate, error) {
    // 業務規則驗證
    if value < 1 || value > 1000 {
        return ConversionRate{}, ErrInvalidConversionRate
    }
    return ConversionRate{value: value}, nil
}

// Value 獲取值
func (r ConversionRate) Value() int {
    return r.value
}

// CalculatePoints 計算積分（核心業務邏輯）
// 使用 unchecked 版本，因為計算結果保證 >= 0（floor 向下取整）
func (r ConversionRate) CalculatePoints(amount decimal.Decimal) PointsAmount {
    // 積分 = floor(金額 / 轉換率)
    rate := decimal.NewFromInt(int64(r.value))
    points := amount.Div(rate).Floor().IntPart()
    // floor 結果保證 >= 0，無需驗證
    return newPointsAmountUnchecked(int(points))
}

// Equals 判斷相等性
func (r ConversionRate) Equals(other ConversionRate) bool {
    return r.value == other.value
}

// --- DateRange 值對象 ---

// DateRange 日期範圍
// 業務規則：開始日期 <= 結束日期
type DateRange struct {
    startDate time.Time
    endDate   time.Time
}

// NewDateRange 創建日期範圍
func NewDateRange(startDate, endDate time.Time) (DateRange, error) {
    // 業務規則驗證
    if startDate.After(endDate) {
        return DateRange{}, ErrInvalidDateRange
    }
    return DateRange{
        startDate: startDate,
        endDate:   endDate,
    }, nil
}

// StartDate 獲取開始日期
func (dr DateRange) StartDate() time.Time {
    return dr.startDate
}

// EndDate 獲取結束日期
func (dr DateRange) EndDate() time.Time {
    return dr.endDate
}

// Contains 判斷日期是否在範圍內
func (dr DateRange) Contains(date time.Time) bool {
    return !date.Before(dr.startDate) && !date.After(dr.endDate)
}

// Overlaps 判斷是否與另一範圍重疊
func (dr DateRange) Overlaps(other DateRange) bool {
    return dr.startDate.Before(other.endDate) && other.startDate.Before(dr.endDate)
}

// --- PointsSource 枚舉 ---

// PointsSource 積分來源
type PointsSource int

const (
    PointsSourceInvoice PointsSource = iota  // 發票
    PointsSourceSurvey                       // 問卷
    PointsSourceRedemption                   // 兌換（V3.2+）
    PointsSourceExpiration                   // 過期（V3.3+）
    PointsSourceTransfer                     // 轉讓（V4.0+）
)

// String 返回字符串表示
func (s PointsSource) String() string {
    switch s {
    case PointsSourceInvoice:
        return "invoice"
    case PointsSourceSurvey:
        return "survey"
    case PointsSourceRedemption:
        return "redemption"
    case PointsSourceExpiration:
        return "expiration"
    case PointsSourceTransfer:
        return "transfer"
    default:
        return "unknown"
    }
}

// --- MemberID 值對象 ---

// MemberID 會員 ID（跨上下文引用）
type MemberID struct {
    value string
}

// NewMemberID 創建會員 ID
func NewMemberID(value string) (MemberID, error) {
    if value == "" {
        return MemberID{}, errors.New("member ID cannot be empty")
    }
    return MemberID{value: value}, nil
}

// String 返回字符串表示
func (id MemberID) String() string {
    return id.value
}

// Equals 判斷相等性
func (id MemberID) Equals(other MemberID) bool {
    return id.value == other.value
}

// IsEmpty 判斷是否為空
func (id MemberID) IsEmpty() bool {
    return id.value == ""
}
```

### 3.2 值對象設計原則

**值對象的特性**:
1. ✅ **不可變性**: 所有字段私有，無 setter 方法
2. ✅ **構造時驗證**: 通過構造函數確保對象始終有效
3. ✅ **值相等性**: 基於值而非引用判斷相等
4. ✅ **無標識符**: 不需要 ID，通過值識別
5. ✅ **封裝業務邏輯**: 例如 `ConversionRate.CalculatePoints()`
6. ✅ **自描述**: 類型名稱明確表達業務概念

**何時使用值對象？**:
- ✅ 基本類型需要業務規則驗證（電話號碼、Email）
- ✅ 基本類型需要封裝業務邏輯（Money 的加減）
- ✅ 組合多個字段形成業務概念（DateRange, Address）
- ✅ 無需唯一標識符的業務概念

**錯誤處理模式**:
```go
// ✅ 正確：公開構造函數返回錯誤
func NewPointsAmount(value int) (PointsAmount, error) {
    if value < 0 {
        return PointsAmount{}, ErrNegativePointsAmount
    }
    return PointsAmount{value: value}, nil
}

// ✅ 正確：內部 unchecked 構造函數（提升性能）
func newPointsAmountUnchecked(value int) PointsAmount {
    return PointsAmount{value: value}
}

// ✅ 正確：操作返回錯誤（透明處理）
func (p PointsAmount) Subtract(other PointsAmount) (PointsAmount, error) {
    result := p.value - other.value
    if result < 0 {
        return PointsAmount{}, ErrNegativePointsAmount
    }
    return newPointsAmountUnchecked(result), nil
}

// ❌ 錯誤：使用 panic（違反錯誤處理原則）
func NewPointsAmount(value int) PointsAmount {
    if value < 0 {
        panic("invalid value")  // 不要這樣做
    }
    return PointsAmount{value: value}
}

// ❌ 錯誤：靜默截斷（隱藏錯誤）
func (p PointsAmount) Subtract(other PointsAmount) PointsAmount {
    result := p.value - other.value
    if result < 0 {
        return PointsAmount{value: 0}  // 調用方無法知道發生了截斷
    }
    return PointsAmount{value: result}
}
```

---

## 4. 領域服務實現

### 4.1 領域服務定義

**何時使用領域服務？**:
- ✅ 跨多個聚合的業務邏輯
- ✅ 複雜的計算邏輯（Strategy Pattern）
- ✅ 無狀態的純函數邏輯
- ❌ 不用於協調多個聚合（那是 Application Layer 的職責）

**文件**: `internal/domain/points/calculation_service.go`

```go
package points

import (
    "github.com/shopspring/decimal"
    "time"
)

// PointsCalculationService 積分計算服務（領域服務）
// 職責：計算積分的業務邏輯
// 設計模式：Strategy Pattern（可擴展多種計算策略）
type PointsCalculationService interface {
    CalculateForTransaction(tx PointsCalculableTransaction) PointsAmount
}

// CompositePointsCalculator 組合積分計算器
// 設計模式：Composite Pattern
// 優勢：符合 OCP（開閉原則），新增策略無需修改現有代碼
type CompositePointsCalculator struct {
    strategies       []PointsCalculationStrategy
    conversionRuleService ConversionRuleService  // 查詢轉換規則
}

// NewCompositePointsCalculator 創建組合計算器
func NewCompositePointsCalculator(
    strategies []PointsCalculationStrategy,
    ruleService ConversionRuleService,
) *CompositePointsCalculator {
    return &CompositePointsCalculator{
        strategies:       strategies,
        conversionRuleService: ruleService,
    }
}

// CalculateForTransaction 計算單筆交易的積分
func (c *CompositePointsCalculator) CalculateForTransaction(
    tx PointsCalculableTransaction,
) PointsAmount {
    totalPoints := 0

    // 執行所有策略，累加積分
    for _, strategy := range c.strategies {
        points := strategy.Calculate(tx, c.conversionRuleService)
        totalPoints += points.Value()
    }

    // 使用 unchecked 版本，因為策略返回的積分保證 >= 0
    return newPointsAmountUnchecked(totalPoints)
}

// --- 策略接口 ---

// PointsCalculationStrategy 積分計算策略接口
type PointsCalculationStrategy interface {
    Calculate(tx PointsCalculableTransaction, ruleService ConversionRuleService) PointsAmount
}

// --- 基礎積分策略 ---

// BasePointsCalculator 基礎積分計算器
type BasePointsCalculator struct{}

func (b *BasePointsCalculator) Calculate(
    tx PointsCalculableTransaction,
    ruleService ConversionRuleService,
) PointsAmount {
    // 1. 查詢適用的轉換規則
    rule, err := ruleService.GetRuleForDate(tx.GetTransactionDate())
    if err != nil {
        return newPointsAmountUnchecked(0)  // 找不到規則，返回 0
    }

    // 2. 計算基礎積分
    amount := tx.GetTransactionAmount()
    points := rule.GetConversionRate().CalculatePoints(amount)

    return points
}

// --- 問卷獎勵策略 ---

// SurveyBonusCalculator 問卷獎勵計算器
type SurveyBonusCalculator struct{}

func (s *SurveyBonusCalculator) Calculate(
    tx PointsCalculableTransaction,
    ruleService ConversionRuleService,
) PointsAmount {
    // 完成問卷獎勵 +1 點
    if tx.HasCompletedSurvey() {
        return newPointsAmountUnchecked(1)
    }
    return newPointsAmountUnchecked(0)
}

// --- 未來擴展策略 (V3.4+) ---

// TierBonusCalculator 會員等級加成計算器
// 預留接口，V3.4 實現
type TierBonusCalculator struct {
    // 未來實現: 根據會員等級添加加成
}

// --- ConversionRuleService 接口 ---

// ConversionRuleService 轉換規則服務接口
type ConversionRuleService interface {
    GetRuleForDate(date time.Time) (*ConversionRule, error)
}
```

### 4.2 領域服務設計原則

**領域服務設計原則**:
1. ✅ **無狀態**: 領域服務不持有狀態，只包含行為
2. ✅ **純函數**: 相同輸入產生相同輸出
3. ✅ **策略模式**: 使用接口和組合模式支持擴展
4. ✅ **依賴接口**: 依賴其他領域服務的接口，而非實現
5. ✅ **單一職責**: 每個服務只負責一種業務邏輯

---

## 5. Repository 接口定義

### 5.1 Repository 接口

**文件**: `internal/domain/points/repository/account_repository.go`

```go
package repository

import (
    "github.com/yourorg/bar_crm/internal/domain/points"
    "github.com/yourorg/bar_crm/internal/domain/shared"
)

// ===========================
// 接口隔離原則（ISP）設計
// ===========================
// 按職責拆分接口，Use Case 只依賴需要的接口，而不是依賴完整的 Repository
//
// 優勢：
// 1. 降低耦合度 - Use Case 只依賴使用的方法
// 2. 提高可測試性 - Mock 更簡單
// 3. 符合 SOLID 原則 - 接口隔離原則（ISP）
// 4. 清晰的職責劃分 - 讀寫分離

// PointsAccountWriter 寫操作接口
// Use Case 只需要寫操作時，依賴此接口
type PointsAccountWriter interface {
    // Create 創建新的積分帳戶
    Create(ctx shared.TransactionContext, account *points.PointsAccount) error

    // Update 更新積分帳戶（支持樂觀鎖）
    Update(ctx shared.TransactionContext, account *points.PointsAccount) error
}

// PointsAccountReader 查詢操作接口
// Use Case 只需要讀操作時，依賴此接口
type PointsAccountReader interface {
    // FindByID 根據帳戶 ID 查詢
    FindByID(ctx shared.TransactionContext, accountID points.AccountID) (*points.PointsAccount, error)

    // FindByMemberID 根據會員 ID 查詢
    FindByMemberID(ctx shared.TransactionContext, memberID points.MemberID) (*points.PointsAccount, error)

    // ExistsByMemberID 檢查會員是否已有積分帳戶
    ExistsByMemberID(ctx shared.TransactionContext, memberID points.MemberID) (bool, error)
}

// PointsAccountBatchReader 批次查詢接口
// Use Case 需要批次操作時，依賴此接口
type PointsAccountBatchReader interface {
    // FindAll 查詢所有積分帳戶（分頁參數可選）
    FindAll(ctx shared.TransactionContext) ([]*points.PointsAccount, error)

    // FindByIDs 批次查詢（根據 ID 列表）
    FindByIDs(ctx shared.TransactionContext, accountIDs []points.AccountID) ([]*points.PointsAccount, error)
}

// PointsAccountRepository 完整的倉儲接口（組合所有小接口）
// Infrastructure 實現此接口，Use Case 只依賴需要的小接口
type PointsAccountRepository interface {
    PointsAccountWriter
    PointsAccountReader
    PointsAccountBatchReader
}

// 倉儲錯誤（屬於 Domain Layer）
var (
    ErrAccountNotFound        = errors.New("points account not found")
    ErrAccountAlreadyExists   = errors.New("points account already exists")
    ErrConcurrentModification = errors.New("concurrent modification detected")
)
```

### 5.2 Repository 設計原則

**Repository 接口設計原則**:
1. ✅ **接口屬於 Domain**: Repository 接口定義在 Domain Layer
2. ✅ **只依賴 Domain 實體**: 參數和返回值只使用 Domain 對象
3. ✅ **Transaction Context 傳遞**: 使用 `shared.TransactionContext` 支持事務
4. ✅ **錯誤屬於 Domain**: Repository 錯誤定義在 Domain Layer
5. ✅ **接口隔離**: 按使用場景拆分接口（Writer vs Reader）

**錯誤處理**:
```go
// ✅ 正確：Repository 返回 Domain 錯誤
func (r *GormPointsAccountRepository) FindByID(...) (*points.PointsAccount, error) {
    // ...
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, repository.ErrAccountNotFound  // 轉換為 Domain 錯誤
    }
    return nil, err
}

// ❌ 錯誤：洩漏 Infrastructure 錯誤
func (r *GormPointsAccountRepository) FindByID(...) (*points.PointsAccount, error) {
    // ...
    return nil, gorm.ErrRecordNotFound  // 洩漏 GORM 錯誤到上層
}
```

### 5.3 使用接口隔離的 Use Case 設計

#### 範例 1：只需要讀操作的 Use Case

```go
// internal/application/usecases/points/get_points_balance.go
package points

import (
    "github.com/yourorg/bar_crm/internal/domain/points/repository"
    "github.com/yourorg/bar_crm/internal/domain/points"
)

// GetPointsBalanceUseCase 查詢積分餘額（只讀操作）
type GetPointsBalanceUseCase struct {
    // ✅ 只依賴 Reader 接口，不依賴完整的 Repository
    accountReader repository.PointsAccountReader
}

func NewGetPointsBalanceUseCase(
    accountReader repository.PointsAccountReader,  // 只需要 Reader
) *GetPointsBalanceUseCase {
    return &GetPointsBalanceUseCase{
        accountReader: accountReader,
    }
}

func (uc *GetPointsBalanceUseCase) Execute(memberID points.MemberID) (int, error) {
    account, err := uc.accountReader.FindByMemberID(ctx, memberID)
    if err != nil {
        return 0, err
    }
    return account.GetAvailablePoints().Value(), nil
}
```

**優勢**：
- ✅ Use Case 只依賴讀接口，明確表達只做查詢
- ✅ Mock 更簡單（只需 mock Reader，不需 mock Writer）
- ✅ 防止誤用（無法調用寫方法）

#### 範例 2：需要讀寫操作的 Use Case

```go
// internal/application/usecases/points/earn_points.go
package points

type EarnPointsUseCase struct {
    // ✅ 讀操作依賴 Reader 接口
    accountReader repository.PointsAccountReader
    // ✅ 寫操作依賴 Writer 接口
    accountWriter repository.PointsAccountWriter
}

func NewEarnPointsUseCase(
    accountReader repository.PointsAccountReader,
    accountWriter repository.PointsAccountWriter,
) *EarnPointsUseCase {
    return &EarnPointsUseCase{
        accountReader: accountReader,
        accountWriter: accountWriter,
    }
}

func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    // 1. 讀取聚合
    account, err := uc.accountReader.FindByMemberID(ctx, cmd.MemberID)
    if err != nil {
        return err
    }

    // 2. 執行業務邏輯
    err = account.EarnPoints(cmd.Amount, cmd.Source, cmd.SourceID, cmd.Description)
    if err != nil {
        return err
    }

    // 3. 持久化
    return uc.accountWriter.Update(ctx, account)
}
```

**優勢**：
- ✅ 讀寫職責分離，更清晰
- ✅ 可以注入不同的實現（例如：讀從緩存，寫到數據庫）
- ✅ Mock 更靈活（可以分別 mock Reader 和 Writer）

#### 範例 3：Infrastructure 實現完整接口

```go
// internal/infrastructure/persistence/points/gorm_account_repository.go
package points

import (
    "github.com/yourorg/bar_crm/internal/domain/points/repository"
    "github.com/yourorg/bar_crm/internal/domain/points"
)

// GormPointsAccountRepository 實現完整的 Repository 接口
// 一個實現滿足所有小接口
type GormPointsAccountRepository struct {
    db *gorm.DB
}

// 確保 GormPointsAccountRepository 實現了所有接口
var (
    _ repository.PointsAccountWriter      = (*GormPointsAccountRepository)(nil)
    _ repository.PointsAccountReader      = (*GormPointsAccountRepository)(nil)
    _ repository.PointsAccountBatchReader = (*GormPointsAccountRepository)(nil)
    _ repository.PointsAccountRepository  = (*GormPointsAccountRepository)(nil)
)

// Create 實現 PointsAccountWriter 接口
func (r *GormPointsAccountRepository) Create(...) error {
    // ...
}

// Update 實現 PointsAccountWriter 接口
func (r *GormPointsAccountRepository) Update(...) error {
    // ...
}

// FindByID 實現 PointsAccountReader 接口
func (r *GormPointsAccountRepository) FindByID(...) (*points.PointsAccount, error) {
    // ...
}

// FindByMemberID 實現 PointsAccountReader 接口
func (r *GormPointsAccountRepository) FindByMemberID(...) (*points.PointsAccount, error) {
    // ...
}

// ExistsByMemberID 實現 PointsAccountReader 接口
func (r *GormPointsAccountRepository) ExistsByMemberID(...) (bool, error) {
    // ...
}

// FindAll 實現 PointsAccountBatchReader 接口
func (r *GormPointsAccountRepository) FindAll(...) ([]*points.PointsAccount, error) {
    // ...
}

// FindByIDs 實現 PointsAccountBatchReader 接口
func (r *GormPointsAccountRepository) FindByIDs(...) ([]*points.PointsAccount, error) {
    // ...
}
```

#### 範例 4：依賴注入配置

```go
// cmd/app/main.go
func main() {
    fx.New(
        // 1. 提供 Repository 實現（單例）
        fx.Provide(func(db *gorm.DB) *points.GormPointsAccountRepository {
            return points.NewGormPointsAccountRepository(db)
        }),

        // 2. 將實現註冊為多個接口
        fx.Provide(
            fx.Annotate(
                func(repo *points.GormPointsAccountRepository) repository.PointsAccountReader {
                    return repo
                },
            ),
            fx.Annotate(
                func(repo *points.GormPointsAccountRepository) repository.PointsAccountWriter {
                    return repo
                },
            ),
            fx.Annotate(
                func(repo *points.GormPointsAccountRepository) repository.PointsAccountBatchReader {
                    return repo
                },
            ),
        ),

        // 3. Use Case 自動注入需要的接口
        fx.Provide(
            usecases.NewGetPointsBalanceUseCase,  // 自動注入 Reader
            usecases.NewEarnPointsUseCase,        // 自動注入 Reader + Writer
        ),
    ).Run()
}
```

### 5.4 接口隔離的優勢總結

| 方面 | 單一大接口 ❌ | 接口隔離（ISP）✅ |
|------|--------------|-----------------|
| **依賴清晰度** | 不清楚 Use Case 使用哪些方法 | 一眼看出依賴的操作類型 |
| **測試複雜度** | Mock 需要實現所有方法 | 只 Mock 需要的接口 |
| **錯誤使用防護** | 可能誤調用不需要的方法 | 編譯期阻止誤用 |
| **讀寫分離** | 混在一起 | 清晰的讀寫分離 |
| **擴展性** | 添加方法影響所有依賴者 | 只影響使用該子接口的依賴者 |

**參考範例**：Audit Context 的接口設計（`01-directory-structure.md` L97-101）也遵循了相同的接口隔離原則。

---

## 6. 領域事件實現

### 6.1 領域事件定義

**文件**: `internal/domain/points/events.go`

```go
package points

import (
    "time"
    "github.com/google/uuid"
    "github.com/yourorg/bar_crm/internal/domain/shared"
)

// --- PointsEarned 事件 ---

// PointsEarned 積分已獲得事件
type PointsEarned struct {
    eventID     string
    accountID   AccountID
    amount      PointsAmount
    source      PointsSource
    sourceID    string
    description string
    occurredAt  time.Time
}

// NewPointsEarnedEvent 創建積分已獲得事件
func NewPointsEarnedEvent(
    accountID AccountID,
    amount PointsAmount,
    source PointsSource,
    sourceID string,
    description string,
) PointsEarned {
    return PointsEarned{
        eventID:     uuid.New().String(),
        accountID:   accountID,
        amount:      amount,
        source:      source,
        sourceID:    sourceID,
        description: description,
        occurredAt:  time.Now(),
    }
}

// 實現 DomainEvent 接口
func (e PointsEarned) EventID() string {
    return e.eventID
}

func (e PointsEarned) EventType() string {
    return "points.earned"
}

func (e PointsEarned) OccurredAt() time.Time {
    return e.occurredAt
}

func (e PointsEarned) AggregateID() string {
    return e.accountID.String()
}

// Getters
func (e PointsEarned) AccountID() AccountID   { return e.accountID }
func (e PointsEarned) Amount() PointsAmount   { return e.amount }
func (e PointsEarned) Source() PointsSource   { return e.source }
func (e PointsEarned) SourceID() string       { return e.sourceID }
func (e PointsEarned) Description() string    { return e.description }

// --- PointsDeducted 事件 ---

// PointsDeducted 積分已扣除事件
type PointsDeducted struct {
    eventID    string
    accountID  AccountID
    amount     PointsAmount
    reason     string
    occurredAt time.Time
}

// NewPointsDeductedEvent 創建積分已扣除事件
func NewPointsDeductedEvent(
    accountID AccountID,
    amount PointsAmount,
    reason string,
) PointsDeducted {
    return PointsDeducted{
        eventID:    uuid.New().String(),
        accountID:  accountID,
        amount:     amount,
        reason:     reason,
        occurredAt: time.Now(),
    }
}

func (e PointsDeducted) EventID() string       { return e.eventID }
func (e PointsDeducted) EventType() string     { return "points.deducted" }
func (e PointsDeducted) OccurredAt() time.Time { return e.occurredAt }
func (e PointsDeducted) AggregateID() string   { return e.accountID.String() }

// Getters
func (e PointsDeducted) AccountID() AccountID { return e.accountID }
func (e PointsDeducted) Amount() PointsAmount { return e.amount }
func (e PointsDeducted) Reason() string       { return e.reason }

// --- PointsRecalculated 事件 ---

// PointsRecalculated 積分已重算事件
type PointsRecalculated struct {
    eventID    string
    accountID  AccountID
    oldPoints  int
    newPoints  int
    occurredAt time.Time
}

// NewPointsRecalculatedEvent 創建積分已重算事件
func NewPointsRecalculatedEvent(
    accountID AccountID,
    oldPoints int,
    newPoints int,
) PointsRecalculated {
    return PointsRecalculated{
        eventID:    uuid.New().String(),
        accountID:  accountID,
        oldPoints:  oldPoints,
        newPoints:  newPoints,
        occurredAt: time.Now(),
    }
}

func (e PointsRecalculated) EventID() string       { return e.eventID }
func (e PointsRecalculated) EventType() string     { return "points.recalculated" }
func (e PointsRecalculated) OccurredAt() time.Time { return e.occurredAt }
func (e PointsRecalculated) AggregateID() string   { return e.accountID.String() }

// Getters
func (e PointsRecalculated) AccountID() AccountID { return e.accountID }
func (e PointsRecalculated) OldPoints() int       { return e.oldPoints }
func (e PointsRecalculated) NewPoints() int       { return e.newPoints }
```

### 6.2 領域事件設計原則

**領域事件設計原則**:
1. ✅ **不可變性**: 事件創建後不可修改
2. ✅ **自描述**: 事件名稱使用過去式（PointsEarned, TransactionVerified）
3. ✅ **包含完整信息**: 事件應包含足夠的信息供處理器使用
4. ✅ **實現基礎接口**: 實現 `shared.DomainEvent` 接口
5. ✅ **命名規範**: `{上下文名}.{動作過去式}` (例如 `points.earned`)

---

## 7. 領域錯誤定義

**文件**: `internal/domain/points/errors.go`

```go
package points

import "errors"

// 積分帳戶相關錯誤
var (
    ErrAccountNotFound         = errors.New("points account not found")
    ErrAccountAlreadyExists    = errors.New("points account already exists for this member")
    ErrInsufficientPoints      = errors.New("insufficient points for this operation")
    ErrNegativePointsAmount    = errors.New("points amount cannot be negative")
    ErrInvalidPointsSource     = errors.New("invalid points source")
    ErrInsufficientEarnedPoints = errors.New("earned points cannot be less than used points")
)

// 轉換規則相關錯誤
var (
    ErrRuleNotFound             = errors.New("conversion rule not found")
    ErrRuleAlreadyExists        = errors.New("conversion rule already exists")
    ErrNoConversionRuleForDate  = errors.New("no conversion rule found for the specified date")
    ErrDateRangeOverlap         = errors.New("date range overlaps with existing conversion rule")
    ErrInvalidDateRange         = errors.New("invalid date range: start date must be before or equal to end date")
    ErrInvalidConversionRate    = errors.New("conversion rate must be between 1 and 1000")
)

// 積分計算相關錯誤
var (
    ErrInvalidAmount            = errors.New("amount must be greater than 0")
    ErrRecalculationFailed      = errors.New("points recalculation failed")
    ErrRecalculationInProgress  = errors.New("points recalculation is already in progress")
)

// 其他錯誤
var (
    ErrInvalidMemberID = errors.New("invalid member ID")
)
```

---

## 8. 共享領域概念

**文件**: `internal/domain/shared/transaction.go`

```go
package shared

// TransactionContext 事務上下文接口
// 這是一個標記接口，Infrastructure Layer 會實現具體的事務封裝
type TransactionContext interface {
    // 標記接口：僅用於傳遞上下文，不暴露方法
}

// TransactionManager 事務管理器接口
type TransactionManager interface {
    InTransaction(fn func(ctx TransactionContext) error) error
}
```

**文件**: `internal/domain/shared/event.go`

```go
package shared

import "time"

// DomainEvent 領域事件基礎接口
type DomainEvent interface {
    EventID() string        // 事件唯一標識
    EventType() string      // 事件類型
    OccurredAt() time.Time  // 發生時間
    AggregateID() string    // 聚合根 ID
}

// EventPublisher 事件發布器接口
// 設計原則：接口定義在 Domain Layer（使用者），由 Infrastructure 實現
// 這遵循依賴反轉原則（DIP），避免 Application Layer 依賴 Infrastructure Layer
type EventPublisher interface {
    Publish(event DomainEvent) error
    PublishBatch(events []DomainEvent) error
}

// EventSubscriber 事件訂閱器接口
// Application Layer 的 Event Handlers 通過此接口註冊
type EventSubscriber interface {
    Subscribe(eventType string, handler EventHandler) error
}

// EventHandler 事件處理器接口
// Application Layer 的具體 Event Handlers 實現此接口
type EventHandler interface {
    Handle(event DomainEvent) error
    EventType() string
}
```

---

## 9. 總結

### Domain Layer 檢查清單

**聚合根**:
- [ ] 所有字段私有
- [ ] 通過構造函數創建
- [ ] 狀態變更通過方法（Tell, Don't Ask）
- [ ] 不變性保護（業務規則檢查）
- [ ] 發布領域事件
- [ ] 輕量級設計（無無界集合）
- [ ] 版本控制（樂觀鎖）

**值對象**:
- [ ] 不可變性（無 setter）
- [ ] 構造時驗證
- [ ] 值相等性
- [ ] 封裝業務邏輯
- [ ] 自描述的類型名稱

**領域服務**:
- [ ] 無狀態
- [ ] 純函數邏輯
- [ ] 使用策略模式
- [ ] 依賴接口

**Repository 接口**:
- [ ] 定義在 Domain Layer
- [ ] 只依賴 Domain 實體
- [ ] 使用 Transaction Context
- [ ] 返回 Domain 錯誤

**領域事件**:
- [ ] 不可變性
- [ ] 過去式命名
- [ ] 實現基礎接口
- [ ] 包含完整信息

**無外部依賴**:
- [ ] 無 import `gorm`
- [ ] 無 import `gin`
- [ ] 無 import `redis`
- [ ] 無 import `infrastructure`
- [ ] 無 import `application`

---

**下一步**: 閱讀 [03-Application Layer 實現指南](./03-application-layer-implementation.md) 了解如何實現應用層
