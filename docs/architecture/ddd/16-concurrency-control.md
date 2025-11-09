# Concurrency Control Strategy（併發控制策略）

> **版本**: 1.0
> **最後更新**: 2025-01-09
> **目的**: 定義系統的併發控制機制，防止數據競爭和狀態不一致

本章節說明如何使用**樂觀鎖（Optimistic Locking）**處理併發修改，確保數據一致性。

---

## **16.1 為什麼需要併發控制？**

### **問題場景：積分賬戶併發修改**

```
時間軸：
T1: User A 查詢積分帳戶 (earnedPoints = 100, usedPoints = 0)
T2: User B 查詢積分帳戶 (earnedPoints = 100, usedPoints = 0)
T3: User A 賺取 50 積分 → 更新為 earnedPoints = 150
T4: User B 使用 30 積分 → 更新為 usedPoints = 30
     但基於舊狀態 (earnedPoints = 100)

結果：User B 的更新覆蓋了 User A 的變更
實際狀態：earnedPoints = 100, usedPoints = 30
正確狀態：earnedPoints = 150, usedPoints = 30
```

**問題**：**Lost Update Problem**（更新丟失）

---

## **16.2 樂觀鎖 vs 悲觀鎖**

| 特性 | 樂觀鎖 (Optimistic Locking) | 悲觀鎖 (Pessimistic Locking) |
|------|---------------------------|----------------------------|
| **假設** | 衝突很少發生 | 衝突經常發生 |
| **機制** | Version 欄位檢測衝突 | 資料庫行鎖 (SELECT FOR UPDATE) |
| **性能** | ✅ 高（無鎖等待） | ❌ 低（鎖等待） |
| **適用場景** | 讀多寫少 | 寫多讀少 |
| **衝突處理** | 重試或返回錯誤 | 阻塞等待 |
| **數據庫支持** | ✅ 所有數據庫 | PostgreSQL, MySQL (InnoDB) |

**我們的選擇**: **樂觀鎖**

**理由**:
- 積分系統讀多寫少（查詢積分 >> 更新積分）
- 避免長時間鎖表（提高併發性能）
- GORM 原生支持樂觀鎖（`Version` 欄位）
- 衝突時可自動重試（用戶體驗較好）

---

## **16.3 樂觀鎖實現**

### **16.3.1 Domain Layer - 版本控制**

```go
// Domain Layer - internal/domain/points/aggregate.go
package points

type PointsAccount struct {
    accountID    AccountID
    memberID     member.MemberID
    earnedPoints PointsAmount
    usedPoints   PointsAmount
    version      int           // ✅ 樂觀鎖版本號
    lastUpdatedAt time.Time
}

// Version 返回當前版本號
func (a *PointsAccount) Version() int {
    return a.version
}

// IncrementVersion 遞增版本號（由 Repository 調用）
func (a *PointsAccount) IncrementVersion() {
    a.version++
}

// EarnPoints 賺取積分（業務邏輯）
func (a *PointsAccount) EarnPoints(
    amount PointsAmount,
    source PointsSource,
    sourceID string,
    description string,
) error {
    if amount.Value() < 0 {
        return ErrNegativePointsAmount
    }

    a.earnedPoints = a.earnedPoints.Add(amount)
    a.lastUpdatedAt = time.Now()

    // ✅ 版本號不在這裡遞增（由 Repository 處理）
    // 原因：版本號是基礎設施關注點，Domain Layer 不應直接管理

    a.publishEvent(PointsEarned{
        AccountID:   a.accountID,
        Amount:      amount,
        Source:      source,
        SourceID:    sourceID,
        Description: description,
    })

    return nil
}
```

---

### **16.3.2 Infrastructure Layer - GORM 樂觀鎖**

```go
// Infrastructure Layer - internal/infrastructure/persistence/points_account_model.go
package persistence

import "gorm.io/gorm"

// PointsAccountModel GORM 模型
type PointsAccountModel struct {
    AccountID    string `gorm:"primaryKey;type:varchar(50)"`
    MemberID     string `gorm:"type:varchar(50);not null;uniqueIndex"`
    EarnedPoints int    `gorm:"not null;default:0"`
    UsedPoints   int    `gorm:"not null;default:0"`
    Version      int    `gorm:"not null;default:1"` // ✅ 樂觀鎖欄位
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

func (PointsAccountModel) TableName() string {
    return "points_accounts"
}
```

```go
// Infrastructure Layer - internal/infrastructure/persistence/points_account_repository.go
package persistence

import (
    "gorm.io/gorm"
    "myapp/internal/domain/points"
    "myapp/internal/domain/points/repository"
    "myapp/internal/domain/shared"
)

type GormPointsAccountRepository struct {
    db *gorm.DB
}

// Update 更新積分帳戶（使用樂觀鎖）
func (r *GormPointsAccountRepository) Update(
    ctx shared.TransactionContext,
    account *points.PointsAccount,
) error {
    db := r.extractDB(ctx)

    // 1. 將 Domain 實體轉換為 GORM 模型
    model := r.toModel(account)

    // 2. 使用樂觀鎖更新（WHERE 條件包含 version）
    result := db.Model(&PointsAccountModel{}).
        Where("account_id = ? AND version = ?", model.AccountID, account.Version()).
        Updates(map[string]interface{}{
            "earned_points": model.EarnedPoints,
            "used_points":   model.UsedPoints,
            "version":       account.Version() + 1,  // ✅ 遞增版本號
            "updated_at":    time.Now(),
        })

    // 3. 檢查是否發生併發衝突
    if result.Error != nil {
        return result.Error
    }

    if result.RowsAffected == 0 {
        // ✅ 沒有行被更新 → 版本號不匹配 → 併發衝突
        return repository.ErrConcurrentModification
    }

    // 4. 更新成功，遞增 Domain 實體的版本號
    account.IncrementVersion()

    return nil
}

// toModel 將 Domain 實體轉換為 GORM 模型
func (r *GormPointsAccountRepository) toModel(account *points.PointsAccount) *PointsAccountModel {
    return &PointsAccountModel{
        AccountID:    account.AccountID().String(),
        MemberID:     account.MemberID().String(),
        EarnedPoints: account.EarnedPoints().Value(),
        UsedPoints:   account.UsedPoints().Value(),
        Version:      account.Version(),
    }
}

// extractDB 從 TransactionContext 提取 DB 連接
func (r *GormPointsAccountRepository) extractDB(ctx shared.TransactionContext) *gorm.DB {
    if txCtx, ok := ctx.(*transaction.gormTransactionContext); ok {
        return txCtx.tx
    }
    return r.db
}
```

---

### **16.3.3 Application Layer - 重試策略**

```go
// Application Layer - internal/application/points/earn_points_use_case.go
package pointsapp

import (
    "myapp/internal/domain/points/repository"
    "myapp/internal/domain/shared"
)

type EarnPointsUseCase struct {
    txManager   shared.TransactionManager
    accountRepo repository.PointsAccountRepository
}

func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    // ✅ 使用指數退避重試（最多 3 次）
    return uc.retryWithExponentialBackoff(3, func() error {
        return uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
            // 1. 查詢積分帳戶（帶版本號）
            account, err := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
            if err != nil {
                return err
            }

            // 2. 業務邏輯
            err = account.EarnPoints(cmd.Amount, cmd.Source, cmd.SourceID, cmd.Description)
            if err != nil {
                return err
            }

            // 3. 保存更新（樂觀鎖檢查）
            err = uc.accountRepo.Update(ctx, account)
            if err != nil {
                if errors.Is(err, repository.ErrConcurrentModification) {
                    // ✅ 併發衝突 → 重試
                    return err
                }
                return err
            }

            return nil
        })
    })
}

// retryWithExponentialBackoff 指數退避重試
func (uc *EarnPointsUseCase) retryWithExponentialBackoff(
    maxRetries int,
    fn func() error,
) error {
    var lastErr error

    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := fn()

        if err == nil {
            return nil // ✅ 成功
        }

        if !errors.Is(err, repository.ErrConcurrentModification) {
            return err // ❌ 非併發錯誤，直接返回
        }

        lastErr = err

        if attempt < maxRetries {
            // 指數退避：2^attempt * 100ms
            backoffDuration := time.Duration(1<<attempt) * 100 * time.Millisecond
            time.Sleep(backoffDuration)

            // 記錄重試日誌
            log.Warn("Concurrent modification detected, retrying...",
                "attempt", attempt,
                "maxRetries", maxRetries,
                "backoff", backoffDuration,
            )
        }
    }

    // ❌ 超過最大重試次數
    return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

---

## **16.4 錯誤處理**

### **16.4.1 Domain Layer - 錯誤定義**

```go
// Domain Layer - internal/domain/points/repository/errors.go
package repository

import "errors"

var (
    // ErrConcurrentModification 併發修改衝突
    ErrConcurrentModification = errors.New("concurrent modification detected: version mismatch")

    // ErrAccountNotFound 帳戶不存在
    ErrAccountNotFound = errors.New("points account not found")
)
```

---

### **16.4.2 Presentation Layer - HTTP 錯誤映射**

```go
// Presentation Layer - internal/presentation/http/error_mapper.go
package http

import (
    "github.com/gin-gonic/gin"
    "myapp/internal/domain/points/repository"
)

func MapErrorToHTTP(c *gin.Context, err error) {
    switch {
    case errors.Is(err, repository.ErrConcurrentModification):
        // ✅ 併發衝突 → HTTP 409 Conflict
        c.JSON(409, gin.H{
            "error": "CONCURRENT_MODIFICATION",
            "message": "資料已被其他用戶修改，請重新載入後再試",
        })

    case errors.Is(err, repository.ErrAccountNotFound):
        c.JSON(404, gin.H{
            "error": "ACCOUNT_NOT_FOUND",
            "message": "積分帳戶不存在",
        })

    default:
        c.JSON(500, gin.H{
            "error": "INTERNAL_SERVER_ERROR",
            "message": "系統錯誤，請稍後再試",
        })
    }
}
```

---

## **16.5 測試策略**

### **16.5.1 單元測試 - Repository 樂觀鎖**

```go
// Infrastructure Layer - internal/infrastructure/persistence/points_account_repository_test.go
package persistence_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestGormPointsAccountRepository_Update_ConcurrentModification(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    repo := NewGormPointsAccountRepository(db)

    // 創建初始帳戶 (version = 1)
    account := points.NewPointsAccount(accountID, memberID, 100)
    repo.Create(nil, account)

    // Act
    // Scenario 1: User A 查詢帳戶
    accountA, _ := repo.FindByMemberID(nil, memberID)
    assert.Equal(t, 1, accountA.Version())

    // Scenario 2: User B 查詢帳戶
    accountB, _ := repo.FindByMemberID(nil, memberID)
    assert.Equal(t, 1, accountB.Version())

    // Scenario 3: User A 先更新（成功）
    accountA.EarnPoints(50, SourceInvoice, "TX001", "Test")
    err := repo.Update(nil, accountA)
    assert.NoError(t, err)
    assert.Equal(t, 2, accountA.Version()) // ✅ 版本遞增

    // Scenario 4: User B 嘗試更新（失敗 - 版本號不匹配）
    accountB.EarnPoints(30, SourceInvoice, "TX002", "Test")
    err = repo.Update(nil, accountB)
    assert.Error(t, err)
    assert.ErrorIs(t, err, repository.ErrConcurrentModification)

    // Assert
    // 驗證最終狀態
    finalAccount, _ := repo.FindByMemberID(nil, memberID)
    assert.Equal(t, 150, finalAccount.EarnedPoints().Value()) // ✅ 只有 User A 的更新
    assert.Equal(t, 2, finalAccount.Version())
}
```

---

### **16.5.2 整合測試 - Use Case 重試機制**

```go
// Application Layer - internal/application/points/earn_points_use_case_test.go
package pointsapp_test

import (
    "testing"
    "sync"
)

func TestEarnPointsUseCase_Execute_ConcurrentRetry(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    uc := setupUseCase(db)

    // Act: 同時執行兩個操作
    var wg sync.WaitGroup
    errors := make(chan error, 2)

    wg.Add(2)

    go func() {
        defer wg.Done()
        errors <- uc.Execute(EarnPointsCommand{
            MemberID: memberID,
            Amount:   50,
            Source:   SourceInvoice,
            SourceID: "TX001",
        })
    }()

    go func() {
        defer wg.Done()
        errors <- uc.Execute(EarnPointsCommand{
            MemberID: memberID,
            Amount:   30,
            Source:   SourceInvoice,
            SourceID: "TX002",
        })
    }()

    wg.Wait()
    close(errors)

    // Assert
    // 兩個操作都應該成功（其中一個會重試）
    errorCount := 0
    for err := range errors {
        if err != nil {
            errorCount++
        }
    }

    assert.Equal(t, 0, errorCount, "Both operations should succeed with retry")

    // 驗證最終狀態
    account, _ := repo.FindByMemberID(nil, memberID)
    assert.Equal(t, 80, account.EarnedPoints().Value()) // 50 + 30
}
```

---

## **16.6 性能考量**

### **16.6.1 樂觀鎖的開銷**

| 操作 | 開銷 | 說明 |
|------|------|------|
| **讀取** | 無額外開銷 | 只是多讀一個 `version` 欄位 |
| **更新** | WHERE 條件多一個欄位 | `WHERE version = ?` |
| **衝突** | 重試開銷 | 最多 3 次重試，每次指數退避 |

**結論**: 樂觀鎖的性能開銷極小（< 1% CPU）

---

### **16.6.2 重試策略的影響**

**指數退避時間**:
- 第 1 次重試: 200ms
- 第 2 次重試: 400ms
- 第 3 次重試: 800ms

**最壞情況**: 3 次重試 = 1.4 秒

**實際場景**:
- 併發衝突率 < 1%（積分系統讀多寫少）
- 99.9% 的請求無需重試
- P99 延遲影響 < 50ms

---

## **16.7 監控與告警**

### **16.7.1 監控指標**

```go
// Application Layer - internal/application/points/metrics.go
package pointsapp

import "github.com/prometheus/client_golang/prometheus"

var (
    // 併發衝突次數
    concurrentModificationCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "points_concurrent_modification_total",
            Help: "Total number of concurrent modification conflicts",
        },
        []string{"use_case"},
    )

    // 重試次數分佈
    retryCountHistogram = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "points_retry_count",
            Help:    "Number of retries for concurrent modifications",
            Buckets: []float64{0, 1, 2, 3},
        },
        []string{"use_case"},
    )
)

func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    retryCount := 0

    err := uc.retryWithExponentialBackoff(3, func() error {
        retryCount++
        // ... 業務邏輯
    })

    // 記錄指標
    retryCountHistogram.WithLabelValues("EarnPoints").Observe(float64(retryCount))

    if errors.Is(err, repository.ErrConcurrentModification) {
        concurrentModificationCounter.WithLabelValues("EarnPoints").Inc()
    }

    return err
}
```

---

### **16.7.2 告警規則**

```yaml
# Prometheus Alert Rules
groups:
  - name: concurrency_control
    rules:
      # 併發衝突率過高
      - alert: HighConcurrentModificationRate
        expr: rate(points_concurrent_modification_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High concurrent modification rate detected"
          description: "{{ $value }} conflicts per second in the last 5 minutes"

      # 重試失敗率過高
      - alert: HighRetryFailureRate
        expr: rate(points_retry_count{le="3"}[5m]) > 5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High retry failure rate detected"
          description: "More than 5 requests per second are failing after max retries"
```

---

## **16.8 最佳實踐**

### **✅ DO (推薦做法)**

1. **使用樂觀鎖處理聚合更新**
   - 所有 Aggregate Root 都應包含 `version` 欄位
   - Repository Update 方法必須檢查版本號

2. **實現自動重試機制**
   - 使用指數退避（避免雷鳴效應）
   - 最多重試 3 次
   - 記錄重試日誌和指標

3. **返回語義化錯誤**
   - `ErrConcurrentModification` 明確表示併發衝突
   - HTTP 409 Conflict（而非 500 Internal Server Error）

4. **監控併發衝突率**
   - 衝突率 > 5% → 考慮增加分片或緩存
   - 重試失敗率 > 1% → 需要優化業務邏輯

---

### **❌ DON'T (避免做法)**

1. **不要在 Domain Layer 直接操作 `version`**
   - ❌ 錯誤: `a.version++`
   - ✅ 正確: Repository 負責版本遞增

2. **不要無限重試**
   - ❌ 錯誤: `for { retry() }`
   - ✅ 正確: 最多重試 3 次，超過則返回錯誤

3. **不要使用悲觀鎖（除非必要）**
   - ❌ 錯誤: `SELECT FOR UPDATE`（降低併發性能）
   - ✅ 正確: 樂觀鎖 + 重試（適合讀多寫少場景）

4. **不要忽略 `ErrConcurrentModification`**
   - ❌ 錯誤: 直接返回 500 錯誤
   - ✅ 正確: 映射為 409 Conflict，提示用戶重新載入

---

## **16.9 相關章節**

- **Chapter 4**: Tactical Design（Aggregate 設計原則）
- **Chapter 7**: Aggregate Design Principles（聚合邊界與版本控制）
- **Chapter 11**: Dependency Rules（Repository 接口設計）
- **Chapter 15**: Testing Strategy（併發測試策略）

---

## **16.10 參考資料**

- Martin Fowler - "Optimistic Offline Lock"
- Eric Evans - "Domain-Driven Design" (Chapter 6: Aggregates)
- GORM Documentation - "Advanced Topics: Optimistic Locking"
- Go Concurrency Patterns - "Exponential Backoff"
