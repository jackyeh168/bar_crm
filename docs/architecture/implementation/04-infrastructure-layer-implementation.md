# Infrastructure Layer 實現指南

> **版本**: 1.0
> **最後更新**: 2025-01-10

## 1. GORM Repository 實現

### 1.1 GORM 模型定義

**文件**: `internal/infrastructure/persistence/gorm/models.go`

```go
package gorm

import "gorm.io/gorm"

// PointsAccountModel GORM 模型（Infrastructure 層）
// 命名規範：加 "Model" 後綴，避免與 Domain 實體衝突
type PointsAccountModel struct {
    gorm.Model
    AccountID     string `gorm:"uniqueIndex;not null"`
    MemberID      string `gorm:"index;not null"`
    EarnedPoints  int    `gorm:"not null;default:0"`
    UsedPoints    int    `gorm:"not null;default:0"`
    Version       int    `gorm:"not null;default:1"`  // 樂觀鎖
}

func (PointsAccountModel) TableName() string {
    return "points_accounts"
}
```

### 1.2 Repository 實現

**文件**: `internal/infrastructure/persistence/points/account_repository.go`

```go
package points

import (
    "errors"
    "gorm.io/gorm"
    "github.com/jackyeh168/bar_crm/internal/domain/points"
    "github.com/jackyeh168/bar_crm/internal/domain/points/repository"
    "github.com/jackyeh168/bar_crm/internal/domain/shared"
    gormModels "github.com/jackyeh168/bar_crm/internal/infrastructure/persistence/gorm"
)

// GormPointsAccountRepository GORM 實現
type GormPointsAccountRepository struct {
    db *gorm.DB  // 預設連接
}

func NewGormPointsAccountRepository(db *gorm.DB) repository.PointsAccountRepository {
    return &GormPointsAccountRepository{db: db}
}

// Create 創建帳戶
func (r *GormPointsAccountRepository) Create(
    ctx shared.TransactionContext,
    account *points.PointsAccount,
) error {
    db := r.extractDB(ctx)
    model := r.toModel(account)
    return db.Create(model).Error
}

// Update 更新帳戶（樂觀鎖）
func (r *GormPointsAccountRepository) Update(
    ctx shared.TransactionContext,
    account *points.PointsAccount,
) error {
    db := r.extractDB(ctx)
    model := r.toModel(account)

    // 樂觀鎖：使用聚合提供的上一個版本號
    // 聚合已經遞增版本號，GetPreviousVersion() 返回遞增前的版本號
    // 這避免了 Repository 需要知道 "version - 1" 的邏輯（職責封裝）
    previousVersion := account.GetPreviousVersion()

    // 樂觀鎖更新（WHERE version = previousVersion 確保並發控制）
    result := db.Model(&gormModels.PointsAccountModel{}).
        Where("account_id = ? AND version = ?", model.AccountID, previousVersion).
        Updates(map[string]interface{}{
            "earned_points": model.EarnedPoints,
            "used_points":   model.UsedPoints,
            "version":       model.Version,  // 使用聚合已遞增的版本號
            "updated_at":    time.Now(),
        })

    if result.Error != nil {
        return result.Error
    }

    if result.RowsAffected == 0 {
        return repository.ErrConcurrentModification
    }

    return nil
}

// FindByMemberID 根據會員 ID 查詢
func (r *GormPointsAccountRepository) FindByMemberID(
    ctx shared.TransactionContext,
    memberID points.MemberID,
) (*points.PointsAccount, error) {
    db := r.extractDB(ctx)
    var model gormModels.PointsAccountModel

    err := db.Where("member_id = ?", memberID.String()).First(&model).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, repository.ErrAccountNotFound
        }
        return nil, err
    }

    return r.toDomain(&model)
}

// --- 轉換方法 ---

// toModel Domain → GORM Model
func (r *GormPointsAccountRepository) toModel(
    account *points.PointsAccount,
) *gormModels.PointsAccountModel {
    return &gormModels.PointsAccountModel{
        AccountID:    account.GetAccountID().String(),
        MemberID:     account.GetMemberID().String(),
        EarnedPoints: account.GetEarnedPoints().Value(),
        UsedPoints:   account.GetUsedPoints().Value(),
        Version:      account.GetVersion(),
    }
}

// toDomain GORM Model → Domain
func (r *GormPointsAccountRepository) toDomain(
    model *gormModels.PointsAccountModel,
) (*points.PointsAccount, error) {
    // 使用 Domain Layer 提供的專用重建方法
    accountID, err := points.AccountIDFromString(model.AccountID)
    if err != nil {
        return nil, err
    }

    memberID, err := points.NewMemberID(model.MemberID)
    if err != nil {
        return nil, err
    }

    // 重建聚合（包含不變條件驗證）
    // 如果資料庫有損壞資料，會返回錯誤而非靜默接受
    account, err := points.ReconstructPointsAccount(
        accountID,
        memberID,
        model.EarnedPoints,
        model.UsedPoints,
        model.Version,
        model.UpdatedAt,
    )
    if err != nil {
        // 資料庫資料損壞，記錄日誌並返回錯誤
        // 實際應用中應該觸發告警
        return nil, fmt.Errorf("failed to reconstruct account from database: %w", err)
    }

    return account, nil
}

// extractDB 從 Transaction Context 提取 DB 連接
func (r *GormPointsAccountRepository) extractDB(
    ctx shared.TransactionContext,
) *gorm.DB {
    if txCtx, ok := ctx.(*gormTransactionContext); ok {
        return txCtx.tx  // 返回事務連接
    }
    return r.db  // 返回普通連接
}
```

## 2. Transaction Context 實現

**文件**: `internal/infrastructure/persistence/gorm/transaction.go`

```go
package gorm

import (
    "gorm.io/gorm"
    "github.com/jackyeh168/bar_crm/internal/domain/shared"
)

// gormTransactionContext 實現 TransactionContext 接口
type gormTransactionContext struct {
    tx *gorm.DB  // 資料庫事務
}

// GormTransactionManager 實現 TransactionManager
type GormTransactionManager struct {
    db *gorm.DB
}

func NewGormTransactionManager(db *gorm.DB) shared.TransactionManager {
    return &GormTransactionManager{db: db}
}

// InTransaction 在事務中執行業務邏輯
func (tm *GormTransactionManager) InTransaction(
    fn func(ctx shared.TransactionContext) error,
) error {
    // 開啟事務
    tx := tm.db.Begin()
    if tx.Error != nil {
        return tx.Error
    }

    // 創建 Context
    ctx := &gormTransactionContext{tx: tx}

    // 執行業務邏輯
    if err := fn(ctx); err != nil {
        tx.Rollback()  // 回滾
        return err
    }

    // 提交事務
    return tx.Commit().Error
}
```

## 3. 外部服務適配器

**文件**: `internal/infrastructure/external/linebot/adapter.go`

```go
package linebot

import (
    "github.com/line/line-bot-sdk-go/v8/linebot"
    "github.com/jackyeh168/bar_crm/internal/domain/member"
)

// LineUserAdapter LINE 用戶適配器（Anti-Corruption Layer）
type LineUserAdapter struct {
    client *linebot.Client
}

func NewLineUserAdapter(client *linebot.Client) *LineUserAdapter {
    return &LineUserAdapter{client: client}
}

// GetUserProfile 獲取用戶資料
func (a *LineUserAdapter) GetUserProfile(lineUserID string) (*member.Member, error) {
    // 調用 LINE SDK
    profile, err := a.client.GetProfile(lineUserID).Do()
    if err != nil {
        return nil, err
    }

    // 轉換為 Domain 對象（ACL 職責）
    return member.NewMember(
        member.NewLineUserID(profile.UserID),
        member.NewDisplayName(profile.DisplayName),
    )
}
```

## 4. Event Bus 實現

### 4.1 Event Bus 設計原則

**關鍵原則**:
* ✅ Event Bus 介面定義在 `internal/domain/shared/event.go`
* ✅ Event Bus 實現在 `internal/infrastructure/messaging/event_bus.go`
* ✅ Event Handlers 在 `internal/application/events/`
* ✅ 透過 DI (Dependency Injection) 註冊 handlers 到 Event Bus
* ❌ Infrastructure Layer 絕不直接依賴 Application Layer 的 Event Handlers

### 4.2 Event Bus 實現

**文件**: `internal/infrastructure/messaging/event_bus.go`

```go
package messaging

import (
    "sync"
    "github.com/jackyeh168/bar_crm/internal/domain/shared"
)

// InMemoryEventBus 記憶體事件匯流排實現
// 實現 shared.EventPublisher 和 shared.EventSubscriber 介面
type InMemoryEventBus struct {
    handlers map[string][]shared.EventHandler  // eventType -> handlers
    mu       sync.RWMutex                       // 併發保護
}

// NewInMemoryEventBus 創建事件匯流排
func NewInMemoryEventBus() *InMemoryEventBus {
    return &InMemoryEventBus{
        handlers: make(map[string][]shared.EventHandler),
    }
}

// Publish 發布單個事件（實現 EventPublisher 介面）
func (bus *InMemoryEventBus) Publish(event shared.DomainEvent) error {
    bus.mu.RLock()
    handlers := bus.handlers[event.EventType()]
    bus.mu.RUnlock()

    // 執行所有訂閱該事件的處理器
    for _, handler := range handlers {
        if err := handler.Handle(event); err != nil {
            // 記錄錯誤但繼續處理其他 handlers
            // 實際應用中應使用 logger
            return err
        }
    }

    return nil
}

// PublishBatch 批次發布事件
func (bus *InMemoryEventBus) PublishBatch(events []shared.DomainEvent) error {
    for _, event := range events {
        if err := bus.Publish(event); err != nil {
            return err
        }
    }
    return nil
}

// Subscribe 訂閱事件（實現 EventSubscriber 介面）
// Application Layer 的 Event Handlers 透過此方法註冊
func (bus *InMemoryEventBus) Subscribe(eventType string, handler shared.EventHandler) error {
    bus.mu.Lock()
    defer bus.mu.Unlock()

    bus.handlers[eventType] = append(bus.handlers[eventType], handler)
    return nil
}
```

### 4.3 依賴注入配置

**文件**: `cmd/app/main.go`

```go
package main

import (
    "go.uber.org/fx"
    "github.com/jackyeh168/bar_crm/internal/domain/shared"
    "github.com/jackyeh168/bar_crm/internal/infrastructure/messaging"
    "github.com/jackyeh168/bar_crm/internal/application/events/points"
)

func main() {
    fx.New(
        // 1. 提供 Event Bus 實現（單例）
        fx.Provide(
            fx.Annotate(
                messaging.NewInMemoryEventBus,
                fx.As(new(shared.EventPublisher)),    // 作為 EventPublisher
                fx.As(new(shared.EventSubscriber)),   // 作為 EventSubscriber
            ),
        ),

        // 2. 提供 Event Handlers
        fx.Provide(
            points.NewTransactionVerifiedHandler,
            points.NewSurveyCompletedHandler,
            // ... 其他 handlers
        ),

        // 3. 註冊 Event Handlers 到 Event Bus
        fx.Invoke(registerEventHandlers),
    ).Run()
}

// registerEventHandlers 註冊所有事件處理器
// 這是唯一將 Application Layer handlers 連接到 Infrastructure Event Bus 的地方
func registerEventHandlers(
    subscriber shared.EventSubscriber,
    txVerifiedHandler *points.TransactionVerifiedHandler,
    surveyCompletedHandler *points.SurveyCompletedHandler,
    // ... 其他 handlers
) error {
    // 註冊所有 handlers
    if err := subscriber.Subscribe(txVerifiedHandler.EventType(), txVerifiedHandler); err != nil {
        return err
    }

    if err := subscriber.Subscribe(surveyCompletedHandler.EventType(), surveyCompletedHandler); err != nil {
        return err
    }

    // ... 註冊其他 handlers

    return nil
}
```

### 4.4 依賴方向詳解與架構圖

#### 4.4.1 完整依賴方向圖

```
┌─────────────────────────────────────────────────────────────────┐
│                        main.go (依賴注入層)                        │
│  職責：連接各層，註冊 Event Handlers 到 Event Bus                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ (組裝依賴)
                              ↓
      ┌──────────────────────────────────────────────┐
      │                                              │
      ↓                                              ↓
┌──────────────────┐                    ┌──────────────────────┐
│ Infrastructure   │                    │ Application Layer    │
│ Layer            │                    │                      │
│                  │                    │ - Use Cases          │
│ - EventBus 實現  │                    │ - Event Handlers     │
│ - Repository 實現│                    │   * TxVerifiedHandler│
│ - External APIs  │                    │   * SurveyHandler    │
└──────────────────┘                    └──────────────────────┘
      │                                              │
      │ (實現介面)                                   │ (依賴介面)
      │                                              │
      └──────────────────┬───────────────────────────┘
                         ↓
                  ┌─────────────────┐
                  │  Domain Layer   │
                  │                 │
                  │  - 介面定義：   │
                  │    * EventPublisher  │
                  │    * EventSubscriber │
                  │    * Repository      │
                  └─────────────────┘
```

#### 4.4.2 事件處理流程圖

```
┌──────────────────────────────────────────────────────────────────┐
│                         事件發布流程                               │
└──────────────────────────────────────────────────────────────────┘

1. 聚合根產生事件
   ┌─────────────────────────────────────┐
   │ Domain Layer (PointsAccount)        │
   │                                     │
   │ func (a *PointsAccount)             │
   │   EarnPoints(...) {                 │
   │     // 業務邏輯                     │
   │     a.publishEvent(PointsEarned{}) │  ← 發布事件到聚合內部
   │ }                                   │
   └─────────────────────────────────────┘
                │
                │ (事件暫存在聚合內部)
                ↓
2. Use Case 執行並收集事件
   ┌─────────────────────────────────────┐
   │ Application Layer (Use Case)        │
   │                                     │
   │ func Execute(...) {                 │
   │   tx.InTransaction(func() {        │
   │     account.EarnPoints(...)         │  ← 修改聚合
   │     repo.Update(account)            │  ← 持久化
   │   })                                │
   │                                     │
   │   events := account.GetEvents()    │  ← 收集事件
   │   eventPublisher.Publish(events)   │  ← 發布到 Event Bus
   │ }                                   │
   └─────────────────────────────────────┘
                │
                │ (透過介面發布)
                ↓
3. Event Bus 分發事件
   ┌─────────────────────────────────────┐
   │ Infrastructure Layer (Event Bus)    │
   │                                     │
   │ func (bus *InMemoryEventBus)        │
   │   Publish(event) {                  │
   │     handlers := bus.handlers[type] │  ← 查找訂閱者
   │     for _, h := range handlers {   │
   │       h.Handle(event)               │  ← 執行 handlers
   │     }                               │
   │ }                                   │
   └─────────────────────────────────────┘
                │
                │ (調用註冊的 handlers)
                ↓
4. Event Handlers 處理事件
   ┌─────────────────────────────────────┐
   │ Application Layer (Event Handler)   │
   │                                     │
   │ func (h *TxVerifiedHandler)         │
   │   Handle(event) {                   │
   │     // 處理業務邏輯                 │
   │     uc.RecalculatePoints(...)       │  ← 觸發 Use Case
   │ }                                   │
   └─────────────────────────────────────┘
```

#### 4.4.3 依賴方向規則

**關鍵原則**：

1. **Infrastructure 絕不依賴 Application**
   

```go
   // ❌ 錯誤：Infrastructure 直接 import Application
   package messaging

   import "github.com/jackyeh168/bar_crm/internal/application/events"

   func NewEventBus() *EventBus {
       bus := &EventBus{}
       // ❌ Infrastructure 不應該知道具體的 Handler 實現
       bus.handlers["PointsEarned"] = &events.PointsEarnedHandler{}
       return bus
   }
   ```

   

```go
   // ✅ 正確：Infrastructure 只提供技術機制
   package messaging

   import "github.com/jackyeh168/bar_crm/internal/domain/shared"

   func NewEventBus() *InMemoryEventBus {
       return &InMemoryEventBus{
           handlers: make(map[string][]shared.EventHandler),
       }
   }

   // ✅ Subscribe 接受 shared.EventHandler 介面
   func (bus *InMemoryEventBus) Subscribe(
       eventType string,
       handler shared.EventHandler,  // 介面類型，不知道具體實現
   ) error {
       // ...
   }
   ```

2. **Application 依賴 Domain 介面**
   

```go
   // ✅ 正確：Application 依賴 Domain 定義的介面
   package usecases

   import (
       "github.com/jackyeh168/bar_crm/internal/domain/shared"
       "github.com/jackyeh168/bar_crm/internal/domain/points/repository"
   )

   type EarnPointsUseCase struct {
       accountRepo     repository.PointsAccountRepository  // Domain 介面
       eventPublisher  shared.EventPublisher               // Domain 介面
   }
   ```

3. **依賴注入在 main.go 連接各層**
   

```go
   // ✅ 正確：main.go 是唯一知道所有層的地方
   package main

   import (
       // Domain Layer
       "github.com/jackyeh168/bar_crm/internal/domain/shared"

       // Application Layer
       "github.com/jackyeh168/bar_crm/internal/application/events/points"

       // Infrastructure Layer
       "github.com/jackyeh168/bar_crm/internal/infrastructure/messaging"
   )

   func main() {
       // 1. 創建 Infrastructure 實現
       eventBus := messaging.NewInMemoryEventBus()

       // 2. 創建 Application Handlers
       handler := points.NewTransactionVerifiedHandler(...)

       // 3. 連接兩者（註冊）
       eventBus.Subscribe("TransactionVerified", handler)
   }
   ```

#### 4.4.4 為什麼這樣設計？

**問題**：為什麼不讓 Event Bus 直接知道所有 Event Handlers？

```go
// ❌ 錯誤設計：緊耦合
type EventBus struct {
    txVerifiedHandler *events.TransactionVerifiedHandler
    surveyHandler     *events.SurveyCompletedHandler
}

func (bus *EventBus) Publish(event DomainEvent) {
    switch event.EventType() {
    case "TransactionVerified":
        bus.txVerifiedHandler.Handle(event)  // Infrastructure 依賴 Application
    case "SurveyCompleted":
        bus.surveyHandler.Handle(event)
    }
}
```

**問題**：
1. ❌ Infrastructure 依賴 Application（違反依賴規則）
2. ❌ 新增 Handler 需要修改 Infrastructure 代碼（違反 OCP）
3. ❌ 無法單獨測試 Event Bus（緊耦合）

**正確設計**：

```go
// ✅ 正確：基於介面的鬆耦合
type InMemoryEventBus struct {
    handlers map[string][]shared.EventHandler  // 介面類型
}

func (bus *InMemoryEventBus) Subscribe(
    eventType string,
    handler shared.EventHandler,  // 接受任何實現介面的 handler
) error {
    bus.handlers[eventType] = append(bus.handlers[eventType], handler)
    return nil
}

func (bus *InMemoryEventBus) Publish(event shared.DomainEvent) error {
    handlers := bus.handlers[event.EventType()]
    for _, handler := range handlers {
        handler.Handle(event)  // 多態調用
    }
    return nil
}
```

**優勢**：
1. ✅ Infrastructure 只知道 `shared.EventHandler` 介面（依賴反轉）
2. ✅ 新增 Handler 無需修改 Infrastructure 代碼（符合 OCP）
3. ✅ 可單獨測試 Event Bus（使用 mock handler）
4. ✅ 依賴方向正確（Infrastructure → Domain ← Application）

### 4.5 架構優勢總結

**依賴方向總結**:

```
Presentation Layer (HTTP Handlers)
         ↓
Application Layer (Use Cases, Event Handlers)
         ↓ (依賴介面)
Domain Layer (EventPublisher 介面定義)
         ↑ (實現介面)
Infrastructure Layer (InMemoryEventBus 實現)
```

**關鍵點**:
1. ✅ Infrastructure 實現 Domain 定義的介面（依賴反轉）
2. ✅ Application 透過介面使用 Event Bus（解耦）
3. ✅ 依賴注入在 main.go 統一配置（清晰可見）
4. ✅ 符合 Clean Architecture 依賴規則（內層不依賴外層）
5. ✅ Infrastructure **絕不**直接依賴 Application Layer

---

**下一步**: 閱讀 [05-Presentation Layer 實現指南](./05-presentation-layer-implementation.md)
