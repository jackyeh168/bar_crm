# 09 - 生產環境安全防護機制

## 目錄
- [1. Panic Recovery 機制](#1-panic-recovery-機制)
- [2. 告警與監控](#2-告警與監控)
- [3. 降級策略](#3-降級策略)
- [4. 健康檢查](#4-健康檢查)
- [5. 錯誤追蹤與日誌](#5-錯誤追蹤與日誌)
- [6. 生產環境最佳實踐](#6-生產環境最佳實踐)

---

## 1. Panic Recovery 機制

### 1.1 為什麼需要 Panic Recovery？

根據 Clean Architecture 的錯誤處理策略：
- **業務錯誤**（可預期）→ 返回 `error`
- **程序錯誤**（不變條件違反）→ `panic`

在開發和測試環境中，panic 應該立即暴露問題。但在生產環境中，我們需要：
1. 捕獲 panic，避免整個服務崩潰
2. 記錄詳細錯誤信息和堆棧跟蹤
3. 觸發告警，通知運維團隊
4. 返回友好的錯誤響應給用戶

### 1.2 HTTP Panic Recovery Middleware

#### 文件位置
```
internal/presentation/http/middleware/recovery.go
```

#### 實現代碼

```go
package middleware

import (
    "fmt"
    "runtime/debug"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "github.com/yourorg/bar_crm/internal/infrastructure/monitoring"
)

// RecoveryMiddleware 捕獲 panic 並進行錯誤處理
// 這是生產環境的最後一道防線，防止不變條件違反導致服務崩潰
func RecoveryMiddleware(logger *zap.Logger, alerter monitoring.Alerter) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                // 1. 捕獲堆棧跟蹤
                stackTrace := string(debug.Stack())

                // 2. 記錄詳細錯誤（包含請求上下文）
                logger.Error("Panic recovered - Invariant violation or critical error",
                    zap.Any("error", err),
                    zap.String("path", c.Request.URL.Path),
                    zap.String("method", c.Request.Method),
                    zap.String("client_ip", c.ClientIP()),
                    zap.String("user_agent", c.Request.UserAgent()),
                    zap.String("request_id", c.GetString("request_id")),
                    zap.String("stack_trace", stackTrace),
                    zap.Time("timestamp", time.Now()),
                )

                // 3. 觸發告警（生產環境）
                if gin.Mode() == gin.ReleaseMode {
                    alerter.SendCriticalAlert(monitoring.Alert{
                        Level:      monitoring.LevelCritical,
                        Title:      "Panic Recovered - Invariant Violation",
                        Message:    fmt.Sprintf("Panic: %v", err),
                        Service:    "bar_crm_api",
                        Path:       c.Request.URL.Path,
                        StackTrace: stackTrace,
                        Timestamp:  time.Now(),
                    })
                }

                // 4. 增加監控指標
                monitoring.PanicCounter.Inc()
                monitoring.PanicByEndpoint.WithLabelValues(c.Request.URL.Path).Inc()

                // 5. 返回通用錯誤給用戶（不暴露內部細節）
                c.JSON(500, gin.H{
                    "success": false,
                    "error":   "internal_server_error",
                    "message": "An unexpected error occurred. Our team has been notified.",
                    "request_id": c.GetString("request_id"), // 用於用戶報告問題
                })

                // 6. 終止請求處理
                c.Abort()
            }
        }()

        c.Next()
    }
}
```

#### 使用方式

```go
// cmd/app/main.go
func setupRouter(
    logger *zap.Logger,
    alerter monitoring.Alerter,
    handlers *handler.Handlers,
) *gin.Engine {
    router := gin.New()

    // 1. Recovery Middleware 必須在所有中間件之前註冊
    router.Use(middleware.RecoveryMiddleware(logger, alerter))

    // 2. 其他中間件
    router.Use(middleware.RequestIDMiddleware())
    router.Use(middleware.LoggerMiddleware(logger))
    router.Use(middleware.CORSMiddleware())

    // 3. 路由註冊
    // ...

    return router
}
```

### 1.3 Application Layer Panic Recovery

對於 Use Case 層的 panic（非 HTTP 請求觸發的場景），例如：
- 定時任務
- 事件處理器
- 背景任務

需要在每個入口點進行 recovery：

```go
// internal/application/events/points_earned_handler.go
package events

import (
    "runtime/debug"
    "go.uber.org/zap"
)

type PointsEarnedHandler struct {
    logger  *zap.Logger
    alerter monitoring.Alerter
    // ...
}

func (h *PointsEarnedHandler) Handle(event shared.DomainEvent) error {
    // Panic Recovery for Event Handler
    defer func() {
        if err := recover(); err != nil {
            h.logger.Error("Panic in event handler",
                zap.String("event_type", event.EventType()),
                zap.Any("error", err),
                zap.String("stack_trace", string(debug.Stack())),
            )

            h.alerter.SendCriticalAlert(monitoring.Alert{
                Level:   monitoring.LevelCritical,
                Title:   "Event Handler Panic",
                Message: fmt.Sprintf("Panic in %s: %v", event.EventType(), err),
            })
        }
    }()

    // 實際業務邏輯
    return h.handlePointsEarned(event)
}
```

---

## 2. 告警與監控

### 2.1 告警系統介面

#### 文件位置
```
internal/infrastructure/monitoring/alerter.go
```

#### 接口定義

```go
package monitoring

import "time"

// AlertLevel 告警級別
type AlertLevel string

const (
    LevelCritical AlertLevel = "critical" // 需要立即處理（不變條件違反、panic）
    LevelError    AlertLevel = "error"    // 需要關注（業務錯誤頻繁）
    LevelWarning  AlertLevel = "warning"  // 警告（性能下降）
    LevelInfo     AlertLevel = "info"     // 信息（配置變更）
)

// Alert 告警結構
type Alert struct {
    Level      AlertLevel
    Title      string
    Message    string
    Service    string
    Path       string
    StackTrace string
    Metadata   map[string]interface{}
    Timestamp  time.Time
}

// Alerter 告警介面
type Alerter interface {
    // SendCriticalAlert 發送緊急告警（不變條件違反、panic）
    SendCriticalAlert(alert Alert) error

    // SendErrorAlert 發送錯誤告警（業務錯誤頻繁）
    SendErrorAlert(alert Alert) error

    // SendWarningAlert 發送警告告警（性能下降）
    SendWarningAlert(alert Alert) error
}
```

### 2.2 告警實現（Slack）

```go
// internal/infrastructure/monitoring/slack_alerter.go
package monitoring

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type SlackAlerter struct {
    webhookURL string
    httpClient *http.Client
}

func NewSlackAlerter(webhookURL string) *SlackAlerter {
    return &SlackAlerter{
        webhookURL: webhookURL,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

func (a *SlackAlerter) SendCriticalAlert(alert Alert) error {
    // Slack Message 格式
    message := map[string]interface{}{
        "text": fmt.Sprintf("🚨 CRITICAL: %s", alert.Title),
        "blocks": []map[string]interface{}{
            {
                "type": "header",
                "text": map[string]string{
                    "type": "plain_text",
                    "text": fmt.Sprintf("🚨 %s", alert.Title),
                },
            },
            {
                "type": "section",
                "fields": []map[string]string{
                    {"type": "mrkdwn", "text": fmt.Sprintf("*Service:*\n%s", alert.Service)},
                    {"type": "mrkdwn", "text": fmt.Sprintf("*Level:*\n%s", alert.Level)},
                    {"type": "mrkdwn", "text": fmt.Sprintf("*Path:*\n%s", alert.Path)},
                    {"type": "mrkdwn", "text": fmt.Sprintf("*Time:*\n%s", alert.Timestamp.Format("2006-01-02 15:04:05"))},
                },
            },
            {
                "type": "section",
                "text": map[string]string{
                    "type": "mrkdwn",
                    "text": fmt.Sprintf("*Message:*\n```%s```", alert.Message),
                },
            },
        },
    }

    // 如果有堆棧跟蹤，添加到消息中（截取前 1000 字符）
    if alert.StackTrace != "" {
        stackTrace := alert.StackTrace
        if len(stackTrace) > 1000 {
            stackTrace = stackTrace[:1000] + "\n... (truncated)"
        }

        message["blocks"] = append(message["blocks"].([]map[string]interface{}), map[string]interface{}{
            "type": "section",
            "text": map[string]string{
                "type": "mrkdwn",
                "text": fmt.Sprintf("*Stack Trace:*\n```%s```", stackTrace),
            },
        })
    }

    // 發送到 Slack
    body, _ := json.Marshal(message)
    resp, err := a.httpClient.Post(a.webhookURL, "application/json", bytes.NewBuffer(body))
    if err != nil {
        return fmt.Errorf("failed to send Slack alert: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Slack API returned non-200 status: %d", resp.StatusCode)
    }

    return nil
}

func (a *SlackAlerter) SendErrorAlert(alert Alert) error {
    // 類似實現，使用不同的圖標（⚠️）
    // ...
}

func (a *SlackAlerter) SendWarningAlert(alert Alert) error {
    // 類似實現，使用不同的圖標（⚡）
    // ...
}
```

### 2.3 Prometheus 監控指標

```go
// internal/infrastructure/monitoring/metrics.go
package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Panic 計數器
    PanicCounter = promauto.NewCounter(prometheus.CounterOpts{
        Name: "bar_crm_panic_total",
        Help: "Total number of panics recovered",
    })

    // 按端點統計的 Panic 計數器
    PanicByEndpoint = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "bar_crm_panic_by_endpoint_total",
            Help: "Total number of panics by endpoint",
        },
        []string{"endpoint"},
    )

    // 不變條件違反計數器
    InvariantViolationCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "bar_crm_invariant_violation_total",
            Help: "Total number of invariant violations",
        },
        []string{"aggregate", "invariant"},
    )

    // 並發衝突計數器（樂觀鎖）
    ConcurrencyConflictCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "bar_crm_concurrency_conflict_total",
            Help: "Total number of optimistic lock conflicts",
        },
        []string{"aggregate"},
    )
)
```

### 2.4 告警規則配置（Prometheus）

```yaml
# prometheus/alerts.yml
groups:
  - name: bar_crm_critical
    interval: 30s
    rules:
      # Panic 告警
      - alert: PanicDetected
        expr: rate(bar_crm_panic_total[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Panic detected in bar_crm service"
          description: "{{ $value }} panics per second in the last 5 minutes"

      # 不變條件違反告警
      - alert: InvariantViolation
        expr: rate(bar_crm_invariant_violation_total[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Invariant violation detected"
          description: "Data corruption detected in {{ $labels.aggregate }}"

      # 高並發衝突率告警
      - alert: HighConcurrencyConflicts
        expr: rate(bar_crm_concurrency_conflict_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High concurrency conflict rate"
          description: "{{ $value }} conflicts per second for {{ $labels.aggregate }}"
```

---

## 3. 降級策略

### 3.1 服務降級場景

當發生以下情況時，應該啟動降級策略：

1. **數據損壞檢測到** - 不變條件違反
2. **外部服務不可用** - LINE API、iChef、Redis
3. **數據庫性能下降** - 連接池耗盡、查詢超時

### 3.2 降級實現

```go
// internal/application/usecases/points/earn_points_graceful.go
package points

import (
    "context"
    "errors"
)

type GracefulEarnPointsUseCase struct {
    primaryUseCase   *EarnPointsUseCase
    circuitBreaker   *CircuitBreaker
    fallbackStrategy FallbackStrategy
}

// Execute 執行積分獲得（帶降級）
func (uc *GracefulEarnPointsUseCase) Execute(
    ctx context.Context,
    cmd EarnPointsCommand,
) (*EarnPointsResult, error) {
    // 1. 檢查熔斷器狀態
    if uc.circuitBreaker.IsOpen() {
        return uc.fallbackStrategy.HandleEarnPoints(cmd)
    }

    // 2. 執行主邏輯
    result, err := uc.primaryUseCase.Execute(ctx, cmd)

    // 3. 根據錯誤類型決定是否記錄失敗
    if err != nil {
        // 不變條件違反或數據庫錯誤 -> 記錄失敗
        if isInvariantViolation(err) || isDatabaseError(err) {
            uc.circuitBreaker.RecordFailure()

            // 降級：使用備用策略
            return uc.fallbackStrategy.HandleEarnPoints(cmd)
        }

        // 業務錯誤 -> 直接返回，不觸發降級
        return nil, err
    }

    // 4. 成功 -> 記錄成功
    uc.circuitBreaker.RecordSuccess()
    return result, nil
}

// FallbackStrategy 降級策略介面
type FallbackStrategy interface {
    // HandleEarnPoints 降級處理積分獲得
    // 例如：寫入消息隊列，延遲處理
    HandleEarnPoints(cmd EarnPointsCommand) (*EarnPointsResult, error)
}
```

### 3.3 熔斷器實現

```go
// internal/infrastructure/resilience/circuit_breaker.go
package resilience

import (
    "sync"
    "time"
)

type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration

    failures    int
    lastFailure time.Time
    state       CircuitBreakerState
    mu          sync.RWMutex
}

type CircuitBreakerState string

const (
    StateClosed   CircuitBreakerState = "closed"   // 正常狀態
    StateOpen     CircuitBreakerState = "open"     // 熔斷狀態
    StateHalfOpen CircuitBreakerState = "half_open" // 半開狀態（嘗試恢復）
)

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        maxFailures:  maxFailures,
        resetTimeout: resetTimeout,
        state:        StateClosed,
    }
}

func (cb *CircuitBreaker) IsOpen() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    // 檢查是否需要從 Open 切換到 HalfOpen
    if cb.state == StateOpen && time.Since(cb.lastFailure) > cb.resetTimeout {
        cb.state = StateHalfOpen
        return false
    }

    return cb.state == StateOpen
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures = 0
    cb.state = StateClosed
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    cb.failures++
    cb.lastFailure = time.Now()

    if cb.failures >= cb.maxFailures {
        cb.state = StateOpen
    }
}
```

---

## 4. 健康檢查

### 4.1 健康檢查端點

```go
// internal/presentation/http/handlers/health_handler.go
package handlers

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type HealthHandler struct {
    db          *gorm.DB
    redisClient *redis.Client
}

// HealthCheck 健康檢查（輕量級）
func (h *HealthHandler) HealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status": "healthy",
        "time":   time.Now().Format(time.RFC3339),
    })
}

// ReadinessCheck 就緒檢查（檢查依賴服務）
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
    checks := map[string]string{
        "database": h.checkDatabase(),
        "redis":    h.checkRedis(),
    }

    // 判斷整體狀態
    allHealthy := true
    for _, status := range checks {
        if status != "healthy" {
            allHealthy = false
            break
        }
    }

    statusCode := http.StatusOK
    if !allHealthy {
        statusCode = http.StatusServiceUnavailable
    }

    c.JSON(statusCode, gin.H{
        "status": map[string]interface{}{
            "overall": allHealthy,
            "checks":  checks,
        },
        "time": time.Now().Format(time.RFC3339),
    })
}

func (h *HealthHandler) checkDatabase() string {
    sqlDB, err := h.db.DB()
    if err != nil {
        return "unhealthy"
    }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    if err := sqlDB.PingContext(ctx); err != nil {
        return "unhealthy"
    }

    return "healthy"
}

func (h *HealthHandler) checkRedis() string {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    if err := h.redisClient.Ping(ctx).Err(); err != nil {
        return "unhealthy"
    }

    return "healthy"
}
```

### 4.2 Kubernetes 健康檢查配置

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bar-crm-api
spec:
  template:
    spec:
      containers:
      - name: api
        image: bar-crm-api:latest
        ports:
        - containerPort: 8080

        # 存活探針（Liveness Probe）- 檢查服務是否存活
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 2
          failureThreshold: 3

        # 就緒探針（Readiness Probe）- 檢查服務是否就緒
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 2
          failureThreshold: 3
```

---

## 5. 錯誤追蹤與日誌

### 5.1 結構化日誌

```go
// internal/infrastructure/logging/logger.go
package logging

import (
    "os"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func NewLogger(env string) (*zap.Logger, error) {
    var config zap.Config

    if env == "production" {
        config = zap.NewProductionConfig()
        config.EncoderConfig.TimeKey = "timestamp"
        config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    } else {
        config = zap.NewDevelopmentConfig()
        config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    }

    // 添加堆棧跟蹤（僅 Error 及以上級別）
    config.EncoderConfig.StacktraceKey = "stacktrace"

    return config.Build(
        zap.AddCaller(),
        zap.AddStacktrace(zapcore.ErrorLevel),
    )
}
```

### 5.2 分佈式追蹤（OpenTelemetry）

```go
// internal/infrastructure/tracing/tracer.go
package tracing

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/resource"
    tracesdk "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func InitTracer(serviceName, jaegerEndpoint string) (*tracesdk.TracerProvider, error) {
    // 創建 Jaeger exporter
    exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
    if err != nil {
        return nil, err
    }

    tp := tracesdk.NewTracerProvider(
        tracesdk.WithBatcher(exp),
        tracesdk.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}
```

---

## 6. 生產環境最佳實踐

### 6.1 配置管理

```bash
# .env.production
# 服務配置
GIN_MODE=release
PORT=8080

# 資料庫配置
DATABASE_URL=postgresql://user:pass@host:5432/dbname?sslmode=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=5m

# Redis 配置
REDIS_ADDR=redis:6379
REDIS_PASSWORD=secret
REDIS_DB=0

# 告警配置
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
ALERTING_ENABLED=true

# 監控配置
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
METRICS_PORT=9090

# 降級配置
CIRCUIT_BREAKER_MAX_FAILURES=5
CIRCUIT_BREAKER_RESET_TIMEOUT=60s
```

### 6.2 部署清單

**部署前檢查清單**:
- [ ] Recovery Middleware 已啟用
- [ ] 告警系統已配置（Slack Webhook）
- [ ] Prometheus 監控已啟用
- [ ] 健康檢查端點可訪問
- [ ] 日誌級別設置為 INFO（生產環境）
- [ ] 數據庫連接池配置正確
- [ ] Redis fallback 機制已測試
- [ ] Jaeger 分佈式追蹤已啟用
- [ ] Kubernetes Liveness/Readiness Probe 已配置

### 6.3 運維手冊

#### 發現 Panic 告警時的處理流程

1. **確認告警嚴重性**
   - 檢查 Slack 告警消息中的堆棧跟蹤
   - 確認是否為不變條件違反（數據損壞）

2. **立即響應**
   - 如果是數據損壞：立即啟動數據修復流程
   - 如果是代碼 bug：評估是否需要回滾

3. **分析根因**
   - 查看 Jaeger 分佈式追蹤
   - 檢查相關日誌（使用 request_id 關聯）
   - 檢查 Prometheus 指標

4. **修復與驗證**
   - 修復代碼或數據
   - 部署修復版本
   - 監控告警是否再次觸發

#### 不變條件違反的調試步驟

```bash
# 1. 查看最近的 panic 日誌
kubectl logs -l app=bar-crm-api --tail=1000 | grep "Panic recovered"

# 2. 查看指標（使用 Prometheus）
# 訪問 http://prometheus:9090
# 查詢: bar_crm_invariant_violation_total

# 3. 查看 Jaeger 追蹤
# 訪問 http://jaeger:16686
# 搜索 request_id

# 4. 檢查數據庫數據
psql $DATABASE_URL -c "
  SELECT * FROM points_accounts
  WHERE used_points > earned_points;
"
```

### 6.4 性能優化建議

1. **數據庫查詢優化**
   - 為常用查詢添加索引
   - 使用連接池（max_open_conns: 25）
   - 啟用查詢緩存（Redis）

2. **並發控制**
   - 樂觀鎖重試機制（最多 3 次）
   - 監控並發衝突率
   - 高衝突場景考慮使用悲觀鎖

3. **緩存策略**
   - 積分餘額緩存（TTL: 5 分鐘）
   - 會員資料緩存（TTL: 30 分鐘）
   - 積分轉換規則緩存（TTL: 1 小時）

---

## 總結

生產環境安全防護機制的核心原則：

1. **Fail Fast + Graceful Degradation** - 快速失敗但優雅降級
2. **Observability** - 可觀測性是生產環境的基礎
3. **Defense in Depth** - 多層防禦（Recovery + Circuit Breaker + Alerting）
4. **Learn from Failures** - 從每次 panic 中學習並改進

**記住**：panic 不是壞事，它是發現數據損壞和邏輯錯誤的最快方式。關鍵是要有完善的監控和告警機制，在問題發生時立即響應。
