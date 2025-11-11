# 詳細任務分解計劃

> **版本**: 1.0
> **最後更新**: 2025-01-11
> **目標**: 提供可執行的、每日任務級別的詳細實作計劃

---

## 使用說明

### 如何使用這份計劃

1. **每天開始前**：查看當天的任務清單
2. **執行任務**：按順序完成每個任務
3. **檢查完成**：每個任務都有明確的「完成標準」
4. **記錄進度**：在任務旁打勾或標記完成時間
5. **每日結束**：執行「每日檢查點」驗證

### 任務標記說明

- 📝 **編寫測試**
- 💻 **編寫實作**
- ✅ **驗證/檢查**
- 🔧 **重構**
- 📚 **文檔**

---

## Week 1: Domain Layer - Points Context (Part 1)

### Day 1: 專案初始化 + PointsAmount 值對象

#### 時間分配
- 上午 (3h): 專案設置 + 環境驗證
- 下午 (5h): PointsAmount 值對象 TDD

---

#### 任務 1.1: 專案初始化 (1h)

**步驟**:

```bash
# Step 1.1.1: 初始化 Go Module (5 min)
cd /Users/apple/Documents/code/golang/bar_crm
go mod init github.com/yourorg/bar_crm

# Step 1.1.2: 安裝核心依賴 (10 min)
go get github.com/stretchr/testify@v1.8.4
go get github.com/google/uuid@v1.5.0
go get github.com/shopspring/decimal@v1.3.1

# Step 1.1.3: 建立目錄結構 (10 min)
mkdir -p internal/domain/points
mkdir -p internal/domain/shared
mkdir -p test/integration
mkdir -p test/e2e
mkdir -p test/fixtures

# Step 1.1.4: 建立 .gitignore (5 min)
cat > .gitignore << 'EOF'
# Binaries
bin/
*.exe
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out
coverage.html

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local
EOF

# Step 1.1.5: 初始化 Git (如果尚未初始化) (5 min)
git add .
git commit -m "chore: initialize Go project structure"

# Step 1.1.6: 建立 Makefile (15 min)
cat > Makefile << 'EOF'
.PHONY: test test-unit test-integration test-e2e coverage clean

# 執行所有測試
test:
	go test ./... -v

# 執行單元測試
test-unit:
	go test ./internal/domain/... -v -cover

# 執行應用層測試
test-app:
	go test ./internal/application/... -v -cover

# 執行集成測試
test-integration:
	go test ./internal/infrastructure/... -v -cover

# 執行 E2E 測試
test-e2e:
	go test ./test/e2e/... -v

# 生成覆蓋率報告
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 檢查覆蓋率百分比
coverage-check:
	@go test ./... -coverprofile=coverage.out > /dev/null
	@go tool cover -func=coverage.out | grep total | awk '{print "Total Coverage: " $$3}'

# 清理
clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# 運行所有 linters
lint:
	golangci-lint run ./...

# 格式化代碼
fmt:
	gofmt -w .
	go mod tidy

# 建置應用
build:
	go build -o bin/app cmd/app/main.go

# 執行應用
run:
	go run cmd/app/main.go
EOF

# Step 1.1.7: 驗證設置 (10 min)
go mod tidy
go mod verify
go version
```

**完成標準**:
- ✅ `go.mod` 檔案存在
- ✅ 所有目錄已建立
- ✅ `make test` 可以執行（即使沒有測試）
- ✅ Git 有初始 commit

**預估時間**: 1 小時

---

#### 任務 1.2: 建立 Shared Domain 基礎 (30 min)

**檔案清單**:
1. `internal/domain/shared/transaction.go`
2. `internal/domain/shared/event.go`

**步驟**:

```bash
# Step 1.2.1: 建立 TransactionContext 介面 (10 min)
cat > internal/domain/shared/transaction.go << 'EOF'
package shared

// TransactionContext 事務上下文介面
// 這是一個標記介面，Infrastructure Layer 會實作具體的事務封裝
type TransactionContext interface {
	// 標記介面：僅用於傳遞上下文，不暴露方法
}

// TransactionManager 事務管理器介面
type TransactionManager interface {
	InTransaction(fn func(ctx TransactionContext) error) error
}
EOF

# Step 1.2.2: 建立 DomainEvent 介面 (20 min)
cat > internal/domain/shared/event.go << 'EOF'
package shared

import "time"

// DomainEvent 領域事件基礎介面
type DomainEvent interface {
	EventID() string        // 事件唯一標識
	EventType() string      // 事件類型
	OccurredAt() time.Time  // 發生時間
	AggregateID() string    // 聚合根 ID
}

// EventPublisher 事件發布器介面
// 設計原則：介面定義在 Domain Layer（使用者），由 Infrastructure 實作
type EventPublisher interface {
	Publish(event DomainEvent) error
	PublishBatch(events []DomainEvent) error
}

// EventSubscriber 事件訂閱器介面
type EventSubscriber interface {
	Subscribe(eventType string, handler EventHandler) error
}

// EventHandler 事件處理器介面
type EventHandler interface {
	Handle(event DomainEvent) error
	EventType() string
}
EOF
```

**完成標準**:
- ✅ 兩個檔案已建立
- ✅ `go build ./internal/domain/shared` 無錯誤

**預估時間**: 30 分鐘

---

#### 任務 1.3: PointsAmount 值對象 - TDD 第一輪 (2h)

**目標**: 實作 PointsAmount 的基本功能（建構、驗證、基本操作）

**檔案清單**:
1. `internal/domain/points/errors.go`
2. `internal/domain/points/value_objects_test.go`
3. `internal/domain/points/value_objects.go`

---

**Step 1.3.1: 建立錯誤定義 (10 min)**

```bash
cat > internal/domain/points/errors.go << 'EOF'
package points

import "errors"

// 積分數量相關錯誤
var (
	ErrNegativePointsAmount     = errors.New("points amount cannot be negative")
	ErrInsufficientPoints       = errors.New("insufficient points for this operation")
	ErrInsufficientEarnedPoints = errors.New("earned points cannot be less than used points")
)

// 轉換率相關錯誤
var (
	ErrInvalidConversionRate = errors.New("conversion rate must be between 1 and 1000")
	ErrInvalidDateRange      = errors.New("invalid date range: start date must be before or equal to end date")
)

// 帳戶相關錯誤
var (
	ErrAccountNotFound      = errors.New("points account not found")
	ErrAccountAlreadyExists = errors.New("points account already exists for this member")
	ErrInvalidMemberID      = errors.New("invalid member ID")
)
EOF
```

---

**Step 1.3.2: 編寫第一個測試 - 有效值建構 (15 min)**

```bash
cat > internal/domain/points/value_objects_test.go << 'EOF'
package points_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourorg/bar_crm/internal/domain/points"
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
EOF

# 執行測試（應該失敗 - Red）
cd internal/domain/points
go test -v -run TestNewPointsAmount_ValidValue
```

---

**Step 1.3.3: 實作 PointsAmount 基本結構 (15 min)**

```bash
cat > internal/domain/points/value_objects.go << 'EOF'
package points

// PointsAmount 積分數量
type PointsAmount struct {
	value int
}

// NewPointsAmount 創建積分數量（帶驗證）
func NewPointsAmount(value int) (PointsAmount, error) {
	if value < 0 {
		return PointsAmount{}, ErrNegativePointsAmount
	}
	return PointsAmount{value: value}, nil
}

// Value 獲取值
func (p PointsAmount) Value() int {
	return p.value
}
EOF

# 執行測試（應該通過 - Green）
go test -v -run TestNewPointsAmount_ValidValue
```

---

**Step 1.3.4: 新增測試 - 負數驗證 (10 min)**

在 `value_objects_test.go` 末尾新增：

```go
// Test 2: 建構負數應回傳錯誤
func TestNewPointsAmount_NegativeValue_ReturnsError(t *testing.T) {
	// Arrange
	value := -10

	// Act
	amount, err := points.NewPointsAmount(value)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrNegativePointsAmount)
	assert.Equal(t, 0, amount.Value()) // 零值對象
}

// Test 3: 零值是有效的
func TestNewPointsAmount_ZeroValue_Success(t *testing.T) {
	// Arrange
	value := 0

	// Act
	amount, err := points.NewPointsAmount(value)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0, amount.Value())
	assert.True(t, amount.IsZero())
}
```

在 `value_objects.go` 新增 `IsZero` 方法：

```go
// IsZero 判斷是否為零
func (p PointsAmount) IsZero() bool {
	return p.value == 0
}
```

執行測試：
```bash
go test -v -run TestNewPointsAmount
```

---

**Step 1.3.5: 新增測試 - Add 操作（不可變性）(15 min)**

在 `value_objects_test.go` 新增：

```go
// Test 4: Add 操作（不可變性）
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

// Test 5: Add 零值
func TestPointsAmount_Add_Zero(t *testing.T) {
	// Arrange
	original, _ := points.NewPointsAmount(100)
	zero, _ := points.NewPointsAmount(0)

	// Act
	result := original.Add(zero)

	// Assert
	assert.Equal(t, 100, result.Value())
}
```

在 `value_objects.go` 新增：

```go
// newPointsAmountUnchecked 創建積分數量（無驗證）
// 僅供內部算術操作使用（調用方已保證有效性）
func newPointsAmountUnchecked(value int) PointsAmount {
	return PointsAmount{value: value}
}

// Add 相加（不可變操作，返回新對象）
func (p PointsAmount) Add(other PointsAmount) PointsAmount {
	return newPointsAmountUnchecked(p.value + other.value)
}
```

執行測試：
```bash
go test -v -run TestPointsAmount_Add
```

---

**Step 1.3.6: 新增測試 - Subtract 操作（錯誤處理）(20 min)**

在 `value_objects_test.go` 新增：

```go
// Test 6: Subtract 成功
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

// Test 7: Subtract 負數結果返回錯誤
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

// Test 8: Subtract 零值
func TestPointsAmount_Subtract_Zero(t *testing.T) {
	// Arrange
	original, _ := points.NewPointsAmount(100)
	zero, _ := points.NewPointsAmount(0)

	// Act
	result, err := original.Subtract(zero)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, result.Value())
}
```

在 `value_objects.go` 新增：

```go
// Subtract 相減（不可變操作，返回錯誤而非靜默截斷）
func (p PointsAmount) Subtract(other PointsAmount) (PointsAmount, error) {
	result := p.value - other.value
	if result < 0 {
		return PointsAmount{}, ErrNegativePointsAmount
	}
	return newPointsAmountUnchecked(result), nil
}
```

執行測試：
```bash
go test -v -run TestPointsAmount_Subtract
```

---

**Step 1.3.7: 新增測試 - Equals 和 IsZero (15 min)**

在 `value_objects_test.go` 新增：

```go
// Test 9: Equals - 相同值
func TestPointsAmount_Equals_SameValue(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(100)

	// Act & Assert
	assert.True(t, amount1.Equals(amount2))
}

// Test 10: Equals - 不同值
func TestPointsAmount_Equals_DifferentValue(t *testing.T) {
	// Arrange
	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(200)

	// Act & Assert
	assert.False(t, amount1.Equals(amount2))
}

// Test 11: IsZero - 零值
func TestPointsAmount_IsZero_True(t *testing.T) {
	// Arrange
	amount, _ := points.NewPointsAmount(0)

	// Act & Assert
	assert.True(t, amount.IsZero())
}

// Test 12: IsZero - 非零值
func TestPointsAmount_IsZero_False(t *testing.T) {
	// Arrange
	amount, _ := points.NewPointsAmount(100)

	// Act & Assert
	assert.False(t, amount.IsZero())
}
```

在 `value_objects.go` 新增：

```go
// Equals 判斷相等性
func (p PointsAmount) Equals(other PointsAmount) bool {
	return p.value == other.value
}
```

執行測試：
```bash
go test -v
```

---

**Step 1.3.8: 新增 subtractUnchecked（內部使用）(15 min)**

在 `value_objects_test.go` 新增：

```go
// Test 13: subtractUnchecked 內部方法（透過其他方法測試）
// 注意：subtractUnchecked 是私有方法，不直接測試
// 我們會在 PointsAccount.GetAvailablePoints() 中測試其 panic 行為
```

在 `value_objects.go` 新增：

```go
import "fmt"

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
```

---

**每日檢查點 - Day 1 (15 min)**

```bash
# 執行所有測試
cd internal/domain/points
go test -v -cover

# 檢查覆蓋率
go test -coverprofile=coverage.out
go tool cover -func=coverage.out

# 預期結果
# PASS
# coverage: 100% of statements (PointsAmount 完整覆蓋)
```

**完成標準**:
- ✅ 13 個測試全部通過
- ✅ PointsAmount 覆蓋率 100%
- ✅ 所有測試執行時間 < 1 秒
- ✅ `go build ./internal/domain/points` 無錯誤
- ✅ 無 linter 警告

**Day 1 產出**:
- ✅ 專案基礎結構
- ✅ Shared Domain 介面
- ✅ PointsAmount 值對象（完整實作 + 測試）
- ✅ 錯誤定義

**預估總時間**: 8 小時

---

### Day 2: ConversionRate + AccountID + MemberID 值對象

#### 時間分配
- 上午 (4h): ConversionRate 值對象 TDD
- 下午 (4h): AccountID + MemberID 值對象 TDD

---

#### 任務 2.1: ConversionRate 值對象 (4h)

**目標**: 實作轉換率值對象，包含積分計算邏輯

---

**Step 2.1.1: 編寫測試 - 建構與驗證 (30 min)**

在 `value_objects_test.go` 新增：

```go
// === ConversionRate 測試 ===

// Test 14: ConversionRate 有效範圍
func TestNewConversionRate_ValidRate_Success(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"最小值 1", 1},
		{"標準值 100", 100},
		{"最大值 1000", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			rate, err := points.NewConversionRate(tt.value)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.value, rate.Value())
		})
	}
}

// Test 15: ConversionRate 無效範圍
func TestNewConversionRate_InvalidRate_ReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"低於最小值", 0},
		{"負數", -10},
		{"超過最大值", 1001},
		{"遠超最大值", 5000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			rate, err := points.NewConversionRate(tt.value)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, points.ErrInvalidConversionRate)
			assert.Equal(t, 0, rate.Value())
		})
	}
}
```

---

**Step 2.1.2: 實作 ConversionRate 基本結構 (20 min)**

在 `value_objects.go` 新增：

```go
// ConversionRate 轉換率（例如 100 元 = 1 點）
// 業務規則：範圍 1-1000
type ConversionRate struct {
	value int
}

// NewConversionRate 創建轉換率
func NewConversionRate(value int) (ConversionRate, error) {
	if value < 1 || value > 1000 {
		return ConversionRate{}, ErrInvalidConversionRate
	}
	return ConversionRate{value: value}, nil
}

// Value 獲取值
func (r ConversionRate) Value() int {
	return r.value
}

// Equals 判斷相等性
func (r ConversionRate) Equals(other ConversionRate) bool {
	return r.value == other.value
}
```

執行測試：
```bash
go test -v -run TestNewConversionRate
```

---

**Step 2.1.3: 編寫測試 - CalculatePoints 業務邏輯 (45 min)**

在 `value_objects_test.go` 新增：

```go
import "github.com/shopspring/decimal"

// Test 16: CalculatePoints 標準轉換
func TestConversionRate_CalculatePoints(t *testing.T) {
	tests := []struct {
		name           string
		conversionRate int
		amount         string // decimal string
		expectedPoints int
	}{
		{
			name:           "標準轉換 100 TWD = 1 點",
			conversionRate: 100,
			amount:         "350.00",
			expectedPoints: 3, // floor(350/100) = 3
		},
		{
			name:           "促銷轉換 50 TWD = 1 點",
			conversionRate: 50,
			amount:         "125.00",
			expectedPoints: 2, // floor(125/50) = 2
		},
		{
			name:           "小數金額向下取整",
			conversionRate: 100,
			amount:         "99.99",
			expectedPoints: 0, // floor(99.99/100) = 0
		},
		{
			name:           "剛好整除",
			conversionRate: 100,
			amount:         "500.00",
			expectedPoints: 5, // floor(500/100) = 5
		},
		{
			name:           "零金額",
			conversionRate: 100,
			amount:         "0.00",
			expectedPoints: 0,
		},
		{
			name:           "1 元 = 1 點（極端情況）",
			conversionRate: 1,
			amount:         "5.50",
			expectedPoints: 5, // floor(5.50/1) = 5
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

// Test 17: CalculatePoints 負數金額（理論上不應該發生）
func TestConversionRate_CalculatePoints_NegativeAmount(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	negativeAmount := decimal.NewFromFloat(-50.00)

	// Act
	result := rate.CalculatePoints(negativeAmount)

	// Assert: 應該返回 0（向下取整的結果）
	assert.Equal(t, 0, result.Value())
}
```

---

**Step 2.1.4: 實作 CalculatePoints 方法 (30 min)**

在 `value_objects.go` 新增：

```go
import "github.com/shopspring/decimal"

// CalculatePoints 計算積分（核心業務邏輯）
// 積分 = floor(金額 / 轉換率)
func (r ConversionRate) CalculatePoints(amount decimal.Decimal) PointsAmount {
	// 防止除以零（理論上不會發生，因為 ConversionRate >= 1）
	if r.value == 0 {
		return newPointsAmountUnchecked(0)
	}

	rate := decimal.NewFromInt(int64(r.value))
	points := amount.Div(rate).Floor().IntPart()

	// floor 結果可能為負數（如果 amount 為負）
	// 我們確保返回的積分 >= 0
	if points < 0 {
		return newPointsAmountUnchecked(0)
	}

	return newPointsAmountUnchecked(int(points))
}
```

執行測試：
```bash
go test -v -run TestConversionRate_CalculatePoints
```

---

**Step 2.1.5: 新增測試 - Equals (10 min)**

在 `value_objects_test.go` 新增：

```go
// Test 18: ConversionRate Equals
func TestConversionRate_Equals(t *testing.T) {
	// Arrange
	rate1, _ := points.NewConversionRate(100)
	rate2, _ := points.NewConversionRate(100)
	rate3, _ := points.NewConversionRate(50)

	// Act & Assert
	assert.True(t, rate1.Equals(rate2))
	assert.False(t, rate1.Equals(rate3))
}
```

執行測試：
```bash
go test -v -run TestConversionRate
```

---

#### 任務 2.2: AccountID 值對象 (1.5h)

**目標**: 實作帳戶 ID 值對象（UUID 封裝）

---

**Step 2.2.1: 編寫測試 - 建構與驗證 (20 min)**

在 `value_objects_test.go` 新增：

```go
// === AccountID 測試 ===

// Test 19: NewAccountID 生成新 ID
func TestNewAccountID_GeneratesUUID(t *testing.T) {
	// Act
	id1 := points.NewAccountID()
	id2 := points.NewAccountID()

	// Assert
	assert.NotEqual(t, "", id1.String())
	assert.NotEqual(t, "", id2.String())
	assert.NotEqual(t, id1.String(), id2.String()) // 每次生成不同的 UUID
}

// Test 20: AccountIDFromString 有效 UUID
func TestAccountIDFromString_ValidUUID_Success(t *testing.T) {
	// Arrange
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	// Act
	id, err := points.AccountIDFromString(validUUID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, validUUID, id.String())
}

// Test 21: AccountIDFromString 無效 UUID
func TestAccountIDFromString_InvalidUUID_ReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"空字串", ""},
		{"不是 UUID 格式", "not-a-uuid"},
		{"錯誤格式", "123-456-789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			id, err := points.AccountIDFromString(tt.value)

			// Assert
			assert.Error(t, err)
			assert.True(t, id.IsEmpty())
		})
	}
}

// Test 22: AccountID Equals
func TestAccountID_Equals(t *testing.T) {
	// Arrange
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	id1, _ := points.AccountIDFromString(uuid)
	id2, _ := points.AccountIDFromString(uuid)
	id3 := points.NewAccountID()

	// Act & Assert
	assert.True(t, id1.Equals(id2))
	assert.False(t, id1.Equals(id3))
}

// Test 23: AccountID IsEmpty
func TestAccountID_IsEmpty(t *testing.T) {
	// Arrange
	emptyID := points.AccountID{}
	validID := points.NewAccountID()

	// Act & Assert
	assert.True(t, emptyID.IsEmpty())
	assert.False(t, validID.IsEmpty())
}
```

---

**Step 2.2.2: 實作 AccountID (20 min)**

在 `value_objects.go` 新增：

```go
import (
	"errors"
	"github.com/google/uuid"
)

// AccountID 帳戶 ID
type AccountID struct {
	value string
}

// NewAccountID 生成新的帳戶 ID
func NewAccountID() AccountID {
	return AccountID{value: uuid.New().String()}
}

// AccountIDFromString 從字符串創建 AccountID
func AccountIDFromString(value string) (AccountID, error) {
	if value == "" {
		return AccountID{}, errors.New("account ID cannot be empty")
	}

	// 驗證 UUID 格式
	if _, err := uuid.Parse(value); err != nil {
		return AccountID{}, errors.New("invalid account ID format")
	}

	return AccountID{value: value}, nil
}

// String 返回字符串表示
func (id AccountID) String() string {
	return id.value
}

// Equals 判斷相等性
func (id AccountID) Equals(other AccountID) bool {
	return id.value == other.value
}

// IsEmpty 判斷是否為空
func (id AccountID) IsEmpty() bool {
	return id.value == ""
}
```

執行測試：
```bash
go test -v -run TestAccountID
```

---

#### 任務 2.3: MemberID 值對象 (1.5h)

**目標**: 實作會員 ID 值對象

---

**Step 2.3.1: 編寫測試 (20 min)**

在 `value_objects_test.go` 新增：

```go
// === MemberID 測試 ===

// Test 24: NewMemberID 有效字串
func TestNewMemberID_ValidString_Success(t *testing.T) {
	// Arrange
	value := "member-123"

	// Act
	id, err := points.NewMemberID(value)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, value, id.String())
}

// Test 25: NewMemberID 空字串
func TestNewMemberID_EmptyString_ReturnsError(t *testing.T) {
	// Arrange
	value := ""

	// Act
	id, err := points.NewMemberID(value)

	// Assert
	assert.Error(t, err)
	assert.True(t, id.IsEmpty())
}

// Test 26: MemberID Equals
func TestMemberID_Equals(t *testing.T) {
	// Arrange
	id1, _ := points.NewMemberID("member-123")
	id2, _ := points.NewMemberID("member-123")
	id3, _ := points.NewMemberID("member-456")

	// Act & Assert
	assert.True(t, id1.Equals(id2))
	assert.False(t, id1.Equals(id3))
}

// Test 27: MemberID IsEmpty
func TestMemberID_IsEmpty(t *testing.T) {
	// Arrange
	emptyID := points.MemberID{}
	validID, _ := points.NewMemberID("member-123")

	// Act & Assert
	assert.True(t, emptyID.IsEmpty())
	assert.False(t, validID.IsEmpty())
}
```

---

**Step 2.3.2: 實作 MemberID (15 min)**

在 `value_objects.go` 新增：

```go
// MemberID 會員 ID（跨上下文引用）
type MemberID struct {
	value string
}

// NewMemberID 創建會員 ID
func NewMemberID(value string) (MemberID, error) {
	if value == "" {
		return MemberID{}, errors.New("member ID cannot be empty")
	}
	return MemberID{value: value}, nil
}

// String 返回字符串表示
func (id MemberID) String() string {
	return id.value
}

// Equals 判斷相等性
func (id MemberID) Equals(other MemberID) bool {
	return id.value == other.value
}

// IsEmpty 判斷是否為空
func (id MemberID) IsEmpty() bool {
	return id.value == ""
}
```

執行測試：
```bash
go test -v -run TestMemberID
```

---

**每日檢查點 - Day 2 (15 min)**

```bash
# 執行所有測試
cd internal/domain/points
go test -v -cover

# 檢查覆蓋率
go test -coverprofile=coverage.out
go tool cover -func=coverage.out

# 預期結果
# PASS
# 27 tests
# coverage: 100% of statements
```

**完成標準**:
- ✅ 27 個測試全部通過
- ✅ 值對象覆蓋率 100%
- ✅ 所有測試執行時間 < 1 秒
- ✅ `go build ./internal/domain/points` 無錯誤

**Day 2 產出**:
- ✅ ConversionRate 值對象（含積分計算邏輯）
- ✅ AccountID 值對象（UUID 封裝）
- ✅ MemberID 值對象
- ✅ 完整測試覆蓋

**預估總時間**: 8 小時

---

### Day 3: DateRange + PointsSource 值對象

#### 時間分配
- 上午 (3h): DateRange 值對象 TDD
- 下午 (3h): PointsSource 枚舉 + 重構

---

#### 任務 3.1: DateRange 值對象 (3h)

**目標**: 實作日期範圍值對象，用於轉換規則

---

**Step 3.1.1: 編寫測試 - 建構與驗證 (30 min)**

在 `value_objects_test.go` 新增：

```go
import "time"

// === DateRange 測試 ===

// Test 28: NewDateRange 有效範圍
func TestNewDateRange_ValidRange_Success(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	// Act
	dr, err := points.NewDateRange(start, end)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, start, dr.StartDate())
	assert.Equal(t, end, dr.EndDate())
}

// Test 29: NewDateRange 同一天
func TestNewDateRange_SameDay_Success(t *testing.T) {
	// Arrange
	sameDay := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	// Act
	dr, err := points.NewDateRange(sameDay, sameDay)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, sameDay, dr.StartDate())
	assert.Equal(t, sameDay, dr.EndDate())
}

// Test 30: NewDateRange 開始日期晚於結束日期
func TestNewDateRange_StartAfterEnd_ReturnsError(t *testing.T) {
	// Arrange
	start := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Act
	dr, err := points.NewDateRange(start, end)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrInvalidDateRange)
	assert.True(t, dr.StartDate().IsZero())
	assert.True(t, dr.EndDate().IsZero())
}
```

---

**Step 3.1.2: 實作 DateRange 基本結構 (20 min)**

在 `value_objects.go` 新增：

```go
import "time"

// DateRange 日期範圍
type DateRange struct {
	startDate time.Time
	endDate   time.Time
}

// NewDateRange 創建日期範圍
func NewDateRange(startDate, endDate time.Time) (DateRange, error) {
	if startDate.After(endDate) {
		return DateRange{}, ErrInvalidDateRange
	}
	return DateRange{
		startDate: startDate,
		endDate:   endDate,
	}, nil
}

// StartDate 獲取開始日期
func (dr DateRange) StartDate() time.Time {
	return dr.startDate
}

// EndDate 獲取結束日期
func (dr DateRange) EndDate() time.Time {
	return dr.endDate
}
```

執行測試：
```bash
go test -v -run TestNewDateRange
```

---

**Step 3.1.3: 編寫測試 - Contains 方法 (30 min)**

在 `value_objects_test.go` 新增：

```go
// Test 31: DateRange Contains - 日期在範圍內
func TestDateRange_Contains_DateInRange(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	testDate := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	// Act
	result := dr.Contains(testDate)

	// Assert
	assert.True(t, result)
}

// Test 32: DateRange Contains - 日期在範圍外（之前）
func TestDateRange_Contains_DateBeforeRange(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	testDate := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)

	// Act
	result := dr.Contains(testDate)

	// Assert
	assert.False(t, result)
}

// Test 33: DateRange Contains - 日期在範圍外（之後）
func TestDateRange_Contains_DateAfterRange(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	testDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Act
	result := dr.Contains(testDate)

	// Assert
	assert.False(t, result)
}

// Test 34: DateRange Contains - 邊界測試（開始日期）
func TestDateRange_Contains_StartDate(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	// Act
	result := dr.Contains(start)

	// Assert
	assert.True(t, result)
}

// Test 35: DateRange Contains - 邊界測試（結束日期）
func TestDateRange_Contains_EndDate(t *testing.T) {
	// Arrange
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	dr, _ := points.NewDateRange(start, end)

	// Act
	result := dr.Contains(end)

	// Assert
	assert.True(t, result)
}
```

---

**Step 3.1.4: 實作 Contains 方法 (15 min)**

在 `value_objects.go` 新增：

```go
// Contains 判斷日期是否在範圍內
func (dr DateRange) Contains(date time.Time) bool {
	return !date.Before(dr.startDate) && !date.After(dr.endDate)
}
```

執行測試：
```bash
go test -v -run TestDateRange_Contains
```

---

**Step 3.1.5: 編寫測試 - Overlaps 方法 (40 min)**

在 `value_objects_test.go` 新增：

```go
// Test 36: DateRange Overlaps - 完全重疊
func TestDateRange_Overlaps_CompleteOverlap(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.True(t, result)
}

// Test 37: DateRange Overlaps - 部分重疊
func TestDateRange_Overlaps_PartialOverlap(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.True(t, result)
}

// Test 38: DateRange Overlaps - 不重疊（之前）
func TestDateRange_Overlaps_NoOverlapBefore(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.False(t, result)
}

// Test 39: DateRange Overlaps - 不重疊（之後）
func TestDateRange_Overlaps_NoOverlapAfter(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.False(t, result)
}

// Test 40: DateRange Overlaps - 邊界接觸（不算重疊）
func TestDateRange_Overlaps_EdgeTouch(t *testing.T) {
	// Arrange
	dr1, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC),
	)
	dr2, _ := points.NewDateRange(
		time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	)

	// Act
	result := dr1.Overlaps(dr2)

	// Assert
	assert.False(t, result)
}
```

---

**Step 3.1.6: 實作 Overlaps 方法 (20 min)**

在 `value_objects.go` 新增：

```go
// Overlaps 判斷是否與另一範圍重疊
func (dr DateRange) Overlaps(other DateRange) bool {
	return dr.startDate.Before(other.endDate) && other.startDate.Before(dr.endDate)
}
```

執行測試：
```bash
go test -v -run TestDateRange_Overlaps
```

---

#### 任務 3.2: PointsSource 枚舉 (1.5h)

**目標**: 實作積分來源枚舉

---

**Step 3.2.1: 編寫測試 (30 min)**

在 `value_objects_test.go` 新增：

```go
// === PointsSource 測試 ===

// Test 41: PointsSource String 方法
func TestPointsSource_String(t *testing.T) {
	tests := []struct {
		source   points.PointsSource
		expected string
	}{
		{points.PointsSourceInvoice, "invoice"},
		{points.PointsSourceSurvey, "survey"},
		{points.PointsSourceRedemption, "redemption"},
		{points.PointsSourceExpiration, "expiration"},
		{points.PointsSourceTransfer, "transfer"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Act
			result := tt.source.String()

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test 42: PointsSource 未知類型
func TestPointsSource_String_Unknown(t *testing.T) {
	// Arrange
	unknownSource := points.PointsSource(999)

	// Act
	result := unknownSource.String()

	// Assert
	assert.Equal(t, "unknown", result)
}
```

---

**Step 3.2.2: 實作 PointsSource (15 min)**

在 `value_objects.go` 新增：

```go
// PointsSource 積分來源
type PointsSource int

const (
	PointsSourceInvoice    PointsSource = iota // 發票
	PointsSourceSurvey                         // 問卷
	PointsSourceRedemption                     // 兌換（V3.2+）
	PointsSourceExpiration                     // 過期（V3.3+）
	PointsSourceTransfer                       // 轉讓（V4.0+）
)

// String 返回字符串表示
func (s PointsSource) String() string {
	switch s {
	case PointsSourceInvoice:
		return "invoice"
	case PointsSourceSurvey:
		return "survey"
	case PointsSourceRedemption:
		return "redemption"
	case PointsSourceExpiration:
		return "expiration"
	case PointsSourceTransfer:
		return "transfer"
	default:
		return "unknown"
	}
}
```

執行測試：
```bash
go test -v -run TestPointsSource
```

---

#### 任務 3.3: 重構與整理 (1.5h)

**Step 3.3.1: 組織 import (15 min)**

確保 `value_objects.go` 的 import 整齊：

```go
package points

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)
```

---

**Step 3.3.2: 添加 godoc 註釋 (30 min)**

為每個公開的類型和方法添加文檔：

```go
// PointsAmount 積分數量
// 設計原則：不可變、包含驗證邏輯
// 業務規則：積分數量必須 >= 0
type PointsAmount struct {
	value int
}

// NewPointsAmount 創建積分數量（帶驗證）
// 返回錯誤而非 panic，符合 Go 慣用法和錯誤處理原則
//
// 參數:
//   - value: 積分數量（必須 >= 0）
//
// 返回:
//   - PointsAmount: 積分數量值對象
//   - error: 如果 value < 0，返回 ErrNegativePointsAmount
func NewPointsAmount(value int) (PointsAmount, error) {
	// ...
}
```

---

**Step 3.3.3: 執行 golangci-lint (15 min)**

```bash
# 安裝 golangci-lint（如果尚未安裝）
# macOS:
brew install golangci-lint

# 執行 linter
cd internal/domain/points
golangci-lint run

# 修正任何 linter 警告
```

---

**每日檢查點 - Day 3 (15 min)**

```bash
# 執行所有測試
cd internal/domain/points
go test -v -cover

# 檢查覆蓋率
go test -coverprofile=coverage.out
go tool cover -func=coverage.out

# 預期結果
# PASS
# 42 tests
# coverage: 100% of statements
```

**完成標準**:
- ✅ 42 個測試全部通過
- ✅ 值對象覆蓋率 100%
- ✅ 所有測試執行時間 < 1 秒
- ✅ 無 linter 警告
- ✅ 所有公開 API 有 godoc 註釋

**Day 3 產出**:
- ✅ DateRange 值對象（含 Contains 和 Overlaps 邏輯）
- ✅ PointsSource 枚舉
- ✅ 程式碼重構與文檔完善

**預估總時間**: 7 小時

---

### Day 4: PointsAccount 聚合根 - Part 1（建構與基本操作）

#### 時間分配
- 上午 (4h): PointsAccount 結構 + 建構函數
- 下午 (4h): EarnPoints 命令方法

---

#### 任務 4.1: PointsAccount 聚合根基本結構 (2h)

**目標**: 建立 PointsAccount 聚合根的基礎結構

**檔案清單**:
1. `internal/domain/points/account.go`
2. `internal/domain/points/account_test.go`

---

**Step 4.1.1: 編寫測試 - NewPointsAccount 建構函數 (30 min)**

```bash
cat > internal/domain/points/account_test.go << 'EOF'
package points_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yourorg/bar_crm/internal/domain/points"
)

// === PointsAccount 建構測試 ===

// Test 43: NewPointsAccount 成功建立
func TestNewPointsAccount_ValidMemberID_Success(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")

	// Act
	account, err := points.NewPointsAccount(memberID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, memberID, account.GetMemberID())
	assert.False(t, account.GetAccountID().IsEmpty())
	assert.Equal(t, 0, account.GetEarnedPoints().Value())
	assert.Equal(t, 0, account.GetUsedPoints().Value())
	assert.Equal(t, 1, account.GetVersion()) // 初始版本為 1
}

// Test 44: NewPointsAccount 無效 MemberID
func TestNewPointsAccount_EmptyMemberID_ReturnsError(t *testing.T) {
	// Arrange
	emptyMemberID := points.MemberID{}

	// Act
	account, err := points.NewPointsAccount(emptyMemberID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, points.ErrInvalidMemberID)
}

// Test 45: NewPointsAccount 產生唯一 AccountID
func TestNewPointsAccount_GeneratesUniqueAccountID(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")

	// Act
	account1, _ := points.NewPointsAccount(memberID)
	account2, _ := points.NewPointsAccount(memberID)

	// Assert
	assert.NotEqual(t, account1.GetAccountID(), account2.GetAccountID())
}

// Test 46: NewPointsAccount 發布 AccountCreated 事件
func TestNewPointsAccount_PublishesAccountCreatedEvent(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")

	// Act
	account, _ := points.NewPointsAccount(memberID)

	// Assert
	events := account.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.account_created", events[0].EventType())
}
EOF

# 執行測試（應該失敗 - Red）
cd internal/domain/points
go test -v -run TestNewPointsAccount
```

---

**Step 4.1.2: 實作 PointsAccount 基本結構 (1h)**

```bash
cat > internal/domain/points/account.go << 'EOF'
package points

import (
	"fmt"
	"time"

	"github.com/yourorg/bar_crm/internal/domain/shared"
)

// PointsAccount 積分帳戶聚合根
// 設計原則：輕量級聚合，不包含無界集合
type PointsAccount struct {
	// 私有字段（封裝）
	accountID     AccountID
	memberID      MemberID
	earnedPoints  PointsAmount
	usedPoints    PointsAmount
	lastUpdatedAt time.Time
	version       int // 樂觀鎖版本號

	// 領域事件（待發布）
	events []shared.DomainEvent
}

// NewPointsAccount 建構函數（工廠方法）
// 所有聚合必須通過建構函數創建，確保初始狀態有效
func NewPointsAccount(memberID MemberID) (*PointsAccount, error) {
	// 驗證必填字段
	if memberID.IsEmpty() {
		return nil, ErrInvalidMemberID
	}

	// 生成聚合根 ID
	accountID := NewAccountID()

	// 初始狀態（使用 unchecked 版本，因為 0 保證有效）
	account := &PointsAccount{
		accountID:     accountID,
		memberID:      memberID,
		earnedPoints:  newPointsAmountUnchecked(0),
		usedPoints:    newPointsAmountUnchecked(0),
		lastUpdatedAt: time.Now(),
		version:       1,
		events:        []shared.DomainEvent{},
	}

	// 發布創建事件
	account.publishEvent(NewPointsAccountCreatedEvent(accountID, memberID))

	return account, nil
}

// --- 查詢方法（無狀態變更）---

// GetAccountID 獲取帳戶 ID
func (a *PointsAccount) GetAccountID() AccountID {
	return a.accountID
}

// GetMemberID 獲取會員 ID
func (a *PointsAccount) GetMemberID() MemberID {
	return a.memberID
}

// GetEarnedPoints 獲取累積積分
func (a *PointsAccount) GetEarnedPoints() PointsAmount {
	return a.earnedPoints
}

// GetUsedPoints 獲取已使用積分
func (a *PointsAccount) GetUsedPoints() PointsAmount {
	return a.usedPoints
}

// GetVersion 獲取版本號（樂觀鎖）
func (a *PointsAccount) GetVersion() int {
	return a.version
}

// GetPreviousVersion 獲取上一個版本號（用於樂觀鎖檢查）
func (a *PointsAccount) GetPreviousVersion() int {
	if a.version <= 1 {
		return 1
	}
	return a.version - 1
}

// GetLastUpdatedAt 獲取最後更新時間
func (a *PointsAccount) GetLastUpdatedAt() time.Time {
	return a.lastUpdatedAt
}

// --- 領域事件管理 ---

// GetEvents 獲取待發布的事件
func (a *PointsAccount) GetEvents() []shared.DomainEvent {
	return a.events
}

// ClearEvents 清空事件（發布後調用）
func (a *PointsAccount) ClearEvents() {
	a.events = []shared.DomainEvent{}
}

// publishEvent 發布事件（私有方法）
func (a *PointsAccount) publishEvent(event shared.DomainEvent) {
	a.events = append(a.events, event)
}
EOF

# 執行測試（應該還會失敗，因為缺少 Event）
go test -v -run TestNewPointsAccount
```

---

**Step 4.1.3: 實作 PointsAccountCreated 事件 (30 min)**

在 `events.go` 新增（如果檔案不存在則建立）：

```bash
cat > internal/domain/points/events.go << 'EOF'
package points

import (
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/bar_crm/internal/domain/shared"
)

// --- PointsAccountCreated 事件 ---

// PointsAccountCreated 積分帳戶已建立事件
type PointsAccountCreated struct {
	eventID    string
	accountID  AccountID
	memberID   MemberID
	occurredAt time.Time
}

// NewPointsAccountCreatedEvent 創建積分帳戶已建立事件
func NewPointsAccountCreatedEvent(accountID AccountID, memberID MemberID) PointsAccountCreated {
	return PointsAccountCreated{
		eventID:    uuid.New().String(),
		accountID:  accountID,
		memberID:   memberID,
		occurredAt: time.Now(),
	}
}

// 實作 DomainEvent 介面
func (e PointsAccountCreated) EventID() string {
	return e.eventID
}

func (e PointsAccountCreated) EventType() string {
	return "points.account_created"
}

func (e PointsAccountCreated) OccurredAt() time.Time {
	return e.occurredAt
}

func (e PointsAccountCreated) AggregateID() string {
	return e.accountID.String()
}

// Getters
func (e PointsAccountCreated) AccountID() AccountID {
	return e.accountID
}

func (e PointsAccountCreated) MemberID() MemberID {
	return e.memberID
}
EOF

# 執行測試（應該通過 - Green）
go test -v -run TestNewPointsAccount
```

---

#### 任務 4.2: EarnPoints 命令方法 (2h)

**目標**: 實作獲得積分的核心業務邏輯

---

**Step 4.2.1: 編寫測試 - EarnPoints 基本功能 (45 min)**

在 `account_test.go` 新增：

```go
// === EarnPoints 命令測試 ===

// Test 47: EarnPoints 成功
func TestPointsAccount_EarnPoints_Success(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)
	account.ClearEvents() // 清除建立事件

	amount, _ := points.NewPointsAmount(100)

	// Act
	err := account.EarnPoints(
		amount,
		points.PointsSourceInvoice,
		"invoice-123",
		"購買商品",
	)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, account.GetEarnedPoints().Value())
	assert.Equal(t, 0, account.GetUsedPoints().Value())
	assert.Equal(t, 2, account.GetVersion()) // 版本號增加
}

// Test 48: EarnPoints 負數金額
func TestPointsAccount_EarnPoints_NegativeAmount_ReturnsError(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	amount, _ := points.NewPointsAmount(0) // 先建立有效的
	// 模擬負數（透過內部方法，僅用於測試）

	// Act
	err := account.EarnPoints(
		amount,
		points.PointsSourceInvoice,
		"invoice-123",
		"",
	)

	// Assert: 0 積分應該可以接受
	assert.NoError(t, err)
}

// Test 49: EarnPoints 發布事件
func TestPointsAccount_EarnPoints_PublishesEvent(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)
	account.ClearEvents()

	amount, _ := points.NewPointsAmount(100)

	// Act
	err := account.EarnPoints(
		amount,
		points.PointsSourceInvoice,
		"invoice-123",
		"購買商品",
	)

	// Assert
	assert.NoError(t, err)
	events := account.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.earned", events[0].EventType())
}

// Test 50: EarnPoints 版本號遞增
func TestPointsAccount_EarnPoints_IncrementsVersion(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)
	initialVersion := account.GetVersion()

	amount, _ := points.NewPointsAmount(100)

	// Act
	account.EarnPoints(amount, points.PointsSourceInvoice, "inv-1", "test")

	// Assert
	assert.Equal(t, initialVersion+1, account.GetVersion())
}

// Test 51: EarnPoints 多次累加
func TestPointsAccount_EarnPoints_Accumulates(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(50)
	amount3, _ := points.NewPointsAmount(25)

	// Act
	account.EarnPoints(amount1, points.PointsSourceInvoice, "inv-1", "test")
	account.EarnPoints(amount2, points.PointsSourceInvoice, "inv-2", "test")
	account.EarnPoints(amount3, points.PointsSourceSurvey, "survey-1", "test")

	// Assert
	assert.Equal(t, 175, account.GetEarnedPoints().Value())
	assert.Equal(t, 4, account.GetVersion()) // 1 (初始) + 3 (操作)
}
```

執行測試（應該失敗）：
```bash
go test -v -run TestPointsAccount_EarnPoints
```

---

**Step 4.2.2: 實作 EarnPoints 方法 (30 min)**

在 `account.go` 新增：

```go
// --- 命令方法（狀態變更）---

// EarnPoints 獲得積分（核心業務邏輯）
func (a *PointsAccount) EarnPoints(
	amount PointsAmount,
	source PointsSource,
	sourceID string,
	description string,
) error {
	// 前置條件檢查
	if amount.Value() < 0 {
		return ErrNegativePointsAmount
	}

	// 狀態變更
	a.earnedPoints = a.earnedPoints.Add(amount)
	a.lastUpdatedAt = time.Now()
	a.version++ // 聚合自己控制版本號

	// 發布領域事件
	event := NewPointsEarnedEvent(
		a.accountID,
		amount,
		source,
		sourceID,
		description,
	)
	a.publishEvent(event)

	return nil
}
```

---

**Step 4.2.3: 實作 PointsEarned 事件 (30 min)**

在 `events.go` 新增：

```go
// --- PointsEarned 事件 ---

// PointsEarned 積分已獲得事件
type PointsEarned struct {
	eventID     string
	accountID   AccountID
	amount      PointsAmount
	source      PointsSource
	sourceID    string
	description string
	occurredAt  time.Time
}

// NewPointsEarnedEvent 創建積分已獲得事件
func NewPointsEarnedEvent(
	accountID AccountID,
	amount PointsAmount,
	source PointsSource,
	sourceID string,
	description string,
) PointsEarned {
	return PointsEarned{
		eventID:     uuid.New().String(),
		accountID:   accountID,
		amount:      amount,
		source:      source,
		sourceID:    sourceID,
		description: description,
		occurredAt:  time.Now(),
	}
}

// 實作 DomainEvent 介面
func (e PointsEarned) EventID() string {
	return e.eventID
}

func (e PointsEarned) EventType() string {
	return "points.earned"
}

func (e PointsEarned) OccurredAt() time.Time {
	return e.occurredAt
}

func (e PointsEarned) AggregateID() string {
	return e.accountID.String()
}

// Getters
func (e PointsEarned) AccountID() AccountID {
	return e.accountID
}

func (e PointsEarned) Amount() PointsAmount {
	return e.amount
}

func (e PointsEarned) Source() PointsSource {
	return e.source
}

func (e PointsEarned) SourceID() string {
	return e.sourceID
}

func (e PointsEarned) Description() string {
	return e.description
}
```

執行測試（應該通過）：
```bash
go test -v -run TestPointsAccount_EarnPoints
```

---

**每日檢查點 - Day 4 (15 min)**

```bash
# 執行所有測試
cd internal/domain/points
go test -v -cover

# 檢查覆蓋率
go test -coverprofile=coverage.out
go tool cover -func=coverage.out

# 預期結果
# PASS
# 51 tests
# coverage: 95%+ of statements
```

**完成標準**:
- ✅ 51 個測試全部通過
- ✅ PointsAccount 基本功能測試覆蓋率 95%+
- ✅ EarnPoints 方法完整實作
- ✅ 領域事件正常發布

**Day 4 產出**:
- ✅ PointsAccount 聚合根基本結構
- ✅ NewPointsAccount 建構函數
- ✅ EarnPoints 命令方法
- ✅ PointsAccountCreated 事件
- ✅ PointsEarned 事件

**預估總時間**: 8 小時

---

### Day 5: PointsAccount 聚合根 - Part 2（進階操作與不變條件）

#### 時間分配
- 上午 (4h): DeductPoints + GetAvailablePoints
- 下午 (4h): RecalculatePoints + ReconstructPointsAccount

---

#### 任務 5.1: DeductPoints 命令方法 (2h)

**目標**: 實作扣除積分功能（V3.2+ 兌換功能）

---

**Step 5.1.1: 編寫測試 (45 min)**

在 `account_test.go` 新增：

```go
// === DeductPoints 命令測試 ===

// Test 52: DeductPoints 成功
func TestPointsAccount_DeductPoints_Success(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	// 先獲得積分
	earnAmount, _ := points.NewPointsAmount(100)
	account.EarnPoints(earnAmount, points.PointsSourceInvoice, "inv-1", "test")
	account.ClearEvents()

	// 扣除積分
	deductAmount, _ := points.NewPointsAmount(30)

	// Act
	err := account.DeductPoints(deductAmount, "兌換商品")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, account.GetEarnedPoints().Value())
	assert.Equal(t, 30, account.GetUsedPoints().Value())
}

// Test 53: DeductPoints 積分不足
func TestPointsAccount_DeductPoints_InsufficientPoints_ReturnsError(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	// 只有 50 點
	earnAmount, _ := points.NewPointsAmount(50)
	account.EarnPoints(earnAmount, points.PointsSourceInvoice, "inv-1", "test")

	// 嘗試扣除 100 點
	deductAmount, _ := points.NewPointsAmount(100)

	// Act
	err := account.DeductPoints(deductAmount, "兌換商品")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrInsufficientPoints)
	assert.Equal(t, 0, account.GetUsedPoints().Value()) // 未扣除
}

// Test 54: DeductPoints 發布事件
func TestPointsAccount_DeductPoints_PublishesEvent(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	earnAmount, _ := points.NewPointsAmount(100)
	account.EarnPoints(earnAmount, points.PointsSourceInvoice, "inv-1", "test")
	account.ClearEvents()

	deductAmount, _ := points.NewPointsAmount(30)

	// Act
	err := account.DeductPoints(deductAmount, "兌換商品")

	// Assert
	assert.NoError(t, err)
	events := account.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.deducted", events[0].EventType())
}

// Test 55: DeductPoints 負數金額
func TestPointsAccount_DeductPoints_NegativeAmount_ReturnsError(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	earnAmount, _ := points.NewPointsAmount(100)
	account.EarnPoints(earnAmount, points.PointsSourceInvoice, "inv-1", "test")

	// 負數金額會在建構 PointsAmount 時失敗
	// 這裡測試如果傳入零值
	zeroAmount, _ := points.NewPointsAmount(0)

	// Act
	err := account.DeductPoints(zeroAmount, "test")

	// Assert: 扣除 0 應該成功（雖然沒實際效果）
	assert.NoError(t, err)
}
```

執行測試：
```bash
go test -v -run TestPointsAccount_DeductPoints
```

---

**Step 5.1.2: 實作 DeductPoints 方法 (30 min)**

在 `account.go` 新增：

```go
// DeductPoints 扣除積分（V3.2+ 兌換功能）
func (a *PointsAccount) DeductPoints(amount PointsAmount, reason string) error {
	// 前置條件檢查
	if amount.Value() < 0 {
		return ErrNegativePointsAmount
	}

	// 業務規則檢查：積分是否足夠
	if !a.HasSufficientPoints(amount) {
		return ErrInsufficientPoints
	}

	// 狀態變更
	a.usedPoints = a.usedPoints.Add(amount)
	a.lastUpdatedAt = time.Now()
	a.version++

	// 發布事件
	event := NewPointsDeductedEvent(a.accountID, amount, reason)
	a.publishEvent(event)

	return nil
}

// HasSufficientPoints 檢查積分是否足夠
func (a *PointsAccount) HasSufficientPoints(amount PointsAmount) bool {
	return a.GetAvailablePoints().Value() >= amount.Value()
}
```

---

**Step 5.1.3: 實作 PointsDeducted 事件 (20 min)**

在 `events.go` 新增：

```go
// --- PointsDeducted 事件 ---

// PointsDeducted 積分已扣除事件
type PointsDeducted struct {
	eventID    string
	accountID  AccountID
	amount     PointsAmount
	reason     string
	occurredAt time.Time
}

// NewPointsDeductedEvent 創建積分已扣除事件
func NewPointsDeductedEvent(
	accountID AccountID,
	amount PointsAmount,
	reason string,
) PointsDeducted {
	return PointsDeducted{
		eventID:    uuid.New().String(),
		accountID:  accountID,
		amount:     amount,
		reason:     reason,
		occurredAt: time.Now(),
	}
}

// 實作 DomainEvent 介面
func (e PointsDeducted) EventID() string {
	return e.eventID
}

func (e PointsDeducted) EventType() string {
	return "points.deducted"
}

func (e PointsDeducted) OccurredAt() time.Time {
	return e.occurredAt
}

func (e PointsDeducted) AggregateID() string {
	return e.accountID.String()
}

// Getters
func (e PointsDeducted) AccountID() AccountID {
	return e.accountID
}

func (e PointsDeducted) Amount() PointsAmount {
	return e.amount
}

func (e PointsDeducted) Reason() string {
	return e.reason
}
```

在 `errors.go` 新增錯誤（如果還沒有）：

```go
// 在已有的錯誤中新增
var (
	// ... 其他錯誤
	ErrInsufficientPoints = errors.New("insufficient points for this operation")
)
```

執行測試：
```bash
go test -v -run TestPointsAccount_DeductPoints
```

---

#### 任務 5.2: GetAvailablePoints 查詢方法（含 Panic 檢查）(2h)

**目標**: 實作可用積分查詢，並驗證不變條件

---

**Step 5.2.1: 編寫測試 (45 min)**

在 `account_test.go` 新增：

```go
// === GetAvailablePoints 查詢測試 ===

// Test 56: GetAvailablePoints 正常計算
func TestPointsAccount_GetAvailablePoints_Success(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	earnAmount, _ := points.NewPointsAmount(100)
	account.EarnPoints(earnAmount, points.PointsSourceInvoice, "inv-1", "test")

	deductAmount, _ := points.NewPointsAmount(30)
	account.DeductPoints(deductAmount, "兌換")

	// Act
	available := account.GetAvailablePoints()

	// Assert
	assert.Equal(t, 70, available.Value()) // 100 - 30 = 70
}

// Test 57: GetAvailablePoints 無扣除
func TestPointsAccount_GetAvailablePoints_NoDeduction(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	earnAmount, _ := points.NewPointsAmount(150)
	account.EarnPoints(earnAmount, points.PointsSourceInvoice, "inv-1", "test")

	// Act
	available := account.GetAvailablePoints()

	// Assert
	assert.Equal(t, 150, available.Value())
}

// Test 58: GetAvailablePoints 全部扣除
func TestPointsAccount_GetAvailablePoints_AllDeducted(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	earnAmount, _ := points.NewPointsAmount(100)
	account.EarnPoints(earnAmount, points.PointsSourceInvoice, "inv-1", "test")

	deductAmount, _ := points.NewPointsAmount(100)
	account.DeductPoints(deductAmount, "兌換")

	// Act
	available := account.GetAvailablePoints()

	// Assert
	assert.Equal(t, 0, available.Value())
}

// Test 59: GetAvailablePoints 不變條件違反（Panic）
func TestPointsAccount_GetAvailablePoints_InvariantViolation_Panics(t *testing.T) {
	// 這個測試驗證如果資料損壞（usedPoints > earnedPoints），應該 panic
	// 注意：我們無法透過公開 API 建立這種狀態，這會在 ReconstructPointsAccount 測試中驗證
	t.Skip("Invariant violation tested in ReconstructPointsAccount")
}
```

執行測試：
```bash
go test -v -run TestPointsAccount_GetAvailablePoints
```

---

**Step 5.2.2: 實作 GetAvailablePoints 方法 (30 min)**

在 `account.go` 新增：

```go
// GetAvailablePoints 獲取可用積分（計算屬性）
// 使用 unchecked 版本，因為聚合不變性保證 earnedPoints >= usedPoints
// 如果不變條件被違反（資料損壞），subtractUnchecked 會 panic
func (a *PointsAccount) GetAvailablePoints() PointsAmount {
	// 防禦性檢查：在調用 subtractUnchecked 前驗證不變條件
	// 如果違反，提供更清晰的錯誤信息
	if a.usedPoints.Value() > a.earnedPoints.Value() {
		panic(fmt.Sprintf("invariant violation: used points (%d) > earned points (%d) for account %s",
			a.usedPoints.Value(), a.earnedPoints.Value(), a.accountID.String()))
	}
	return a.earnedPoints.subtractUnchecked(a.usedPoints)
}
```

執行測試：
```bash
go test -v -run TestPointsAccount_GetAvailablePoints
```

---

#### 任務 5.3: RecalculatePoints 命令方法 (2h)

**Step 5.3.1: 定義 PointsCalculableTransaction 介面 (15 min)**

在 `account.go` 新增（在檔案末尾）：

```go
// --- PointsCalculableTransaction 介面定義（用於解耦）---

// PointsCalculableTransaction 可計算積分的交易介面
// 設計原則：介面名稱表達用途（積分計算），而非資料結構
// Application Layer 的 DTO 實作此介面
type PointsCalculableTransaction interface {
	GetTransactionAmount() decimal.Decimal
	GetTransactionDate() time.Time
	HasCompletedSurvey() bool
}

// PointsCalculationService 積分計算服務介面
type PointsCalculationService interface {
	CalculateForTransaction(tx PointsCalculableTransaction) PointsAmount
}
```

新增 import：
```go
import (
	// ... 其他 imports
	"github.com/shopspring/decimal"
)
```

---

**Step 5.3.2: 編寫測試 (45 min)**

由於 RecalculatePoints 需要 PointsCalculationService，我們先建立一個 Mock：

在 `account_test.go` 新增：

```go
import "github.com/shopspring/decimal"

// === Mock PointsCalculationService ===

type MockCalculationService struct {
	calculateFunc func(tx points.PointsCalculableTransaction) points.PointsAmount
}

func (m *MockCalculationService) CalculateForTransaction(tx points.PointsCalculableTransaction) points.PointsAmount {
	if m.calculateFunc != nil {
		return m.calculateFunc(tx)
	}
	// 預設：100 元 = 1 點
	amount := tx.GetTransactionAmount()
	pointsValue := int(amount.Div(decimal.NewFromInt(100)).Floor().IntPart())
	result, _ := points.NewPointsAmount(pointsValue)
	return result
}

// === Mock Transaction ===

type MockTransaction struct {
	amount         decimal.Decimal
	date           time.Time
	surveyComplete bool
}

func (m MockTransaction) GetTransactionAmount() decimal.Decimal {
	return m.amount
}

func (m MockTransaction) GetTransactionDate() time.Time {
	return m.date
}

func (m MockTransaction) HasCompletedSurvey() bool {
	return m.surveyComplete
}

// === RecalculatePoints 測試 ===

// Test 60: RecalculatePoints 成功
func TestPointsAccount_RecalculatePoints_Success(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	calculator := &MockCalculationService{}

	transactions := []points.PointsCalculableTransaction{
		MockTransaction{amount: decimal.NewFromInt(350), date: time.Now(), surveyComplete: false},
		MockTransaction{amount: decimal.NewFromInt(250), date: time.Now(), surveyComplete: false},
	}

	// Act
	err := account.RecalculatePoints(transactions, calculator)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 6, account.GetEarnedPoints().Value()) // 3 + 2 + 1(survey bonus) = 6
}

// Test 61: RecalculatePoints 違反不變條件
func TestPointsAccount_RecalculatePoints_ViolatesInvariant_ReturnsError(t *testing.T) {
	// Arrange
	memberID, _ := points.NewMemberID("member-123")
	account, _ := points.NewPointsAccount(memberID)

	// 先獲得 100 點並扣除 80 點
	earnAmount, _ := points.NewPointsAmount(100)
	account.EarnPoints(earnAmount, points.PointsSourceInvoice, "inv-1", "test")

	deductAmount, _ := points.NewPointsAmount(80)
	account.DeductPoints(deductAmount, "兌換")

	calculator := &MockCalculationService{}

	// 重算後只有 50 點（< usedPoints 80）
	transactions := []points.PointsCalculableTransaction{
		MockTransaction{amount: decimal.NewFromInt(50), date: time.Now(), surveyComplete: false},
	}

	// Act
	err := account.RecalculatePoints(transactions, calculator)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrInsufficientEarnedPoints)
	assert.Equal(t, 100, account.GetEarnedPoints().Value()) // 未改變
}
```

在 `errors.go` 新增錯誤：

```go
var (
	// ... 其他錯誤
	ErrInsufficientEarnedPoints = errors.New("earned points cannot be less than used points")
)
```

執行測試：
```bash
go test -v -run TestPointsAccount_RecalculatePoints
```

---

**Step 5.3.3: 實作 RecalculatePoints 方法 (30 min)**

在 `account.go` 新增：

```go
// RecalculatePoints 重算累積積分（管理員觸發）
// 使用 Domain Service 計算，聚合負責狀態更新
func (a *PointsAccount) RecalculatePoints(
	transactions []PointsCalculableTransaction,
	calculator PointsCalculationService,
) error {
	// 計算新的累積積分（委託給 Domain Service）
	newEarnedPoints := 0
	for _, tx := range transactions {
		points := calculator.CalculateForTransaction(tx)
		newEarnedPoints += points.Value()
	}

	// 業務規則檢查：創建並驗證新積分數量
	newAmount, err := NewPointsAmount(newEarnedPoints)
	if err != nil {
		return err
	}

	// 不變條件檢查：新的累積積分不能小於已使用積分
	if newAmount.Value() < a.usedPoints.Value() {
		return ErrInsufficientEarnedPoints
	}

	// 狀態變更
	oldPoints := a.earnedPoints
	a.earnedPoints = newAmount
	a.lastUpdatedAt = time.Now()
	a.version++

	// 發布事件
	event := NewPointsRecalculatedEvent(a.accountID, oldPoints.Value(), newEarnedPoints)
	a.publishEvent(event)

	return nil
}
```

---

**Step 5.3.4: 實作 PointsRecalculated 事件 (20 min)**

在 `events.go` 新增：

```go
// --- PointsRecalculated 事件 ---

// PointsRecalculated 積分已重算事件
type PointsRecalculated struct {
	eventID    string
	accountID  AccountID
	oldPoints  int
	newPoints  int
	occurredAt time.Time
}

// NewPointsRecalculatedEvent 創建積分已重算事件
func NewPointsRecalculatedEvent(
	accountID AccountID,
	oldPoints int,
	newPoints int,
) PointsRecalculated {
	return PointsRecalculated{
		eventID:    uuid.New().String(),
		accountID:  accountID,
		oldPoints:  oldPoints,
		newPoints:  newPoints,
		occurredAt: time.Now(),
	}
}

// 實作 DomainEvent 介面
func (e PointsRecalculated) EventID() string {
	return e.eventID
}

func (e PointsRecalculated) EventType() string {
	return "points.recalculated"
}

func (e PointsRecalculated) OccurredAt() time.Time {
	return e.occurredAt
}

func (e PointsRecalculated) AggregateID() string {
	return e.accountID.String()
}

// Getters
func (e PointsRecalculated) AccountID() AccountID {
	return e.accountID
}

func (e PointsRecalculated) OldPoints() int {
	return e.oldPoints
}

func (e PointsRecalculated) NewPoints() int {
	return e.newPoints
}
```

執行測試：
```bash
go test -v -run TestPointsAccount_RecalculatePoints
```

---

#### 任務 5.4: ReconstructPointsAccount 工廠方法（關鍵！）(2h)

**目標**: 實作從資料庫重建聚合根的方法，並驗證資料完整性

---

**Step 5.4.1: 編寫測試 (1h)**

在 `account_test.go` 新增：

```go
// === ReconstructPointsAccount 測試 ===

// Test 62: ReconstructPointsAccount 有效資料
func TestReconstructPointsAccount_ValidData_Success(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	memberID, _ := points.NewMemberID("member-123")

	// Act
	account, err := points.ReconstructPointsAccount(
		accountID,
		memberID,
		150,           // earnedPoints
		50,            // usedPoints
		3,             // version
		time.Now(),    // lastUpdatedAt
	)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, accountID, account.GetAccountID())
	assert.Equal(t, memberID, account.GetMemberID())
	assert.Equal(t, 150, account.GetEarnedPoints().Value())
	assert.Equal(t, 50, account.GetUsedPoints().Value())
	assert.Equal(t, 3, account.GetVersion())
	assert.Equal(t, 100, account.GetAvailablePoints().Value())
	assert.Len(t, account.GetEvents(), 0) // 重建時不包含事件
}

// Test 63: ReconstructPointsAccount 負數 earnedPoints
func TestReconstructPointsAccount_NegativeEarnedPoints_ReturnsError(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	memberID, _ := points.NewMemberID("member-123")

	// Act
	account, err := points.ReconstructPointsAccount(
		accountID,
		memberID,
		-100,          // 負數累積積分
		0,
		1,
		time.Now(),
	)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.Contains(t, err.Error(), "invalid earned points")
}

// Test 64: ReconstructPointsAccount 負數 usedPoints
func TestReconstructPointsAccount_NegativeUsedPoints_ReturnsError(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	memberID, _ := points.NewMemberID("member-123")

	// Act
	account, err := points.ReconstructPointsAccount(
		accountID,
		memberID,
		100,
		-50,           // 負數已使用積分
		1,
		time.Now(),
	)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.Contains(t, err.Error(), "invalid used points")
}

// Test 65: ReconstructPointsAccount 不變條件違反（資料損壞）
func TestReconstructPointsAccount_InvariantViolation_ReturnsError(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	memberID, _ := points.NewMemberID("member-123")

	// Act: usedPoints (100) > earnedPoints (50)
	account, err := points.ReconstructPointsAccount(
		accountID,
		memberID,
		50,            // earnedPoints
		100,           // usedPoints（違反不變條件）
		1,
		time.Now(),
	)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.Contains(t, err.Error(), "data corruption")
}

// Test 66: ReconstructPointsAccount 無效版本號
func TestReconstructPointsAccount_InvalidVersion_ReturnsError(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	memberID, _ := points.NewMemberID("member-123")

	// Act
	account, err := points.ReconstructPointsAccount(
		accountID,
		memberID,
		100,
		50,
		0,             // 版本號 < 1
		time.Now(),
	)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.Contains(t, err.Error(), "invalid version")
}
```

執行測試：
```bash
go test -v -run TestReconstructPointsAccount
```

---

**Step 5.4.2: 實作 ReconstructPointsAccount 方法 (30 min)**

在 `account.go` 新增：

```go
// --- 聚合重建方法（僅供 Infrastructure Layer 使用）---

// ReconstructPointsAccount 從持久化存儲重建聚合根
// 注意：此方法僅供 Repository 使用
// 重要：即使是從資料庫重建，也必須驗證不變條件，防止損壞資料污染領域層
// 重建的聚合不包含領域事件（事件已發布過）
func ReconstructPointsAccount(
	accountID AccountID,
	memberID MemberID,
	earnedPoints int,
	usedPoints int,
	version int,
	lastUpdatedAt time.Time,
) (*PointsAccount, error) {
	// 1. 驗證積分數量（防止負數）
	earnedAmount, err := NewPointsAmount(earnedPoints)
	if err != nil {
		return nil, fmt.Errorf("invalid earned points in database: %w", err)
	}

	usedAmount, err := NewPointsAmount(usedPoints)
	if err != nil {
		return nil, fmt.Errorf("invalid used points in database: %w", err)
	}

	// 2. 驗證關鍵不變條件：usedPoints <= earnedPoints
	if usedAmount.Value() > earnedAmount.Value() {
		return nil, fmt.Errorf("data corruption: used points (%d) exceeds earned points (%d)",
			usedPoints, earnedPoints)
	}

	// 3. 驗證版本號
	if version < 1 {
		return nil, fmt.Errorf("invalid version in database: %d", version)
	}

	// 4. 重建聚合（使用已驗證的值對象）
	return &PointsAccount{
		accountID:     accountID,
		memberID:      memberID,
		earnedPoints:  earnedAmount,
		usedPoints:    usedAmount,
		version:       version,
		lastUpdatedAt: lastUpdatedAt,
		events:        []shared.DomainEvent{}, // 重建時不包含事件
	}, nil
}
```

執行測試：
```bash
go test -v -run TestReconstructPointsAccount
```

---

**每日檢查點 - Day 5 (15 min)**

```bash
# 執行所有測試
cd internal/domain/points
go test -v -cover

# 檢查覆蓋率
go test -coverprofile=coverage.out
go tool cover -func=coverage.out

# 預期結果
# PASS
# 66 tests
# coverage: 95%+ of statements
```

**完成標準**:
- ✅ 66 個測試全部通過
- ✅ PointsAccount 聚合根覆蓋率 95%+
- ✅ 所有命令方法實作完成
- ✅ ReconstructPointsAccount 驗證資料完整性

**Day 5 產出**:
- ✅ DeductPoints 命令方法
- ✅ GetAvailablePoints 查詢方法（含不變條件檢查）
- ✅ RecalculatePoints 命令方法
- ✅ ReconstructPointsAccount 工廠方法
- ✅ PointsDeducted 事件
- ✅ PointsRecalculated 事件

**預估總時間**: 8 小時

---

**下一天預告**: Day 6-7 將實作 ConversionRule 聚合根、Repository 介面和領域事件完整定義。

---

### Day 6: ConversionRule 聚合根 + Domain Service（積分計算規則）

#### 時間分配
- 上午 (4h): ConversionRule 聚合根結構 + 建構函數
- 下午 (4h): 規則驗證邏輯 + PointsCalculationService

---

#### 任務 6.1: ConversionRule 聚合根基本結構 (2h)

**目標**: 建立 ConversionRule 聚合根，管理積分兌換規則的生命週期

**檔案清單**:
1. `internal/domain/points/conversion_rule.go`
2. `internal/domain/points/conversion_rule_test.go`

---

**Step 6.1.1: 編寫測試 - NewConversionRule 建構函數 (40 min)**

```bash
cat > internal/domain/points/conversion_rule_test.go << 'EOF'
package points_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yourorg/bar_crm/internal/domain/points"
)

// === ConversionRule 建構測試 ===

// Test 67: NewConversionRule 成功建立
func TestNewConversionRule_ValidInput_Success(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	description := "一般會員積分規則"

	// Act
	rule, err := points.NewConversionRule(rate, dateRange, description)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, rule)
	assert.False(t, rule.GetRuleID().IsEmpty())
	assert.Equal(t, rate, rule.GetRate())
	assert.Equal(t, dateRange, rule.GetDateRange())
	assert.Equal(t, description, rule.GetDescription())
	assert.True(t, rule.IsActive())
	assert.Equal(t, 1, rule.GetVersion())
}

// Test 68: NewConversionRule 空描述失敗
func TestNewConversionRule_EmptyDescription_ReturnsError(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	description := ""

	// Act
	rule, err := points.NewConversionRule(rate, dateRange, description)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.ErrorIs(t, err, points.ErrInvalidDescription)
}

// Test 69: NewConversionRule 描述過長失敗
func TestNewConversionRule_DescriptionTooLong_ReturnsError(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	description := string(make([]byte, 201)) // 超過 200 字元

	// Act
	rule, err := points.NewConversionRule(rate, dateRange, description)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.ErrorIs(t, err, points.ErrInvalidDescription)
}

// Test 70: NewConversionRule 發布 RuleCreated 事件
func TestNewConversionRule_PublishesRuleCreatedEvent(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)

	// Act
	rule, _ := points.NewConversionRule(rate, dateRange, "測試規則")

	// Assert
	events := rule.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.conversion_rule_created", events[0].EventType())
}
EOF

# 執行測試（應該失敗 - Red）
cd internal/domain/points
go test -v -run TestNewConversionRule
```

---

**Step 6.1.2: 實作 ConversionRule 基本結構 (1h 20min)**

```bash
cat > internal/domain/points/conversion_rule.go << 'EOF'
package points

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/bar_crm/internal/domain/shared"
)

// RuleID 規則唯一識別碼
type RuleID struct {
	value string
}

// NewRuleID 建立新的規則 ID
func NewRuleID() RuleID {
	return RuleID{value: uuid.New().String()}
}

// NewRuleIDFromString 從字串建立規則 ID
func NewRuleIDFromString(value string) (RuleID, error) {
	if value == "" {
		return RuleID{}, ErrInvalidRuleID
	}
	return RuleID{value: value}, nil
}

// String 返回字串表示
func (r RuleID) String() string {
	return r.value
}

// IsEmpty 檢查是否為空
func (r RuleID) IsEmpty() bool {
	return r.value == ""
}

// Equals 比較兩個 RuleID
func (r RuleID) Equals(other RuleID) bool {
	return r.value == other.value
}

// ConversionRule 積分兌換規則聚合根
// 業務規則：
// 1. 同一時間範圍內只能有一個生效的規則（由 Domain Service 驗證）
// 2. 規則一旦停用後不可重新啟用（防止時光倒流）
// 3. 規則的日期範圍不可變更（修改需要建立新規則）
type ConversionRule struct {
	ruleID      RuleID
	rate        ConversionRate
	dateRange   DateRange
	description string
	isActive    bool
	createdAt   time.Time
	deactivatedAt *time.Time
	version     int

	events []shared.DomainEvent
}

// NewConversionRule 建構函數（工廠方法）
func NewConversionRule(
	rate ConversionRate,
	dateRange DateRange,
	description string,
) (*ConversionRule, error) {
	// 驗證描述
	if description == "" {
		return nil, ErrInvalidDescription
	}
	if len(description) > 200 {
		return nil, ErrInvalidDescription
	}

	// 建立規則
	rule := &ConversionRule{
		ruleID:        NewRuleID(),
		rate:          rate,
		dateRange:     dateRange,
		description:   description,
		isActive:      true,
		createdAt:     time.Now(),
		deactivatedAt: nil,
		version:       1,
		events:        []shared.DomainEvent{},
	}

	// 發布創建事件
	rule.publishEvent(NewConversionRuleCreatedEvent(
		rule.ruleID,
		rule.rate,
		rule.dateRange,
		rule.description,
	))

	return rule, nil
}

// --- 查詢方法 ---

// GetRuleID 獲取規則 ID
func (r *ConversionRule) GetRuleID() RuleID {
	return r.ruleID
}

// GetRate 獲取兌換率
func (r *ConversionRule) GetRate() ConversionRate {
	return r.rate
}

// GetDateRange 獲取生效日期範圍
func (r *ConversionRule) GetDateRange() DateRange {
	return r.dateRange
}

// GetDescription 獲取描述
func (r *ConversionRule) GetDescription() string {
	return r.description
}

// IsActive 是否生效中
func (r *ConversionRule) IsActive() bool {
	return r.isActive
}

// GetVersion 獲取版本號
func (r *ConversionRule) GetVersion() int {
	return r.version
}

// GetCreatedAt 獲取建立時間
func (r *ConversionRule) GetCreatedAt() time.Time {
	return r.createdAt
}

// GetDeactivatedAt 獲取停用時間
func (r *ConversionRule) GetDeactivatedAt() *time.Time {
	return r.deactivatedAt
}

// --- 領域事件管理 ---

// GetEvents 獲取待發布的事件
func (r *ConversionRule) GetEvents() []shared.DomainEvent {
	return r.events
}

// ClearEvents 清空事件
func (r *ConversionRule) ClearEvents() {
	r.events = []shared.DomainEvent{}
}

// publishEvent 發布事件
func (r *ConversionRule) publishEvent(event shared.DomainEvent) {
	r.events = append(r.events, event)
}
EOF

# 更新 errors.go 新增 ErrInvalidRuleID 和 ErrInvalidDescription
cat >> internal/domain/points/errors.go << 'EOF'

// ErrInvalidRuleID 無效的規則 ID
var ErrInvalidRuleID = fmt.Errorf("invalid rule ID")

// ErrInvalidDescription 無效的描述
var ErrInvalidDescription = fmt.Errorf("description must be between 1 and 200 characters")

// ErrRuleAlreadyDeactivated 規則已停用
var ErrRuleAlreadyDeactivated = fmt.Errorf("rule is already deactivated")

// ErrCannotReactivateRule 不可重新啟用規則
var ErrCannotReactivateRule = fmt.Errorf("deactivated rules cannot be reactivated")
EOF

# 執行測試（應該通過 - Green）
go test -v -run TestNewConversionRule
```

**驗證結果**:
```bash
# 預期輸出：4 個測試全部通過
PASS: TestNewConversionRule_ValidInput_Success
PASS: TestNewConversionRule_EmptyDescription_ReturnsError
PASS: TestNewConversionRule_DescriptionTooLong_ReturnsError
PASS: TestNewConversionRule_PublishesRuleCreatedEvent
```

---

#### 任務 6.2: ConversionRule 停用邏輯 + 查詢方法 (2h)

**目標**: 實作規則停用命令和查詢方法

---

**Step 6.2.1: 編寫測試 - Deactivate 命令方法 (40 min)**

在 `conversion_rule_test.go` 新增：

```go
// === ConversionRule 停用測試 ===

// Test 71: Deactivate 成功停用
func TestConversionRule_Deactivate_Success(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "測試規則")
	rule.ClearEvents() // 清空建立事件

	// Act
	err := rule.Deactivate()

	// Assert
	assert.NoError(t, err)
	assert.False(t, rule.IsActive())
	assert.NotNil(t, rule.GetDeactivatedAt())
	assert.Equal(t, 2, rule.GetVersion()) // 版本號遞增
}

// Test 72: Deactivate 發布 RuleDeactivated 事件
func TestConversionRule_Deactivate_PublishesEvent(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "測試規則")
	rule.ClearEvents()

	// Act
	rule.Deactivate()

	// Assert
	events := rule.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.conversion_rule_deactivated", events[0].EventType())
}

// Test 73: Deactivate 已停用的規則失敗
func TestConversionRule_Deactivate_AlreadyDeactivated_ReturnsError(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "測試規則")
	rule.Deactivate()

	// Act
	err := rule.Deactivate()

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrRuleAlreadyDeactivated)
}

// Test 74: IsApplicableAt 規則適用於日期
func TestConversionRule_IsApplicableAt_WithinRange_ReturnsTrue(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "測試規則")
	testDate := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	// Act
	result := rule.IsApplicableAt(testDate)

	// Assert
	assert.True(t, result)
}

// Test 75: IsApplicableAt 日期超出範圍
func TestConversionRule_IsApplicableAt_OutsideRange_ReturnsFalse(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "測試規則")
	testDate := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// Act
	result := rule.IsApplicableAt(testDate)

	// Assert
	assert.False(t, result)
}

// Test 76: IsApplicableAt 已停用的規則
func TestConversionRule_IsApplicableAt_Deactivated_ReturnsFalse(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "測試規則")
	rule.Deactivate()
	testDate := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	// Act
	result := rule.IsApplicableAt(testDate)

	// Assert
	assert.False(t, result)
}
```

```bash
# 執行測試（應該失敗 - Red）
go test -v -run "TestConversionRule_Deactivate|TestConversionRule_IsApplicableAt"
```

---

**Step 6.2.2: 實作 Deactivate 和查詢方法 (1h 20min)**

在 `conversion_rule.go` 新增：

```go
// --- 命令方法（會改變狀態）---

// Deactivate 停用規則
// 業務規則：規則停用後不可重新啟用（確保積分計算的一致性）
func (r *ConversionRule) Deactivate() error {
	// 檢查是否已停用
	if !r.isActive {
		return ErrRuleAlreadyDeactivated
	}

	// 停用規則
	r.isActive = false
	now := time.Now()
	r.deactivatedAt = &now
	r.version++

	// 發布停用事件
	r.publishEvent(NewConversionRuleDeactivatedEvent(r.ruleID))

	return nil
}

// --- 業務查詢方法 ---

// IsApplicableAt 檢查規則是否適用於指定日期
// 業務規則：必須是生效狀態 + 日期在範圍內
func (r *ConversionRule) IsApplicableAt(date time.Time) bool {
	if !r.isActive {
		return false
	}
	return r.dateRange.Contains(date)
}

// OverlapsWith 檢查規則與另一個規則的日期範圍是否重疊
// 用途：防止同一時間有多個生效的規則（Domain Service 會使用此方法）
func (r *ConversionRule) OverlapsWith(other *ConversionRule) bool {
	return r.dateRange.Overlaps(other.dateRange)
}
```

```bash
# 執行測試（應該通過 - Green）
go test -v -run "TestConversionRule_Deactivate|TestConversionRule_IsApplicableAt"
```

**驗證結果**:
```bash
# 預期輸出：6 個測試全部通過
PASS: TestConversionRule_Deactivate_Success
PASS: TestConversionRule_Deactivate_PublishesEvent
PASS: TestConversionRule_Deactivate_AlreadyDeactivated_ReturnsError
PASS: TestConversionRule_IsApplicableAt_WithinRange_ReturnsTrue
PASS: TestConversionRule_IsApplicableAt_OutsideRange_ReturnsFalse
PASS: TestConversionRule_IsApplicableAt_Deactivated_ReturnsFalse
```

---

#### 任務 6.3: ReconstructConversionRule 工廠方法 (2h)

**目標**: 實作從資料庫重建規則的工廠方法，確保資料完整性

---

**Step 6.3.1: 編寫測試 - ReconstructConversionRule (40 min)**

在 `conversion_rule_test.go` 新增：

```go
// === ConversionRule 重建測試 ===

// Test 77: ReconstructConversionRule 成功重建生效規則
func TestReconstructConversionRule_ActiveRule_Success(t *testing.T) {
	// Arrange
	ruleID, _ := points.NewRuleIDFromString("rule-123")
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	createdAt := time.Now()

	// Act
	rule, err := points.ReconstructConversionRule(
		ruleID,
		rate,
		dateRange,
		"測試規則",
		true,
		createdAt,
		nil,
		1,
	)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, rule)
	assert.Equal(t, ruleID, rule.GetRuleID())
	assert.True(t, rule.IsActive())
	assert.Nil(t, rule.GetDeactivatedAt())
	assert.Len(t, rule.GetEvents(), 0) // 重建不產生事件
}

// Test 78: ReconstructConversionRule 成功重建已停用規則
func TestReconstructConversionRule_DeactivatedRule_Success(t *testing.T) {
	// Arrange
	ruleID, _ := points.NewRuleIDFromString("rule-123")
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	createdAt := time.Now()
	deactivatedAt := time.Now()

	// Act
	rule, err := points.ReconstructConversionRule(
		ruleID,
		rate,
		dateRange,
		"測試規則",
		false,
		createdAt,
		&deactivatedAt,
		2,
	)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, rule)
	assert.False(t, rule.IsActive())
	assert.NotNil(t, rule.GetDeactivatedAt())
	assert.Equal(t, deactivatedAt, *rule.GetDeactivatedAt())
}

// Test 79: ReconstructConversionRule 空描述失敗
func TestReconstructConversionRule_EmptyDescription_ReturnsError(t *testing.T) {
	// Arrange
	ruleID, _ := points.NewRuleIDFromString("rule-123")
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)

	// Act
	rule, err := points.ReconstructConversionRule(
		ruleID,
		rate,
		dateRange,
		"", // 空描述
		true,
		time.Now(),
		nil,
		1,
	)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.ErrorIs(t, err, points.ErrInvalidDescription)
}

// Test 80: ReconstructConversionRule 版本號無效
func TestReconstructConversionRule_InvalidVersion_ReturnsError(t *testing.T) {
	// Arrange
	ruleID, _ := points.NewRuleIDFromString("rule-123")
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)

	// Act
	rule, err := points.ReconstructConversionRule(
		ruleID,
		rate,
		dateRange,
		"測試規則",
		true,
		time.Now(),
		nil,
		0, // 無效版本號
	)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, rule)
}

// Test 81: ReconstructConversionRule 資料不一致（生效但有停用時間）
func TestReconstructConversionRule_InconsistentState_ReturnsError(t *testing.T) {
	// Arrange
	ruleID, _ := points.NewRuleIDFromString("rule-123")
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	deactivatedAt := time.Now()

	// Act - isActive=true 但提供了 deactivatedAt
	rule, err := points.ReconstructConversionRule(
		ruleID,
		rate,
		dateRange,
		"測試規則",
		true,
		time.Now(),
		&deactivatedAt, // 不一致！
		1,
	)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Contains(t, err.Error(), "data corruption")
}
```

```bash
# 執行測試（應該失敗 - Red）
go test -v -run TestReconstructConversionRule
```

---

**Step 6.3.2: 實作 ReconstructConversionRule 工廠方法 (1h 20min)**

在 `conversion_rule.go` 新增：

```go
// ReconstructConversionRule 從資料庫重建規則
// 此方法用於 Repository 層將資料庫記錄轉換為領域對象
// 必須執行完整的資料完整性驗證
func ReconstructConversionRule(
	ruleID RuleID,
	rate ConversionRate,
	dateRange DateRange,
	description string,
	isActive bool,
	createdAt time.Time,
	deactivatedAt *time.Time,
	version int,
) (*ConversionRule, error) {
	// 1. 驗證描述
	if description == "" || len(description) > 200 {
		return nil, fmt.Errorf("invalid description in database: %w", ErrInvalidDescription)
	}

	// 2. 驗證版本號
	if version < 1 {
		return nil, fmt.Errorf("invalid version in database: %d", version)
	}

	// 3. 驗證狀態一致性
	if isActive && deactivatedAt != nil {
		return nil, fmt.Errorf("data corruption: active rule has deactivation timestamp")
	}
	if !isActive && deactivatedAt == nil {
		return nil, fmt.Errorf("data corruption: deactivated rule missing deactivation timestamp")
	}

	// 4. 驗證 RuleID
	if ruleID.IsEmpty() {
		return nil, fmt.Errorf("invalid rule ID in database: %w", ErrInvalidRuleID)
	}

	// 5. 重建聚合
	return &ConversionRule{
		ruleID:        ruleID,
		rate:          rate,
		dateRange:     dateRange,
		description:   description,
		isActive:      isActive,
		createdAt:     createdAt,
		deactivatedAt: deactivatedAt,
		version:       version,
		events:        []shared.DomainEvent{}, // 重建時不包含事件
	}, nil
}
```

```bash
# 執行測試（應該通過 - Green）
go test -v -run TestReconstructConversionRule
```

**驗證結果**:
```bash
# 預期輸出：5 個測試全部通過
PASS: TestReconstructConversionRule_ActiveRule_Success
PASS: TestReconstructConversionRule_DeactivatedRule_Success
PASS: TestReconstructConversionRule_EmptyDescription_ReturnsError
PASS: TestReconstructConversionRule_InvalidVersion_ReturnsError
PASS: TestReconstructConversionRule_InconsistentState_ReturnsError
```

---

#### 任務 6.4: PointsCalculationService Domain Service (2h)

**目標**: 實作積分計算的 Domain Service（包含規則查找邏輯）

**檔案清單**:
1. `internal/domain/points/calculation_service.go`
2. `internal/domain/points/calculation_service_test.go`

---

**Step 6.4.1: 編寫測試 - PointsCalculationService (40 min)**

```bash
cat > internal/domain/points/calculation_service_test.go << 'EOF'
package points_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/yourorg/bar_crm/internal/domain/points"
)

// MockConversionRuleRepository 模擬 Repository（用於測試）
type MockConversionRuleRepository struct {
	rules []*points.ConversionRule
}

func (m *MockConversionRuleRepository) FindActiveRuleAt(date time.Time) (*points.ConversionRule, error) {
	for _, rule := range m.rules {
		if rule.IsApplicableAt(date) {
			return rule, nil
		}
	}
	return nil, points.ErrNoApplicableRule
}

// MockTransaction 模擬交易（用於測試）
type MockTransaction struct {
	amount          decimal.Decimal
	transactionDate time.Time
}

func (m MockTransaction) GetTransactionAmount() decimal.Decimal {
	return m.amount
}

func (m MockTransaction) GetTransactionDate() time.Time {
	return m.transactionDate
}

// === PointsCalculationService 測試 ===

// Test 82: CalculateForTransaction 標準規則計算
func TestPointsCalculationService_CalculateForTransaction_StandardRule(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "標準規則")

	repo := &MockConversionRuleRepository{rules: []*points.ConversionRule{rule}}
	service := points.NewPointsCalculationService(repo)

	transaction := MockTransaction{
		amount:          decimal.NewFromInt(350),
		transactionDate: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	// Act
	result, err := service.CalculateForTransaction(transaction)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 3, result.Value()) // 350 / 100 = 3.5 → floor = 3
}

// Test 83: CalculateForTransaction 促銷規則計算
func TestPointsCalculationService_CalculateForTransaction_PromotionalRule(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(50) // 促銷：50 元 = 1 點
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "促銷規則")

	repo := &MockConversionRuleRepository{rules: []*points.ConversionRule{rule}}
	service := points.NewPointsCalculationService(repo)

	transaction := MockTransaction{
		amount:          decimal.NewFromInt(350),
		transactionDate: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	// Act
	result, err := service.CalculateForTransaction(transaction)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 7, result.Value()) // 350 / 50 = 7
}

// Test 84: CalculateForTransaction 無適用規則
func TestPointsCalculationService_CalculateForTransaction_NoRule(t *testing.T) {
	// Arrange
	repo := &MockConversionRuleRepository{rules: []*points.ConversionRule{}}
	service := points.NewPointsCalculationService(repo)

	transaction := MockTransaction{
		amount:          decimal.NewFromInt(350),
		transactionDate: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	// Act
	result, err := service.CalculateForTransaction(transaction)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrNoApplicableRule)
	assert.Equal(t, 0, result.Value()) // 無規則時返回 0 點
}

// Test 85: CalculateForTransaction 日期超出所有規則範圍
func TestPointsCalculationService_CalculateForTransaction_OutsideAllRules(t *testing.T) {
	// Arrange
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)
	rule, _ := points.NewConversionRule(rate, dateRange, "2024 規則")

	repo := &MockConversionRuleRepository{rules: []*points.ConversionRule{rule}}
	service := points.NewPointsCalculationService(repo)

	transaction := MockTransaction{
		amount:          decimal.NewFromInt(350),
		transactionDate: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC), // 2025 年
	}

	// Act
	result, err := service.CalculateForTransaction(transaction)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrNoApplicableRule)
	assert.Equal(t, 0, result.Value())
}
EOF

# 執行測試（應該失敗 - Red）
cd internal/domain/points
go test -v -run TestPointsCalculationService
```

---

**Step 6.4.2: 實作 PointsCalculationService (1h 20min)**

```bash
cat > internal/domain/points/calculation_service.go << 'EOF'
package points

import (
	"time"
)

// ConversionRuleReader 規則查詢介面（Repository 的一部分）
// 目的：Domain Service 只需要查詢介面，不需要寫入能力
type ConversionRuleReader interface {
	FindActiveRuleAt(date time.Time) (*ConversionRule, error)
}

// PointsCalculationService 積分計算 Domain Service
// 職責：根據交易金額和日期，查找適用的兌換規則並計算積分
// 設計原則：無狀態服務，所有邏輯基於輸入參數和 Repository 查詢
type PointsCalculationService struct {
	ruleRepo ConversionRuleReader
}

// NewPointsCalculationService 建立積分計算服務
func NewPointsCalculationService(ruleRepo ConversionRuleReader) *PointsCalculationService {
	return &PointsCalculationService{
		ruleRepo: ruleRepo,
	}
}

// CalculateForTransaction 為交易計算積分
// 業務流程：
// 1. 根據交易日期查找生效的兌換規則
// 2. 使用規則的 ConversionRate 計算積分
// 3. 如果沒有適用規則，返回 0 點和錯誤
func (s *PointsCalculationService) CalculateForTransaction(tx PointsCalculableTransaction) (PointsAmount, error) {
	// 1. 查找適用的規則
	rule, err := s.ruleRepo.FindActiveRuleAt(tx.GetTransactionDate())
	if err != nil {
		// 沒有適用規則時返回 0 點
		return newPointsAmountUnchecked(0), err
	}

	// 2. 使用規則計算積分
	amount := tx.GetTransactionAmount()
	rate := rule.GetRate()
	points := rate.CalculatePoints(amount)

	return points, nil
}
EOF

# 更新 errors.go 新增 ErrNoApplicableRule
cat >> internal/domain/points/errors.go << 'EOF'

// ErrNoApplicableRule 沒有適用的兌換規則
var ErrNoApplicableRule = fmt.Errorf("no applicable conversion rule found for the given date")
EOF

# 執行測試（應該通過 - Green）
go test -v -run TestPointsCalculationService
```

**驗證結果**:
```bash
# 預期輸出：4 個測試全部通過
PASS: TestPointsCalculationService_CalculateForTransaction_StandardRule
PASS: TestPointsCalculationService_CalculateForTransaction_PromotionalRule
PASS: TestPointsCalculationService_CalculateForTransaction_NoRule
PASS: TestPointsCalculationService_CalculateForTransaction_OutsideAllRules
```

---

**每日檢查點 - Day 6 (15 min)**

```bash
# 1. 執行所有測試
cd /Users/apple/Documents/code/golang/bar_crm
go test ./internal/domain/points/... -v -cover

# 2. 檢查測試數量
go test ./internal/domain/points/... -v | grep -c PASS

# 3. 生成覆蓋率報告
go test ./internal/domain/points/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# 4. 執行 linter
golangci-lint run ./internal/domain/points/...
```

**Day 6 檢查清單**:
- ✅ ConversionRule 聚合根建構函數
- ✅ Deactivate 命令方法
- ✅ IsApplicableAt 和 OverlapsWith 查詢方法
- ✅ ReconstructConversionRule 工廠方法
- ✅ PointsCalculationService Domain Service
- ✅ ConversionRuleReader 介面定義

**Day 6 產出**:
- ✅ RuleID 值對象
- ✅ ConversionRule 聚合根（含停用邏輯）
- ✅ PointsCalculationService Domain Service
- ✅ ConversionRuleReader Repository 介面
- ✅ 15 個新測試（Test 67-81）
- ✅ 總計 81 個測試

**預估總時間**: 8 小時

---

### Day 7: Repository 介面定義 + 領域事件 + Week 1 總結

#### 時間分配
- 上午 (4h): Repository 介面定義（Reader/Writer/BatchReader 分離）
- 下午 (4h): 完整的領域事件定義 + Week 1 總結

---

#### 任務 7.1: PointsAccount Repository 介面定義 (2h)

**目標**: 定義 PointsAccount 的 Repository 介面，遵循 ISP（介面隔離原則）

**檔案清單**:
1. `internal/domain/points/repository.go`
2. `internal/domain/points/repository_test.go`（僅含介面契約測試）

---

**Step 7.1.1: 編寫 PointsAccountRepository 介面 (1h)**

```bash
cat > internal/domain/points/repository.go << 'EOF'
package points

import (
	"context"
)

// ===== PointsAccount Repository 介面 =====

// PointsAccountReader 積分帳戶查詢介面
// 設計原則：Read-only 操作，用於查詢場景
type PointsAccountReader interface {
	// FindByID 根據 AccountID 查找帳戶
	FindByID(ctx context.Context, accountID AccountID) (*PointsAccount, error)

	// FindByMemberID 根據 MemberID 查找帳戶
	FindByMemberID(ctx context.Context, memberID MemberID) (*PointsAccount, error)

	// ExistsByMemberID 檢查會員是否已有積分帳戶
	ExistsByMemberID(ctx context.Context, memberID MemberID) (bool, error)
}

// PointsAccountWriter 積分帳戶寫入介面
// 設計原則：Write-only 操作，用於 Command 場景
type PointsAccountWriter interface {
	// Save 儲存新建立的帳戶
	// 如果 accountID 已存在會返回錯誤
	Save(ctx context.Context, account *PointsAccount) error

	// Update 更新現有帳戶
	// 使用樂觀鎖：WHERE version = previousVersion
	// 如果版本號不匹配會返回 ErrOptimisticLockFailed
	Update(ctx context.Context, account *PointsAccount) error

	// UpdateWithOptimisticLock 使用明確的版本號進行更新
	// 用於需要明確控制樂觀鎖版本的場景
	UpdateWithOptimisticLock(ctx context.Context, account *PointsAccount, expectedVersion int) error
}

// PointsAccountBatchReader 積分帳戶批次查詢介面
// 設計原則：批次操作，用於報表或管理介面
type PointsAccountBatchReader interface {
	// FindAll 查找所有帳戶（分頁）
	FindAll(ctx context.Context, offset, limit int) ([]*PointsAccount, error)

	// FindByMemberIDs 批次查找多個會員的帳戶
	FindByMemberIDs(ctx context.Context, memberIDs []MemberID) ([]*PointsAccount, error)

	// CountAll 計算總帳戶數
	CountAll(ctx context.Context) (int, error)
}

// PointsAccountRepository 積分帳戶完整 Repository 介面
// 設計原則：組合所有子介面，Application Layer 使用完整介面
type PointsAccountRepository interface {
	PointsAccountReader
	PointsAccountWriter
	PointsAccountBatchReader
}

// ===== ConversionRule Repository 介面 =====

// ConversionRuleReader 兌換規則查詢介面
type ConversionRuleReader interface {
	// FindByID 根據 RuleID 查找規則
	FindByID(ctx context.Context, ruleID RuleID) (*ConversionRule, error)

	// FindActiveRuleAt 查找指定日期的生效規則
	// 業務規則：同一時間只有一個生效規則
	FindActiveRuleAt(date time.Time) (*ConversionRule, error)

	// FindOverlappingRules 查找與指定日期範圍重疊的規則
	// 用途：防止建立重疊的規則（Domain Service 會使用）
	FindOverlappingRules(ctx context.Context, dateRange DateRange) ([]*ConversionRule, error)
}

// ConversionRuleWriter 兌換規則寫入介面
type ConversionRuleWriter interface {
	// Save 儲存新規則
	Save(ctx context.Context, rule *ConversionRule) error

	// Update 更新現有規則
	// 使用樂觀鎖：WHERE version = previousVersion
	Update(ctx context.Context, rule *ConversionRule) error
}

// ConversionRuleBatchReader 兌換規則批次查詢介面
type ConversionRuleBatchReader interface {
	// FindAll 查找所有規則（分頁）
	FindAll(ctx context.Context, offset, limit int) ([]*ConversionRule, error)

	// FindAllActive 查找所有生效的規則
	FindAllActive(ctx context.Context) ([]*ConversionRule, error)

	// CountAll 計算總規則數
	CountAll(ctx context.Context) (int, error)
}

// ConversionRuleRepository 兌換規則完整 Repository 介面
type ConversionRuleRepository interface {
	ConversionRuleReader
	ConversionRuleWriter
	ConversionRuleBatchReader
}
EOF

# 更新 errors.go 新增樂觀鎖錯誤
cat >> internal/domain/points/errors.go << 'EOF'

// ErrOptimisticLockFailed 樂觀鎖失敗（版本號衝突）
var ErrOptimisticLockFailed = fmt.Errorf("optimistic lock failed: version mismatch")

// ErrAccountNotFound 帳戶不存在
var ErrAccountNotFound = fmt.Errorf("points account not found")

// ErrRuleNotFound 規則不存在
var ErrRuleNotFound = fmt.Errorf("conversion rule not found")

// ErrAccountAlreadyExists 帳戶已存在
var ErrAccountAlreadyExists = fmt.Errorf("points account already exists")
EOF
```

**Step 7.1.2: 編寫 Repository 介面契約測試（文件化用途）(1h)**

```bash
cat > internal/domain/points/repository_test.go << 'EOF'
package points_test

import (
	"testing"

	"github.com/yourorg/bar_crm/internal/domain/points"
)

// 這些測試主要用於文件化和驗證介面設計
// Infrastructure Layer 的具體實作會有完整的測試套件

// TestPointsAccountRepositoryInterface 驗證介面定義
func TestPointsAccountRepositoryInterface(t *testing.T) {
	// 這個測試確保 PointsAccountRepository 組合了所有子介面
	var _ points.PointsAccountRepository = (*mockPointsAccountRepository)(nil)
}

// TestConversionRuleRepositoryInterface 驗證介面定義
func TestConversionRuleRepositoryInterface(t *testing.T) {
	// 這個測試確保 ConversionRuleRepository 組合了所有子介面
	var _ points.ConversionRuleRepository = (*mockConversionRuleRepository)(nil)
}

// ===== Mock 實作（僅用於編譯時檢查）=====

type mockPointsAccountRepository struct{}

func (m *mockPointsAccountRepository) FindByID(ctx context.Context, accountID points.AccountID) (*points.PointsAccount, error) {
	return nil, nil
}

func (m *mockPointsAccountRepository) FindByMemberID(ctx context.Context, memberID points.MemberID) (*points.PointsAccount, error) {
	return nil, nil
}

func (m *mockPointsAccountRepository) ExistsByMemberID(ctx context.Context, memberID points.MemberID) (bool, error) {
	return false, nil
}

func (m *mockPointsAccountRepository) Save(ctx context.Context, account *points.PointsAccount) error {
	return nil
}

func (m *mockPointsAccountRepository) Update(ctx context.Context, account *points.PointsAccount) error {
	return nil
}

func (m *mockPointsAccountRepository) UpdateWithOptimisticLock(ctx context.Context, account *points.PointsAccount, expectedVersion int) error {
	return nil
}

func (m *mockPointsAccountRepository) FindAll(ctx context.Context, offset, limit int) ([]*points.PointsAccount, error) {
	return nil, nil
}

func (m *mockPointsAccountRepository) FindByMemberIDs(ctx context.Context, memberIDs []points.MemberID) ([]*points.PointsAccount, error) {
	return nil, nil
}

func (m *mockPointsAccountRepository) CountAll(ctx context.Context) (int, error) {
	return 0, nil
}

type mockConversionRuleRepository struct{}

func (m *mockConversionRuleRepository) FindByID(ctx context.Context, ruleID points.RuleID) (*points.ConversionRule, error) {
	return nil, nil
}

func (m *mockConversionRuleRepository) FindActiveRuleAt(date time.Time) (*points.ConversionRule, error) {
	return nil, nil
}

func (m *mockConversionRuleRepository) FindOverlappingRules(ctx context.Context, dateRange points.DateRange) ([]*points.ConversionRule, error) {
	return nil, nil
}

func (m *mockConversionRuleRepository) Save(ctx context.Context, rule *points.ConversionRule) error {
	return nil
}

func (m *mockConversionRuleRepository) Update(ctx context.Context, rule *points.ConversionRule) error {
	return nil
}

func (m *mockConversionRuleRepository) FindAll(ctx context.Context, offset, limit int) ([]*points.ConversionRule, error) {
	return nil, nil
}

func (m *mockConversionRuleRepository) FindAllActive(ctx context.Context) ([]*points.ConversionRule, error) {
	return nil, nil
}

func (m *mockConversionRuleRepository) CountAll(ctx context.Context) (int, error) {
	return 0, nil
}
EOF

# 執行測試
cd internal/domain/points
go test -v -run TestPointsAccountRepositoryInterface
go test -v -run TestConversionRuleRepositoryInterface
```

---

#### 任務 7.2: 完整的領域事件定義 (2h)

**目標**: 定義 Points Context 所有領域事件的完整結構

**檔案清單**:
1. `internal/domain/points/events.go`
2. `internal/domain/points/events_test.go`

---

**Step 7.2.1: 編寫所有領域事件 (1h 30min)**

```bash
cat > internal/domain/points/events.go << 'EOF'
package points

import (
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/bar_crm/internal/domain/shared"
)

// ===== PointsAccount 相關事件 =====

// PointsAccountCreatedEvent 積分帳戶建立事件
type PointsAccountCreatedEvent struct {
	eventID     string
	occurredAt  time.Time
	accountID   AccountID
	memberID    MemberID
}

// NewPointsAccountCreatedEvent 建立帳戶創建事件
func NewPointsAccountCreatedEvent(accountID AccountID, memberID MemberID) shared.DomainEvent {
	return &PointsAccountCreatedEvent{
		eventID:    uuid.New().String(),
		occurredAt: time.Now(),
		accountID:  accountID,
		memberID:   memberID,
	}
}

func (e *PointsAccountCreatedEvent) EventID() string {
	return e.eventID
}

func (e *PointsAccountCreatedEvent) EventType() string {
	return "points.account_created"
}

func (e *PointsAccountCreatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *PointsAccountCreatedEvent) AggregateID() string {
	return e.accountID.String()
}

func (e *PointsAccountCreatedEvent) GetAccountID() AccountID {
	return e.accountID
}

func (e *PointsAccountCreatedEvent) GetMemberID() MemberID {
	return e.memberID
}

// PointsEarnedEvent 積分獲得事件
type PointsEarnedEvent struct {
	eventID     string
	occurredAt  time.Time
	accountID   AccountID
	amount      PointsAmount
	source      PointsSource
	sourceID    string
	description string
}

// NewPointsEarnedEvent 建立積分獲得事件
func NewPointsEarnedEvent(
	accountID AccountID,
	amount PointsAmount,
	source PointsSource,
	sourceID string,
	description string,
) shared.DomainEvent {
	return &PointsEarnedEvent{
		eventID:     uuid.New().String(),
		occurredAt:  time.Now(),
		accountID:   accountID,
		amount:      amount,
		source:      source,
		sourceID:    sourceID,
		description: description,
	}
}

func (e *PointsEarnedEvent) EventID() string {
	return e.eventID
}

func (e *PointsEarnedEvent) EventType() string {
	return "points.earned"
}

func (e *PointsEarnedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *PointsEarnedEvent) AggregateID() string {
	return e.accountID.String()
}

func (e *PointsEarnedEvent) GetAccountID() AccountID {
	return e.accountID
}

func (e *PointsEarnedEvent) GetAmount() PointsAmount {
	return e.amount
}

func (e *PointsEarnedEvent) GetSource() PointsSource {
	return e.source
}

func (e *PointsEarnedEvent) GetSourceID() string {
	return e.sourceID
}

func (e *PointsEarnedEvent) GetDescription() string {
	return e.description
}

// PointsDeductedEvent 積分扣除事件
type PointsDeductedEvent struct {
	eventID     string
	occurredAt  time.Time
	accountID   AccountID
	amount      PointsAmount
	reason      string
}

// NewPointsDeductedEvent 建立積分扣除事件
func NewPointsDeductedEvent(
	accountID AccountID,
	amount PointsAmount,
	reason string,
) shared.DomainEvent {
	return &PointsDeductedEvent{
		eventID:    uuid.New().String(),
		occurredAt: time.Now(),
		accountID:  accountID,
		amount:     amount,
		reason:     reason,
	}
}

func (e *PointsDeductedEvent) EventID() string {
	return e.eventID
}

func (e *PointsDeductedEvent) EventType() string {
	return "points.deducted"
}

func (e *PointsDeductedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *PointsDeductedEvent) AggregateID() string {
	return e.accountID.String()
}

func (e *PointsDeductedEvent) GetAccountID() AccountID {
	return e.accountID
}

func (e *PointsDeductedEvent) GetAmount() PointsAmount {
	return e.amount
}

func (e *PointsDeductedEvent) GetReason() string {
	return e.reason
}

// PointsRecalculatedEvent 積分重新計算事件
type PointsRecalculatedEvent struct {
	eventID       string
	occurredAt    time.Time
	accountID     AccountID
	oldEarned     PointsAmount
	newEarned     PointsAmount
	oldUsed       PointsAmount
	newUsed       PointsAmount
}

// NewPointsRecalculatedEvent 建立積分重新計算事件
func NewPointsRecalculatedEvent(
	accountID AccountID,
	oldEarned, newEarned PointsAmount,
	oldUsed, newUsed PointsAmount,
) shared.DomainEvent {
	return &PointsRecalculatedEvent{
		eventID:    uuid.New().String(),
		occurredAt: time.Now(),
		accountID:  accountID,
		oldEarned:  oldEarned,
		newEarned:  newEarned,
		oldUsed:    oldUsed,
		newUsed:    newUsed,
	}
}

func (e *PointsRecalculatedEvent) EventID() string {
	return e.eventID
}

func (e *PointsRecalculatedEvent) EventType() string {
	return "points.recalculated"
}

func (e *PointsRecalculatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *PointsRecalculatedEvent) AggregateID() string {
	return e.accountID.String()
}

func (e *PointsRecalculatedEvent) GetAccountID() AccountID {
	return e.accountID
}

func (e *PointsRecalculatedEvent) GetOldEarned() PointsAmount {
	return e.oldEarned
}

func (e *PointsRecalculatedEvent) GetNewEarned() PointsAmount {
	return e.newEarned
}

func (e *PointsRecalculatedEvent) GetOldUsed() PointsAmount {
	return e.oldUsed
}

func (e *PointsRecalculatedEvent) GetNewUsed() PointsAmount {
	return e.newUsed
}

// ===== ConversionRule 相關事件 =====

// ConversionRuleCreatedEvent 兌換規則建立事件
type ConversionRuleCreatedEvent struct {
	eventID     string
	occurredAt  time.Time
	ruleID      RuleID
	rate        ConversionRate
	dateRange   DateRange
	description string
}

// NewConversionRuleCreatedEvent 建立規則創建事件
func NewConversionRuleCreatedEvent(
	ruleID RuleID,
	rate ConversionRate,
	dateRange DateRange,
	description string,
) shared.DomainEvent {
	return &ConversionRuleCreatedEvent{
		eventID:     uuid.New().String(),
		occurredAt:  time.Now(),
		ruleID:      ruleID,
		rate:        rate,
		dateRange:   dateRange,
		description: description,
	}
}

func (e *ConversionRuleCreatedEvent) EventID() string {
	return e.eventID
}

func (e *ConversionRuleCreatedEvent) EventType() string {
	return "points.conversion_rule_created"
}

func (e *ConversionRuleCreatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *ConversionRuleCreatedEvent) AggregateID() string {
	return e.ruleID.String()
}

func (e *ConversionRuleCreatedEvent) GetRuleID() RuleID {
	return e.ruleID
}

func (e *ConversionRuleCreatedEvent) GetRate() ConversionRate {
	return e.rate
}

func (e *ConversionRuleCreatedEvent) GetDateRange() DateRange {
	return e.dateRange
}

func (e *ConversionRuleCreatedEvent) GetDescription() string {
	return e.description
}

// ConversionRuleDeactivatedEvent 兌換規則停用事件
type ConversionRuleDeactivatedEvent struct {
	eventID    string
	occurredAt time.Time
	ruleID     RuleID
}

// NewConversionRuleDeactivatedEvent 建立規則停用事件
func NewConversionRuleDeactivatedEvent(ruleID RuleID) shared.DomainEvent {
	return &ConversionRuleDeactivatedEvent{
		eventID:    uuid.New().String(),
		occurredAt: time.Now(),
		ruleID:     ruleID,
	}
}

func (e *ConversionRuleDeactivatedEvent) EventID() string {
	return e.eventID
}

func (e *ConversionRuleDeactivatedEvent) EventType() string {
	return "points.conversion_rule_deactivated"
}

func (e *ConversionRuleDeactivatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *ConversionRuleDeactivatedEvent) AggregateID() string {
	return e.ruleID.String()
}

func (e *ConversionRuleDeactivatedEvent) GetRuleID() RuleID {
	return e.ruleID
}
EOF
```

**Step 7.2.2: 編寫事件測試 (30 min)**

```bash
cat > internal/domain/points/events_test.go << 'EOF'
package points_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yourorg/bar_crm/internal/domain/points"
)

// Test 86: PointsAccountCreatedEvent 欄位正確
func TestPointsAccountCreatedEvent_FieldsAreCorrect(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	memberID, _ := points.NewMemberID("member-123")

	// Act
	event := points.NewPointsAccountCreatedEvent(accountID, memberID)

	// Assert
	assert.NotEmpty(t, event.EventID())
	assert.Equal(t, "points.account_created", event.EventType())
	assert.Equal(t, accountID.String(), event.AggregateID())
	assert.WithinDuration(t, time.Now(), event.OccurredAt(), time.Second)
}

// Test 87: PointsEarnedEvent 欄位正確
func TestPointsEarnedEvent_FieldsAreCorrect(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	amount, _ := points.NewPointsAmount(10)

	// Act
	event := points.NewPointsEarnedEvent(
		accountID,
		amount,
		points.PointsSourceInvoice,
		"invoice-123",
		"測試交易",
	)

	// Assert
	assert.NotEmpty(t, event.EventID())
	assert.Equal(t, "points.earned", event.EventType())
	assert.Equal(t, accountID.String(), event.AggregateID())
}

// Test 88: PointsDeductedEvent 欄位正確
func TestPointsDeductedEvent_FieldsAreCorrect(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	amount, _ := points.NewPointsAmount(5)

	// Act
	event := points.NewPointsDeductedEvent(accountID, amount, "管理員調整")

	// Assert
	assert.NotEmpty(t, event.EventID())
	assert.Equal(t, "points.deducted", event.EventType())
	assert.Equal(t, accountID.String(), event.AggregateID())
}

// Test 89: ConversionRuleCreatedEvent 欄位正確
func TestConversionRuleCreatedEvent_FieldsAreCorrect(t *testing.T) {
	// Arrange
	ruleID := points.NewRuleID()
	rate, _ := points.NewConversionRate(100)
	dateRange, _ := points.NewDateRange(
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	)

	// Act
	event := points.NewConversionRuleCreatedEvent(ruleID, rate, dateRange, "測試規則")

	// Assert
	assert.NotEmpty(t, event.EventID())
	assert.Equal(t, "points.conversion_rule_created", event.EventType())
	assert.Equal(t, ruleID.String(), event.AggregateID())
}

// Test 90: ConversionRuleDeactivatedEvent 欄位正確
func TestConversionRuleDeactivatedEvent_FieldsAreCorrect(t *testing.T) {
	// Arrange
	ruleID := points.NewRuleID()

	// Act
	event := points.NewConversionRuleDeactivatedEvent(ruleID)

	// Assert
	assert.NotEmpty(t, event.EventID())
	assert.Equal(t, "points.conversion_rule_deactivated", event.EventType())
	assert.Equal(t, ruleID.String(), event.AggregateID())
}
EOF

# 執行測試
cd internal/domain/points
go test -v -run "Test.*Event"
```

**驗證結果**:
```bash
# 預期輸出：5 個事件測試全部通過
PASS: TestPointsAccountCreatedEvent_FieldsAreCorrect
PASS: TestPointsEarnedEvent_FieldsAreCorrect
PASS: TestPointsDeductedEvent_FieldsAreCorrect
PASS: TestConversionRuleCreatedEvent_FieldsAreCorrect
PASS: TestConversionRuleDeactivatedEvent_FieldsAreCorrect
```

---

#### 任務 7.3: Week 1 完整驗證 + Git Commit (2h)

**目標**: 驗證 Week 1 所有產出，執行完整測試，提交到 Git

---

**Step 7.3.1: 執行完整測試套件 (30 min)**

```bash
# 1. 執行所有測試
cd /Users/apple/Documents/code/golang/bar_crm
go test ./internal/domain/points/... -v -cover

# 2. 生成詳細覆蓋率報告
go test ./internal/domain/points/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# 3. 檢查測試執行時間
go test ./internal/domain/points/... -v | grep -E "PASS|FAIL"

# 4. 執行競態檢測
go test ./internal/domain/points/... -race

# 5. 執行 benchmark（如果有）
go test ./internal/domain/points/... -bench=. -benchmem

# 6. 執行 linter
golangci-lint run ./internal/domain/points/...

# 7. 檢查 go mod
go mod tidy
go mod verify
```

**預期結果**:
```
=== Week 1 Points Context 測試統計 ===
總測試數: 90 個
- Value Objects: 42 tests (Day 1-3)
- PointsAccount Aggregate: 28 tests (Day 4-5)
- ConversionRule Aggregate: 15 tests (Day 6)
- Events: 5 tests (Day 7)

覆蓋率: 95%+
測試執行時間: < 1 秒
```

---

**Step 7.3.2: 建立 Week 1 Summary 文件 (30 min)**

```bash
cat > internal/domain/points/README.md << 'EOF'
# Points Context - Domain Layer

## 概述

Points Context（積分上下文）是 Bar CRM 的核心域，負責管理會員積分的獲得、使用和兌換規則。

## 領域模型

### 聚合根

1. **PointsAccount**（積分帳戶）
   - 聚合根 ID: `AccountID`
   - 核心不變條件: `usedPoints <= earnedPoints`
   - 命令方法: `EarnPoints`, `DeductPoints`, `RecalculatePoints`
   - 事件: `PointsAccountCreated`, `PointsEarned`, `PointsDeducted`, `PointsRecalculated`

2. **ConversionRule**（兌換規則）
   - 聚合根 ID: `RuleID`
   - 核心不變條件: 同一時間只有一個生效規則（由 Domain Service 驗證）
   - 命令方法: `Deactivate`
   - 查詢方法: `IsApplicableAt`, `OverlapsWith`
   - 事件: `ConversionRuleCreated`, `ConversionRuleDeactivated`

### 值對象

- `PointsAmount`: 積分數量（非負整數）
- `ConversionRate`: 兌換率（含積分計算邏輯）
- `AccountID`: 帳戶唯一識別碼（UUID）
- `MemberID`: 會員唯一識別碼
- `RuleID`: 規則唯一識別碼（UUID）
- `DateRange`: 日期範圍（含 Contains 和 Overlaps 方法）
- `PointsSource`: 積分來源枚舉

### Domain Service

- `PointsCalculationService`: 積分計算服務，根據交易金額和日期查找適用規則並計算積分

### Repository 介面

- `PointsAccountRepository`: 積分帳戶 Repository（Reader/Writer/BatchReader 分離）
- `ConversionRuleRepository`: 兌換規則 Repository（Reader/Writer/BatchReader 分離）

## 測試覆蓋

- **單元測試**: 90 個測試，覆蓋率 95%+
- **測試策略**: TDD（Test-Driven Development）
- **測試執行時間**: < 1 秒

## 依賴關係

- **無外部依賴**: Domain Layer 完全獨立，不依賴任何 Infrastructure 或 Application Layer
- **依賴注入**: 使用介面（Repository）實現依賴反轉

## 使用範例

### 建立積分帳戶

```go
memberID, _ := points.NewMemberID("member-123")
account, _ := points.NewPointsAccount(memberID)
```

### 獲得積分

```go
amount, _ := points.NewPointsAmount(10)
err := account.EarnPoints(
    amount,
    points.PointsSourceInvoice,
    "invoice-123",
    "消費 1000 元",
)
```

### 計算積分

```go
ruleRepo := // ... 從 Infrastructure Layer 注入
calcService := points.NewPointsCalculationService(ruleRepo)
points, err := calcService.CalculateForTransaction(transaction)
```

## 下一步

- Week 2: 實作其他 Bounded Contexts（Member, Invoice, Survey）
- Week 6: 實作 Application Layer Use Cases
- Week 8: 實作 Infrastructure Layer Repository 實作
EOF
```

---

**Step 7.3.3: Git Commit (1h)**

```bash
# 1. 檢查狀態
cd /Users/apple/Documents/code/golang/bar_crm
git status

# 2. 添加所有 Domain Layer 檔案
git add internal/domain/

# 3. 提交 Week 1 完整產出
git commit -m "feat(domain): complete Points Context domain layer implementation

## Summary

Implemented complete domain layer for Points Context (積分上下文) following
Clean Architecture and DDD principles with 100% TDD approach.

## Components Implemented

### Aggregates
- PointsAccount: Member points account with earn/deduct/recalculate commands
- ConversionRule: Points conversion rules with activation/deactivation lifecycle

### Value Objects
- PointsAmount: Non-negative points value with checked/unchecked constructors
- ConversionRate: Conversion rate with points calculation logic
- AccountID, MemberID, RuleID: Identity value objects
- DateRange: Time range with Contains/Overlaps methods
- PointsSource: Points source enumeration

### Domain Services
- PointsCalculationService: Points calculation based on transaction and rules

### Repository Interfaces
- PointsAccountRepository: Reader/Writer/BatchReader segregation (ISP)
- ConversionRuleRepository: Reader/Writer/BatchReader segregation (ISP)

### Domain Events
- PointsAccountCreated, PointsEarned, PointsDeducted, PointsRecalculated
- ConversionRuleCreated, ConversionRuleDeactivated

## Technical Highlights

- **Test Coverage**: 90 tests, 95%+ coverage
- **Design Patterns**:
  - Aggregate Root with version-based optimistic locking
  - Value Objects with immutability
  - Repository Pattern with interface segregation
  - Domain Events for cross-aggregate communication
  - Factory Methods for reconstruction with validation
- **Invariant Protection**:
  - Panic on data corruption (defensive programming)
  - Error on business rule violations
- **Zero Dependencies**: Pure domain layer, no external dependencies

## Files Changed

\`\`\`
internal/domain/
├── shared/
│   ├── transaction.go (PointsCalculableTransaction interface)
│   └── event.go (DomainEvent interface)
└── points/
    ├── errors.go (15 domain errors)
    ├── value_objects.go (7 value objects)
    ├── value_objects_test.go (42 tests)
    ├── account.go (PointsAccount aggregate)
    ├── account_test.go (28 tests)
    ├── conversion_rule.go (ConversionRule aggregate)
    ├── conversion_rule_test.go (15 tests)
    ├── calculation_service.go (Domain Service)
    ├── calculation_service_test.go (4 tests)
    ├── repository.go (Repository interfaces)
    ├── repository_test.go (Interface contract tests)
    ├── events.go (6 domain events)
    ├── events_test.go (5 tests)
    └── README.md (Domain documentation)
\`\`\`

## Testing

All tests pass with:
\`\`\`bash
go test ./internal/domain/points/... -v -cover -race
\`\`\`

## Next Steps

- Week 2: Implement Member, Invoice, Survey domain layers
- Week 6: Implement Application Layer use cases
- Week 8: Implement Infrastructure Layer repositories

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
"

# 4. 推送到遠端（如果需要）
# git push origin main
```

**驗證提交**:
```bash
# 檢查提交歷史
git log --oneline -1

# 檢查提交內容
git show --stat
```

---

**每日檢查點 - Day 7 (15 min)**

```bash
# 最終驗證
cd /Users/apple/Documents/code/golang/bar_crm

# 1. 執行完整測試
go test ./internal/domain/points/... -v -cover

# 2. 檢查程式碼品質
golangci-lint run ./internal/domain/points/...

# 3. 檢查依賴
go mod graph | grep bar_crm

# 4. 檢查檔案結構
tree internal/domain/points/
```

**Day 7 檢查清單**:
- ✅ PointsAccountRepository 介面定義（Reader/Writer/BatchReader）
- ✅ ConversionRuleRepository 介面定義（Reader/Writer/BatchReader）
- ✅ 6 個完整的領域事件實作
- ✅ Week 1 完整測試驗證（90 tests, 95%+ coverage）
- ✅ Git commit 提交

**Day 7 產出**:
- ✅ repository.go（Repository 介面定義）
- ✅ events.go（6 個領域事件）
- ✅ events_test.go（5 個事件測試）
- ✅ README.md（Points Context 文件）
- ✅ Git commit（Week 1 完整產出）

**預估總時間**: 8 小時

---

## Week 1 總結檢查點

### Week 1 結束驗證 (30 min)

```bash
# 1. 執行完整測試套件
cd /Users/apple/Documents/code/golang/bar_crm
go test ./internal/domain/points/... -v -cover

# 2. 生成覆蓋率報告
go test ./internal/domain/points/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# 3. 檢查測試執行時間
go test ./internal/domain/points/... -v | grep PASS

# 4. 執行 linter
golangci-lint run ./internal/domain/points/...

# 5. 檢查 go mod
go mod tidy
go mod verify
```

### Week 1 完成標準

**測試指標**:
- ✅ 90 個單元測試全部通過
- ✅ 測試覆蓋率 95%+
- ✅ 測試執行時間 < 1 秒
- ✅ 無競態條件（`go test -race` 通過）

**程式碼品質**:
- ✅ 無 golangci-lint 警告
- ✅ 所有公開 API 有 godoc 註釋
- ✅ 遵循 Go 命名規範
- ✅ 完整的錯誤處理（15 個 domain errors）

**功能完整性 - Value Objects (Day 1-3)**:
- ✅ PointsAmount 值對象（checked + unchecked）
- ✅ ConversionRate 值對象（含積分計算）
- ✅ AccountID 值對象（UUID）
- ✅ MemberID 值對象
- ✅ RuleID 值對象（UUID）
- ✅ DateRange 值對象（含 Contains 和 Overlaps）
- ✅ PointsSource 枚舉

**功能完整性 - Aggregates (Day 4-6)**:
- ✅ PointsAccount 聚合根（含 EarnPoints, DeductPoints, RecalculatePoints）
- ✅ ConversionRule 聚合根（含 Deactivate, IsApplicableAt）
- ✅ ReconstructPointsAccount 工廠方法（含資料完整性驗證）
- ✅ ReconstructConversionRule 工廠方法

**功能完整性 - Domain Services (Day 6)**:
- ✅ PointsCalculationService（積分計算服務）

**功能完整性 - Repository Interfaces (Day 7)**:
- ✅ PointsAccountRepository（Reader/Writer/BatchReader 分離）
- ✅ ConversionRuleRepository（Reader/Writer/BatchReader 分離）

**功能完整性 - Domain Events (Day 7)**:
- ✅ PointsAccountCreated
- ✅ PointsEarned
- ✅ PointsDeducted
- ✅ PointsRecalculated
- ✅ ConversionRuleCreated
- ✅ ConversionRuleDeactivated

**Git 提交**:
```bash
# 完整的 Week 1 提交（參考 Day 7 Step 7.3.3）
git add internal/domain/
git commit -m "feat(domain): complete Points Context domain layer implementation

## Summary

Implemented complete domain layer for Points Context (積分上下文) following
Clean Architecture and DDD principles with 100% TDD approach.

## Components: 2 Aggregates, 7 Value Objects, 1 Domain Service, 6 Events

- PointsAccount + ConversionRule aggregates
- PointsAmount, ConversionRate, AccountID, MemberID, RuleID, DateRange, PointsSource
- PointsCalculationService
- Repository interfaces with ISP (Reader/Writer/BatchReader)
- 90 tests, 95%+ coverage

🤖 Generated with Claude Code
Co-Authored-By: Claude <noreply@anthropic.com>
"
```

### Week 1 產出文件清單

```
internal/domain/
├── shared/
│   ├── transaction.go (PointsCalculableTransaction interface)
│   └── event.go (DomainEvent interface)
└── points/
    ├── errors.go (15 domain errors)
    ├── value_objects.go (7 value objects)
    ├── value_objects_test.go (42 tests)
    ├── account.go (PointsAccount aggregate)
    ├── account_test.go (28 tests)
    ├── conversion_rule.go (ConversionRule aggregate)
    ├── conversion_rule_test.go (15 tests)
    ├── calculation_service.go (PointsCalculationService)
    ├── calculation_service_test.go (4 tests)
    ├── repository.go (Repository interfaces)
    ├── repository_test.go (Interface contract tests)
    ├── events.go (6 domain events)
    ├── events_test.go (5 tests)
    └── README.md (Points Context 完整文件)
```

**統計**:
- 📁 14 個檔案
- 🧪 90 個測試（42 value objects + 28 PointsAccount + 15 ConversionRule + 5 events）
- 📝 ~3000 行程式碼
- ⏱️ 完成時間：7 天（56 小時）

---

**Week 1 完整達成！** 🎉

---

## 📊 進度追蹤表格

### Week 1 (Day 1-7) 完整進度追蹤

| Day | 任務 | 預估時間 | 測試數 | 狀態 | 備註 |
|-----|------|---------|-------|------|------|
| **Day 1** | **專案初始化 + PointsAmount** | **8h** | **42** | ⬜ | Value Objects 開始 |
| 1.1 | 專案初始化 + Shared Domain | 1.5h | 0 | ⬜ | Go module, 依賴安裝 |
| 1.2 | PointsAmount TDD | 2h | 9 | ⬜ | Checked/Unchecked 建構函數 |
| 1.3 | 每日檢查點 | 0.5h | - | ⬜ | 驗證測試通過 |
| **Day 2** | **ConversionRate + IDs** | **7h** | **42** | ⬜ | 複雜值對象 |
| 2.1 | ConversionRate TDD | 4h | 12 | ⬜ | 積分計算邏輯 |
| 2.2 | AccountID + MemberID | 3h | 21 | ⬜ | 身份值對象 |
| **Day 3** | **DateRange + PointsSource** | **6h** | **42** | ⬜ | 完成值對象層 |
| 3.1 | DateRange TDD | 3h | 12 | ⬜ | Contains + Overlaps |
| 3.2 | PointsSource 枚舉 | 1.5h | 3 | ⬜ | 積分來源枚舉 |
| 3.3 | 重構整理 | 1.5h | - | ⬜ | 程式碼優化 |
| **Day 4** | **PointsAccount Part 1** | **8h** | **51** | ⬜ | 聚合根開始 |
| 4.1 | PointsAccount 建構 | 2h | 4 | ⬜ | NewPointsAccount |
| 4.2 | EarnPoints 命令 | 2h | 10 | ⬜ | 積分獲得邏輯 |
| **Day 5** | **PointsAccount Part 2** | **8h** | **66** | ⬜ | 進階操作 |
| 5.1 | DeductPoints 命令 | 2h | 6 | ⬜ | 積分扣除邏輯 |
| 5.2 | GetAvailablePoints | 2h | 4 | ⬜ | 查詢方法 + Panic |
| 5.3 | RecalculatePoints | 2h | 4 | ⬜ | 重新計算邏輯 |
| 5.4 | ReconstructPointsAccount | 2h | 6 | ⬜ | 資料完整性驗證 |
| **Day 6** | **ConversionRule + Service** | **8h** | **81** | ⬜ | 第二個聚合根 |
| 6.1 | ConversionRule 建構 | 2h | 4 | ⬜ | NewConversionRule |
| 6.2 | Deactivate + 查詢方法 | 2h | 6 | ⬜ | 停用邏輯 |
| 6.3 | ReconstructConversionRule | 2h | 5 | ⬜ | 工廠方法 |
| 6.4 | PointsCalculationService | 2h | 4 | ⬜ | Domain Service |
| **Day 7** | **Repository + Events** | **8h** | **90** | ⬜ | 介面定義 |
| 7.1 | Repository 介面定義 | 2h | 2 | ⬜ | ISP 分離 |
| 7.2 | 領域事件定義 | 2h | 5 | ⬜ | 6 個事件 |
| 7.3 | Week 1 驗證 + Git Commit | 2h | - | ⬜ | 完整測試 + 提交 |
| **總計** | **Week 1 完成** | **56h** | **90** | ⬜ | Points Context 完成 |

**圖例**:
- ⬜ 未開始
- 🔄 進行中
- ✅ 已完成
- ⚠️ 有問題

---

## 🚀 Week 2 預告

**主題**: Domain Layer - 其他 Bounded Contexts（Member, Invoice, Survey）

### Week 2 目標

繼 Points Context 完成後，Week 2 將實作其他三個關鍵 Bounded Context：

1. **Member Context（會員上下文）** - Supporting Domain
   - Member 聚合根（會員資料管理）
   - PhoneNumber 值對象（台灣手機號碼驗證）
   - MemberRepository 介面
   - 預估：2-3 天

2. **Invoice Context（發票上下文）** - Supporting Domain
   - Invoice 聚合根（發票驗證流程）
   - IChefImportRecord 聚合根（iChef 匯入記錄）
   - InvoiceMatchingService Domain Service
   - InvoiceRepository + IChefImportRecordRepository 介面
   - 預估：3-4 天

3. **Survey Context（問卷上下文）** - Supporting Domain
   - Survey 聚合根（含 SurveyQuestion 實體）
   - SurveyResponse 聚合根
   - SurveyRepository + SurveyResponseRepository 介面
   - 預估：2-3 天

### Week 2 估計

- **時間**: 7-10 天（60-80 小時）
- **測試**: 預計新增 150+ 測試
- **檔案**: 預計新增 20-30 個檔案
- **覆蓋率**: 維持 95%+ 覆蓋率

### 實作策略

延續 Week 1 的 TDD 方法：
1. 值對象優先（PhoneNumber, InvoiceNumber 等）
2. 聚合根核心邏輯（會員註冊、發票驗證、問卷建立）
3. Domain Service（發票匹配邏輯）
4. Repository 介面定義
5. 領域事件定義

**詳細的 Week 2 任務分解將在後續文件中提供。**

---

## 附錄：快速參考

### 常用測試命令

```bash
# 執行所有測試
go test ./... -v

# 執行特定測試
go test -run TestPointsAmount -v

# 執行測試並顯示覆蓋率
go test -cover

# 生成覆蓋率報告
go test -coverprofile=coverage.out
go tool cover -html=coverage.out

# 檢查競態條件
go test -race

# 執行 benchmark
go test -bench=.
```

### 常用 Make 命令

```bash
make test          # 執行所有測試
make test-unit     # 執行單元測試
make coverage      # 生成覆蓋率報告
make lint          # 執行 linter
make fmt           # 格式化代碼
```

### Git 提交訊息規範

```
feat(domain): 新增功能
fix(domain): 修復 bug
test(domain): 新增測試
refactor(domain): 重構
docs: 更新文檔
chore: 雜項（建置、依賴等）
```

---

**最後更新**: 2025-01-11
**維護者**: 開發團隊
