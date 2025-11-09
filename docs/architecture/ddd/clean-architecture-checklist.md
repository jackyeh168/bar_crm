# Clean Architecture Code Review Checklist

> **版本**: 1.0
> **最後更新**: 2025-01-09
> **用途**: 代碼審查時確保符合 Clean Architecture 和 SOLID 原則

本清單用於代碼審查（Code Review）時檢查實現是否符合 Clean Architecture 原則。

---

## **使用方式**

### **審查流程**

```
1. 開啟 Pull Request
2. 識別變更涉及的層級（Domain/Application/Infrastructure/Presentation）
3. 根據對應章節檢查清單逐項審查
4. 發現違規 → 標記為 "Request Changes"
5. 全部通過 → 標記為 "Approved"
```

### **嚴重性標記**

- 🔴 **MUST FIX**（必須修復）：違反核心原則，阻塞合併
- 🟡 **SHOULD FIX**（應該修復）：不符合最佳實踐，建議修改
- 🟢 **NICE TO HAVE**（可選優化）：改進建議，不阻塞合併

---

## **目錄**

1. [依賴規則檢查](#1-依賴規則檢查)
2. [Domain Layer 檢查](#2-domain-layer-檢查)
3. [Application Layer 檢查](#3-application-layer-檢查)
4. [Infrastructure Layer 檢查](#4-infrastructure-layer-檢查)
5. [Presentation Layer 檢查](#5-presentation-layer-檢查)
6. [事務管理檢查](#6-事務管理檢查)
7. [事件處理檢查](#7-事件處理檢查)
8. [併發控制檢查](#8-併發控制檢查)
9. [測試檢查](#9-測試檢查)
10. [通用代碼質量檢查](#10-通用代碼質量檢查)

---

## **1. 依賴規則檢查**

> **參考**: `/docs/architecture/ddd/11-dependency-rules.md`

### **1.1 依賴方向 🔴 MUST FIX**

- [ ] **Domain Layer 不依賴外層**
  ```go
  // ❌ 錯誤
  import "myapp/internal/application/usecases"
  import "myapp/internal/infrastructure/gorm"
  import "github.com/gin-gonic/gin"

  // ✅ 正確
  import "myapp/internal/domain/member"  // 同層依賴
  import "time"                          // 標準庫
  ```

- [ ] **Application Layer 不依賴 Infrastructure/Presentation**
  ```go
  // ❌ 錯誤
  import "myapp/internal/infrastructure/persistence"
  import "myapp/internal/presentation/http"

  // ✅ 正確
  import "myapp/internal/domain/points"
  import "myapp/internal/domain/points/repository"
  ```

- [ ] **Infrastructure Layer 不依賴 Presentation**
  ```go
  // ❌ 錯誤
  import "myapp/internal/presentation/http"

  // ✅ 正確
  import "myapp/internal/domain/points/repository"
  import "gorm.io/gorm"
  ```

### **1.2 接口所有權 🔴 MUST FIX**

- [ ] **Repository 接口定義在 Domain Layer**
  ```go
  // ✅ 正確位置
  // internal/domain/points/repository/points_account_repository.go
  package repository

  type PointsAccountRepository interface {
      FindByMemberID(ctx shared.TransactionContext, id MemberID) (*PointsAccount, error)
      Update(ctx shared.TransactionContext, account *PointsAccount) error
  }
  ```

- [ ] **接口方法參數使用 Domain 類型**
  ```go
  // ❌ 錯誤：使用 Infrastructure 類型
  type Repository interface {
      Update(tx *gorm.DB, account *PointsAccount) error
  }

  // ✅ 正確：使用 Domain 類型
  type Repository interface {
      Update(ctx shared.TransactionContext, account *PointsAccount) error
  }
  ```

### **1.3 TransactionContext 位置 🔴 MUST FIX**

- [ ] **TransactionContext 定義在 Domain Layer**
  ```go
  // ✅ 正確位置
  // internal/domain/shared/transaction.go
  package shared

  type TransactionContext interface {
      // 標記接口
  }
  ```

- [ ] **TransactionManager 定義在 Domain Layer**
  ```go
  // ✅ 正確位置
  // internal/domain/shared/transaction.go
  type TransactionManager interface {
      InTransaction(fn func(ctx TransactionContext) error) error
  }
  ```

---

## **2. Domain Layer 檢查**

> **參考**: `/docs/architecture/ddd/04-tactical-design.md`, `/docs/architecture/ddd/07-aggregate-design-principles.md`

### **2.1 Aggregate 設計 🔴 MUST FIX**

- [ ] **Aggregate 保護不變量**
  ```go
  // ✅ 正確：內部驗證
  func (a *PointsAccount) DeductPoints(amount PointsAmount, reason string) error {
      if a.usedPoints.Add(amount).GreaterThan(a.earnedPoints) {
          return ErrInsufficientPoints
      }
      a.usedPoints = a.usedPoints.Add(amount)
      return nil
  }

  // ❌ 錯誤：不驗證不變量
  func (a *PointsAccount) SetUsedPoints(amount PointsAmount) {
      a.usedPoints = amount  // 可能違反 usedPoints <= earnedPoints
  }
  ```

- [ ] **Aggregate 輕量級設計**
  ```go
  // ✅ 正確：不包含無界集合
  type PointsAccount struct {
      accountID    AccountID
      earnedPoints PointsAmount
      usedPoints   PointsAmount
      // ✅ 不包含 []PointsTransaction（避免無界增長）
  }

  // ❌ 錯誤：包含無界集合
  type PointsAccount struct {
      transactions []PointsTransaction  // 可能增長到 10,000+
  }
  ```

- [ ] **Aggregate 不直接引用其他 Aggregate**
  ```go
  // ❌ 錯誤：跨 Aggregate 引用
  type PointsAccount struct {
      member *Member  // 跨 Aggregate 引用
  }

  // ✅ 正確：使用 ID 引用
  type PointsAccount struct {
      memberID MemberID  // ID 引用
  }
  ```

### **2.2 Value Object 設計 🔴 MUST FIX**

- [ ] **Value Object 不可變**
  ```go
  // ✅ 正確：返回新實例
  func (m Money) Add(other Money) Money {
      return Money{amount: m.amount.Add(other.amount)}
  }

  // ❌ 錯誤：修改自身
  func (m *Money) Add(other Money) {
      m.amount = m.amount.Add(other.amount)
  }
  ```

- [ ] **Value Object 在構造時驗證**
  ```go
  // ✅ 正確：構造時驗證
  func NewPhoneNumber(value string) (PhoneNumber, error) {
      if !isValidPhoneNumber(value) {
          return PhoneNumber{}, ErrInvalidPhoneNumber
      }
      return PhoneNumber{value: value}, nil
  }

  // ❌ 錯誤：無驗證
  func NewPhoneNumber(value string) PhoneNumber {
      return PhoneNumber{value: value}
  }
  ```

### **2.3 Domain Service 職責 🔴 MUST FIX**

- [ ] **業務邏輯在 Domain Service，不在 Use Case**
  ```go
  // ✅ 正確：Domain Service 包含業務邏輯
  // internal/domain/points/service.go
  func (s *PointsCalculationService) CalculateTotalPoints(
      transactions []dto.VerifiedTransactionDTO,
  ) int {
      totalPoints := 0
      for _, tx := range transactions {
          totalPoints += s.CalculateForTransaction(tx)
      }
      return totalPoints
  }

  // ❌ 錯誤：業務邏輯在 Use Case
  // internal/application/points/use_case.go
  func (uc *UseCase) Execute(cmd Command) error {
      totalPoints := 0
      for _, tx := range txs {
          totalPoints += calculatePoints(tx)  // 業務邏輯洩漏到 Use Case
      }
  }
  ```

- [ ] **Domain Service 無狀態**
  ```go
  // ✅ 正確：無狀態
  type PointsCalculationService struct {
      // 無實例變量（或僅配置/策略）
  }

  // ❌ 錯誤：有狀態
  type PointsCalculationService struct {
      cachedRules map[string]Rule  // 狀態應由外部管理
  }
  ```

### **2.4 Domain Events 🟡 SHOULD FIX**

- [ ] **Aggregate 收集事件，不發布**
  ```go
  // ✅ 正確：收集事件
  func (a *PointsAccount) EarnPoints(...) error {
      a.earnedPoints = a.earnedPoints.Add(amount)
      a.RecordEvent(PointsEarned{...})  // 收集
      return nil
  }

  // ❌ 錯誤：直接發布
  func (a *PointsAccount) EarnPoints(...) error {
      a.earnedPoints = a.earnedPoints.Add(amount)
      eventBus.Publish(PointsEarned{...})  // ❌ 不應依賴 EventBus
      return nil
  }
  ```

- [ ] **事件命名使用過去式**
  ```go
  // ✅ 正確
  type PointsEarned struct { ... }
  type MemberRegistered struct { ... }

  // ❌ 錯誤
  type EarnPoints struct { ... }
  type RegisterMember struct { ... }
  ```

---

## **3. Application Layer 檢查**

> **參考**: `/docs/architecture/ddd/09-use-case-definitions.md`

### **3.1 Use Case 職責 🔴 MUST FIX**

- [ ] **Use Case 只做編排，不實現業務邏輯**
  ```go
  // ✅ 正確：純編排
  func (uc *EarnPointsUseCase) Execute(cmd Command) error {
      return uc.txManager.InTransaction(func(ctx Context) error {
          account := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
          points := uc.calculator.Calculate(cmd.Amount)  // 調用 Domain Service
          account.EarnPoints(points, ...)                // 調用 Aggregate
          uc.accountRepo.Update(ctx, account)
          return nil
      })
  }

  // ❌ 錯誤：包含業務邏輯
  func (uc *EarnPointsUseCase) Execute(cmd Command) error {
      points := cmd.Amount / 100  // ❌ 業務規則不應在 Use Case
      if points < 0 {
          return errors.New("invalid")  // ❌ 驗證應在 Domain Layer
      }
  }
  ```

- [ ] **Use Case 管理事務邊界**
  ```go
  // ✅ 正確：Use Case 管理事務
  func (uc *UseCase) Execute(cmd Command) error {
      return uc.txManager.InTransaction(func(ctx Context) error {
          // 業務邏輯...
      })
  }

  // ❌ 錯誤：Repository 管理事務
  func (uc *UseCase) Execute(cmd Command) error {
      uc.repo.BeginTransaction()
      // ...
      uc.repo.Commit()
  }
  ```

### **3.2 DTO 使用 🟡 SHOULD FIX**

- [ ] **DTO 定義在 Application Layer**
  ```go
  // ✅ 正確位置
  // internal/application/dto/transaction_dto.go
  package dto

  type VerifiedTransactionDTO struct {
      TransactionID   string
      Amount          decimal.Decimal
      InvoiceDate     time.Time
      SurveySubmitted bool
  }
  ```

- [ ] **DTO 純數據，無業務邏輯**
  ```go
  // ✅ 正確：純數據
  type VerifiedTransactionDTO struct {
      TransactionID string
      Amount        decimal.Decimal
  }

  // ❌ 錯誤：包含業務邏輯
  type VerifiedTransactionDTO struct {
      TransactionID string
      Amount        decimal.Decimal
  }

  func (d *VerifiedTransactionDTO) CalculatePoints() int {
      return int(d.Amount.Div(decimal.NewFromInt(100)).IntPart())
  }
  ```

### **3.3 事件發布 🔴 MUST FIX**

- [ ] **Use Case 在事務提交後發布事件**
  ```go
  // ✅ 正確：事務提交後發布
  func (uc *UseCase) Execute(cmd Command) error {
      return uc.txManager.InTransaction(func(ctx Context) error {
          account := uc.repo.FindByMemberID(ctx, cmd.MemberID)
          account.EarnPoints(...)
          uc.repo.Update(ctx, account)

          // 註冊事件到 Context（事務提交後才發布）
          for _, event := range account.GetEvents() {
              ctx.AddEvent(event)
          }
          account.ClearEvents()

          return nil
      })
  }
  ```

---

## **4. Infrastructure Layer 檢查**

> **參考**: `/docs/architecture/ddd/11-dependency-rules.md`

### **4.1 Repository 實現 🔴 MUST FIX**

- [ ] **Repository 實現 Domain 接口**
  ```go
  // ✅ 正確
  // internal/infrastructure/persistence/points_account_repository.go
  package persistence

  import (
      "myapp/internal/domain/points"
      "myapp/internal/domain/points/repository"
  )

  type GormPointsAccountRepository struct {
      db *gorm.DB
  }

  // 實現 Domain 接口
  func (r *GormPointsAccountRepository) FindByMemberID(
      ctx shared.TransactionContext,
      memberID points.MemberID,
  ) (*points.PointsAccount, error) {
      // ...
  }
  ```

- [ ] **不將 GORM Model 暴露給 Domain Layer**
  ```go
  // ✅ 正確：轉換為 Domain 實體
  func (r *GormRepository) FindByID(id string) (*points.PointsAccount, error) {
      var model PointsAccountModel
      r.db.First(&model, id)
      return model.ToDomainEntity(), nil  // 轉換
  }

  // ❌ 錯誤：直接返回 GORM Model
  func (r *GormRepository) FindByID(id string) (*PointsAccountModel, error) {
      var model PointsAccountModel
      r.db.First(&model, id)
      return &model, nil  // ❌ 洩漏 Infrastructure 類型
  }
  ```

### **4.2 Error 轉換 🟡 SHOULD FIX**

- [ ] **將 Infrastructure 錯誤轉換為 Domain 錯誤**
  ```go
  // ✅ 正確：錯誤轉換
  func (r *GormRepository) FindByID(id string) (*PointsAccount, error) {
      var model PointsAccountModel
      err := r.db.First(&model, id).Error

      if errors.Is(err, gorm.ErrRecordNotFound) {
          return nil, repository.ErrAccountNotFound  // Domain 錯誤
      }
      if err != nil {
          return nil, err
      }

      return model.ToDomainEntity(), nil
  }

  // ❌ 錯誤：直接返回 GORM 錯誤
  func (r *GormRepository) FindByID(id string) (*PointsAccount, error) {
      var model PointsAccountModel
      err := r.db.First(&model, id).Error
      return model.ToDomainEntity(), err  // ❌ 洩漏 gorm.ErrRecordNotFound
  }
  ```

---

## **5. Presentation Layer 檢查**

> **參考**: `/docs/architecture/ddd/11-dependency-rules.md`

### **5.1 Handler 職責 🟡 SHOULD FIX**

- [ ] **Handler 只做輸入驗證和錯誤映射**
  ```go
  // ✅ 正確
  func (h *PointsHandler) EarnPoints(c *gin.Context) {
      var req EarnPointsRequest
      if err := c.ShouldBindJSON(&req); err != nil {
          c.JSON(400, gin.H{"error": "invalid request"})
          return
      }

      cmd := toCommand(req)  // DTO 轉換
      err := h.useCase.Execute(cmd)

      if err != nil {
          h.mapErrorToHTTP(c, err)  // 錯誤映射
          return
      }

      c.JSON(200, gin.H{"success": true})
  }

  // ❌ 錯誤：包含業務邏輯
  func (h *PointsHandler) EarnPoints(c *gin.Context) {
      points := calculatePoints(amount)  // ❌ 業務邏輯
      if points < 0 {
          c.JSON(400, ...)  // ❌ 業務驗證
      }
  }
  ```

- [ ] **HTTP 錯誤映射語義化**
  ```go
  // ✅ 正確：語義化映射
  func (h *Handler) mapErrorToHTTP(c *gin.Context, err error) {
      switch {
      case errors.Is(err, repository.ErrConcurrentModification):
          c.JSON(409, gin.H{"error": "CONCURRENT_MODIFICATION"})
      case errors.Is(err, points.ErrInsufficientPoints):
          c.JSON(400, gin.H{"error": "INSUFFICIENT_POINTS"})
      default:
          c.JSON(500, gin.H{"error": "INTERNAL_ERROR"})
      }
  }
  ```

---

## **6. 事務管理檢查**

> **參考**: `/docs/architecture/decisions/ADR-005-transaction-context-pattern.md`

### **6.1 TransactionContext 使用 🔴 MUST FIX**

- [ ] **Repository 方法接受 TransactionContext**
  ```go
  // ✅ 正確
  func (r *Repository) Update(
      ctx shared.TransactionContext,
      account *PointsAccount,
  ) error

  // ❌ 錯誤：接受 *gorm.DB
  func (r *Repository) Update(
      tx *gorm.DB,
      account *PointsAccount,
  ) error
  ```

- [ ] **Infrastructure Layer 從 Context 提取事務**
  ```go
  // ✅ 正確
  func (r *GormRepository) Update(
      ctx shared.TransactionContext,
      account *PointsAccount,
  ) error {
      db := r.extractDB(ctx)  // 提取事務
      return db.Save(toModel(account)).Error
  }

  func (r *GormRepository) extractDB(ctx shared.TransactionContext) *gorm.DB {
      if txCtx, ok := ctx.(*transaction.gormTransactionContext); ok {
          return txCtx.tx
      }
      return r.db
  }
  ```

---

## **7. 事件處理檢查**

> **參考**: `/docs/architecture/ddd/14-event-handling-implementation.md`

### **7.1 冪等性檢查 🔴 MUST FIX**

- [ ] **Event Handler 實現冪等性**
  ```go
  // ✅ 正確：冪等性檢查
  func (h *Handler) Handle(ctx context.Context, event DomainEvent) error {
      cacheKey := fmt.Sprintf("event:processed:%s", event.EventID())

      // 第一道防線：Cache
      if h.cache.Exists(cacheKey) {
          return nil
      }

      // 第二道防線：Database
      if isProcessed, _ := h.eventLogRepo.IsProcessed(event.EventID()); isProcessed {
          h.cache.Set(cacheKey, true, 24*time.Hour)
          return nil
      }

      // 執行業務邏輯...

      // 標記為已處理
      h.eventLogRepo.MarkAsProcessed(event.EventID(), event.EventType(), "HandlerName")
      h.cache.Set(cacheKey, true, 24*time.Hour)

      return nil
  }

  // ❌ 錯誤：無冪等性保護
  func (h *Handler) Handle(ctx context.Context, event DomainEvent) error {
      // 直接執行業務邏輯（可能重複執行）
      h.notificationService.Send(...)
  }
  ```

### **7.2 事件內容 🟡 SHOULD FIX**

- [ ] **事件僅包含必要數據，不包含整個 Aggregate**
  ```go
  // ✅ 正確：僅必要數據
  type PointsEarned struct {
      AccountID AccountID
      MemberID  MemberID
      Amount    PointsAmount
  }

  // ❌ 錯誤：包含整個 Aggregate
  type PointsEarned struct {
      Account *PointsAccount  // ❌ 過度耦合
  }
  ```

---

## **8. 併發控制檢查**

> **參考**: `/docs/architecture/ddd/16-concurrency-control.md`

### **8.1 樂觀鎖實現 🔴 MUST FIX**

- [ ] **Aggregate 包含 version 欄位**
  ```go
  // ✅ 正確
  type PointsAccount struct {
      accountID    AccountID
      earnedPoints PointsAmount
      version      int  // 樂觀鎖版本號
  }
  ```

- [ ] **Repository Update 檢查版本號**
  ```go
  // ✅ 正確：樂觀鎖檢查
  func (r *GormRepository) Update(ctx Context, account *PointsAccount) error {
      result := r.db.Model(&PointsAccountModel{}).
          Where("account_id = ? AND version = ?", account.ID(), account.Version()).
          Updates(map[string]interface{}{
              "earned_points": account.EarnedPoints(),
              "version":       account.Version() + 1,
          })

      if result.RowsAffected == 0 {
          return repository.ErrConcurrentModification
      }

      account.IncrementVersion()
      return nil
  }

  // ❌ 錯誤：無版本檢查
  func (r *GormRepository) Update(ctx Context, account *PointsAccount) error {
      return r.db.Save(toModel(account)).Error  // ❌ 無併發保護
  }
  ```

### **8.2 重試策略 🟡 SHOULD FIX**

- [ ] **Use Case 實現重試機制**
  ```go
  // ✅ 正確：重試機制
  func (uc *UseCase) Execute(cmd Command) error {
      return uc.retryWithExponentialBackoff(3, func() error {
          return uc.txManager.InTransaction(func(ctx Context) error {
              // 業務邏輯...
          })
      })
  }

  // ❌ 錯誤：無重試
  func (uc *UseCase) Execute(cmd Command) error {
      return uc.txManager.InTransaction(func(ctx Context) error {
          // 業務邏輯...（併發衝突直接失敗）
      })
  }
  ```

---

## **9. 測試檢查**

> **參考**: `/docs/architecture/ddd/15-testing-strategy.md`

### **9.1 測試覆蓋率 🟡 SHOULD FIX**

- [ ] **Domain Layer 單元測試覆蓋率 >= 70%**
- [ ] **關鍵業務邏輯覆蓋率 >= 90%**
- [ ] **每個 Use Case 至少有 1 個整合測試**

### **9.2 測試命名 🟢 NICE TO HAVE**

- [ ] **使用 AAA 模式（Arrange-Act-Assert）**
  ```go
  // ✅ 正確
  func TestPointsAccount_DeductPoints_InsufficientBalance(t *testing.T) {
      // Arrange
      account := points.NewPointsAccount(accountID, memberID, 100)

      // Act
      err := account.DeductPoints(points.PointsAmount(150), "Redemption")

      // Assert
      assert.Error(t, err)
      assert.ErrorIs(t, err, points.ErrInsufficientPoints)
  }
  ```

- [ ] **測試名稱格式: Test{StructName}_{MethodName}_{Scenario}**
  ```go
  // ✅ 正確
  func TestPointsAccount_EarnPoints_ValidInput(t *testing.T)
  func TestPointsAccount_DeductPoints_InsufficientBalance(t *testing.T)

  // ❌ 錯誤
  func TestEarnPoints(t *testing.T)
  func Test_DeductPoints_1(t *testing.T)
  ```

---

## **10. 通用代碼質量檢查**

### **10.1 錯誤處理 🔴 MUST FIX**

- [ ] **使用語義化錯誤，不用魔術字串**
  ```go
  // ✅ 正確
  var (
      ErrInsufficientPoints     = errors.New("insufficient points")
      ErrConcurrentModification = errors.New("concurrent modification detected")
  )

  // ❌ 錯誤
  return errors.New("points not enough")  // 魔術字串
  ```

- [ ] **使用 errors.Is/errors.As 判斷錯誤**
  ```go
  // ✅ 正確
  if errors.Is(err, repository.ErrAccountNotFound) {
      // ...
  }

  // ❌ 錯誤
  if err.Error() == "account not found" {  // 字串比對
      // ...
  }
  ```

### **10.2 日誌記錄 🟡 SHOULD FIX**

- [ ] **使用結構化日誌（zap）**
  ```go
  // ✅ 正確
  logger.Info("Points earned",
      zap.String("memberID", memberID),
      zap.Int("amount", amount),
  )

  // ❌ 錯誤
  log.Println("Points earned: " + memberID + ", amount: " + strconv.Itoa(amount))
  ```

- [ ] **不記錄敏感資料**
  ```go
  // ✅ 正確：遮罩敏感資料
  logger.Info("Member registered",
      zap.String("phoneNumber", maskPhoneNumber(phone)),
  )

  // ❌ 錯誤：記錄完整手機號碼
  logger.Info("Member registered",
      zap.String("phoneNumber", phone),  // ❌ 隱私洩漏
  )
  ```

### **10.3 命名規範 🟢 NICE TO HAVE**

- [ ] **使用領域語言命名（Ubiquitous Language）**
  ```go
  // ✅ 正確：領域語言
  type PointsAccount struct { ... }
  func (a *PointsAccount) EarnPoints(...) error

  // ❌ 錯誤：技術語言
  type PointsData struct { ... }
  func (d *PointsData) AddPoints(...) error
  ```

- [ ] **避免縮寫，除非是業界通用**
  ```go
  // ✅ 正確
  memberID   // ID 是通用縮寫
  earnedPoints

  // ❌ 錯誤
  memID
  earnPts
  ```

---

## **審查範例**

### **❌ 不通過範例**

```go
// Pull Request: Add EarnPoints feature

// ❌ 問題 1: Domain Layer 依賴 Application Layer
// internal/domain/points/points_account.go
package points

import "myapp/internal/application/usecases"  // ❌ 違反依賴規則

// ❌ 問題 2: Aggregate 包含無界集合
type PointsAccount struct {
    transactions []PointsTransaction  // ❌ 無界集合
}

// ❌ 問題 3: 無不變量保護
func (a *PointsAccount) SetUsedPoints(amount int) {
    a.usedPoints = amount  // ❌ 可能 > earnedPoints
}

// ❌ 問題 4: Use Case 包含業務邏輯
// internal/application/points/earn_points_use_case.go
func (uc *EarnPointsUseCase) Execute(cmd Command) error {
    points := cmd.Amount / 100  // ❌ 業務邏輯應在 Domain Service
    if points < 0 {
        return errors.New("invalid")  // ❌ 驗證應在 Domain Layer
    }
}
```

**Code Review Comment**:
```
❌ Request Changes

1. 🔴 依賴規則違規（internal/domain/points/points_account.go:3）
   - Domain Layer 不能依賴 Application Layer
   - 移除 `import "myapp/internal/application/usecases"`

2. 🔴 Aggregate 設計問題（points_account.go:10）
   - 不應包含無界集合 `[]PointsTransaction`
   - 參考 ADR-002: Lightweight Aggregates

3. 🔴 缺少不變量保護（points_account.go:15）
   - `SetUsedPoints` 無驗證，可能違反 usedPoints <= earnedPoints
   - 使用 `DeductPoints` 並檢查餘額

4. 🔴 業務邏輯洩漏到 Use Case（earn_points_use_case.go:5-7）
   - 積分計算應在 PointsCalculationService
   - 參考 /docs/architecture/ddd/09-use-case-definitions.md

請修復後重新提交 PR。
```

---

### **✅ 通過範例**

```go
// Pull Request: Add EarnPoints feature (revised)

// ✅ Domain Layer: 純淨，無外部依賴
// internal/domain/points/points_account.go
package points

import "time"  // ✅ 僅標準庫

type PointsAccount struct {
    accountID    AccountID
    earnedPoints PointsAmount
    usedPoints   PointsAmount
    version      int  // ✅ 樂觀鎖
}

// ✅ 不變量保護
func (a *PointsAccount) DeductPoints(amount PointsAmount, reason string) error {
    if a.usedPoints.Add(amount).GreaterThan(a.earnedPoints) {
        return ErrInsufficientPoints
    }
    a.usedPoints = a.usedPoints.Add(amount)
    a.RecordEvent(PointsDeducted{...})
    return nil
}

// ✅ Use Case: 純編排
// internal/application/points/earn_points_use_case.go
func (uc *EarnPointsUseCase) Execute(cmd Command) error {
    return uc.retryWithExponentialBackoff(3, func() error {
        return uc.txManager.InTransaction(func(ctx Context) error {
            account := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
            points := uc.calculator.Calculate(cmd.Amount)  // ✅ Domain Service
            account.EarnPoints(points, cmd.Source, cmd.SourceID)
            uc.accountRepo.Update(ctx, account)
            return nil
        })
    })
}
```

**Code Review Comment**:
```
✅ Approved

良好的實現！符合所有 Clean Architecture 原則：
- ✅ 依賴方向正確
- ✅ Aggregate 輕量級設計
- ✅ 不變量保護完整
- ✅ Use Case 純編排
- ✅ 樂觀鎖實現
- ✅ 重試機制

建議：
- 🟢 考慮為 `DeductPoints` 添加單元測試覆蓋邊界情況
```

---

## **快速檢查指令**

### **自動化檢查（建議整合到 CI）**

```bash
# 檢查 Domain Layer 不依賴外層
grep -r "internal/application\|internal/infrastructure\|internal/presentation" internal/domain/ && echo "❌ FAIL: Domain dependency violation" || echo "✅ PASS"

# 檢查 Application Layer 不依賴 Infrastructure
grep -r "internal/infrastructure/persistence\|internal/presentation" internal/application/ && echo "❌ FAIL: Application dependency violation" || echo "✅ PASS"

# 檢查測試覆蓋率
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total | awk '{print $3}'
```

---

## **相關文檔**

- `/docs/architecture/ddd/11-dependency-rules.md` - 依賴規則詳解
- `/docs/architecture/ddd/07-aggregate-design-principles.md` - Aggregate 設計原則
- `/docs/architecture/ddd/09-use-case-definitions.md` - Use Case 定義
- `/docs/architecture/ddd/16-concurrency-control.md` - 併發控制策略
- `/docs/architecture/ddd/14-event-handling-implementation.md` - 事件處理實作
- `/docs/architecture/decisions/ADR-005-transaction-context-pattern.md` - Transaction Context Pattern

---

## **附錄：常見違規模式**

### **反模式 1: Anemic Domain Model（貧血模型）**

```go
// ❌ 錯誤：貧血模型
type PointsAccount struct {
    EarnedPoints int  // 公開字段
    UsedPoints   int
}

// 業務邏輯在 Service，不在 Domain
func (s *PointsService) DeductPoints(account *PointsAccount, amount int) error {
    if account.UsedPoints + amount > account.EarnedPoints {
        return errors.New("insufficient")
    }
    account.UsedPoints += amount
}

// ✅ 正確：充血模型
type PointsAccount struct {
    earnedPoints PointsAmount  // 私有字段
    usedPoints   PointsAmount
}

func (a *PointsAccount) DeductPoints(amount PointsAmount) error {
    if a.usedPoints.Add(amount).GreaterThan(a.earnedPoints) {
        return ErrInsufficientPoints
    }
    a.usedPoints = a.usedPoints.Add(amount)
    return nil
}
```

### **反模式 2: God Object（上帝對象）**

```go
// ❌ 錯誤：上帝對象
type MemberService struct {
    // 承擔過多職責
    func RegisterMember(...)
    func BindPhoneNumber(...)
    func UnbindPhoneNumber(...)
    func EarnPoints(...)
    func DeductPoints(...)
    func RecalculatePoints(...)
    func ImportIChefBatch(...)
    func SendNotification(...)
}

// ✅ 正確：職責分離
type MemberRegistrationService struct {
    func RegisterMember(...)
}

type PointsService struct {
    func EarnPoints(...)
    func DeductPoints(...)
}

type NotificationService struct {
    func SendNotification(...)
}
```

### **反模式 3: Transaction Script（事務腳本）**

```go
// ❌ 錯誤：事務腳本（所有邏輯在 Use Case）
func (uc *UseCase) Execute(cmd Command) error {
    account := uc.repo.FindByID(cmd.AccountID)

    // ❌ 業務邏輯全在 Use Case
    if account.EarnedPoints - account.UsedPoints < cmd.Amount {
        return errors.New("insufficient")
    }

    account.UsedPoints += cmd.Amount
    uc.repo.Update(account)
}

// ✅ 正確：業務邏輯在 Domain
func (uc *UseCase) Execute(cmd Command) error {
    account := uc.repo.FindByID(cmd.AccountID)
    err := account.DeductPoints(cmd.Amount, cmd.Reason)  // ✅ Aggregate 封裝邏輯
    if err != nil {
        return err
    }
    uc.repo.Update(account)
}
```

---

**最後更新**: 2025-01-09
**維護者**: Architecture Team
**反饋**: 發現遺漏項目請提交 Issue
