# ADR-004: Audit Log Consistency Model (Synchronous vs Asynchronous)

**Date**: 2025-01-09
**Status**: Accepted
**Supersedes**: N/A

---

## Context

稽核日誌系統（Audit Log）需要記錄所有資料變更操作，用於：

1. **法規遵循 (Compliance)**：
   - GDPR 要求記錄所有個人資料處理活動
   - 金融監管要求保存交易歷史 7 年以上

2. **問題追蹤 (Troubleshooting)**：
   - 會員回報積分異常時，查詢積分變動歷史
   - 發現重複交易時，追溯交易創建過程

3. **安全稽核 (Security Audit)**：
   - 偵測異常操作（如大量積分變動）
   - 追蹤管理員敏感操作

### **業務規則要求**

根據 `US-007-audit-log-system.md` 的業務規則：

| Rule ID | Description |
|---------|-------------|
| BR-007-01 | **同步記錄**: 資料變更操作與稽核日誌創建必須在同一資料庫事務中 |
| BR-007-02 | **原子性**: 如果稽核日誌寫入失敗，原操作必須回滾 |
| BR-007-04 | **完整性**: 所有 CREATE、UPDATE、DELETE 操作都必須記錄（100% 覆蓋率） |
| BR-007-16 | **效能要求**: 稽核日誌寫入不應顯著影響業務操作效能（< 50ms overhead） |

**問題**：如何在**資料一致性**與**系統效能**之間取得平衡？

---

## Decision

**採用同步記錄模式（Synchronous Audit Logging）**：

1. **稽核日誌與業務操作在同一資料庫事務中**
2. **稽核日誌寫入失敗 → 業務操作自動回滾**
3. **提交事務 → 業務資料與稽核日誌同時持久化**

---

## Rationale

### **方案比較**

#### **方案 A：同步記錄（Synchronous）**

```go
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    return uc.txManager.InTransaction(func(ctx TransactionContext) error {
        // 1. 業務操作
        account := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
        account.EarnPoints(cmd.Amount, cmd.Source, cmd.SourceID, cmd.Description)
        uc.accountRepo.Update(ctx, account)

        // 2. 稽核日誌（同一個事務）
        auditLog := createAuditLog(account, cmd)
        uc.auditRepo.Create(ctx, auditLog)  // ✅ 同一事務

        // 3. 兩者同時提交
        return nil  // Commit or Rollback together
    })
}
```

**優勢**：
- ✅ **100% 資料一致性**：業務資料與稽核日誌保證同步
- ✅ **簡單可靠**：不需要額外的事件總線或訊息佇列
- ✅ **易於調試**：所有操作在同一個事務中，失敗時完整回滾
- ✅ **符合 ACID 特性**：Atomicity, Consistency, Isolation, Durability

**代價**：
- ❌ **效能開銷**：每個業務操作多一次資料庫寫入（~10-50ms）
- ❌ **事務鎖定時間延長**：可能增加 Deadlock 風險
- ❌ **寫入放大**：每次業務操作產生額外的稽核日誌記錄

#### **方案 B：非同步記錄（Asynchronous via Event Bus）**

```go
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    return uc.txManager.InTransaction(func(ctx TransactionContext) error {
        // 1. 業務操作
        account := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
        account.EarnPoints(cmd.Amount, cmd.Source, cmd.SourceID, cmd.Description)
        uc.accountRepo.Update(ctx, account)

        // 2. 發出 Domain Event（事務提交後發布）
        ctx.AddEvent(PointsEarnedEvent{...})  // ✅ 非阻塞

        return nil
    })
}

// 事件處理器（在另一個 Goroutine 中執行）
func (h *AuditLogEventHandler) Handle(event PointsEarnedEvent) {
    auditLog := createAuditLogFromEvent(event)
    h.auditRepo.Create(auditLog)  // ❌ 可能失敗，無法回滾業務操作
}
```

**優勢**：
- ✅ **效能優秀**：業務操作不阻塞於稽核日誌寫入
- ✅ **解耦**：業務邏輯與稽核邏輯分離
- ✅ **可擴展**：可添加多個事件處理器（如發送通知、更新統計）

**代價**：
- ❌ **最終一致性 (Eventual Consistency)**：稽核日誌可能延遲寫入
- ❌ **資料遺失風險**：事件總線故障可能導致稽核日誌遺失（違反 BR-007-04）
- ❌ **複雜度增加**：需要 Event Bus、Dead Letter Queue、重試機制
- ❌ **調試困難**：業務操作成功但稽核日誌失敗時，難以追蹤

#### **方案 C：混合模式（Hybrid）**

```go
// 關鍵操作：同步記錄
func (uc *DeleteMemberUseCase) Execute(cmd DeleteMemberCommand) error {
    return uc.txManager.InTransaction(func(ctx TransactionContext) error {
        member := uc.memberRepo.FindByID(ctx, cmd.MemberID)
        uc.memberRepo.Delete(ctx, member)

        // ✅ 關鍵操作（刪除會員）必須同步記錄
        uc.auditRepo.Create(ctx, auditLog)

        return nil
    })
}

// 一般操作：非同步記錄
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    return uc.txManager.InTransaction(func(ctx TransactionContext) error {
        account.EarnPoints(...)
        uc.accountRepo.Update(ctx, account)

        // ❌ 一般操作（積分變動）可以非同步記錄
        ctx.AddEvent(PointsEarnedEvent{...})

        return nil
    })
}
```

**優勢**：
- ✅ 平衡效能與一致性

**代價**：
- ❌ **複雜度極高**：需要維護兩套稽核日誌機制
- ❌ **難以保證完整性**：如何定義「關鍵操作」與「一般操作」？
- ❌ **維護成本**：團隊需要理解何時用同步、何時用非同步

---

## Why Synchronous?

### **1. 法規遵循優先於效能**

```
稽核日誌完整性 > 系統效能

- GDPR 違規罰款：最高 €20,000,000 或全球年營收 4%
- 稽核日誌遺失無法事後補救
- 效能問題可透過優化解決（快取、索引、批次寫入）
```

### **2. 簡化系統架構**

```
同步模式：
┌───────────────────┐
│ Application Layer │
│  ┌─────────────┐  │
│  │ Use Case    │  │
│  └─────────────┘  │
│         │         │
│         ▼         │
│ ┌───────────────┐ │
│ │ Transaction   │ │
│ │ - Business Op │ │
│ │ - Audit Log   │ │
│ └───────────────┘ │
└───────────────────┘

非同步模式：
┌───────────────────┐       ┌───────────────────┐
│ Application Layer │──────>│ Event Bus         │
│  ┌─────────────┐  │       │  ┌─────────────┐  │
│  │ Use Case    │  │       │  │ RabbitMQ    │  │
│  └─────────────┘  │       │  └─────────────┘  │
└───────────────────┘       └──────┬────────────┘
                                    │
                            ┌───────▼────────────┐
                            │ Event Handlers     │
                            │  ┌─────────────┐   │
                            │  │ Audit Log   │   │
                            │  │ Handler     │   │
                            │  └─────────────┘   │
                            └────────────────────┘

額外需要：Event Bus、Dead Letter Queue、Retry Logic、Monitoring
```

### **3. 效能開銷可接受**

根據 BR-007-16 要求，稽核日誌寫入開銷應 < 50ms：

| 操作 | 平均耗時 | 說明 |
|------|---------|------|
| 業務操作（更新積分） | 20ms | UPDATE points_accounts |
| 稽核日誌寫入 | 10-15ms | INSERT audit_logs |
| **總計** | **30-35ms** | ✅ 符合 < 50ms 要求 |

**優化策略**：
- **索引優化**：在 `audit_logs` 表上創建適當索引（已設計）
- **批次寫入**（未來）：批次導入 iChef 資料時，使用 `CreateBatch()`
- **資料庫分區**：按月分區（已設計），減少單表大小
- **快取策略**：不快取稽核日誌寫入，但可快取稽核日誌查詢結果

### **4. 已有成功案例**

許多關鍵系統採用同步稽核日誌：

- **銀行系統**：所有交易記錄與稽核日誌同步寫入
- **AWS CloudTrail**：API 調用與稽核日誌同步記錄
- **Kubernetes Audit**：API Server 操作同步記錄稽核日誌

---

## Consequences

### **優勢**

1. **100% 資料一致性**：
   - 業務操作成功 ⇒ 稽核日誌必定存在
   - 業務操作失敗 ⇒ 稽核日誌不存在（無需清理）

2. **簡化故障排查**：
   - 所有操作在同一事務中，失敗時完整回滾
   - 不需要處理「業務成功但稽核失敗」的邊緣案例

3. **符合法規要求**：
   - 滿足 GDPR、金融監管的稽核要求
   - 可證明所有操作都有完整記錄

4. **架構簡單**：
   - 不需要 Event Bus、Message Queue
   - 減少維運成本與學習曲線

### **代價**

1. **效能開銷**：
   - 每個業務操作增加 10-15ms 延遲
   - 事務鎖定時間延長（可能增加 Deadlock 風險）

2. **寫入放大**：
   - 每次業務操作產生額外的稽核日誌記錄
   - 資料庫儲存成本增加（緩解：按月分區 + 歸檔）

3. **無法處理高並發**：
   - 同步寫入可能成為瓶頸（緩解：批次寫入 + 資料庫優化）

### **緩解策略**

#### **1. 資料庫分區（已設計）**

```sql
-- 按月分區，減少單表大小
CREATE TABLE audit_logs_2025_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

#### **2. 批次寫入（未來優化）**

```go
// 批次導入 iChef 資料時，批次寫入稽核日誌
func (uc *ImportIChefBatchUseCase) Execute(cmd ImportBatchCommand) error {
    return uc.txManager.InTransaction(func(ctx TransactionContext) error {
        // 批次創建交易
        for _, record := range cmd.Records {
            transaction := createTransaction(record)
            uc.txRepo.Create(ctx, transaction)
        }

        // ✅ 批次寫入稽核日誌（一次 INSERT 多筆）
        auditLogs := createAuditLogs(cmd.Records)
        uc.auditRepo.CreateBatch(ctx, auditLogs)

        return nil
    })
}
```

#### **3. 稽核日誌歸檔**

```go
// 定期歸檔舊稽核日誌（保留熱資料在主庫）
func (uc *ArchiveAuditLogsUseCase) Execute() error {
    olderThan := time.Now().AddDate(-1, 0, 0)  // 1 年前
    return uc.auditRepo.ArchiveOldLogs(olderThan)
}
```

#### **4. 監控與告警**

```go
// 監控稽核日誌寫入延遲
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    start := time.Now()
    err := uc.txManager.InTransaction(func(ctx TransactionContext) error {
        // ... business logic + audit log
    })
    duration := time.Since(start)

    // ✅ 監控：如果超過閾值（50ms），發出告警
    if duration > 50*time.Millisecond {
        uc.logger.Warn("audit log write slow", zap.Duration("duration", duration))
        uc.metrics.RecordSlowAuditLog(duration)
    }

    return err
}
```

---

## Future Considerations

### **階段性演進策略**

#### **Phase 1: 同步模式（當前）**
- 所有操作同步記錄稽核日誌
- 滿足法規要求與 100% 資料一致性
- 監控效能指標，識別瓶頸

#### **Phase 2: 混合模式（V3.2+）**
- 如果效能成為瓶頸，考慮混合模式：
  - **關鍵操作**（會員刪除、積分規則變更）：同步記錄
  - **一般操作**（積分賺取、交易創建）：非同步記錄（使用 Event Sourcing）

#### **Phase 3: Event Sourcing（V3.4+）**
- 完整事件溯源架構
- 所有業務操作寫入 Event Store
- 稽核日誌從 Event Store 重建
- 支援時間旅行查詢

---

## References

- `/docs/product/stories/US-007-audit-log-system.md` - 稽核日誌系統完整需求
- `/docs/architecture/ddd/02-bounded-contexts.md` - Audit Context 設計
- Martin Fowler - "Audit Log Pattern" (https://martinfowler.com/eaaDev/AuditLog.html)
- AWS CloudTrail - "Event History and Compliance" (Synchronous audit logging example)

---

## Notes

- **2025-01-09**: 初始版本，基於 uncle-bob-code-mentor 建議創建
- 本決策優先考慮**法規遵循與資料一致性**，犧牲部分效能
- 效能問題可透過資料庫優化、分區、歸檔等策略緩解
- 如果未來業務量成長導致效能瓶頸，可重新評估採用混合模式或 Event Sourcing
