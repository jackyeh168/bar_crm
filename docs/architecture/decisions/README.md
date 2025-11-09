# Architecture Decision Records (ADRs)

> **版本**: 1.0
> **最後更新**: 2025-01-09

---

## 📖 Overview

本目錄包含餐廳會員管理 LINE Bot 系統的所有重要架構決策記錄（Architecture Decision Records, ADRs）。

每個 ADR 記錄了：
- **Context**: 決策的背景與問題
- **Decision**: 最終採用的方案
- **Rationale**: 決策的理由與分析
- **Consequences**: 決策的優勢、代價與緩解策略

---

## 📚 ADR Index

| ADR ID | Title | Status | Date | Summary |
|--------|-------|--------|------|---------|
| [ADR-001](./ADR-001-ddd-over-crud.md) | Why DDD over CRUD | Accepted | 2025-01-09 | 採用 DDD 架構而非傳統 CRUD，以應對複雜業務邏輯與多個 Bounded Contexts |
| [ADR-002](./ADR-002-lightweight-aggregates.md) | Why Lightweight Aggregates Over Rich Object Graphs | Accepted | 2025-01-09 | 採用輕量級聚合避免載入無界集合，提升效能與可擴展性 |
| [ADR-003](./ADR-003-domain-accepts-dtos.md) | Domain Layer Accepting Application DTOs | Accepted | 2025-01-09 | 允許 Domain Layer 方法接受 Application DTOs，平衡依賴規則與實用性 |
| [ADR-004](./ADR-004-audit-log-consistency.md) | Audit Log Consistency Model | Accepted | 2025-01-09 | 採用同步稽核日誌（在同一事務中），保證 100% 資料一致性與法規遵循 |
| [ADR-005](./ADR-005-transaction-context-pattern.md) | Transaction Context Pattern Choice | Accepted | 2025-01-09 | 採用 Opaque TransactionContext 模式管理資料庫事務，保持 Clean Architecture 依賴方向 |

---

## 🎯 Core Architectural Principles

所有架構決策遵循以下核心原則：

### **1. Clean Architecture**
- **依賴規則**: 內層不依賴外層（Domain → Application → Infrastructure）
- **依賴反轉**: 使用接口抽象基礎設施細節
- **可測試性**: Domain Layer 可獨立測試，不依賴基礎設施

參考：[ADR-001](./ADR-001-ddd-over-crud.md), [ADR-005](./ADR-005-transaction-context-pattern.md)

### **2. Domain-Driven Design (DDD)**
- **Bounded Contexts**: 明確業務邊界，減少耦合
- **Ubiquitous Language**: 代碼即業務文件
- **Rich Domain Model**: 業務邏輯封裝在 Aggregates 與 Value Objects

參考：[ADR-001](./ADR-001-ddd-over-crud.md), [ADR-002](./ADR-002-lightweight-aggregates.md)

### **3. SOLID Principles**
- **SRP (單一職責原則)**: Repository 僅負責資料存取，不管理事務
- **DIP (依賴反轉原則)**: 使用接口隔離基礎設施依賴
- **ISP (接口隔離原則)**: 分離讀寫 Repository 接口

參考：[ADR-005](./ADR-005-transaction-context-pattern.md)

### **4. Pragmatism over Purity (實用主義)**
- **允許實用調整**: 在嚴格原則與開發效率間取得平衡
- **明確文檔例外**: 所有調整都有 ADR 記錄與 Code Review 檢查

參考：[ADR-003](./ADR-003-domain-accepts-dtos.md)

---

## 🔑 Key Design Patterns

### **Domain-Driven Design Patterns**
- **Aggregate Pattern**: 封裝業務不變性規則
- **Value Object Pattern**: 不可變的業務概念（如 `PhoneNumber`, `Money`）
- **Domain Event Pattern**: 鬆耦合的上下文協作
- **Repository Pattern**: 聚合持久化抽象

參考：`/docs/architecture/ddd/04-tactical-design.md`

### **Clean Architecture Patterns**
- **Dependency Inversion**: 使用接口抽象基礎設施
- **Hexagonal Architecture**: Ports & Adapters 隔離外部系統
- **Anti-Corruption Layer**: LINE SDK Adapter 防止外部模型污染

參考：`/docs/architecture/ddd/06-layered-architecture.md`

### **Transaction Patterns**
- **Unit of Work Pattern**: 協調事務與事件發布
- **Transaction Context Pattern**: 管理資料庫事務不污染 Domain Layer

參考：[ADR-005](./ADR-005-transaction-context-pattern.md)

### **Data Patterns**
- **DTO (Data Transfer Object)**: 跨層/跨上下文數據傳遞
- **Lightweight Aggregate**: 避免載入無界集合，按需查詢
- **CQRS (Command Query Responsibility Segregation)**: 分離讀寫操作

參考：[ADR-002](./ADR-002-lightweight-aggregates.md), [ADR-003](./ADR-003-domain-accepts-dtos.md)

---

## 📊 Decision Matrix

### **效能 vs 一致性**

| 決策 | 效能 | 一致性 | 選擇 | ADR |
|------|------|--------|------|-----|
| 稽核日誌模式 | 非同步 > 同步 | 同步 > 非同步 | 同步 | [ADR-004](./ADR-004-audit-log-consistency.md) |
| Aggregate 載入 | Rich Object Graph > Lightweight | Lightweight > Rich | Lightweight | [ADR-002](./ADR-002-lightweight-aggregates.md) |

### **純粹性 vs 實用性**

| 決策 | 純粹 Clean Architecture | 實用主義調整 | 選擇 | ADR |
|------|------------------------|-------------|------|-----|
| Domain Layer 依賴 | 不依賴任何外層 | 允許依賴 Application DTOs | 實用主義 | [ADR-003](./ADR-003-domain-accepts-dtos.md) |
| 事務管理 | Domain Layer 完全不知道事務 | TransactionContext 作為標記接口 | 平衡 | [ADR-005](./ADR-005-transaction-context-pattern.md) |

---

## 📝 How to Use ADRs

### **For Developers**

1. **實現新功能前**：
   - 檢查是否有相關 ADR
   - 遵循 ADR 中的設計模式與原則

2. **遇到架構問題時**：
   - 查閱相關 ADR 的 Rationale 與 Consequences
   - 如果 ADR 不適用，提出新的 ADR

3. **Code Review 時**：
   - 驗證代碼是否符合 ADR 決策
   - 檢查是否違反 ADR 中的禁止項

### **For Architects**

1. **重大決策後**：
   - 創建新的 ADR 記錄決策過程
   - 包含 Context, Decision, Rationale, Consequences

2. **架構演進時**：
   - 更新相關 ADR 的 Status（如 Superseded, Deprecated）
   - 記錄演進原因與新 ADR 連結

3. **定期回顧**：
   - 每季度回顧 ADR，驗證決策是否仍然適用
   - 根據實際運行情況調整 Consequences 中的緩解策略

---

## 📖 ADR Template

創建新 ADR 時，請使用以下模板：

```markdown
# ADR-XXX: [Title]

**Date**: YYYY-MM-DD
**Status**: [Proposed | Accepted | Deprecated | Superseded]
**Supersedes**: ADR-XXX (if applicable)

---

## Context

[描述決策的背景、問題、需求]

---

## Decision

[最終採用的方案，簡潔明確]

---

## Rationale

[決策的理由、分析、方案比較]

### 方案比較

| 方案 | 優勢 | 代價 |
|------|------|------|
| A | ... | ... |
| B | ... | ... |

---

## Consequences

### 優勢
1. ...
2. ...

### 代價
1. ...
2. ...

### 緩解策略
1. ...
2. ...

---

## References

- [相關文檔連結]
- [外部參考資料]

---

## Notes

- **YYYY-MM-DD**: [變更記錄]
```

---

## 🔗 Related Documentation

- **DDD Architecture**: `/docs/architecture/ddd/README.md`
  - 完整的 DDD 架構設計文檔（13 章）

- **Product Requirements**: `/docs/product/PRD.md`
  - 產品需求文件

- **Testing Standards**: `/docs/qa/testing-conventions.md`
  - 測試標準與慣例

- **Deployment Guide**: `/docs/operations/DEPLOYMENT.md`
  - 部署與維運指南

---

## 📋 Change Log

| Date | Change | Author |
|------|--------|--------|
| 2025-01-09 | 初始版本：創建 ADR-001 至 ADR-005 | Development Team |

---

## 📞 Contact

如有架構相關問題，請聯繫：
- **Technical Owner**: Backend Team
- **Documentation**: `/docs/README.md`
- **Issue Tracking**: [GitHub Issues](https://github.com/your-org/bar_crm/issues)
