# Clean Architecture 實現指南

> **版本**: 1.0
> **最後更新**: 2025-01-10
> **目標**: 提供從 DDD 架構設計到 Go 代碼實現的完整指南

---

## 關於本指南

本目錄包含 Clean Architecture 的具體實現指南，將 DDD 架構設計文檔轉化為可執行的 Go 代碼結構。

### 與 DDD 文檔的關係

```
docs/architecture/ddd/          ← 架構設計（What & Why）
         ↓
docs/architecture/implementation/ ← 實現指南（How）
         ↓
internal/                        ← 實際代碼（Code）
```

---

## 文檔目錄

### 1. **[目錄結構設計](./01-directory-structure.md)** ⭐ 必讀
   - 完整的 Go 項目目錄結構
   - 各層級的目錄組織
   - 文件命名規範
   - 包（Package）劃分原則

### 2. **[Domain Layer 實現指南](./02-domain-layer-implementation.md)**
   - 聚合根的 Go 實現
   - 值對象的構造與驗證
   - 領域服務的實現模式
   - Repository 接口定義
   - 領域事件的實現
   - 領域錯誤的定義

### 3. **[Application Layer 實現指南](./03-application-layer-implementation.md)**
   - Use Case 的實現模式
   - DTO 的設計與轉換
   - 事務管理（Transaction Context Pattern）
   - Command/Query Handlers
   - 事件處理器（Event Handlers）

### 4. **[Infrastructure Layer 實現指南](./04-infrastructure-layer-implementation.md)**
   - GORM Repository 實現
   - Redis 緩存實現
   - 外部服務適配器（LINE SDK, Google OAuth）
   - 事件總線實現
   - 配置管理

### 5. **[Presentation Layer 實現指南](./05-presentation-layer-implementation.md)**
   - Gin HTTP Handler 實現
   - LINE Bot Webhook Handler
   - 請求驗證與錯誤處理
   - DTO 映射

### 6. **[依賴注入配置](./06-dependency-injection.md)**
   - Uber FX 模組設計
   - 依賴注入的模塊順序
   - 接口綁定與生命週期管理
   - 測試時的依賴替換

### 7. **[包命名規範](./07-package-naming.md)**
   - Go 包命名最佳實踐
   - 避免循環依賴
   - 包的可見性控制
   - 內部包（internal/）的使用

### 8. **[完整代碼範例](./08-code-examples.md)**
   - 積分管理 Context 完整實現
   - 從 HTTP 請求到數據庫的完整流程
   - 測試代碼範例
   - 常見錯誤與解決方案

### 9. **[生產環境保護措施](./09-production-safeguards.md)**
   - Panic Recovery Middleware
   - 監控與告警機制
   - 資料完整性保護
   - 錯誤處理最佳實踐

### 10. **[實作路線圖](./10-implementation-roadmap.md)** ⭐ 開始實作必讀
   - 設計決策總結
   - 10 階段實作計劃（12 週）
   - 每週檢查點與里程碑
   - TDD 工作流程
   - 測試策略與覆蓋率目標
   - 立即行動步驟（Day 1 任務）
   - 風險管理

### 11. **[詳細任務分解計劃](./11-detailed-task-breakdown.md)** 🔥 每日執行指南
   - 精確到小時的任務分配
   - Day-by-Day 執行步驟（Week 1-3 天完整範例）
   - 完整測試程式碼範例（TDD 流程）
   - 完整實作程式碼範例
   - 每個任務的完成標準
   - 每日檢查點驗證
   - 進度追蹤表格
   - 常用命令快速參考

---

## 快速開始

### 我想知道...

- **🚀 我要開始實作了，從哪裡開始？** → 閱讀 [10-實作路線圖](./10-implementation-roadmap.md)（包含第一天任務）
- **如何組織項目目錄？** → 閱讀 [01-目錄結構設計](./01-directory-structure.md)
- **如何實現聚合根？** → 閱讀 [02-Domain Layer 實現指南](./02-domain-layer-implementation.md) 第 2.2 節
- **如何實現 Use Case？** → 閱讀 [03-Application Layer 實現指南](./03-application-layer-implementation.md) 第 3.2 節
- **如何實現 Repository？** → 閱讀 [04-Infrastructure Layer 實現指南](./04-infrastructure-layer-implementation.md) 第 4.2 節
- **如何配置依賴注入？** → 閱讀 [06-依賴注入配置](./06-dependency-injection.md)
- **如何避免循環依賴？** → 閱讀 [07-包命名規範](./07-package-naming.md) 第 7.3 節
- **完整的實現範例？** → 閱讀 [08-完整代碼範例](./08-code-examples.md)
- **生產環境需要注意什麼？** → 閱讀 [09-生產環境保護措施](./09-production-safeguards.md)

---

## 實現原則

### 核心原則

1. **依賴規則** - 依賴只能指向內層（Infrastructure → Application → Domain）
2. **接口所有權** - 接口由使用者定義，而非實現者
3. **SOLID 原則** - 特別是 SRP（單一職責）和 DIP（依賴反轉）
4. **明確的邊界** - 清晰的包邊界，避免循環依賴
5. **測試友好** - 所有外部依賴可替換為 Mock

### Go 語言特性

1. **接口即合約** - 使用 Go 接口實現依賴反轉
2. **組合優於繼承** - 使用 struct 嵌入實現代碼復用
3. **錯誤處理** - 顯式錯誤返回，避免 panic
4. **並發安全** - 使用 Context 傳遞取消信號
5. **包可見性** - 使用小寫/大寫控制訪問權限

---

## 目錄結構總覽

```
bar_crm/
├── cmd/
│   ├── app/              # 主應用入口
│   │   └── main.go
│   └── migrate/          # 數據庫遷移工具
│       └── main.go
├── internal/             # 私有代碼（不可被外部 import）
│   ├── domain/           # 領域層（Domain Layer）
│   │   ├── member/       # 會員管理上下文
│   │   ├── points/       # 積分管理上下文（核心域）
│   │   ├── invoice/      # 發票處理上下文
│   │   ├── survey/       # 問卷管理上下文
│   │   ├── external/     # 外部系統整合上下文
│   │   ├── identity/     # 身份與訪問上下文
│   │   ├── notification/ # 通知服務上下文
│   │   ├── audit/        # 稽核追蹤上下文
│   │   └── shared/       # 共享的領域概念
│   ├── application/      # 應用層（Application Layer）
│   │   ├── usecases/     # Use Cases
│   │   ├── dto/          # Data Transfer Objects
│   │   └── events/       # Event Handlers
│   ├── infrastructure/   # 基礎設施層（Infrastructure Layer）
│   │   ├── persistence/  # GORM Repositories
│   │   ├── cache/        # Redis Cache
│   │   ├── external/     # 外部服務適配器
│   │   ├── events/       # Event Bus 實現
│   │   └── config/       # 配置管理
│   └── presentation/     # 展示層（Presentation Layer）
│       ├── http/         # HTTP Handlers
│       └── linebot/      # LINE Bot Handlers
├── test/                 # 測試代碼
│   ├── integration/      # 集成測試
│   ├── e2e/              # 端到端測試
│   └── fixtures/         # 測試數據
├── docs/                 # 文檔
│   ├── architecture/     # 架構設計
│   ├── product/          # 產品需求
│   ├── operations/       # 運維文檔
│   └── qa/               # 測試策略
├── scripts/              # 腳本工具
├── configs/              # 配置文件
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── docker-compose.yml
```

---

## 實現流程建議

### Phase 1: 基礎設施搭建（Week 1）
1. 創建項目目錄結構
2. 配置 Go Modules
3. 設置 GORM + PostgreSQL
4. 配置 Uber FX 依賴注入框架
5. 實現基礎的 HTTP Server（Gin）

### Phase 2: 核心域實現（Week 2-3）
1. 實現 Points Management Context（核心域）
   - Domain Layer: Aggregate + Value Objects
   - Application Layer: Use Cases
   - Infrastructure Layer: GORM Repositories
   - Presentation Layer: HTTP Handlers
2. 編寫單元測試與集成測試

### Phase 3: 支撐域實現（Week 4-5）
1. 實現 Member Management Context
2. 實現 Invoice Processing Context
3. 實現 Survey Management Context
4. 實現跨上下文的事件集成

### Phase 4: 外部集成（Week 6）
1. 實現 LINE Bot SDK 適配器
2. 實現 Google OAuth 適配器
3. 實現 iChef 匯入功能
4. 實現通知服務

### Phase 5: 生產就緒（Week 7-8）
1. 實現 Audit Context（稽核追蹤）
2. 完善錯誤處理與日誌
3. 添加監控與告警
4. 性能優化與壓力測試
5. 編寫部署文檔

---

## 測試策略

### 測試金字塔

```
        /\
       /E2E\         3% - 黑盒端到端測試
      /------\
     / Contr. \      5% - 契約測試（外部服務）
    /----------\
   /    Int.    \    15% - 集成測試（真實數據庫）
  /--------------\
 /     Unit      \   77% - 單元測試（快速、隔離）
/------------------\
```

### 各層測試重點

| 層級 | 測試類型 | Mock 策略 | 覆蓋率目標 |
|------|---------|----------|-----------|
| **Domain** | 單元測試 | 無 Mock（純邏輯） | 90%+ |
| **Application** | 單元測試 | Mock Repositories | 80%+ |
| **Infrastructure** | 集成測試 | SQLite in-memory | 70%+ |
| **Presentation** | 集成測試 | Mock Use Cases | 70%+ |
| **External Adapters** | 契約測試 | Mock 外部 API | 關鍵適配器 |
| **E2E** | 端到端測試 | 真實環境 | 關鍵流程 |

### 契約測試 (Contract Tests)

**目的**: 確保外部服務適配器正確處理 API 響應格式變更

**測試範例** (LINE Bot Adapter):

```go
// test/contract/linebot_adapter_test.go
package contract

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/yourorg/bar_crm/internal/infrastructure/external/linebot"
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

    // 創建模擬 HTTP 服務器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(mockResponse))
    }))
    defer server.Close()

    // 創建適配器（指向模擬服務器）
    adapter := linebot.NewLineUserAdapter(server.URL, "test-token")

    // Act: 調用適配器方法
    member, err := adapter.GetUserProfile("U1234567890")

    // Assert: 驗證適配器正確解析響應
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

// TestLineBotAdapter_GetProfile_APIBroken 測試 API 破壞性變更
func TestLineBotAdapter_GetProfile_APIBroken(t *testing.T) {
    // Arrange: 模擬 LINE API 移除了必要字段
    mockResponse := `{
        "userId": "U1234567890",
        "displayName": "Test User"
    }`

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(mockResponse))
    }))
    defer server.Close()

    adapter := linebot.NewLineUserAdapter(server.URL, "test-token")

    // Act & Assert: 適配器應該能夠優雅降級或返回錯誤
    member, err := adapter.GetUserProfile("U1234567890")
    if err != nil {
        t.Logf("Expected: Adapter handles missing fields gracefully")
    } else {
        assert.NotNil(t, member)
        t.Logf("Adapter uses default values for missing fields")
    }
}
```

**測試執行**:
```bash
# 運行契約測試
go test ./test/contract/... -v

# 在 CI/CD 中定期執行（檢測外部 API 變更）
go test ./test/contract/... -tags=contract -v
```

**契約測試的價值**:
1. **早期發現 API 變更**: 在外部服務更新時及時發現不兼容問題
2. **文檔化 API 依賴**: 測試本身就是對外部 API 的文檔
3. **向後兼容性驗證**: 確保適配器能處理 API 的演進
4. **減少生產事故**: 避免因外部 API 變更導致的運行時錯誤

**適用場景**:
- ✅ LINE Bot SDK (官方 API)
- ✅ Google OAuth2 (認證 API)
- ✅ iChef POS (Excel 格式變更偵測)
- ✅ 任何第三方 HTTP API

---

### 值對象單元測試

**目的**: 確保值對象的不變性約束、錯誤處理和業務邏輯正確性

**測試範例** (PointsAmount Value Object):

```go
// internal/domain/points/value_objects_test.go
package points_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/yourorg/bar_crm/internal/domain/points"
)

// --- 構造函數驗證測試 ---

func TestNewPointsAmount_ValidValue(t *testing.T) {
    // Arrange
    value := 100

    // Act
    amount, err := points.NewPointsAmount(value)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 100, amount.Value())
}

func TestNewPointsAmount_ZeroValue(t *testing.T) {
    // Arrange
    value := 0

    // Act
    amount, err := points.NewPointsAmount(value)

    // Assert: 0 是有效值
    assert.NoError(t, err)
    assert.Equal(t, 0, amount.Value())
    assert.True(t, amount.IsZero())
}

func TestNewPointsAmount_NegativeValue_ReturnsError(t *testing.T) {
    // Arrange
    value := -10

    // Act
    amount, err := points.NewPointsAmount(value)

    // Assert: 負數應該返回錯誤而非 panic
    assert.Error(t, err)
    assert.ErrorIs(t, err, points.ErrNegativePointsAmount)
    assert.Equal(t, 0, amount.Value()) // 零值對象
}

// --- 不變性測試 ---

func TestPointsAmount_Add_Immutability(t *testing.T) {
    // Arrange
    original, _ := points.NewPointsAmount(100)
    toAdd, _ := points.NewPointsAmount(50)

    // Act
    result := original.Add(toAdd)

    // Assert: 原始對象未改變（不可變性）
    assert.Equal(t, 100, original.Value())
    assert.Equal(t, 150, result.Value())
}

func TestPointsAmount_Subtract_Success(t *testing.T) {
    // Arrange
    minuend, _ := points.NewPointsAmount(100)
    subtrahend, _ := points.NewPointsAmount(30)

    // Act
    result, err := minuend.Subtract(subtrahend)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 70, result.Value())
    assert.Equal(t, 100, minuend.Value()) // 不可變性
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
    assert.Equal(t, 0, result.Value()) // 零值對象
}

// --- 相等性測試 ---

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

// --- 業務邏輯測試 (ConversionRate) ---

func TestConversionRate_CalculatePoints(t *testing.T) {
    tests := []struct {
        name           string
        conversionRate int
        amount         string // decimal string
        expectedPoints int
    }{
        {
            name:           "Standard conversion 100 TWD = 1 point",
            conversionRate: 100,
            amount:         "350.00",
            expectedPoints: 3, // floor(350/100) = 3
        },
        {
            name:           "Promotional rate 50 TWD = 1 point",
            conversionRate: 50,
            amount:         "125.00",
            expectedPoints: 2, // floor(125/50) = 2
        },
        {
            name:           "Fractional amount rounds down",
            conversionRate: 100,
            amount:         "99.99",
            expectedPoints: 0, // floor(99.99/100) = 0
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            rate, err := points.NewConversionRate(tt.conversionRate)
            assert.NoError(t, err)

            amount, err := decimal.NewFromString(tt.amount)
            assert.NoError(t, err)

            // Act
            result := rate.CalculatePoints(amount)

            // Assert
            assert.Equal(t, tt.expectedPoints, result.Value())
        })
    }
}

// --- 邊界值測試 ---

func TestConversionRate_Boundaries(t *testing.T) {
    tests := []struct {
        name        string
        value       int
        expectError bool
    }{
        {"Minimum valid rate", 1, false},
        {"Maximum valid rate", 1000, false},
        {"Below minimum", 0, true},
        {"Above maximum", 1001, true},
        {"Negative rate", -10, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Act
            rate, err := points.NewConversionRate(tt.value)

            // Assert
            if tt.expectError {
                assert.Error(t, err)
                assert.ErrorIs(t, err, points.ErrInvalidConversionRate)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.value, rate.Value())
            }
        })
    }
}
```

**測試覆蓋的關鍵點**:
1. ✅ **構造函數驗證**: 測試有效和無效輸入
2. ✅ **錯誤處理**: 使用 `assert.ErrorIs()` 驗證特定錯誤類型
3. ✅ **不變性**: 驗證操作不改變原始對象
4. ✅ **值相等性**: 測試基於值的相等判斷
5. ✅ **業務邏輯**: 測試封裝的計算邏輯（如積分轉換）
6. ✅ **邊界值**: 測試有效範圍的邊界情況

**測試執行**:
```bash
# 運行值對象測試
go test ./internal/domain/points -v -run TestPointsAmount
go test ./internal/domain/points -v -run TestConversionRate

# 檢查測試覆蓋率
go test ./internal/domain/points -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**值對象測試的設計原則**:
- **快速執行**: 值對象測試無 I/O，應在毫秒內完成
- **完全隔離**: 無需 Mock，直接測試純邏輯
- **高覆蓋率**: 目標 90%+ 代碼覆蓋率
- **表格驅動**: 使用 table-driven tests 覆蓋多種場景

---

## 錯誤處理策略

### 錯誤分層原則

**1. Domain Layer - 定義業務錯誤**
```go
// internal/domain/points/errors.go
var (
    ErrInsufficientPoints = errors.New("insufficient points")
    ErrInvalidMemberID    = errors.New("invalid member ID")
    ErrNegativeAmount     = errors.New("negative amount not allowed")
)
```

**2. Infrastructure Layer - 轉換技術錯誤**
```go
// internal/infrastructure/persistence/points/account_repository.go
func (r *GormPointsAccountRepository) FindByID(...) (*points.PointsAccount, error) {
    // ...
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, repository.ErrAccountNotFound  // 轉換為 Domain 錯誤
    }
    return nil, fmt.Errorf("database error: %w", err)  // 包裝技術錯誤
}
```

**3. Application Layer - 透傳 Domain 錯誤**
```go
// internal/application/usecases/points/earn_points.go
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) (*EarnPointsResult, error) {
    err := uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        account, err := uc.accountRepo.FindByMemberID(ctx, memberID)
        if err != nil {
            return err  // 透傳 Domain 錯誤
        }
        return account.EarnPoints(...)
    })
    return result, err
}
```

**4. Presentation Layer - 映射 HTTP 狀態碼**
```go
// internal/presentation/http/handlers/points_handler.go
func (h *PointsHandler) HandleEarnPoints(c *gin.Context) {
    result, err := h.earnPointsUseCase.Execute(cmd)
    if err != nil {
        // 根據錯誤類型返回不同狀態碼
        switch {
        case errors.Is(err, points.ErrInsufficientPoints):
            responses.Error(c, http.StatusBadRequest, "Insufficient points", err)
        case errors.Is(err, repository.ErrAccountNotFound):
            responses.Error(c, http.StatusNotFound, "Account not found", err)
        default:
            responses.Error(c, http.StatusInternalServerError, "Internal error", err)
        }
        return
    }
    responses.Success(c, result)
}
```

### 錯誤檢查最佳實踐

**使用 errors.Is 和 errors.As**:
```go
// ✅ 正確：使用 errors.Is 檢查錯誤類型
if errors.Is(err, points.ErrInsufficientPoints) {
    // 處理積分不足錯誤
}

// ✅ 正確：使用 errors.As 提取特定錯誤類型
var validationErr *ValidationError
if errors.As(err, &validationErr) {
    // 處理驗證錯誤
}

// ❌ 錯誤：直接比較錯誤（不支持錯誤包裝）
if err == points.ErrInsufficientPoints {
    // 如果錯誤被 fmt.Errorf("%w", err) 包裝過，這將失敗
}
```

### Panic vs Error 使用時機

**何時使用 error（業務錯誤）**:
```go
// ✅ 業務規則違反 - 返回 error
func (a *PointsAccount) DeductPoints(amount PointsAmount) error {
    if !a.HasSufficientPoints(amount) {
        return ErrInsufficientPoints  // 用戶輸入錯誤，可恢復
    }
    // ...
}

// ✅ 外部依賴失敗 - 返回 error
func (r *Repository) FindByID(id string) (*Entity, error) {
    entity, err := r.db.Query(...)
    if err != nil {
        return nil, fmt.Errorf("database error: %w", err)  // 網絡錯誤，可重試
    }
    // ...
}
```

**何時使用 panic（程序錯誤）**:
```go
// ✅ 不變條件違反 - panic（數據損壞或邏輯錯誤）
func (a *PointsAccount) GetAvailablePoints() PointsAmount {
    if a.usedPoints.Value() > a.earnedPoints.Value() {
        // 不變條件被違反：這不應該發生，必須立即暴露
        panic(fmt.Sprintf("invariant violation: used (%d) > earned (%d)",
            a.usedPoints.Value(), a.earnedPoints.Value()))
    }
    return a.earnedPoints.subtractUnchecked(a.usedPoints)
}

// ✅ 配置錯誤 - panic（啟動時檢查）
func NewService(config Config) *Service {
    if config.DatabaseURL == "" {
        panic("DATABASE_URL is required")  // 配置錯誤，無法啟動
    }
    // ...
}

// ❌ 絕不使用：靜默截斷（掩蓋錯誤）
func (p PointsAmount) subtract(other PointsAmount) PointsAmount {
    result := p.value - other.value
    if result < 0 {
        return PointsAmount{value: 0}  // ❌ 掩蓋了數據損壞！
    }
    return PointsAmount{value: result}
}
```

**關鍵原則**:
- **業務錯誤（可預期）→ 返回 error**：用戶輸入錯誤、外部服務失敗、資源不存在
- **程序錯誤（不應發生）→ panic**：不變條件違反、配置錯誤、邏輯錯誤
- **Fail Fast 原則**：錯誤應該立即暴露，而非靜默處理
- **生產環境**：使用 `recover()` 在頂層捕獲 panic，記錄日誌並告警

### 錯誤日誌記錄

**分層日誌策略**:
```go
// Infrastructure Layer - 記錄技術錯誤詳情
func (r *GormPointsAccountRepository) Update(...) error {
    result := db.Updates(...)
    if result.Error != nil {
        logger.Error("Failed to update points account",
            zap.String("accountID", accountID),
            zap.Error(result.Error),
        )
        return fmt.Errorf("database error: %w", result.Error)
    }
}

// Application Layer - 記錄業務操作失敗
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) (*EarnPointsResult, error) {
    err := uc.txManager.InTransaction(...)
    if err != nil {
        logger.Warn("Failed to earn points",
            zap.String("memberID", cmd.MemberID),
            zap.Error(err),
        )
        return nil, err
    }
}

// Presentation Layer - 記錄 HTTP 請求錯誤
func (h *PointsHandler) HandleEarnPoints(c *gin.Context) {
    result, err := h.earnPointsUseCase.Execute(cmd)
    if err != nil {
        logger.Info("HTTP request failed",
            zap.String("path", c.Request.URL.Path),
            zap.String("method", c.Request.Method),
            zap.Error(err),
        )
    }
}
```

---

## 常見問題 (FAQ)

### Q1: 為什麼使用 internal/ 目錄？
**A**: Go 的 `internal/` 目錄是語言級別的可見性控制，防止外部包 import 內部代碼，確保 API 邊界清晰。

### Q2: Domain Layer 可以依賴 Application Layer 的 DTO 嗎？
**A**: 不可以直接依賴。Domain Layer 應該定義接口（如 `PointsCalculableTransaction`），由 Application Layer 的 DTO 實現。詳見 [02-Domain Layer 實現指南](./02-domain-layer-implementation.md) 第 2.10 節。

### Q3: 如何避免 Repository 洩漏 GORM 模型到 Domain Layer？
**A**: Repository 在 Infrastructure Layer 進行 GORM Model ↔ Domain Entity 的轉換，使用 Domain Layer 提供的 `Reconstruct*` 方法重建聚合。詳見 [04-Infrastructure Layer 實現指南](./04-infrastructure-layer-implementation.md) 第 4.2 節。

### Q4: 事務管理應該放在哪一層？
**A**: Application Layer 使用 Transaction Context Pattern 管理事務。詳見 [03-Application Layer 實現指南](./03-application-layer-implementation.md) 第 3.4 節。

### Q5: 如何處理跨上下文的數據查詢？
**A**: 使用 DTO + Application Layer 協調。避免 Domain Layer 直接引用其他上下文的實體。詳見 [03-Application Layer 實現指南](./03-application-layer-implementation.md) 第 3.5 節。

### Q6: 領域事件應該如何實現？
**A**: Domain Layer 收集事件，Application Layer 在事務提交後發布事件。詳見 [02-Domain Layer 實現指南](./02-domain-layer-implementation.md) 第 6 節 和 DDD 文檔的 [14-事件處理實作指南](../ddd/14-event-handling-implementation.md)。

### Q7: 如何處理錯誤傳播？
**A**: Domain 定義業務錯誤，Infrastructure 轉換技術錯誤，Application 透傳，Presentation 映射 HTTP 狀態碼。使用 `errors.Is` 和 `errors.As` 進行錯誤檢查。詳見上方「錯誤處理策略」章節。

---

## 參考資料

### DDD 架構設計文檔
- [DDD 指南總覽](../ddd/README.md)
- [限界上下文劃分](../ddd/02-bounded-contexts.md)
- [分層架構設計](../ddd/06-layered-architecture.md)
- [依賴規則](../ddd/11-dependency-rules.md)
- [聚合設計原則](../ddd/07-aggregate-design-principles.md)

### Go 語言最佳實踐
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Package Layout](https://github.com/golang-standards/project-layout)

### Clean Architecture
- Robert C. Martin - "Clean Architecture: A Craftsman's Guide to Software Structure and Design"
- [The Clean Architecture Blog Post](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

---

## 維護指南

### 文檔更新原則
1. **架構設計變更** → 先更新 DDD 文檔（設計層），再更新實現指南（技術層）
2. **新增上下文** → 按照現有模式添加對應章節
3. **代碼範例** → 保持與實際代碼同步
4. **版本管理** → 使用 ADR 記錄重大決策

### 文檔所有權
- **DDD 文檔**（ddd/）: 架構師負責
- **實現指南**（implementation/）: 技術負責人負責
- **代碼實現**（internal/）: 開發團隊負責

---

**最後更新**: 2025-01-10
**維護者**: 開發團隊
**審核者**: 架構師
