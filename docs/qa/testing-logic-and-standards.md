# 測試邏輯與規範 (Testing Logic and Standards)

本文檔定義了 LINE Bot 應用程序的完整測試邏輯、規範和最佳實踐。

## 目錄

1. [測試架構概述](#測試架構概述)
2. [測試層級定義](#測試層級定義)
3. [測試責任邊界](#測試責任邊界)
4. [Mock 策略和使用規範](#mock-策略和使用規範)
5. [測試命名和組織規範](#測試命名和組織規範)
6. [測試數據管理](#測試數據管理)
7. [測試執行和持續整合](#測試執行和持續整合)
8. [測試最佳實踐](#測試最佳實踐)
9. [常見測試模式](#常見測試模式)
10. [故障診斷指南](#故障診斷指南)
11. [FX 依賴注入測試](#fx-依賴注入測試)
12. [LINE Bot SDK 測試](#line-bot-sdk-測試)
13. [資料庫測試策略](#資料庫測試策略)
14. [測試工具和輔助函數](#測試工具和輔助函數)
15. [測試環境配置](#測試環境配置)
16. [測試質量指標和追蹤](#測試質量指標和追蹤)
17. [測試代碼質量檢查](#測試代碼質量檢查)
18. [測試自動化腳本和工具](#測試自動化腳本和工具)
19. [文檔版本和更新歷史](#文檔版本和更新歷史)
20. [快速開始檢查清單](#快速開始檢查清單)

## 測試架構概述

### 當前測試覆蓋率狀況 (2025-01-06)
- **Handler層**: 100% (優秀 ✅)
- **Repository層**: 88.9% (優秀 ✅) 
- **Config層**: 88.9% (優秀 ✅)
- **服務層**: 測試建構中 (需要修復編譯錯誤)
- **整體目標**: >85%
- **近期目標**: 修復服務層測試並達到 >60%
- **急需改進的領域**: 
  - 服務層單元測試編譯錯誤修復
  - 錯誤處理和邊界條件測試
  - 整合測試完整性

### 整體測試金字塔

```
                    E2E Tests (少量)
                  /               \
              Contract Tests      API Tests  
            /                              \
        Integration Tests              Component Tests
      /                                              \
  Unit Tests (大量)                            Repository Tests
```

### 測試類型比例建議

- **單元測試 (Unit Tests)**: 70% - 快速、獨立、可靠
- **整合測試 (Integration Tests)**: 20% - 跨組件交互
- **契約測試 (Contract Tests)**: 7% - 外部服務契約
- **端對端測試 (E2E Tests)**: 3% - 完整用戶流程

## 測試層級定義

### 1. 單元測試 (Unit Tests)

**位置**: `internal/*/.*_test.go`

**職責**:
- 測試單個函數或方法的業務邏輯
- 驗證邊界條件和錯誤處理
- 確保程式碼的正確性和可靠性

**特徵**:
- 使用 Mock 隔離外部依賴
- 執行速度快 (<10ms per test)
- 覆蓋率目標: >90%
- 獨立運行，無外部依賴

**示例**:
```go
func TestRegistrationService_ValidatePhoneNumber(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 使用 MockSet 中的服務實例而非直接創建
        service := h.GetMockSet().RegistrationService
        
        t.Run("ValidPhoneNumbers", func(t *testing.T) {
            validPhones := h.GetValidPhones(5)
            for _, phone := range validPhones {
                err := service.ValidatePhoneNumber(phone)
                assert.NoError(t, err, "Phone %s should be valid", phone)
            }
        })
        
        t.Run("InvalidPhoneNumbers", func(t *testing.T) {
            invalidPhones := h.GetInvalidPhones()
            for _, phone := range invalidPhones {
                err := service.ValidatePhoneNumber(phone)
                assert.Error(t, err, "Phone %s should be invalid", phone)
            }
        })
    })
}
```

### 2. 整合測試 (Integration Tests)

**位置**: `test/integration/.*_integration_test.go`

**職責**:
- 測試多個組件之間的交互
- 驗證資料庫操作和事務處理
- 測試完整的業務流程

**特徵**:
- 使用真實資料庫 (SQLite/PostgreSQL)
- Mock 外部 API 服務
- 執行速度中等 (100ms-1s per test)
- 覆蓋率目標: >80%

**示例**:
```go
// 實際整合測試檔案位於 test/integration/
func TestUserRegistration_DatabaseIntegration(t *testing.T) {
    // 注意：此測試需要真實資料庫連接
    testutil.WithDatabaseSetup(t, func(setup *testutil.TestSetup) {
        scenario := setup.SetupUserRegistrationScenario()
        
        t.Run("CompleteRegistrationFlow", func(t *testing.T) {
            // 執行完整註冊流程
            result, err := scenario.ExecuteRegistration()
            assert.NoError(t, err)
            assert.True(t, result.IsNewUser)
            
            // 驗證資料庫中的用戶數據
            var savedUser model.User
            err = setup.GetDB().Where("line_user_id = ?", 
                scenario.NewUser.LineUserID).First(&savedUser).Error
            assert.NoError(t, err)
            assert.Equal(t, scenario.Phone, savedUser.PhoneNumber)
        })
    })
}

// 參考: test/integration/user_registration_comprehensive_integration_test.go
// 參考: test/integration/user_registration_integration_test.go
// 參考: test/integration/user_registration_phone_integration_test.go
```

### 3. 契約測試 (Contract Tests)

**位置**: `test/contract/.*_contract_test.go`

**職責**:
- 驗證與外部服務的接口契約
- 測試 API 請求和響應格式
- 確保服務間通信的正確性

**特徵**:
- Mock 外部服務響應
- 專注於接口格式驗證
- 執行速度快 (<50ms per test)
- 覆蓋率目標: >95% 的 API 端點

**示例**:
```go
func TestLineBot_APIContract(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        t.Run("LineProfileAPI_ResponseFormat", func(t *testing.T) {
            mockClient := h.GetMockSet().LineBotClient
            
            // 設置 Mock 響應
            userID := "test_user_123"
            expectedName := "Test User"
            
            // 使用實際存在的方法設置 Mock
            if mockLC, ok := mockClient.(*testutil.MockLineBotClient); ok {
                mockLC.SetProfileResponse(userID, expectedName)
            }
            
            // 測試契約 (注意：需要根據實際 Mock 接口調整)
            // profile, err := mockClient.GetProfile(userID).Do()
            // assert.NoError(t, err)
            // assert.Equal(t, expectedName, profile.DisplayName)
        })
    })
}

// 注意：契約測試目前尚未完整實現，需要根據實際 MockLineBotClient 接口調整
// 建議優先實現核心的 Mock 方法，如 SetProfileResponse、CreateTextMessage 等
```

### 4. 端對端測試 (E2E Tests)

**位置**: `test/e2e/.*_e2e_test.go`

**職責**:
- 測試完整的用戶場景
- 驗證系統的整體功能
- 模擬真實用戶操作

**特徵**:
- 使用真實 HTTP 請求
- 最小化 Mock 使用
- 執行速度慢 (1s-10s per test)
- 覆蓋率目標: >90% 的關鍵業務流程

## 測試責任邊界

### 明確的測試責任分工

| 測試類型 | 資料庫 | LINE Bot API | HTTP 請求 | 業務邏輯 | 錯誤處理 |
|---------|--------|-------------|-----------|----------|----------|
| Unit | Mock | Mock | Mock | ✅ 完整 | ✅ 完整 |
| Integration | ✅ 真實 | Mock | Mock | ✅ 流程 | ✅ 部分 |
| Contract | Mock | ✅ 真實格式 | ✅ 真實 | ❌ 最小 | ✅ API錯誤 |
| E2E | ✅ 真實 | Mock/真實 | ✅ 真實 | ✅ 完整 | ✅ 完整 |

### 測試邊界原則

1. **單一職責**: 每個測試只驗證一個特定功能
2. **最小依賴**: 使用最少的外部依賴完成測試目標
3. **快速回饋**: 優先保證快速測試的穩定性和完整性
4. **真實場景**: 高層測試應盡可能模擬真實使用情況

## Mock 策略和使用規範

### 統一 Mock 架構

#### 使用 TestHelper 統一接口

```go
// ✅ 正確的 Mock 使用方式
func TestWithUnifiedMocks(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 使用統一的 Mock 集合
        mockSet := h.GetMockSet()
        
        // 標準化的測試數據創建
        user := h.CreateUniqueUser()
        phone := h.GetValidPhone()
        
        // 模擬成功註冊流程
        result := h.SimulateSuccessfulRegistration(user.LineUserID, phone)
        assert.True(t, result.IsNewUser)
    })
}
```

#### Mock 行為配置

```go
// 創建自定義 Mock 行為
behavior := testutil.MockBehavior{
    EnableAutoSuccess: true,
    ErrorRate:         0.1,  // 10% 錯誤率
    DefaultDelay:      5 * time.Millisecond,
    EnableDetailedLogs: true,
}

testutil.WithTestHelperAndMocks(t, behavior, func(h *testutil.TestHelper) {
    // 測試邏輯
})
```

### Mock 使用原則

1. **接口優於實現**: Mock 接口而非具體類型
2. **行為驗證**: 關注 Mock 被如何調用，而非內部實現
3. **狀態隔離**: 每個測試都使用獨立的 Mock 實例
4. **清理資源**: 測試結束後自動重置 Mock 狀態

### 何時使用 Mock vs 真實實現

#### ✅ 使用 Mock 的情況:
- 外部 API 調用 (LINE Bot API)
- 慢速操作 (文件 I/O, 網絡請求)
- 難以控制的依賴 (時間、隨機數)
- 錯誤場景模擬

#### ✅ 使用真實實現的情況:
- 資料庫操作 (整合測試)
- 業務邏輯計算
- 數據轉換和驗證
- 內部組件交互

## 測試命名和組織規範

### 測試文件命名規範

```
# 單元測試
internal/service/registration_service.go
internal/service/registration_service_test.go

# 整合測試  
test/integration/user_registration_integration_test.go

# 契約測試
test/contract/linebot_api_contract_test.go

# 端對端測試
test/e2e/user_registration_e2e_test.go
```

### 測試函數命名規範

```go
// 格式: Test[Component]_[Method]_[Scenario]
func TestRegistrationService_ValidatePhoneNumber_InvalidFormat(t *testing.T) {}
func TestRegistrationService_RegisterUser_DuplicatePhone(t *testing.T) {}
func TestRegistrationService_RegisterUser_LineAPIError(t *testing.T) {}

// 使用 TestHelper 的格式: Test[Component]_[Feature]_WithTestHelper  
func TestRegistrationService_PhoneValidation_WithTestHelper(t *testing.T) {}
func TestRegistrationService_UserRegistration_WithTestHelper(t *testing.T) {}
```

### 子測試組織規範

```go
func TestRegistrationService_ComprehensiveScenarios(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 按功能分組
        t.Run("PhoneValidation", func(t *testing.T) {
            t.Run("ValidFormats", func(t *testing.T) { /* ... */ })
            t.Run("InvalidFormats", func(t *testing.T) { /* ... */ })
            t.Run("EdgeCases", func(t *testing.T) { /* ... */ })
        })
        
        t.Run("UserRegistration", func(t *testing.T) {
            t.Run("NewUser", func(t *testing.T) { /* ... */ })
            t.Run("ExistingUser", func(t *testing.T) { /* ... */ })
            t.Run("DuplicatePhone", func(t *testing.T) { /* ... */ })
        })
        
        t.Run("ErrorHandling", func(t *testing.T) {
            t.Run("DatabaseError", func(t *testing.T) { /* ... */ })
            t.Run("LineAPIError", func(t *testing.T) { /* ... */ })
            t.Run("ValidationError", func(t *testing.T) { /* ... */ })
        })
    })
}
```

## 測試數據管理

### 測試數據工廠 (TestDataFactory)

```go
// 創建唯一測試數據
func TestWithUniqueData(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 自動生成唯一數據，避免測試間衝突
        user1 := h.CreateUniqueUser()
        user2 := h.CreateUniqueUser()
        
        assert.NotEqual(t, user1.LineUserID, user2.LineUserID)
        
        // 批量創建測試數據
        userBatch := h.CreateUserBatch(10)
        assert.Len(t, userBatch, 10)
        
        // 創建特定特徵的數據
        userWithPhone := h.CreateUniqueUserWithPhone(h.GetValidPhone())
        assert.NotEmpty(t, userWithPhone.PhoneNumber)
    })
}
```

### 測試數據類型

1. **靜態數據**: 預定義的固定測試數據
2. **動態數據**: 運行時生成的唯一數據  
3. **邊界數據**: 測試邊界條件的特殊數據
4. **錯誤數據**: 觸發錯誤場景的無效數據

### 測試數據清理

```go
// 自動清理機制
testutil.WithTestSetup(t, func(setup *testutil.TestSetup) {
    // 測試數據會在測試結束時自動清理
    user := setup.SetupTestUserWithPhone("0912345678")
    
    // 手動清理（如果需要）
    setup.AddCleanup(func() {
        // 清理邏輯
    })
})
```

## 測試執行和持續整合

### 測試執行策略

```bash
# 快速測試 (開發期間)
go test ./internal/...           # 測試內部包
go test ./internal/service       # 測試服務層
go test ./internal/repository    # 測試Repository層
go test ./internal/handler       # 測試Handler層

# 完整測試 (CI/CD)
go test ./...                    # 運行所有測試
go test ./... -race              # 包含競態檢測
go test ./... -coverprofile=coverage.out  # 生成覆蓋率報告

# 分層測試
go test ./internal/service -run TestRegistrationService -v
go test ./internal/service -run TestLineBotService -v
go test ./test/integration -v    # 整合測試
# go test ./test/contract -v     # 契約測試 (待實現)

# 性能測試
go test ./... -bench=.
go test ./... -benchmem

# 專項測試
go test ./internal/testutil -v   # 測試工具測試
go test -coverprofile=coverage.out ./internal/service
go tool cover -html=coverage.out # 查看覆蓋率報告
```

### 測試覆蓋率目標

| 測試類型 | 目標覆蓋率 | 最低要求 |
|---------|-----------|----------|
| 單元測試 | >90% | >80% |
| 整合測試 | >80% | >70% |
| 整體覆蓋率 | >85% | >75% |

### CI/CD 整合

```yaml
# .github/workflows/test.yml 示例
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:13
        env:
          POSTGRES_PASSWORD: testpass
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      # 快速測試 (並行)
      - name: Unit Tests
        run: go test ./internal/... -v
      
      # Handler 測試
      - name: Handler Tests
        run: go test ./internal/handler/... -v
        
      # 數據庫測試 (需要 PostgreSQL)
      - name: Integration Tests  
        env:
          DB_HOST: localhost
          DB_USER: postgres
          DB_PASSWORD: testpass
          DB_NAME: testdb
          DB_SSL_MODE: disable
        run: go test ./test/integration/... -v
        
      # 覆蓋率檢查
      - name: Coverage Check
        run: |
          go test ./... -coverprofile=coverage.out
          go tool cover -func=coverage.out | grep total
          
      # 競態檢測
      - name: Race Condition Tests
        run: go test ./... -race
        
      # 代碼質量檢查
      - name: Go Vet
        run: go vet ./...
        
      # 格式檢查
      - name: Go Format
        run: |
          if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
            exit 1
          fi
```

## 測試最佳實踐

### 1. AAA 模式 (Arrange-Act-Assert)

```go
func TestRegistrationService_RegisterUser_Success(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // Arrange - 準備測試數據和環境
        lineUserID := h.CreateUniqueLineUserID()
        phoneNumber := h.GetValidPhone()
        
        // 設置必要的 Mock 行為
        if mockClient, ok := h.GetMockSet().LineBotClient.(*testutil.MockLineBotClient); ok {
            mockClient.SetProfileResponse(lineUserID, "Test User")
        }
        
        // Act - 執行被測試的操作
        result, err := h.GetMockSet().RegistrationService.RegisterUserWithPhone(lineUserID, phoneNumber)
        
        // Assert - 驗證結果
        assert.NoError(t, err)
        assert.NotNil(t, result)
        assert.True(t, result.IsNewUser)
        assert.False(t, result.NeedsPhoneNumber)
        assert.Equal(t, phoneNumber, result.User.PhoneNumber)
    })
}
```

### 2. 表驅動測試 (Table-Driven Tests)

```go
func TestPhoneValidation_Comprehensive(t *testing.T) {
    tests := []struct {
        name        string
        phoneNumber string
        expectError bool
        errorMsg    string
    }{
        {"Valid_Basic", "0912345678", false, ""},
        {"Valid_Different_Prefix", "0923456789", false, ""},
        {"Invalid_TooShort", "091234567", true, "手機號碼格式錯誤"},
        {"Invalid_TooLong", "09123456789", true, "手機號碼格式錯誤"},
        {"Invalid_WrongPrefix", "0812345678", true, "手機號碼格式錯誤"},
        {"Invalid_Empty", "", true, "手機號碼格式錯誤"},
        {"Invalid_NonNumeric", "091234567a", true, "手機號碼格式錯誤"},
    }
    
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                service := h.GetMockSet().RegistrationService
                err := service.ValidatePhoneNumber(tt.phoneNumber)
                
                if tt.expectError {
                    assert.Error(t, err)
                    assert.Contains(t, err.Error(), tt.errorMsg)
                } else {
                    assert.NoError(t, err)
                }
            })
        }
    })
}
```

### 3. 並發測試

```go
func TestRegistrationService_ConcurrentAccess(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 測試並發安全性
        h.AssertConcurrentSuccess(10, 5, func(goroutineID, opID int) error {
            lineUserID := h.CreateUniqueLineUserID()
            phone := h.GetValidPhone()
            
            result := h.SimulateSuccessfulRegistration(lineUserID, phone)
            if !result.IsNewUser {
                return errors.New("registration failed")
            }
            return nil
        })
    })
}
```

### 4. 性能測試

```go
func TestRegistrationService_Performance(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 測試執行時間
        h.AssertExecutionTime(func() {
            lineUserID := h.CreateUniqueLineUserID()
            phone := h.GetValidPhone()
            h.SimulateSuccessfulRegistration(lineUserID, phone)
        }, 100*time.Millisecond, "user registration")
    })
}
```

## 常見測試模式

### 1. 成功路徑測試 (Happy Path)

```go
func TestHappyPath(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 測試正常業務流程
        result := h.SimulateSuccessfulRegistration(
            h.CreateUniqueLineUserID(), 
            h.GetValidPhone(),
        )
        
        assert.True(t, result.IsNewUser)
        assert.False(t, result.NeedsPhoneNumber)
        assert.Empty(t, result.ValidationError)
    })
}
```

### 2. 錯誤路徑測試 (Error Path)

```go
func TestErrorPaths(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        t.Run("InvalidPhone", func(t *testing.T) {
            result, err := h.GetMockSet().RegistrationService.
                RegisterUserWithPhone("user", "invalid_phone")
                
            assert.NoError(t, err) // 業務錯誤不應該 panic
            assert.NotEmpty(t, result.ValidationError)
        })
        
        t.Run("DuplicatePhone", func(t *testing.T) {
            phone := h.GetValidPhone()
            h.CreateUniqueUserWithPhone(phone) // 先佔用手機號
            
            result, err := h.GetMockSet().RegistrationService.
                RegisterUserWithPhone(h.CreateUniqueLineUserID(), phone)
                
            assert.NoError(t, err)
            assert.Contains(t, result.ValidationError, "已被註冊")
        })
    })
}
```

### 3. 邊界條件測試 (Boundary Testing)

```go
func TestBoundaryConditions(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        t.Run("EmptyInputs", func(t *testing.T) {
            // 測試空輸入
            result, err := h.GetMockSet().RegistrationService.
                RegisterUserWithPhone("", "")
            // 驗證錯誤處理
        })
        
        t.Run("MaxLengthInputs", func(t *testing.T) {
            // 測試最大長度輸入
            longPhone := strings.Repeat("0", 100)
            result, err := h.GetMockSet().RegistrationService.
                RegisterUserWithPhone("user", longPhone)
            // 驗證邊界處理
        })
    })
}
```

## 故障診斷指南

### 常見測試失敗原因和解決方案

#### 1. 測試數據相關問題

**測試數據衝突**
```
症狀: 隨機測試失敗，"用戶已存在" 或 "UNIQUE constraint failed" 錯誤
原因: 測試間數據共享或清理不完整
解決: 
- 使用 h.CreateUniqueUser() 確保數據唯一性
- 檢查 TestHelper 清理機制是否正常運行
- 避免在測試中使用硬編碼的固定數據
```

**測試數據不一致**
```
症狀: 測試在本地通過但在 CI 中失敗
原因: 測試依賴特定的數據狀態或時間
解決:
- 使用 TestDataFactory 創建獨立的測試數據
- 避免依賴系統時間，使用固定時間進行測試
- 確保測試環境的一致性
```

#### 2. Mock 和依賴注入問題

**Mock 配置錯誤**
```
症狀: "unexpected method call" 或 "nil pointer" 錯誤
原因: Mock 期望配置不正確或缺少必要的 Mock 設置
解決: 
- 檢查 Mock 設置，使用統一的 TestHelper
- 確保所有必要的依賴都已 Mock
- 驗證 Mock 方法調用的參數和次數
```

**FX 依賴注入問題**
```
症狀: "could not build arguments for function" 錯誤
原因: FX 容器中缺少必要的依賴提供者
解決:
- 確保測試模塊包含所有必要的提供者
- 檢查 testutil.TestModule 是否正確配置
- 使用 fx.NopLogger 避免日誌相關的依賴問題
```

#### 3. 並發和競態問題

**異步操作競態**
```
症狀: 並發測試偶發失敗，"data race" 警告
原因: 未正確處理並發訪問或共享狀態
解決: 
- 使用 h.AssertConcurrentSuccess() 測試並發安全性
- 避免在測試中使用共享狀態
- 使用 go test -race 檢測競態條件
```

#### 4. 測試執行環境問題

**資料庫連接失敗**
```
症狀: "connection refused" 或資料庫相關錯誤
原因: 測試環境沒有正確設置資料庫
解決:
- 確保測試使用 SQLite 內存資料庫或正確的測試資料庫
- 檢查 WithDatabaseSetup 是否正確初始化
- 確認環境變數設置正確
```

**權限問題**
```
症狀: "permission denied" 或文件訪問錯誤
原因: 測試嘗試訪問不存在的文件或沒有權限的目錄
解決:
- 使用相對路徑或臨時目錄進行測試
- 確保測試文件的讀寫權限
- 避免在測試中創建永久文件
```

#### 5. LINE Bot SDK 相關問題

**Mock LINE Client 問題**
```
症狀: LINE Bot 相關的測試失敗
原因: MockLineBotClient 配置不正確
解決:
- 檢查 SetProfileResponse 等 Mock 方法是否正確調用
- 確認 Mock 響應格式符合預期
- 驗證事件創建方法的參數
```

### 測試調試技巧

```go
// 1. 啟用詳細日誌
func TestWithDebugLogs(t *testing.T) {
    // 創建開發模式日誌
    logger, _ := zap.NewDevelopment()
    
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 可以在需要的地方使用 logger
        logger.Info("Starting test with debug logs")
        
        // 測試邏輯...
    })
}

// 2. 檢查測試狀態
func TestWithStateInspection(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        result := h.SimulateSuccessfulRegistration("user", "0912345678")
        
        // 詳細檢查結果狀態
        t.Logf("Result: %+v", result)
        t.Logf("User: %+v", result.User)
        
        // 檢查 Mock 調用歷史
        if mockSet := h.GetMockSet(); mockSet != nil {
            // 驗證 Mock 調用次數
        }
    })
}

// 3. 隔離測試運行
// go test -run TestSpecificFunction -v
```

#### 6. 測試性能問題

**測試執行過慢**
```
症狀: 單元測試執行時間超過 100ms
原因: 測試中包含了不必要的 I/O 操作或複雜計算
解決:
- 檢查是否正確使用了 Mock
- 避免在單元測試中進行真實的網絡請求
- 使用 h.AssertExecutionTime() 監控性能
```

**記憶體洩漏**
```
症狀: 測試執行時記憶體使用量持續增長
原因: 測試中創建的資源沒有正確清理
解決:
- 確保使用 defer 或清理函數
- 檢查 goroutine 是否正確關閉
- 使用 go test -memprofile 分析記憶體使用
```

### 性能問題診斷和優化

```go
func TestPerformanceDiagnosis(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 測量操作耗時
        duration := h.MeasureExecutionTime(func() {
            h.SimulateSuccessfulRegistration(
                h.CreateUniqueLineUserID(), 
                h.GetValidPhone(),
            )
        })
        
        t.Logf("Registration took: %v", duration)
        
        // 如果超過預期時間，調查原因
        if duration > 50*time.Millisecond {
            t.Logf("Performance warning: operation took %v", duration)
        }
    })
}
```

## 測試維護和演進

### 定期測試維護

1. **每週**:
   - 檢查測試覆蓋率報告 (`go tool cover -func=coverage.out | grep total`)
   - 清理過時或重複的測試
   - 更新測試數據和 Mock 配置
   - 檢查失敗測試的根本原因

2. **每月**:
   - 回顧測試執行時間 (`go test -v ./... | grep -E "(PASS|FAIL|ok|FAIL).*[0-9]+\.[0-9]+s"`)
   - 優化慢速測試 (>1s 的單元測試需要優化)
   - 更新測試文檔和最佳實踐
   - 檢查測試工具和依賴更新

3. **每季度**:
   - 評估測試架構和策略有效性
   - 重構測試代碼以提高可維護性
   - 培訓團隊成員新的測試技術
   - 分析測試 ROI 和改進方向

### 當前測試健康狀況檢查

運行以下命令定期檢查測試狀況：

```bash
# 快速健康檢查
go test ./... -v | grep -E "(PASS|FAIL)" | tail -10

# 覆蓋率趨勢
go test ./... -coverprofile=coverage.out
echo "當前覆蓋率: $(go tool cover -func=coverage.out | grep total | awk '{print $3}')"

# 慢速測試識別
go test -v ./... 2>&1 | grep -E "ok.*[1-9][0-9]*\.[0-9]+s"

# Mock 使用統計
grep -r "testutil\.WithTestHelper" internal/ --include="*_test.go" | wc -l
```

### 測試代碼質量標準

- **可讀性**: 測試應該是自文檔化的
- **可維護性**: 測試應該易於修改和擴展  
- **可靠性**: 測試結果應該穩定一致
- **效率性**: 測試應該快速執行

---

## 結論

這套測試邏輯與規範提供了：

✅ **完整的測試策略** - 從單元到端對端的全覆蓋  
✅ **清晰的責任邊界** - 每層測試都有明確職責  
✅ **統一的工具和模式** - 一致的測試風格和實踐  
✅ **實用的最佳實踐** - 經過驗證的測試技巧和模式  
✅ **完善的維護指南** - 長期測試健康的保障  

遵循這些規範可以確保測試代碼的質量、可維護性和可靠性，為產品的穩定交付提供堅實保障。

## FX 依賴注入測試

### FX 測試應用程序

本應用使用 **Uber FX** 進行依賴注入，測試時需要特別處理 FX 容器。

```go
func TestWithFXApp(t *testing.T) {
    // 創建測試 FX 應用
    app := fx.New(
        // 使用測試模塊
        testutil.TestModule,
        config.ConfigModule,
        linebot.LineBotModule,
        
        // 測試特定的配置
        fx.Provide(func() *config.AppConfig {
            return &config.AppConfig{
                Port: "8080",
                LineBot: config.LineBotConfig{
                    ChannelSecret: "test_secret",
                    ChannelToken:  "test_token",
                },
            }
        }),
        
        // 測試邏輯
        fx.Invoke(func(service *service.LineBotService) {
            assert.NotNil(t, service)
            // 執行測試邏輯
        }),
    )
    
    // 啟動和停止應用
    ctx := context.Background()
    assert.NoError(t, app.Start(ctx))
    defer app.Stop(ctx)
}
```

### Handler 層 FX 測試

```go
func TestLinebotHandler_WithFX(t *testing.T) {
    app := fx.New(
        testutil.TestModule,
        handler.HandlerModule,
        fx.Invoke(func(h *handler.LinebotHandler) {
            // 創建測試請求
            req := httptest.NewRequest("POST", "/callback", 
                strings.NewReader(`{"events": []}`))
            req.Header.Set("Content-Type", "application/json")
            
            recorder := httptest.NewRecorder()
            h.HandleCallback(recorder, req)
            
            assert.Equal(t, http.StatusOK, recorder.Code)
        }),
    )
    
    ctx := context.Background()
    assert.NoError(t, app.Start(ctx))
    defer app.Stop(ctx)
}
```

## LINE Bot SDK 測試

### LINE Webhook 測試

```go
func TestLINEWebhook_Processing(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 創建 LINE 事件
        textEvent := h.CreateTextMessageEvent("test_user", "Hello")
        joinEvent := h.CreateMemberJoinEvent("new_user") 
        postbackEvent := h.CreatePostbackEvent("user", "action=register")
        
        // 創建 Webhook 請求
        events := []linebot.Event{textEvent, joinEvent, postbackEvent}
        req := h.CreateLINEWebhookRequest(events)
        
        // 測試事件處理
        recorder := httptest.NewRecorder()
        // 這裡需要實際的 handler 處理邏輯
        
        h.AssertHTTPSuccess(recorder)
    })
}
```

### LINE Bot Client Mock 測試

```go
func TestLineBot_ProfileRetrieval(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        mockClient := h.GetMockSet().LineBotClient
        
        // 設置 Mock 響應
        userID := "test_user_123"
        expectedName := "Test User"
        
        // 確保 Mock 類型轉換正確
        if mockLC, ok := mockClient.(*testutil.MockLineBotClient); ok {
            mockLC.SetProfileResponse(userID, expectedName)
            
            // 注意：這裡的實際調用需要根據 MockLineBotClient 的實際接口調整
            // 以下是概念性示例，實際使用時需要查看 Mock 實現
            
            // 例如可能的調用方式:
            // profile := mockLC.GetMockedProfile(userID)
            // assert.Equal(t, expectedName, profile.DisplayName)
            
            // 或者測試 Mock 是否正確設置:
            assert.NotNil(t, mockLC, "Mock client should be properly initialized")
        }
    })
}
```

## 資料庫測試策略

### 資料庫連接測試

```go
func TestDatabase_Connection(t *testing.T) {
    testutil.WithDatabaseSetup(t, func(setup *testutil.TestSetup) {
        db := setup.GetDB()
        
        // 測試資料庫連接
        sqlDB, err := db.DB()
        assert.NoError(t, err)
        assert.NoError(t, sqlDB.Ping())
        
        // 測試自動遷移
        assert.NoError(t, db.AutoMigrate(&model.User{}))
    })
}
```

### 事務測試

```go
func TestDatabase_TransactionRollback(t *testing.T) {
    testutil.WithDatabaseSetup(t, func(setup *testutil.TestSetup) {
        db := setup.GetDB()
        
        // 開始事務
        tx := db.Begin()
        
        // 在事務中創建用戶
        user := setup.GetTestDataFactory().CreateUniqueUser()
        err := tx.Create(user).Error
        assert.NoError(t, err)
        
        // 回滾事務
        tx.Rollback()
        
        // 驗證用戶不存在
        var count int64
        db.Model(&model.User{}).Where("line_user_id = ?", user.LineUserID).Count(&count)
        assert.Equal(t, int64(0), count)
    })
}
```

### GORM 約束測試

```go
func TestDatabase_Constraints(t *testing.T) {
    testutil.WithDatabaseSetup(t, func(setup *testutil.TestSetup) {
        db := setup.GetDB()
        user := setup.SetupTestUserWithFields(map[string]interface{}{})
        
        // 創建用戶
        err := db.Create(user).Error
        assert.NoError(t, err)
        
        // 測試唯一約束 - 重複的 LineUserID
        duplicateUser := *user
        duplicateUser.ID = 0 // 重置 ID
        err = db.Create(&duplicateUser).Error
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "UNIQUE constraint failed")
        
        // 測試唯一約束 - 重複的手機號碼
        anotherUser := setup.SetupTestUserWithFields(map[string]interface{}{
            "phone_number": user.PhoneNumber,
        })
        err = db.Create(anotherUser).Error
        assert.Error(t, err)
    })
}
```

## 測試工具和輔助函數

### 自定義 Matcher

```go
// 自定義斷言函數
func AssertValidTaiwanPhone(t *testing.T, phone string) {
    assert.Len(t, phone, 10, "Phone should be 10 digits")
    assert.True(t, strings.HasPrefix(phone, "09"), "Phone should start with 09")
    
    // 檢查所有字符都是數字
    for _, char := range phone {
        assert.True(t, char >= '0' && char <= '9', "All characters should be digits")
    }
}

// 使用示例
func TestPhoneGeneration(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        phone := h.GetValidPhone()
        AssertValidTaiwanPhone(t, phone)
    })
}
```

### 測試時間控制

```go
func TestWithTimeControl(t *testing.T) {
    // 固定時間進行測試
    fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
    
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        user := h.CreateUserWithCustomFields(map[string]interface{}{
            "created_at": fixedTime,
        })
        
        assert.Equal(t, fixedTime.Unix(), user.CreatedAt.Unix())
    })
}
```

## 測試環境配置

### 環境變量管理

```go
func TestWithEnvironment(t *testing.T) {
    testutil.WithTestSetup(t, func(setup *testutil.TestSetup) {
        // 設置測試環境變量
        setup.SetupTestEnvironment(map[string]string{
            "CHANNEL_SECRET": "test_secret",
            "CHANNEL_TOKEN":  "test_token", 
            "DB_HOST":        "localhost",
            "TEST_MODE":      "true",
        })
        
        // 環境變量會在測試結束時自動復原
        // 測試邏輯...
    })
}
```

### 配置驗證測試

```go
func TestConfig_Validation(t *testing.T) {
    tests := []struct {
        name     string
        envVars  map[string]string
        wantErr  bool
        errMsg   string
    }{
        {
            name: "ValidConfig",
            envVars: map[string]string{
                "CHANNEL_SECRET": "valid_secret",
                "CHANNEL_TOKEN":  "valid_token",
            },
            wantErr: false,
        },
        {
            name: "MissingSecret",
            envVars: map[string]string{
                "CHANNEL_TOKEN": "valid_token",
            },
            wantErr: true,
            errMsg:  "CHANNEL_SECRET is required",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            testutil.WithTestSetup(t, func(setup *testutil.TestSetup) {
                setup.SetupTestEnvironment(tt.envVars)
                
                cfg, err := config.LoadConfig()
                if tt.wantErr {
                    assert.Error(t, err)
                    assert.Contains(t, err.Error(), tt.errMsg)
                } else {
                    assert.NoError(t, err)
                    assert.NotNil(t, cfg)
                }
            })
        })
    }
}
```

## 測試質量指標和追蹤

### 關鍵測試指標

1. **覆蓋率指標**：
   ```bash
   # 生成詳細覆蓋率報告
   go test ./... -coverprofile=coverage.out -covermode=atomic
   go tool cover -func=coverage.out | grep -E "(service|repository|handler)"
   
   # 覆蓋率門檻檢查
   COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
   if (( $(echo "$COVERAGE < 75" | bc -l) )); then
       echo "Warning: Coverage $COVERAGE% below 75% threshold"
   fi
   ```

2. **測試執行指標**：
   ```bash
   # 測試執行時間分析
   go test -v ./... 2>&1 | grep -E "^--- (PASS|FAIL)" | awk '{print $4, $5}' | sort -n
   
   # 並發安全測試
   go test ./... -race -count=5
   
   # 記憶體洩漏檢測
   go test ./... -run=TestRegistrationService -memprofile=mem.prof
   go tool pprof mem.prof
   ```

3. **Mock 使用統計**：
   ```bash
   # 統計 Mock 使用情況
   grep -r "WithTestHelper" internal/ --include="*_test.go" | wc -l
   grep -r "GetMockSet" internal/ --include="*_test.go" | wc -l
   grep -r "SimulateSuccessfulRegistration" internal/ --include="*_test.go" | wc -l
   ```

### 測試債務追蹤

維護一個測試債務清單，定期檢視和解決：

```go
// 在測試文件中標註待辦事項
func TestRegistrationService_EdgeCases(t *testing.T) {
    // TODO: 增加網絡錯誤場景測試
    // TODO: 增加資料庫連接失敗測試  
    // TODO: 增加並發註冊衝突測試
    // FIXME: 現有測試在某些條件下會失敗
    
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // 現有測試邏輯
    })
}
```

### 測試質量檢查清單

**每個新測試都應該滿足**：
- [ ] 使用統一的 TestHelper 架構
- [ ] 測試名稱清楚描述測試場景
- [ ] 包含成功路徑和錯誤路徑測試
- [ ] Mock 配置正確且有意義
- [ ] 斷言詳細且具描述性
- [ ] 測試執行時間 <100ms (單元測試)
- [ ] 無競態條件
- [ ] 測試間相互獨立

## 測試代碼質量檢查

### Lint 和格式檢查

```bash
# 代碼格式檢查
go fmt ./...

# 靜態分析
go vet ./...

# 更嚴格的 lint 檢查 (需要安裝 golangci-lint)
golangci-lint run ./...

# 測試代碼覆蓋率檢查
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

### 測試命令別名

當前專案的 `Makefile` 主要用於 Docker 容器管理，測試命令建議直接使用 Go 工具：

```bash
# 基本測試命令
go test ./... -v                    # 運行所有測試
go test ./internal/... -v           # 測試內部包
go test ./internal/handler -v       # Handler 層測試
go test ./internal/repository -v    # Repository 層測試
go test ./test/integration/... -v   # 整合測試

# 覆蓋率和質量檢查
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
go test ./... -race                 # 競態條件檢查
go test ./... -bench=. -benchmem    # 性能測試
go vet ./...                        # 靜態分析
```

## 測試自動化腳本和工具

### 測試執行腳本

創建一個 `scripts/test.sh` 腳本自動化常見測試任務：

```bash
#!/bin/bash
# scripts/test.sh - 測試自動化腳本

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 測試類型
case "${1:-all}" in
    "unit")
        echo -e "${GREEN}🧪 Running unit tests...${NC}"
        go test ./internal/... -v -count=1
        ;;
    "integration")  
        echo -e "${GREEN}🔗 Running integration tests...${NC}"
        go test ./test/integration/... -v
        ;;
    "coverage")
        echo -e "${GREEN}📊 Generating coverage report...${NC}"
        go test ./... -coverprofile=coverage.out -covermode=atomic
        go tool cover -html=coverage.out -o coverage.html
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
        echo -e "${GREEN}Total coverage: ${COVERAGE}%${NC}"
        
        if (( $(echo "$COVERAGE < 75" | bc -l) )); then
            echo -e "${YELLOW}⚠️  Warning: Coverage below 75% threshold${NC}"
        fi
        ;;
    "race")
        echo -e "${GREEN}🏁 Running race condition tests...${NC}"
        go test ./... -race -count=3
        ;;
    "all")
        echo -e "${GREEN}🚀 Running full test suite...${NC}"
        ./scripts/test.sh unit
        ./scripts/test.sh integration  
        ./scripts/test.sh coverage
        ./scripts/test.sh race
        echo -e "${GREEN}✅ All tests completed successfully${NC}"
        ;;
    *)
        echo "Usage: $0 [unit|integration|coverage|race|all]"
        exit 1
        ;;
esac
```

### 測試數據重置腳本

創建 `scripts/reset-test-data.sh` 清理測試環境：

```bash
#!/bin/bash
# scripts/reset-test-data.sh - 重置測試數據

echo "🗑️  Cleaning test artifacts..."

# 清理覆蓋率文件
rm -f coverage.out coverage.html *.prof
rm -f service_coverage.out handler_coverage.out

# 清理測試臨時文件
find . -name "*.test" -delete
find . -name "*_test.db" -delete

# 清理 Docker 測試容器 (如果存在)
docker ps -q --filter "name=test_" | xargs -r docker stop
docker ps -aq --filter "name=test_" | xargs -r docker rm

echo "✅ Test environment cleaned"
```

### 測試報告生成器

創建 `scripts/generate-test-report.sh` 生成詳細測試報告：

```bash
#!/bin/bash
# scripts/generate-test-report.sh - 生成詳細測試報告

REPORT_DIR="docs/test-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="${REPORT_DIR}/test_report_${TIMESTAMP}.md"

mkdir -p $REPORT_DIR

echo "# 測試報告 - $(date)" > $REPORT_FILE
echo "" >> $REPORT_FILE

echo "## 測試覆蓋率" >> $REPORT_FILE
go test ./... -coverprofile=coverage.out -covermode=atomic > /dev/null 2>&1
go tool cover -func=coverage.out >> $REPORT_FILE
echo "" >> $REPORT_FILE

echo "## 各層覆蓋率詳情" >> $REPORT_FILE
echo "### 服務層" >> $REPORT_FILE
go test ./internal/service -coverprofile=service_coverage.out > /dev/null 2>&1
go tool cover -func=service_coverage.out | grep -v "total" >> $REPORT_FILE
echo "" >> $REPORT_FILE

echo "### Repository 層" >> $REPORT_FILE  
go test ./internal/repository -coverprofile=repo_coverage.out > /dev/null 2>&1
go tool cover -func=repo_coverage.out | grep -v "total" >> $REPORT_FILE
echo "" >> $REPORT_FILE

echo "## 測試執行統計" >> $REPORT_FILE
echo "\`\`\`" >> $REPORT_FILE
go test -v ./... 2>&1 | grep -E "(^=== RUN|^--- PASS|^--- FAIL|^ok)" >> $REPORT_FILE
echo "\`\`\`" >> $REPORT_FILE

echo "測試報告已生成: $REPORT_FILE"
```

### CI/CD 整合腳本

創建 `scripts/ci-test.sh` 供 CI/CD 使用：

```bash
#!/bin/bash
# scripts/ci-test.sh - CI/CD 測試腳本

set -e

echo "🚀 Starting CI test pipeline..."

# 1. 代碼格式檢查
echo "📋 Checking code format..."
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo "❌ Code format issues found:"
    gofmt -s -l .
    exit 1
fi

# 2. 靜態分析
echo "🔍 Running static analysis..."
go vet ./...

# 3. 單元測試
echo "🧪 Running unit tests..."
go test ./internal/... -v -count=1

# 4. 整合測試
echo "🔗 Running integration tests..."
if [ -d "test/integration" ]; then
    go test ./test/integration/... -v
fi

# 5. 覆蓋率檢查
echo "📊 Checking coverage..."
go test ./... -coverprofile=coverage.out -covermode=atomic
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
echo "Total coverage: ${COVERAGE}%"

MIN_COVERAGE=${MIN_COVERAGE:-75}
if (( $(echo "$COVERAGE < $MIN_COVERAGE" | bc -l) )); then
    echo "❌ Coverage $COVERAGE% is below minimum $MIN_COVERAGE%"
    exit 1
fi

# 6. 競態檢測
echo "🏁 Running race condition tests..."
go test ./... -race -count=2

echo "✅ All CI tests passed successfully!"
```

---

## 文檔版本和更新歷史

### 版本 v2.1.0 (2025-01-06)

**主要更新**：
- ✅ 更新實際測試覆蓋率數據
- ✅ 修復過時的代碼示例和方法引用
- ✅ 簡化文檔結構，減少冗餘內容
- ✅ 與實際專案結構對齊
- ✅ 改進語言一致性

**主要改進**：
- 統一了所有測試使用 `testutil.WithTestHelper` 架構
- 消除了重複測試代碼，提升測試維護性
- 增加了實用的測試自動化腳本和工具
- 完善了 CI/CD 整合和測試報告生成
- 強化了測試債務追蹤和質量檢查清單

### 使用指南

1. **新加入團隊成員**：
   - 先閱讀「測試架構概述」了解整體策略
   - 學習「測試最佳實踐」中的 AAA 模式和表驅動測試
   - 熟悉 `testutil.WithTestHelper` 的使用方法

2. **現有開發者**：
   - 重點關注「測試質量指標和追蹤」章節
   - 使用新增的自動化腳本提升開發效率
   - 參考「常見測試模式」改進現有測試

3. **測試負責人**：
   - 定期執行「測試健康檢查」腳本
   - 監控覆蓋率趨勢和測試債務
   - 組織團隊進行測試質量回顧

## 快速開始檢查清單

**設置測試環境**：
- [ ] 確認 Go 1.21+ 和必要依賴已安裝
- [ ] 運行 `go test ./internal/testutil -v` 驗證測試工具正常
- [ ] 執行 `go test ./... -coverprofile=coverage.out` 檢查測試環境狀態

**編寫第一個測試**：
- [ ] 使用 `testutil.WithTestHelper` 框架
- [ ] 遵循 AAA 模式（Arrange-Act-Assert）
- [ ] 包含成功和失敗路徑測試
- [ ] 運行 `go test -v` 確認測試通過

**持續改進**：
- [ ] 定期檢查覆蓋率：`go test ./... -coverprofile=coverage.out`
- [ ] 運行完整測試套件：`go test ./... -race`
- [ ] 監控測試執行時間和性能指標

---

## 結論

這套測試邏輯與規範經過實戰驗證，提供了：

🎯 **完整的測試策略** - 從單元測試到端對端測試的全覆蓋架構  
🏗️ **統一的工具框架** - TestHelper 和 Mock 統一管理系統  
📊 **質量追蹤機制** - 覆蓋率監控和測試債務管理  
🤖 **自動化工具鏈** - 腳本化的測試執行和報告生成  
📚 **實用的最佳實踐** - 經過驗證的測試模式和技巧  
🔄 **持續改進流程** - 定期維護和演進機制  

**當前測試狀態** (2025-01-06)：
- Handler層：100% → 優秀 ✅
- Repository層：88.9% → 優秀 ✅  
- Config層：88.9% → 優秀 ✅
- 服務層：需修復 → 目標：60%+
- 整體覆蓋率：需要服務層修復後重新計算

**近期重點任務**：
1. 提升服務層測試覆蓋率，重點關注 LineBotService 和 RegistrationService
2. 完善錯誤處理和邊界條件測試
3. 建立自動化測試報告和持續監控機制
4. 推廣 TestHelper 使用，統一測試風格

遵循這些規範可以確保測試代碼的質量、可維護性和可靠性，為 LINE Bot 應用的穩定交付提供堅實保障。

**下一步行動建議**：
- 立即運行 `go test ./... -race` 檢查當前測試狀態
- 每週執行 `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out` 監控覆蓋率變化  
- 定期檢查 `go test -v ./... 2>&1 | grep FAIL` 識別失敗測試
- 持續改進測試質量和覆蓋率

## 附錄：故障排除快速參考

### 快速診斷命令

```bash
# 1. 檢查特定測試失敗
go test -v ./internal/service -run TestRegistrationService

# 2. 檢查測試覆蓋率
go test -coverprofile=coverage.out ./internal/service
go tool cover -func=coverage.out

# 3. 檢查競態條件
go test -race ./internal/service

# 4. 檢查記憶體使用
go test -memprofile=mem.prof ./internal/service
go tool pprof mem.prof

# 5. 詳細測試輸出
go test -v -count=1 ./internal/service

# 6. 檢查測試執行時間
go test -v ./internal/service 2>&1 | grep -E "(PASS|FAIL).*[0-9]+\.[0-9]+s"
```

### 常用測試調試技巧

1. **隔離問題測試**：使用 `-run` 參數只運行特定測試
2. **禁用並行**：使用 `-p=1` 避免並行執行
3. **增加詳細輸出**：使用 `-v` 查看詳細日誌
4. **重複執行**：使用 `-count=N` 重複測試檢查穩定性
5. **測試超時**：使用 `-timeout` 設置測試超時時間

### 測試環境檢查清單

- [ ] Go 版本 >= 1.21
- [ ] 所有依賴正確安裝 (`go mod tidy`)
- [ ] 測試工具包正常 (`go test ./internal/testutil -v`)
- [ ] 環境變數設置正確
- [ ] 資料庫配置正確（如果需要）
- [ ] Docker 環境正常（如果使用）

## 測試模板和代碼片段

### 基本單元測試模板

```go
package service

import (
    "testing"
    "linebot_bar/internal/testutil"
    "github.com/stretchr/testify/assert"
)

func TestServiceName_MethodName_Scenario(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        // Arrange - 準備測試數據
        user := h.CreateUniqueUser()
        expectedResult := "expected_value"
        
        // Act - 執行被測試的方法
        result, err := h.GetMockSet().ServiceName.MethodName(user.ID)
        
        // Assert - 驗證結果
        assert.NoError(t, err)
        assert.Equal(t, expectedResult, result)
    })
}
```

### 表驅動測試模板

```go
func TestServiceName_ValidationMethod(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectError bool
        errorMsg    string
    }{
        {"Valid_Case", "valid_input", false, ""},
        {"Invalid_Empty", "", true, "不能為空"},
        {"Invalid_Format", "invalid", true, "格式錯誤"},
    }
    
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                service := h.GetMockSet().ServiceName
                err := service.ValidationMethod(tt.input)
                
                if tt.expectError {
                    assert.Error(t, err)
                    assert.Contains(t, err.Error(), tt.errorMsg)
                } else {
                    assert.NoError(t, err)
                }
            })
        }
    })
}
```

### 整合測試模板

```go
func TestServiceName_DatabaseIntegration(t *testing.T) {
    testutil.WithDatabaseSetup(t, func(setup *testutil.TestSetup) {
        t.Run("DatabaseOperation", func(t *testing.T) {
            // Arrange
            testData := setup.SetupTestUserWithFields(map[string]interface{}{})
            
            // Act
            err := setup.GetDB().Create(testData).Error
            
            // Assert
            assert.NoError(t, err)
            
            // Verify in database
            var retrieved model.User
            err = setup.GetDB().Where("line_user_id = ?", testData.LineUserID).First(&retrieved).Error
            assert.NoError(t, err)
            assert.Equal(t, testData.LineUserID, retrieved.LineUserID)
        })
    })
}
```

### 錯誤處理測試模板

```go
func TestServiceName_ErrorHandling(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        t.Run("DatabaseError", func(t *testing.T) {
            // 模擬資料庫錯誤
            // 這需要根據實際 Mock 實現調整
            
            result, err := h.GetMockSet().ServiceName.MethodName("test_id")
            
            assert.Error(t, err)
            assert.Nil(t, result)
            assert.Contains(t, err.Error(), "database")
        })
        
        t.Run("ValidationError", func(t *testing.T) {
            result, err := h.GetMockSet().ServiceName.MethodName("")
            
            assert.NoError(t, err) // 業務錯誤通常不返回 error
            assert.NotNil(t, result)
            assert.NotEmpty(t, result.ValidationError)
        })
    })
}
```

### HTTP Handler 測試模板

```go
func TestHandler_HTTPEndpoint(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        t.Run("SuccessfulRequest", func(t *testing.T) {
            // 創建測試請求
            requestBody := `{"field": "value"}`
            req := httptest.NewRequest("POST", "/api/endpoint", 
                strings.NewReader(requestBody))
            req.Header.Set("Content-Type", "application/json")
            
            recorder := httptest.NewRecorder()
            
            // 執行請求（這裡需要實際的 handler）
            // handler.ServeHTTP(recorder, req)
            
            // 驗證響應
            h.AssertHTTPSuccess(recorder)
            
            // 驗證響應內容
            expectedJSON := `{"success": true}`
            h.AssertJSONResponse(recorder, expectedJSON)
        })
    })
}
```

### 並發測試模板

```go
func TestServiceName_ConcurrentAccess(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        t.Run("ConcurrentOperations", func(t *testing.T) {
            // 測試並發安全性
            h.AssertConcurrentSuccess(5, 10, func(goroutineID, opID int) error {
                user := h.CreateUniqueUser()
                result, err := h.GetMockSet().ServiceName.MethodName(user.ID)
                if err != nil {
                    return err
                }
                if result == nil {
                    return errors.New("unexpected nil result")
                }
                return nil
            })
        })
    })
}
```

### 性能測試模板

```go
func TestServiceName_Performance(t *testing.T) {
    testutil.WithTestHelper(t, func(h *testutil.TestHelper) {
        t.Run("ExecutionTime", func(t *testing.T) {
            user := h.CreateUniqueUser()
            
            // 測試執行時間
            h.AssertExecutionTime(func() {
                _, err := h.GetMockSet().ServiceName.MethodName(user.ID)
                assert.NoError(t, err)
            }, 50*time.Millisecond, "service method execution")
        })
    })
}
```

---

> 📝 **文檔最後更新**: 2025-09-06  
> 🔄 **版本**: v2.0.0  
> 👥 **維護者**: 開發團隊  
> 📧 **反饋**: 如有問題請提交 Issue 或聯繫團隊