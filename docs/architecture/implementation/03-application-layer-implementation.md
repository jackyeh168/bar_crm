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

---

**下一步**: 閱讀 [04-Infrastructure Layer 實現指南](./04-infrastructure-layer-implementation.md)
