# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ⚠️ 開始實作前必讀

**CRITICAL**: 在開始任何代碼實作前，請先閱讀：
- **實作檢查清單**: `docs/implementation-checklist.md` - 必須在每個任務開始前確認
- **架構約束速查**: 見下方"核心架構約束"章節

---

## 🏗️ 核心架構約束（Implementation Constraints）

### ❌ 絕對禁止事項

| 層次 | 禁止行為 | 正確做法 | 文檔參考 |
|------|---------|---------|----------|
| **Domain Layer** | 使用 `fmt.Errorf()` 或 `errors.New()` | 必須使用 `DomainError` 結構 | `13-error-handling-strategy.md` |
| **Domain Layer** | Import 外部框架 (`gorm`, `gin`, `redis`) | 只依賴標準庫和 domain 接口 | `12-dependency-rules.md` |
| **Value Object** | 提供 Setter 方法或 exported fields | 必須不可變（unexported fields） | `10-value-object-validation.md` |
| **所有層次** | 跳過測試或低於目標覆蓋率 | 遵循 TDD，覆蓋率 >= 85% | `qa/testing-conventions.md` |

### ✅ 必須遵守規則

#### 1. 錯誤處理架構

**錯誤的做法**：
```go
// ❌ 禁止：使用 fmt.Errorf
var ErrNegativePointsAmount = fmt.Errorf("points amount cannot be negative")
```

**正確的做法**：
```go
// ✅ 必須：使用 DomainError 結構
const ErrCodeNegativePointsAmount ErrorCode = "POINTS_NEGATIVE"

var ErrNegativePointsAmount = &DomainError{
    Code:    ErrCodeNegativePointsAmount,
    Message: "積分數量不能為負數",
}
```

**參考**：`docs/architecture/ddd/13-error-handling-strategy.md`

---

#### 2. 值對象設計

**必須特性**：
- ✅ 不可變性（unexported fields，無 setters）
- ✅ 自我驗證（建構函數檢查約束）
- ✅ Checked vs Unchecked 建構函數模式

**示例**：
```go
type PointsAmount struct {
    value int  // ✅ unexported
}

func NewPointsAmount(value int) (PointsAmount, error) {
    if value < 0 {
        return PointsAmount{}, ErrNegativePointsAmount  // ✅ 使用 DomainError
    }
    return PointsAmount{value: value}, nil
}
```

**參考**：`docs/architecture/ddd/10-value-object-validation.md`

---

#### 3. 依賴規則

**依賴方向** (只能向內依賴)：
```
Presentation → Application → Domain
Infrastructure → Domain (實現接口)
```

**Domain Layer 禁止依賴**：
- ❌ Application Layer
- ❌ Infrastructure Layer
- ❌ Presentation Layer
- ❌ 外部框架 (GORM, Gin, Redis, LINE SDK)

**參考**：`docs/architecture/ddd/12-dependency-rules.md`

---

#### 4. 測試規範

**AAA 模式**（強制）：
```go
func TestPointsAmount_Subtract_ExceedsValue_ReturnsError(t *testing.T) {
    // Arrange - 準備測試數據
    amount1, _ := points.NewPointsAmount(50)
    amount2, _ := points.NewPointsAmount(100)

    // Act - 執行被測試的操作
    result, err := amount1.Subtract(amount2)

    // Assert - 驗證結果
    assert.Error(t, err)
    assert.ErrorIs(t, err, points.ErrNegativePointsAmount)
}
```

**覆蓋率要求**：
- Unit Tests: >= 85%
- Integration Tests: 20%
- E2E Tests: 3%

**參考**：`docs/qa/testing-conventions.md`

---

### 📚 實作前必讀文檔矩陣

| 實作內容 | 必讀文檔 |
|---------|---------|
| **值對象** | `10-value-object-validation.md`, `13-error-handling-strategy.md` |
| **聚合根** | `05-tactical-design.md`, `08-aggregate-design-patterns.md` |
| **領域服務** | `05-tactical-design.md`, `09-domain-services.md` |
| **Repository** | `06-repository-pattern.md`, `12-dependency-rules.md` |
| **Use Case** | `10-use-case-definitions.md` |
| **錯誤處理** | `13-error-handling-strategy.md` (所有實作必讀) |

---

## Project Status

**IMPORTANT**: This is currently a **documentation-only project** in the planning phase. No source code has been implemented yet. All content resides in the `/docs` directory with comprehensive architectural specifications.

## Project Overview

**Name**: Restaurant Member Management LINE Bot (餐廳會員管理 Line Bot)
**Version**: 3.1 (Clean Architecture Compliant)
**Language**: Go 1.21+ (planned)
**Architecture**: Clean Architecture + Domain-Driven Design (DDD)

### Core Features (Planned)
- LINE Bot integration for member registration and points management
- QR Code invoice scanning with automatic points calculation
- iChef POS system integration for invoice verification
- Survey system with points rewards
- Admin portal with Google OAuth2 authentication
- Dynamic points conversion rules

## Documentation Structure

All documentation is in the `/docs` directory:

- **docs/README.md** - Documentation navigation hub
- **docs/product/PRD.md** - Complete product requirements (465 lines)
- **docs/architecture/ddd/** - Domain-Driven Design guide (split into 13 chapters)
  - See `docs/architecture/ddd/README.md` for chapter navigation
- **docs/operations/DEPLOYMENT.md** - Deployment and DevOps guide
- **docs/qa/** - Testing strategies and conventions

### Reading Order for New Team Members
1. docs/README.md - Overview
2. docs/product/PRD.md - Understand business requirements
3. docs/architecture/ddd/README.md - DDD architecture overview (then read specific chapters as needed)
4. docs/operations/DEPLOYMENT.md - Deployment knowledge

### Key DDD Chapters to Read First
- **01-design-overview.md** - Design goals and domain classification
- **02-strategic-design.md** - Domain events, commands, and business rules
- **03-bounded-contexts.md** - All 7 bounded contexts (comprehensive)
- **05-tactical-design.md** - Core domain tactical patterns

## Planned Architecture

### Clean Architecture Layers
```
├── Domain Layer (Core business logic, no dependencies)
│   ├── Membership Context (Core Domain)
│   ├── Invoice Context (Supporting Domain)
│   └── Survey Context (Supporting Domain)
├── Application Layer (Use Cases)
├── Interface Adapters Layer
│   ├── HTTP Handlers
│   ├── LINE Bot Webhook Handlers
│   └── Repository Implementations (GORM)
└── Infrastructure Layer
    ├── Database (PostgreSQL + GORM)
    ├── External Services (LINE SDK, iChef)
    └── Caching (Redis with in-memory fallback)
```

### Planned Directory Structure
```
bar_crm/
├── cmd/
│   ├── app/              # Main application entry point
│   └── migrate/          # Database migration tools
├── internal/
│   ├── domain/           # Domain layer (DDD entities, value objects)
│   │   ├── membership/   # Core: Points & member management
│   │   ├── invoice/      # Supporting: Invoice verification
│   │   └── survey/       # Supporting: Survey system
│   ├── application/      # Use cases (business workflows)
│   ├── infrastructure/
│   │   ├── persistence/  # GORM repositories
│   │   └── external/     # LINE adapter (Anti-Corruption Layer)
│   └── interfaces/       # HTTP/LINE Bot handlers
├── test/
│   ├── integration/      # Database integration tests
│   ├── contract/         # External service contract tests
│   └── e2e/             # End-to-end tests
└── docs/                # Current comprehensive documentation
```

## Technology Stack (Planned)

### Backend
- **Language**: Go 1.21+
- **Web Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL (SQLite for testing)
- **Cache**: Redis (optional, with in-memory fallback)
- **Dependency Injection**: Uber FX
- **Logging**: Zap (structured logging)

### External Integrations
- **LINE Bot SDK**: Official LINE Messaging API
- **Google OAuth2**: Admin portal authentication
- **iChef**: POS system (Excel import integration)

### Frontend (Admin Portal)
- **Framework**: React/Vite
- **API Base URL**: Configurable via `VITE_API_URL`

## Build and Run Commands (When Implemented)

### Local Development
```bash
# Set required environment variables
export CHANNEL_SECRET=your_line_channel_secret
export CHANNEL_TOKEN=your_line_channel_token
export PORT=8080

# Run application
go run cmd/app/main.go

# Or using Make
make dev
```

### Docker Compose
```bash
# Start all services (App + PostgreSQL + Redis)
make start

# View logs
make logs

# Stop services
make down
```

### Health Check
```bash
curl http://localhost:8080/health
# Returns: database, redis, linebot status
```

## Testing Strategy

### Test Organization
```
test/
├── integration/          # Database integration tests
├── contract/            # LINE API contract tests
└── e2e/                 # End-to-end scenarios
internal/
└── *_test.go           # Unit tests alongside source files
```

### Running Tests
```bash
# Run all tests
go test ./...

# Run with coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run with race detection
go test ./... -race

# Run specific layer tests
go test ./internal/handler -v
go test ./internal/repository -v
go test ./test/integration/... -v

# Run single test
go test ./internal/service -run TestRegistrationService_ValidatePhoneNumber -v
```

### Test Naming Convention
```go
// Standard unit tests
func Test{ServiceName}_{MethodName}_{Scenario}(t *testing.T)

// Examples:
// - TestRegistrationService_ValidatePhoneNumber_ValidInput
// - TestLineBotService_HandleRequest_InvalidSignature

// TestSuite methods
func (suite *{ServiceName}TestSuite) Test{MethodName}_{Scenario}()
```

### Test Coverage Targets
- **Unit Tests**: 70%+ coverage (fast, isolated, reliable)
- **Integration Tests**: 20% (cross-component interactions)
- **Contract Tests**: 7% (external service contracts)
- **E2E Tests**: 3% (complete user flows)

### Testing Patterns
- **AAA Pattern**: Arrange-Act-Assert structure
- **Mock Strategy**: Unified TestHelper framework with 3 strategies:
  - Interface Strategy (simple, pre-defined behavior)
  - Testify Strategy (full mock verification)
  - Hybrid Strategy (combination of both)

See `docs/qa/testing-conventions.md` for complete testing standards.

## Database Management

### Auto-Migration (Development/Testing)
```bash
# Enable auto-migration
export AUTO_MIGRATE=true
go run cmd/app/main.go

# Or using Make
make migrate
```

### Manual Migration (Production)
```bash
# 1. Backup database first
./scripts/backup-db.sh

# 2. Run migration
make migrate

# 3. Verify tables
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "\dt"
```

### Points Recalculation
```bash
# After changing points conversion rules
make recalculate-points
```

## Configuration Management

### Required Environment Variables
```bash
CHANNEL_SECRET      # LINE Bot channel secret
CHANNEL_TOKEN       # LINE Bot access token
PORT               # HTTP service port (default: 8080)
GIN_MODE           # Environment: release/debug/test
```

### Optional Configuration
```bash
# Database (PostgreSQL) - Falls back to mock repository if not set
DATABASE_URL       # PostgreSQL connection string

# Redis - Falls back to in-memory storage
REDIS_ADDR         # Redis server address
REDIS_PASSWORD     # Redis password

# Admin OAuth - Disables admin features if not configured
GOOGLE_CLIENT_ID   # Google OAuth2 client ID
GOOGLE_CLIENT_SECRET  # Google OAuth2 client secret
DEFAULT_ADMIN_EMAIL   # First user with this email gets Admin role
```

### FX Module Loading Order
The system uses Uber FX for dependency injection with strict module ordering:

```
1. LoggerModule (no dependencies)
2. ConfigModule (depends on Logger)
3. DatabaseModule (depends on Config, Logger)
4. RepositoryModule (depends on Database)
5. RedisModule (depends on Config, Logger)
6-14. Business Logic Modules (Registration, Points, LineBot, etc.)
15. HandlerModule (depends on all services)
16. ServerModule (depends on Handler, Config)
```

See `docs/operations/DEPLOYMENT.md` section 1.1 for complete details.

## Domain-Driven Design Patterns

### Core Domain: Membership Points System
- **Aggregates**: `MembershipAccount`, `PointsConversionRule`
- **Value Objects**: `PhoneNumber`, `Money`
- **Domain Services**: `PointsCalculationService`
- **Key Invariants**:
  - `EarnedPoints >= 0`
  - `UsedPoints >= 0`
  - `UsedPoints <= EarnedPoints`

### Supporting Domain: Invoice Verification
- **Aggregates**: `Invoice`, `IChefImportRecord`
- **Domain Services**: `InvoiceMatchingService`
- **Key Invariants**:
  - Unique invoice number (no duplicates)
  - Valid status transitions only
  - 60-day validity period enforcement

### Supporting Domain: Survey System
- **Aggregates**: `Survey` (with nested `SurveyQuestion`)
- **Key Invariants**:
  - Survey must have at least one question
  - Only one active survey at a time
  - One response per transaction

### Design Principles
1. **Tell, Don't Ask**: Objects should encapsulate behavior
2. **Aggregate Boundaries = Transaction Boundaries**: One transaction modifies one aggregate
3. **Value Object Immutability**: Return new instances, never mutate
4. **Repository Interface Segregation**: Separate read/write interfaces

## Anti-Corruption Layer

The system uses ACL pattern to isolate from external services:

```go
// LineUserAdapter wraps LINE SDK to prevent domain pollution
type LineUserAdapter struct {
    linebotClient *linebot.Client
}

// Converts LINE models to internal domain models
func (a *LineUserAdapter) GetUserProfile(lineUserID string) (*domain.MembershipAccount, error)
```

This prevents LINE Platform's data structures from leaking into the domain layer.

## Business Rules Summary

### Points Calculation
```
Base Points = Amount ÷ Conversion Rate (floor division)
Survey Bonus = +1 point (when survey completed)
Total Points = Sum of all verified transactions

Default Conversion Rate: 100 TWD = 1 point
Admin can configure promotional rates (e.g., 50 TWD = 1 point)
```

### Invoice Validation
- **Validity Period**: 60 days from invoice date
- **Duplicate Detection**: Same invoice number cannot be registered twice
- **Status Flow**: `imported → verified / failed`
- **Verification**: Requires match on invoice number + date + amount

### Points Status
- **Pending**: Transaction awaiting iChef verification (not counted in balance)
- **Verified**: Transaction confirmed (counted in balance)
- **Failed**: Invalid transaction (not counted)

## External Integration Points

### LINE Platform
- **Type**: Conformist (must follow LINE's model)
- **Pattern**: Anti-Corruption Layer
- **Integration**: Webhook for events, API for profile/messages

### iChef POS System
- **Method**: Excel file batch import
- **Matching**: Invoice number + date + amount
- **Frequency**: Batch (weekly/monthly)

### Google OAuth2
- **Purpose**: Admin portal authentication only
- **Roles**: Admin / User / Guest
- **First Admin**: Configured via `DEFAULT_ADMIN_EMAIL`

## Common Development Workflows

### Adding a New Use Case
1. Define domain entities/value objects in `internal/domain/`
2. Create use case interface in `internal/application/`
3. Implement repository interface in `internal/infrastructure/persistence/`
4. Add HTTP handler in `internal/interfaces/`
5. Register in FX module
6. Write tests (unit → integration → e2e)

### Changing Points Calculation Rules
1. Update domain logic in `PointsCalculationService`
2. Add migration if database schema changes
3. Run `make recalculate-points` to recompute all points
4. Update tests to reflect new rules

### Adding New Survey Questions
1. Update `Survey` aggregate in domain layer
2. Modify survey handler to support new question types
3. Update frontend survey component
4. Add validation tests

## Important Constraints

### Phone Number Validation
- Format: 10 digits starting with "09" (Taiwan mobile)
- One LINE account = One phone number (enforced at database level)
- Cannot self-unbind (requires admin intervention)

### Transaction Constraints
- Cannot delete verified transactions (immutability)
- Status transitions are one-way (cannot revert from failed to verified)
- Points recalculation triggered on status change

### Survey Constraints
- Only one active survey at a time
- One response per transaction maximum
- Token-based access (no login required for filling)

## Development Best Practices

1. **Keep domain layer pure**: No external dependencies (no GORM tags, no HTTP)
2. **Use value objects for validation**: Phone numbers, email, money
3. **Domain events for cross-aggregate operations**: Invoice verified → recalculate points
4. **Repository pattern**: Never expose ORM models outside infrastructure layer
5. **Mock external services in tests**: Never call real LINE API or iChef
6. **Follow AAA pattern in tests**: Arrange-Act-Assert structure

## Related Documentation

For implementation details, always refer to:
- **DDD Architecture**: `docs/architecture/ddd/README.md` (see individual chapters for specific topics)
  - Strategic Design: `docs/architecture/ddd/02-strategic-design.md`
  - Bounded Contexts: `docs/architecture/ddd/03-bounded-contexts.md`
  - Use Cases: `docs/architecture/ddd/10-use-case-definitions.md`
  - Dependency Rules: `docs/architecture/ddd/12-dependency-rules.md`
- **Product Requirements**: `docs/product/PRD.md`
- **Testing Standards**: `docs/qa/testing-conventions.md`
- **Deployment**: `docs/operations/DEPLOYMENT.md`
- **User Stories**: `docs/product/stories/US-00*.md`
