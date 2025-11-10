# 包命名規範

> **版本**: 1.0
> **最後更新**: 2025-01-10

## 1. Go 包命名原則

### 1.1 基本規則

- ✅ 小寫字母，無下劃線：`package member`
- ✅ 單數形式：`package user` 而非 `package users`
- ✅ 簡短描述性：`package http` 而非 `package httpserver`
- ❌ 避免泛型名稱：`common`, `util`, `helper`

### 1.2 命名範例

**✅ 正確**:
```go
package points      // 積分管理
package member      // 會員管理
package repository  // 倉儲接口
package linebot     // LINE Bot 適配器
```

**❌ 錯誤**:
```go
package utils       // 過於泛型
package pt          // 過於簡寫
package members     // 複數形式
package line_bot    // 使用下劃線
```

## 2. Import Path 組織

```go
import (
    // 1. 標準庫（按字母排序）
    "context"
    "errors"
    "time"

    // 2. 第三方庫（按字母排序）
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    // 3. 內部包 - Domain Layer
    "github.com/yourorg/bar_crm/internal/domain/points"
    "github.com/yourorg/bar_crm/internal/domain/points/repository"

    // 4. 內部包 - Application Layer
    "github.com/yourorg/bar_crm/internal/application/usecases/points"

    // 5. 內部包 - Infrastructure Layer
    gormPkg "github.com/yourorg/bar_crm/internal/infrastructure/persistence/gorm"
)
```

## 3. 避免循環依賴

### 3.1 常見問題

**問題**: Domain ↔ Application 循環依賴

**解決方案**: Domain 定義接口，Application 的 DTO 實現接口

```go
// Domain Layer
type TransactionData interface {
    GetAmount() decimal.Decimal
    GetInvoiceDate() time.Time
}

// Application Layer
type TransactionDTO struct {
    Amount      decimal.Decimal
    InvoiceDate time.Time
}

func (d TransactionDTO) GetAmount() decimal.Decimal { 
    return d.Amount 
}
```

## 4. 包可見性控制

**公開 (Exported)**: 大寫開頭
```go
type Member struct {}       // ✅ 公開
func NewMember() *Member {} // ✅ 公開
```

**私有 (Unexported)**: 小寫開頭
```go
type memberModel struct {}  // ✅ 私有
func toModel() *memberModel {} // ✅ 私有
```

**下一步**: 閱讀 [08-完整代碼範例](./08-code-examples.md)
