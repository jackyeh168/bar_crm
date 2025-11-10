# Application Layer 實現指南

> **版本**: 1.0
> **最後更新**: 2025-01-10

## 1. Use Case 實現

### 1.1 Use Case 結構

**文件**: `internal/application/usecases/points/earn_points.go`

```go
package points

import (
    "github.com/yourorg/bar_crm/internal/domain/points"
    "github.com/yourorg/bar_crm/internal/domain/points/repository"
    "github.com/yourorg/bar_crm/internal/domain/shared"
)

// EarnPointsCommand 輸入命令
type EarnPointsCommand struct {
    MemberID    string
    Amount      decimal.Decimal
    InvoiceDate time.Time
    Source      string
    SourceID    string
}

// EarnPointsResult 輸出結果
type EarnPointsResult struct {
    AccountID       string
    NewBalance      int
    EarnedPoints    int
}

// EarnPointsUseCase 獲得積分用例
type EarnPointsUseCase struct {
    accountRepo  repository.PointsAccountRepository
    calculator   points.PointsCalculationService
    txManager    shared.TransactionManager
}

func NewEarnPointsUseCase(
    accountRepo  repository.PointsAccountRepository,
    calculator   points.PointsCalculationService,
    txManager    shared.TransactionManager,
) *EarnPointsUseCase {
    return &EarnPointsUseCase{
        accountRepo:  accountRepo,
        calculator:   calculator,
        txManager:    txManager,
    }
}

// Execute 執行用例（純協調邏輯）
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) (*EarnPointsResult, error) {
    var result *EarnPointsResult

    // 使用事務管理器（Application Layer 職責）
    err := uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        // 1. 查詢聚合
        memberID, err := points.NewMemberID(cmd.MemberID)
        if err != nil {
            return err
        }

        account, err := uc.accountRepo.FindByMemberID(ctx, memberID)
        if err != nil {
            return err
        }

        // 2. 委託給 Domain Layer 計算積分
        // 使用集中定義的 DTO（實現 Domain 接口）
        transactionDTO := dto.InvoiceTransactionDTO{
            Amount:          cmd.Amount,
            InvoiceDate:     cmd.InvoiceDate,
            SurveySubmitted: false,
        }
        pointsAmount := uc.calculator.CalculateForTransaction(transactionDTO)

        // 3. 調用聚合方法（業務邏輯在 Domain）
        source := parsePointsSource(cmd.Source)
        err = account.EarnPoints(pointsAmount, source, cmd.SourceID, "Invoice verified")
        if err != nil {
            return err
        }

        // 4. 保存聚合
        err = uc.accountRepo.Update(ctx, account)
        if err != nil {
            return err
        }

        // 5. 構造返回結果
        result = &EarnPointsResult{
            AccountID:    account.GetAccountID().String(),
            NewBalance:   account.GetAvailablePoints().Value(),
            EarnedPoints: pointsAmount.Value(),
        }

        return nil
    })

    return result, err
}
```

### 1.2 Use Case 設計原則

- ✅ **薄薄一層**: 只做協調，不包含業務邏輯
- ✅ **事務管理**: Application Layer 管理事務邊界
- ✅ **依賴接口**: 依賴 Repository 接口，非實現
- ✅ **DTO 轉換**: Application Layer 負責 Entity ↔ DTO

## 2. DTO 設計

**文件**: `internal/application/dto/points_dto.go`

```go
package dto

import (
    "time"
    "github.com/shopspring/decimal"
)

// PointsAccountDTO 積分帳戶 DTO（對外輸出）
type PointsAccountDTO struct {
    AccountID       string
    MemberID        string
    EarnedPoints    int
    UsedPoints      int
    AvailablePoints int
    LastUpdatedAt   time.Time
}
```

**文件**: `internal/application/dto/invoice_dto.go`

```go
package dto

import (
    "time"
    "github.com/shopspring/decimal"
)

// InvoiceTransactionDTO 發票交易 DTO
// 實現 Domain Layer 的 PointsCalculableTransaction 接口
type InvoiceTransactionDTO struct {
    TransactionID   string
    Amount          decimal.Decimal
    InvoiceDate     time.Time
    SurveySubmitted bool
}

// 實現 Domain 接口方法
func (d InvoiceTransactionDTO) GetTransactionAmount() decimal.Decimal {
    return d.Amount
}

func (d InvoiceTransactionDTO) GetTransactionDate() time.Time {
    return d.InvoiceDate
}

func (d InvoiceTransactionDTO) HasCompletedSurvey() bool {
    return d.SurveySubmitted
}

// VerifiedTransactionDTO 已驗證交易 DTO（跨上下文數據傳輸）
type VerifiedTransactionDTO struct {
    TransactionID   string
    Amount          decimal.Decimal
    InvoiceDate     time.Time
    SurveySubmitted bool
}
```

## 3. 事件處理器

**文件**: `internal/application/events/points/transaction_verified_handler.go`

```go
package points

import (
    "github.com/yourorg/bar_crm/internal/application/usecases/points"
    "github.com/yourorg/bar_crm/internal/domain/invoice"
)

// TransactionVerifiedHandler 處理交易驗證事件
type TransactionVerifiedHandler struct {
    earnPointsUseCase *points.EarnPointsUseCase
}

func NewTransactionVerifiedHandler(
    earnPointsUseCase *points.EarnPointsUseCase,
) *TransactionVerifiedHandler {
    return &TransactionVerifiedHandler{
        earnPointsUseCase: earnPointsUseCase,
    }
}

// Handle 處理事件
func (h *TransactionVerifiedHandler) Handle(event invoice.TransactionVerified) error {
    cmd := points.EarnPointsCommand{
        MemberID:    event.MemberID().String(),
        Amount:      event.Amount(),
        InvoiceDate: event.InvoiceDate(),
        Source:      "invoice",
        SourceID:    event.TransactionID().String(),
    }

    _, err := h.earnPointsUseCase.Execute(cmd)
    return err
}

// EventType 返回處理的事件類型
func (h *TransactionVerifiedHandler) EventType() string {
    return "invoice.transaction_verified"
}
```

## 4. Application Service（統一事件發布）

### 4.1 設計目標

**問題**：
- ❌ 每個 Use Case 都需要手動發布事件（容易遺漏）
- ❌ Use Case 需要依賴 EventPublisher（增加耦合）
- ❌ 事件發布邏輯重複出現在每個 Use Case

**解決方案**：
- ✅ 創建 Application Service 統一處理事務和事件發布
- ✅ Use Case 只關注業務邏輯，無需處理事件
- ✅ 事務成功後自動發布所有聚合的事件

### 4.2 Application Service 實現

**文件**: `internal/application/service/application_service.go`

```go
package service

import (
    "github.com/yourorg/bar_crm/internal/domain/shared"
)

// ApplicationService 應用服務
// 職責：統一處理事務管理和事件發布
type ApplicationService struct {
    txManager    shared.TransactionManager
    eventBus     shared.EventPublisher
}

func NewApplicationService(
    txManager shared.TransactionManager,
    eventBus shared.EventPublisher,
) *ApplicationService {
    return &ApplicationService{
        txManager: txManager,
        eventBus:  eventBus,
    }
}

// ExecuteInTransaction 在事務中執行業務邏輯，成功後自動發布事件
// fn 是業務邏輯函數，返回修改過的聚合根列表
func (s *ApplicationService) ExecuteInTransaction(
    fn func(ctx shared.TransactionContext) ([]AggregateRoot, error),
) error {
    var aggregates []AggregateRoot

    // 1. 在事務中執行業務邏輯
    err := s.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        aggs, err := fn(ctx)
        if err != nil {
            return err
        }
        aggregates = aggs
        return nil
    })

    if err != nil {
        return err  // 事務失敗，不發布事件
    }

    // 2. 事務成功後，收集並發布所有事件
    events := s.collectEvents(aggregates)
    if len(events) > 0 {
        if err := s.eventBus.PublishBatch(events); err != nil {
            // 事件發布失敗記錄日誌，但不影響事務結果
            // 實際應用中應該有重試機制或補償邏輯
            // TODO: 記錄事件發布失敗到 outbox 表
            return err
        }
    }

    // 3. 清空聚合事件
    s.clearEvents(aggregates)

    return nil
}

// collectEvents 收集所有聚合的事件
func (s *ApplicationService) collectEvents(aggregates []AggregateRoot) []shared.DomainEvent {
    var events []shared.DomainEvent
    for _, agg := range aggregates {
        events = append(events, agg.GetEvents()...)
    }
    return events
}

// clearEvents 清空所有聚合的事件
func (s *ApplicationService) clearEvents(aggregates []AggregateRoot) {
    for _, agg := range aggregates {
        agg.ClearEvents()
    }
}

// AggregateRoot 聚合根介面（所有聚合根實現此介面）
type AggregateRoot interface {
    GetEvents() []shared.DomainEvent
    ClearEvents()
}
```

### 4.3 Use Case 重構

**原始 Use Case（手動發布事件）**：

```go
// ❌ 舊方式：Use Case 需要處理事件
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) (*EarnPointsResult, error) {
    var result *EarnPointsResult
    var account *points.PointsAccount

    err := uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        // ... 業務邏輯 ...
        account, _ = uc.accountRepo.FindByMemberID(ctx, memberID)
        account.EarnPoints(...)
        uc.accountRepo.Update(ctx, account)
        return nil
    })

    // 手動發布事件（容易遺漏）
    if err == nil {
        for _, event := range account.GetEvents() {
            uc.eventBus.Publish(event)
        }
        account.ClearEvents()
    }

    return result, err
}
```

**重構後（使用 Application Service）**：

```go
// ✅ 新方式：Use Case 只關注業務邏輯
type EarnPointsUseCase struct {
    accountRepo repository.PointsAccountRepository
    calculator  points.PointsCalculationService
    appService  *service.ApplicationService  // 使用 Application Service
}

func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) (*EarnPointsResult, error) {
    var result *EarnPointsResult

    // 使用 Application Service 執行業務邏輯
    err := uc.appService.ExecuteInTransaction(func(ctx shared.TransactionContext) ([]service.AggregateRoot, error) {
        // 1. 查詢聚合
        memberID, err := points.NewMemberID(cmd.MemberID)
        if err != nil {
            return nil, err
        }

        account, err := uc.accountRepo.FindByMemberID(ctx, memberID)
        if err != nil {
            return nil, err
        }

        // 2. 計算積分
        transactionDTO := dto.InvoiceTransactionDTO{
            Amount:          cmd.Amount,
            InvoiceDate:     cmd.InvoiceDate,
            SurveySubmitted: false,
        }
        pointsAmount := uc.calculator.CalculateForTransaction(transactionDTO)

        // 3. 調用聚合方法
        source := parsePointsSource(cmd.Source)
        err = account.EarnPoints(pointsAmount, source, cmd.SourceID, "Invoice verified")
        if err != nil {
            return nil, err
        }

        // 4. 保存聚合
        err = uc.accountRepo.Update(ctx, account)
        if err != nil {
            return nil, err
        }

        // 5. 構造返回結果
        result = &EarnPointsResult{
            AccountID:    account.GetAccountID().String(),
            NewBalance:   account.GetAvailablePoints().Value(),
            EarnedPoints: pointsAmount.Value(),
        }

        // 6. 返回修改過的聚合（Application Service 會自動發布事件）
        return []service.AggregateRoot{account}, nil
    })

    return result, err
}
```

### 4.4 架構優勢

**職責分離**:
- ✅ Use Case：純業務邏輯協調
- ✅ Application Service：事務管理 + 事件發布
- ✅ Event Bus：事件分發（Infrastructure）

**關鍵改進**:
1. ✅ Use Case 無需知道 EventPublisher 的存在
2. ✅ 事件發布自動化，不會遺漏
3. ✅ 統一的事件處理邏輯（未來可添加 Outbox Pattern）
4. ✅ 更容易測試（Use Case 只需 Mock Repository 和 Calculator）

### 4.5 進階：Outbox Pattern

**未來擴展** - 確保事件最終一致性：

```go
// 事務表（Outbox Table）
type EventOutbox struct {
    ID        string
    EventType string
    EventData []byte
    Published bool
    CreatedAt time.Time
}

// 在事務中保存事件到 Outbox 表
func (s *ApplicationService) ExecuteInTransactionWithOutbox(
    fn func(ctx shared.TransactionContext) ([]AggregateRoot, error),
) error {
    var aggregates []AggregateRoot

    err := s.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        aggs, err := fn(ctx)
        if err != nil {
            return err
        }
        aggregates = aggs

        // 將事件保存到 Outbox 表（與業務在同一事務）
        events := s.collectEvents(aggregates)
        for _, event := range events {
            if err := s.saveToOutbox(ctx, event); err != nil {
                return err
            }
        }
        return nil
    })

    if err != nil {
        return err
    }

    // 後台異步任務從 Outbox 發布事件
    // 確保事件最終會被發布（即使當前發布失敗）

    return nil
}
```

## 5. 跨聚合事務與最終一致性

### 5.1 核心原則

在 DDD 和 Clean Architecture 中，有一個重要的設計原則：

> **一個事務只能修改一個聚合根**

這個原則的目的是：
1. 保持聚合邊界清晰
2. 避免分佈式事務的複雜性
3. 提高系統可擴展性
4. 減少鎖競爭

**當需要同時修改多個聚合時，應該使用領域事件實現最終一致性**。

### 5.2 反模式：在一個事務中修改多個聚合

#### 錯誤示範

```go
// ❌ 錯誤：在一個事務中修改多個聚合
func (uc *RedeemPointsUseCase) Execute(cmd RedeemPointsCommand) error {
    return uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        // 1. 修改第一個聚合（PointsAccount）
        account, err := uc.accountRepo.FindByID(ctx, cmd.AccountID)
        if err != nil {
            return err
        }

        err = account.DeductPoints(cmd.Points, cmd.RewardID)
        if err != nil {
            return err
        }

        uc.accountRepo.Update(ctx, account)

        // 2. 修改第二個聚合（Redemption）
        redemption, err := uc.redemptionRepo.FindByID(ctx, cmd.RedemptionID)
        if err != nil {
            return err
        }

        redemption.MarkAsCompleted()
        uc.redemptionRepo.Update(ctx, redemption)

        // ❌ 問題：
        // - 違反聚合邊界原則
        // - 增加事務持有時間（鎖競爭）
        // - 兩個聚合緊耦合
        // - 如果 redemption 更新失敗，account 已經扣減積分（數據不一致）

        return nil
    })
}
```

**問題分析**：

1. **違反聚合邊界**：`PointsAccount` 和 `Redemption` 是兩個獨立的聚合，不應在同一事務中修改
2. **事務時間過長**：持有兩個聚合的鎖，增加並發衝突
3. **緊耦合**：`PointsAccount` 的修改依賴於 `Redemption` 的存在
4. **錯誤處理複雜**：如果第二個聚合更新失敗，第一個聚合的修改需要回滾

### 5.3 正確模式：使用領域事件實現最終一致性

#### 正確示範

**Step 1: 扣除積分（第一個事務）**

```go
// ✅ 正確：只修改一個聚合
func (uc *DeductPointsUseCase) Execute(cmd DeductPointsCommand) error {
    return uc.appService.ExecuteInTransaction(func(ctx shared.TransactionContext) ([]shared.AggregateRoot, error) {
        // 1. 查找聚合
        account, err := uc.accountRepo.FindByID(ctx, cmd.AccountID)
        if err != nil {
            return nil, err
        }

        // 2. 執行業務邏輯
        err = account.DeductPoints(cmd.Points, cmd.Reason)
        if err != nil {
            return nil, err  // 業務規則檢查失敗（如積分不足）
        }

        // 3. 持久化聚合
        err = uc.accountRepo.Update(ctx, account)
        if err != nil {
            return nil, err
        }

        // 4. 返回聚合（ApplicationService 會自動收集並發布事件）
        return []shared.AggregateRoot{account}, nil

        // ✅ 事件 PointsDeducted 會在事務成功後自動發布
    })
}
```

**Step 2: 事件處理器創建兌換記錄（第二個事務）**

```go
// internal/application/events/points/points_deducted_handler.go
package points

import (
    "github.com/yourorg/bar_crm/internal/application/usecases/redemption"
    "github.com/yourorg/bar_crm/internal/domain/shared"
)

// PointsDeductedHandler 處理積分扣除事件
type PointsDeductedHandler struct {
    createRedemptionUC *redemption.CreateRedemptionUseCase
}

func NewPointsDeductedHandler(
    createRedemptionUC *redemption.CreateRedemptionUseCase,
) *PointsDeductedHandler {
    return &PointsDeductedHandler{
        createRedemptionUC: createRedemptionUC,
    }
}

func (h *PointsDeductedHandler) EventType() string {
    return "PointsDeducted"
}

// Handle 處理事件（在獨立的事務中執行）
func (h *PointsDeductedHandler) Handle(event shared.DomainEvent) error {
    pointsDeducted := event.(PointsDeductedEvent)

    // ✅ 在新的事務中創建兌換記錄
    cmd := redemption.CreateRedemptionCommand{
        AccountID: pointsDeducted.AccountID(),
        Points:    pointsDeducted.Amount(),
        Reason:    pointsDeducted.Reason(),
    }

    _, err := h.createRedemptionUC.Execute(cmd)
    if err != nil {
        // ⚠️ 如果創建失敗，應該記錄錯誤並重試
        // 可以使用 Outbox Pattern 確保事件最終被處理
        return err
    }

    return nil
}
```

**Step 3: 創建兌換記錄 Use Case**

```go
// internal/application/usecases/redemption/create_redemption.go
package redemption

func (uc *CreateRedemptionUseCase) Execute(cmd CreateRedemptionCommand) (*Redemption, error) {
    return uc.appService.ExecuteInTransaction(func(ctx shared.TransactionContext) ([]shared.AggregateRoot, error) {
        // 1. 創建兌換記錄聚合
        redemption := redemption.NewRedemption(
            cmd.AccountID,
            cmd.Points,
            cmd.Reason,
        )

        // 2. 持久化
        err := uc.redemptionRepo.Create(ctx, redemption)
        if err != nil {
            return nil, err
        }

        // 3. 返回聚合
        return []shared.AggregateRoot{redemption}, nil

        // ✅ 事件 RedemptionCreated 會在事務成功後發布
    })
}
```

### 5.4 設計優勢對比

| 方面 | 一個事務修改多個聚合 ❌ | 領域事件 + 最終一致性 ✅ |
|------|------------------------|------------------------|
| **聚合邊界** | 違反邊界原則 | 保持邊界清晰 |
| **事務持有時間** | 長（多個聚合） | 短（單個聚合） |
| **鎖競爭** | 高 | 低 |
| **可擴展性** | 差（緊耦合） | 好（鬆耦合） |
| **錯誤處理** | 複雜（需回滾） | 簡單（獨立處理） |
| **測試性** | 難（多個聚合依賴） | 易（單個聚合測試） |

### 5.5 最終一致性的保證

#### 問題：如果事件處理失敗怎麼辦？

**場景**：
1. ✅ 事務 1 成功：積分已扣除，`PointsDeducted` 事件發布
2. ❌ 事件處理失敗：創建兌換記錄失敗

**結果**：積分已扣除，但兌換記錄未創建 → 數據不一致

#### 解決方案 1：事件重試機制

```go
// internal/application/events/points/points_deducted_handler.go
func (h *PointsDeductedHandler) Handle(event shared.DomainEvent) error {
    maxRetries := 3
    var err error

    for attempt := 1; attempt <= maxRetries; attempt++ {
        err = h.createRedemptionUC.Execute(cmd)
        if err == nil {
            return nil  // 成功
        }

        // 如果是業務錯誤（如重複創建），不重試
        if errors.Is(err, redemption.ErrDuplicateRedemption) {
            return nil  // 冪等性：已經創建過了
        }

        // 技術錯誤（數據庫連接失敗等），重試
        time.Sleep(time.Duration(attempt) * time.Second)
    }

    // 所有重試失敗，記錄到死信隊列
    return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}
```

#### 解決方案 2：Outbox Pattern（生產推薦）

參考 [4.3 Outbox Pattern（未來擴展）](04-infrastructure-layer-implementation.md#43-outbox-pattern)，確保事件最終會被處理。

**流程**：
1. 在同一事務中保存事件到 `outbox` 表
2. 後台任務從 `outbox` 表讀取事件並發布
3. 發布成功後標記事件為已處理
4. 保證事件不會丟失

### 5.6 冪等性設計

由於事件可能被重複處理（重試、網絡問題），Event Handler 必須設計為冪等的。

#### 冪等性實現方式

**方式 1：業務唯一鍵檢查**

```go
func (uc *CreateRedemptionUseCase) Execute(cmd CreateRedemptionCommand) (*Redemption, error) {
    return uc.appService.ExecuteInTransaction(func(ctx shared.TransactionContext) ([]shared.AggregateRoot, error) {
        // 1. 檢查是否已存在（冪等性保證）
        existing, err := uc.redemptionRepo.FindByAccountIDAndReason(ctx, cmd.AccountID, cmd.Reason)
        if err == nil && existing != nil {
            return []shared.AggregateRoot{existing}, nil  // 已存在，直接返回
        }

        // 2. 創建新記錄
        redemption := redemption.NewRedemption(cmd.AccountID, cmd.Points, cmd.Reason)
        // ...
    })
}
```

**方式 2：事件 ID 追蹤**

```go
// 在數據庫中記錄已處理的事件 ID
type ProcessedEvent struct {
    EventID   string
    EventType string
    ProcessedAt time.Time
}

func (h *PointsDeductedHandler) Handle(event shared.DomainEvent) error {
    // 1. 檢查事件是否已處理
    processed, _ := h.eventRepo.IsProcessed(event.EventID())
    if processed {
        return nil  // 已處理，跳過
    }

    // 2. 處理事件
    err := h.createRedemptionUC.Execute(cmd)
    if err != nil {
        return err
    }

    // 3. 標記事件為已處理
    h.eventRepo.MarkAsProcessed(event.EventID())

    return nil
}
```

### 5.7 實際案例：積分兌換流程

#### 完整流程圖

```
┌─────────────────────────────────────────────────────────────────┐
│                       積分兌換完整流程                            │
└─────────────────────────────────────────────────────────────────┘

Step 1: 用戶請求兌換
  ↓
┌─────────────────────────────────────┐
│ Presentation Layer                  │
│ POST /api/points/redeem             │
└─────────────────────────────────────┘
  ↓
Step 2: 扣除積分（事務 1）
┌─────────────────────────────────────┐
│ DeductPointsUseCase                 │
│                                     │
│ 1. account.DeductPoints()           │
│ 2. accountRepo.Update()             │
│ 3. 發布 PointsDeducted 事件         │
└─────────────────────────────────────┘
  ↓
Step 3: 事件處理（事務 2）
┌─────────────────────────────────────┐
│ PointsDeductedHandler               │
│                                     │
│ Handle(PointsDeducted) {            │
│   CreateRedemptionUseCase.Execute() │
│ }                                   │
└─────────────────────────────────────┘
  ↓
Step 4: 創建兌換記錄（事務 2）
┌─────────────────────────────────────┐
│ CreateRedemptionUseCase             │
│                                     │
│ 1. redemption = NewRedemption()     │
│ 2. redemptionRepo.Create()          │
│ 3. 發布 RedemptionCreated 事件      │
└─────────────────────────────────────┘
  ↓
Step 5: 通知用戶（事務 3，可選）
┌─────────────────────────────────────┐
│ RedemptionCreatedHandler            │
│                                     │
│ Handle(RedemptionCreated) {         │
│   notificationService.Send(...)     │
│ }                                   │
└─────────────────────────────────────┘
```

#### 時序圖

```
用戶         API          DeductPointsUC    EventBus    Handler    CreateRedemptionUC
 │            │                 │              │           │              │
 │── POST ───>│                 │              │           │              │
 │            │── Execute ─────>│              │           │              │
 │            │                 │              │           │              │
 │            │                 │── TX BEGIN ──┤           │              │
 │            │                 │              │           │              │
 │            │                 │ DeductPoints │           │              │
 │            │                 │ Update(DB)   │           │              │
 │            │                 │              │           │              │
 │            │                 │── TX COMMIT ─┤           │              │
 │            │                 │              │           │              │
 │            │                 │─ Publish Event ────────>│              │
 │            │                 │              │           │              │
 │            │<── Success ─────│              │           │              │
 │<── 200 ────│                 │              │           │              │
 │            │                 │              │           │              │
 │            │                 │              │           │─ Handle ────>│
 │            │                 │              │           │              │
 │            │                 │              │           │── TX BEGIN ──┤
 │            │                 │              │           │              │
 │            │                 │              │           │ Create       │
 │            │                 │              │           │ Redemption   │
 │            │                 │              │           │              │
 │            │                 │              │           │── TX COMMIT ─┤
 │            │                 │              │           │              │
 │            │                 │              │           │<─ Success ───│
```

### 5.8 何時可以在一個事務中修改多個聚合？

**例外情況**（需謹慎使用）：

1. **兩個聚合屬於同一個 Bounded Context，且業務上必須原子性操作**
   - 例如：轉帳（從帳戶 A 扣款，向帳戶 B 入款）
   - 但即使這種情況，也建議拆分為兩個事務 + Saga 模式

2. **性能關鍵路徑，且聚合非常簡單**
   - 例如：計數器 + 日誌記錄
   - 但這通常意味著聚合邊界劃分有問題

**一般建議**：
- ✅ 優先使用領域事件 + 最終一致性
- ⚠️ 只在極特殊情況下才考慮多聚合事務
- ❌ 避免跨 Bounded Context 的事務

### 5.9 總結

**核心原則**：
1. 一個事務只修改一個聚合根
2. 跨聚合操作使用領域事件
3. 接受最終一致性（Eventual Consistency）
4. 確保事件處理的冪等性
5. 使用 Outbox Pattern 保證事件不丟失

**設計優勢**：
- ✅ 保持聚合邊界清晰
- ✅ 降低鎖競爭和事務持有時間
- ✅ 提高系統可擴展性
- ✅ 簡化錯誤處理
- ✅ 更好的測試性

**記住**：最終一致性不是缺陷，而是分佈式系統的必然選擇。關鍵是要有完善的事件處理機制（重試、Outbox、監控）來保證數據最終一致。

---

**下一步**: 閱讀 [04-Infrastructure Layer 實現指南](./04-infrastructure-layer-implementation.md)
