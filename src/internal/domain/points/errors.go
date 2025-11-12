package points

import "fmt"

// ===========================
// 錯誤代碼定義
// ===========================

// ErrorCode 錯誤代碼類型
type ErrorCode string

// 錯誤代碼常量
const (
	// 積分數量相關
	ErrCodeNegativePointsAmount ErrorCode = "POINTS_NEGATIVE"
	ErrCodeInvalidPointsAmount  ErrorCode = "POINTS_INVALID"
	ErrCodeInsufficientPoints   ErrorCode = "POINTS_INSUFFICIENT"

	// 轉換率相關
	ErrCodeInvalidConversionRate ErrorCode = "CONVERSION_RATE_INVALID"

	// 帳戶相關
	ErrCodeInvalidAccountID ErrorCode = "ACCOUNT_ID_INVALID"
	ErrCodeInvalidMemberID  ErrorCode = "MEMBER_ID_INVALID"
)

// ===========================
// DomainError 結構
// ===========================

// DomainError 領域錯誤
// 設計原則：
// 1. 包含結構化的錯誤代碼（用於 HTTP 狀態碼映射）
// 2. 支持上下文信息（用於調試和日誌）
// 3. 不可變性（創建後不可修改）
type DomainError struct {
	Code    ErrorCode
	Message string
	Context map[string]interface{}
}

// Error 實現 error 接口
func (e *DomainError) Error() string {
	if len(e.Context) == 0 {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return fmt.Sprintf("[%s] %s (context: %+v)", e.Code, e.Message, e.Context)
}

// WithContext 添加上下文信息（返回新的錯誤實例，保持不可變性）
func (e *DomainError) WithContext(keyValues ...interface{}) error {
	if len(keyValues)%2 != 0 {
		panic("WithContext requires even number of arguments (key-value pairs)")
	}

	ctx := make(map[string]interface{}, len(e.Context)+len(keyValues)/2)

	// 複製現有上下文
	for k, v := range e.Context {
		ctx[k] = v
	}

	// 添加新上下文
	for i := 0; i < len(keyValues); i += 2 {
		key, ok := keyValues[i].(string)
		if !ok {
			panic(fmt.Sprintf("context key must be string, got %T", keyValues[i]))
		}
		ctx[key] = keyValues[i+1]
	}

	return &DomainError{
		Code:    e.Code,
		Message: e.Message,
		Context: ctx,
	}
}

// Is 實現 errors.Is 接口（用於錯誤類型判斷）
func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// ===========================
// 預定義錯誤
// ===========================

// 積分數量相關錯誤
var (
	ErrNegativePointsAmount = &DomainError{
		Code:    ErrCodeNegativePointsAmount,
		Message: "積分數量不能為負數",
	}

	ErrInvalidPointsAmount = &DomainError{
		Code:    ErrCodeInvalidPointsAmount,
		Message: "無效的積分數量",
	}

	ErrInsufficientPoints = &DomainError{
		Code:    ErrCodeInsufficientPoints,
		Message: "積分餘額不足",
	}
)

// 轉換率相關錯誤
var (
	ErrInvalidConversionRate = &DomainError{
		Code:    ErrCodeInvalidConversionRate,
		Message: "轉換率必須在 1-1000 之間",
	}
)

// 帳戶相關錯誤
var (
	ErrInvalidAccountID = &DomainError{
		Code:    ErrCodeInvalidAccountID,
		Message: "無效的帳戶 ID",
	}

	ErrInvalidMemberID = &DomainError{
		Code:    ErrCodeInvalidMemberID,
		Message: "無效的會員 ID",
	}
)
