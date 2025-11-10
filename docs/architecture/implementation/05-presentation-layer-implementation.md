# Presentation Layer 實現指南

> **版本**: 1.0
> **最後更新**: 2025-01-10

## 1. HTTP Handler 實現

**文件**: `internal/presentation/http/handlers/points_handler.go`

```go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/yourorg/bar_crm/internal/application/usecases/points"
    "github.com/yourorg/bar_crm/internal/presentation/http/responses"
)

// PointsHandler 積分端點處理器
type PointsHandler struct {
    earnPointsUseCase  *points.EarnPointsUseCase
    queryPointsUseCase *points.QueryPointsUseCase
}

func NewPointsHandler(
    earnPointsUseCase  *points.EarnPointsUseCase,
    queryPointsUseCase *points.QueryPointsUseCase,
) *PointsHandler {
    return &PointsHandler{
        earnPointsUseCase:  earnPointsUseCase,
        queryPointsUseCase: queryPointsUseCase,
    }
}

// RegisterRoutes 註冊路由
func (h *PointsHandler) RegisterRoutes(router *gin.Engine) {
    pointsGroup := router.Group("/api/v1/points")
    {
        pointsGroup.POST("/earn", h.HandleEarnPoints)
        pointsGroup.GET("/balance/:memberID", h.HandleQueryBalance)
    }
}

// HandleEarnPoints 處理獲得積分
func (h *PointsHandler) HandleEarnPoints(c *gin.Context) {
    // 1. 綁定請求參數
    var req struct {
        MemberID    string  `json:"member_id" binding:"required"`
        Amount      float64 `json:"amount" binding:"required,gt=0"`
        InvoiceDate string  `json:"invoice_date" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        responses.Error(c, http.StatusBadRequest, "Invalid request", err)
        return
    }

    // 2. 構造 Command
    cmd := points.EarnPointsCommand{
        MemberID:    req.MemberID,
        Amount:      decimal.NewFromFloat(req.Amount),
        InvoiceDate: parseDate(req.InvoiceDate),
        Source:      "manual",
        SourceID:    "admin",
    }

    // 3. 執行 Use Case
    result, err := h.earnPointsUseCase.Execute(cmd)
    if err != nil {
        responses.Error(c, http.StatusInternalServerError, "Failed to earn points", err)
        return
    }

    // 4. 返回成功響應
    responses.Success(c, result)
}

// HandleQueryBalance 查詢積分餘額
func (h *PointsHandler) HandleQueryBalance(c *gin.Context) {
    memberID := c.Param("memberID")

    result, err := h.queryPointsUseCase.Execute(memberID)
    if err != nil {
        responses.Error(c, http.StatusNotFound, "Member not found", err)
        return
    }

    responses.Success(c, result)
}
```

## 2. 統一響應格式

**文件**: `internal/presentation/http/responses/success.go`

```go
package responses

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// SuccessResponse 成功響應
type SuccessResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
}

// Success 返回成功響應
func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, SuccessResponse{
        Success: true,
        Data:    data,
    })
}
```

**文件**: `internal/presentation/http/responses/error.go`

```go
package responses

import (
    "github.com/gin-gonic/gin"
)

// ErrorResponse 錯誤響應
type ErrorResponse struct {
    Success bool   `json:"success"`
    Error   string `json:"error"`
    Message string `json:"message"`
}

// Error 返回錯誤響應
func Error(c *gin.Context, statusCode int, message string, err error) {
    c.JSON(statusCode, ErrorResponse{
        Success: false,
        Error:   err.Error(),
        Message: message,
    })
}
```

**下一步**: 閱讀 [06-依賴注入配置](./06-dependency-injection.md)
