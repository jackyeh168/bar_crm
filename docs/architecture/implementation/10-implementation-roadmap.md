# 實作路線圖

> **版本**: 1.0
> **最後更新**: 2025-01-11
> **目標**: 提供完整的 Clean Architecture 實作計劃，基於專家級團隊的設計決策

---

## 目錄

1. [設計決策總結](#1-設計決策總結)
2. [實作原則](#2-實作原則)
3. [10 階段實作計劃](#3-10-階段實作計劃)
4. [測試策略](#4-測試策略)
5. [進度追蹤指標](#5-進度追蹤指標)
6. [立即行動步驟](#6-立即行動步驟)
7. [風險管理](#7-風險管理)

---

## 1. 設計決策總結

### 1.1 核心決策

基於專家級團隊的需求和 Clean Code 原則審查，以下為確認的設計決策：

| 決策項目 | 選擇 | 理由 |
|---------|------|------|
| **實作順序** | Domain → Application → Infrastructure → Presentation | 依賴規則，內層優先 |
| **Repository 介面** | 介面隔離（Reader/Writer/BatchReader） | ISP 原則，明確職責 |
| **錯誤處理** | 業務錯誤用 error，不變條件用 panic | Fail Fast 原則 |
| **值對象建構子** | Checked + Unchecked 雙版本 | 效能優化，內部使用 unchecked |
| **測試策略** | TDD（Domain Layer） | 測試驅動設計 |
| **測試覆蓋率** | 80% 目標 | 高品質代碼 |
| **Bounded Contexts** | 7 個（Audit 留 V2） | 完整功能，分階段實作 |

### 1.2 專案特性

* **專案性質**: 可以追求完美的 Clean Architecture
* **團隊經驗**: Go 專家級，有 DDD 實戰經驗
* **Code Review**: 有專人負責
* **最大風險**: 維護困難、業務邏輯錯誤
* **技術棧**: PostgreSQL（確定）、Redis（可選）、LINE Bot SDK（已驗證）

### 1.3 架構審查要點

**Clean Code 專家建議保留的設計**:
* ✅ 樂觀鎖由聚合控制 version，Repository 驗證 WHERE 條件
* ✅ Panic 用於不變條件檢查（需 Recovery Middleware）
* ✅ Reconstruction 驗證（防止數據損壞）
* ✅ TDD 驅動 Domain Layer 開發

**需要注意的設計權衡**:
* ⚠️ 介面隔離：專家認為過度設計，但團隊堅持使用（需確保 DI 配置正確）
* ⚠️ Unchecked 建構子：微小的效能提升，增加維護成本（需謹慎使用）

---

## 2. 實作原則

### 2.1 SOLID 原則應用

1. **SRP (單一職責原則)**
   - 每個聚合只負責一個業務概念
   - Repository 介面按職責分離（Reader/Writer）

2. **OCP (開閉原則)**
   - 使用策略模式（積分計算策略）
   - 新增功能不修改現有代碼

3. **LSP (里氏替換原則)**
   - 介面實作可互相替換
   - Mock 可替換真實實作

4. **ISP (介面隔離原則)**
   - Use Case 只依賴需要的介面
   - Reader/Writer 分離

5. **DIP (依賴反轉原則)**
   - 介面由 Domain 定義
   - Infrastructure 實作介面

### 2.2 DDD 戰術模式

* **Aggregates**: 輕量級設計，無無界集合
* **Value Objects**: 不可變，建構時驗證
* **Domain Services**: 無狀態，純函數邏輯
* **Repository**: 介面在 Domain，實作在 Infrastructure
* **Domain Events**: 所有狀態變更發布事件
* **Anti-Corruption Layer**: 隔離外部服務（LINE SDK）

### 2.3 錯誤處理策略

**業務錯誤 → 返回 error**:

```go
func NewPointsAmount(value int) (PointsAmount, error) {
    if value < 0 {
        return PointsAmount{}, ErrNegativePointsAmount  // 用戶輸入錯誤
    }
    return PointsAmount{value: value}, nil
}
```

**不變條件違反 → panic**:

```go
func (a *PointsAccount) GetAvailablePoints() PointsAmount {
    if a.usedPoints.Value() > a.earnedPoints.Value() {
        panic(fmt.Sprintf("invariant violation: used (%d) > earned (%d)",
            a.usedPoints.Value(), a.earnedPoints.Value()))
    }
    return a.earnedPoints.subtractUnchecked(a.usedPoints)
}
```

**生產環境保護**:
* Recovery Middleware 捕獲 panic
* 記錄日誌 + 發送告警
* 回傳 500 錯誤，防止服務中斷

---

## 3. 10 階段實作計劃

### Phase 1: Domain Layer - Points Context（Week 1-2）

**目標**: 完成核心域的 Domain Layer，建立範例模式

#### Week 1: Value Objects + Aggregates

**Day 1-2: PointsAmount 值對象**

**檔案結構**:

```
internal/domain/points/
├── errors.go
├── value_objects.go
└── value_objects_test.go
```

**TDD 流程**:

1. 寫測試（`value_objects_test.go`）:

```go
package points_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/jackyeh168/bar_crm/internal/domain/points"
)

// Test 1: 建構有效的 PointsAmount
func TestNewPointsAmount_ValidValue_ReturnsPointsAmount(t *testing.T) {
    // Arrange
    value := 100

    // Act
    amount, err := points.NewPointsAmount(value)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 100, amount.Value())
}

// Test 2: 建構負數應回傳錯誤
func TestNewPointsAmount_NegativeValue_ReturnsError(t *testing.T) {
    // Arrange
    value := -10

    // Act
    amount, err := points.NewPointsAmount(value)

    // Assert
    assert.Error(t, err)
    assert.ErrorIs(t, err, points.ErrNegativePointsAmount)
    assert.Equal(t, 0, amount.Value())  // 零值對象
}

// Test 3: Add 操作（不可變性）
func TestPointsAmount_Add_Immutability(t *testing.T) {
    // Arrange
    original, _ := points.NewPointsAmount(100)
    toAdd, _ := points.NewPointsAmount(50)

    // Act
    result := original.Add(toAdd)

    // Assert: 原始對象未改變
    assert.Equal(t, 100, original.Value())
    assert.Equal(t, 150, result.Value())
}

// Test 4: Subtract 操作（透明錯誤處理）
func TestPointsAmount_Subtract_Success(t *testing.T) {
    // Arrange
    minuend, _ := points.NewPointsAmount(100)
    subtrahend, _ := points.NewPointsAmount(30)

    // Act
    result, err := minuend.Subtract(subtrahend)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 70, result.Value())
    assert.Equal(t, 100, minuend.Value())  // 不可變性
}

func TestPointsAmount_Subtract_NegativeResult_ReturnsError(t *testing.T) {
    // Arrange
    minuend, _ := points.NewPointsAmount(50)
    subtrahend, _ := points.NewPointsAmount(100)

    // Act
    result, err := minuend.Subtract(subtrahend)

    // Assert: 透明的錯誤處理，不靜默截斷
    assert.Error(t, err)
    assert.ErrorIs(t, err, points.ErrNegativePointsAmount)
    assert.Equal(t, 0, result.Value())
}

// Test 5: subtractUnchecked 應該 panic（不變條件違反）
func TestPointsAmount_SubtractUnchecked_Panic(t *testing.T) {
    // 此測試驗證內部 unchecked 方法的 panic 行為
    // 注意：subtractUnchecked 是私有方法，透過 GetAvailablePoints 測試
}

// Test 6: 值相等性
func TestPointsAmount_Equals_SameValue(t *testing.T) {
    // Arrange
    amount1, _ := points.NewPointsAmount(100)
    amount2, _ := points.NewPointsAmount(100)

    // Act & Assert: 值相等性（非引用相等）
    assert.True(t, amount1.Equals(amount2))
}

func TestPointsAmount_Equals_DifferentValue(t *testing.T) {
    // Arrange
    amount1, _ := points.NewPointsAmount(100)
    amount2, _ := points.NewPointsAmount(200)

    // Act & Assert
    assert.False(t, amount1.Equals(amount2))
}

// Test 7: IsZero
func TestPointsAmount_IsZero_True(t *testing.T) {
    // Arrange
    amount, _ := points.NewPointsAmount(0)

    // Act & Assert
    assert.True(t, amount.IsZero())
}
```

2. 實作（`value_objects.go`）:

```go
package points

import (
    "errors"
    "fmt"
)

// PointsAmount 積分數量
// 設計原則：不可變、包含驗證邏輯
type PointsAmount struct {
    value int
}

// NewPointsAmount 創建積分數量（帶驗證）
// 返回錯誤而非 panic，符合 Go 慣用法和錯誤處理原則
func NewPointsAmount(value int) (PointsAmount, error) {
    if value < 0 {
        return PointsAmount{}, ErrNegativePointsAmount
    }
    return PointsAmount{value: value}, nil
}

// newPointsAmountUnchecked 創建積分數量（無驗證）
// 僅供內部算術操作使用（調用方已保證有效性）
func newPointsAmountUnchecked(value int) PointsAmount {
    return PointsAmount{value: value}
}

// Value 獲取值
func (p PointsAmount) Value() int {
    return p.value
}

// Add 相加（不可變操作，返回新對象）
func (p PointsAmount) Add(other PointsAmount) PointsAmount {
    return newPointsAmountUnchecked(p.value + other.value)
}

// Subtract 相減（不可變操作，返回錯誤而非靜默截斷）
func (p PointsAmount) Subtract(other PointsAmount) (PointsAmount, error) {
    result := p.value - other.value
    if result < 0 {
        return PointsAmount{}, ErrNegativePointsAmount
    }
    return newPointsAmountUnchecked(result), nil
}

// subtractUnchecked 相減（無驗證，假設調用方已保證有效性）
// 如果結果為負數，說明不變條件被違反，直接 panic
func (p PointsAmount) subtractUnchecked(other PointsAmount) PointsAmount {
    result := p.value - other.value
    if result < 0 {
        panic(fmt.Sprintf("subtractUnchecked: invariant violation (%d - %d < 0)",
            p.value, other.value))
    }
    return newPointsAmountUnchecked(result)
}

// Equals 判斷相等性
func (p PointsAmount) Equals(other PointsAmount) bool {
    return p.value == other.value
}

// IsZero 判斷是否為零
func (p PointsAmount) IsZero() bool {
    return p.value == 0
}
```

3. 定義錯誤（`errors.go`）:

```go
package points

import "errors"

// 積分帳戶相關錯誤
var (
    ErrNegativePointsAmount    = errors.New("points amount cannot be negative")
    ErrInsufficientPoints      = errors.New("insufficient points for this operation")
    ErrInsufficientEarnedPoints = errors.New("earned points cannot be less than used points")
)
```

4. 執行測試:

```bash
cd internal/domain/points
go test -v -cover
# 目標：100% 覆蓋率
```

**Day 3: ConversionRate + 其他值對象**

實作：
* `ConversionRate` - 包含 `CalculatePoints()` 業務邏輯
* `DateRange` - 包含 `Contains()`,  `Overlaps()` 邏輯
* `AccountID`,  `MemberID` - UUID 封裝
* `PointsSource` - 枚舉類型

**Day 4-5: PointsAccount 聚合根**

**檔案**:

```
internal/domain/points/
├── account.go
└── account_test.go
```

**測試重點**:

```go
// 命令操作測試
func TestPointsAccount_EarnPoints_Success(t *testing.T)
func TestPointsAccount_EarnPoints_NegativeAmount_ReturnsError(t *testing.T)
func TestPointsAccount_EarnPoints_PublishesEvent(t *testing.T)
func TestPointsAccount_EarnPoints_IncrementsVersion(t *testing.T)

func TestPointsAccount_DeductPoints_Success(t *testing.T)
func TestPointsAccount_DeductPoints_InsufficientPoints_ReturnsError(t *testing.T)

func TestPointsAccount_RecalculatePoints_Success(t *testing.T)
func TestPointsAccount_RecalculatePoints_ViolatesInvariant_ReturnsError(t *testing.T)

// 查詢操作測試
func TestPointsAccount_GetAvailablePoints_Success(t *testing.T)
func TestPointsAccount_GetAvailablePoints_InvariantViolation_Panics(t *testing.T) {
    // Arrange: 手動建立違反不變條件的聚合（僅測試用）
    account := &PointsAccount{
        earnedPoints: newPointsAmountUnchecked(50),
        usedPoints:   newPointsAmountUnchecked(100),  // 違反不變條件
    }

    // Act & Assert: 應該 panic
    assert.Panics(t, func() {
        account.GetAvailablePoints()
    })
}

// 版本控制測試
func TestPointsAccount_Version_IncrementsOnStateChange(t *testing.T)
func TestPointsAccount_GetPreviousVersion(t *testing.T)

// 聚合重建測試（關鍵！）
func TestReconstructPointsAccount_ValidData_Success(t *testing.T)
func TestReconstructPointsAccount_InvalidEarnedPoints_ReturnsError(t *testing.T)
func TestReconstructPointsAccount_InvalidUsedPoints_ReturnsError(t *testing.T)
func TestReconstructPointsAccount_InvariantViolation_ReturnsError(t *testing.T) {
    // Arrange: 資料庫中的損壞資料
    accountID, _ := points.AccountIDFromString("...")
    memberID, _ := points.NewMemberID("...")

    // Act: 嘗試重建（usedPoints > earnedPoints）
    account, err := points.ReconstructPointsAccount(
        accountID,
        memberID,
        50,   // earnedPoints
        100,  // usedPoints（違反不變條件）
        1,
        time.Now(),
    )

    // Assert: 應該返回錯誤，不允許載入損壞資料
    assert.Error(t, err)
    assert.Nil(t, account)
    assert.Contains(t, err.Error(), "data corruption")
}
```

**實作要點**:
* 所有字段私有
* 樂觀鎖版本控制（`version` 字段）
* `GetPreviousVersion()` 方法（供 Repository 使用）
* 領域事件收集（`events []shared.DomainEvent`）
* `ReconstructPointsAccount()` 驗證不變條件

**Day 6: ConversionRule 聚合根**

**檔案**:

```
internal/domain/points/
├── conversion_rule.go
└── conversion_rule_test.go
```

**測試重點**:
* 日期範圍重疊檢查
* 轉換率業務規則驗證
* 聚合重建驗證

**Day 7: Domain Services**

**檔案**:

```
internal/domain/points/
├── calculation_service.go
├── calculation_service_test.go
├── recalculation_service.go
└── recalculation_service_test.go
```

**實作重點**:
* 策略模式（`PointsCalculationStrategy`）
* 組合計算器（`CompositePointsCalculator`）
* 基礎積分策略（`BasePointsCalculator`）
* 問卷獎勵策略（`SurveyBonusCalculator`）

#### Week 2: Repository Interfaces + Domain Events

**Day 8-9: Repository 介面**

**檔案**:

```
internal/domain/points/repository/
├── account_repository.go
└── rule_repository.go
```

**介面隔離設計**:

```go
// account_repository.go
package repository

// PointsAccountReader 查詢操作介面
type PointsAccountReader interface {
    FindByID(ctx shared.TransactionContext, accountID points.AccountID) (*points.PointsAccount, error)
    FindByMemberID(ctx shared.TransactionContext, memberID points.MemberID) (*points.PointsAccount, error)
    ExistsByMemberID(ctx shared.TransactionContext, memberID points.MemberID) (bool, error)
}

// PointsAccountWriter 寫操作介面
type PointsAccountWriter interface {
    Create(ctx shared.TransactionContext, account *points.PointsAccount) error
    Update(ctx shared.TransactionContext, account *points.PointsAccount) error
}

// PointsAccountBatchReader 批次查詢介面
type PointsAccountBatchReader interface {
    FindAll(ctx shared.TransactionContext) ([]*points.PointsAccount, error)
    FindByIDs(ctx shared.TransactionContext, accountIDs []points.AccountID) ([]*points.PointsAccount, error)
}

// PointsAccountRepository 完整倉儲介面（組合所有小介面）
type PointsAccountRepository interface {
    PointsAccountReader
    PointsAccountWriter
    PointsAccountBatchReader
}

// 倉儲錯誤
var (
    ErrAccountNotFound        = errors.New("points account not found")
    ErrAccountAlreadyExists   = errors.New("points account already exists")
    ErrConcurrentModification = errors.New("concurrent modification detected")
)
```

**Day 10: 領域事件**

**檔案**:

```
internal/domain/points/
├── events.go
└── events_test.go

internal/domain/shared/
├── transaction.go
└── event.go
```

**實作要點**:
* 所有事件實作 `shared.DomainEvent` 介面
* 事件不可變
* 事件名稱使用過去式（`PointsEarned`,  `PointsDeducted`）
* 事件包含完整信息

**檢查點（Week 2 結束）**:

```bash
✅ Points Context 完整的 Domain Layer
✅ 100% 的單元測試覆蓋率（Domain 層）
✅ 0 個外部依賴（無 GORM, 無 HTTP）
✅ 可以跑的測試套件

# 執行檢查
cd internal/domain/points
go test -v -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out

# 預期結果
PASS
coverage: 95.0% of statements
```

---

### Phase 2: Domain Layer - Member + Invoice Context（Week 3）

**目標**: 完成支撐域的 Domain Layer

#### Week 3: Member + Invoice Context

**Member Context 結構**:

```
internal/domain/member/
├── member.go                    # Member 聚合根
├── member_test.go
├── value_objects.go             # PhoneNumber, LineUserID, DisplayName
├── value_objects_test.go
├── events.go                    # MemberRegistered, PhoneNumberBound
├── events_test.go
├── service.go                   # MemberRegistrationService
├── service_test.go
├── errors.go
└── repository/
    └── member_repository.go     # Reader/Writer 介面隔離
```

**PhoneNumber 值對象測試重點**:

```go
func TestNewPhoneNumber_ValidFormat_Success(t *testing.T) {
    // 測試台灣手機格式：09XXXXXXXX
}

func TestNewPhoneNumber_InvalidFormat_ReturnsError(t *testing.T) {
    // 測試無效格式
}

func TestPhoneNumber_Equals(t *testing.T) {
    // 測試值相等性
}
```

**Invoice Context 結構**:

```
internal/domain/invoice/
├── transaction.go               # InvoiceTransaction 聚合根
├── transaction_test.go
├── value_objects.go             # InvoiceNumber, Money, InvoiceStatus
├── value_objects_test.go
├── parsing_service.go           # QR Code 解析服務
├── parsing_service_test.go
├── validation_service.go        # 發票驗證服務（60天有效期）
├── validation_service_test.go
├── events.go                    # TransactionVerified, TransactionFailed
├── errors.go
└── repository/
    └── transaction_repository.go
```

**InvoiceTransaction 測試重點**:

```go
func TestInvoiceTransaction_Verify_Success(t *testing.T)
func TestInvoiceTransaction_Verify_AlreadyVerified_ReturnsError(t *testing.T)
func TestInvoiceTransaction_Verify_PublishesEvent(t *testing.T)
func TestInvoiceTransaction_IsExpired_60DaysRule(t *testing.T)
```

**檢查點（Week 3 結束）**:

```bash
✅ Member Context Domain Layer 完成
✅ Invoice Context Domain Layer 完成
✅ 測試覆蓋率 90%+
✅ 所有值對象都有 checked/unchecked 建構子
```

---

### Phase 3: Domain Layer - Survey + External Context（Week 4）

**Survey Context 結構**:

```
internal/domain/survey/
├── survey.go                    # Survey 聚合根（含巢狀 Question 實體）
├── survey_test.go
├── response.go                  # SurveyResponse 聚合根
├── response_test.go
├── value_objects.go             # QuestionType, RatingScore, Token
├── value_objects_test.go
├── events.go                    # SurveyActivated, SurveyResponseSubmitted
├── errors.go
└── repository/
    ├── survey_repository.go
    └── response_repository.go
```

**Survey 聚合根測試重點**:

```go
func TestSurvey_AddQuestion_Success(t *testing.T)
func TestSurvey_Activate_OnlyOneActiveSurvey_Rule(t *testing.T)
func TestSurvey_Deactivate_Success(t *testing.T)
func TestReconstructSurvey_NoQuestions_ReturnsError(t *testing.T) {
    // 不變條件：Survey 必須至少有一個問題
}
```

**External Context** (iChef 整合):

```
internal/domain/external/
├── import_batch.go              # ImportBatch 聚合根
├── import_batch_test.go
├── import_record.go             # ImportedInvoiceRecord 實體
├── import_record_test.go
├── value_objects.go             # ImportStatistics, MatchStatus
├── matching_service.go          # 發票匹配服務
├── matching_service_test.go
├── events.go                    # BatchImportCompleted
├── errors.go
└── repository/
    └── import_repository.go
```

**檢查點（Week 4 結束）**:

```bash
✅ Survey Context Domain Layer 完成
✅ External Context Domain Layer 完成
✅ 4 個 Context 的 Domain Layer 全部完成
```

---

### Phase 4: Domain Layer - Identity + Notification Context（Week 5）

**Identity Context 結構**:

```
internal/domain/identity/
├── admin_user.go                # AdminUser 聚合根
├── admin_user_test.go
├── value_objects.go             # Role, Permission, Email
├── value_objects_test.go
├── events.go                    # AdminUserCreated, RoleChanged
├── errors.go
└── repository/
    └── admin_repository.go
```

**Role 值對象測試**:

```go
func TestRole_HasPermission_AdminRole(t *testing.T)
func TestRole_HasPermission_UserRole(t *testing.T)
func TestRole_HasPermission_GuestRole(t *testing.T)
```

**Notification Context 結構**:

```
internal/domain/notification/
├── notification.go              # Notification 聚合根
├── notification_test.go
├── value_objects.go             # MessageContent, NotificationType, Recipient
├── value_objects_test.go
├── events.go                    # NotificationSent
├── errors.go
└── repository/
    └── notification_repository.go
```

**檢查點（Week 5 結束）**:

```bash
✅ Identity Context Domain Layer 完成
✅ Notification Context Domain Layer 完成
✅ 6 個 Context（除 Audit）的 Domain Layer 全部完成
✅ 整體測試覆蓋率 85%+
✅ 所有聚合根都實作樂觀鎖
✅ 所有不變條件都有 panic 檢查
✅ 所有 Reconstruction 都驗證資料完整性
```

---

### Phase 5: Application Layer（Week 6-7）

**目標**: 實作 Use Cases、DTOs、Event Handlers

#### Week 6: Use Cases (TDD with Mocks)

**Points Use Cases**:

```
internal/application/usecases/points/
├── earn_points.go
├── earn_points_test.go
├── deduct_points.go
├── deduct_points_test.go
├── query_points.go
├── query_points_test.go
├── recalculate_points.go
├── recalculate_points_test.go
├── create_rule.go
├── create_rule_test.go
└── update_rule.go
```

**EarnPointsUseCase 測試範例**:

```go
package points_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/jackyeh168/bar_crm/internal/domain/points"
    "github.com/jackyeh168/bar_crm/internal/domain/points/repository"
    "github.com/jackyeh168/bar_crm/internal/application/usecases/points"
)

// MockPointsAccountReader Mock 讀取介面
type MockPointsAccountReader struct {
    mock.Mock
}

func (m *MockPointsAccountReader) FindByMemberID(ctx shared.TransactionContext, memberID points.MemberID) (*points.PointsAccount, error) {
    args := m.Called(ctx, memberID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*points.PointsAccount), args.Error(1)
}

// MockPointsAccountWriter Mock 寫入介面
type MockPointsAccountWriter struct {
    mock.Mock
}

func (m *MockPointsAccountWriter) Update(ctx shared.TransactionContext, account *points.PointsAccount) error {
    args := m.Called(ctx, account)
    return args.Error(0)
}

// Test: 成功獲得積分
func TestEarnPointsUseCase_Execute_Success(t *testing.T) {
    // Arrange
    mockReader := new(MockPointsAccountReader)
    mockWriter := new(MockPointsAccountWriter)
    mockTxManager := new(MockTransactionManager)

    memberID, _ := points.NewMemberID("member-123")
    account, _ := points.NewPointsAccount(memberID)

    // Mock 行為
    mockReader.On("FindByMemberID", mock.Anything, memberID).Return(account, nil)
    mockWriter.On("Update", mock.Anything, mock.AnythingOfType("*points.PointsAccount")).Return(nil)
    mockTxManager.On("InTransaction", mock.AnythingOfType("func(shared.TransactionContext) error")).
        Run(func(args mock.Arguments) {
            fn := args.Get(0).(func(shared.TransactionContext) error)
            fn(nil)  // 執行事務函數
        }).Return(nil)

    useCase := points.NewEarnPointsUseCase(mockReader, mockWriter, mockTxManager)

    cmd := points.EarnPointsCommand{
        MemberID:    memberID,
        Amount:      100,
        Source:      points.PointsSourceInvoice,
        SourceID:    "invoice-123",
        Description: "購買商品",
    }

    // Act
    result, err := useCase.Execute(cmd)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 100, result.EarnedPoints)

    // 驗證 Mock 被正確調用
    mockReader.AssertExpectations(t)
    mockWriter.AssertExpectations(t)
}

// Test: 帳戶不存在
func TestEarnPointsUseCase_Execute_AccountNotFound_ReturnsError(t *testing.T) {
    // Arrange
    mockReader := new(MockPointsAccountReader)
    memberID, _ := points.NewMemberID("member-999")

    mockReader.On("FindByMemberID", mock.Anything, memberID).
        Return(nil, repository.ErrAccountNotFound)

    useCase := points.NewEarnPointsUseCase(mockReader, nil, nil)

    cmd := points.EarnPointsCommand{
        MemberID: memberID,
        Amount:   100,
    }

    // Act
    result, err := useCase.Execute(cmd)

    // Assert
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.ErrorIs(t, err, repository.ErrAccountNotFound)
}

// Test: 負數積分
func TestEarnPointsUseCase_Execute_NegativeAmount_ReturnsError(t *testing.T) {
    // Arrange
    mockReader := new(MockPointsAccountReader)
    memberID, _ := points.NewMemberID("member-123")
    account, _ := points.NewPointsAccount(memberID)

    mockReader.On("FindByMemberID", mock.Anything, memberID).Return(account, nil)

    useCase := points.NewEarnPointsUseCase(mockReader, nil, nil)

    cmd := points.EarnPointsCommand{
        MemberID: memberID,
        Amount:   -100,  // 負數
    }

    // Act
    result, err := useCase.Execute(cmd)

    // Assert
    assert.Error(t, err)
    assert.Nil(t, result)
}
```

**其他 Context 的 Use Cases**:

```
internal/application/usecases/
├── member/
│   ├── register_member.go
│   ├── register_member_test.go
│   ├── bind_phone.go
│   ├── bind_phone_test.go
│   ├── get_member.go
│   └── unbind_phone.go
├── invoice/
│   ├── scan_invoice.go
│   ├── scan_invoice_test.go
│   ├── verify_transaction.go
│   └── query_transactions.go
├── survey/
│   ├── create_survey.go
│   ├── activate_survey.go
│   ├── submit_response.go
│   └── query_active_survey.go
└── external/
    └── import_ichef_batch.go
```

#### Week 7: DTOs + Event Handlers

**DTOs**:

```
internal/application/dto/
├── member_dto.go
├── points_dto.go
├── invoice_dto.go
└── survey_dto.go
```

**TransactionDTO 實作 Domain 介面**（解耦 Application 和 Domain）:

```go
package dto

import (
    "time"
    "github.com/shopspring/decimal"
    "github.com/jackyeh168/bar_crm/internal/domain/points"
)

// TransactionDTO 交易 DTO
type TransactionDTO struct {
    TransactionID   string          `json:"transaction_id"`
    Amount          decimal.Decimal `json:"amount"`
    InvoiceDate     time.Time       `json:"invoice_date"`
    SurveySubmitted bool            `json:"survey_submitted"`
}

// 實作 points.PointsCalculableTransaction 介面
var _ points.PointsCalculableTransaction = (*TransactionDTO)(nil)

func (d TransactionDTO) GetTransactionAmount() decimal.Decimal {
    return d.Amount
}

func (d TransactionDTO) GetTransactionDate() time.Time {
    return d.InvoiceDate
}

func (d TransactionDTO) HasCompletedSurvey() bool {
    return d.SurveySubmitted
}
```

**Event Handlers**:

```
internal/application/events/
├── points/
│   ├── transaction_verified_handler.go   # 發票驗證後計算積分
│   ├── transaction_verified_handler_test.go
│   ├── survey_completed_handler.go       # 問卷完成後加分
│   └── survey_completed_handler_test.go
└── notification/
    ├── welcome_notification_handler.go   # 會員註冊後發送歡迎訊息
    ├── points_earned_handler.go          # 積分獲得後發送通知
    └── transaction_verified_handler.go   # 發票驗證後發送通知
```

**TransactionVerifiedHandler 範例**:

```go
package points

import (
    "github.com/jackyeh168/bar_crm/internal/domain/shared"
    "github.com/jackyeh168/bar_crm/internal/domain/invoice"
    "github.com/jackyeh168/bar_crm/internal/application/usecases/points"
)

// TransactionVerifiedHandler 處理 TransactionVerified 事件
type TransactionVerifiedHandler struct {
    earnPointsUseCase *points.EarnPointsUseCase
}

func NewTransactionVerifiedHandler(
    earnPointsUseCase *points.EarnPointsUseCase,
) *TransactionVerifiedHandler {
    return &TransactionVerifiedHandler{
        earnPointsUseCase: earnPointsUseCase,
    }
}

// Handle 處理事件
func (h *TransactionVerifiedHandler) Handle(event shared.DomainEvent) error {
    // 類型斷言
    verifiedEvent, ok := event.(invoice.TransactionVerified)
    if !ok {
        return fmt.Errorf("unexpected event type: %T", event)
    }

    // 執行 Use Case：獲得積分
    cmd := points.EarnPointsCommand{
        MemberID:    verifiedEvent.MemberID(),
        Amount:      verifiedEvent.Amount().IntPart(),
        Source:      points.PointsSourceInvoice,
        SourceID:    verifiedEvent.TransactionID().String(),
        Description: fmt.Sprintf("發票 %s 驗證通過", verifiedEvent.InvoiceNumber()),
    }

    _, err := h.earnPointsUseCase.Execute(cmd)
    return err
}

// EventType 返回處理的事件類型
func (h *TransactionVerifiedHandler) EventType() string {
    return "invoice.transaction_verified"
}
```

**檢查點（Week 7 結束）**:

```bash
✅ 所有 Use Cases 實作完成
✅ Use Case 測試覆蓋率 80%+（使用 Mock）
✅ DTOs 實作 Domain 介面（解耦）
✅ Event Handlers 實作完成
✅ Event Handlers 測試完成
```

---

### Phase 6: Infrastructure Layer - Persistence（Week 8）

**目標**: 實作 GORM Repositories、Transaction Context、Integration Tests

#### Week 8: GORM Repositories

**Step 1: GORM Models 集中定義**

**檔案**: `internal/infrastructure/persistence/gorm/models.go`

```go
package gorm

import "gorm.io/gorm"

// PointsAccountModel 積分帳戶表
type PointsAccountModel struct {
    gorm.Model
    AccountID    string `gorm:"uniqueIndex;type:varchar(36);not null"`
    MemberID     string `gorm:"index;type:varchar(36);not null"`
    EarnedPoints int    `gorm:"not null;default:0;check:earned_points >= 0"`
    UsedPoints   int    `gorm:"not null;default:0;check:used_points >= 0"`
    Version      int    `gorm:"not null;default:1"`

    // 索引
    // Index: idx_member_id
    // Unique Index: idx_account_id
}

// TableName 指定表名
func (PointsAccountModel) TableName() string {
    return "points_accounts"
}

// ConversionRuleModel 轉換規則表
type ConversionRuleModel struct {
    gorm.Model
    RuleID         string `gorm:"uniqueIndex;type:varchar(36);not null"`
    ConversionRate int    `gorm:"not null;check:conversion_rate >= 1 AND conversion_rate <= 1000"`
    StartDate      time.Time `gorm:"not null;index:idx_date_range"`
    EndDate        time.Time `gorm:"not null;index:idx_date_range"`
    Version        int    `gorm:"not null;default:1"`
}

func (ConversionRuleModel) TableName() string {
    return "conversion_rules"
}

// MemberModel 會員表
type MemberModel struct {
    gorm.Model
    MemberID      string `gorm:"uniqueIndex;type:varchar(36);not null"`
    LineUserID    string `gorm:"uniqueIndex;type:varchar(100);not null"`
    PhoneNumber   string `gorm:"uniqueIndex;type:varchar(20)"`
    DisplayName   string `gorm:"type:varchar(100)"`
    IsBound       bool   `gorm:"not null;default:false"`
    Version       int    `gorm:"not null;default:1"`
}

func (MemberModel) TableName() string {
    return "members"
}

// InvoiceTransactionModel 發票交易表
type InvoiceTransactionModel struct {
    gorm.Model
    TransactionID  string `gorm:"uniqueIndex;type:varchar(36);not null"`
    MemberID       string `gorm:"index;type:varchar(36);not null"`
    InvoiceNumber  string `gorm:"uniqueIndex;type:varchar(50);not null"`
    Amount         string `gorm:"type:decimal(10,2);not null"`  // 使用 string 儲存 decimal
    InvoiceDate    time.Time `gorm:"not null;index"`
    Status         string `gorm:"type:varchar(20);not null;index"`
    SurveyToken    string `gorm:"type:varchar(100);index"`
    Version        int    `gorm:"not null;default:1"`
}

func (InvoiceTransactionModel) TableName() string {
    return "invoice_transactions"
}

// SurveyModel 問卷表
type SurveyModel struct {
    gorm.Model
    SurveyID    string `gorm:"uniqueIndex;type:varchar(36);not null"`
    Title       string `gorm:"type:varchar(200);not null"`
    Description string `gorm:"type:text"`
    IsActive    bool   `gorm:"not null;default:false;index"`
    Version     int    `gorm:"not null;default:1"`

    // 關聯（巢狀實體）
    Questions []SurveyQuestionModel `gorm:"foreignKey:SurveyID;references:SurveyID"`
}

func (SurveyModel) TableName() string {
    return "surveys"
}

// SurveyQuestionModel 問卷題目表（實體，非聚合根）
type SurveyQuestionModel struct {
    gorm.Model
    QuestionID   string `gorm:"uniqueIndex;type:varchar(36);not null"`
    SurveyID     string `gorm:"index;type:varchar(36);not null"`
    QuestionText string `gorm:"type:text;not null"`
    QuestionType string `gorm:"type:varchar(20);not null"`
    Order        int    `gorm:"not null"`
}

func (SurveyQuestionModel) TableName() string {
    return "survey_questions"
}

// SurveyResponseModel 問卷回覆表
type SurveyResponseModel struct {
    gorm.Model
    ResponseID    string `gorm:"uniqueIndex;type:varchar(36);not null"`
    SurveyID      string `gorm:"index;type:varchar(36);not null"`
    TransactionID string `gorm:"uniqueIndex;type:varchar(36)"`
    Token         string `gorm:"uniqueIndex;type:varchar(100);not null"`
    SubmittedAt   time.Time
    Version       int `gorm:"not null;default:1"`

    // 關聯
    Answers []SurveyAnswerModel `gorm:"foreignKey:ResponseID;references:ResponseID"`
}

func (SurveyResponseModel) TableName() string {
    return "survey_responses"
}

// SurveyAnswerModel 問卷答案表
type SurveyAnswerModel struct {
    gorm.Model
    AnswerID   string `gorm:"uniqueIndex;type:varchar(36);not null"`
    ResponseID string `gorm:"index;type:varchar(36);not null"`
    QuestionID string `gorm:"index;type:varchar(36);not null"`
    AnswerText string `gorm:"type:text"`
    Rating     int    `gorm:"check:rating >= 1 AND rating <= 5"`
}

func (SurveyAnswerModel) TableName() string {
    return "survey_answers"
}

// ImportBatchModel iChef 匯入批次表
type ImportBatchModel struct {
    gorm.Model
    BatchID     string `gorm:"uniqueIndex;type:varchar(36);not null"`
    FileName    string `gorm:"type:varchar(255);not null"`
    TotalCount  int    `gorm:"not null"`
    MatchedCount int   `gorm:"not null;default:0"`
    ImportedAt  time.Time `gorm:"not null"`
    Version     int    `gorm:"not null;default:1"`

    // 關聯
    Records []ImportRecordModel `gorm:"foreignKey:BatchID;references:BatchID"`
}

func (ImportBatchModel) TableName() string {
    return "import_batches"
}

// ImportRecordModel 匯入記錄表
type ImportRecordModel struct {
    gorm.Model
    RecordID      string `gorm:"uniqueIndex;type:varchar(36);not null"`
    BatchID       string `gorm:"index;type:varchar(36);not null"`
    InvoiceNumber string `gorm:"index;type:varchar(50);not null"`
    Amount        string `gorm:"type:decimal(10,2);not null"`
    InvoiceDate   time.Time `gorm:"not null"`
    MatchStatus   string `gorm:"type:varchar(20);not null;index"`
}

func (ImportRecordModel) TableName() string {
    return "import_records"
}

// AdminUserModel 管理員表
type AdminUserModel struct {
    gorm.Model
    UserID   string `gorm:"uniqueIndex;type:varchar(36);not null"`
    Email    string `gorm:"uniqueIndex;type:varchar(100);not null"`
    Role     string `gorm:"type:varchar(20);not null;index"`
    Version  int    `gorm:"not null;default:1"`
}

func (AdminUserModel) TableName() string {
    return "admin_users"
}

// NotificationModel 通知表
type NotificationModel struct {
    gorm.Model
    NotificationID string `gorm:"uniqueIndex;type:varchar(36);not null"`
    RecipientID    string `gorm:"index;type:varchar(36);not null"`
    Type           string `gorm:"type:varchar(50);not null;index"`
    Content        string `gorm:"type:text;not null"`
    SentAt         time.Time
    Status         string `gorm:"type:varchar(20);not null;index"`
    Version        int    `gorm:"not null;default:1"`
}

func (NotificationModel) TableName() string {
    return "notifications"
}
```

**Step 2: Transaction Context 實作**

**檔案**: `internal/infrastructure/persistence/gorm/transaction.go`

```go
package gorm

import (
    "gorm.io/gorm"
    "github.com/jackyeh168/bar_crm/internal/domain/shared"
)

// GormTransactionContext GORM 事務上下文
type GormTransactionContext struct {
    tx *gorm.DB
}

// 確保實作介面
var _ shared.TransactionContext = (*GormTransactionContext)(nil)

// GetDB 取得 GORM DB（僅供 Infrastructure 層使用）
func (ctx *GormTransactionContext) GetDB() *gorm.DB {
    return ctx.tx
}

// GormTransactionManager GORM 事務管理器
type GormTransactionManager struct {
    db *gorm.DB
}

// 確保實作介面
var _ shared.TransactionManager = (*GormTransactionManager)(nil)

func NewGormTransactionManager(db *gorm.DB) *GormTransactionManager {
    return &GormTransactionManager{db: db}
}

// InTransaction 在事務中執行函數
func (m *GormTransactionManager) InTransaction(fn func(ctx shared.TransactionContext) error) error {
    return m.db.Transaction(func(tx *gorm.DB) error {
        ctx := &GormTransactionContext{tx: tx}
        return fn(ctx)
    })
}
```

**Step 3: Repository 實作**

**檔案**: `internal/infrastructure/persistence/points/gorm_account_repository.go`

```go
package points

import (
    "errors"
    "fmt"
    "gorm.io/gorm"

    "github.com/jackyeh168/bar_crm/internal/domain/points"
    "github.com/jackyeh168/bar_crm/internal/domain/points/repository"
    "github.com/jackyeh168/bar_crm/internal/domain/shared"
    gorminfra "github.com/jackyeh168/bar_crm/internal/infrastructure/persistence/gorm"
)

// GormPointsAccountRepository GORM 積分帳戶倉儲
type GormPointsAccountRepository struct {
    db *gorm.DB
}

// 確保實作所有介面
var (
    _ repository.PointsAccountReader      = (*GormPointsAccountRepository)(nil)
    _ repository.PointsAccountWriter      = (*GormPointsAccountRepository)(nil)
    _ repository.PointsAccountBatchReader = (*GormPointsAccountRepository)(nil)
    _ repository.PointsAccountRepository  = (*GormPointsAccountRepository)(nil)
)

func NewGormPointsAccountRepository(db *gorm.DB) *GormPointsAccountRepository {
    return &GormPointsAccountRepository{db: db}
}

// --- Reader 介面實作 ---

// FindByID 根據帳戶 ID 查詢
func (r *GormPointsAccountRepository) FindByID(
    ctx shared.TransactionContext,
    accountID points.AccountID,
) (*points.PointsAccount, error) {
    db := r.getDB(ctx)

    var model gorminfra.PointsAccountModel
    err := db.Where("account_id = ?", accountID.String()).First(&model).Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, repository.ErrAccountNotFound
        }
        return nil, fmt.Errorf("database error: %w", err)
    }

    return r.toDomain(model)
}

// FindByMemberID 根據會員 ID 查詢
func (r *GormPointsAccountRepository) FindByMemberID(
    ctx shared.TransactionContext,
    memberID points.MemberID,
) (*points.PointsAccount, error) {
    db := r.getDB(ctx)

    var model gorminfra.PointsAccountModel
    err := db.Where("member_id = ?", memberID.String()).First(&model).Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, repository.ErrAccountNotFound
        }
        return nil, fmt.Errorf("database error: %w", err)
    }

    return r.toDomain(model)
}

// ExistsByMemberID 檢查會員是否已有積分帳戶
func (r *GormPointsAccountRepository) ExistsByMemberID(
    ctx shared.TransactionContext,
    memberID points.MemberID,
) (bool, error) {
    db := r.getDB(ctx)

    var count int64
    err := db.Model(&gorminfra.PointsAccountModel{}).
        Where("member_id = ?", memberID.String()).
        Count(&count).Error

    if err != nil {
        return false, fmt.Errorf("database error: %w", err)
    }

    return count > 0, nil
}

// --- Writer 介面實作 ---

// Create 創建新的積分帳戶
func (r *GormPointsAccountRepository) Create(
    ctx shared.TransactionContext,
    account *points.PointsAccount,
) error {
    db := r.getDB(ctx)

    model := r.toModel(account)

    err := db.Create(&model).Error
    if err != nil {
        if errors.Is(err, gorm.ErrDuplicatedKey) {
            return repository.ErrAccountAlreadyExists
        }
        return fmt.Errorf("database error: %w", err)
    }

    return nil
}

// Update 更新積分帳戶（支持樂觀鎖）
func (r *GormPointsAccountRepository) Update(
    ctx shared.TransactionContext,
    account *points.PointsAccount,
) error {
    db := r.getDB(ctx)

    // 關鍵：使用樂觀鎖
    result := db.Model(&gorminfra.PointsAccountModel{}).
        Where("account_id = ? AND version = ?",
            account.GetAccountID().String(),
            account.GetPreviousVersion()).  // 使用上一個版本號
        Updates(map[string]interface{}{
            "earned_points": account.GetEarnedPoints().Value(),
            "used_points":   account.GetUsedPoints().Value(),
            "version":       account.GetVersion(),  // 新版本號
            "updated_at":    account.GetLastUpdatedAt(),
        })

    if result.Error != nil {
        return fmt.Errorf("database error: %w", result.Error)
    }

    // 檢查樂觀鎖
    if result.RowsAffected == 0 {
        return repository.ErrConcurrentModification
    }

    return nil
}

// --- BatchReader 介面實作 ---

// FindAll 查詢所有積分帳戶
func (r *GormPointsAccountRepository) FindAll(
    ctx shared.TransactionContext,
) ([]*points.PointsAccount, error) {
    db := r.getDB(ctx)

    var models []gorminfra.PointsAccountModel
    err := db.Find(&models).Error
    if err != nil {
        return nil, fmt.Errorf("database error: %w", err)
    }

    accounts := make([]*points.PointsAccount, 0, len(models))
    for _, model := range models {
        account, err := r.toDomain(model)
        if err != nil {
            return nil, err
        }
        accounts = append(accounts, account)
    }

    return accounts, nil
}

// FindByIDs 批次查詢
func (r *GormPointsAccountRepository) FindByIDs(
    ctx shared.TransactionContext,
    accountIDs []points.AccountID,
) ([]*points.PointsAccount, error) {
    db := r.getDB(ctx)

    ids := make([]string, len(accountIDs))
    for i, id := range accountIDs {
        ids[i] = id.String()
    }

    var models []gorminfra.PointsAccountModel
    err := db.Where("account_id IN ?", ids).Find(&models).Error
    if err != nil {
        return nil, fmt.Errorf("database error: %w", err)
    }

    accounts := make([]*points.PointsAccount, 0, len(models))
    for _, model := range models {
        account, err := r.toDomain(model)
        if err != nil {
            return nil, err
        }
        accounts = append(accounts, account)
    }

    return accounts, nil
}

// --- 私有輔助方法 ---

// getDB 從上下文取得 DB（事務或普通 DB）
func (r *GormPointsAccountRepository) getDB(ctx shared.TransactionContext) *gorm.DB {
    if ctx == nil {
        return r.db
    }

    if gormCtx, ok := ctx.(*gorminfra.GormTransactionContext); ok {
        return gormCtx.GetDB()
    }

    return r.db
}

// toDomain 將 GORM Model 轉換為 Domain Entity
func (r *GormPointsAccountRepository) toDomain(model gorminfra.PointsAccountModel) (*points.PointsAccount, error) {
    accountID, err := points.AccountIDFromString(model.AccountID)
    if err != nil {
        return nil, fmt.Errorf("invalid account ID in database: %w", err)
    }

    memberID, err := points.NewMemberID(model.MemberID)
    if err != nil {
        return nil, fmt.Errorf("invalid member ID in database: %w", err)
    }

    // 使用 ReconstructPointsAccount（會驗證不變條件）
    account, err := points.ReconstructPointsAccount(
        accountID,
        memberID,
        model.EarnedPoints,
        model.UsedPoints,
        model.Version,
        model.UpdatedAt,
    )

    if err != nil {
        return nil, fmt.Errorf("failed to reconstruct account: %w", err)
    }

    return account, nil
}

// toModel 將 Domain Entity 轉換為 GORM Model
func (r *GormPointsAccountRepository) toModel(account *points.PointsAccount) gorminfra.PointsAccountModel {
    return gorminfra.PointsAccountModel{
        AccountID:    account.GetAccountID().String(),
        MemberID:     account.GetMemberID().String(),
        EarnedPoints: account.GetEarnedPoints().Value(),
        UsedPoints:   account.GetUsedPoints().Value(),
        Version:      account.GetVersion(),
    }
}
```

**Step 4: Integration Tests**

**檔案**: `internal/infrastructure/persistence/points/gorm_account_repository_test.go`

```go
package points_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "github.com/jackyeh168/bar_crm/internal/domain/points"
    "github.com/jackyeh168/bar_crm/internal/domain/points/repository"
    gorminfra "github.com/jackyeh168/bar_crm/internal/infrastructure/persistence/gorm"
    pointsrepo "github.com/jackyeh168/bar_crm/internal/infrastructure/persistence/points"
)

// RepositoryTestSuite 倉儲測試套件
type RepositoryTestSuite struct {
    suite.Suite
    db   *gorm.DB
    repo *pointsrepo.GormPointsAccountRepository
}

// SetupTest 每個測試前執行
func (suite *RepositoryTestSuite) SetupTest() {
    // 使用 SQLite in-memory 資料庫
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    assert.NoError(suite.T(), err)

    // 自動遷移
    err = db.AutoMigrate(&gorminfra.PointsAccountModel{})
    assert.NoError(suite.T(), err)

    suite.db = db
    suite.repo = pointsrepo.NewGormPointsAccountRepository(db)
}

// TearDownTest 每個測試後執行
func (suite *RepositoryTestSuite) TearDownTest() {
    sqlDB, _ := suite.db.DB()
    sqlDB.Close()
}

// Test 1: 創建積分帳戶
func (suite *RepositoryTestSuite) TestCreate_Success() {
    // Arrange
    memberID, _ := points.NewMemberID("member-123")
    account, _ := points.NewPointsAccount(memberID)

    // Act
    err := suite.repo.Create(nil, account)

    // Assert
    assert.NoError(suite.T(), err)

    // 驗證資料庫中有資料
    var model gorminfra.PointsAccountModel
    err = suite.db.Where("account_id = ?", account.GetAccountID().String()).First(&model).Error
    assert.NoError(suite.T(), err)
    assert.Equal(suite.T(), memberID.String(), model.MemberID)
}

// Test 2: 重複創建應失敗
func (suite *RepositoryTestSuite) TestCreate_Duplicate_ReturnsError() {
    // Arrange
    memberID, _ := points.NewMemberID("member-123")
    account, _ := points.NewPointsAccount(memberID)

    // Act: 第一次創建
    err := suite.repo.Create(nil, account)
    assert.NoError(suite.T(), err)

    // Act: 第二次創建（相同 MemberID）
    account2, _ := points.NewPointsAccount(memberID)
    err = suite.repo.Create(nil, account2)

    // Assert
    assert.Error(suite.T(), err)
    assert.ErrorIs(suite.T(), err, repository.ErrAccountAlreadyExists)
}

// Test 3: 根據 MemberID 查詢
func (suite *RepositoryTestSuite) TestFindByMemberID_Success() {
    // Arrange
    memberID, _ := points.NewMemberID("member-123")
    account, _ := points.NewPointsAccount(memberID)
    account.EarnPoints(pointsAmount(100), points.PointsSourceInvoice, "inv-1", "test")

    err := suite.repo.Create(nil, account)
    assert.NoError(suite.T(), err)

    // Act
    found, err := suite.repo.FindByMemberID(nil, memberID)

    // Assert
    assert.NoError(suite.T(), err)
    assert.NotNil(suite.T(), found)
    assert.Equal(suite.T(), account.GetAccountID().String(), found.GetAccountID().String())
    assert.Equal(suite.T(), 100, found.GetEarnedPoints().Value())
}

// Test 4: 樂觀鎖測試（關鍵！）
func (suite *RepositoryTestSuite) TestUpdate_OptimisticLock_Success() {
    // Arrange: 創建帳戶
    memberID, _ := points.NewMemberID("member-123")
    account, _ := points.NewPointsAccount(memberID)
    err := suite.repo.Create(nil, account)
    assert.NoError(suite.T(), err)

    // Act 1: 第一次更新
    account.EarnPoints(pointsAmount(100), points.PointsSourceInvoice, "inv-1", "test")
    err = suite.repo.Update(nil, account)
    assert.NoError(suite.T(), err)

    // Act 2: 載入同一個帳戶（獲取最新版本）
    account2, err := suite.repo.FindByMemberID(nil, memberID)
    assert.NoError(suite.T(), err)

    // Act 3: account2 進行第二次更新
    account2.EarnPoints(pointsAmount(50), points.PointsSourceInvoice, "inv-2", "test")
    err = suite.repo.Update(nil, account2)
    assert.NoError(suite.T(), err)

    // Act 4: 嘗試用舊的 account 更新（版本已過期）
    account.EarnPoints(pointsAmount(30), points.PointsSourceInvoice, "inv-3", "test")
    err = suite.repo.Update(nil, account)

    // Assert: 應該失敗（樂觀鎖衝突）
    assert.Error(suite.T(), err)
    assert.ErrorIs(suite.T(), err, repository.ErrConcurrentModification)
}

// Test 5: 重建驗證（資料損壞檢測）
func (suite *RepositoryTestSuite) TestReconstruction_DataCorruption_ReturnsError() {
    // Arrange: 手動插入損壞資料
    corruptedModel := gorminfra.PointsAccountModel{
        AccountID:    "acc-999",
        MemberID:     "member-999",
        EarnedPoints: 50,
        UsedPoints:   100,  // 違反不變條件：used > earned
        Version:      1,
    }
    err := suite.db.Create(&corruptedModel).Error
    assert.NoError(suite.T(), err)

    // Act: 嘗試載入損壞資料
    accountID, _ := points.AccountIDFromString("acc-999")
    account, err := suite.repo.FindByID(nil, accountID)

    // Assert: 應該返回錯誤，拒絕載入損壞資料
    assert.Error(suite.T(), err)
    assert.Nil(suite.T(), account)
    assert.Contains(suite.T(), err.Error(), "data corruption")
}

// 執行測試套件
func TestRepositoryTestSuite(t *testing.T) {
    suite.Run(t, new(RepositoryTestSuite))
}

// 輔助函數
func pointsAmount(value int) points.PointsAmount {
    amount, _ := points.NewPointsAmount(value)
    return amount
}
```

**執行 Integration Tests**:

```bash
cd internal/infrastructure/persistence/points
go test -v -cover

# 預期結果
=== RUN   TestRepositoryTestSuite
=== RUN   TestRepositoryTestSuite/TestCreate_Success
=== RUN   TestRepositoryTestSuite/TestCreate_Duplicate_ReturnsError
=== RUN   TestRepositoryTestSuite/TestFindByMemberID_Success
=== RUN   TestRepositoryTestSuite/TestUpdate_OptimisticLock_Success
=== RUN   TestRepositoryTestSuite/TestReconstruction_DataCorruption_ReturnsError
--- PASS: TestRepositoryTestSuite (0.05s)
PASS
coverage: 85.0% of statements
```

**檢查點（Week 8 結束）**:

```bash
✅ GORM Models 全部定義
✅ Transaction Context 實作完成
✅ 所有 Repository 實作完成（7 個 Context）
✅ 樂觀鎖機制驗證通過
✅ 資料損壞檢測驗證通過
✅ Integration Tests 覆蓋率 70%+
```

---

### Phase 7: Infrastructure Layer - Adapters + Event Bus（Week 9）

#### External Adapters

**LINE Bot Adapter**（Anti-Corruption Layer）:

```
internal/infrastructure/external/linebot/
├── adapter.go
├── adapter_test.go        # Contract Tests
└── webhook.go
```

**Contract Tests 範例**:

```go
package linebot_test

import (
    "testing"
    "net/http/httptest"
    "github.com/stretchr/testify/assert"
    "github.com/jackyeh168/bar_crm/internal/infrastructure/external/linebot"
)

// TestLineBotAdapter_GetProfile_Contract 測試 LINE API 契約
func TestLineBotAdapter_GetProfile_Contract(t *testing.T) {
    // Arrange: 使用真實的 LINE API 響應範例
    mockResponse := `{
        "userId": "U1234567890",
        "displayName": "Test User",
        "pictureUrl": "https://example.com/avatar.jpg",
        "statusMessage": "Hello World"
    }`

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(mockResponse))
    }))
    defer server.Close()

    adapter := linebot.NewLineUserAdapter(server.URL, "test-token")

    // Act
    member, err := adapter.GetUserProfile("U1234567890")

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, member)
    assert.Equal(t, "U1234567890", member.GetLineUserID().String())
    assert.Equal(t, "Test User", member.GetDisplayName().String())
}

// TestLineBotAdapter_GetProfile_APIChanged 測試 API 變更偵測
func TestLineBotAdapter_GetProfile_APIChanged(t *testing.T) {
    // Arrange: 模擬 LINE API 新增了新字段
    mockResponse := `{
        "userId": "U1234567890",
        "displayName": "Test User",
        "pictureUrl": "https://example.com/avatar.jpg",
        "statusMessage": "Hello World",
        "newField": "some new data"
    }`

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(mockResponse))
    }))
    defer server.Close()

    adapter := linebot.NewLineUserAdapter(server.URL, "test-token")

    // Act & Assert: 適配器應該能夠忽略新字段，向後兼容
    member, err := adapter.GetUserProfile("U1234567890")
    assert.NoError(t, err)
    assert.NotNil(t, member)
}
```

**Event Bus 實作**:

```
internal/infrastructure/messaging/
├── event_bus.go
├── event_bus_test.go
├── subscriber.go
└── publisher.go
```

**檢查點（Week 9 結束）**:

```bash
✅ LINE Bot Adapter 的 Contract Tests 通過
✅ Google OAuth Adapter 實作完成
✅ iChef Excel Parser 實作完成
✅ Event Bus 註冊機制運作正常
✅ Redis Cache（可選）實作完成
```

---

### Phase 8: Presentation Layer（Week 10）

#### HTTP Handlers + Recovery Middleware

**Recovery Middleware**（關鍵！）:

```go
// internal/presentation/http/middleware/recovery.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "net/http"
    "runtime/debug"
)

func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                // 1. 記錄 panic 詳情
                logger.Error("Panic recovered",
                    zap.Any("error", err),
                    zap.String("path", c.Request.URL.Path),
                    zap.String("method", c.Request.Method),
                    zap.String("client_ip", c.ClientIP()),
                    zap.ByteString("stack", debug.Stack()),
                )

                // 2. 發送告警（整合監控系統）
                // TODO: alerter.SendAlert(...)

                // 3. 回傳 500 錯誤
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error":    "Internal server error",
                    "trace_id": c.GetString("trace_id"),
                    "message":  "An unexpected error occurred. Please try again later.",
                })

                // 終止請求處理
                c.Abort()
            }
        }()

        c.Next()
    }
}
```

**LINE Bot Webhook Handlers**:

```
internal/presentation/linebot/
├── webhook_handler.go
├── message_handler.go
└── event_router.go
```

**檢查點（Week 10 結束）**:

```bash
✅ Recovery Middleware 實作並測試
✅ HTTP Handlers 全部完成
✅ LINE Bot Webhook 運作正常
✅ HTTP Server 可以啟動並接受請求
```

---

### Phase 9: E2E Tests + Documentation（Week 11）

#### E2E Tests

```
test/e2e/
├── scan_and_earn_test.go       # 掃描發票獲得積分流程
├── survey_reward_test.go       # 問卷獎勵流程
└── ichef_import_test.go        # iChef 匯入流程
```

**E2E Test 範例**:

```go
package e2e_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "net/http/httptest"
)

// TestScanAndEarnPoints_E2E 端到端測試：掃描發票獲得積分
func TestScanAndEarnPoints_E2E(t *testing.T) {
    // Arrange: 啟動測試 Server
    server := setupTestServer(t)
    defer server.Close()

    // Step 1: 註冊會員
    memberResp := registerMember(t, server, "U1234567890", "Test User")
    assert.NotNil(t, memberResp)

    // Step 2: 綁定手機號碼
    bindResp := bindPhone(t, server, "U1234567890", "0912345678")
    assert.NoError(t, bindResp)

    // Step 3: 掃描發票
    invoiceResp := scanInvoice(t, server, "U1234567890", "AB12345678", "350.00")
    assert.NotNil(t, invoiceResp)
    assert.Equal(t, "pending", invoiceResp.Status)

    // Step 4: iChef 匯入並驗證發票
    batchResp := importIChefBatch(t, server, "ichef_sample.xlsx")
    assert.NotNil(t, batchResp)

    // Step 5: 查詢積分（應該已獲得積分）
    pointsResp := queryPoints(t, server, "U1234567890")
    assert.Equal(t, 3, pointsResp.EarnedPoints)  // 350 / 100 = 3
    assert.Equal(t, 3, pointsResp.AvailablePoints)
}
```

**檢查點（Week 11 結束）**:

```bash
✅ E2E Tests 覆蓋關鍵流程
✅ 整體測試覆蓋率 80%+
✅ 所有測試通過
```

---

### Phase 10: CI/CD Pipeline（Week 12）

#### GitHub Actions Workflow

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Run unit tests (Domain Layer)
        run: go test ./internal/domain/... -v -cover -race

      - name: Run unit tests (Application Layer)
        run: go test ./internal/application/... -v -cover -race

      - name: Run integration tests (Infrastructure Layer)
        run: go test ./internal/infrastructure/... -v -cover

      - name: Run E2E tests
        run: go test ./test/e2e/... -v

      - name: Check overall coverage
        run: |
          go test ./... -coverprofile=coverage.out
          go tool cover -func=coverage.out | grep total | awk '{print $3}'

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest

  build:
    runs-on: ubuntu-latest
    needs: test

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Build application
        run: go build -o bin/app cmd/app/main.go

      - name: Build migration tool
        run: go build -o bin/migrate cmd/migrate/main.go
```

**檢查點（Week 12 結束）**:

```bash
✅ CI/CD Pipeline 運作正常
✅ 所有測試在 CI 中通過
✅ 程式碼品質檢查通過
✅ 專案可以部署
```

---

## 4. 測試策略

### 4.1 測試金字塔

```
        /\
       /E2E\         3% - 端到端測試
      /------\
     /Contract\      5% - 契約測試（外部服務）
    /----------\
   /    Int.    \    15% - 集成測試（真實數據庫）
  /--------------\
 /     Unit      \   77% - 單元測試（快速、隔離）
/------------------\
```

### 4.2 各層測試策略

| 層級 | 測試類型 | Mock 策略 | 覆蓋率目標 | TDD |
|------|---------|----------|-----------|-----|
| **Domain** | 單元測試 | 無 Mock（純邏輯） | 90%+ | ✅ |
| **Application** | 單元測試 | Mock Repositories | 80%+ | ✅ |
| **Infrastructure** | 集成測試 | SQLite in-memory | 70%+ | ❌ |
| **Presentation** | 集成測試 | Mock Use Cases | 70%+ | ❌ |
| **External Adapters** | 契約測試 | Mock 外部 API | 關鍵適配器 | ❌ |
| **E2E** | 端到端測試 | 真實環境 | 關鍵流程 | ❌ |

### 4.3 測試命名規範

```go
// 單元測試
func Test{ServiceName}_{MethodName}_{Scenario}(t *testing.T)

// 範例
func TestPointsAmount_Add_Immutability(t *testing.T)
func TestPointsAccount_EarnPoints_NegativeAmount_ReturnsError(t *testing.T)
func TestReconstructPointsAccount_InvariantViolation_ReturnsError(t *testing.T)
```

### 4.4 測試執行命令

```bash
# 執行所有測試
go test ./...

# 執行特定層測試
go test ./internal/domain/...
go test ./internal/application/...
go test ./internal/infrastructure/...

# 執行測試並生成覆蓋率報告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 執行測試並檢查競態條件
go test ./... -race

# 執行 Integration Tests（帶 tag）
go test ./internal/infrastructure/... -tags=integration -v

# 執行 E2E Tests
go test ./test/e2e/... -v
```

---

## 5. 進度追蹤指標

### 5.1 每週檢查點

| Week | 階段 | 里程碑 | 測試覆蓋率 | 可運行 |
|------|------|--------|-----------|--------|
| 1-2 | Phase 1 | Points Domain 完成 | 90%+ | ✅ 單元測試 |
| 3 | Phase 2 | Member/Invoice Domain 完成 | 85%+ | ✅ 單元測試 |
| 4 | Phase 3 | Survey/External Domain 完成 | 85%+ | ✅ 單元測試 |
| 5 | Phase 4 | Identity/Notification Domain 完成 | 85%+ | ✅ 單元測試 |
| 6-7 | Phase 5 | Application Layer 完成 | 80%+ | ✅ Mock 測試 |
| 8 | Phase 6 | GORM Repositories 完成 | 70%+ | ✅ Integration 測試 |
| 9 | Phase 7 | Infrastructure 完成 | 70%+ | ✅ Contract 測試 |
| 10 | Phase 8 | Presentation 完成 | 75%+ | ✅ HTTP Server 運行 |
| 11 | Phase 9 | E2E Tests 完成 | 整體 80%+ | ✅ 完整流程 |
| 12 | Phase 10 | CI/CD Pipeline 完成 | - | ✅ 自動化部署 |

### 5.2 品質指標

* **程式碼覆蓋率**: 整體 80%+
* **Domain Layer 覆蓋率**: 90%+
* **單元測試執行時間**: < 5 秒
* **Integration 測試執行時間**: < 30 秒
* **E2E 測試執行時間**: < 2 分鐘
* **CI 總執行時間**: < 5 分鐘

### 5.3 程式碼品質檢查

```bash
# golangci-lint 檢查
golangci-lint run ./...

# gofmt 檢查格式
gofmt -d .

# go vet 靜態分析
go vet ./...

# 檢查循環依賴
go mod graph | grep internal
```

---

## 6. 立即行動步驟

### Day 1 任務清單

**Step 1: 初始化專案**

```bash
# 1. 初始化 Go Module
cd /Users/apple/Documents/code/golang/bar_crm
go mod init github.com/jackyeh168/bar_crm

# 2. 安裝測試依賴
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock
go get github.com/google/uuid
go get github.com/shopspring/decimal
```

**Step 2: 建立目錄結構**

```bash
# 建立 Domain Layer 目錄
mkdir -p internal/domain/points
mkdir -p internal/domain/shared

# 建立測試目錄
mkdir -p test/integration
mkdir -p test/e2e
mkdir -p test/fixtures
```

**Step 3: 建立第一個檔案**

```bash
# Shared Domain 概念
touch internal/domain/shared/transaction.go
touch internal/domain/shared/event.go

# Points Context
touch internal/domain/points/errors.go
touch internal/domain/points/value_objects.go
touch internal/domain/points/value_objects_test.go
```

**Step 4: 寫第一個測試（TDD）**

開啟 `internal/domain/points/value_objects_test.go` ，寫第一個測試：

```go
package points_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/jackyeh168/bar_crm/internal/domain/points"
)

func TestNewPointsAmount_ValidValue_ReturnsPointsAmount(t *testing.T) {
    // Arrange
    value := 100

    // Act
    amount, err := points.NewPointsAmount(value)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 100, amount.Value())
}
```

**Step 5: 執行測試（Red）**

```bash
cd internal/domain/points
go test -v

# 預期：測試失敗（尚未實作）
```

**Step 6: 實作（Green）**

開啟 `internal/domain/points/value_objects.go` ，實作：

```go
package points

type PointsAmount struct {
    value int
}

func NewPointsAmount(value int) (PointsAmount, error) {
    if value < 0 {
        return PointsAmount{}, ErrNegativePointsAmount
    }
    return PointsAmount{value: value}, nil
}

func (p PointsAmount) Value() int {
    return p.value
}
```

開啟 `internal/domain/points/errors.go` ：

```go
package points

import "errors"

var ErrNegativePointsAmount = errors.New("points amount cannot be negative")
```

**Step 7: 執行測試（Green）**

```bash
go test -v

# 預期：測試通過
```

**Day 1 目標達成**:
* ✅ 專案結構建立
* ✅ 第一個測試通過
* ✅ TDD 流程驗證

---

## 7. 風險管理

### 7.1 技術風險

| 風險 | 影響 | 機率 | 緩解策略 |
|------|------|------|---------|
| 樂觀鎖實作錯誤 | 高 | 中 | 詳細的 Integration Tests，模擬並發場景 |
| Panic 導致服務中斷 | 高 | 低 | Recovery Middleware + 監控告警 |
| 介面隔離過度複雜 | 中 | 中 | 完善的 DI 配置，清晰的文檔 |
| Repository 效能問題 | 中 | 低 | 索引優化，分頁查詢 |
| LINE Bot 整合失敗 | 高 | 低 | Contract Tests，早期驗證 |

### 7.2 進度風險

| 風險 | 影響 | 機率 | 緩解策略 |
|------|------|------|---------|
| 測試編寫時間超出預期 | 中 | 中 | 優先核心功能，非核心功能可降低覆蓋率 |
| Domain 設計需求變更 | 高 | 低 | 早期與業務確認需求，凍結核心域設計 |
| 團隊成員學習曲線 | 低 | 低 | Code Review，Pair Programming |

### 7.3 品質風險

| 風險 | 影響 | 機率 | 緩解策略 |
|------|------|------|---------|
| 不變條件遺漏 | 高 | 中 | Code Review 檢查清單 |
| 資料損壞未檢測 | 高 | 低 | Reconstruction 驗證，資料庫約束 |
| 記憶體洩漏 | 中 | 低 | Go Profiler 定期檢查 |

---

## 附錄

### A. Code Review 檢查清單

**Domain Layer**:
* [ ] 所有字段私有
* [ ] 通過構造函數創建
* [ ] 狀態變更通過方法（Tell, Don't Ask）
* [ ] 不變性保護（業務規則檢查）
* [ ] 發布領域事件
* [ ] 輕量級設計（無無界集合）
* [ ] 版本控制（樂觀鎖）
* [ ] Reconstruction 驗證資料完整性

**Application Layer**:
* [ ] Use Case 單一職責
* [ ] 使用 Transaction Context
* [ ] 透傳 Domain 錯誤
* [ ] DTOs 實作 Domain 介面
* [ ] Event Handlers 正確註冊

**Infrastructure Layer**:
* [ ] Repository WHERE 條件驗證樂觀鎖
* [ ] GORM Model ↔ Domain Entity 正確轉換
* [ ] 錯誤轉換為 Domain 錯誤
* [ ] Integration Tests 覆蓋關鍵場景

**Presentation Layer**:
* [ ] Recovery Middleware 啟用
* [ ] 錯誤映射為正確的 HTTP 狀態碼
* [ ] 日誌記錄完整
* [ ] 沒有業務邏輯

### B. 參考資料

* [Clean Architecture - Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
* [Domain-Driven Design - Eric Evans](https://www.domainlanguage.com/ddd/)
* [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
* [Effective Go](https://go.dev/doc/effective_go)

---

**最後更新**: 2025-01-11
**維護者**: 開發團隊
**審核者**: 架構師
