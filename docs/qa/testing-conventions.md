# 🧪 測試命名規範

## 測試檔案命名規範

### 檔案命名
- **主要測試套件**: `{service}_test.go`
- **專門測試套件**: `{service}_{purpose}_test.go`
  - 例: `registration_service_complete_test.go` (完整Mock測試)
  - 例: `registration_service_basic_test.go` (基礎整合測試)

### 測試方法命名

#### 1. 標準單元測試
```go
func Test{ServiceName}_{MethodName}_{Scenario}(t *testing.T)
```
**範例**:
- `TestRegistrationService_ValidatePhoneNumber_ValidInput`
- `TestLineBotService_HandleRequest_InvalidSignature`

#### 2. TestSuite方法
```go
func (suite *{ServiceName}TestSuite) Test{MethodName}_{Scenario}()
```
**範例**:
- `func (suite *RegistrationServiceCompleteTestSuite) TestRegisterUserWithPhone_LineAPISuccess()`
- `func (suite *RegistrationServiceBasicTestSuite) TestCheckUserRegistration_DatabaseIntegration()`

#### 3. 子測試(Sub-tests)
```go
t.Run("{Scenario_Description}", func(t *testing.T) {
    // test implementation
})
```
**範例**:
```go
func TestRegistrationService_ValidatePhoneNumber(t *testing.T) {
    t.Run("Valid phone number", func(t *testing.T) { ... })
    t.Run("Invalid phone number - too short", func(t *testing.T) { ... })
}
```

## 場景命名規範

### 成功場景
- `_Success` / `_ValidInput` / `_HappyPath`
- 例: `TestRegisterUserWithPhone_Success`

### 錯誤場景  
- `_Error` / `_InvalidInput` / `_Failure`
- `_DatabaseError` / `_NetworkError` / `_ValidationError`
- 例: `TestRegisterUserWithPhone_DatabaseError`

### 邊界條件
- `_EmptyInput` / `_NilInput` / `_BoundaryCase`
- 例: `TestRegisterUserWithPhone_EmptyLineUserID`

### 業務邏輯場景
- `_ExistingUser` / `_NewUser` / `_DuplicatePhone`
- 例: `TestRegisterUserWithPhone_ExistingUser`

## 測試架構標準

### 測試套件結構
```go
type {ServiceName}TestSuite struct {
    testutil.DatabaseTestSuite  // 如需資料庫
    repo    repository.UserRepository
    service *ServiceName
    // mock物件
}

func (suite *{ServiceName}TestSuite) SetupTest() {
    // 初始化邏輯
}
```

### 測試方法結構 (AAA Pattern)
```go
func (suite *TestSuite) TestMethodName_Scenario() {
    // 🔧 Arrange - 準備測試資料和Mock
    // 設置Mock行為
    // 創建測試資料
    
    // 🎬 Act - 執行被測試方法
    result, err := suite.service.MethodName(params)
    
    // ✅ Assert - 驗證結果
    assert.NoError(suite.T(), err)
    assert.Equal(suite.T(), expected, result)
    
    // 🔍 Additional Verifications - 額外驗證
    // Mock呼叫驗證
    // 資料庫狀態驗證
}
```

## 當前測試檔案架構

### Registration Service Tests
1. **`registration_service_complete_test.go`** - 主要測試套件
   - 使用Mock LINE Bot Client
   - 覆蓋所有業務場景
   - 16個詳細測試案例

2. **`registration_service_basic_test.go`** - 基礎整合測試
   - 資料庫整合測試
   - 不涉及外部API
   - 合約測試(Contract Testing)

3. **`registration_service_phone_test.go`** - 手機號驗證專門測試
   - 簡單的單元測試
   - 專注於電話號碼驗證邏輯

4. **`registration_service_test.go`** - 原始測試套件
   - Mock Repository模式
   - 保留用於特定場景測試

## 測試覆蓋率目標

- **Unit Tests**: 70%+ (當前 71.1%)
- **Integration Tests**: 完整業務流程覆蓋
- **Mock Tests**: 100% 業務邏輯覆蓋
- **Contract Tests**: 外部依賴介面測試

## 最佳實踐

### ✅ 應該做的
- 使用描述性的測試名稱
- 遵循AAA模式 (Arrange-Act-Assert)
- 每個測試方法只測試一個場景
- 使用testify/assert進行斷言
- Mock外部依賴
- 測試邊界條件和錯誤路徑

### ❌ 避免的
- 測試名稱過於簡短 (如 `TestMethod`)
- 在一個測試中測試多個場景
- 依賴外部服務 (如真實的LINE API)
- 測試之間的相互依賴
- 忽略錯誤路徑測試