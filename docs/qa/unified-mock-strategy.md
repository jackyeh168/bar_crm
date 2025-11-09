# 🎯 統一Mock策略文檔

**日期**: 2025-09-05  
**版本**: 1.0.0  
**狀態**: ✅ 已完成實施

## 📋 總覽

統一Mock策略是一個完整的測試框架，旨在標準化和統一整個程式碼庫中的Mock使用方式。該系統提供了三種不同的Mock策略，支援多種測試場景，並提供了簡潔的API供開發者使用。

## 🏗️ 核心架構

### 1. 統一接口設計

為了避免循環依賴，創建了獨立的接口包：

```go
// internal/interfaces/service_interfaces.go
package interfaces

// RegistrationServiceInterface 定義用戶註冊服務接口
type RegistrationServiceInterface interface {
    RegisterUser(lineUserID string) (*RegistrationResult, error)
    RegisterUserWithPhone(lineUserID, phoneNumber string) (*RegistrationResult, error)
    CheckUserRegistration(lineUserID string) (*RegistrationResult, error)
    ValidatePhoneNumber(phoneNumber string) error
}

// LineBotServiceInterface 定義LINE Bot服務接口
type LineBotServiceInterface interface {
    HandleMessage(userID, message string) (string, error)
    HandleMemberJoined(userID string) (string, error)
    HandleMemberLeft(userID string) error
    HandlePostback(userID, data string) (string, error)
    HandleGroupJoined(groupID string) (string, error)
    HandleGroupLeft(groupID string) error
}
```

### 2. 三種Mock策略

```go
// MockStrategy 定義Mock實施策略
type MockStrategy string

const (
    // StrategyInterface 使用接口化Mock，預定義行為
    StrategyInterface MockStrategy = "interface"
    
    // StrategyTestify 使用testify/mock，完整Mock驗證和期望
    StrategyTestify MockStrategy = "testify"
    
    // StrategyHybrid 結合接口Mock和testify驗證能力
    StrategyHybrid MockStrategy = "hybrid"
)
```

### 3. MockFactory - 統一Mock創建工廠

```go
// MockFactory 提供統一的Mock創建，支援可配置策略
type MockFactory struct {
    strategy        MockStrategy
    dataFactory     *TestDataFactory
    defaultBehavior MockBehavior
    mu              sync.RWMutex
}

// MockBehavior 定義Mock的預設行為
type MockBehavior struct {
    EnableAutoSuccess   bool          // 自動返回成功結果
    DefaultDelay        time.Duration // 預設延遲
    ErrorRate           float64       // 錯誤率 (0.0-1.0)
    EnableDetailedLogs  bool          // 啟用詳細日志
}
```

## 🎨 使用方式

### 基本使用

```go
// 1. 快速創建Mock集合
mocks := testutil.CreateQuickMocks()

// 2. 使用Mock進行測試
result, err := mocks.RegistrationService.RegisterUser("test_user_123")
assert.NoError(t, err)
assert.True(t, result.IsNewUser)

response, err := mocks.LineBotService.HandleMessage("test_user", "hello")
assert.NoError(t, err)
assert.NotEmpty(t, response)
```

### 高級使用 - 策略選擇

```go
// Interface策略：簡單快速，預定義行為
factory := testutil.GetMockFactory(testutil.StrategyInterface)
mockSet := factory.CreateSuccessScenarioMocks()

// Testify策略：完整驗證，適合複雜測試
factory := testutil.GetMockFactory(testutil.StrategyTestify)
mockSet := factory.CreateSuccessScenarioMocks()

// 自定義行為
customBehavior := testutil.MockBehavior{
    EnableAutoSuccess:  false,
    ErrorRate:          0.2,  // 20% 錯誤率
    DefaultDelay:       50 * time.Millisecond,
    EnableDetailedLogs: true,
}
mockSet := testutil.CreateCustomMocks(customBehavior)
```

### 場景化Mock創建

```go
// 成功場景Mock
successMocks := factory.CreateSuccessScenarioMocks()

// 錯誤場景Mock
errorMocks := factory.CreateErrorScenarioMocks()

// 效能測試Mock（有延遲）
perfMocks := factory.CreatePerformanceTestMocks(100 * time.Millisecond)
```

### Testify期望設置

```go
// 使用期望構建器設置複雜期望
builder := testutil.NewMockExpectationsBuilder(mockSet)

// 設置成功用戶註冊流程
builder.ExpectUserRegistrationSuccess("test_user_123").
    ExpectPhoneNumberValidation("0912345678", true).
    ExpectLineBotMessageHandling("test_user_123", "hello", "Welcome!")

// 設置錯誤流程
builder.ExpectUserRegistrationFailure("error_user", "service unavailable").
    ExpectLineBotError("error_user", "hello", "service error")
```

## 📊 功能特色

### 1. 併發安全

所有Mock實現都是線程安全的，支援併發測試：

```go
func TestConcurrentMockUsage(t *testing.T) {
    mockSet := testutil.CreateQuickMocks()
    
    const goroutines = 10
    const operationsPerGoroutine = 20
    
    for i := 0; i < goroutines; i++ {
        go func(id int) {
            for j := 0; j < operationsPerGoroutine; j++ {
                userID := fmt.Sprintf("user_%d_%d", id, j)
                result, err := mockSet.RegistrationService.RegisterUser(userID)
                assert.NoError(t, err)
                assert.NotNil(t, result)
            }
        }(i)
    }
}
```

### 2. 狀態追蹤

Interface策略Mock提供完整的狀態追蹤：

```go
// 獲取方法調用次數
mockService := mockSet.RegistrationService.(*testutil.MockRegistrationServiceInterface)
callCount := mockService.GetCallCount("RegisterUser")

// 獲取儲存的用戶資料
mockRepo := mockSet.UserRepository.(*testutil.MockUserRepositoryInterface)
storedUser := mockRepo.GetStoredUser("test_user")
```

### 3. 測試資料整合

Mock系統與TestDataFactory完全整合：

```go
factory := testutil.NewMockFactory(testutil.StrategyInterface)
mockSet := factory.CreateSuccessScenarioMocks()

// Mock自動使用唯一的測試資料
result, err := mockSet.RegistrationService.RegisterUser("test_user")
// result.User 包含唯一生成的用戶資料
```

### 4. 向後兼容性

提供遷移輔助工具，支援從舊Mock平滑升級：

```go
// 遷移輔助器
helper := testutil.NewMockMigrationHelper()

// 創建相容舊測試的Mock
lineBotService, registrationService := helper.CreateTestifyCompatibleMocks()

// 或創建統一Mock集合
mockSet := helper.CreateUnifiedMockSet()
```

## 🔧 配置選項

### MockBehavior 詳細配置

```go
behavior := testutil.MockBehavior{
    // 自動成功模式：大多數操作返回成功結果
    EnableAutoSuccess: true,
    
    // 預設延遲：模擬網路延遲或處理時間
    DefaultDelay: 10 * time.Millisecond,
    
    // 錯誤率：隨機錯誤的概率（0.0-1.0）
    ErrorRate: 0.1, // 10% 錯誤率
    
    // 詳細日志：啟用詳細的操作日志記錄
    EnableDetailedLogs: false,
}
```

### 錯誤模擬

```go
// 創建高錯誤率Mock，用於測試錯誤處理
errorBehavior := testutil.MockBehavior{
    EnableAutoSuccess: false,
    ErrorRate:         0.8, // 80% 錯誤率
}

mockSet := testutil.CreateCustomMocks(errorBehavior)
```

## 🌟 最佳實踐

### 1. 策略選擇指南

| 場景 | 推薦策略 | 原因 |
|------|----------|------|
| **單元測試** | Interface | 快速設置，預定義行為，易於理解 |
| **集成測試** | Interface | 狀態追蹤，真實的內存存儲模擬 |
| **Contract測試** | Testify | 完整的期望驗證，確保契約遵循 |
| **效能測試** | Interface | 可配置延遲，錯誤率模擬 |
| **複雜交互測試** | Hybrid | 結合兩者優勢 |

### 2. 測試組織

```go
func TestUserRegistrationFlow(t *testing.T) {
    // 使用場景化Mock
    mockSet := testutil.CreateQuickMocks()
    
    t.Run("New User Registration", func(t *testing.T) {
        result, err := mockSet.RegistrationService.RegisterUser("new_user")
        assert.NoError(t, err)
        assert.True(t, result.IsNewUser)
    })
    
    t.Run("Existing User Check", func(t *testing.T) {
        // Mock會自動追蹤狀態
        result, err := mockSet.RegistrationService.CheckUserRegistration("new_user")
        assert.NoError(t, err)
        assert.False(t, result.IsNewUser) // 因為上面已註冊
    })
}
```

### 3. 錯誤場景測試

```go
func TestErrorHandling(t *testing.T) {
    // 創建錯誤場景Mock
    errorMocks := testutil.GetMockFactory(testutil.StrategyInterface).CreateErrorScenarioMocks()
    
    // 測試錯誤處理邏輯
    _, err := errorMocks.RegistrationService.RegisterUser("error_test")
    assert.Error(t, err)
    
    // 測試重試邏輯等
}
```

### 4. 效能測試

```go
func TestPerformanceWithLatency(t *testing.T) {
    // 創建有延遲的Mock，模擬真實環境
    perfMocks := testutil.GetMockFactory(testutil.StrategyInterface).
        CreatePerformanceTestMocks(100 * time.Millisecond)
    
    start := time.Now()
    _, err := perfMocks.RegistrationService.RegisterUser("perf_test")
    duration := time.Since(start)
    
    assert.NoError(t, err)
    assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
}
```

## 🚀 遷移指南

### 從舊Mock遷移

#### 步驟1：評估現有Mock

```go
// 舊方式：手動創建testify mock
oldMock := &testutil.MockRegistrationService{}
oldMock.On("RegisterUser", "test").Return(result, nil)

// 新方式：使用統一工廠
mockSet := testutil.CreateQuickMocks()
result, err := mockSet.RegistrationService.RegisterUser("test")
```

#### 步驟2：逐步遷移

```go
// 使用遷移輔助器保持兼容性
helper := testutil.NewMockMigrationHelper()
lineBotMock, registrationMock := helper.CreateTestifyCompatibleMocks()

// 這些Mock包含預設的成功行為，減少遷移工作量
```

#### 步驟3：完整遷移

```go
// 最終目標：統一Mock使用
func TestWithUnifiedMocks(t *testing.T) {
    mockSet := testutil.CreateQuickMocks()
    
    // 所有Mock都通過mockSet訪問
    // 自動享受狀態追蹤、併發安全等特性
}
```

## 📈 效益總結

### 統一性改善
- **標準化接口**: 所有Mock使用相同的創建和配置方式
- **一致性行為**: 相同的預設行為和錯誤處理模式
- **統一測試資料**: 與TestDataFactory無縫整合，確保測試資料唯一性

### 開發效率提升
- **快速設置**: `CreateQuickMocks()` 一行代碼創建完整Mock集合
- **場景模板**: 預定義的成功、錯誤、效能測試場景
- **智能預設**: 自動配置合理的預設行為，減少樣板代碼

### 測試品質提升
- **併發安全**: 所有Mock支援並發測試，避免競爭條件
- **狀態追蹤**: 完整的呼叫統計和狀態管理
- **錯誤模擬**: 可配置的錯誤率，測試錯誤處理邏輯

### 維護性改善
- **向後兼容**: 遷移輔助工具確保平滑升級
- **接口隔離**: 獨立的interfaces包避免循環依賴
- **文檔完整**: 全面的API文檔和使用範例

## 🔮 未來擴展

### 計劃中的功能
1. **Mock行為記錄**: 記錄和回放Mock交互
2. **性能分析**: Mock調用的性能統計
3. **自動化期望生成**: 基於實際API呼叫自動生成Mock期望
4. **契約測試集成**: 與API契約測試框架整合

### 潛在改進
1. **GraphQL Mock支援**: 支援GraphQL查詢和變異的Mock
2. **時間控制**: 更精確的時間和順序控制
3. **資料庫Mock**: 擴展到資料庫操作的Mock
4. **分佈式Mock**: 支援微服務間的Mock通信

---

## 📞 支援與貢獻

### 使用問題
- 查看測試檔案中的範例使用
- 參考現有遷移的測試檔案
- 使用 `go test ./internal/testutil/ -v` 運行範例測試

### 貢獻指南
1. 所有新Mock實現都應遵循統一接口
2. 新增功能需包含完整的測試覆蓋
3. 保持向後兼容性
4. 更新此文檔說明新功能

---

**統一Mock策略** - 讓測試更簡單、更可靠、更易維護！ 🎯✅