# Dependency Rules（依賴規則）

> **版本**: 1.0
> **最後更新**: 2025-01-08

本章節定義系統各層之間的依賴方向和約束，確保符合 Clean Architecture 的 Dependency Rule。

---

## **12.1 核心原則**

**Dependency Rule**:
> 源代碼依賴必須**只能指向內層**，朝向更高層級的策略。

```
┌──────────────────────────────────────┐
│      Presentation Layer              │
│      (外層 - 最具體)                  │
│  ┌────────────────────────────────┐  │
│  │    Application Layer           │  │
│  │  ┌──────────────────────────┐  │  │
│  │  │    Domain Layer          │  │  │
│  │  │    (內層 - 最抽象)        │  │  │
│  │  └──────────────────────────┘  │  │
│  └────────────────────────────────┘  │
│  Infrastructure Layer (插件)         │
└──────────────────────────────────────┘

✅ 允許的依賴方向: 外 → 內
❌ 禁止的依賴方向: 內 → 外
```

---

## **12.2 允許的依賴關係**

### **✅ Presentation Layer 可以依賴**:
```go
// ✅ 可以 import Application Layer
import "myapp/internal/application/usecases"

// ✅ 可以 import Domain Layer (僅用於 DTO 轉換)
import "myapp/internal/domain/points"

// ❌ 禁止 import Infrastructure Layer
// import "myapp/internal/infrastructure/gorm"
```

**範例 (Gin HTTP Handler)**:
```go
type PointsHandler struct {
    earnPointsUseCase *usecases.EarnPointsUseCase
}

func (h *PointsHandler) HandleTransaction(c *gin.Context) {
    cmd := usecases.EarnPointsCommand{...}
    result, err := h.earnPointsUseCase.Execute(cmd)
    c.JSON(200, result)
}
```

---

### **✅ Application Layer 可以依賴**:
```go
// ✅ 可以 import Domain Layer
import "myapp/internal/domain/points"
import "myapp/internal/domain/points/repository"

// ❌ 禁止 import Infrastructure Layer
// import "myapp/internal/infrastructure/gorm"

// ❌ 禁止 import Presentation Layer
// import "myapp/internal/presentation/http"
```

**範例 (Application Service)**:
```go
type EarnPointsUseCase struct {
    accountRepo repository.PointsAccountRepository
    ruleService *points.ConversionRuleValidationService
}

func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    account, _ := uc.accountRepo.FindByMemberID(cmd.MemberID)
    rule, _ := uc.ruleService.GetRuleForDate(cmd.InvoiceDate)
    account.EarnPoints(points, source, sourceID, description)
    uc.accountRepo.Update(account)
}
```

**事務管理模式** (Application Layer 職責):

### **Transaction Context Pattern 完整說明**

**問題**：如何在不污染 Domain Layer 的情況下進行事務管理？

**❌ 錯誤做法 1：Repository 接口洩漏基礎設施細節**
```go
// Domain Layer - Repository 接口
type PointsAccountRepository interface {
    UpdateInTransaction(account *PointsAccount, db *sql.Tx) error  // ← 錯誤！
}

// 問題：
// 1. Domain 接口依賴 Infrastructure 類型（*sql.Tx）
// 2. 違反 Dependency Inversion Principle
// 3. 難以測試（Mock 需要真實的 *sql.Tx）
// 4. 換數據庫時需要修改 Domain 接口
```

**❌ 錯誤做法 2：聚合直接管理事務**
```go
// Domain Layer - Aggregate
func (a *PointsAccount) SaveWithTransaction(db *sql.Tx) error {
    // ❌ 聚合不應該知道如何保存自己
    // ❌ 聚合不應該依賴 Infrastructure
}
```

**✅ 正確做法：Transaction Context Pattern**

### **步驟 1：定義純淨的 Context 接口（Domain Layer）**

```go
// Domain Layer - internal/domain/shared/transaction.go
package shared

// TransactionContext 封裝事務上下文
// 這是一個標記接口，Infrastructure Layer 會實現具體的事務封裝
// 接口屬於調用者（Domain Layer），而非實現者（Infrastructure Layer）
type TransactionContext interface {
    // 標記接口：僅用於傳遞上下文，不暴露方法
    // Infrastructure Layer 會進行類型斷言提取實際事務
}

// TransactionManager 管理事務生命週期
// 定義在 Domain Layer，由 Application Layer 使用，Infrastructure Layer 實現
type TransactionManager interface {
    InTransaction(fn func(ctx TransactionContext) error) error
}
```

### **步驟 2：Domain Layer Repository 使用 Context**

```go
// Domain Layer - Repository 接口定義
package repository

import "myapp/internal/domain/shared"  // ✅ 同層依賴，無違反

type PointsAccountRepository interface {
    // ✅ 接受 Context，但不知道 Context 內部有什麼
    FindAll(ctx shared.TransactionContext) ([]*PointsAccount, error)
    Update(ctx shared.TransactionContext, account *PointsAccount) error
}

// Domain Layer 完全不知道 Context 內部有 *sql.Tx
// 只是將 Context 傳遞給 Repository（依賴反轉）
```

### **步驟 3：Infrastructure Layer 實現 Context（提取事務）**

```go
// Infrastructure Layer - internal/infrastructure/transaction/
package transaction

import (
    "gorm.io/gorm"
    "myapp/internal/domain/shared"  // ✅ 依賴 Domain Layer 接口
)

// gormTransactionContext 實現 TransactionContext 接口
// 內部持有 *gorm.DB 事務對象
type gormTransactionContext struct {
    tx *gorm.DB  // 實際的資料庫事務
}

// GormTransactionManager 實現 TransactionManager 接口
type GormTransactionManager struct {
    db *gorm.DB
}

func (tm *GormTransactionManager) InTransaction(
    fn func(ctx shared.TransactionContext) error,
) error {
    // 1. 開啟資料庫事務
    tx := tm.db.Begin()
    if tx.Error != nil {
        return tx.Error
    }

    // 2. 創建 Context（封裝事務）
    ctx := &gormTransactionContext{tx: tx}

    // 3. 執行業務邏輯
    if err := fn(ctx); err != nil {
        tx.Rollback()  // 回滾
        return err
    }

    // 4. 提交事務
    if err := tx.Commit().Error; err != nil {
        return err
    }

    return nil
}

// GormPointsAccountRepository 實現 Repository 接口
type GormPointsAccountRepository struct {
    db *gorm.DB  // 預設連接（非事務）
}

func (r *GormPointsAccountRepository) Update(
    ctx shared.TransactionContext,
    account *domain.PointsAccount,
) error {
    // ✅ Infrastructure Layer 從 Context 提取事務
    db := r.extractDB(ctx)

    // 使用提取的 DB 連接（可能是事務或普通連接）
    model := toGormModel(account)
    return db.Save(model).Error
}

// 私有方法：從 Context 提取 DB 連接
func (r *GormPointsAccountRepository) extractDB(
    ctx shared.TransactionContext,
) *gorm.DB {
    // 類型斷言：檢查是否是 gormTransactionContext
    if txCtx, ok := ctx.(*gormTransactionContext); ok {
        return txCtx.tx  // ✅ 返回事務連接
    }
    return r.db  // ✅ 返回普通連接（非事務模式）
}
```

### **步驟 4：Application Layer 使用 TransactionManager**

```go
// Application Layer - Use Case
package usecases

import (
    "myapp/internal/domain/shared"
    "myapp/internal/domain/points/repository"
)

type RecalculateAllPointsUseCase struct {
    txManager   shared.TransactionManager      // ✅ 依賴 Domain 接口
    accountRepo repository.PointsAccountRepository
    txRepo      repository.InvoiceTransactionRepository
    calculator  *points.PointsCalculationService
}

func (uc *RecalculateAllPointsUseCase) Execute(cmd RecalculateAllPointsCommand) error {
    // ✅ Application Layer 管理事務
    return uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        // 1. 查詢所有帳戶（在事務中）
        accounts := uc.accountRepo.FindAll(ctx)

        for _, account := range accounts {
            // 2. 查詢交易（在事務中）
            txs := uc.txRepo.FindVerifiedByMemberID(ctx, account.MemberID())

            // 3. Domain Layer 計算積分
            err := account.RecalculatePoints(txs, uc.calculator)
            if err != nil {
                return err  // ← 錯誤會觸發回滾
            }

            // 4. 保存聚合（在事務中）
            uc.accountRepo.Update(ctx, account)
        }

        return nil  // ← 成功會觸發提交
    })
}
```

### **完整數據流向**

```
1. Application Layer:
   txManager.InTransaction(fn)
        ↓
2. Infrastructure Layer:
   創建 gormTransactionContext{tx: *gorm.DB}
        ↓
3. Application Layer:
   調用 accountRepo.Update(ctx, account)
        ↓
4. Infrastructure Layer:
   extractDB(ctx) → 提取 tx (*gorm.DB)
   使用 tx.Save(model)
        ↓
5. Application Layer:
   fn 返回 nil → 提交
   fn 返回 error → 回滾
```

### **設計優勢**

| 層級 | 職責 | 依賴 |
|------|------|------|
| **Domain Layer** | 定義 TransactionContext 接口和 Repository 接口 | ❌ 不依賴外層 |
| **Application Layer** | 事務協調，調用 TransactionManager | ✅ 依賴 Domain 接口 |
| **Infrastructure Layer** | 實現 TransactionManager 和 Repository | ✅ 依賴 Domain 接口 |

**關鍵原則**:
- ✅ **接口屬於調用者**：TransactionContext 定義在 Domain Layer（Repository 的調用者）
- ✅ **事務管理是 Application Layer 的職責**
- ✅ **Context 是不透明的**：Domain Layer 不知道內部實現
- ✅ **類型斷言在 Infrastructure Layer**：只有 Infrastructure 知道具體類型
- ✅ **依賴方向正確**：Infrastructure → Application → Domain（全部指向內層）
- ✅ **易於測試**：Mock TransactionContext 即可
- ✅ **易於擴展**：換數據庫只需修改 Infrastructure Layer

**可選：更嚴格的類型安全（使用泛型）**

```go
// Go 1.18+ 泛型版本
type TransactionContext[T any] interface {
    GetTransaction() T
}

type gormTransactionContext struct {
    tx *gorm.DB
}

func (ctx *gormTransactionContext) GetTransaction() *gorm.DB {
    return ctx.tx
}

// Repository 接受泛型 Context
func (r *GormPointsAccountRepository) Update(
    ctx TransactionContext[*gorm.DB],
    account *domain.PointsAccount,
) error {
    db := ctx.GetTransaction()  // 類型安全
    return db.Save(toGormModel(account)).Error
}
```

但這種方法會讓 Domain 接口依賴具體數據庫類型（`*gorm.DB`），違反依賴反轉，因此不推薦。

**總結**:
- Transaction Context Pattern 平衡了類型安全和依賴反轉
- Application Layer 協調事務，Domain Layer 保持純淨
- Infrastructure Layer 負責具體實現和類型轉換

---

### **✅ Domain Layer 可以依賴**:
```go
// ✅ 可以 import 同層的其他 Domain 對象
import "myapp/internal/domain/member"  // 同層依賴

// ❌ 禁止 import Application Layer
// import "myapp/internal/application"  ← 錯誤！

// ❌ 禁止 import Infrastructure Layer
// import "myapp/internal/infrastructure/gorm"  ← 錯誤！

// ❌ 禁止 import Presentation Layer
// import "github.com/gin-gonic/gin"  ← 錯誤！

// ❌ 禁止 import 外部技術框架（除標準庫）
// import "gorm.io/gorm"  ← 錯誤！
```

**範例 (Aggregate Root - 輕量級設計)**:
```go
package points

type PointsAccount struct {
    accountID      AccountID
    memberID       member.MemberID
    earnedPoints   PointsAmount
    usedPoints     PointsAmount
    lastUpdatedAt  time.Time
}

func (a *PointsAccount) EarnPoints(amount PointsAmount, source PointsSource, sourceID string, desc string) error {
    if amount.Value() < 0 {
        return ErrNegativePointsAmount
    }

    a.earnedPoints = a.earnedPoints.Add(amount)
    a.lastUpdatedAt = time.Now()

    PublishEvent(PointsEarned{
        AccountID:   a.accountID,
        Amount:      amount,
        Source:      source,
        SourceID:    sourceID,
        Description: desc,
    })

    return nil
}
```

**設計原則**:
- ✅ **輕量級聚合**: 不包含 transactions 集合，避免無界增長
- ✅ **事件驅動**: 發布 PointsEarned 事件，Application Layer 處理交易記錄
- ✅ **職責分離**: 聚合負責狀態，PointsTransaction 獨立管理審計日誌
- ✅ **性能優化**: 加載快速，不受交易歷史數量影響

**跨上下文數據傳遞 - DTO 模式**:

### **DTO (Data Transfer Object) 定義與職責**

**DTO 是什麼？**
- **定義**: 純數據結構，用於在層之間或上下文之間傳輸數據
- **位置**: Application Layer (`internal/application/dto/`)
- **特性**: 無業務邏輯，無驗證，僅包含 getter/setter（或公開字段）
- **用途**: 解耦不同上下文的實體，避免直接依賴

**DTO vs Value Object 區別**:

| 特性 | DTO | Value Object |
|------|-----|--------------|
| **定義位置** | Application Layer | Domain Layer |
| **業務邏輯** | ❌ 無 | ✅ 有（不變性保護、驗證） |
| **可變性** | ✅ 可變（可修改） | ❌ 不可變 |
| **用途** | 跨層/跨上下文傳輸 | 封裝業務概念 |
| **驗證** | ❌ 無驗證 | ✅ 構造時驗證 |
| **相等性** | 結構相等 | 值相等 |

**示例對比**:

```go
// ❌ 錯誤：DTO 放在 Domain Layer
package domain

type VerifiedTransactionDTO struct {  // ← 錯誤！不應在 Domain Layer
    TransactionID   string
    Amount          decimal.Decimal
}

// ✅ 正確：DTO 定義在 Application Layer
package dto  // internal/application/dto/

type VerifiedTransactionDTO struct {
    TransactionID   string
    Amount          decimal.Decimal
    InvoiceDate     time.Time
    SurveySubmitted bool
}
// 無驗證邏輯，無業務規則，純數據結構

// ✅ 正確：Value Object 定義在 Domain Layer
package domain

type Money struct {
    amount decimal.Decimal
}

func NewMoney(amount decimal.Decimal) (Money, error) {
    if amount.LessThan(decimal.Zero) {
        return Money{}, ErrNegativeAmount  // ← 驗證邏輯
    }
    return Money{amount: amount}, nil
}

func (m Money) Add(other Money) Money {  // ← 業務邏輯
    return Money{amount: m.amount.Add(other.amount)}
}
```

### **跨上下文通信：正確使用 DTO**

**❌ 錯誤的做法 - Domain Layer 直接引用其他上下文的實體**:
```go
// Domain Layer - Points Context
package points

import "myapp/internal/domain/invoice"  // ← 錯誤！跨上下文耦合

func (a *PointsAccount) RecalculatePoints(
    invoiceTransactions []invoice.Transaction,  // ← 錯誤！直接依賴 Invoice 實體
    calculator PointsCalculationService,
) error {
    // 這會導致 Points Context 與 Invoice Context 緊耦合
}
```

**✅ 正確的做法 - Application Layer 使用 DTO 解耦**:

**步驟 1: Application Layer 定義 DTO**
```go
// Application Layer - internal/application/dto/
package dto

type VerifiedTransactionDTO struct {
    TransactionID   string
    Amount          decimal.Decimal
    InvoiceDate     time.Time
    SurveySubmitted bool
}
// 無業務邏輯，純數據結構
```

**步驟 2: Domain Layer 接受 DTO（作為參數）**
```go
// Domain Layer - Points Context
package points

import "myapp/internal/application/dto"  // ✅ 可以依賴 Application Layer 的 DTO

type PointsAccount struct {
    accountID    AccountID
    earnedPoints PointsAmount
    usedPoints   PointsAmount
}

func (a *PointsAccount) RecalculatePoints(
    transactionDTOs []dto.VerifiedTransactionDTO,  // ✅ 使用 DTO
    calculator PointsCalculationService,
) error {
    totalPoints := 0
    for _, txDTO := range transactionDTOs {
        // Domain Service 從 DTO 提取數據進行計算
        points := calculator.CalculateFromDTO(txDTO)
        totalPoints += points
    }

    if totalPoints < a.usedPoints.Value() {
        return ErrInsufficientEarnedPoints
    }

    oldPoints := a.earnedPoints
    a.earnedPoints = PointsAmount{value: totalPoints}

    a.publishEvent(PointsRecalculated{
        AccountID:  a.accountID,
        OldPoints:  oldPoints.Value(),
        NewPoints:  totalPoints,
    })

    return nil
}
```

**步驟 3: Application Layer 負責 Entity → DTO 轉換**
```go
// Application Layer - Use Case
package usecases

import (
    "myapp/internal/application/dto"
    "myapp/internal/domain/invoice"
    "myapp/internal/domain/points"
)

func (uc *RecalculatePointsUseCase) Execute(memberID string) error {
    // 1. 查詢 Invoice Context 的實體
    invoiceTxs := uc.invoiceTxRepo.FindVerifiedByMemberID(memberID)

    // 2. Application Layer 負責轉換（Entity → DTO）
    txDTOs := make([]dto.VerifiedTransactionDTO, len(invoiceTxs))
    for i, tx := range invoiceTxs {
        txDTOs[i] = dto.VerifiedTransactionDTO{
            TransactionID:   tx.ID(),
            Amount:          tx.Amount(),
            InvoiceDate:     tx.InvoiceDate(),
            SurveySubmitted: tx.IsSurveySubmitted(),
        }
    }

    // 3. 將 DTO 傳遞給 Domain Layer
    account := uc.pointsAccountRepo.FindByMemberID(memberID)
    err := account.RecalculatePoints(txDTOs, uc.calculator)
    if err != nil {
        return err
    }

    // 4. 保存聚合
    uc.pointsAccountRepo.Update(account)
    return nil
}
```

### **關鍵設計原則**

- ✅ **DTO 屬於 Application Layer** - 不是 Domain Layer
- ✅ **Domain Layer 可以接受 DTO 作為參數** - 但不定義 DTO
- ✅ **DTO 轉換在 Application Layer** - Use Case 負責 Entity ↔ DTO
- ✅ **Value Object 屬於 Domain Layer** - 包含業務邏輯和驗證
- ✅ **聚合不應引用其他聚合的實體** - 使用 DTO 解耦
- ❌ **DTO 不包含業務邏輯** - 只是數據載體
- ❌ **Domain Layer 不定義 DTO** - 避免將應用層關注點混入領域層

### **依賴方向總結**

```
Presentation Layer
       ↓ 依賴
Application Layer (定義 DTO)
       ↓ 依賴
Domain Layer (接受 DTO 作為參數，但不定義 DTO)
```

**為什麼 Domain Layer 可以依賴 Application Layer 的 DTO？**
- 在 Clean Architecture 中，內層（Domain）通常不應依賴外層（Application）
- 但 DTO 是一個**例外**，因為：
  1. DTO 是純數據結構，無業務邏輯
  2. DTO 的目的是解耦，而非引入業務邏輯
  3. 替代方案是在 Domain Layer 定義 DTO，但這會混淆 DTO 和 Value Object 的職責

**更嚴格的替代方案（可選）**:
如果要完全遵守依賴規則，可以在 Domain Layer 定義**輸入接口**：

```go
// Domain Layer
package points

type TransactionData interface {
    GetTransactionID() string
    GetAmount() decimal.Decimal
    GetInvoiceDate() time.Time
    IsSurveySubmitted() bool
}

func (a *PointsAccount) RecalculatePoints(
    transactions []TransactionData,  // 接口而非 DTO
    calculator PointsCalculationService,
) error {
    // ...
}

// Application Layer 的 DTO 實現此接口
package dto

func (d VerifiedTransactionDTO) GetTransactionID() string { return d.TransactionID }
func (d VerifiedTransactionDTO) GetAmount() decimal.Decimal { return d.Amount }
// ...
```

但這種方法增加了複雜度，實際項目中使用 DTO 作為參數更常見且實用。

---

### **✅ Infrastructure Layer 可以依賴**:
```go
// ✅ 可以 import Domain Layer (實現接口)
import "myapp/internal/domain/points/repository"

// ✅ 可以 import 外部技術框架
import "gorm.io/gorm"
import "github.com/redis/go-redis/v9"

// ❌ 禁止 import Application Layer
// import "myapp/internal/application"  ← 除非是事件處理器

// ❌ 禁止 import Presentation Layer
// import "myapp/internal/presentation/http"  ← 錯誤！
```

**範例 (Repository 實現)**:
```go
// ✅ 正確的做法
package gorm

import (
    "myapp/internal/domain/points"          // Domain 接口
    "myapp/internal/domain/points/repository"  // 倉儲接口
    "gorm.io/gorm"                          // 外部框架
)

// 實現 Domain 層定義的接口
type GormPointsAccountRepository struct {
    db *gorm.DB
}

// 接口方法實現
func (r *GormPointsAccountRepository) FindByMemberID(memberID points.MemberID) (*points.PointsAccount, error) {
    var model PointsAccountModel  // GORM 模型（Infrastructure 層）

    err := r.db.Where("member_id = ?", memberID.String()).First(&model).Error
    if err != nil {
        return nil, repository.ErrAccountNotFound
    }

    // 轉換為 Domain 對象
    return model.ToDomainEntity(), nil
}
```

---

## **12.3 接口所有權規則**

**重要原則**: 接口由**使用者**定義，而非實現者。

```go
// ✅ 正確: 接口在 Domain Layer 定義
package repository  // Domain Layer

type PointsAccountRepository interface {
    FindByMemberID(memberID MemberID) (*PointsAccount, error)
    Create(account *PointsAccount) error
    Update(account *PointsAccount) error
}

// ✅ 正確: 實現在 Infrastructure Layer
package gorm  // Infrastructure Layer

type GormPointsAccountRepository struct {
    db *gorm.DB
}

func (r *GormPointsAccountRepository) FindByMemberID(...) {...}
```

**為什麼？**
- Domain Layer 定義「需要什麼」（業務需求）
- Infrastructure Layer 實現「如何做」（技術實現）
- 這樣 Domain 不依賴 Infrastructure，符合 Dependency Inversion Principle

---

## **12.4 依賴檢查清單**

在代碼審查時，檢查以下規則：

**Domain Layer** (`internal/domain/`):
- ✅ 可以 import 標準庫 (`time`, `errors`, `fmt` 等)
- ✅ 可以 import 同層其他 Domain 包
- ✅ 可以 import `github.com/shopspring/decimal` (數學計算)
- ❌ 禁止 import `gorm.io/gorm`
- ❌ 禁止 import `github.com/gin-gonic/gin`
- ❌ 禁止 import `internal/application`
- ❌ 禁止 import `internal/infrastructure`
- ❌ 禁止 import `internal/presentation`

**Application Layer** (`internal/application/`):
- ✅ 可以 import `internal/domain/*`
- ✅ 可以 import 標準庫
- ❌ 禁止 import `internal/infrastructure/*` (除了注入點)
- ❌ 禁止 import `internal/presentation/*`
- ❌ 禁止 import `gorm.io/gorm`, `gin` 等技術框架

**Infrastructure Layer** (`internal/infrastructure/`):
- ✅ 可以 import `internal/domain/*` (實現接口)
- ✅ 可以 import `gorm.io/gorm`, `redis`, 等技術框架
- ⚠️ 謹慎 import `internal/application/*` (僅事件處理器)
- ❌ 禁止 import `internal/presentation/*`

**Presentation Layer** (`internal/presentation/`):
- ✅ 可以 import `internal/application/*`
- ✅ 可以 import `internal/domain/*` (僅用於 DTO 轉換)
- ✅ 可以 import `github.com/gin-gonic/gin` 等 Web 框架
- ❌ 禁止 import `internal/infrastructure/*`

---

## **12.5 依賴注入配置**

**使用 Uber FX 管理依賴**:

```go
// cmd/app/main.go
func main() {
    fx.New(
        // Infrastructure 模組（最底層，提供實現）
        providers.DatabaseModule,    // 提供 *gorm.DB
        providers.RedisModule,        // 提供 *redis.Client

        // Repository 模組（實現 Domain 接口）
        providers.RepositoryModule,   // 提供 repository.PointsAccountRepository

        // Domain Service 模組
        providers.PointsModule,       // 提供 PointsCalculationService

        // Application 模組（Use Cases）
        providers.UseCaseModule,      // 提供 EarnPointsUseCase

        // Presentation 模組（HTTP Handlers）
        providers.HandlerModule,      // 提供 PointsHandler
        providers.ServerModule,       // 提供 HTTP Server
    ).Run()
}
```

**依賴流向**:
```
HTTP Handler (Presentation)
    ↓ 依賴
Use Case (Application)
    ↓ 依賴
Repository Interface (Domain)
    ↑ 實現（依賴反轉）
Repository Implementation (Infrastructure)
```
