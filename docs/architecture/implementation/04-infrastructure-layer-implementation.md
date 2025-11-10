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
    "github.com/yourorg/bar_crm/internal/domain/points"
    "github.com/yourorg/bar_crm/internal/domain/points/repository"
    "github.com/yourorg/bar_crm/internal/domain/shared"
    gormModels "github.com/yourorg/bar_crm/internal/infrastructure/persistence/gorm"
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

    // 樂觀鎖：使用聚合當前的版本號（已由聚合自己遞增）
    // WHERE 條件檢查舊版本號（version - 1），確保並發控制
    oldVersion := account.GetVersion() - 1

    // 樂觀鎖更新（WHERE version = oldVersion 確保並發控制）
    result := db.Model(&gormModels.PointsAccountModel{}).
        Where("account_id = ? AND version = ?", model.AccountID, oldVersion).
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
    "github.com/yourorg/bar_crm/internal/domain/shared"
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
    "github.com/yourorg/bar_crm/internal/domain/member"
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
- ✅ Event Bus 介面定義在 `internal/domain/shared/event.go`
- ✅ Event Bus 實現在 `internal/infrastructure/messaging/event_bus.go`
- ✅ Event Handlers 在 `internal/application/events/`
- ✅ 透過 DI (Dependency Injection) 註冊 handlers 到 Event Bus
- ❌ Infrastructure Layer 絕不直接依賴 Application Layer 的 Event Handlers

### 4.2 Event Bus 實現

**文件**: `internal/infrastructure/messaging/event_bus.go`

```go
package messaging

import (
    "sync"
    "github.com/yourorg/bar_crm/internal/domain/shared"
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
    "github.com/yourorg/bar_crm/internal/domain/shared"
    "github.com/yourorg/bar_crm/internal/infrastructure/messaging"
    "github.com/yourorg/bar_crm/internal/application/events/points"
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

### 4.4 架構優勢

**依賴方向**:
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

---

**下一步**: 閱讀 [05-Presentation Layer 實現指南](./05-presentation-layer-implementation.md)
