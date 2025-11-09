# 錯誤處理架構 (Error Handling Architecture)

> **版本**: 1.0
> **最後更新**: 2025-01-09
> **狀態**: Production Ready

---

## **目錄**

1. [錯誤處理原則](#1-錯誤處理原則)
2. [錯誤類型層次](#2-錯誤類型層次)
3. [錯誤傳播規則](#3-錯誤傳播規則)
4. [HTTP 狀態碼映射](#4-http-狀態碼映射)
5. [日誌記錄策略](#5-日誌記錄策略)
6. [錯誤處理範例](#6-錯誤處理範例)
7. [常見錯誤處理反模式](#7-常見錯誤處理反模式)

---

## **1. 錯誤處理原則**

### **1.1 Clean Architecture 的錯誤處理規則**

```
┌────────────────────────────────────────────────────────┐
│ Presentation Layer (HTTP Handlers)                     │
│ - 將 Application Error 轉換為 HTTP Status Code        │
│ - 記錄請求失敗的上下文（IP、User Agent）               │
│ - 返回用戶友好的錯誤訊息                                │
└────────────────────┬───────────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────────┐
│ Application Layer (Use Cases)                          │
│ - 包裝 Domain Error 為 Application Error               │
│ - 記錄業務操作失敗的詳細上下文                          │
│ - 不創建新的錯誤類型（除了 workflow 錯誤）             │
└────────────────────┬───────────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────────┐
│ Domain Layer (Aggregates, Services)                    │
│ - 定義所有業務規則錯誤（Domain Errors）                │
│ - 返回語義明確的錯誤（ErrInsufficientPoints）          │
│ - 不記錄日誌（無 logger 依賴）                          │
└────────────────────┬───────────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────────┐
│ Infrastructure Layer (Repositories, Adapters)          │
│ - 將基礎設施錯誤轉換為 Domain Errors                    │
│ - 不洩漏基礎設施錯誤（gorm.ErrRecordNotFound）         │
│ - 僅記錄技術錯誤（資料庫連線失敗）                      │
└────────────────────────────────────────────────────────┘
```

### **1.2 核心原則**

| 原則 | 說明 | 範例 |
|------|------|------|
| **錯誤來源於 Domain** | 所有業務錯誤在 Domain Layer 定義 | `ErrInsufficientPoints`, `ErrInvalidPhoneNumber` |
| **不洩漏基礎設施** | Infrastructure 錯誤必須轉換 | ❌ `gorm.ErrRecordNotFound` → ✅ `ErrAccountNotFound` |
| **語義明確** | 錯誤名稱描述業務問題，非技術細節 | ✅ `ErrDuplicateInvoice` ❌ `ErrUniqueConstraintViolation` |
| **可檢查性** | 使用 `errors.Is()` 和 `errors.As()` | `if errors.Is(err, ErrInsufficientPoints)` |
| **上下文富集** | 錯誤攜帶診斷信息 | `ErrInsufficientPoints.WithContext("required", 100, "available", 50)` |

---

## **2. 錯誤類型層次**

### **2.1 Domain Errors（業務規則錯誤）**

**定義位置**: `internal/domain/{context}/errors.go`

**用途**: 表示業務規則違反（不變性保護、前置條件失敗）

```go
// Domain Layer - internal/domain/points/errors.go
package points

import "fmt"

// DomainError 業務規則錯誤的基礎結構
type DomainError struct {
    Code    ErrorCode
    Message string
    Context map[string]interface{}
}

func (e *DomainError) Error() string {
    return e.Message
}

func (e *DomainError) WithContext(keyValues ...interface{}) *DomainError {
    if e.Context == nil {
        e.Context = make(map[string]interface{})
    }

    for i := 0; i < len(keyValues); i += 2 {
        if i+1 < len(keyValues) {
            key := fmt.Sprintf("%v", keyValues[i])
            e.Context[key] = keyValues[i+1]
        }
    }

    return e
}

// ErrorCode 錯誤代碼（用於 HTTP 映射與多語言）
type ErrorCode string

const (
    ErrCodeInsufficientPoints   ErrorCode = "INSUFFICIENT_POINTS"
    ErrCodeInvalidAmount        ErrorCode = "INVALID_AMOUNT"
    ErrCodeAccountNotFound      ErrorCode = "ACCOUNT_NOT_FOUND"
    ErrCodeDuplicateTransaction ErrorCode = "DUPLICATE_TRANSACTION"
    ErrCodeInvalidPhoneNumber   ErrorCode = "INVALID_PHONE_NUMBER"
    ErrCodeInvoiceExpired       ErrorCode = "INVOICE_EXPIRED"
)

// 預定義錯誤（Sentinel Errors）
var (
    ErrInsufficientPoints = &DomainError{
        Code:    ErrCodeInsufficientPoints,
        Message: "可用積分不足以完成此操作",
    }

    ErrInvalidAmount = &DomainError{
        Code:    ErrCodeInvalidAmount,
        Message: "金額必須為正數",
    }

    ErrAccountNotFound = &DomainError{
        Code:    ErrCodeAccountNotFound,
        Message: "找不到積分帳戶",
    }

    ErrDuplicateTransaction = &DomainError{
        Code:    ErrCodeDuplicateTransaction,
        Message: "此交易已經存在",
    }
)
```

**使用範例**:

```go
// Domain Layer - PointsAccount Aggregate
func (a *PointsAccount) DeductPoints(
    amount PointsAmount,
    reason string,
    referenceID string,
) error {
    if amount.Value() <= 0 {
        return ErrInvalidAmount.WithContext("amount", amount.Value())
    }

    availablePoints := a.GetAvailablePoints()
    if availablePoints.LessThan(amount) {
        return ErrInsufficientPoints.WithContext(
            "required", amount.Value(),
            "available", availablePoints.Value(),
            "accountID", a.accountID.String(),
        )
    }

    a.usedPoints = a.usedPoints.Add(amount)
    a.RecordEvent(PointsDeducted{
        AccountID:   a.accountID,
        Amount:      amount,
        Reason:      reason,
        ReferenceID: referenceID,
    })

    return nil
}
```

---

### **2.2 Application Errors（工作流程錯誤）**

**定義位置**: `internal/application/errors.go`

**用途**: 表示 Use Case 執行失敗（跨聚合協調失敗、事務回滾）

```go
// Application Layer - internal/application/errors.go
package application

import (
    "fmt"
)

// ApplicationError 包裝 Domain Error 並添加應用層上下文
type ApplicationError struct {
    Operation   string
    DomainErr   error
    Context     map[string]interface{}
}

func (e *ApplicationError) Error() string {
    return fmt.Sprintf("operation '%s' failed: %v", e.Operation, e.DomainErr)
}

func (e *ApplicationError) Unwrap() error {
    return e.DomainErr
}

func NewApplicationError(operation string, domainErr error) *ApplicationError {
    return &ApplicationError{
        Operation: operation,
        DomainErr: domainErr,
        Context:   make(map[string]interface{}),
    }
}

func (e *ApplicationError) WithContext(keyValues ...interface{}) *ApplicationError {
    for i := 0; i < len(keyValues); i += 2 {
        if i+1 < len(keyValues) {
            key := fmt.Sprintf("%v", keyValues[i])
            e.Context[key] = keyValues[i+1]
        }
    }
    return e
}
```

**使用範例**:

```go
// Application Layer - Use Case
func (uc *DeductPointsUseCase) Execute(cmd DeductPointsCommand) error {
    return uc.txManager.InTransaction(func(ctx TransactionContext) error {
        account, err := uc.accountRepo.FindByID(ctx, cmd.AccountID)
        if err != nil {
            return NewApplicationError("DeductPoints", err).
                WithContext("accountID", cmd.AccountID)
        }

        err = account.DeductPoints(cmd.Amount, cmd.Reason, cmd.ReferenceID)
        if err != nil {
            return NewApplicationError("DeductPoints", err).
                WithContext(
                    "accountID", cmd.AccountID,
                    "amount", cmd.Amount.Value(),
                    "reason", cmd.Reason,
                )
        }

        err = uc.accountRepo.Update(ctx, account)
        if err != nil {
            return NewApplicationError("DeductPoints", err).
                WithContext("accountID", cmd.AccountID)
        }

        return nil
    })
}
```

---

### **2.3 Infrastructure Errors（基礎設施錯誤）**

**規則**: Infrastructure Layer **不創建錯誤**，只負責將基礎設施錯誤轉換為 Domain Errors。

```go
// Infrastructure Layer - internal/infrastructure/persistence/gorm_points_repository.go
package persistence

import (
    "errors"

    "internal/domain/points"
    "gorm.io/gorm"
)

type GormPointsAccountRepository struct {
    db *gorm.DB
}

func (r *GormPointsAccountRepository) FindByID(
    ctx TransactionContext,
    accountID points.AccountID,
) (*points.PointsAccount, error) {
    db := r.extractDB(ctx)

    var model PointsAccountModel
    err := db.Where("account_id = ?", accountID.String()).First(&model).Error

    if err != nil {
        // 將基礎設施錯誤轉換為 Domain Error
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, points.ErrAccountNotFound.WithContext(
                "accountID", accountID.String(),
            )
        }
        // 未預期的資料庫錯誤（保留原始錯誤以便調試）
        return nil, fmt.Errorf("database error while finding account %s: %w", accountID, err)
    }

    return toDomainModel(&model), nil
}

func (r *GormPointsAccountRepository) Update(
    ctx TransactionContext,
    account *points.PointsAccount,
) error {
    db := r.extractDB(ctx)
    model := toGormModel(account)

    err := db.Save(model).Error
    if err != nil {
        // 檢查是否是樂觀鎖錯誤
        if isOptimisticLockError(err) {
            return points.ErrOptimisticLockFailure.WithContext(
                "accountID", account.AccountID().String(),
            )
        }

        // 檢查是否是唯一性約束錯誤
        if isDuplicateKeyError(err) {
            return points.ErrDuplicateTransaction
        }

        return fmt.Errorf("database error while updating account: %w", err)
    }

    return nil
}

// 輔助函數：檢查 GORM 錯誤類型
func isOptimisticLockError(err error) bool {
    // PostgreSQL: 0 rows affected (version mismatch)
    return err != nil && err.Error() == "optimistic lock error"
}

func isDuplicateKeyError(err error) bool {
    // PostgreSQL: duplicate key value violates unique constraint
    return err != nil && strings.Contains(err.Error(), "duplicate key")
}
```

---

## **3. 錯誤傳播規則**

### **3.1 跨層級錯誤傳播**

```
┌───────────────────────────────────────────────────────────┐
│ HTTP Handler (Presentation Layer)                         │
├───────────────────────────────────────────────────────────┤
│ func (h *Handler) DeductPoints(c *gin.Context) {         │
│     err := h.useCase.Execute(cmd)                         │
│     if err != nil {                                       │
│         status := mapErrorToHTTPStatus(err) // 映射       │
│         c.JSON(status, toErrorResponse(err))              │
│         return                                            │
│     }                                                     │
│ }                                                         │
└───────────────────────────────────────────────────────────┘
                         ▲
                         │ ApplicationError{
                         │   Operation: "DeductPoints",
                         │   DomainErr: ErrInsufficientPoints,
                         │ }
                         │
┌───────────────────────────────────────────────────────────┐
│ Use Case (Application Layer)                              │
├───────────────────────────────────────────────────────────┤
│ func (uc *DeductPointsUseCase) Execute(cmd) error {      │
│     account := uc.repo.FindByID(cmd.AccountID)           │
│     err := account.DeductPoints(cmd.Amount, ...)         │
│     if err != nil {                                       │
│         return NewApplicationError("DeductPoints", err)   │
│     }                                                     │
│ }                                                         │
└───────────────────────────────────────────────────────────┘
                         ▲
                         │ ErrInsufficientPoints
                         │
┌───────────────────────────────────────────────────────────┐
│ Aggregate (Domain Layer)                                  │
├───────────────────────────────────────────────────────────┤
│ func (a *PointsAccount) DeductPoints(amount) error {     │
│     if a.GetAvailablePoints() < amount {                 │
│         return ErrInsufficientPoints.WithContext(...)    │
│     }                                                     │
│ }                                                         │
└───────────────────────────────────────────────────────────┘
```

### **3.2 錯誤檢查與處理**

```go
// Application Layer - Use Case
func (uc *EarnPointsFromInvoiceUseCase) Execute(cmd EarnPointsFromInvoiceCommand) error {
    return uc.txManager.InTransaction(func(ctx TransactionContext) error {
        // 1. 查詢發票交易
        invoice, err := uc.invoiceRepo.FindByNumber(ctx, cmd.InvoiceNumber)
        if err != nil {
            // 檢查是否是 Domain Error
            if errors.Is(err, invoice.ErrInvoiceNotFound) {
                uc.logger.Warn("Invoice not found",
                    zap.String("invoiceNumber", cmd.InvoiceNumber),
                )
                return NewApplicationError("EarnPointsFromInvoice", err).
                    WithContext("invoiceNumber", cmd.InvoiceNumber)
            }

            // 未預期的錯誤（資料庫連線失敗等）
            uc.logger.Error("Failed to query invoice",
                zap.String("invoiceNumber", cmd.InvoiceNumber),
                zap.Error(err),
            )
            return err
        }

        // 2. 驗證發票狀態
        if !invoice.IsVerified() {
            return NewApplicationError("EarnPointsFromInvoice", invoice.ErrInvoiceNotVerified).
                WithContext("invoiceNumber", cmd.InvoiceNumber, "status", invoice.Status())
        }

        // 3. 查詢積分帳戶
        account, err := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
        if err != nil {
            return NewApplicationError("EarnPointsFromInvoice", err).
                WithContext("memberID", cmd.MemberID)
        }

        // 4. 計算積分
        amount := uc.calculator.CalculatePoints(invoice.Amount(), invoice.InvoiceDate())

        // 5. 賺取積分
        err = account.EarnPoints(amount, points.SourceInvoice, invoice.InvoiceNumber(), "發票驗證通過")
        if err != nil {
            // Domain Error（如金額無效）
            return NewApplicationError("EarnPointsFromInvoice", err).
                WithContext("memberID", cmd.MemberID, "amount", amount.Value())
        }

        // 6. 持久化
        err = uc.accountRepo.Update(ctx, account)
        if err != nil {
            return NewApplicationError("EarnPointsFromInvoice", err).
                WithContext("accountID", account.AccountID())
        }

        return nil
    })
}
```

---

## **4. HTTP 狀態碼映射**

### **4.1 映射規則**

**位置**: Application Layer（**不是** Infrastructure Layer）

**原因**: HTTP 狀態碼是應用層關注點，不應洩漏到 Domain Layer。

```go
// Application Layer - internal/application/http/error_mapper.go
package http

import (
    "errors"
    "net/http"

    "internal/domain/points"
    "internal/domain/invoice"
    "internal/domain/member"
    "internal/application"
)

func MapErrorToHTTPStatus(err error) int {
    if err == nil {
        return http.StatusOK
    }

    // 檢查是否是 ApplicationError
    var appErr *application.ApplicationError
    if errors.As(err, &appErr) {
        err = appErr.DomainErr // 提取 Domain Error
    }

    // 檢查是否是 DomainError
    var domainErr *points.DomainError
    if errors.As(err, &domainErr) {
        return mapDomainErrorToStatus(domainErr)
    }

    // 其他未預期錯誤
    return http.StatusInternalServerError
}

func mapDomainErrorToStatus(err *points.DomainError) int {
    switch err.Code {
    // 業務規則違反 → 400 Bad Request
    case points.ErrCodeInsufficientPoints,
        points.ErrCodeInvalidAmount,
        points.ErrCodeInvalidPhoneNumber,
        invoice.ErrCodeInvoiceExpired,
        invoice.ErrCodeDuplicateInvoice:
        return http.StatusBadRequest

    // 資源未找到 → 404 Not Found
    case points.ErrCodeAccountNotFound,
        invoice.ErrCodeInvoiceNotFound,
        member.ErrCodeMemberNotFound:
        return http.StatusNotFound

    // 衝突（樂觀鎖失敗、重複註冊）→ 409 Conflict
    case points.ErrCodeOptimisticLockFailure,
        member.ErrCodePhoneNumberAlreadyRegistered:
        return http.StatusConflict

    // 權限不足 → 403 Forbidden
    case member.ErrCodePermissionDenied:
        return http.StatusForbidden

    // 未預期的 Domain Error → 500 Internal Server Error
    default:
        return http.StatusInternalServerError
    }
}
```

### **4.2 錯誤回應格式**

```go
// Application Layer - internal/application/http/error_response.go
package http

type ErrorResponse struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
    TraceID string                 `json:"trace_id,omitempty"`
}

func ToErrorResponse(err error, traceID string) ErrorResponse {
    var appErr *application.ApplicationError
    if errors.As(err, &appErr) {
        var domainErr *points.DomainError
        if errors.As(appErr.DomainErr, &domainErr) {
            return ErrorResponse{
                Code:    string(domainErr.Code),
                Message: domainErr.Message,
                Details: domainErr.Context,
                TraceID: traceID,
            }
        }

        return ErrorResponse{
            Code:    "APPLICATION_ERROR",
            Message: appErr.Error(),
            Details: appErr.Context,
            TraceID: traceID,
        }
    }

    // 未預期錯誤（不暴露內部細節）
    return ErrorResponse{
        Code:    "INTERNAL_ERROR",
        Message: "系統發生錯誤，請稍後再試",
        TraceID: traceID,
    }
}
```

### **4.3 HTTP Handler 範例**

```go
// Presentation Layer - internal/interfaces/http/points_handler.go
package http

func (h *PointsHandler) DeductPoints(c *gin.Context) {
    traceID := c.GetHeader("X-Trace-ID")

    var req DeductPointsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Code:    "INVALID_REQUEST",
            Message: "請求格式錯誤",
            Details: map[string]interface{}{"error": err.Error()},
            TraceID: traceID,
        })
        return
    }

    cmd := DeductPointsCommand{
        AccountID:   points.AccountID(req.AccountID),
        Amount:      points.PointsAmount(req.Amount),
        Reason:      req.Reason,
        ReferenceID: req.ReferenceID,
    }

    err := h.deductPointsUseCase.Execute(cmd)
    if err != nil {
        status := MapErrorToHTTPStatus(err)
        c.JSON(status, ToErrorResponse(err, traceID))

        // 記錄錯誤（Presentation Layer 職責）
        h.logger.Error("DeductPoints failed",
            zap.String("traceID", traceID),
            zap.String("accountID", req.AccountID),
            zap.Int("httpStatus", status),
            zap.Error(err),
        )
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "積分扣除成功"})
}
```

---

## **5. 日誌記錄策略**

### **5.1 各層級的日誌職責**

| Layer | 是否記錄日誌 | 記錄內容 | 原因 |
|-------|------------|---------|------|
| **Domain Layer** | ❌ 不記錄 | N/A | 不依賴基礎設施（logger） |
| **Application Layer** | ✅ 記錄 | 業務操作失敗、執行上下文 | 協調層，有 logger 依賴 |
| **Infrastructure Layer** | ✅ 記錄 | 技術錯誤（資料庫連線失敗） | 基礎設施層，記錄技術問題 |
| **Presentation Layer** | ✅ 記錄 | HTTP 請求失敗、IP、User Agent | 對外接口，記錄請求上下文 |

### **5.2 Application Layer 日誌範例**

```go
// Application Layer - Use Case with Logging
type DeductPointsUseCase struct {
    accountRepo repository.PointsAccountRepository
    txManager   transaction.TransactionManager
    logger      *zap.Logger
}

func (uc *DeductPointsUseCase) Execute(cmd DeductPointsCommand) error {
    uc.logger.Info("Executing DeductPoints",
        zap.String("accountID", cmd.AccountID.String()),
        zap.Int("amount", cmd.Amount.Value()),
        zap.String("reason", cmd.Reason),
    )

    err := uc.txManager.InTransaction(func(ctx TransactionContext) error {
        account, err := uc.accountRepo.FindByID(ctx, cmd.AccountID)
        if err != nil {
            // 查詢失敗（可能是 Domain Error 或資料庫錯誤）
            uc.logger.Warn("Failed to find account",
                zap.String("accountID", cmd.AccountID.String()),
                zap.Error(err),
            )
            return NewApplicationError("DeductPoints", err).
                WithContext("accountID", cmd.AccountID)
        }

        err = account.DeductPoints(cmd.Amount, cmd.Reason, cmd.ReferenceID)
        if err != nil {
            // 業務規則錯誤（如積分不足）
            uc.logger.Warn("Business rule violation",
                zap.String("accountID", cmd.AccountID.String()),
                zap.Error(err),
            )
            return NewApplicationError("DeductPoints", err)
        }

        err = uc.accountRepo.Update(ctx, account)
        if err != nil {
            // 持久化失敗
            uc.logger.Error("Failed to update account",
                zap.String("accountID", cmd.AccountID.String()),
                zap.Error(err),
            )
            return NewApplicationError("DeductPoints", err)
        }

        return nil
    })

    if err != nil {
        uc.logger.Error("DeductPoints failed",
            zap.String("accountID", cmd.AccountID.String()),
            zap.Error(err),
        )
        return err
    }

    uc.logger.Info("DeductPoints completed successfully",
        zap.String("accountID", cmd.AccountID.String()),
        zap.Int("amount", cmd.Amount.Value()),
    )

    return nil
}
```

### **5.3 日誌級別指南**

| 日誌級別 | 使用時機 | 範例 |
|---------|---------|------|
| **DEBUG** | 開發調試信息 | 變數值、中間狀態 |
| **INFO** | 正常業務操作 | 積分扣除成功、會員註冊完成 |
| **WARN** | 預期中的錯誤 | 積分不足、發票已過期、資源未找到 |
| **ERROR** | 未預期的錯誤 | 資料庫連線失敗、外部 API 調用失敗 |
| **FATAL** | 系統無法繼續運行 | 配置文件缺失、必要服務啟動失敗 |

---

## **6. 錯誤處理範例**

### **6.1 完整流程範例**

```go
// ===== Domain Layer =====
package points

var ErrInsufficientPoints = &DomainError{
    Code:    "INSUFFICIENT_POINTS",
    Message: "可用積分不足",
}

func (a *PointsAccount) DeductPoints(amount PointsAmount, reason string) error {
    if a.GetAvailablePoints().LessThan(amount) {
        return ErrInsufficientPoints.WithContext(
            "required", amount.Value(),
            "available", a.GetAvailablePoints().Value(),
        )
    }

    a.usedPoints = a.usedPoints.Add(amount)
    return nil
}

// ===== Application Layer =====
package application

func (uc *DeductPointsUseCase) Execute(cmd DeductPointsCommand) error {
    uc.logger.Info("Deducting points", zap.String("accountID", cmd.AccountID))

    err := uc.txManager.InTransaction(func(ctx TransactionContext) error {
        account, err := uc.accountRepo.FindByID(ctx, cmd.AccountID)
        if err != nil {
            uc.logger.Warn("Account not found", zap.Error(err))
            return NewApplicationError("DeductPoints", err)
        }

        err = account.DeductPoints(cmd.Amount, cmd.Reason)
        if err != nil {
            uc.logger.Warn("Deduction failed", zap.Error(err))
            return NewApplicationError("DeductPoints", err)
        }

        return uc.accountRepo.Update(ctx, account)
    })

    if err != nil {
        uc.logger.Error("DeductPoints use case failed", zap.Error(err))
    }

    return err
}

// ===== Presentation Layer =====
package http

func (h *PointsHandler) DeductPoints(c *gin.Context) {
    traceID := c.GetHeader("X-Trace-ID")

    var req DeductPointsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{
            Code:    "INVALID_REQUEST",
            Message: err.Error(),
            TraceID: traceID,
        })
        return
    }

    cmd := DeductPointsCommand{
        AccountID: points.AccountID(req.AccountID),
        Amount:    points.PointsAmount(req.Amount),
        Reason:    req.Reason,
    }

    err := h.useCase.Execute(cmd)
    if err != nil {
        status := MapErrorToHTTPStatus(err)
        c.JSON(status, ToErrorResponse(err, traceID))

        h.logger.Error("HTTP request failed",
            zap.String("traceID", traceID),
            zap.String("path", c.Request.URL.Path),
            zap.String("method", c.Request.Method),
            zap.Int("status", status),
            zap.Error(err),
        )
        return
    }

    c.JSON(200, gin.H{"message": "積分扣除成功"})
}
```

**錯誤傳播流程**:

```
Domain Layer:
  ↓ ErrInsufficientPoints

Application Layer:
  ↓ ApplicationError{Operation: "DeductPoints", DomainErr: ErrInsufficientPoints}
  ↓ Logger.Warn("Deduction failed")

Presentation Layer:
  ↓ MapErrorToHTTPStatus() → 400 Bad Request
  ↓ ToErrorResponse() → {"code": "INSUFFICIENT_POINTS", "message": "...", "details": {...}}
  ↓ Logger.Error("HTTP request failed", status: 400)
```

---

## **7. 常見錯誤處理反模式**

### **❌ 反模式 1: Domain Layer 依賴 Logger**

```go
// ❌ 錯誤：Domain Layer 不應依賴基礎設施
package points

type PointsAccount struct {
    logger *zap.Logger // ❌ 違反 Clean Architecture
}

func (a *PointsAccount) DeductPoints(amount PointsAmount) error {
    if a.GetAvailablePoints() < amount {
        a.logger.Warn("Insufficient points") // ❌ Domain Layer 不記錄日誌
        return ErrInsufficientPoints
    }
}
```

**✅ 正確做法**:

```go
// ✅ Domain Layer 僅返回錯誤
func (a *PointsAccount) DeductPoints(amount PointsAmount) error {
    if a.GetAvailablePoints().LessThan(amount) {
        return ErrInsufficientPoints.WithContext(
            "required", amount.Value(),
            "available", a.GetAvailablePoints().Value(),
        )
    }
    a.usedPoints = a.usedPoints.Add(amount)
    return nil
}

// ✅ Application Layer 負責記錄日誌
func (uc *DeductPointsUseCase) Execute(cmd Command) error {
    err := account.DeductPoints(cmd.Amount)
    if err != nil {
        uc.logger.Warn("Points deduction failed", zap.Error(err))
        return err
    }
    return nil
}
```

---

### **❌ 反模式 2: 基礎設施錯誤洩漏**

```go
// ❌ 錯誤：基礎設施錯誤直接返回
func (r *GormPointsAccountRepository) FindByID(id AccountID) (*PointsAccount, error) {
    var model PointsAccountModel
    err := r.db.First(&model, "account_id = ?", id).Error
    if err != nil {
        return nil, err // ❌ 返回 gorm.ErrRecordNotFound
    }
    return toDomainModel(&model), nil
}

// ❌ Application Layer 檢查基礎設施錯誤
func (uc *UseCase) Execute(cmd Command) error {
    account, err := uc.repo.FindByID(cmd.AccountID)
    if errors.Is(err, gorm.ErrRecordNotFound) { // ❌ Application Layer 知道 GORM
        return ErrAccountNotFound
    }
}
```

**✅ 正確做法**:

```go
// ✅ Infrastructure Layer 轉換錯誤
func (r *GormPointsAccountRepository) FindByID(id AccountID) (*PointsAccount, error) {
    var model PointsAccountModel
    err := r.db.First(&model, "account_id = ?", id).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, points.ErrAccountNotFound.WithContext("accountID", id)
        }
        return nil, fmt.Errorf("database error: %w", err)
    }
    return toDomainModel(&model), nil
}

// ✅ Application Layer 僅處理 Domain Errors
func (uc *UseCase) Execute(cmd Command) error {
    account, err := uc.repo.FindByID(cmd.AccountID)
    if errors.Is(err, points.ErrAccountNotFound) {
        uc.logger.Warn("Account not found")
        return err
    }
}
```

---

### **❌ 反模式 3: HTTP 狀態碼在 Domain Layer**

```go
// ❌ 錯誤：Domain Error 包含 HTTP 狀態碼
package points

type DomainError struct {
    Code       string
    Message    string
    HTTPStatus int // ❌ Domain Layer 不應知道 HTTP
}

var ErrInsufficientPoints = &DomainError{
    Code:       "INSUFFICIENT_POINTS",
    Message:    "積分不足",
    HTTPStatus: 400, // ❌
}
```

**✅ 正確做法**:

```go
// ✅ Domain Layer 僅定義業務錯誤
var ErrInsufficientPoints = &DomainError{
    Code:    "INSUFFICIENT_POINTS",
    Message: "積分不足",
}

// ✅ Application Layer 負責映射 HTTP 狀態碼
func MapErrorToHTTPStatus(err error) int {
    var domainErr *DomainError
    if errors.As(err, &domainErr) {
        switch domainErr.Code {
        case "INSUFFICIENT_POINTS":
            return http.StatusBadRequest
        case "ACCOUNT_NOT_FOUND":
            return http.StatusNotFound
        }
    }
    return http.StatusInternalServerError
}
```

---

### **❌ 反模式 4: 吞掉錯誤**

```go
// ❌ 錯誤：吞掉錯誤不處理
func (uc *UseCase) Execute(cmd Command) error {
    account, err := uc.repo.FindByID(cmd.AccountID)
    if err != nil {
        // ❌ 吞掉錯誤，不返回也不記錄
        return nil
    }
}
```

**✅ 正確做法**:

```go
// ✅ 明確處理錯誤
func (uc *UseCase) Execute(cmd Command) error {
    account, err := uc.repo.FindByID(cmd.AccountID)
    if err != nil {
        uc.logger.Error("Failed to find account", zap.Error(err))
        return NewApplicationError("Execute", err)
    }
}
```

---

## **總結**

### **錯誤處理關鍵原則**

1. **Domain Layer 定義錯誤，不記錄日誌**
2. **Application Layer 包裝錯誤，記錄上下文**
3. **Infrastructure Layer 轉換錯誤，不洩漏基礎設施**
4. **Presentation Layer 映射 HTTP 狀態碼，返回友好訊息**

### **檢查清單**

- [ ] 所有 Domain Errors 在 `internal/domain/{context}/errors.go` 定義
- [ ] Domain Layer 不依賴 `*zap.Logger`
- [ ] Infrastructure Layer 不返回 `gorm.ErrRecordNotFound` 等基礎設施錯誤
- [ ] Application Layer 使用 `ApplicationError` 包裝並記錄日誌
- [ ] HTTP Handler 使用 `MapErrorToHTTPStatus()` 映射狀態碼
- [ ] 錯誤訊息對用戶友好（不暴露內部實現細節）
- [ ] 所有錯誤都可以用 `errors.Is()` 和 `errors.As()` 檢查

---

**相關文檔**:
- `/docs/architecture/ddd/11-dependency-rules.md` - 依賴規則
- `/docs/architecture/ddd/06-layered-architecture.md` - 分層架構
- `/docs/qa/testing-conventions.md` - 測試慣例
