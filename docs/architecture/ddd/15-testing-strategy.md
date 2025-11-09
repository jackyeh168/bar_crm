# 測試策略 (Testing Strategy)

> **版本**: 1.0
> **最後更新**: 2025-01-09
> **狀態**: Production Ready

---

## **目錄**

1. [測試金字塔與覆蓋率目標](#1-測試金字塔與覆蓋率目標)
2. [Domain Layer 測試](#2-domain-layer-測試)
3. [Application Layer 測試](#3-application-layer-測試)
4. [Infrastructure Layer 測試](#4-infrastructure-layer-測試)
5. [End-to-End 測試](#5-end-to-end-測試)
6. [測試組織與命名慣例](#6-測試組織與命名慣例)
7. [Mock 策略](#7-mock-策略)
8. [持續整合與測試自動化](#8-持續整合與測試自動化)

---

## **1. 測試金字塔與覆蓋率目標**

### **1.1 測試金字塔**

```
        /\
       /  \      E2E Tests (3%)
      /────\     - 完整用戶流程
     /      \    - Black-box HTTP API 測試
    /────────\   Integration Tests (20%)
   /          \  - Repository 實現
  /────────────\ - Event Handlers
 /              \ - 跨組件協作
/────────────────\ Unit Tests (77%)
                   - Domain 邏輯（純函數，無 Mocks）
                   - Use Cases（Mock Repository）
                   - Value Objects 驗證
```

### **1.2 覆蓋率目標**

| 測試層級 | 覆蓋率目標 | 執行速度 | 測試數量佔比 | 重點 |
|---------|----------|---------|------------|------|
| **Unit Tests** | 90%+ | 毫秒級 | 77% | Domain 邏輯、Use Cases |
| **Integration Tests** | 60%+ | 秒級 | 20% | Repository、外部服務 Adapter |
| **E2E Tests** | N/A | 分鐘級 | 3% | 關鍵業務流程 |

**總體目標**: 80%+ 代碼覆蓋率（透過 `go test -coverprofile=coverage.out`）

### **1.3 各層級測試範圍**

| Layer | 測試內容 | 測試方法 | 依賴 Mocks | 資料庫 |
|-------|---------|---------|-----------|--------|
| **Domain** | 業務邏輯、不變性、狀態轉換 | Pure Unit Tests | ❌ 不需要 | ❌ 不需要 |
| **Application** | Use Case 編排、事務邊界 | Unit Tests with Mocks | ✅ Mock Repo/Service | ❌ 不需要 |
| **Infrastructure** | Repository 實現、SQL 查詢 | Integration Tests | ❌ 不需要 | ✅ SQLite in-memory |
| **Presentation** | HTTP Handlers、錯誤映射 | Unit Tests with Mocks | ✅ Mock Use Cases | ❌ 不需要 |
| **E2E** | 完整用戶流程 | Black-box Tests | ❌ 真實組件 | ✅ Test DB |

---

## **2. Domain Layer 測試**

### **2.1 測試原則**

- ✅ **純單元測試**（無 Mocks、無基礎設施）
- ✅ **測試業務邏輯**（不變性、狀態轉換、業務規則）
- ✅ **AAA 模式**（Arrange-Act-Assert）
- ✅ **高覆蓋率**（90%+，容易達成）

### **2.2 Aggregate 測試範例**

#### **測試不變性保護**

```go
// internal/domain/points/points_account_test.go
package points_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "internal/domain/points"
)

func TestPointsAccount_DeductPoints_InsufficientBalance(t *testing.T) {
    // Arrange
    accountID := points.NewAccountID()
    memberID := points.MemberID("M123")
    account := points.NewPointsAccount(accountID, memberID, points.PointsAmount(100))

    // Act
    err := account.DeductPoints(
        points.PointsAmount(150),
        "Redemption",
        "REF123",
    )

    // Assert
    assert.Error(t, err)
    assert.ErrorIs(t, err, points.ErrInsufficientPoints)

    // 驗證狀態未改變（不變性保護）
    assert.Equal(t, points.PointsAmount(100), account.EarnedPoints())
    assert.Equal(t, points.PointsAmount(0), account.UsedPoints())

    // 驗證未發出事件
    assert.Empty(t, account.GetEvents())
}

func TestPointsAccount_DeductPoints_Success(t *testing.T) {
    // Arrange
    account := points.NewPointsAccount(
        points.NewAccountID(),
        points.MemberID("M123"),
        points.PointsAmount(100),
    )

    // Act
    err := account.DeductPoints(
        points.PointsAmount(50),
        "Redemption",
        "REF123",
    )

    // Assert
    assert.NoError(t, err)

    // 驗證狀態變更
    assert.Equal(t, points.PointsAmount(100), account.EarnedPoints())
    assert.Equal(t, points.PointsAmount(50), account.UsedPoints())
    assert.Equal(t, points.PointsAmount(50), account.GetAvailablePoints())

    // 驗證事件發出
    events := account.GetEvents()
    assert.Len(t, events, 1)

    pointsDeducted, ok := events[0].(points.PointsDeducted)
    assert.True(t, ok)
    assert.Equal(t, points.PointsAmount(50), pointsDeducted.Amount)
    assert.Equal(t, "Redemption", pointsDeducted.Reason)
}
```

#### **測試 Value Object 驗證**

```go
// internal/domain/member/phone_number_test.go
package member_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "internal/domain/member"
)

func TestPhoneNumber_NewPhoneNumber_ValidInput(t *testing.T) {
    testCases := []struct {
        name  string
        input string
    }{
        {"Valid Taiwan Mobile", "0912345678"},
        {"Valid Taiwan Mobile 2", "0987654321"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Act
            phoneNumber, err := member.NewPhoneNumber(tc.input)

            // Assert
            assert.NoError(t, err)
            assert.Equal(t, tc.input, phoneNumber.String())
        })
    }
}

func TestPhoneNumber_NewPhoneNumber_InvalidInput(t *testing.T) {
    testCases := []struct {
        name  string
        input string
    }{
        {"Too short", "091234567"},
        {"Too long", "09123456789"},
        {"Not starting with 09", "0812345678"},
        {"Contains letters", "091234567A"},
        {"Empty string", ""},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Act
            _, err := member.NewPhoneNumber(tc.input)

            // Assert
            assert.Error(t, err)
            assert.ErrorIs(t, err, member.ErrInvalidPhoneNumber)
        })
    }
}
```

#### **測試 Domain Service**

```go
// internal/domain/points/points_calculation_service_test.go
package points_test

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "internal/domain/points"
)

func TestPointsCalculationService_CalculatePoints(t *testing.T) {
    // Arrange
    service := points.NewPointsCalculationService()

    defaultRule := points.NewConversionRule(
        points.ConversionRuleID("RULE001"),
        100, // 100 元 = 1 點
        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
    )

    testCases := []struct {
        name          string
        amount        int
        expectedPoints int
    }{
        {"Exact division", 300, 3},
        {"With remainder (floor)", 250, 2},
        {"Less than conversion rate", 99, 0},
        {"Zero amount", 0, 0},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Act
            points := service.CalculatePoints(
                points.Money(tc.amount),
                defaultRule,
            )

            // Assert
            assert.Equal(t, tc.expectedPoints, points.Value())
        })
    }
}
```

### **2.3 Domain Layer 測試檢查清單**

- [ ] 所有 Aggregate 方法都有測試（成功與失敗場景）
- [ ] 不變性規則被強制執行（狀態不一致時拋出錯誤）
- [ ] Value Objects 驗證邏輯完整（有效與無效輸入）
- [ ] Domain Events 正確發出（事件類型與內容）
- [ ] Domain Services 計算邏輯正確（各種邊界情況）
- [ ] 測試無外部依賴（無 DB、無 HTTP、無 Mocks）

---

## **3. Application Layer 測試**

### **3.1 測試原則**

- ✅ **Use Case 編排邏輯**（不測試 Domain 邏輯）
- ✅ **Mock Repositories 與 Services**
- ✅ **驗證事務邊界**（`InTransaction` 被調用）
- ✅ **驗證錯誤處理**（Domain Error 被正確包裝）

### **3.2 Use Case 測試範例（使用 Testify Suite）**

```go
// internal/application/points/deduct_points_usecase_test.go
package pointsapp_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"
    "go.uber.org/zap"

    "internal/application/points"
    "internal/application/transaction"
    domainPoints "internal/domain/points"
)

type DeductPointsUseCaseTestSuite struct {
    suite.Suite
    useCase     *pointsapp.DeductPointsUseCase
    mockRepo    *MockPointsAccountRepository
    mockTxMgr   *MockTransactionManager
    logger      *zap.Logger
}

func (suite *DeductPointsUseCaseTestSuite) SetupTest() {
    suite.mockRepo = new(MockPointsAccountRepository)
    suite.mockTxMgr = new(MockTransactionManager)
    suite.logger = zap.NewNop()

    suite.useCase = pointsapp.NewDeductPointsUseCase(
        suite.mockRepo,
        suite.mockTxMgr,
        suite.logger,
    )
}

func (suite *DeductPointsUseCaseTestSuite) TestExecute_Success() {
    // Arrange
    accountID := domainPoints.AccountID("ACC123")
    memberID := domainPoints.MemberID("M456")
    account := domainPoints.NewPointsAccount(accountID, memberID, domainPoints.PointsAmount(100))

    cmd := pointsapp.DeductPointsCommand{
        AccountID:   accountID,
        Amount:      domainPoints.PointsAmount(50),
        Reason:      "Redemption",
        ReferenceID: "REF123",
    }

    // Mock: FindByID returns account
    suite.mockRepo.On("FindByID", mock.Anything, accountID).
        Return(account, nil)

    // Mock: Update succeeds
    suite.mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*points.PointsAccount")).
        Return(nil)

    // Mock: InTransaction executes function
    suite.mockTxMgr.On("InTransaction", mock.AnythingOfType("func(transaction.TransactionContext) error")).
        Run(func(args mock.Arguments) {
            fn := args.Get(0).(func(transaction.TransactionContext) error)
            fn(&mockTxContext{}) // Execute function
        }).
        Return(nil)

    // Act
    err := suite.useCase.Execute(cmd)

    // Assert
    assert.NoError(suite.T(), err)
    suite.mockRepo.AssertExpectations(suite.T())
    suite.mockTxMgr.AssertExpectations(suite.T())
}

func (suite *DeductPointsUseCaseTestSuite) TestExecute_InsufficientPoints() {
    // Arrange
    accountID := domainPoints.AccountID("ACC123")
    memberID := domainPoints.MemberID("M456")
    account := domainPoints.NewPointsAccount(accountID, memberID, domainPoints.PointsAmount(30))

    cmd := pointsapp.DeductPointsCommand{
        AccountID:   accountID,
        Amount:      domainPoints.PointsAmount(50),
        Reason:      "Redemption",
        ReferenceID: "REF123",
    }

    suite.mockRepo.On("FindByID", mock.Anything, accountID).
        Return(account, nil)

    suite.mockTxMgr.On("InTransaction", mock.AnythingOfType("func(transaction.TransactionContext) error")).
        Run(func(args mock.Arguments) {
            fn := args.Get(0).(func(transaction.TransactionContext) error)
            fn(&mockTxContext{})
        }).
        Return(domainPoints.ErrInsufficientPoints)

    // Act
    err := suite.useCase.Execute(cmd)

    // Assert
    assert.Error(suite.T(), err)
    assert.ErrorIs(suite.T(), err, domainPoints.ErrInsufficientPoints)

    // Verify Update was NOT called (transaction rolled back)
    suite.mockRepo.AssertNotCalled(suite.T(), "Update")
}

func (suite *DeductPointsUseCaseTestSuite) TestExecute_AccountNotFound() {
    // Arrange
    accountID := domainPoints.AccountID("ACC123")
    cmd := pointsapp.DeductPointsCommand{
        AccountID:   accountID,
        Amount:      domainPoints.PointsAmount(50),
        Reason:      "Redemption",
        ReferenceID: "REF123",
    }

    suite.mockRepo.On("FindByID", mock.Anything, accountID).
        Return(nil, domainPoints.ErrAccountNotFound)

    suite.mockTxMgr.On("InTransaction", mock.AnythingOfType("func(transaction.TransactionContext) error")).
        Run(func(args mock.Arguments) {
            fn := args.Get(0).(func(transaction.TransactionContext) error)
            fn(&mockTxContext{})
        }).
        Return(domainPoints.ErrAccountNotFound)

    // Act
    err := suite.useCase.Execute(cmd)

    // Assert
    assert.Error(suite.T(), err)
    assert.ErrorIs(suite.T(), err, domainPoints.ErrAccountNotFound)
}

func TestDeductPointsUseCaseTestSuite(t *testing.T) {
    suite.Run(t, new(DeductPointsUseCaseTestSuite))
}

// ===== Mock 實現 =====

type MockPointsAccountRepository struct {
    mock.Mock
}

func (m *MockPointsAccountRepository) FindByID(
    ctx transaction.TransactionContext,
    accountID domainPoints.AccountID,
) (*domainPoints.PointsAccount, error) {
    args := m.Called(ctx, accountID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domainPoints.PointsAccount), args.Error(1)
}

func (m *MockPointsAccountRepository) Update(
    ctx transaction.TransactionContext,
    account *domainPoints.PointsAccount,
) error {
    args := m.Called(ctx, account)
    return args.Error(0)
}

type MockTransactionManager struct {
    mock.Mock
}

func (m *MockTransactionManager) InTransaction(
    fn func(ctx transaction.TransactionContext) error,
) error {
    args := m.Called(fn)
    return args.Error(0)
}

type mockTxContext struct{}

func (ctx *mockTxContext) AddEvent(event domain.DomainEvent) {}
func (ctx *mockTxContext) GetEvents() []domain.DomainEvent  { return nil }
```

### **3.3 Event Handler 測試範例**

```go
// internal/application/event/points_earned_handler_test.go
package event_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "go.uber.org/zap"

    "internal/application/event"
    "internal/domain/points"
)

func TestPointsEarnedHandler_Handle_Success(t *testing.T) {
    // Arrange
    mockNotificationService := new(MockNotificationService)
    mockCache := new(MockCache)
    logger := zap.NewNop()

    handler := event.NewPointsEarnedHandler(
        mockNotificationService,
        mockCache,
        logger,
    )

    event := points.NewPointsEarned(
        points.AccountID("ACC123"),
        points.MemberID("M456"),
        points.PointsAmount(10),
        points.SourceInvoice,
        "INV789",
        "發票驗證",
    )

    // Mock: Event not yet processed
    mockCache.On("Exists", mock.MatchedBy(func(key string) bool {
        return key == "event:processed:"+event.EventID()
    })).Return(false)

    // Mock: Send notification succeeds
    mockNotificationService.On("SendLineMessage",
        mock.Anything,
        "M456",
        mock.MatchedBy(func(msg string) bool {
            return msg != ""
        }),
    ).Return(nil)

    // Mock: Mark as processed
    mockCache.On("Set",
        mock.MatchedBy(func(key string) bool {
            return key == "event:processed:"+event.EventID()
        }),
        true,
        24*time.Hour,
    ).Return(nil)

    // Act
    err := handler.Handle(context.Background(), event)

    // Assert
    assert.NoError(t, err)
    mockNotificationService.AssertExpectations(t)
    mockCache.AssertExpectations(t)
}

func TestPointsEarnedHandler_Handle_AlreadyProcessed(t *testing.T) {
    // Arrange
    mockNotificationService := new(MockNotificationService)
    mockCache := new(MockCache)
    logger := zap.NewNop()

    handler := event.NewPointsEarnedHandler(
        mockNotificationService,
        mockCache,
        logger,
    )

    event := points.NewPointsEarned(
        points.AccountID("ACC123"),
        points.MemberID("M456"),
        points.PointsAmount(10),
        points.SourceInvoice,
        "INV789",
        "發票驗證",
    )

    // Mock: Event already processed
    mockCache.On("Exists", mock.Anything).Return(true)

    // Act
    err := handler.Handle(context.Background(), event)

    // Assert
    assert.NoError(t, err)

    // Verify notification NOT sent (idempotency)
    mockNotificationService.AssertNotCalled(t, "SendLineMessage")
}

// ===== Mocks =====

type MockNotificationService struct {
    mock.Mock
}

func (m *MockNotificationService) SendLineMessage(
    ctx context.Context,
    memberID string,
    message string,
) error {
    args := m.Called(ctx, memberID, message)
    return args.Error(0)
}

type MockCache struct {
    mock.Mock
}

func (m *MockCache) Exists(key string) bool {
    args := m.Called(key)
    return args.Bool(0)
}

func (m *MockCache) Set(key string, value interface{}, ttl time.Duration) error {
    args := m.Called(key, value, ttl)
    return args.Error(0)
}
```

### **3.4 Application Layer 測試檢查清單**

- [ ] 所有 Use Cases 都有測試（成功與失敗場景）
- [ ] Repository 調用被驗證（FindByID, Update）
- [ ] TransactionManager.InTransaction 被調用
- [ ] Domain Errors 被正確傳播
- [ ] Event Handlers 驗證冪等性（重複處理返回成功）
- [ ] Mocks 使用 `AssertExpectations()` 驗證調用

---

## **4. Infrastructure Layer 測試**

### **4.1 測試原則**

- ✅ **真實資料庫測試**（SQLite in-memory）
- ✅ **測試 SQL 查詢邏輯**（GORM 映射、索引、JOIN）
- ✅ **測試樂觀鎖**（併發更新）
- ✅ **測試唯一性約束**

### **4.2 Repository Integration 測試範例**

```go
// internal/infrastructure/persistence/gorm_points_repository_test.go
package persistence_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "internal/domain/points"
    "internal/infrastructure/persistence"
)

type GormPointsAccountRepositoryTestSuite struct {
    suite.Suite
    db   *gorm.DB
    repo *persistence.GormPointsAccountRepository
}

func (suite *GormPointsAccountRepositoryTestSuite) SetupTest() {
    // 使用 SQLite in-memory 資料庫
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    assert.NoError(suite.T(), err)

    // Auto-migrate
    err = db.AutoMigrate(&persistence.PointsAccountModel{})
    assert.NoError(suite.T(), err)

    suite.db = db
    suite.repo = persistence.NewGormPointsAccountRepository(db)
}

func (suite *GormPointsAccountRepositoryTestSuite) TearDownTest() {
    sqlDB, _ := suite.db.DB()
    sqlDB.Close()
}

func (suite *GormPointsAccountRepositoryTestSuite) TestCreate_Success() {
    // Arrange
    account := points.NewPointsAccount(
        points.AccountID("ACC123"),
        points.MemberID("M456"),
        points.PointsAmount(100),
    )

    // Act
    err := suite.repo.Create(nil, account)

    // Assert
    assert.NoError(suite.T(), err)

    // Verify persisted to database
    var model persistence.PointsAccountModel
    result := suite.db.Where("account_id = ?", "ACC123").First(&model)
    assert.NoError(suite.T(), result.Error)
    assert.Equal(suite.T(), "M456", model.MemberID)
    assert.Equal(suite.T(), 100, model.EarnedPoints)
}

func (suite *GormPointsAccountRepositoryTestSuite) TestFindByID_NotFound() {
    // Act
    account, err := suite.repo.FindByID(nil, points.AccountID("NONEXISTENT"))

    // Assert
    assert.Error(suite.T(), err)
    assert.ErrorIs(suite.T(), err, points.ErrAccountNotFound)
    assert.Nil(suite.T(), account)
}

func (suite *GormPointsAccountRepositoryTestSuite) TestUpdate_OptimisticLocking() {
    // Arrange: Create account
    account := points.NewPointsAccount(
        points.AccountID("ACC123"),
        points.MemberID("M456"),
        points.PointsAmount(100),
    )
    suite.repo.Create(nil, account)

    // Act: Two concurrent updates
    account1, _ := suite.repo.FindByID(nil, points.AccountID("ACC123")) // Version = 1
    account2, _ := suite.repo.FindByID(nil, points.AccountID("ACC123")) // Version = 1

    account1.EarnPoints(points.PointsAmount(10), points.SourceInvoice, "INV1", "Test")
    err1 := suite.repo.Update(nil, account1) // Version 1 → 2 (succeeds)

    account2.EarnPoints(points.PointsAmount(20), points.SourceInvoice, "INV2", "Test")
    err2 := suite.repo.Update(nil, account2) // Version 1 → 2 (fails)

    // Assert
    assert.NoError(suite.T(), err1)
    assert.Error(suite.T(), err2)
    assert.ErrorIs(suite.T(), err2, points.ErrOptimisticLockFailure)

    // Verify final state (only first update applied)
    finalAccount, _ := suite.repo.FindByID(nil, points.AccountID("ACC123"))
    assert.Equal(suite.T(), points.PointsAmount(110), finalAccount.EarnedPoints())
}

func TestGormPointsAccountRepositoryTestSuite(t *testing.T) {
    suite.Run(t, new(GormPointsAccountRepositoryTestSuite))
}
```

### **4.3 External Adapter 測試（LINE SDK）**

```go
// internal/infrastructure/line/line_user_adapter_test.go
package line_test

import (
    "testing"

    "github.com/line/line-bot-sdk-go/linebot"
    "github.com/stretchr/testify/assert"

    "internal/infrastructure/line"
)

func TestLineUserAdapter_GetUserProfile(t *testing.T) {
    // 這需要真實的 LINE Bot Token 或 Mock HTTP Server
    // 建議使用 httptest 模擬 LINE API

    // Arrange
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock LINE API response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{
            "userId": "U123",
            "displayName": "小陳",
            "pictureUrl": "https://example.com/pic.jpg"
        }`))
    }))
    defer server.Close()

    client, _ := linebot.New("test-secret", "test-token")
    adapter := line.NewLineUserAdapter(client, server.URL)

    // Act
    profile, err := adapter.GetUserProfile("U123")

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "小陳", profile.DisplayName)
}
```

### **4.4 Infrastructure Layer 測試檢查清單**

- [ ] Repository 的 CRUD 操作都有測試
- [ ] 樂觀鎖測試（併發更新）
- [ ] 唯一性約束測試（重複插入）
- [ ] NULL 值處理測試
- [ ] 外部服務 Adapter 使用 Mock HTTP Server

---

## **5. End-to-End 測試**

### **5.1 測試原則**

- ✅ **Black-box 測試**（透過 HTTP API）
- ✅ **完整用戶流程**（註冊 → 掃描發票 → 賺取積分）
- ✅ **使用測試資料庫**（隔離環境）
- ✅ **僅測試關鍵路徑**（Happy Path + 主要錯誤場景）

### **5.2 E2E 測試範例**

```go
// test/e2e/invoice_scan_test.go
package e2e_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "internal/cmd/app"
)

func TestE2E_ScanInvoiceAndEarnPoints(t *testing.T) {
    // Setup: Start HTTP server with test database
    testDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    testDB.AutoMigrate(&persistence.MemberModel{}, &persistence.PointsAccountModel{})

    server := app.NewTestServer(testDB)
    defer server.Close()

    // Step 1: Register member
    registerReq := map[string]interface{}{
        "line_user_id":  "U123",
        "phone_number":  "0912345678",
        "display_name":  "小陳",
    }
    registerResp := httpPost(t, server.URL+"/api/members/register", registerReq)
    assert.Equal(t, http.StatusOK, registerResp.StatusCode)

    var registerResult map[string]interface{}
    json.NewDecoder(registerResp.Body).Decode(&registerResult)
    memberID := registerResult["member_id"].(string)

    // Step 2: Scan invoice
    scanReq := map[string]interface{}{
        "member_id":   memberID,
        "qr_code_data": "AB12345678|1140108|250|...",
    }
    scanResp := httpPost(t, server.URL+"/api/invoices/scan", scanReq)
    assert.Equal(t, http.StatusOK, scanResp.StatusCode)

    var scanResult map[string]interface{}
    json.NewDecoder(scanResp.Body).Decode(&scanResult)
    transactionID := scanResult["transaction_id"].(string)
    assert.NotEmpty(t, transactionID)

    // Step 3: Verify transaction created with "pending" status
    txResp := httpGet(t, server.URL+"/api/transactions/"+transactionID)
    assert.Equal(t, http.StatusOK, txResp.StatusCode)

    var txResult map[string]interface{}
    json.NewDecoder(txResp.Body).Decode(&txResult)
    assert.Equal(t, "pending", txResult["status"])

    // Step 4: Simulate iChef import (admin operation)
    importReq := map[string]interface{}{
        "invoices": []map[string]interface{}{
            {
                "invoice_number": "AB12345678",
                "invoice_date":   "2025-01-08",
                "amount":         250,
            },
        },
    }
    importResp := httpPost(t, server.URL+"/api/admin/import", importReq)
    assert.Equal(t, http.StatusOK, importResp.StatusCode)

    // Step 5: Verify points earned (250 / 100 = 2 points)
    pointsResp := httpGet(t, server.URL+"/api/members/"+memberID+"/points")
    assert.Equal(t, http.StatusOK, pointsResp.StatusCode)

    var pointsResult map[string]interface{}
    json.NewDecoder(pointsResp.Body).Decode(&pointsResult)
    assert.Equal(t, 2, int(pointsResult["earned_points"].(float64)))
    assert.Equal(t, 0, int(pointsResult["used_points"].(float64)))
}

// Helper functions
func httpPost(t *testing.T, url string, body interface{}) *http.Response {
    jsonBody, _ := json.Marshal(body)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    assert.NoError(t, err)
    return resp
}

func httpGet(t *testing.T, url string) *http.Response {
    resp, err := http.Get(url)
    assert.NoError(t, err)
    return resp
}
```

### **5.3 E2E 測試檢查清單**

- [ ] 關鍵用戶流程有 E2E 測試（Happy Path）
- [ ] 主要錯誤場景有測試（如發票已存在）
- [ ] 使用測試資料庫（不污染開發環境）
- [ ] 測試可獨立執行（不依賴其他測試順序）

---

## **6. 測試組織與命名慣例**

### **6.1 目錄結構**

```
bar_crm/
├── internal/
│   ├── domain/
│   │   ├── points/
│   │   │   ├── points_account.go
│   │   │   └── points_account_test.go       # Domain 單元測試
│   ├── application/
│   │   ├── points/
│   │   │   ├── earn_points_usecase.go
│   │   │   └── earn_points_usecase_test.go  # Application 單元測試
│   └── infrastructure/
│       ├── persistence/
│       │   ├── gorm_points_repository.go
│       │   └── gorm_points_repository_test.go # Infrastructure 整合測試
├── test/
│   ├── integration/
│   │   └── points_workflow_test.go          # 跨 Repository 測試
│   ├── e2e/
│   │   └── invoice_scan_test.go             # HTTP 層級測試
│   └── testdata/
│       └── ichef_batch.xlsx                 # 測試資料
└── Makefile
```

### **6.2 測試命名慣例**

#### **單元測試命名**

```go
// 格式: Test{StructName}_{MethodName}_{Scenario}
func TestPointsAccount_DeductPoints_Success(t *testing.T)
func TestPointsAccount_DeductPoints_InsufficientBalance(t *testing.T)
func TestPhoneNumber_NewPhoneNumber_InvalidFormat(t *testing.T)
```

#### **TestSuite 命名**

```go
// 格式: {ServiceName}TestSuite
type DeductPointsUseCaseTestSuite struct {
    suite.Suite
    // ...
}

// 格式: Test{MethodName}_{Scenario}
func (suite *DeductPointsUseCaseTestSuite) TestExecute_Success()
func (suite *DeductPointsUseCaseTestSuite) TestExecute_AccountNotFound()
```

#### **整合測試命名**

```go
func TestGormPointsAccountRepository_Create_Success(t *testing.T)
func TestGormPointsAccountRepository_Update_OptimisticLocking(t *testing.T)
```

#### **E2E 測試命名**

```go
func TestE2E_ScanInvoiceAndEarnPoints(t *testing.T)
func TestE2E_RegisterMemberAndDeductPoints(t *testing.T)
```

---

## **7. Mock 策略**

### **7.1 Mock 使用場景**

| Layer | 需要 Mock | Mock 對象 | 工具 |
|-------|---------|----------|------|
| **Domain** | ❌ 不需要 | N/A | N/A |
| **Application** | ✅ 需要 | Repository, Service, TransactionManager | Testify Mock |
| **Infrastructure** | ❌ 不需要 | N/A（使用真實資料庫） | N/A |
| **Presentation** | ✅ 需要 | Use Cases | Testify Mock |

### **7.2 Testify Mock 範例**

```go
// Mock Repository
type MockPointsAccountRepository struct {
    mock.Mock
}

func (m *MockPointsAccountRepository) FindByID(
    ctx transaction.TransactionContext,
    accountID points.AccountID,
) (*points.PointsAccount, error) {
    args := m.Called(ctx, accountID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*points.PointsAccount), args.Error(1)
}

func (m *MockPointsAccountRepository) Update(
    ctx transaction.TransactionContext,
    account *points.PointsAccount,
) error {
    args := m.Called(ctx, account)
    return args.Error(0)
}

// 使用 Mock
func TestUseCase(t *testing.T) {
    mockRepo := new(MockPointsAccountRepository)

    // Setup expectations
    mockRepo.On("FindByID", mock.Anything, mock.Anything).
        Return(testAccount, nil)

    mockRepo.On("Update", mock.Anything, mock.Anything).
        Return(nil)

    // Execute test
    useCase := NewUseCase(mockRepo)
    err := useCase.Execute(cmd)

    // Verify expectations
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### **7.3 Fake 實現（Integration Tests）**

```go
// Fake Repository (in-memory)
type InMemoryPointsAccountRepository struct {
    accounts map[points.AccountID]*points.PointsAccount
    mu       sync.RWMutex
}

func (r *InMemoryPointsAccountRepository) FindByID(
    ctx transaction.TransactionContext,
    accountID points.AccountID,
) (*points.PointsAccount, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    account, exists := r.accounts[accountID]
    if !exists {
        return nil, points.ErrAccountNotFound
    }

    return account, nil
}

func (r *InMemoryPointsAccountRepository) Create(
    ctx transaction.TransactionContext,
    account *points.PointsAccount,
) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.accounts[account.AccountID()] = account
    return nil
}
```

---

## **8. 持續整合與測試自動化**

### **8.1 Makefile 測試指令**

```makefile
# Run all tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run only unit tests (exclude integration)
test-unit:
	go test ./internal/domain/... ./internal/application/... -v

# Run only integration tests
test-integration:
	go test ./internal/infrastructure/... -v

# Run E2E tests
test-e2e:
	go test ./test/e2e/... -v

# Run with race detection
test-race:
	go test ./... -race

# Run benchmarks
benchmark:
	go test ./... -bench=. -benchmem
```

### **8.2 GitHub Actions CI 配置**

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run Unit Tests
        run: make test-unit

      - name: Run Integration Tests
        run: make test-integration

      - name: Run E2E Tests
        run: make test-e2e

      - name: Generate Coverage Report
        run: make test-coverage

      - name: Upload Coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

### **8.3 測試執行策略**

| 環境 | 執行頻率 | 測試範圍 | 執行時間 |
|------|---------|---------|---------|
| **本地開發** | 每次 `git commit` | Unit Tests | < 5 秒 |
| **Pre-commit Hook** | 每次提交前 | Unit Tests + Linting | < 10 秒 |
| **CI (Pull Request)** | 每次 PR | Unit + Integration + E2E | < 3 分鐘 |
| **CD (Deployment)** | 部署前 | All Tests + Benchmarks | < 5 分鐘 |

---

## **總結**

### **測試策略關鍵原則**

1. **Domain Layer**: 純單元測試，無 Mocks，高覆蓋率（90%+）
2. **Application Layer**: Mock Repositories，測試編排邏輯
3. **Infrastructure Layer**: 真實資料庫（SQLite in-memory）
4. **E2E Tests**: Black-box HTTP 測試，僅測關鍵路徑
5. **快速反饋循環**: Unit Tests < 5 秒，Integration Tests < 30 秒

### **檢查清單**

- [ ] 所有 Domain Aggregates 有單元測試（成功與失敗場景）
- [ ] 所有 Use Cases 有測試（Mock Repositories）
- [ ] Repository 實現有整合測試（SQLite in-memory）
- [ ] 關鍵業務流程有 E2E 測試
- [ ] 測試覆蓋率 > 80%（透過 CI 驗證）
- [ ] 測試命名遵循慣例（`Test{Struct}_{Method}_{Scenario}`）
- [ ] Pre-commit Hook 執行 Unit Tests
- [ ] CI Pipeline 執行所有測試

---

**相關文檔**:
- `/docs/architecture/ddd/07-aggregate-design-principles.md` - Aggregate 設計原則
- `/docs/architecture/ddd/11-dependency-rules.md` - 依賴規則
- `/docs/qa/testing-conventions.md` - 測試慣例
- ADR-005: Transaction Context Pattern - 如何測試事務
