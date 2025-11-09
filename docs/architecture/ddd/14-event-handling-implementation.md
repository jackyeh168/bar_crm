# 事件處理實作指南 (Event Handling Implementation Guide)

> **版本**: 1.0
> **最後更新**: 2025-01-09
> **狀態**: Production Ready

---

## **目錄**

1. [事件驅動架構概覽](#1-事件驅動架構概覽)
2. [事件收集（Domain Layer）](#2-事件收集domain-layer)
3. [事件發布（Application Layer）](#3-事件發布application-layer)
4. [事件處理器（Application Layer）](#4-事件處理器application-layer)
5. [FX 模組配置](#5-fx-模組配置)
6. [事件持久化與重試](#6-事件持久化與重試)
7. [監控與告警](#7-監控與告警)
8. [常見問題與最佳實踐](#8-常見問題與最佳實踐)

---

## **1. 事件驅動架構概覽**

### **1.1 為什麼需要 Domain Events?**

**問題**：在 DDD 中，聚合（Aggregate）之間不應該直接引用。但業務流程常常需要跨聚合協調。

**範例業務需求**：

```
當發票驗證通過時：
1. 更新 InvoiceTransaction 狀態為 "verified"
2. 賺取積分到 PointsAccount
3. 發送 LINE 通知給會員
4. 記錄稽核日誌
```

**錯誤做法**：聚合間直接調用

```go
// ❌ InvoiceTransaction 直接調用 PointsAccount（違反聚合邊界）
func (tx *InvoiceTransaction) MarkAsVerified() error {
    tx.status = StatusVerified

    // ❌ 跨聚合直接調用
    pointsAccount := pointsRepo.FindByMemberID(tx.memberID)
    pointsAccount.EarnPoints(tx.CalculatedPoints(), SourceInvoice, tx.InvoiceNumber())

    return nil
}
```

**正確做法**：使用 Domain Events 解耦

```go
// ✅ InvoiceTransaction 發出事件
func (tx *InvoiceTransaction) MarkAsVerified() error {
    tx.status = StatusVerified

    // ✅ 發出 Domain Event（不知道誰會訂閱）
    tx.RecordEvent(InvoiceVerified{
        InvoiceNumber: tx.invoiceNumber,
        MemberID:      tx.memberID,
        Amount:        tx.amount,
        InvoiceDate:   tx.invoiceDate,
    })

    return nil
}

// ✅ Application Layer 訂閱事件並協調積分賺取
type InvoiceVerifiedHandler struct {
    earnPointsUseCase *EarnPointsUseCase
}

func (h *InvoiceVerifiedHandler) Handle(event InvoiceVerified) error {
    return h.earnPointsUseCase.Execute(EarnPointsCommand{
        MemberID:    event.MemberID,
        Source:      PointsSourceInvoice,
        SourceID:    event.InvoiceNumber,
        Amount:      calculatePoints(event.Amount),
    })
}
```

### **1.2 事件生命週期**

```
┌───────────────────────────────────────────────────────────┐
│ 1. Domain Layer: 事件收集                                 │
│    Aggregate.RecordEvent(DomainEvent)                     │
│    → 存入 Aggregate 內部的 events []DomainEvent           │
└───────────────────┬───────────────────────────────────────┘
                    │
                    ▼
┌───────────────────────────────────────────────────────────┐
│ 2. Application Layer: 事務提交後發布                      │
│    txManager.InTransaction(func(ctx) {                    │
│        repo.Update(ctx, aggregate)                        │
│        events := aggregate.GetEvents()                    │
│        ctx.AddEvents(events)  // 註冊事件                 │
│    })                                                     │
│    → 事務成功提交後，發布所有事件                          │
└───────────────────┬───────────────────────────────────────┘
                    │
                    ▼
┌───────────────────────────────────────────────────────────┐
│ 3. Event Bus: 分發事件給所有訂閱者                        │
│    eventBus.Publish(event)                                │
│    → 調用所有註冊的 EventHandler                          │
└───────────────────┬───────────────────────────────────────┘
                    │
                    ▼
┌───────────────────────────────────────────────────────────┐
│ 4. Event Handlers: 執行業務邏輯                           │
│    handler.Handle(event)                                  │
│    → 協調其他 Use Cases、發送通知、更新快取等              │
└───────────────────────────────────────────────────────────┘
```

### **1.3 關鍵設計決策**

| 問題 | 方案 | 原因 |
|------|------|------|
| **何時發布事件？** | 事務提交後 | 保證事務一致性（避免發布後事務回滾） |
| **誰負責清理事件？** | Application Layer | Repository 職責單一，不管理事件 |
| **事件處理失敗怎麼辦？** | 重試 + Dead Letter Queue | 保證 at-least-once delivery |
| **如何避免重複處理？** | 冪等性檢查（Cache Key） | 防止重複執行副作用（如重複發送通知） |

---

## **2. 事件收集（Domain Layer）**

### **2.1 Domain Event 定義**

```go
// Domain Layer - internal/domain/event.go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// DomainEvent 所有領域事件的基礎接口
type DomainEvent interface {
    EventID() string
    EventType() string
    OccurredAt() time.Time
    AggregateID() string
}

// BaseDomainEvent 提供通用實現
type BaseDomainEvent struct {
    eventID     string
    eventType   string
    occurredAt  time.Time
    aggregateID string
}

func NewBaseDomainEvent(eventType string, aggregateID string) BaseDomainEvent {
    return BaseDomainEvent{
        eventID:     uuid.New().String(),
        eventType:   eventType,
        occurredAt:  time.Now(),
        aggregateID: aggregateID,
    }
}

func (e BaseDomainEvent) EventID() string     { return e.eventID }
func (e BaseDomainEvent) EventType() string   { return e.eventType }
func (e BaseDomainEvent) OccurredAt() time.Time { return e.occurredAt }
func (e BaseDomainEvent) AggregateID() string { return e.aggregateID }
```

### **2.2 具體事件定義**

```go
// Domain Layer - internal/domain/points/events.go
package points

import "internal/domain"

const (
    EventTypePointsEarned   = "PointsEarned"
    EventTypePointsDeducted = "PointsDeducted"
)

// PointsEarned 積分賺取事件
type PointsEarned struct {
    domain.BaseDomainEvent

    AccountID   AccountID
    MemberID    MemberID
    Amount      PointsAmount
    Source      PointsSource
    SourceID    string
    Description string
}

func NewPointsEarned(
    accountID AccountID,
    memberID MemberID,
    amount PointsAmount,
    source PointsSource,
    sourceID string,
    description string,
) PointsEarned {
    return PointsEarned{
        BaseDomainEvent: domain.NewBaseDomainEvent(
            EventTypePointsEarned,
            accountID.String(),
        ),
        AccountID:   accountID,
        MemberID:    memberID,
        Amount:      amount,
        Source:      source,
        SourceID:    sourceID,
        Description: description,
    }
}

// PointsDeducted 積分扣除事件
type PointsDeducted struct {
    domain.BaseDomainEvent

    AccountID   AccountID
    MemberID    MemberID
    Amount      PointsAmount
    Reason      string
    ReferenceID string
}

func NewPointsDeducted(
    accountID AccountID,
    memberID MemberID,
    amount PointsAmount,
    reason string,
    referenceID string,
) PointsDeducted {
    return PointsDeducted{
        BaseDomainEvent: domain.NewBaseDomainEvent(
            EventTypePointsDeducted,
            accountID.String(),
        ),
        AccountID:   accountID,
        MemberID:    memberID,
        Amount:      amount,
        Reason:      reason,
        ReferenceID: referenceID,
    }
}
```

### **2.3 Aggregate 收集事件**

```go
// Domain Layer - internal/domain/points/points_account.go
package points

import "internal/domain"

type PointsAccount struct {
    accountID    AccountID
    memberID     MemberID
    earnedPoints PointsAmount
    usedPoints   PointsAmount

    events []domain.DomainEvent // 事件收集
}

// RecordEvent 記錄領域事件（不發布）
func (a *PointsAccount) RecordEvent(event domain.DomainEvent) {
    a.events = append(a.events, event)
}

// GetEvents 獲取所有事件（Application Layer 使用）
func (a *PointsAccount) GetEvents() []domain.DomainEvent {
    return a.events
}

// ClearEvents 清除事件（Application Layer 職責）
func (a *PointsAccount) ClearEvents() {
    a.events = nil
}

// EarnPoints 賺取積分（發出事件）
func (a *PointsAccount) EarnPoints(
    amount PointsAmount,
    source PointsSource,
    sourceID string,
    description string,
) error {
    if amount.Value() <= 0 {
        return ErrInvalidAmount
    }

    a.earnedPoints = a.earnedPoints.Add(amount)

    // 記錄事件（不立即發布）
    a.RecordEvent(NewPointsEarned(
        a.accountID,
        a.memberID,
        amount,
        source,
        sourceID,
        description,
    ))

    return nil
}

// DeductPoints 扣除積分（發出事件）
func (a *PointsAccount) DeductPoints(
    amount PointsAmount,
    reason string,
    referenceID string,
) error {
    if amount.Value() <= 0 {
        return ErrInvalidAmount
    }

    availablePoints := a.GetAvailablePoints()
    if availablePoints.LessThan(amount) {
        return ErrInsufficientPoints.WithContext(
            "required", amount.Value(),
            "available", availablePoints.Value(),
        )
    }

    a.usedPoints = a.usedPoints.Add(amount)

    // 記錄事件
    a.RecordEvent(NewPointsDeducted(
        a.accountID,
        a.memberID,
        amount,
        reason,
        referenceID,
    ))

    return nil
}
```

---

## **3. 事件發布（Application Layer）**

### **3.1 方案選擇：Unit of Work vs Transactional Outbox**

#### **方案 A: Unit of Work Pattern（推薦用於單體應用）**

**優勢**:
- ✅ 實現簡單（無需額外表）
- ✅ 事件即時發布（事務提交後立即發布）
- ✅ 適合單體應用與同步處理

**代價**:
- ❌ 事件發布失敗無法重試（已提交事務，無法回滾）
- ❌ 不適合分散式系統

```go
// Application Layer - internal/application/transaction/unit_of_work.go
package transaction

import (
    "internal/domain"
    "gorm.io/gorm"
)

type EventBus interface {
    Publish(event domain.DomainEvent) error
}

type UnitOfWork struct {
    db       *gorm.DB
    eventBus EventBus
}

func NewUnitOfWork(db *gorm.DB, eventBus EventBus) *UnitOfWork {
    return &UnitOfWork{
        db:       db,
        eventBus: eventBus,
    }
}

func (uow *UnitOfWork) InTransaction(
    fn func(ctx TransactionContext) error,
) error {
    // 1. 開啟資料庫事務
    tx := uow.db.Begin()
    if tx.Error != nil {
        return tx.Error
    }

    // 2. 創建事務上下文（收集事件）
    ctx := &transactionContext{
        tx:     tx,
        events: []domain.DomainEvent{},
    }

    // 3. 執行業務邏輯
    if err := fn(ctx); err != nil {
        tx.Rollback()
        return err // 事務失敗，不發布事件
    }

    // 4. 提交事務
    if err := tx.Commit().Error; err != nil {
        return err // 提交失敗，不發布事件
    }

    // 5. ✅ 事務成功後才發布事件（保證一致性）
    for _, event := range ctx.GetEvents() {
        if err := uow.eventBus.Publish(event); err != nil {
            // 事件發布失敗（但事務已提交，無法回滾）
            // 記錄錯誤，稍後重試（需要 Dead Letter Queue）
            log.Error("Failed to publish event", zap.Error(err), zap.Any("event", event))
        }
    }

    return nil
}

// transactionContext 實現 TransactionContext 接口
type transactionContext struct {
    tx     *gorm.DB
    events []domain.DomainEvent
}

func (ctx *transactionContext) AddEvent(event domain.DomainEvent) {
    ctx.events = append(ctx.events, event)
}

func (ctx *transactionContext) GetEvents() []domain.DomainEvent {
    return ctx.events
}
```

#### **方案 B: Transactional Outbox Pattern（推薦用於分散式系統）**

**優勢**:
- ✅ 100% 可靠（事件與業務資料在同一事務中）
- ✅ 支援重試（背景 Worker 處理）
- ✅ 適合分散式系統與非同步處理

**代價**:
- ❌ 實現複雜（需要 Outbox 表 + Worker）
- ❌ 事件發布延遲（Worker 輪詢間隔）

```go
// Application Layer - internal/application/transaction/transactional_outbox.go
package transaction

import (
    "encoding/json"
    "time"

    "internal/domain"
    "gorm.io/gorm"
)

// OutboxMessage Outbox 表結構
type OutboxMessage struct {
    ID          string    `gorm:"primaryKey"`
    EventType   string    `gorm:"index"`
    Payload     []byte    `gorm:"type:jsonb"`
    Published   bool      `gorm:"index"`
    PublishedAt *time.Time
    CreatedAt   time.Time
}

type TransactionalOutboxUnitOfWork struct {
    db       *gorm.DB
    eventBus EventBus
}

func (uow *TransactionalOutboxUnitOfWork) InTransaction(
    fn func(ctx TransactionContext) error,
) error {
    tx := uow.db.Begin()
    if tx.Error != nil {
        return tx.Error
    }

    ctx := &transactionContext{
        tx:     tx,
        events: []domain.DomainEvent{},
    }

    if err := fn(ctx); err != nil {
        tx.Rollback()
        return err
    }

    // ✅ 將事件寫入 Outbox 表（在同一事務中）
    for _, event := range ctx.GetEvents() {
        payload, err := json.Marshal(event)
        if err != nil {
            tx.Rollback()
            return err
        }

        outboxMsg := OutboxMessage{
            ID:        event.EventID(),
            EventType: event.EventType(),
            Payload:   payload,
            Published: false,
            CreatedAt: time.Now(),
        }

        if err := tx.Create(&outboxMsg).Error; err != nil {
            tx.Rollback()
            return err
        }
    }

    // 提交事務（業務資料 + 事件都寫入資料庫）
    if err := tx.Commit().Error; err != nil {
        return err
    }

    // ✅ 事務提交成功，背景 Worker 會發布事件
    return nil
}

// OutboxPublisherWorker 背景 Worker（定期輪詢並發布事件）
type OutboxPublisherWorker struct {
    db       *gorm.DB
    eventBus EventBus
    interval time.Duration
}

func (w *OutboxPublisherWorker) Start() {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()

    for range ticker.C {
        w.publishPendingEvents()
    }
}

func (w *OutboxPublisherWorker) publishPendingEvents() {
    var messages []OutboxMessage

    // 查詢未發布的事件（限制 100 筆）
    err := w.db.Where("published = ?", false).
        Order("created_at ASC").
        Limit(100).
        Find(&messages).Error

    if err != nil {
        log.Error("Failed to query outbox messages", zap.Error(err))
        return
    }

    for _, msg := range messages {
        // 反序列化事件
        event, err := deserializeEvent(msg.EventType, msg.Payload)
        if err != nil {
            log.Error("Failed to deserialize event", zap.Error(err))
            continue
        }

        // 發布事件
        if err := w.eventBus.Publish(event); err != nil {
            log.Error("Failed to publish event", zap.Error(err))
            continue
        }

        // 標記為已發布
        now := time.Now()
        w.db.Model(&OutboxMessage{}).
            Where("id = ?", msg.ID).
            Updates(map[string]interface{}{
                "published":    true,
                "published_at": &now,
            })

        log.Info("Event published successfully", zap.String("eventID", msg.ID))
    }
}
```

### **3.2 Use Case 使用 Unit of Work**

```go
// Application Layer - internal/application/points/earn_points_usecase.go
package pointsapp

import (
    "internal/application/transaction"
    "internal/domain/points"
)

type EarnPointsUseCase struct {
    accountRepo points.Repository
    txManager   transaction.TransactionManager
    logger      *zap.Logger
}

func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    uc.logger.Info("Executing EarnPoints",
        zap.String("memberID", cmd.MemberID.String()),
        zap.Int("amount", cmd.Amount.Value()),
    )

    return uc.txManager.InTransaction(func(ctx transaction.TransactionContext) error {
        // 1. 查詢聚合
        account, err := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
        if err != nil {
            return err
        }

        // 2. 執行業務邏輯（產生事件）
        err = account.EarnPoints(cmd.Amount, cmd.Source, cmd.SourceID, cmd.Description)
        if err != nil {
            return err
        }

        // 3. 持久化聚合
        err = uc.accountRepo.Update(ctx, account)
        if err != nil {
            return err
        }

        // 4. 收集事件並註冊到 Context
        events := account.GetEvents()
        for _, event := range events {
            ctx.AddEvent(event)
        }

        // 5. 清理聚合中的事件（Application Layer 職責）
        account.ClearEvents()

        return nil // 提交事務 + 發布事件
    })
}
```

---

## **4. 事件處理器（Application Layer）**

### **4.1 Event Handler 接口定義**

```go
// Application Layer - internal/application/event/handler.go
package event

import (
    "context"
    "internal/domain"
)

type EventHandler interface {
    EventType() string
    Handle(ctx context.Context, event domain.DomainEvent) error
}
```

### **4.2 具體 Event Handler 實現**

```go
// Application Layer - internal/application/event/points_earned_handler.go
package event

import (
    "context"
    "fmt"
    "time"

    "internal/domain/points"
    "internal/application/notification"
)

type PointsEarnedHandler struct {
    notificationService notification.Service
    cache               Cache
    logger              *zap.Logger
}

func NewPointsEarnedHandler(
    notificationService notification.Service,
    cache Cache,
    logger *zap.Logger,
) *PointsEarnedHandler {
    return &PointsEarnedHandler{
        notificationService: notificationService,
        cache:               cache,
        logger:              logger,
    }
}

func (h *PointsEarnedHandler) EventType() string {
    return points.EventTypePointsEarned
}

func (h *PointsEarnedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    // 類型斷言
    pointsEarnedEvent, ok := event.(points.PointsEarned)
    if !ok {
        return fmt.Errorf("invalid event type: expected PointsEarned, got %T", event)
    }

    h.logger.Info("Handling PointsEarned event",
        zap.String("eventID", event.EventID()),
        zap.String("memberID", pointsEarnedEvent.MemberID.String()),
        zap.Int("amount", pointsEarnedEvent.Amount.Value()),
    )

    // ✅ 冪等性檢查（防止重複處理）
    cacheKey := fmt.Sprintf("event:processed:%s", event.EventID())
    if h.cache.Exists(cacheKey) {
        h.logger.Warn("Event already processed, skipping",
            zap.String("eventID", event.EventID()),
        )
        return nil // 已處理，跳過
    }

    // 發送 LINE 通知
    message := fmt.Sprintf(
        "🎉 您獲得了 %d 積分！\n來源：%s",
        pointsEarnedEvent.Amount.Value(),
        pointsEarnedEvent.Description,
    )

    err := h.notificationService.SendLineMessage(
        ctx,
        pointsEarnedEvent.MemberID.String(),
        message,
    )
    if err != nil {
        h.logger.Error("Failed to send LINE notification",
            zap.String("eventID", event.EventID()),
            zap.Error(err),
        )
        return err // 返回錯誤，觸發重試
    }

    // ✅ 標記為已處理（24 小時 TTL）
    h.cache.Set(cacheKey, true, 24*time.Hour)

    h.logger.Info("PointsEarned event handled successfully",
        zap.String("eventID", event.EventID()),
    )

    return nil
}
```

### **4.3 Event Bus 實現**

```go
// Application Layer - internal/application/event/event_bus.go
package event

import (
    "context"
    "fmt"
    "sync"

    "internal/domain"
)

type EventBus interface {
    Subscribe(eventType string, handler EventHandler)
    Publish(event domain.DomainEvent) error
}

// InMemoryEventBus 記憶體事件總線（適用於單體應用）
type InMemoryEventBus struct {
    handlers map[string][]EventHandler
    mu       sync.RWMutex
    logger   *zap.Logger
}

func NewInMemoryEventBus(logger *zap.Logger) *InMemoryEventBus {
    return &InMemoryEventBus{
        handlers: make(map[string][]EventHandler),
        logger:   logger,
    }
}

func (bus *InMemoryEventBus) Subscribe(eventType string, handler EventHandler) {
    bus.mu.Lock()
    defer bus.mu.Unlock()

    bus.handlers[eventType] = append(bus.handlers[eventType], handler)
    bus.logger.Info("Event handler subscribed",
        zap.String("eventType", eventType),
        zap.String("handler", fmt.Sprintf("%T", handler)),
    )
}

func (bus *InMemoryEventBus) Publish(event domain.DomainEvent) error {
    bus.mu.RLock()
    handlers := bus.handlers[event.EventType()]
    bus.mu.RUnlock()

    if len(handlers) == 0 {
        bus.logger.Warn("No handlers for event type",
            zap.String("eventType", event.EventType()),
        )
        return nil
    }

    ctx := context.Background()

    // 調用所有訂閱者（同步執行）
    for _, handler := range handlers {
        if err := handler.Handle(ctx, event); err != nil {
            bus.logger.Error("Event handler failed",
                zap.String("eventID", event.EventID()),
                zap.String("eventType", event.EventType()),
                zap.String("handler", fmt.Sprintf("%T", handler)),
                zap.Error(err),
            )
            // 繼續執行其他 handler（不中斷）
        }
    }

    return nil
}
```

---

## **5. FX 模組配置**

### **5.1 完整 FX 配置範例**

```go
// cmd/app/main.go
package main

import (
    "go.uber.org/fx"
    "gorm.io/gorm"

    "internal/application/event"
    "internal/application/transaction"
    pointsapp "internal/application/points"
    "internal/infrastructure/persistence"
)

func main() {
    fx.New(
        // === 基礎設施層 ===
        fx.Provide(NewLogger),
        fx.Provide(NewDatabase),
        fx.Provide(NewCache),

        // === Event Bus ===
        fx.Provide(func(logger *zap.Logger) event.EventBus {
            return event.NewInMemoryEventBus(logger)
        }),

        // === Unit of Work（含 Event Bus）===
        fx.Provide(func(db *gorm.DB, eventBus event.EventBus) transaction.TransactionManager {
            return transaction.NewUnitOfWork(db, eventBus)
        }),

        // === Repositories ===
        fx.Provide(func(db *gorm.DB) points.Repository {
            return persistence.NewGormPointsAccountRepository(db)
        }),

        // === Use Cases ===
        fx.Provide(pointsapp.NewEarnPointsUseCase),
        fx.Provide(pointsapp.NewDeductPointsUseCase),

        // === Event Handlers ===
        fx.Provide(event.NewPointsEarnedHandler),
        fx.Provide(event.NewPointsDeductedHandler),

        // === 註冊 Event Handlers ===
        fx.Invoke(func(
            eventBus event.EventBus,
            pointsEarnedHandler *event.PointsEarnedHandler,
            pointsDeductedHandler *event.PointsDeductedHandler,
        ) {
            eventBus.Subscribe(points.EventTypePointsEarned, pointsEarnedHandler)
            eventBus.Subscribe(points.EventTypePointsDeducted, pointsDeductedHandler)
        }),

        // === HTTP Server ===
        fx.Invoke(StartHTTPServer),
    ).Run()
}
```

---

## **6. 事件持久化與重試**

### **6.1 重試策略（帶指數退避）**

```go
// Application Layer - internal/application/event/retrying_event_bus.go
package event

import (
    "context"
    "fmt"
    "math"
    "time"

    "internal/domain"
)

type RetryingEventBus struct {
    baseEventBus EventBus
    dlq          DeadLetterQueue
    maxRetries   int
    logger       *zap.Logger
}

func NewRetryingEventBus(
    baseEventBus EventBus,
    dlq DeadLetterQueue,
    maxRetries int,
    logger *zap.Logger,
) *RetryingEventBus {
    return &RetryingEventBus{
        baseEventBus: baseEventBus,
        dlq:          dlq,
        maxRetries:   maxRetries,
        logger:       logger,
    }
}

func (bus *RetryingEventBus) Publish(event domain.DomainEvent) error {
    for attempt := 0; attempt < bus.maxRetries; attempt++ {
        err := bus.baseEventBus.Publish(event)
        if err == nil {
            return nil // 成功
        }

        // 指數退避（1s, 2s, 4s, 8s, ...）
        backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
        bus.logger.Warn("Event publish failed, retrying...",
            zap.String("eventID", event.EventID()),
            zap.Int("attempt", attempt+1),
            zap.Duration("backoff", backoffDuration),
            zap.Error(err),
        )

        time.Sleep(backoffDuration)
    }

    // 所有重試失敗 → Dead Letter Queue
    bus.logger.Error("Event publish failed after max retries, sending to DLQ",
        zap.String("eventID", event.EventID()),
        zap.Int("maxRetries", bus.maxRetries),
    )

    if err := bus.dlq.Add(event); err != nil {
        bus.logger.Error("Failed to add event to DLQ",
            zap.String("eventID", event.EventID()),
            zap.Error(err),
        )
    }

    return fmt.Errorf("event publish failed after %d retries", bus.maxRetries)
}

func (bus *RetryingEventBus) Subscribe(eventType string, handler EventHandler) {
    bus.baseEventBus.Subscribe(eventType, handler)
}
```

### **6.2 Dead Letter Queue 實現**

```go
// Application Layer - internal/application/event/dead_letter_queue.go
package event

import (
    "encoding/json"
    "time"

    "gorm.io/gorm"
    "internal/domain"
)

type DeadLetterMessage struct {
    ID        string    `gorm:"primaryKey"`
    EventType string    `gorm:"index"`
    Payload   []byte    `gorm:"type:jsonb"`
    Reason    string
    CreatedAt time.Time
}

type DeadLetterQueue interface {
    Add(event domain.DomainEvent) error
}

type GormDeadLetterQueue struct {
    db *gorm.DB
}

func NewGormDeadLetterQueue(db *gorm.DB) *GormDeadLetterQueue {
    return &GormDeadLetterQueue{db: db}
}

func (dlq *GormDeadLetterQueue) Add(event domain.DomainEvent) error {
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }

    msg := DeadLetterMessage{
        ID:        event.EventID(),
        EventType: event.EventType(),
        Payload:   payload,
        Reason:    "Max retries exceeded",
        CreatedAt: time.Now(),
    }

    return dlq.db.Create(&msg).Error
}
```

---

## **7. 監控與告警**

### **7.1 Metrics 收集（Prometheus）**

```go
// Application Layer - internal/application/event/instrumented_event_bus.go
package event

import (
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "internal/domain"
)

var (
    eventsPublishedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "events_published_total",
        Help: "Total number of events published",
    }, []string{"event_type"})

    eventsFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "events_failed_total",
        Help: "Total number of events that failed to publish",
    }, []string{"event_type"})

    eventPublishDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "event_publish_duration_seconds",
        Help:    "Time taken to publish an event",
        Buckets: prometheus.DefBuckets,
    }, []string{"event_type"})

    eventHandlerDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "event_handler_duration_seconds",
        Help:    "Time taken to handle an event",
        Buckets: prometheus.DefBuckets,
    }, []string{"event_type", "handler"})
)

type InstrumentedEventBus struct {
    baseEventBus EventBus
}

func NewInstrumentedEventBus(baseEventBus EventBus) *InstrumentedEventBus {
    return &InstrumentedEventBus{baseEventBus: baseEventBus}
}

func (bus *InstrumentedEventBus) Publish(event domain.DomainEvent) error {
    start := time.Now()
    err := bus.baseEventBus.Publish(event)
    duration := time.Since(start)

    eventPublishDuration.WithLabelValues(event.EventType()).Observe(duration.Seconds())

    if err != nil {
        eventsFailedTotal.WithLabelValues(event.EventType()).Inc()
    } else {
        eventsPublishedTotal.WithLabelValues(event.EventType()).Inc()
    }

    return err
}

func (bus *InstrumentedEventBus) Subscribe(eventType string, handler EventHandler) {
    bus.baseEventBus.Subscribe(eventType, handler)
}
```

### **7.2 Grafana Dashboard 查詢範例**

```promql
# 每分鐘發布的事件數量
rate(events_published_total[1m])

# 事件發布失敗率
rate(events_failed_total[1m]) / rate(events_published_total[1m])

# 事件發布延遲（P95）
histogram_quantile(0.95, rate(event_publish_duration_seconds_bucket[5m]))

# 各 Event Handler 的處理時間
histogram_quantile(0.95, rate(event_handler_duration_seconds_bucket[5m]))
```

---

## **8. 常見問題與最佳實踐**

### **8.1 Q: 事件處理失敗會回滾業務操作嗎？**

**A**: 不會。事件在**事務提交後**才發布，此時業務資料已持久化。

**解決方案**：
- 使用重試機制（指數退避）
- 使用 Dead Letter Queue 保存失敗事件
- Event Handler 必須設計為**冪等**（重複執行結果相同）

### **8.2 Q: 如何保證事件不被重複處理？（冪等性保證）**

**A**: Event Handler 必須實現冪等性檢查。

#### **8.2.1 為什麼需要冪等性？**

**問題場景：事件可能被重複投遞**

```
情境 1: Event Bus 重試機制
- Handler 處理成功但返回時網路中斷
- Event Bus 認為失敗，觸發重試
- 同一事件被處理兩次

情境 2: 訊息隊列 At-Least-Once Delivery
- Kafka/RabbitMQ 保證至少投遞一次
- 消費者處理完成但 commit offset 前當機
- 重啟後再次消費相同訊息

情境 3: Transactional Outbox Worker 重啟
- Outbox Worker 發布事件後，更新 published 欄位前當機
- Worker 重啟後再次發布相同事件
```

**後果**（無冪等性保護）：
- ❌ 重複發送通知（用戶收到多次 LINE 訊息）
- ❌ 重複計算積分（100 分變成 200 分）
- ❌ 重複創建審計記錄

---

#### **8.2.2 冪等性實現策略**

**策略 A: Cache-Based Idempotency（基於快取）**

```go
// Application Layer - internal/application/event/points_earned_handler.go
package event

import (
    "context"
    "fmt"
    "time"
)

type PointsEarnedHandler struct {
    notificationService notification.Service
    cache               Cache
    logger              *zap.Logger
}

func (h *PointsEarnedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    pointsEarnedEvent := event.(points.PointsEarned)

    // ✅ 步驟 1: 檢查事件是否已處理
    cacheKey := fmt.Sprintf("event:processed:%s", event.EventID())
    if h.cache.Exists(cacheKey) {
        h.logger.Warn("Event already processed, skipping",
            zap.String("eventID", event.EventID()),
        )
        return nil // 已處理，跳過
    }

    // 步驟 2: 執行業務邏輯
    message := fmt.Sprintf(
        "🎉 您獲得了 %d 積分！",
        pointsEarnedEvent.Amount.Value(),
    )

    err := h.notificationService.SendLineMessage(
        ctx,
        pointsEarnedEvent.MemberID.String(),
        message,
    )
    if err != nil {
        return err // 失敗則不標記，允許重試
    }

    // ✅ 步驟 3: 標記為已處理（24 小時 TTL）
    h.cache.Set(cacheKey, true, 24*time.Hour)

    return nil
}
```

**優勢**:
- ✅ 實現簡單
- ✅ 性能高（Redis 查詢快速）
- ✅ TTL 自動清理（避免無限增長）

**限制**:
- ❌ 依賴 Redis 可用性（Cache 故障會失去保護）
- ❌ TTL 過期後失去保護（24 小時後重複事件無法檢測）

---

**策略 B: Database-Based Idempotency（基於資料庫）**

```go
// Infrastructure Layer - internal/infrastructure/persistence/event_log.go
package persistence

import (
    "time"
    "gorm.io/gorm"
)

// ProcessedEventLog 已處理事件記錄表
type ProcessedEventLog struct {
    EventID      string    `gorm:"primaryKey;type:varchar(100)"`
    EventType    string    `gorm:"index;type:varchar(100)"`
    ProcessedAt  time.Time `gorm:"not null"`
    HandlerName  string    `gorm:"type:varchar(200)"`
}

type ProcessedEventLogRepository struct {
    db *gorm.DB
}

func (r *ProcessedEventLogRepository) IsProcessed(eventID string) (bool, error) {
    var count int64
    err := r.db.Model(&ProcessedEventLog{}).
        Where("event_id = ?", eventID).
        Count(&count).Error

    return count > 0, err
}

func (r *ProcessedEventLogRepository) MarkAsProcessed(eventID, eventType, handlerName string) error {
    log := ProcessedEventLog{
        EventID:     eventID,
        EventType:   eventType,
        ProcessedAt: time.Now(),
        HandlerName: handlerName,
    }

    return r.db.Create(&log).Error
}
```

```go
// Application Layer - Event Handler 使用資料庫檢查
func (h *PointsEarnedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    // ✅ 步驟 1: 檢查資料庫是否已處理
    isProcessed, err := h.eventLogRepo.IsProcessed(event.EventID())
    if err != nil {
        return err
    }

    if isProcessed {
        h.logger.Warn("Event already processed, skipping",
            zap.String("eventID", event.EventID()),
        )
        return nil
    }

    // 步驟 2: 執行業務邏輯
    err = h.notificationService.SendLineMessage(...)
    if err != nil {
        return err
    }

    // ✅ 步驟 3: 標記為已處理（寫入資料庫）
    return h.eventLogRepo.MarkAsProcessed(
        event.EventID(),
        event.EventType(),
        "PointsEarnedHandler",
    )
}
```

**優勢**:
- ✅ 100% 可靠（持久化到資料庫）
- ✅ 無 TTL 限制（永久保護）
- ✅ 可查詢歷史（稽核追蹤）

**限制**:
- ❌ 性能較低（資料庫查詢比 Redis 慢）
- ❌ 需要額外表（`processed_event_logs`）
- ❌ 無自動清理（需要定期歸檔）

---

**策略 C: Business-Based Idempotency（基於業務唯一鍵）**

```go
// 適用於有自然唯一鍵的場景
func (h *EarnPointsFromTransactionHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    txVerifiedEvent := event.(invoice.TransactionVerified)

    // ✅ 步驟 1: 檢查是否已存在此 sourceID 的積分交易
    exists, err := h.pointsTxRepo.ExistsBySourceID(
        ctx,
        string(points.SourceInvoice),
        txVerifiedEvent.TransactionID,
    )
    if err != nil {
        return err
    }

    if exists {
        h.logger.Warn("Points already earned for this transaction",
            zap.String("transactionID", txVerifiedEvent.TransactionID),
        )
        return nil // 已處理，跳過
    }

    // 步驟 2: 賺取積分（會創建 PointsTransaction 記錄）
    err = h.earnPointsUseCase.Execute(EarnPointsCommand{
        MemberID: txVerifiedEvent.MemberID,
        Amount:   calculatePoints(txVerifiedEvent.Amount),
        Source:   points.SourceInvoice,
        SourceID: txVerifiedEvent.TransactionID, // ← 唯一鍵
    })

    return err
}
```

```go
// Domain Layer - Repository 實現
func (r *GormPointsTransactionRepository) ExistsBySourceID(
    ctx shared.TransactionContext,
    source string,
    sourceID string,
) (bool, error) {
    db := r.extractDB(ctx)

    var count int64
    err := db.Model(&PointsTransactionModel{}).
        Where("source = ? AND source_id = ?", source, sourceID).
        Count(&count).Error

    return count > 0, err
}
```

**優勢**:
- ✅ 利用業務邏輯自然保護（無額外開銷）
- ✅ 資料一致性強（依賴資料庫唯一約束）
- ✅ 不需要額外的冪等性表

**限制**:
- ❌ 僅適用於有唯一業務鍵的場景
- ❌ 不適用於無副作用的 Handler（如發送通知）

---

#### **8.2.3 推薦方案（綜合策略）**

**方案：Hybrid Approach（混合策略）**

```go
type PointsEarnedHandler struct {
    notificationService notification.Service
    cache               Cache               // 快速檢查（第一道防線）
    eventLogRepo        ProcessedEventLogRepo // 可靠檢查（第二道防線）
    logger              *zap.Logger
}

func (h *PointsEarnedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    pointsEarnedEvent := event.(points.PointsEarned)

    // ✅ 第一道防線：Cache 快速檢查（低成本）
    cacheKey := fmt.Sprintf("event:processed:%s", event.EventID())
    if h.cache.Exists(cacheKey) {
        h.logger.Debug("Event already processed (cache hit)",
            zap.String("eventID", event.EventID()),
        )
        return nil
    }

    // ✅ 第二道防線：Database 可靠檢查（Cache Miss 時）
    isProcessed, err := h.eventLogRepo.IsProcessed(event.EventID())
    if err != nil {
        // 資料庫查詢失敗，為安全起見，返回錯誤（允許重試）
        return fmt.Errorf("failed to check idempotency: %w", err)
    }

    if isProcessed {
        // 資料庫確認已處理 → 更新 Cache（避免下次查資料庫）
        h.cache.Set(cacheKey, true, 24*time.Hour)
        h.logger.Warn("Event already processed (database hit)",
            zap.String("eventID", event.EventID()),
        )
        return nil
    }

    // 執行業務邏輯
    err = h.notificationService.SendLineMessage(
        ctx,
        pointsEarnedEvent.MemberID.String(),
        fmt.Sprintf("🎉 您獲得了 %d 積分！", pointsEarnedEvent.Amount.Value()),
    )
    if err != nil {
        return err // 失敗則不標記，允許重試
    }

    // ✅ 標記為已處理（同時寫入 Cache 和 Database）
    if err := h.eventLogRepo.MarkAsProcessed(event.EventID(), event.EventType(), "PointsEarnedHandler"); err != nil {
        h.logger.Error("Failed to mark event as processed in database",
            zap.String("eventID", event.EventID()),
            zap.Error(err),
        )
        // 繼續設置 Cache（部分失敗比完全失敗好）
    }

    h.cache.Set(cacheKey, true, 24*time.Hour)

    h.logger.Info("Event processed successfully",
        zap.String("eventID", event.EventID()),
    )

    return nil
}
```

**設計優勢**:
- ✅ **高性能**: 99% 請求由 Cache 攔截（< 1ms）
- ✅ **高可靠**: Cache 失效時 Database 兜底（100% 保護）
- ✅ **自修復**: Cache Miss 時自動回填 Cache
- ✅ **漸進式降級**: Database 故障時仍有 Cache 保護

---

#### **8.2.4 冪等性測試**

```go
// Test: 事件重複投遞應該被跳過
func TestPointsEarnedHandler_Handle_Idempotency(t *testing.T) {
    // Arrange
    mockCache := &MockCache{}
    mockEventLogRepo := &MockProcessedEventLogRepo{}
    mockNotificationService := &MockNotificationService{}
    handler := NewPointsEarnedHandler(mockNotificationService, mockCache, mockEventLogRepo, logger)

    event := points.NewPointsEarned(
        points.AccountID("ACC123"),
        points.MemberID("M456"),
        points.PointsAmount(10),
        points.SourceInvoice,
        "INV789",
        "發票驗證",
    )

    // Scenario 1: 第一次處理（Cache Miss, DB Miss）
    mockCache.On("Exists", mock.Anything).Return(false).Once()
    mockEventLogRepo.On("IsProcessed", event.EventID()).Return(false, nil).Once()
    mockNotificationService.On("SendLineMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
    mockEventLogRepo.On("MarkAsProcessed", event.EventID(), event.EventType(), mock.Anything).Return(nil).Once()
    mockCache.On("Set", mock.Anything, true, 24*time.Hour).Return(nil).Once()

    err := handler.Handle(context.Background(), event)
    assert.NoError(t, err)

    // Scenario 2: 第二次處理（Cache Hit）→ 應該跳過
    mockCache.On("Exists", mock.Anything).Return(true).Once()

    err = handler.Handle(context.Background(), event)
    assert.NoError(t, err)

    // Assert: 通知服務只被調用一次
    mockNotificationService.AssertNumberOfCalls(t, "SendLineMessage", 1)

    // Scenario 3: Cache 過期後，Database Hit → 應該跳過
    mockCache.On("Exists", mock.Anything).Return(false).Once()
    mockEventLogRepo.On("IsProcessed", event.EventID()).Return(true, nil).Once()
    mockCache.On("Set", mock.Anything, true, 24*time.Hour).Return(nil).Once()

    err = handler.Handle(context.Background(), event)
    assert.NoError(t, err)

    // Assert: 通知服務仍然只被調用一次
    mockNotificationService.AssertNumberOfCalls(t, "SendLineMessage", 1)

    mockCache.AssertExpectations(t)
    mockEventLogRepo.AssertExpectations(t)
    mockNotificationService.AssertExpectations(t)
}
```

---

#### **8.2.5 冪等性最佳實踐總結**

| 策略 | 適用場景 | 性能 | 可靠性 |
|------|---------|------|--------|
| **Cache-Based** | 發送通知、更新快取 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Database-Based** | 關鍵業務操作、稽核日誌 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Business-Based** | 有自然唯一鍵的業務 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Hybrid** | 高流量 + 高可靠要求 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |

**推薦**:
- **積分計算**: Business-Based（`sourceID` 唯一約束）
- **發送通知**: Hybrid（Cache + Database）
- **稽核日誌**: Database-Based（100% 完整性要求）

### **8.3 Q: 事件應該同步處理還是非同步處理？**

**建議**：

| 場景 | 處理方式 | 原因 |
|------|---------|------|
| 發送通知（LINE、Email） | 非同步 | 外部 API 可能延遲，不阻塞業務 |
| 更新統計資料（快取） | 非同步 | 容許最終一致性 |
| 協調其他聚合（積分計算） | 同步 | 保證業務一致性 |
| 稽核日誌 | 同步 | 100% 完整性要求（見 ADR-004） |

### **8.4 Q: 事件應該包含整個 Aggregate 還是僅包含 ID？**

**建議**：僅包含必要資料（避免事件過大）。

```go
// ✅ 好：僅包含必要資料
type PointsEarned struct {
    AccountID   AccountID
    MemberID    MemberID
    Amount      PointsAmount
    Source      PointsSource
}

// ❌ 差：包含整個聚合（事件過大，耦合嚴重）
type PointsEarned struct {
    Account *PointsAccount // ❌ 包含整個聚合
}
```

**原則**：Event Handler 如果需要更多資料，應該透過 Repository 查詢。

### **8.5 Q: 如何測試 Event Handlers?**

```go
func TestPointsEarnedHandler_Handle(t *testing.T) {
    // Arrange
    mockNotificationService := &MockNotificationService{}
    mockCache := &MockCache{}
    handler := NewPointsEarnedHandler(mockNotificationService, mockCache, logger)

    event := points.NewPointsEarned(
        points.AccountID("ACC123"),
        points.MemberID("M456"),
        points.PointsAmount(10),
        points.SourceInvoice,
        "INV789",
        "發票驗證",
    )

    mockCache.On("Exists", mock.Anything).Return(false)
    mockNotificationService.On("SendLineMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil)
    mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)

    // Act
    err := handler.Handle(context.Background(), event)

    // Assert
    assert.NoError(t, err)
    mockNotificationService.AssertExpectations(t)
    mockCache.AssertExpectations(t)
}
```

---

## **總結**

### **事件處理架構關鍵原則**

1. **Domain Layer 收集事件，不發布**
2. **Application Layer 在事務提交後發布事件**
3. **Event Handlers 必須冪等**
4. **使用重試 + Dead Letter Queue 保證可靠性**
5. **監控事件發布與處理的延遲與失敗率**

### **檢查清單**

- [ ] Domain Events 繼承 `BaseDomainEvent`，包含 `EventID`, `EventType`, `OccurredAt`
- [ ] Aggregates 使用 `RecordEvent()` 收集事件（不立即發布）
- [ ] Application Layer 使用 Unit of Work 在事務提交後發布事件
- [ ] Event Handlers 實現冪等性檢查（Cache Key: `event:processed:{EventID}`）
- [ ] Event Bus 包含重試機制與 Dead Letter Queue
- [ ] FX 配置正確註冊所有 Event Handlers
- [ ] Prometheus Metrics 監控事件發布與處理
- [ ] 測試覆蓋 Event Handlers 的成功與失敗場景

---

**相關文檔**:
- `/docs/architecture/ddd/07-aggregate-design-principles.md` - 事件生命週期（第 8.5 節）
- `/docs/architecture/ddd/11-dependency-rules.md` - Unit of Work 模式
- `/docs/architecture/decisions/ADR-004-audit-log-consistency.md` - 同步 vs 非同步事件
