# ADR-002: Why Lightweight Aggregates Over Rich Object Graphs

**Date**: 2025-01-09
**Status**: Accepted
**Supersedes**: N/A

---

## Context

在設計 `PointsAccount` Aggregate 時，面臨以下設計選擇：

### **方案 A：Rich Object Graph（豐富對象圖）**

```go
type PointsAccount struct {
    id             AccountID
    memberID       MemberID
    earnedPoints   PointsAmount
    usedPoints     PointsAmount

    // ❌ 載入所有交易記錄（可能上千筆）
    transactions   []PointsTransaction
}

// 查詢交易記錄直接從記憶體取得
func (a *PointsAccount) GetTransactionHistory() []PointsTransaction {
    return a.transactions  // 快速，但載入成本高
}
```

### **方案 B：Lightweight Aggregate（輕量級聚合）**

```go
type PointsAccount struct {
    id             AccountID
    memberID       MemberID
    earnedPoints   PointsAmount
    usedPoints     PointsAmount

    // ✅ 不載入交易記錄，僅保留必要的聚合數據
}

// 需要查詢交易記錄時，透過 Repository
type PointsTransactionRepository interface {
    FindByAccountID(accountID AccountID, opts QueryOptions) ([]PointsTransaction, error)
}
```

**問題**：是否在 Aggregate 中保留所有關聯對象（Rich Object Graph），還是僅保留必要狀態（Lightweight Aggregate）？

---

## Decision

**採用 Lightweight Aggregate（輕量級聚合）設計模式**：

1. **Aggregate 僅保留聚合計算值**（如 `earnedPoints`、`usedPoints`）
2. **不保留無界集合**（如 `transactions []PointsTransaction`）
3. **使用 Repository 按需查詢歷史數據**
4. **使用 Pagination / Filtering 限制查詢範圍**

---

## Rationale

### **Rich Object Graph 的問題**

| 問題 | 影響 | 範例 |
|------|------|------|
| **記憶體消耗** | 每個 Account 載入可能上千筆交易 | 1000 會員 × 500 筆交易 = 50 萬物件常駐記憶體 |
| **載入時間** | Eager Loading 導致延遲 | 載入單一帳戶需要 500ms（JOIN 查詢） |
| **N+1 問題** | 批次載入時觸發大量 SQL | 載入 100 個帳戶 = 1 + 100 次查詢 |
| **快取失效** | 交易新增後需要清除整個帳戶快取 | 無法精細控制快取粒度 |
| **事務邊界混亂** | 修改帳戶 + 修改交易需要兩個寫鎖 | 容易產生 Deadlock |

### **Lightweight Aggregate 的優勢**

#### **1. 效能提升**

```go
// ❌ Rich Object Graph：載入帳戶必須載入所有交易
account := repo.FindByMemberID(memberID)
// SQL: SELECT * FROM accounts WHERE member_id = ?
//      SELECT * FROM transactions WHERE account_id = ?  (500 rows)
// 總耗時: 500ms

// ✅ Lightweight Aggregate：僅載入聚合計算值
account := accountRepo.FindByMemberID(memberID)
// SQL: SELECT id, member_id, earned_points, used_points FROM accounts WHERE member_id = ?
// 總耗時: 10ms
```

#### **2. 按需查詢**

```go
// 使用場景：顯示會員積分餘額（不需要交易記錄）
func (uc *GetMemberBalanceUseCase) Execute(memberID MemberID) (BalanceDTO, error) {
    account := uc.accountRepo.FindByMemberID(memberID)  // ✅ 僅載入帳戶（快）

    return BalanceDTO{
        EarnedPoints: account.EarnedPoints().Value(),
        UsedPoints:   account.UsedPoints().Value(),
        Balance:      account.Balance().Value(),
    }, nil
}

// 使用場景：顯示交易歷史（需要分頁查詢）
func (uc *GetTransactionHistoryUseCase) Execute(
    memberID MemberID,
    page int,
) ([]TransactionDTO, error) {
    account := uc.accountRepo.FindByMemberID(memberID)  // ✅ 載入帳戶

    // ✅ 按需查詢交易記錄（分頁）
    txs := uc.txRepo.FindByAccountID(account.ID(), QueryOptions{
        Limit:  20,
        Offset: (page - 1) * 20,
        Order:  "created_at DESC",
    })

    return toTxDTOs(txs), nil
}
```

#### **3. 快取策略精細化**

```go
// ✅ 帳戶快取（長期有效，變更頻率低）
cache.Set("account:"+memberID, account, 1*time.Hour)

// ✅ 交易列表快取（短期有效，變更頻率高）
cache.Set("transactions:"+accountID+":page:1", txs, 5*time.Minute)

// 新增交易時，只需清除交易列表快取，不影響帳戶快取
cache.Delete("transactions:" + accountID + ":*")
```

#### **4. 事務邊界清晰**

```go
// ✅ 修改 Aggregate 本身（聚合根負責不變性）
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    return uc.unitOfWork.InTransaction(func(ctx TransactionContext) error {
        // 載入 Aggregate
        account := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)

        // 業務邏輯（在 Aggregate 內部）
        account.EarnPoints(cmd.Amount, cmd.Source, cmd.SourceID, cmd.Description)

        // 持久化（單一 Aggregate，事務邊界清晰）
        uc.accountRepo.Update(ctx, account)

        return nil
    })
}
// 事務鎖定範圍：僅 accounts 表的單一行
```

---

## Consequences

### **優勢**

1. **效能可預測**：
   - 載入 Aggregate 的時間固定（與交易數量無關）
   - 避免 N+1 查詢問題

2. **可擴展性**：
   - 支援百萬級會員，每個會員上萬筆交易
   - 分頁查詢避免一次載入過多數據

3. **快取友好**：
   - 帳戶數據可長期快取
   - 交易列表可短期快取或不快取

4. **事務簡化**：
   - 事務僅鎖定 Aggregate Root
   - 減少 Deadlock 風險

### **代價**

1. **額外查詢**：
   - 需要交易記錄時，必須額外查詢 Repository
   - 增加一次資料庫往返

2. **一致性視圖**：
   - 帳戶與交易記錄分兩次查詢，可能不一致（Read Committed 隔離級別）
   - 緩解方案：使用 `FOR UPDATE` 或 Snapshot Isolation

3. **開發複雜度**：
   - 需要設計額外的 `PointsTransactionRepository`
   - 需要在 Application Layer 組合數據

### **緩解策略**

#### **1. 使用 Query Object 封裝查詢邏輯**

```go
// Application Layer - Query Service
type PointsQueryService struct {
    accountRepo AccountRepository
    txRepo      TransactionRepository
}

func (s *PointsQueryService) GetAccountSummary(
    memberID MemberID,
) (AccountSummaryDTO, error) {
    // 組合查詢邏輯
    account := s.accountRepo.FindByMemberID(memberID)
    recentTxs := s.txRepo.FindRecentByAccountID(account.ID(), 5)

    return AccountSummaryDTO{
        Balance:      account.Balance().Value(),
        RecentTxs:    toTxDTOs(recentTxs),
    }, nil
}
```

#### **2. 使用事務保證一致性視圖**

```go
// 需要一致性視圖時，在同一個事務中查詢
func (s *PointsQueryService) GetConsistentView(
    memberID MemberID,
) (AccountSummaryDTO, error) {
    return s.unitOfWork.InTransaction(func(ctx TransactionContext) error {
        account := s.accountRepo.FindByMemberID(ctx, memberID)
        txs := s.txRepo.FindByAccountID(ctx, account.ID(), ...)

        // 在同一個事務中，保證視圖一致性
        return toSummaryDTO(account, txs), nil
    })
}
```

#### **3. 僅在關鍵業務流程保留少量關聯**

```go
// ✅ 允許保留有界集合（數量可控）
type Survey struct {
    id          SurveyID
    title       string
    description string

    // ✅ 問題數量可控（通常 < 10 題），可直接載入
    questions   []SurveyQuestion  // 有界集合
}

// ❌ 不保留無界集合
type Survey struct {
    responses   []SurveyResponse  // ❌ 可能上萬筆，不應載入
}
```

---

## References

- `/docs/architecture/ddd/07-aggregate-design-principles.md` - Aggregate 設計原則（第 8.4 節）
- `/docs/architecture/ddd/04-tactical-design.md` - Repository 模式實踐
- Martin Fowler - "Patterns of Enterprise Application Architecture" (Repository Pattern)
- Vernon Vaughn - "Implementing Domain-Driven Design" (Chapter 10: Aggregates)

---

## Notes

- **2025-01-09**: 初始版本，基於 uncle-bob-code-mentor 建議創建
- 本決策與 ADR-003（Domain Layer 接受 DTOs）相輔相成：避免聚合間直接引用
