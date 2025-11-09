# ADR-001: Why DDD over CRUD

**Date**: 2025-01-09
**Status**: Accepted
**Supersedes**: N/A

---

## Context

餐廳會員管理 LINE Bot 系統需要處理以下複雜業務場景：

1. **積分計算邏輯複雜**：
   - 發票金額轉換積分（可動態調整轉換率）
   - 問卷完成獎勵積分
   - 積分使用與扣除
   - 積分歷史追蹤與重新計算

2. **多個業務子系統**：
   - 會員註冊與管理（Membership Context）
   - 發票驗證與匹配（Invoice Context）
   - 問卷系統（Survey Context）
   - 積分管理（Points Context）
   - 審計日誌（Audit Context）

3. **外部系統整合**：
   - LINE Platform（會員身份綁定）
   - iChef POS 系統（發票驗證）
   - Google OAuth（管理員登入）

4. **業務規則頻繁變動**：
   - 積分轉換率調整（促銷活動）
   - 問卷題型擴充
   - 發票驗證規則變更

**問題**：是否採用傳統 CRUD（Create-Read-Update-Delete）架構，還是 Domain-Driven Design (DDD) 架構？

---

## Decision

**採用 Domain-Driven Design (DDD) 戰略設計與戰術模式**：

- 使用 Bounded Contexts 分離不同業務關注點
- 使用 Aggregates 封裝業務不變性規則
- 使用 Value Objects 封裝業務概念與驗證邏輯
- 使用 Domain Services 處理跨實體業務邏輯
- 使用 Domain Events 實現鬆耦合的上下文協作

---

## Rationale

### **DDD 優勢**

| 需求 | CRUD 局限性 | DDD 解決方案 |
|------|-------------|-------------|
| **業務規則封裝** | 業務邏輯散落在 Service Layer，難以追蹤 | 充血模型（Rich Domain Model）將規則封裝在 Entity/Value Object |
| **上下文隔離** | 所有業務共用同一套 Model，耦合嚴重 | Bounded Contexts 明確邊界，各自演化 |
| **不變性保護** | 手動檢查約束條件，容易遺漏 | Aggregate Root 強制執行不變性規則 |
| **統一語言** | 技術術語（DTO、Service、Manager）與業務脫節 | Ubiquitous Language 確保代碼即業務文件 |
| **變更影響範圍** | 一處修改可能影響全系統 | Context Map 明確依賴關係，減少變更衝擊 |

### **具體範例**

#### **業務規則封裝**

```go
// ❌ CRUD 風格：業務邏輯在 Service Layer
func (s *PointsService) EarnPoints(memberID string, amount int) error {
    account := s.repo.FindByMemberID(memberID)

    // 業務規則散落在 Service 中
    if amount <= 0 {
        return errors.New("amount must be positive")
    }

    account.EarnedPoints += amount
    s.repo.Update(account)
    return nil
}

// ✅ DDD 風格：業務邏輯在 Domain Entity
func (a *PointsAccount) EarnPoints(
    amount PointsAmount,  // Value Object 已驗證
    source PointsSource,
    sourceID string,
    description string,
) error {
    // Aggregate 保護不變性規則
    if amount.Value() <= 0 {
        return ErrInvalidPointsAmount
    }

    a.earnedPoints = a.earnedPoints.Add(amount)  // Value Object 不可變

    // 發出 Domain Event（不需要知道誰訂閱）
    a.RecordEvent(PointsEarnedEvent{...})

    return nil
}
```

#### **上下文隔離**

```go
// ❌ CRUD 風格：所有上下文共用同一個 Transaction Model
type Transaction struct {
    ID            int
    MemberID      int       // Points Context 需要
    InvoiceNumber string    // Invoice Context 需要
    SurveyID      *int      // Survey Context 需要
    Amount        int       // 所有人都需要
}
// 任何一個上下文變更都可能影響其他上下文

// ✅ DDD 風格：各 Context 有獨立模型
// Points Context
type PointsTransaction struct {
    id          TransactionID
    accountID   AccountID
    amount      PointsAmount
    source      PointsSource  // invoice / survey / admin
    sourceID    string
}

// Invoice Context
type InvoiceRecord struct {
    id              InvoiceID
    invoiceNumber   InvoiceNumber
    amount          Money
    status          InvoiceStatus
}

// 使用 DTO 在 Application Layer 進行跨 Context 數據傳遞
```

---

## Consequences

### **優勢**

1. **可維護性提升**：
   - 業務邏輯集中在 Domain Layer，修改點明確
   - Bounded Contexts 隔離變更影響範圍

2. **可測試性提升**：
   - Domain Layer 純 Go 代碼，無需 Mock 資料庫
   - Value Objects 可獨立測試驗證邏輯

3. **團隊溝通改善**：
   - 使用統一語言（Ubiquitous Language）減少誤解
   - Context Map 明確團隊協作邊界

4. **技術債務降低**：
   - 新增功能遵循既有模式，不會累積技術債
   - 重構範圍限制在單一 Context 內

### **代價**

1. **初期開發成本**：
   - 需要額外設計 Aggregates、Value Objects、Repository Interfaces
   - 學習曲線較陡（團隊需要理解 DDD 概念）

2. **代碼量增加**：
   - Value Objects、Domain Events 增加文件數量
   - Repository Interface + Implementation 雙重代碼

3. **性能考量**：
   - Aggregate 載入可能觸發多次資料庫查詢（需使用 Lightweight Aggregates 模式）
   - Domain Events 需要額外事件總線機制

### **緩解策略**

1. **分階段導入**：
   - 核心域（Points Management）優先使用完整 DDD
   - 支撐域（Invoice Verification）使用簡化 DDD
   - 通用域（Audit Log）可使用 CRUD

2. **實用主義調整**：
   - 允許 Domain Layer 接受 Application DTOs（見 ADR-003）
   - 使用 Transaction Context Pattern 避免基礎設施洩漏（見 ADR-005）

3. **團隊培訓**：
   - 提供 DDD 模式文檔（`docs/architecture/ddd/`）
   - Code Review 強制執行依賴規則
   - Pair Programming 加速知識轉移

---

## References

- `/docs/architecture/ddd/01-strategic-overview.md` - DDD 戰略設計總覽
- `/docs/architecture/ddd/02-bounded-contexts.md` - 8 個 Bounded Contexts 定義
- `/docs/architecture/ddd/04-tactical-design.md` - DDD 戰術模式實踐
- `/docs/architecture/ddd/11-dependency-rules.md` - Clean Architecture 依賴規則

---

## Notes

- **2025-01-09**: 初始版本，基於 uncle-bob-code-mentor 建議創建
