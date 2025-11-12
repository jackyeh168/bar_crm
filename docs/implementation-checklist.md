# 實作檢查清單（Implementation Checklist）

## 使用說明

**目的**：確保每次實作都遵循架構約束，避免常見錯誤。

**使用時機**：在開始實作任何新的 Domain、Application、Infrastructure 代碼之前。

**如何使用**：
1. 根據實作內容選擇對應的檢查清單（Domain / Application / Infrastructure）
2. 逐項確認每個檢查項
3. 如果有任何項目不確定，先閱讀對應的架構文檔
4. **全部確認後才開始寫代碼**

---

## 📋 Domain Layer 實作檢查清單

### 開始實作前（Pre-Implementation）

在開始實作任何 Domain Layer 代碼前，必須確認：

- [ ] **已閱讀相關架構文檔**
  - [ ] 錯誤處理架構：`docs/architecture/ddd/13-error-handling-strategy.md`
  - [ ] 依賴規則：`docs/architecture/ddd/12-dependency-rules.md`
  - [ ] 實作層指南：根據實作內容選讀
    - 值對象：`docs/architecture/ddd/10-value-object-validation.md`
    - 聚合根：`docs/architecture/ddd/08-aggregate-design-patterns.md`
    - 領域服務：`docs/architecture/ddd/09-domain-services.md`

- [ ] **了解架構約束**
  - [ ] 確認錯誤定義使用 `DomainError` 結構（不使用 `fmt.Errorf`）
  - [ ] 確認不依賴外部框架（GORM, Gin, Redis, LINE SDK）
  - [ ] 確認值對象設計為不可變（unexported fields，無 setters）

- [ ] **了解測試要求**
  - [ ] 測試必須遵循 AAA 模式（Arrange-Act-Assert）
  - [ ] 目標覆蓋率：Unit Tests >= 85%
  - [ ] 測試命名規範：`Test{Type}_{Method}_{Scenario}`

### 實作中（During Implementation）

#### 錯誤處理

- [ ] **所有錯誤都使用 DomainError 結構**
  ```go
  // ✅ 正確
  var ErrNegativePointsAmount = &DomainError{
      Code:    ErrCodeNegativePointsAmount,
      Message: "積分數量不能為負數",
  }

  // ❌ 錯誤
  var ErrNegativePointsAmount = fmt.Errorf("points amount cannot be negative")
  ```

- [ ] **錯誤類型語義正確**
  - [ ] 建構約束違反：使用建構約束相關錯誤（如 `ErrNegativePointsAmount`）
  - [ ] 業務規則違反：使用業務規則相關錯誤（如 `ErrInsufficientPoints`，但應在聚合根層）

#### 值對象設計

- [ ] **不可變性**
  - [ ] 所有字段都是 unexported (`value int` 而非 `Value int`)
  - [ ] 沒有 Setter 方法
  - [ ] 修改操作返回新實例（如 `Add`, `Subtract`）

- [ ] **自我驗證**
  - [ ] Checked 建構函數驗證輸入 (`NewPointsAmount(int) (PointsAmount, error)`)
  - [ ] Unchecked 建構函數僅供內部使用 (`newPointsAmountUnchecked(int) PointsAmount`)

- [ ] **職責清晰**
  - [ ] 值對象只處理建構約束，不包含業務規則
  - [ ] 業務規則放在聚合根或領域服務

#### 依賴規則

- [ ] **無外部依賴**
  ```go
  // ❌ 禁止
  import "gorm.io/gorm"
  import "github.com/gin-gonic/gin"
  import "github.com/line/line-bot-sdk-go"

  // ✅ 允許
  import "fmt"
  import "time"
  import "github.com/shopspring/decimal"  // 純計算庫
  import "github.com/google/uuid"          // 純數據庫
  import "github.com/yourorg/bar_crm/internal/domain/shared"
  ```

- [ ] **只依賴同層或內層**
  - [ ] 可依賴 `internal/domain/shared`
  - [ ] 可依賴同一 bounded context 內的其他包
  - [ ] 不依賴 Application, Infrastructure, Presentation

### 實作後（Post-Implementation）

- [ ] **測試完整性**
  - [ ] 所有公開方法都有測試
  - [ ] 覆蓋率 >= 85%
  - [ ] 測試遵循 AAA 模式
  - [ ] 測試名稱清晰（`Test{Type}_{Method}_{Scenario}`）

- [ ] **代碼審查自檢**
  - [ ] 所有字段都是 unexported
  - [ ] 所有錯誤都使用 DomainError
  - [ ] 沒有 import 外部框架
  - [ ] 注釋解釋 WHY 而非 WHAT
  - [ ] 沒有噪音注釋（如 `// Value 獲取積分數量`）

---

## 📋 Application Layer 實作檢查清單

### 開始實作前

- [ ] **已閱讀相關架構文檔**
  - [ ] Use Case 定義：`docs/architecture/ddd/10-use-case-definitions.md`
  - [ ] 事件處理：`docs/architecture/ddd/02-strategic-design.md`（領域事件部分）
  - [ ] DTO 設計：`docs/architecture/implementation/03-application-layer-implementation.md`

- [ ] **了解職責邊界**
  - [ ] Application Layer 只做協調，不包含業務邏輯
  - [ ] 業務邏輯在 Domain Layer（聚合根、領域服務）
  - [ ] Application Layer 管理事務邊界

### 實作中

#### Use Case 設計

- [ ] **職責清晰**
  - [ ] Use Case 只做協調（讀取聚合、調用方法、保存聚合）
  - [ ] 業務邏輯在聚合根方法中（如 `account.DeductPoints()`）
  - [ ] 不在 Use Case 中寫業務邏輯（如 `if balance < amount`）

- [ ] **事務管理**
  - [ ] 使用 `TransactionManager.InTransaction()` 包裹操作
  - [ ] 事務邊界 = 一個 Use Case 執行
  - [ ] 一個事務只修改一個聚合

#### DTO 設計

- [ ] **DTO 集中管理**
  - [ ] 所有 DTO 在 `application/dto/` 目錄
  - [ ] DTO 是純數據結構（無行為）

- [ ] **轉換職責**
  - [ ] Application Layer 負責 Entity ↔ DTO 轉換
  - [ ] Domain Layer 不知道 DTO 的存在

#### 事件處理

- [ ] **事件訂閱**
  - [ ] Event Handlers 在 `application/events/` 目錄
  - [ ] Handler 實現 `EventHandler` 接口
  - [ ] 透過 DI 註冊到 Event Bus

### 實作後

- [ ] **測試完整性**
  - [ ] Use Case 有單元測試（mock repositories）
  - [ ] Event Handlers 有單元測試
  - [ ] 覆蓋率 >= 70%

---

## 📋 Infrastructure Layer 實作檢查清單

### 開始實作前

- [ ] **已閱讀相關架構文檔**
  - [ ] Repository Pattern：`docs/architecture/ddd/06-repository-pattern.md`
  - [ ] 依賴規則：`docs/architecture/ddd/12-dependency-rules.md`
  - [ ] Anti-Corruption Layer：`CLAUDE.md`（ACL 章節）

- [ ] **了解職責**
  - [ ] Infrastructure 實現 Domain 定義的接口
  - [ ] 不暴露技術細節到 Domain
  - [ ] 使用 ACL 隔離外部服務

### 實作中

#### Repository 實現

- [ ] **實現 Domain 接口**
  - [ ] Repository 接口定義在 `domain/{context}/repository/`
  - [ ] Repository 實現在 `infrastructure/persistence/{context}/`
  - [ ] 只依賴接口定義，不修改接口

- [ ] **模型轉換**
  - [ ] GORM Model 在 Infrastructure Layer（如 `PointsAccountModel`）
  - [ ] Domain Entity 在 Domain Layer（如 `PointsAccount`）
  - [ ] Repository 負責 Model ↔ Entity 轉換

- [ ] **錯誤映射**
  - [ ] GORM 錯誤轉換為 Domain 錯誤
  - [ ] 不暴露 `gorm.ErrRecordNotFound` 到外層

#### Anti-Corruption Layer

- [ ] **外部服務隔離**
  - [ ] LINE SDK, iChef 等外部服務有 Adapter
  - [ ] Adapter 將外部模型轉換為 Domain 模型
  - [ ] Domain Layer 不知道外部服務的存在

### 實作後

- [ ] **測試**
  - [ ] Repository 有 Integration Tests（實際數據庫）
  - [ ] 外部服務有 Contract Tests（模擬外部 API）

---

## 📋 快速參考：常見錯誤與修正

| 錯誤行為 | 正確做法 | 檢查清單位置 |
|---------|---------|------------|
| 使用 `fmt.Errorf` | 使用 `DomainError` 結構 | Domain > 錯誤處理 |
| 值對象有 Setter | 返回新實例（不可變） | Domain > 值對象設計 |
| Domain import `gorm` | 只 import 標準庫和 domain 包 | Domain > 依賴規則 |
| Use Case 包含業務邏輯 | 業務邏輯在聚合根 | Application > Use Case 設計 |
| Repository 返回 GORM 錯誤 | 轉換為 Domain 錯誤 | Infrastructure > 錯誤映射 |

---

## 🔍 自檢問題清單

實作完成後，問自己以下問題：

### Domain Layer
1. ✅ 所有錯誤都使用 `DomainError` 了嗎？
2. ✅ 值對象是不可變的嗎（無 setters，unexported fields）？
3. ✅ 有沒有 import 外部框架？
4. ✅ 業務邏輯在聚合根/領域服務中，而非值對象中？
5. ✅ 測試覆蓋率 >= 85% 了嗎？

### Application Layer
6. ✅ Use Case 只做協調，沒有業務邏輯？
7. ✅ 事務邊界正確嗎（一個 Use Case = 一個事務 = 一個聚合）？
8. ✅ DTO 轉換在 Application Layer，Domain 不知道 DTO？

### Infrastructure Layer
9. ✅ Repository 實現了 Domain 接口？
10. ✅ GORM Model ↔ Domain Entity 轉換正確？
11. ✅ 外部服務有 ACL 隔離？

---

## 📚 延伸閱讀

- **完整架構指南**: `docs/architecture/ddd/README.md`
- **錯誤處理策略**: `docs/architecture/ddd/13-error-handling-strategy.md`
- **測試規範**: `docs/qa/testing-conventions.md`
- **部署指南**: `docs/operations/DEPLOYMENT.md`

---

**最後提醒**：如果對任何檢查項有疑問，**請先閱讀對應的架構文檔，而不是猜測**。架構文檔是單一真相來源（Single Source of Truth）。
