# Domain-Driven Design 架構設計指南

> **版本**: 1.0
> **最後更新**: 2025-01-08
> **設計原則**: 基於 DDD 戰略設計,中粒度限界上下文,核心域為積分管理

---

## 關於本文檔

本目錄包含完整的 DDD 架構設計指南,原文檔 `DDD_GUIDE.md` 已被拆分為多個章節,以便於閱讀和維護。

---

## 目錄

### 戰略設計部分

1. **[戰略設計總覽](./01-strategic-overview.md)**
   - 設計概述: 設計目標與原則
   - 領域劃分策略: 核心域與支撐域定義
   - 戰略設計: 領域事件識別 (Event Storming)
   - 命令定義與業務不變性規則

2. **[限界上下文劃分](./02-bounded-contexts.md)**
   - 8 個限界上下文的詳細設計
   - 聚合、值對象、倉儲接口定義
   - 領域事件與領域錯誤
   - 包含稽核追蹤上下文（Audit Context）

3. **[上下文映射圖](./03-context-map.md)**
   - 上下文間的關係與集成方式
   - 事件驅動架構
   - 防腐層設計

### 戰術設計部分

4. **[戰術設計](./04-tactical-design.md)**
   - 核心域深度設計 (積分管理)
   - 積分計算完整流程
   - 積分重算機制
   - 未來擴展設計

5. **[關鍵業務流程](./05-key-business-flows.md)**
   - QR Code 掃描與積分獲得
   - iChef 批次匯入與交易驗證
   - 問卷填寫與獎勵發放
   - 積分規則變更與重算

6. **[分層架構設計](./06-layered-architecture.md)**
   - 六角架構 (Hexagonal Architecture)
   - 層次職責定義
   - Ports and Adapters 模式

7. **[聚合設計原則與反模式避免](./07-aggregate-design-principles.md)** ⭐ **必讀**
   - 核心設計原則：告別貧血領域模型
   - 職責邊界劃分：Domain Layer vs Application Layer
   - 常見反模式識別：貧血模型、God Use Case、Tell Don't Ask 違反
   - 正確設計模式：DTO 傳遞、領域服務、事件驅動
   - 設計檢查清單：聚合、Use Case、整體架構
   - 實戰案例分析：PointsAccount、ConversionRule、Invoice

### 擴展與實施

8. **[實施路線圖](./08-implementation-roadmap.md)**
   - 實施建議: 開發優先級與技術實施要點
   - 未來擴展考量: V3.2+ 積分兌換、優惠券、會員等級
   - 技術演進路線圖: 單體應用 → 模組化單體 → 微服務

### 規格文檔

9. **[Use Case 定義](./09-use-case-definitions.md)**
   - 應用層規格
   - 輸入輸出契約
   - 執行流程與事務邊界
   - 錯誤處理策略

10. **[Value Object 驗證規則](./10-value-object-validation.md)**
    - 所有值對象的構造方法
    - 驗證規則與正規化策略
    - 錯誤訊息定義

11. **[Dependency Rules](./11-dependency-rules.md)**
    - Clean Architecture 依賴規則
    - 各層允許的依賴關係
    - 接口所有權規則
    - 依賴注入配置

### 實作指南

13. **[錯誤處理架構](./13-error-handling-strategy.md)** ⭐ **關鍵章節**
    - 錯誤類型層次（Domain、Application、Infrastructure Errors）
    - 跨層級錯誤傳播規則
    - HTTP 狀態碼映射策略
    - 日誌記錄策略（各層職責）
    - 常見錯誤處理反模式

14. **[事件處理實作指南](./14-event-handling-implementation.md)** ⭐ **關鍵章節**
    - 事件驅動架構完整實作
    - 事件收集（Domain Layer）
    - 事件發布（Unit of Work vs Transactional Outbox）
    - Event Handlers 與冪等性設計
    - FX 模組配置
    - 重試策略與 Dead Letter Queue
    - 監控與告警

15. **[測試策略](./15-testing-strategy.md)** ⭐ **關鍵章節**
    - 測試金字塔與覆蓋率目標（77% Unit + 20% Integration + 3% E2E）
    - Domain Layer 測試（純單元測試，無 Mocks）
    - Application Layer 測試（Mock Repositories）
    - Infrastructure Layer 測試（SQLite in-memory）
    - End-to-End 測試（Black-box HTTP API）
    - Mock 策略與測試組織

### 附錄

12. **[參考資料](./12-references.md)**
    - 相關書籍與文章
    - DDD 經典文獻

---

## 快速導航

### 我想了解...

- **系統整體設計** → 從 [01-戰略設計總覽](./01-strategic-overview.md) 開始
- **領域事件與命令** → 查看 [01-戰略設計總覽](./01-strategic-overview.md) 第 2 節
- **具體上下文實現** → 參考 [02-限界上下文劃分](./02-bounded-contexts.md)
- **稽核追蹤與合規** → 查看 [02-限界上下文劃分](./02-bounded-contexts.md) 第 3.9 節（稽核上下文）
- **積分計算邏輯** → 閱讀 [04-戰術設計](./04-tactical-design.md)
- **業務流程** → 查看 [05-關鍵業務流程](./05-key-business-flows.md)
- **代碼分層** → 參考 [06-分層架構設計](./06-layered-architecture.md)
- **聚合設計原則** ⭐ → **必讀** [07-聚合設計原則與反模式避免](./07-aggregate-design-principles.md)
- **如何避免貧血領域模型** → 查看 [07-聚合設計原則](./07-aggregate-design-principles.md) 第 7.3 節
- **Domain Layer vs Application Layer 職責** → 查看 [07-聚合設計原則](./07-aggregate-design-principles.md) 第 7.2 節
- **Tell, Don't Ask 原則** → 查看 [07-聚合設計原則](./07-aggregate-design-principles.md) 第 7.3.4 節
- **未來擴展規劃** → 查看 [08-實施路線圖](./08-implementation-roadmap.md)
- **Use Case 實現** → 查看 [09-Use Case 定義](./09-use-case-definitions.md)
- **稽核日誌 Use Cases** → 查看 [09-Use Case 定義](./09-use-case-definitions.md) 第 10.4 節
- **事務一致性保證** → 查看 [03-上下文映射圖](./03-context-map.md) 第 4.3 節（稽核日誌記錄流程）
- **依賴規則** → 參考 [11-Dependency Rules](./11-dependency-rules.md)
- **錯誤處理** ⭐ → **必讀** [13-錯誤處理架構](./13-error-handling-strategy.md)
- **事件處理** ⭐ → **必讀** [14-事件處理實作指南](./14-event-handling-implementation.md)
- **測試策略** ⭐ → **必讀** [15-測試策略](./15-testing-strategy.md)

---

## 文檔維護

- 本文檔隨代碼演進持續更新
- 使用 ADR (Architecture Decision Records) 記錄重大決策
- 維護統一語言詞彙表

**維護者**: 開發團隊
**文檔版本**: 1.4
**最後更新**: 2025-01-09
**變更記錄**:

### **2025-01-09: Critical Gaps Fixed（關鍵缺口已修復）**
- ✅ **新增第 13 章：錯誤處理架構**（⭐ 關鍵章節）
  - 錯誤類型層次（Domain、Application、Infrastructure）
  - 跨層級錯誤傳播規則與 HTTP 狀態碼映射
  - 日誌記錄策略（各層職責明確定義）
  - 7 個錯誤處理反模式與正確做法
  - 📊 文檔長度：~1,100 行
  - 🎯 狀態：**Production Ready**

- ✅ **新增第 14 章：事件處理實作指南**（⭐ 關鍵章節）
  - 完整事件生命週期（收集 → 發布 → 處理）
  - Unit of Work vs Transactional Outbox 模式對比
  - Event Handlers 與冪等性設計
  - FX 模組配置完整範例
  - 重試策略、Dead Letter Queue、監控告警
  - 📊 文檔長度：~1,200 行
  - 🎯 狀態：**Production Ready**

- ✅ **新增第 15 章：測試策略**（⭐ 關鍵章節）
  - 測試金字塔（77% Unit + 20% Integration + 3% E2E）
  - 各層級測試完整範例（Domain、Application、Infrastructure、E2E）
  - Mock 策略（Testify Mock + Fake 實現）
  - 測試組織與命名慣例
  - CI/CD 整合與自動化
  - 📊 文檔長度：~1,400 行
  - 🎯 狀態：**Production Ready**

- 📈 **Uncle Bob Code Mentor Review Result**:
  - **評分提升**: 7.5/10 → **8.5/10** (修復關鍵缺口後)
  - **結論**: "Ready to Implement with Critical Fixes" → **"Ready to Implement"**
  - **關鍵改善**: 錯誤處理、事件生命週期、測試策略 100% 完整

### **2025-01-08: 初始版本**
- 新增稽核追蹤上下文（Audit Context），支援完整稽核日誌與 GDPR 合規
- **新增第 7 章：聚合設計原則與反模式避免**（⭐ 必讀） - **已完成並通過 Uncle Bob 審查（9.0/10）**
  - ✅ SOLID 原則系統性教學（7.1.3）
  - ✅ Transaction Script vs Domain Model 決策（7.1.4）
  - ✅ 聚合大小原則（7.2.4）
  - ✅ 7 個常見反模式識別（貧血模型、God Use Case、並發控制、原始類型執著、外部服務污染等）
  - ✅ 4 個正確設計模式
  - ✅ 3 個完整案例研究（PointsAccount、ConversionRule、Invoice）
  - ✅ 實用設計檢查清單與決策樹
  - 📊 文檔長度：~2,400 行，包含 ~700 行新增內容
  - 🎯 狀態：**Ready to Ship**（可作為團隊正式設計指南）
