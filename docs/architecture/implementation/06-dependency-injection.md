# 依賴注入配置

> **版本**: 1.0
> **最後更新**: 2025-01-10

## 1. Uber FX 模組設計

### 1.1 主應用入口

**文件**: `cmd/app/main.go`

```go
package main

import (
    "go.uber.org/fx"
    "github.com/jackyeh168/bar_crm/internal/infrastructure/config"
    "github.com/jackyeh168/bar_crm/internal/infrastructure/persistence"
    "github.com/jackyeh168/bar_crm/internal/domain/points"
    "github.com/jackyeh168/bar_crm/internal/application/usecases"
    "github.com/jackyeh168/bar_crm/internal/presentation/http"
)

func main() {
    fx.New(
        // 1. 基礎設施模組（無依賴）
        config.Module,
        
        // 2. 數據庫模組
        persistence.DatabaseModule,
        
        // 3. Repository 模組
        persistence.RepositoryModule,
        
        // 4. Domain Service 模組
        points.DomainServiceModule,
        
        // 5. Use Case 模組
        usecases.UseCaseModule,
        
        // 6. Handler 模組
        http.HandlerModule,
        
        // 7. Server 模組
        http.ServerModule,
    ).Run()
}
```

### 1.2 模組定義範例

**DatabaseModule**:

```go
// internal/infrastructure/persistence/module.go
package persistence

import (
    "go.uber.org/fx"
    "gorm.io/gorm"
)

var DatabaseModule = fx.Module("database",
    fx.Provide(NewGormDB),           // 提供 *gorm.DB
    fx.Provide(NewTransactionManager), // 提供 TransactionManager
)

func NewGormDB(cfg *config.Config) (*gorm.DB, error) {
    return gorm.Open(/* config */)
}
```

**RepositoryModule**:

```go
var RepositoryModule = fx.Module("repositories",
    fx.Provide(
        fx.Annotate(
            NewGormPointsAccountRepository,
            fx.As(new(repository.PointsAccountRepository)),
        ),
    ),
)
```

## 2. 依賴注入順序

```
01. LoggerModule (無依賴)
02. ConfigModule (依賴 Logger)
03. DatabaseModule (依賴 Config, Logger)
04. RepositoryModule (依賴 Database)
05. RedisModule (依賴 Config, Logger)
6-14. Business Logic Modules
15. HandlerModule (依賴所有 UseCases)
16. ServerModule (依賴 Handlers, Config)
```

## 3. 測試時的依賴替換

**測試時使用 Mock**:

```go
func setupTestApp(t *testing.T) *fx.App {
    return fx.New(
        // 使用 SQLite in-memory 替代 PostgreSQL
        fx.Supply(&config.Config{DatabaseURL: ":memory:"}),
        
        // 使用 Mock Repositories
        fx.Provide(func() repository.PointsAccountRepository {
            return mocks.NewMockPointsAccountRepository()
        }),
        
        // 其他正常模組
        usecases.UseCaseModule,
        http.HandlerModule,
    )
}
```

**下一步**: 閱讀 [08-完整代碼範例](./08-code-examples.md)
