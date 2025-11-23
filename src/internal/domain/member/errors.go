package member

// ===========================
// Member Domain 錯誤定義
// ===========================

// ErrorCode Member Domain 錯誤代碼
type ErrorCode string

// Member Domain 錯誤代碼常量
const (
	ErrCodeInvalidPhoneNumberFormat ErrorCode = "INVALID_PHONE_NUMBER_FORMAT"
	ErrCodeInvalidLineUserID        ErrorCode = "INVALID_LINE_USER_ID"
	ErrCodePhoneNumberAlreadyBound  ErrorCode = "PHONE_NUMBER_ALREADY_BOUND"
	ErrCodeMemberAlreadyExists      ErrorCode = "MEMBER_ALREADY_EXISTS"
	ErrCodeMemberNotFound           ErrorCode = "MEMBER_NOT_FOUND"
	ErrCodeInvalidMemberID          ErrorCode = "INVALID_MEMBER_ID"
	ErrCodeInvalidDisplayName       ErrorCode = "INVALID_DISPLAY_NAME"
	ErrCodePhoneAlreadyBound        ErrorCode = "PHONE_ALREADY_BOUND"
)

// DomainError Member Domain 錯誤結構
//
// 設計原則：
// 1. 不使用 fmt.Errorf 或 errors.New（避免字串錯誤）
// 2. 使用結構化錯誤（ErrorCode + Message + Context）
// 3. 支援錯誤包裝（errors.Is 檢查）
// 4. 提供上下文信息（WithContext 方法）
type DomainError struct {
	Code    ErrorCode
	Message string
	Context map[string]interface{}
}

// Error 實作 error 介面
func (e *DomainError) Error() string {
	if len(e.Context) == 0 {
		return e.Message
	}

	// 包含上下文信息
	return e.Message + " (context: " + formatContext(e.Context) + ")"
}

// WithContext 添加上下文信息
//
// 使用範例：
//   return ErrInvalidPhoneNumberFormat.WithContext("phone", phoneNumber, "reason", "不是10位數字")
func (e *DomainError) WithContext(keyValues ...interface{}) *DomainError {
	if len(keyValues)%2 != 0 {
		panic("WithContext requires even number of arguments (key-value pairs)")
	}

	newErr := &DomainError{
		Code:    e.Code,
		Message: e.Message,
		Context: make(map[string]interface{}),
	}

	// 複製現有上下文
	for k, v := range e.Context {
		newErr.Context[k] = v
	}

	// 添加新上下文
	for i := 0; i < len(keyValues); i += 2 {
		key, ok := keyValues[i].(string)
		if !ok {
			panic("WithContext keys must be strings")
		}
		newErr.Context[key] = keyValues[i+1]
	}

	return newErr
}

// Is 實作 errors.Is 比較
func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// formatContext 格式化上下文信息
func formatContext(context map[string]interface{}) string {
	if len(context) == 0 {
		return ""
	}

	result := ""
	for k, v := range context {
		if result != "" {
			result += ", "
		}
		result += k + "=" + formatValue(v)
	}
	return result
}

// formatValue 格式化單個值
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return "<value>"
	}
}

// ===========================
// Member Domain 錯誤實例
// ===========================

var (
	// ErrInvalidPhoneNumberFormat 手機號碼格式無效
	//
	// 觸發條件：
	// - 不是10位數字
	// - 不是以 "09" 開頭
	// - 包含非數字字符
	ErrInvalidPhoneNumberFormat = &DomainError{
		Code:    ErrCodeInvalidPhoneNumberFormat,
		Message: "手機號碼格式無效（必須是10位數字，且以09開頭）",
	}

	// ErrInvalidLineUserID LINE UserID 格式無效
	//
	// 觸發條件：
	// - 不是以 "U" 開頭
	// - 長度不正確
	ErrInvalidLineUserID = &DomainError{
		Code:    ErrCodeInvalidLineUserID,
		Message: "LINE UserID 格式無效（必須以 U 開頭）",
	}

	// ErrPhoneNumberAlreadyBound 手機號碼已被其他會員綁定
	ErrPhoneNumberAlreadyBound = &DomainError{
		Code:    ErrCodePhoneNumberAlreadyBound,
		Message: "手機號碼已被其他會員綁定",
	}

	// ErrMemberAlreadyExists 會員已存在
	ErrMemberAlreadyExists = &DomainError{
		Code:    ErrCodeMemberAlreadyExists,
		Message: "會員已存在",
	}

	// ErrMemberNotFound 會員不存在
	ErrMemberNotFound = &DomainError{
		Code:    ErrCodeMemberNotFound,
		Message: "會員不存在",
	}

	// ErrInvalidMemberID 會員 ID 無效
	ErrInvalidMemberID = &DomainError{
		Code:    ErrCodeInvalidMemberID,
		Message: "會員 ID 格式無效",
	}

	// ErrInvalidDisplayName 顯示名稱無效
	//
	// 觸發條件：
	// - 顯示名稱為空字串
	ErrInvalidDisplayName = &DomainError{
		Code:    ErrCodeInvalidDisplayName,
		Message: "顯示名稱不能為空",
	}

	// ErrPhoneAlreadyBound 手機號碼已綁定
	//
	// 觸發條件：
	// - 嘗試修改已綁定的手機號碼
	//
	// 業務規則：
	// - 手機號碼綁定後不可變更（需管理員介入）
	ErrPhoneAlreadyBound = &DomainError{
		Code:    ErrCodePhoneAlreadyBound,
		Message: "手機號碼已綁定，無法修改（需管理員介入）",
	}
)
