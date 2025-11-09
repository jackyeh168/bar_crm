# ADR-003: Domain Layer Accepting Application DTOs

**Date**: 2025-01-09
**Status**: Accepted
**Supersedes**: N/A

---

## Context

在實踐 Clean Architecture 與 DDD 時，面臨**跨 Bounded Context 數據傳遞**的設計挑戰：

### **業務場景：積分計算需要發票數據**

```
┌─────────────────┐         需要發票數據         ┌─────────────────┐
│  Points Context │ ◄────────────────────────── │ Invoice Context │
│                 │                              │                 │
│ PointsAccount   │                              │ Invoice         │
│ EarnPoints()    │                              │ InvoiceNumber   │
│                 │                              │ Amount          │
└─────────────────┘                              └─────────────────┘
```

**問題**：`PointsAccount.EarnPoints()` 方法需要發票數據來計算積分，但不應該直接依賴 `Invoice` 實體（避免 Bounded Context 耦合）。

### **可行方案比較**

#### **方案 A：Domain Layer 定義自己的 DTO**

```go
// Domain Layer - internal/domain/points/
package points

// ❌ Domain Layer 定義 DTO（違反 Clean Architecture）
type InvoiceDataDTO struct {
    InvoiceNumber string
    Amount        int
    Date          time.Time
}

func (a *PointsAccount) EarnPointsFromInvoice(dto InvoiceDataDTO) error {
    // 使用 DTO 計算積分
}
```

**問題**：
- Domain Layer 不應該定義 DTO（DTO 屬於 Application Layer 職責）
- 增加 Domain Layer 複雜度

#### **方案 B：Application Layer 轉換為 Value Objects**

```go
// Application Layer - internal/application/points/
package pointsapp

func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    // ❌ Application Layer 創建 Value Objects
    invoiceNumber := points.NewInvoiceNumber(cmd.InvoiceNumber)
    amount := points.NewMoney(cmd.Amount)

    account.EarnPointsFromInvoice(invoiceNumber, amount, cmd.Date)
}
```

**問題**：
- 過多參數（Primitive Obsession）
- Value Objects 包含驗證邏輯，但 Application Layer 已經驗證過（重複驗證）
- 每次新增參數需要修改方法簽名

#### **方案 C：Domain Layer 接受 Application DTOs**（本方案）

```go
// Application Layer - internal/application/dto/
package dto

type InvoiceDTO struct {
    InvoiceNumber string
    Amount        int
    Date          time.Time
}

// Domain Layer - internal/domain/points/
package points

import "internal/application/dto"  // ✅ 允許依賴 DTO

func (a *PointsAccount) EarnPointsFromInvoice(invoice dto.InvoiceDTO) error {
    // 內部轉換為 Value Objects
    invoiceNumber := NewInvoiceNumber(invoice.InvoiceNumber)
    amount := NewMoney(invoice.Amount)

    // 業務邏輯
    points := a.calculatePoints(amount)
    a.earnedPoints = a.earnedPoints.Add(points)

    return nil
}
```

---

## Decision

**允許 Domain Layer 方法接受 Application Layer 的 DTOs**，但需遵守以下規則：

1. **DTOs 定義在 Application Layer**（`internal/application/dto/`）
2. **DTOs 僅用於方法參數**（不在 Domain Entity 中儲存）
3. **Domain Layer 內部必須轉換為 Value Objects**（不直接使用 DTO 的原始值）
4. **DTOs 僅用於跨 Bounded Context 數據傳遞**（不用於同一 Context 內）

---

## Rationale

### **符合實用主義 (Pragmatism over Purity)**

| 原則 | 嚴格遵守 | 實用主義調整 |
|------|---------|-------------|
| **Clean Architecture** | Domain Layer 不依賴任何外層 | Domain Layer 可依賴 Application DTOs（僅數據結構） |
| **DDD Bounded Context** | Context 間完全隔離 | 使用 DTO 作為 Anti-Corruption Layer |
| **開發效率** | 每次參數變更需要修改多處 | 僅需修改 DTO 定義 |

### **依賴方向仍然正確**

```
┌──────────────────────────────────────────────┐
│          Presentation Layer                   │
└───────────────┬──────────────────────────────┘
                │
                ▼
┌──────────────────────────────────────────────┐
│          Application Layer                    │
│  ┌────────────────────────────────┐          │
│  │  DTOs (Plain Data Structures)  │          │
│  └────────────────────────────────┘          │
└───────────────┬──────────────────────────────┘
                │
                ▼ (✅ 允許依賴 DTOs)
┌──────────────────────────────────────────────┐
│          Domain Layer                         │
│  - 內部轉換 DTO → Value Objects              │
│  - 不儲存 DTO 引用                            │
└──────────────────────────────────────────────┘
```

**關鍵原則**：
- ✅ DTOs 是純數據結構（無業務邏輯），不會污染 Domain Layer
- ✅ Domain Layer 僅在方法簽名中使用 DTO，內部立即轉換為 Value Objects
- ✅ 依賴方向：`Domain → Application DTOs`（不依賴 Application Services）

### **範例：3 步模式**

#### **Step 1: Application Layer 定義 DTO**

```go
// Application Layer - internal/application/dto/
package dto

type InvoiceDTO struct {
    InvoiceNumber string    `json:"invoice_number"`
    Amount        int       `json:"amount"`
    InvoiceDate   time.Time `json:"invoice_date"`
}
```

#### **Step 2: Domain Layer 接受 DTO 並轉換**

```go
// Domain Layer - internal/domain/points/
package points

import "internal/application/dto"

func (a *PointsAccount) EarnPointsFromInvoice(
    invoice dto.InvoiceDTO,
) error {
    // ✅ 內部轉換為 Value Objects（驗證邏輯在此執行）
    invoiceNumber, err := NewInvoiceNumber(invoice.InvoiceNumber)
    if err != nil {
        return err  // 驗證失敗
    }

    amount, err := NewMoney(invoice.Amount)
    if err != nil {
        return err
    }

    // ✅ 使用 Value Objects 執行業務邏輯
    points := a.calculatePoints(amount)
    a.earnedPoints = a.earnedPoints.Add(points)

    // ✅ 發出 Domain Event（不包含 DTO）
    a.RecordEvent(PointsEarnedEvent{
        AccountID:     a.id,
        Amount:        points,
        Source:        PointsSourceInvoice,
        SourceID:      invoiceNumber.String(),
    })

    return nil
}
```

#### **Step 3: Application Layer 使用**

```go
// Application Layer - internal/application/points/
package pointsapp

func (uc *EarnPointsFromInvoiceUseCase) Execute(
    cmd EarnPointsFromInvoiceCommand,
) error {
    // 1. 從 Invoice Context 取得發票數據（透過 Repository 或 Event）
    invoice := uc.invoiceRepo.FindByNumber(cmd.InvoiceNumber)

    // 2. 轉換為 DTO（跨 Context 數據傳遞）
    invoiceDTO := dto.InvoiceDTO{
        InvoiceNumber: invoice.Number(),
        Amount:        invoice.Amount(),
        InvoiceDate:   invoice.Date(),
    }

    // 3. Domain Layer 接受 DTO 並處理業務邏輯
    account := uc.accountRepo.FindByMemberID(cmd.MemberID)
    if err := account.EarnPointsFromInvoice(invoiceDTO); err != nil {
        return err
    }

    // 4. 持久化
    return uc.accountRepo.Update(account)
}
```

---

## Consequences

### **優勢**

1. **避免 Bounded Context 耦合**：
   - Points Context 不直接依賴 Invoice Entity
   - 使用 DTO 作為 Anti-Corruption Layer

2. **簡化方法簽名**：
   - 避免 Primitive Obsession（多個原始型別參數）
   - 新增參數僅需修改 DTO 定義

3. **保持 Domain 純粹性**：
   - Domain Layer 內部仍使用 Value Objects
   - 驗證邏輯集中在 Value Object 構造函數

4. **提升可測試性**：
   - 測試時可直接構造 DTO（無需 Mock Entity）
   - Domain Layer 測試不依賴其他 Context

### **代價**

1. **輕微違反 Clean Architecture**：
   - Domain Layer 依賴 Application Layer 的 DTO（雖然只是數據結構）
   - 需要在文檔中明確說明此例外

2. **額外轉換成本**：
   - DTO → Value Objects 轉換增加代碼量
   - 需要驗證兩次（Application Layer 輸入驗證 + Value Object 構造驗證）

3. **開發者需理解規則**：
   - 必須遵守「DTO 僅用於方法參數」規則
   - 需要 Code Review 防止濫用（如在 Entity 中儲存 DTO）

### **緩解策略**

#### **1. 文檔明確說明例外**

```markdown
# Clean Architecture 依賴規則（帶例外）

✅ 允許：Domain Layer 方法接受 Application DTOs
❌ 禁止：Domain Layer Entity 儲存 DTO 引用
❌ 禁止：Domain Layer 調用 Application Services
```

#### **2. Code Review Checklist**

- [ ] DTO 是否定義在 `internal/application/dto/`？
- [ ] Domain Entity 是否儲存 DTO 引用？（❌ 禁止）
- [ ] Domain Method 是否內部轉換為 Value Objects？（✅ 必須）
- [ ] Domain Event 是否包含 DTO？（❌ 禁止，應使用 Value Objects）

#### **3. 使用 Linter 檢查**

```go
// 可以配置 golangci-lint 禁止 Domain Entity 儲存 DTO
type PointsAccount struct {
    id    AccountID
    // ❌ Linter 應報錯：Domain Entity 不應儲存 DTO
    // invoice dto.InvoiceDTO
}
```

---

## References

- `/docs/architecture/ddd/11-dependency-rules.md` - Section 3.2（DTO vs Value Object）
- `/docs/architecture/ddd/02-bounded-contexts.md` - 跨上下文數據傳遞
- Robert C. Martin - "Clean Architecture" (Chapter 22: The Clean Architecture)
- Eric Evans - "Domain-Driven Design" (Chapter 14: Maintaining Model Integrity)

---

## Notes

- **2025-01-09**: 初始版本，基於 uncle-bob-code-mentor 建議創建
- 此決策是實用主義調整（Pragmatism），在嚴格 Clean Architecture 與開發效率間取得平衡
- 必須在 Code Review 中嚴格檢查 DTO 使用規則
