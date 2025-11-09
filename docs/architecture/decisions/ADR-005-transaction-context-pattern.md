# ADR-005: Transaction Context Pattern Choice

**Date**: 2025-01-09
**Status**: Accepted
**Supersedes**: N/A

---

## Context

在實踐 Clean Architecture 時，面臨**資料庫事務管理**的設計挑戰：

### **問題場景：業務操作需要事務保證**

```go
// Use Case: 積分賺取需要在事務中完成
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    // 1. 更新積分帳戶
    // 2. 創建交易記錄
    // 3. 記錄稽核日誌
    // ✅ 三個操作必須在同一個事務中，保證原子性
}
```

**問題**：如何在 Domain Layer 與 Application Layer 使用資料庫事務，而不違反 Clean Architecture 依賴規則？

### **可行方案比較**

#### **方案 A：Domain Layer 直接依賴 `*gorm.DB`**（❌ 違反 Clean Architecture）

```go
// Domain Layer - internal/domain/repository/
package repository

import "gorm.io/gorm"  // ❌ Domain Layer 依賴基礎設施

type PointsAccountRepository interface {
    FindByMemberID(tx *gorm.DB, memberID MemberID) (*PointsAccount, error)
    Update(tx *gorm.DB, account *PointsAccount) error
}
```

**問題**：
- ❌ Domain Layer 依賴 GORM（違反 Dependency Rule）
- ❌ 無法替換 ORM（緊耦合）
- ❌ 測試時必須 Mock GORM（測試困難）

#### **方案 B：Repository 自己管理事務**（❌ 違反 SRP）

```go
// Domain Layer - internal/domain/repository/
type PointsAccountRepository interface {
    BeginTransaction() error
    Commit() error
    Rollback() error
    FindByMemberID(memberID MemberID) (*PointsAccount, error)
    Update(account *PointsAccount) error
}
```

**問題**：
- ❌ Repository 承擔過多職責（資料存取 + 事務管理）
- ❌ 無法跨 Repository 使用同一事務
- ❌ Application Layer 必須手動管理事務生命週期

```go
// Application Layer 需要手動管理事務（容易出錯）
repo.BeginTransaction()
account := repo.FindByMemberID(memberID)
account.EarnPoints(...)
repo.Update(account)
repo.Commit()  // ❌ 忘記 Rollback？
```

#### **方案 C：Application Layer 傳遞 `context.Context`**（❌ 無法攜帶事務）

```go
// Domain Layer
type PointsAccountRepository interface {
    FindByMemberID(ctx context.Context, memberID MemberID) (*PointsAccount, error)
    Update(ctx context.Context, account *PointsAccount) error
}
```

**問題**：
- ❌ `context.Context` 無法攜帶事務信息（除非使用 `WithValue`，但不推薦）
- ❌ 需要定義 Context Key（容易衝突）
- ❌ 類型不安全（`ctx.Value(txKey)` 返回 `interface{}`）

---

## Decision

**採用 Transaction Context Pattern（事務上下文模式）**：

1. **Domain Layer 定義 `TransactionContext` 接口**（標記接口，遵循「接口屬於調用者」原則）
2. **Infrastructure Layer 實現 `TransactionContext`**（封裝 `*gorm.DB` 事務）
3. **Domain Layer Repository 接受 `TransactionContext` 參數**（不知道內部實現）
4. **Infrastructure Layer Repository 實現從 `TransactionContext` 提取事務**（類型斷言）

---

## Rationale

### **模式架構**

```
┌─────────────────────────────────────────────────────────┐
│                   Domain Layer (內層)                    │
│  ┌───────────────────────────────────────────────────┐  │
│  │ TransactionContext interface {}                   │  │
│  │ (標記接口，無方法)                                 │  │
│  │ ✅ 接口屬於調用者（Domain Layer）                  │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ TransactionManager interface {                    │  │
│  │   InTransaction(fn func(ctx TransactionContext)   │  │
│  │     error) error                                  │  │
│  │ }                                                 │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ PointsAccountRepository interface {               │  │
│  │   FindByMemberID(                                 │  │
│  │     ctx TransactionContext,  // ✅ 同層依賴       │  │
│  │     memberID MemberID                             │  │
│  │   ) (*PointsAccount, error)                       │  │
│  │ }                                                 │  │
│  └───────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     ▲ (依賴反轉)
                     │
┌────────────────────┴────────────────────────────────────┐
│                  Application Layer                       │
│  ┌───────────────────────────────────────────────────┐  │
│  │ Use Cases 調用 TransactionManager                 │  │
│  │ ✅ 依賴 Domain Layer 接口                         │  │
│  └───────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     ▲ (依賴反轉)
                     │
┌────────────────────┴────────────────────────────────────┐
│                Infrastructure Layer (外層)               │
│  ┌───────────────────────────────────────────────────┐  │
│  │ gormTransactionContext struct {                   │  │
│  │   tx *gorm.DB  // ✅ 實際的資料庫事務             │  │
│  │ }                                                 │  │
│  │ ✅ 實現 Domain 的 TransactionContext 接口         │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ GormTransactionManager struct {                   │  │
│  │   InTransaction(fn) error {                       │  │
│  │     tx := db.Begin()                              │  │
│  │     ctx := &gormTransactionContext{tx: tx}        │  │
│  │     fn(ctx)  // ✅ 傳遞封裝的 Context             │  │
│  │     tx.Commit()                                   │  │
│  │   }                                               │  │
│  │ }                                                 │  │
│  │ ✅ 實現 Domain 的 TransactionManager 接口         │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │ GormPointsAccountRepository struct {              │  │
│  │   extractDB(ctx TransactionContext) *gorm.DB {    │  │
│  │     if txCtx, ok := ctx.(*gormTransactionContext) │  │
│  │       return txCtx.tx  // ✅ 類型斷言提取事務     │  │
│  │     return r.db  // ✅ 非事務模式                 │  │
│  │   }                                               │  │
│  │ }                                                 │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### **完整實現範例**

#### **Step 1: Domain Layer 定義接口**

```go
// Domain Layer - internal/domain/shared/transaction.go
package shared

// TransactionContext 封裝事務上下文
// 這是一個標記接口，Infrastructure Layer 會實現具體的事務封裝
// ✅ 接口屬於調用者（Domain Layer），而非實現者（Infrastructure Layer）
type TransactionContext interface {
    // 標記接口：僅用於傳遞上下文，不暴露方法
}

// TransactionManager 管理事務生命週期
// ✅ 定義在 Domain Layer，由 Application Layer 使用，Infrastructure Layer 實現
type TransactionManager interface {
    InTransaction(fn func(ctx TransactionContext) error) error
}
```

#### **Step 2: Domain Layer Repository 使用**

```go
// Domain Layer - internal/domain/repository/
package repository

import "internal/domain/shared"  // ✅ 同層依賴，遵循 Clean Architecture

type PointsAccountRepository interface {
    FindByMemberID(
        ctx shared.TransactionContext,  // ✅ 接受 TransactionContext
        memberID MemberID,
    ) (*PointsAccount, error)

    Update(
        ctx shared.TransactionContext,
        account *PointsAccount,
    ) error
}
```

#### **Step 3: Infrastructure Layer 實現**

```go
// Infrastructure Layer - internal/infrastructure/transaction/
package transaction

import (
    "internal/domain/shared"  // ✅ 依賴 Domain Layer 接口
    "gorm.io/gorm"
)

// gormTransactionContext 實現 TransactionContext 接口
// 內部持有 *gorm.DB 事務對象
type gormTransactionContext struct {
    tx *gorm.DB  // ✅ 實際的資料庫事務
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
```

```go
// Infrastructure Layer - internal/infrastructure/persistence/
package persistence

import (
    "internal/domain/shared"
    "internal/domain/points"
    "gorm.io/gorm"
)

// GormPointsAccountRepository 實現 Repository 接口
type GormPointsAccountRepository struct {
    db *gorm.DB  // 預設連接（非事務）
}

func (r *GormPointsAccountRepository) Update(
    ctx shared.TransactionContext,
    account *points.PointsAccount,
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
    if txCtx, ok := ctx.(*transaction.gormTransactionContext); ok {
        return txCtx.tx  // ✅ 返回事務連接
    }
    return r.db  // ✅ 返回普通連接（非事務模式）
}
```

#### **Step 4: Application Layer 使用**

```go
// Application Layer - internal/application/points/
package pointsapp

import (
    "internal/domain/shared"  // ✅ 依賴 Domain Layer 接口
    "internal/domain/points/repository"
)

type EarnPointsUseCase struct {
    txManager   shared.TransactionManager      // ✅ 依賴 Domain 接口
    accountRepo repository.PointsAccountRepository
    auditRepo   auditrepository.AuditLogRepository
}

func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    // ✅ 使用 TransactionManager 管理事務
    return uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        // 1. 業務邏輯（傳遞 TransactionContext）
        account, err := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
        if err != nil {
            return err
        }

        account.EarnPoints(cmd.Amount, cmd.Source, cmd.SourceID, cmd.Description)

        // 2. 持久化（同一事務）
        if err := uc.accountRepo.Update(ctx, account); err != nil {
            return err
        }

        // 3. 稽核日誌（同一事務）
        auditLog := createAuditLog(account, cmd)
        if err := uc.auditRepo.Create(ctx, auditLog); err != nil {
            return err
        }

        return nil  // ✅ 全部成功，事務提交
    })
}
```

---

## Why This Pattern?

### **1. 保持 Clean Architecture 依賴方向**

```
✅ 依賴方向正確（全部指向內層）：
Infrastructure Layer → Domain Layer (實現 TransactionContext 和 TransactionManager)
Application Layer → Domain Layer (使用 TransactionManager)
Domain Layer → 無外層依賴（僅定義接口）

✅ 接口屬於調用者原則：
TransactionContext 定義在 Domain Layer（Repository 的調用者）
而非 Infrastructure Layer（實現者）

❌ 絕不允許：
Domain Layer → Infrastructure Layer (*gorm.DB)
Domain Layer → Application Layer（舊設計的錯誤）
```

### **2. 類型安全 + 靈活性**

```go
// ✅ 類型安全：TransactionContext 是強類型接口
func (r *Repository) Update(
    ctx shared.TransactionContext,  // ✅ 編譯時檢查
    account *PointsAccount,
) error

// ❌ 不安全：使用 context.WithValue
func (r *Repository) Update(
    ctx context.Context,
    account *PointsAccount,
) error {
    tx := ctx.Value("tx")  // ❌ interface{}，無類型檢查
    gormTx := tx.(*gorm.DB)  // ❌ 可能 panic
}
```

### **3. 支援非事務模式**

```go
// ✅ 非事務模式：直接調用 Repository
account := accountRepo.FindByMemberID(nil, memberID)

// Infrastructure Layer 處理 nil Context
func (r *GormPointsAccountRepository) extractDB(
    ctx appTx.TransactionContext,
) *gorm.DB {
    if ctx == nil {
        return r.db  // ✅ 使用普通連接
    }
    if txCtx, ok := ctx.(*gormTransactionContext); ok {
        return txCtx.tx  // ✅ 使用事務連接
    }
    return r.db
}
```

### **4. 可測試性**

```go
// 測試時，可以提供 Mock TransactionContext
type MockTransactionContext struct{}

func TestEarnPointsUseCase(t *testing.T) {
    mockCtx := &MockTransactionContext{}

    // Mock Repository 可以忽略 TransactionContext
    mockRepo := &MockPointsAccountRepository{
        FindByMemberIDFunc: func(ctx shared.TransactionContext, id MemberID) (*PointsAccount, error) {
            // ✅ 測試時忽略事務，直接返回數據
            return testAccount, nil
        },
    }

    uc := &EarnPointsUseCase{
        txManager: &MockTxManager{},  // Mock 事務管理器
        accountRepo: mockRepo,
    }

    err := uc.Execute(cmd)
    assert.NoError(t, err)
}
```

---

## Consequences

### **優勢**

1. **保持 Clean Architecture**：
   - Domain Layer 不依賴基礎設施（GORM、SQL）
   - Domain Layer 不依賴 Application Layer
   - 依賴方向正確（Infrastructure → Application → Domain，全部指向內層）
   - 接口屬於調用者（TransactionContext 在 Domain Layer，遵循 DIP）

2. **可替換性**：
   - 可替換 GORM → sqlx → pgx（只需修改 Infrastructure Layer）
   - Domain Layer 不受影響

3. **可測試性**：
   - Domain Layer 可獨立測試（不需要 Mock GORM）
   - Application Layer 可 Mock TransactionManager

4. **事務管理集中化**：
   - Application Layer 控制事務邊界
   - Repository 僅負責資料存取（SRP）

5. **支援非事務模式**：
   - 允許傳遞 `nil` 以使用普通連接
   - 靈活性高

### **代價**

1. **類型斷言成本**：
   - Infrastructure Layer 需要類型斷言提取事務
   - 每次 Repository 調用都需要執行斷言（效能損耗極小）

2. **增加抽象層**：
   - 需要定義 `TransactionContext` 和 `TransactionManager` 接口
   - 增加代碼量

3. **開發者學習曲線**：
   - 團隊需要理解 Transaction Context Pattern
   - 需要文檔說明此模式

4. **Opaque Context 可能被誤用**：
   - 開發者可能嘗試在 Domain Layer 使用 Context 的方法（但標記接口無方法）
   - 需要 Code Review 防止濫用

### **緩解策略**

#### **1. 文檔與範例**

```markdown
# Transaction Context Pattern 使用指南

## DO ✅
- Application Layer 使用 `txManager.InTransaction()`
- Repository 接受 `TransactionContext` 參數
- Infrastructure Layer 從 Context 提取事務

## DON'T ❌
- Domain Layer 不調用 `TransactionContext` 的方法（標記接口無方法）
- Domain Layer 不依賴 GORM/SQL
- Repository 不管理事務生命週期
```

#### **2. Code Review Checklist**

- [ ] Repository 方法是否接受 `TransactionContext` 參數？
- [ ] Domain Layer 是否依賴 GORM？（❌ 禁止）
- [ ] Infrastructure Layer 是否使用類型斷言提取事務？（✅ 必須）
- [ ] Use Case 是否使用 `txManager.InTransaction()`？（✅ 推薦）

#### **3. 單元測試範例**

```go
// 提供 Mock TransactionContext 和 Mock TransactionManager
type MockTransactionContext struct{}

type MockTransactionManager struct{}

func (m *MockTransactionManager) InTransaction(
    fn func(ctx shared.TransactionContext) error,
) error {
    // ✅ 測試時直接執行，不需要真實事務
    return fn(&MockTransactionContext{})
}
```

---

## Alternative Approaches (Rejected)

### **Approach 1: Application Layer 定義 TransactionContext**（本ADR初版採用，已修正）

**為什麼拒絕**：
- 違反「接口屬於調用者」原則
- Domain Layer 的 Repository 需要依賴 Application Layer 的 TransactionContext
- 造成 Domain Layer → Application Layer 依賴（違反 Clean Architecture）
- **已修正為**: TransactionContext 定義在 Domain Layer

### **Approach 2: 使用 `context.Context.WithValue`**

**為什麼拒絕**：
- 不安全（`ctx.Value()` 返回 `interface{}`）
- 容易產生 Key 衝突
- 違反 Go 官方建議（不推薦用 WithValue 傳遞業務數據）

### **Approach 3: Application Layer 傳遞 `*gorm.DB`**

**為什麼拒絕**：
- Application Layer 依賴 GORM（違反依賴反轉）
- 無法替換 ORM

---

## References

- `/docs/architecture/ddd/11-dependency-rules.md` - Section 4（Transaction Context Pattern 完整實現）
- Robert C. Martin - "Clean Architecture" (Chapter 22: The Clean Architecture)
- Go Best Practices - "Don't use context.WithValue for request-scoped data"
- Vaughn Vernon - "Implementing Domain-Driven Design" (Chapter 12: Repositories)

---

## Notes

- **2025-01-09**: 初始版本，基於 uncle-bob-code-mentor 建議創建
- **2025-01-09 (修訂)**: 修正依賴方向違規，將 TransactionContext 從 Application Layer 移至 Domain Layer
  - **原因**: 遵循「接口屬於調用者」原則（Dependency Inversion Principle）
  - **影響**: Domain Layer 不再依賴 Application Layer，完全符合 Clean Architecture
  - **感謝**: uncle-bob-code-mentor 代理指出依賴方向錯誤
- 此模式是 Clean Architecture 與實務需求的平衡點
- Opaque Context Pattern 在 Go 社群已有成功案例（如 `http.Request.Context()`）
- 必須在文檔與 Code Review 中強調正確用法
