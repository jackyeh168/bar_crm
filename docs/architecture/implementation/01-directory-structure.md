# 目錄結構設計

> **版本**: 1.0
> **最後更新**: 2025-01-10
> **原則**: 依據 Clean Architecture 分層原則，遵循 Go 語言慣例

---

## 1. 完整目錄結構

```
bar_crm/
├── cmd/                                    # 應用程序入口
│   ├── app/
│   │   └── main.go                         # 主應用啟動
│   └── migrate/
│       └── main.go                         # 數據庫遷移工具
│
├── internal/                               # 私有代碼（不可被外部 import）
│   │
│   ├── domain/                             # 領域層（Domain Layer）
│   │   ├── shared/                         # 跨上下文的共享概念
│   │   │   ├── transaction.go             # Transaction Context 接口
│   │   │   ├── event.go                   # Domain Event 基礎接口
│   │   │   └── errors.go                  # 通用領域錯誤
│   │   │
│   │   ├── member/                         # 會員管理上下文
│   │   │   ├── member.go                  # Member 聚合根
│   │   │   ├── value_objects.go           # MemberID, PhoneNumber, etc.
│   │   │   ├── events.go                  # MemberRegistered, PhoneNumberBound
│   │   │   ├── errors.go                  # ErrMemberNotFound, etc.
│   │   │   ├── service.go                 # MemberRegistrationService
│   │   │   └── repository/                # Repository 接口（屬於 Domain）
│   │   │       └── member_repository.go   # Interface: MemberRepository
│   │   │
│   │   ├── points/                         # 積分管理上下文（核心域）
│   │   │   ├── account.go                 # PointsAccount 聚合根
│   │   │   ├── conversion_rule.go         # ConversionRule 聚合根
│   │   │   ├── value_objects.go           # PointsAmount, ConversionRate, etc.
│   │   │   ├── events.go                  # PointsEarned, PointsDeducted, etc.
│   │   │   ├── errors.go                  # ErrInsufficientPoints, etc.
│   │   │   ├── calculation_service.go     # PointsCalculationService
│   │   │   ├── recalculation_service.go   # PointsRecalculationService
│   │   │   └── repository/
│   │   │       ├── account_repository.go  # Interface: PointsAccountRepository
│   │   │       └── rule_repository.go     # Interface: ConversionRuleRepository
│   │   │
│   │   ├── invoice/                        # 發票處理上下文
│   │   │   ├── transaction.go             # InvoiceTransaction 聚合根
│   │   │   ├── value_objects.go           # InvoiceNumber, Money, etc.
│   │   │   ├── events.go                  # TransactionVerified, etc.
│   │   │   ├── errors.go                  # ErrInvoiceDuplicate, etc.
│   │   │   ├── parsing_service.go         # InvoiceParsingService
│   │   │   ├── validation_service.go      # InvoiceValidationService
│   │   │   └── repository/
│   │   │       └── transaction_repository.go
│   │   │
│   │   ├── survey/                         # 問卷管理上下文
│   │   │   ├── survey.go                  # Survey 聚合根
│   │   │   ├── response.go                # SurveyResponse 聚合根
│   │   │   ├── value_objects.go           # QuestionType, RatingScore, etc.
│   │   │   ├── events.go                  # SurveyActivated, etc.
│   │   │   ├── errors.go
│   │   │   └── repository/
│   │   │       ├── survey_repository.go
│   │   │       └── response_repository.go
│   │   │
│   │   ├── external/                       # 外部系統整合上下文
│   │   │   ├── import_batch.go            # ImportBatch 聚合根
│   │   │   ├── import_record.go           # ImportedInvoiceRecord 實體
│   │   │   ├── value_objects.go           # ImportStatistics, MatchStatus, etc.
│   │   │   ├── events.go                  # BatchImportCompleted, etc.
│   │   │   ├── errors.go
│   │   │   ├── matching_service.go        # InvoiceMatchingService
│   │   │   └── repository/
│   │   │       └── import_repository.go
│   │   │
│   │   ├── identity/                       # 身份與訪問上下文
│   │   │   ├── admin_user.go              # AdminUser 聚合根
│   │   │   ├── value_objects.go           # Role, Permission, etc.
│   │   │   ├── errors.go
│   │   │   └── repository/
│   │   │       └── admin_repository.go
│   │   │
│   │   ├── notification/                   # 通知服務上下文
│   │   │   ├── notification.go            # Notification 聚合根
│   │   │   ├── value_objects.go           # MessageContent, NotificationType, etc.
│   │   │   ├── events.go
│   │   │   ├── errors.go
│   │   │   └── repository/
│   │   │       └── notification_repository.go
│   │   │
│   │   └── audit/                          # 稽核追蹤上下文
│   │       ├── audit_log.go               # AuditLog 聚合根（不可變）
│   │       ├── value_objects.go           # Actor, Target, Changes, etc.
│   │       ├── errors.go                  # ErrAuditLogWriteFailed, etc.
│   │       └── repository/                # 分離的 Repository 接口
│   │           ├── writer.go              # AuditLogWriter 接口
│   │           ├── reader.go              # AuditLogReader 接口
│   │           ├── statistics.go          # AuditLogStatistics 接口
│   │           └── archiver.go            # AuditLogArchiver 接口
│   │
│   ├── application/                        # 應用層（Application Layer）
│   │   ├── usecases/                      # Use Cases（業務用例）
│   │   │   ├── member/                    # 會員相關 Use Cases
│   │   │   │   ├── register_member.go    # RegisterMemberUseCase
│   │   │   │   ├── bind_phone.go         # BindPhoneNumberUseCase
│   │   │   │   └── get_member.go         # GetMemberUseCase
│   │   │   │
│   │   │   ├── points/                    # 積分相關 Use Cases
│   │   │   │   ├── earn_points.go        # EarnPointsUseCase
│   │   │   │   ├── query_points.go       # QueryPointsUseCase
│   │   │   │   ├── recalculate_points.go # RecalculatePointsUseCase
│   │   │   │   ├── create_rule.go        # CreateConversionRuleUseCase
│   │   │   │   └── update_rule.go        # UpdateConversionRuleUseCase
│   │   │   │
│   │   │   ├── invoice/                   # 發票相關 Use Cases
│   │   │   │   ├── scan_invoice.go       # ScanInvoiceUseCase
│   │   │   │   ├── verify_transaction.go # VerifyTransactionUseCase
│   │   │   │   └── query_transactions.go # QueryTransactionsUseCase
│   │   │   │
│   │   │   ├── survey/                    # 問卷相關 Use Cases
│   │   │   │   ├── create_survey.go      # CreateSurveyUseCase
│   │   │   │   ├── submit_response.go    # SubmitSurveyResponseUseCase
│   │   │   │   └── activate_survey.go    # ActivateSurveyUseCase
│   │   │   │
│   │   │   ├── external/                  # 外部系統相關 Use Cases
│   │   │   │   └── import_ichef_batch.go # ImportIChefBatchUseCase
│   │   │   │
│   │   │   └── audit/                     # 稽核相關 Use Cases
│   │   │       ├── query_audit_logs.go   # QueryAuditLogsUseCase
│   │   │       └── export_audit_report.go # ExportAuditReportUseCase
│   │   │
│   │   ├── dto/                           # Data Transfer Objects
│   │   │   ├── member_dto.go             # MemberDTO
│   │   │   ├── points_dto.go             # PointsAccountDTO, TransactionDTO
│   │   │   ├── invoice_dto.go            # VerifiedTransactionDTO
│   │   │   ├── survey_dto.go             # SurveyDTO, ResponseDTO
│   │   │   └── audit_dto.go              # AuditLogDTO
│   │   │
│   │   └── events/                        # Event Handlers（事件處理器）
│   │       ├── points/                    # 積分相關事件處理
│   │       │   ├── transaction_verified_handler.go  # 處理 TransactionVerified
│   │       │   └── survey_completed_handler.go      # 處理 SurveyResponseSubmitted
│   │       │
│   │       ├── audit/                     # 稽核相關事件處理
│   │       │   ├── member_event_handler.go          # 記錄會員相關事件
│   │       │   ├── points_event_handler.go          # 記錄積分相關事件
│   │       │   └── transaction_event_handler.go     # 記錄交易相關事件
│   │       │
│   │       └── notification/              # 通知相關事件處理
│   │           ├── welcome_notification_handler.go  # 發送歡迎訊息
│   │           └── points_earned_handler.go         # 發送積分獲得通知
│   │
│   ├── infrastructure/                     # 基礎設施層（Infrastructure Layer）
│   │   ├── persistence/                   # 持久化實現（GORM）
│   │   │   ├── gorm/                      # GORM 相關
│   │   │   │   ├── models.go             # GORM 模型定義（所有表）
│   │   │   │   ├── migrations.go         # Auto-Migration 邏輯
│   │   │   │   └── transaction.go        # GORM Transaction Context 實現
│   │   │   │
│   │   │   ├── member/                    # 會員 Repository 實現
│   │   │   │   └── member_repository.go  # GormMemberRepository
│   │   │   │
│   │   │   ├── points/                    # 積分 Repository 實現
│   │   │   │   ├── account_repository.go # GormPointsAccountRepository
│   │   │   │   └── rule_repository.go    # GormConversionRuleRepository
│   │   │   │
│   │   │   ├── invoice/                   # 發票 Repository 實現
│   │   │   │   └── transaction_repository.go
│   │   │   │
│   │   │   ├── survey/                    # 問卷 Repository 實現
│   │   │   │   ├── survey_repository.go
│   │   │   │   └── response_repository.go
│   │   │   │
│   │   │   ├── external/                  # 外部系統 Repository 實現
│   │   │   │   └── import_repository.go
│   │   │   │
│   │   │   ├── identity/                  # 身份 Repository 實現
│   │   │   │   └── admin_repository.go
│   │   │   │
│   │   │   ├── notification/              # 通知 Repository 實現
│   │   │   │   └── notification_repository.go
│   │   │   │
│   │   │   └── audit/                     # 稽核 Repository 實現
│   │   │       └── audit_log_repository.go  # 實現所有 4 個接口
│   │   │
│   │   ├── cache/                         # 緩存實現（Redis）
│   │   │   ├── redis_client.go           # Redis 客戶端封裝
│   │   │   ├── points_cache.go           # 積分緩存
│   │   │   └── inmemory_cache.go         # In-Memory Fallback
│   │   │
│   │   ├── external/                      # 外部服務適配器
│   │   │   ├── linebot/                   # LINE Bot SDK 適配器
│   │   │   │   ├── adapter.go            # LineUserAdapter（Anti-Corruption Layer）
│   │   │   │   └── webhook.go            # Webhook 事件處理
│   │   │   │
│   │   │   ├── google/                    # Google OAuth 適配器
│   │   │   │   └── oauth.go              # OAuth2 實現
│   │   │   │
│   │   │   └── ichef/                     # iChef POS 適配器
│   │   │       └── excel_parser.go       # Excel 解析器
│   │   │
│   │   ├── messaging/                     # 事件總線實現（僅技術機制）
│   │   │   ├── event_bus.go              # In-Memory Event Bus
│   │   │   ├── subscriber.go             # Event Subscriber 管理
│   │   │   └── publisher.go              # Event Publisher
│   │   │
│   │   └── config/                        # 配置管理
│   │       ├── config.go                 # 配置結構定義
│   │       └── loader.go                 # 環境變量加載
│   │
│   └── presentation/                       # 展示層（Presentation Layer）
│       ├── http/                          # HTTP API Handlers
│       │   ├── server.go                 # Gin Server 配置
│       │   ├── middleware/               # 中間件
│       │   │   ├── auth.go               # 認證中間件
│       │   │   ├── logger.go             # 日誌中間件
│       │   │   └── error_handler.go      # 錯誤處理中間件
│       │   │
│       │   ├── handlers/                 # HTTP Handlers
│       │   │   ├── member_handler.go     # 會員相關端點
│       │   │   ├── points_handler.go     # 積分相關端點
│       │   │   ├── invoice_handler.go    # 發票相關端點
│       │   │   ├── survey_handler.go     # 問卷相關端點
│       │   │   ├── admin_handler.go      # 管理後台端點
│       │   │   └── health_handler.go     # 健康檢查端點
│       │   │
│       │   └── responses/                # HTTP 響應結構
│       │       ├── success.go            # 成功響應格式
│       │       └── error.go              # 錯誤響應格式
│       │
│       └── linebot/                       # LINE Bot Handlers
│           ├── webhook_handler.go        # LINE Webhook 處理
│           ├── message_handler.go        # 訊息處理
│           └── event_router.go           # 事件路由
│
├── test/                                   # 測試代碼
│   ├── integration/                       # 集成測試
│   │   ├── member_test.go                # 會員集成測試
│   │   ├── points_test.go                # 積分集成測試
│   │   └── invoice_test.go               # 發票集成測試
│   │
│   ├── e2e/                               # 端到端測試
│   │   ├── scan_and_earn_test.go        # 掃描發票獲得積分流程
│   │   ├── survey_reward_test.go        # 問卷獎勵流程
│   │   └── ichef_import_test.go         # iChef 匯入流程
│   │
│   ├── fixtures/                          # 測試數據
│   │   ├── members.json                  # 測試會員數據
│   │   ├── invoices.json                 # 測試發票數據
│   │   └── ichef_sample.xlsx             # iChef 範例檔案
│   │
│   └── mocks/                             # Mock 實現
│       ├── repository/                    # Mock Repositories
│       │   ├── member_repository_mock.go
│       │   └── points_repository_mock.go
│       │
│       └── external/                      # Mock 外部服務
│           └── linebot_mock.go
│
├── scripts/                                # 腳本工具
│   ├── setup.sh                           # 環境設置腳本
│   ├── backup-db.sh                       # 數據庫備份
│   └── run-tests.sh                       # 測試執行腳本
│
├── configs/                                # 配置文件
│   ├── config.yaml                        # 默認配置
│   ├── config.dev.yaml                    # 開發環境配置
│   └── config.prod.yaml                   # 生產環境配置
│
├── deployments/                            # 部署配置
│   ├── docker/
│   │   ├── Dockerfile                    # 應用鏡像
│   │   └── docker-compose.yml            # 本地開發環境
│   │
│   └── k8s/                               # Kubernetes 配置（未來）
│       ├── deployment.yaml
│       └── service.yaml
│
├── docs/                                   # 文檔（當前目錄）
│   ├── architecture/                      # 架構設計
│   ├── product/                           # 產品需求
│   ├── operations/                        # 運維文檔
│   └── qa/                                # 測試策略
│
├── go.mod                                  # Go Modules 依賴
├── go.sum                                  # 依賴校驗和
├── Makefile                                # Make 命令
├── .gitignore                              # Git 忽略規則
├── .golangci.yml                           # Go Linter 配置
├── README.md                               # 項目說明
├── CLAUDE.md                               # Claude Code 指南
└── LICENSE                                 # 許可證
```

---

## 2. 各層級目錄詳解

### 2.1 Domain Layer（領域層）

**位置**: `internal/domain/`

**組織原則**:
- ✅ **按限界上下文劃分**：每個上下文一個目錄
- ✅ **Repository 接口屬於 Domain**：定義在 `repository/` 子目錄
- ✅ **無外部依賴**：只能依賴標準庫和同層其他包
- ✅ **統一命名**：值對象、事件、錯誤統一命名

**文件命名規範**:
- `聚合名.go` - 聚合根實現（例如 `account.go`, `member.go`）
- `value_objects.go` - 所有值對象定義
- `events.go` - 所有領域事件定義
- `errors.go` - 所有領域錯誤定義
- `*_service.go` - 領域服務（例如 `calculation_service.go`）
- `repository/` - Repository 接口目錄

**示例** (Points Context):
```
internal/domain/points/
├── account.go                 # PointsAccount 聚合根
├── conversion_rule.go         # ConversionRule 聚合根
├── value_objects.go           # PointsAmount, ConversionRate, DateRange
├── events.go                  # PointsEarned, PointsDeducted, PointsRecalculated
├── errors.go                  # ErrInsufficientPoints, ErrNegativePointsAmount
├── calculation_service.go     # PointsCalculationService
├── recalculation_service.go   # PointsRecalculationService
└── repository/
    ├── account_repository.go  # Interface: PointsAccountRepository
    └── rule_repository.go     # Interface: ConversionRuleRepository
```

---

### 2.2 Application Layer（應用層）

**位置**: `internal/application/`

**組織原則**:
- ✅ **Use Cases 按上下文劃分**：每個上下文一個子目錄
- ✅ **一個文件一個 Use Case**：清晰的單一職責
- ✅ **DTO 集中管理**：所有 DTO 在 `dto/` 目錄
- ✅ **Event Handlers 按職責劃分**：積分、稽核、通知分開

**文件命名規範**:
- `動詞_名詞.go` - Use Case 文件（例如 `earn_points.go`, `register_member.go`）
- `*_dto.go` - DTO 文件（例如 `member_dto.go`, `points_dto.go`）
- `*_handler.go` - Event Handler 文件（例如 `transaction_verified_handler.go`）

**Use Case 命名**:
```go
// ✅ 正確：清晰的業務動作
type EarnPointsUseCase struct {}
type RecalculatePointsUseCase struct {}
type ScanInvoiceUseCase struct {}

// ❌ 錯誤：過於技術化或模糊
type PointsService struct {}  // 不明確的服務名
type Handler struct {}        // 過於通用
```

**示例** (Application Layer):
```
internal/application/
├── usecases/
│   ├── points/
│   │   ├── earn_points.go        # EarnPointsUseCase
│   │   ├── recalculate_points.go # RecalculatePointsUseCase
│   │   └── query_points.go       # QueryPointsUseCase
│   │
│   └── member/
│       ├── register_member.go    # RegisterMemberUseCase
│       └── bind_phone.go         # BindPhoneNumberUseCase
│
├── dto/
│   ├── points_dto.go             # PointsAccountDTO, TransactionDTO
│   └── member_dto.go             # MemberDTO
│
└── events/
    ├── points/
    │   └── transaction_verified_handler.go
    │
    └── audit/
        └── points_event_handler.go
```

---

### 2.3 Infrastructure Layer（基礎設施層）

**位置**: `internal/infrastructure/`

**組織原則**:
- ✅ **按技術類型劃分**：persistence, cache, external, events
- ✅ **Repository 實現按上下文劃分**：與 Domain 結構對應
- ✅ **GORM 模型集中定義**：所有表結構在 `gorm/models.go`
- ✅ **外部服務適配器隔離**：每個外部服務一個子目錄

**文件命名規範**:
- `*_repository.go` - Repository 實現（例如 `member_repository.go`）
- `*_adapter.go` - 外部服務適配器（例如 `linebot_adapter.go`）
- `*_cache.go` - 緩存實現（例如 `points_cache.go`）
- `models.go` - GORM 模型定義
- `transaction.go` - Transaction Context 實現

**GORM 模型命名**:
```go
// ✅ 正確：模型名加 "Model" 後綴
type PointsAccountModel struct {
    gorm.Model
    // ...
}

type MemberModel struct {
    gorm.Model
    // ...
}

// ❌ 錯誤：與 Domain 實體名稱衝突
type PointsAccount struct {  // 與 Domain 實體同名
    gorm.Model
}
```

**示例** (Infrastructure Layer):
```
internal/infrastructure/
├── persistence/
│   ├── gorm/
│   │   ├── models.go             # 所有 GORM 模型定義
│   │   ├── migrations.go         # Auto-Migration
│   │   └── transaction.go        # GormTransactionContext
│   │
│   ├── points/
│   │   ├── account_repository.go # GormPointsAccountRepository
│   │   └── rule_repository.go    # GormConversionRuleRepository
│   │
│   └── member/
│       └── member_repository.go  # GormMemberRepository
│
├── cache/
│   ├── redis_client.go           # Redis 客戶端
│   └── points_cache.go           # 積分緩存邏輯
│
├── external/
│   ├── linebot/
│   │   ├── adapter.go            # LineUserAdapter（ACL）
│   │   └── webhook.go            # Webhook 處理
│   │
│   └── google/
│       └── oauth.go              # Google OAuth2
│
└── events/
    ├── event_bus.go              # In-Memory Event Bus
    └── subscriber.go             # Event Subscriber
```

---

### 2.4 Presentation Layer（展示層）

**位置**: `internal/presentation/`

**組織原則**:
- ✅ **按協議類型劃分**：http, linebot, grpc（未來）
- ✅ **Handlers 按上下文劃分**：每個上下文一個 Handler 文件
- ✅ **中間件集中管理**：統一的認證、日誌、錯誤處理
- ✅ **響應格式統一**：success, error 結構標準化

**文件命名規範**:
- `*_handler.go` - HTTP Handler（例如 `points_handler.go`）
- `server.go` - Server 配置與啟動
- `middleware/*.go` - 中間件（auth.go, logger.go）
- `responses/*.go` - 響應結構（success.go, error.go）

**Handler 結構**:
```go
// ✅ 正確：每個 Handler 處理一個上下文的所有端點
type PointsHandler struct {
    earnPointsUseCase       *usecases.EarnPointsUseCase
    queryPointsUseCase      *usecases.QueryPointsUseCase
    recalculatePointsUseCase *usecases.RecalculatePointsUseCase
}

func (h *PointsHandler) RegisterRoutes(router *gin.Engine) {
    points := router.Group("/api/v1/points")
    points.POST("/earn", h.HandleEarnPoints)
    points.GET("/balance/:memberID", h.HandleQueryBalance)
    points.POST("/recalculate", h.HandleRecalculate)
}

// ❌ 錯誤：過於分散
type EarnPointsHandler struct {}  // 只處理一個端點，過於細分
```

**示例** (Presentation Layer):
```
internal/presentation/
├── http/
│   ├── server.go                 # Gin Server 配置
│   │
│   ├── middleware/
│   │   ├── auth.go               # JWT 認證中間件
│   │   ├── logger.go             # 請求日誌中間件
│   │   └── error_handler.go      # 統一錯誤處理
│   │
│   ├── handlers/
│   │   ├── points_handler.go     # 積分端點（3-5 個方法）
│   │   ├── member_handler.go     # 會員端點
│   │   └── invoice_handler.go    # 發票端點
│   │
│   └── responses/
│       ├── success.go            # type SuccessResponse struct {}
│       └── error.go              # type ErrorResponse struct {}
│
└── linebot/
    ├── webhook_handler.go        # LINE Webhook 入口
    ├── message_handler.go        # 訊息處理邏輯
    └── event_router.go           # 事件路由分發
```

---

## 3. 文件命名規範

### 3.1 Go 文件命名

**原則**:
- ✅ 小寫 + 下劃線：`member_repository.go`, `points_account.go`
- ✅ 單數形式：`member.go` 而非 `members.go`（除非是集合工具）
- ✅ 描述內容而非技術：`calculation_service.go` 而非 `service.go`
- ❌ 避免通用名稱：`handler.go`, `service.go`, `util.go`

**特殊文件**:
- `*_test.go` - 單元測試（與源文件同目錄）
- `*_mock.go` - Mock 實現（在 `test/mocks/` 目錄）
- `doc.go` - 包文檔（可選，用於 godoc）

### 3.2 測試文件命名

**規則**:
```
source_file.go       → source_file_test.go        # 單元測試
member_repository.go → member_repository_test.go
```

**測試類型標記** (使用 Build Tags):
```go
//go:build integration
// +build integration

package points_test  // 集成測試

//go:build e2e
// +build e2e

package e2e_test  // 端到端測試
```

---

## 4. 包命名規範

### 4.1 包名原則

**規則**:
- ✅ 單數形式：`package member` 而非 `package members`
- ✅ 小寫無下劃線：`package linebot` 而非 `package line_bot`
- ✅ 描述性：`package calculation` 而非 `package calc`
- ❌ 避免泛型名稱：`common`, `util`, `helper`

**示例**:
```go
// ✅ 正確
package points         // internal/domain/points/
package member         // internal/domain/member/
package repository     // internal/domain/points/repository/

// ❌ 錯誤
package util           // 過於泛型
package helpers        // 不明確
package pt             // 過於簡寫
```

### 4.2 Import Path

**結構**:
```go
import (
    // 標準庫
    "context"
    "errors"
    "time"

    // 第三方庫
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    // 內部包 - Domain Layer
    "github.com/yourorg/bar_crm/internal/domain/points"
    "github.com/yourorg/bar_crm/internal/domain/points/repository"

    // 內部包 - Application Layer
    "github.com/yourorg/bar_crm/internal/application/usecases/points"
    "github.com/yourorg/bar_crm/internal/application/dto"

    // 內部包 - Infrastructure Layer
    "github.com/yourorg/bar_crm/internal/infrastructure/persistence/gorm"
)
```

### 4.3 包可見性

**規則**:
- ✅ 公開（Exported）：大寫開頭 - `type Member struct {}`
- ✅ 私有（Unexported）：小寫開頭 - `type memberModel struct {}`

**最佳實踐**:
```go
// Domain Layer - 聚合根公開
type PointsAccount struct {
    accountID    AccountID  // 私有字段
    earnedPoints PointsAmount
}

// 公開方法（業務行為）
func (a *PointsAccount) EarnPoints(...) error {}

// 公開查詢方法
func (a *PointsAccount) GetEarnedPoints() PointsAmount {}

// Infrastructure Layer - GORM 模型私有
type pointsAccountModel struct {  // 小寫開頭，包內可見
    gorm.Model
    AccountID string
    // ...
}

// 公開的構造函數
func NewGormPointsAccountRepository(db *gorm.DB) repository.PointsAccountRepository {
    return &gormPointsAccountRepository{db: db}
}
```

---

## 5. 跨層依賴管理

### 5.1 允許的 Import 路徑

**Domain Layer** (`internal/domain/`):
```go
// ✅ 允許
import "github.com/yourorg/bar_crm/internal/domain/member"   // 同層依賴
import "github.com/yourorg/bar_crm/internal/domain/shared"   // 共享層
import "time"                                                // 標準庫
import "github.com/shopspring/decimal"                       // 數學庫（純計算）

// ❌ 禁止
import "github.com/yourorg/bar_crm/internal/application"     // 外層依賴
import "github.com/yourorg/bar_crm/internal/infrastructure"  // 外層依賴
import "gorm.io/gorm"                                        // 技術框架
```

**Application Layer** (`internal/application/`):
```go
// ✅ 允許
import "github.com/yourorg/bar_crm/internal/domain/points"   // 內層依賴
import "github.com/yourorg/bar_crm/internal/domain/points/repository"
import "github.com/yourorg/bar_crm/internal/application/dto" // 同層依賴

// ❌ 禁止
import "github.com/yourorg/bar_crm/internal/infrastructure"  // 外層依賴
import "gorm.io/gorm"                                        // 技術框架（除注入點）
```

**Infrastructure Layer** (`internal/infrastructure/`):
```go
// ✅ 允許
import "github.com/yourorg/bar_crm/internal/domain/points"   // 實現接口
import "github.com/yourorg/bar_crm/internal/domain/points/repository"
import "github.com/yourorg/bar_crm/internal/domain/shared"   // Event Bus 介面定義
import "gorm.io/gorm"                                        // 技術框架
import "github.com/redis/go-redis/v9"                        // 技術框架

// ❌ 禁止
import "github.com/yourorg/bar_crm/internal/application"     // 外層依賴（違反依賴規則）
import "github.com/yourorg/bar_crm/internal/presentation"    // 外層依賴

// 注意：Event Bus 實現在 Infrastructure Layer，但只依賴 shared.EventPublisher 介面
// Event Handlers 在 Application Layer，透過 DI 註冊到 Event Bus
```

**Presentation Layer** (`internal/presentation/`):
```go
// ✅ 允許
import "github.com/yourorg/bar_crm/internal/application/usecases"
import "github.com/yourorg/bar_crm/internal/domain/points"   // 僅用於 DTO 轉換
import "github.com/gin-gonic/gin"                            // Web 框架

// ❌ 禁止
import "github.com/yourorg/bar_crm/internal/infrastructure"  // 跨層依賴
```

### 5.2 避免循環依賴

**常見問題與解決方案**:

**問題 1: Domain ↔ Application 循環依賴**
```go
// ❌ 錯誤：Domain 依賴 Application 的 DTO
package points  // Domain Layer

import "myapp/internal/application/dto"  // 依賴外層

func (a *PointsAccount) RecalculatePoints(dtos []dto.TransactionDTO) error {}
```

**解決方案**：Domain 定義接口，Application 的 DTO 實現接口
```go
// ✅ 正確：Domain 定義接口
package points  // Domain Layer

type TransactionData interface {
    GetAmount() decimal.Decimal
    GetInvoiceDate() time.Time
    IsSurveySubmitted() bool
}

func (a *PointsAccount) RecalculatePoints(txs []TransactionData) error {}

// Application Layer 的 DTO 實現接口
package dto

func (d TransactionDTO) GetAmount() decimal.Decimal { return d.Amount }
func (d TransactionDTO) GetInvoiceDate() time.Time { return d.InvoiceDate }
func (d TransactionDTO) IsSurveySubmitted() bool { return d.SurveySubmitted }
```

**問題 2: Repository ↔ Model 循環依賴**
```go
// ❌ 錯誤：Domain Repository 接口依賴 Infrastructure 的 Model
package repository  // Domain Layer

import "myapp/internal/infrastructure/gorm"  // 依賴外層

type PointsAccountRepository interface {
    Save(model *gorm.PointsAccountModel) error  // 洩漏 GORM 模型
}
```

**解決方案**：Repository 接口只依賴 Domain 實體
```go
// ✅ 正確：Repository 接口只依賴 Domain 實體
package repository  // Domain Layer

import "myapp/internal/domain/points"

type PointsAccountRepository interface {
    Save(account *points.PointsAccount) error  // 使用 Domain 實體
}

// Infrastructure Layer 負責轉換
package gorm

func (r *GormPointsAccountRepository) Save(account *points.PointsAccount) error {
    model := r.toModel(account)  // Domain → GORM Model
    return r.db.Save(model).Error
}
```

---

## 6. 特殊目錄說明

### 6.1 `cmd/` 目錄

**用途**: 應用程序入口點

**規則**:
- ✅ 每個可執行程序一個子目錄
- ✅ main 函數應該簡短（< 100 行）
- ✅ 依賴注入配置在 main 函數中
- ❌ 避免業務邏輯

**示例**:
```go
// cmd/app/main.go
package main

import (
    "go.uber.org/fx"
    "github.com/yourorg/bar_crm/internal/infrastructure/config"
    "github.com/yourorg/bar_crm/internal/presentation/http"
)

func main() {
    fx.New(
        config.Module,
        // ... 其他模組
        http.Module,
    ).Run()
}
```

### 6.2 `internal/` 目錄

**用途**: Go 語言級別的可見性控制

**規則**:
- ✅ `internal/` 下的包不能被外部 import
- ✅ 所有業務代碼應放在 `internal/`
- ✅ 防止外部依賴內部實現

**為什麼使用 internal/？**
```go
// ❌ 錯誤：沒有 internal/，外部項目可以 import
import "github.com/yourorg/bar_crm/domain/points"

// ✅ 正確：有 internal/，外部 import 會報錯
import "github.com/yourorg/bar_crm/internal/domain/points"
// → Error: use of internal package not allowed
```

### 6.3 `shared/` vs `common/`

**推薦使用 `shared/`**:
- ✅ `internal/domain/shared/` - 跨上下文的領域概念
  - TransactionContext 接口
  - DomainEvent 基礎接口
  - 通用錯誤

**避免使用 `common/` 或 `util/`**:
- ❌ `common/` - 容易變成垃圾堆（God Package）
- ❌ `util/` - 不符合領域模型思維

**正確使用**:
```go
// ✅ 正確：shared/ 只放跨上下文的領域概念
package shared

type DomainEvent interface {
    EventID() string
    OccurredAt() time.Time
    EventType() string
}

type TransactionContext interface {
    // 標記接口
}
```

### 6.4 測試目錄

**單元測試** - 與源文件同目錄:
```
internal/domain/points/
├── account.go
└── account_test.go  # 單元測試
```

**集成測試** - 獨立的 test/ 目錄:
```
test/
├── integration/
│   ├── points_test.go       # 積分集成測試
│   └── member_test.go       # 會員集成測試
│
└── e2e/
    └── scan_and_earn_test.go # 端到端測試
```

---

## 7. 文件大小建議

**推薦大小**:
- ✅ 聚合根文件：200-500 行
- ✅ Use Case 文件：50-150 行
- ✅ Repository 實現：100-300 行
- ✅ Handler 文件：100-200 行

**何時拆分文件？**
- ⚠️ > 500 行：考慮拆分
- ⚠️ > 1000 行：必須拆分

**拆分策略**:
```go
// 原始文件過大
account.go  // 800 lines

// 拆分後
account.go           // 聚合根結構與核心方法（300 lines）
account_commands.go  // 命令方法（200 lines）
account_queries.go   // 查詢方法（100 lines）
account_events.go    // 事件處理（100 lines）
```

---

## 8. 檢查清單

### 代碼審查時的目錄結構檢查

**Domain Layer**:
- [ ] 所有聚合根都有獨立文件
- [ ] Repository 接口定義在 `repository/` 子目錄
- [ ] 無 import `infrastructure`, `application`, `presentation`
- [ ] 無 import `gorm`, `gin`, `redis` 等技術框架

**Application Layer**:
- [ ] Use Cases 按上下文劃分
- [ ] 每個 Use Case 一個文件
- [ ] DTO 集中在 `dto/` 目錄
- [ ] Event Handlers 按職責劃分

**Infrastructure Layer**:
- [ ] Repository 實現按上下文劃分
- [ ] GORM 模型集中定義在 `gorm/models.go`
- [ ] 外部服務適配器隔離在獨立子目錄
- [ ] 無 import `presentation` 層（除 Event Handlers）

**Presentation Layer**:
- [ ] Handlers 按上下文劃分
- [ ] 中間件集中在 `middleware/` 目錄
- [ ] 響應結構統一在 `responses/` 目錄
- [ ] 無 import `infrastructure` 層

---

## 9. 常見錯誤

### ❌ 錯誤 1: 過度扁平化
```
internal/
├── member.go
├── points.go
├── invoice.go
├── member_repository.go
├── points_repository.go
└── ...  # 所有文件混在一起
```

**問題**: 無法區分層次，依賴關係混亂

### ❌ 錯誤 2: 過度嵌套
```
internal/
└── domain/
    └── contexts/
        └── points/
            └── aggregates/
                └── account/
                    └── value_objects/
                        └── points_amount.go
```

**問題**: 過度嵌套導致 import 路徑過長

### ❌ 錯誤 3: 技術驅動的目錄結構
```
internal/
├── models/       # GORM 模型
├── services/     # 所有業務邏輯
├── repositories/ # 所有 Repositories
└── controllers/  # 所有 Controllers
```

**問題**: 按技術分層而非業務領域分層，違反 DDD 原則

---

## 10. 總結

### 目錄結構原則

1. **領域驅動**：按業務領域（限界上下文）劃分，而非技術層
2. **依賴規則**：外層依賴內層（Infrastructure → Application → Domain）
3. **清晰邊界**：使用 `internal/` 控制可見性
4. **測試友好**：清晰的包邊界，易於 Mock
5. **Go 慣例**：遵循 Go 語言的最佳實踐

### 快速參考

| 層級 | 目錄 | 職責 | 依賴 |
|------|------|------|------|
| **Domain** | `internal/domain/` | 業務邏輯與規則 | 標準庫 + 同層 |
| **Application** | `internal/application/` | 用例協調 | Domain |
| **Infrastructure** | `internal/infrastructure/` | 技術實現 | Domain + 外部框架 |
| **Presentation** | `internal/presentation/` | 協議適配 | Application + Web 框架 |

---

**下一步**: 閱讀 [02-Domain Layer 實現指南](./02-domain-layer-implementation.md) 了解如何實現領域層代碼
