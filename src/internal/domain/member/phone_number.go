package member

import (
	"regexp"
)

// ===========================
// PhoneNumber Value Object
// ===========================

// PhoneNumber 手機號碼值對象
//
// 業務規則：
// 1. 必須是台灣手機號碼格式
// 2. 10位數字
// 3. 以 "09" 開頭
// 4. 只包含數字（0-9）
//
// 設計原則：
// - 不可變性（Immutability）：所有欄位為 unexported
// - 自我驗證（Self-validation）：建構函數強制驗證
// - 值相等（Value Equality）：基於內容比較，而非引用
//
// 使用範例：
//   phoneNumber, err := NewPhoneNumber("0912345678")
//   if err != nil {
//       return err // ErrInvalidPhoneNumberFormat
//   }
//   fmt.Println(phoneNumber.String()) // "0912345678"
type PhoneNumber struct {
	value string
}

// taiwanMobilePattern 台灣手機號碼正則表達式
//
// 規則：
// - ^09：必須以 09 開頭
// - [0-9]{8}：後面接 8 位數字
// - $：結尾
var taiwanMobilePattern = regexp.MustCompile(`^09[0-9]{8}$`)

// NewPhoneNumber 創建新的手機號碼值對象（Checked Constructor）
//
// 參數：
// - value: 原始手機號碼字串
//
// 返回：
// - PhoneNumber: 驗證通過的手機號碼值對象
// - error: 驗證失敗時返回 ErrInvalidPhoneNumberFormat
//
// 驗證規則：
// 1. 長度必須是 10 位
// 2. 必須以 "09" 開頭
// 3. 只能包含數字字符
//
// 錯誤範例：
// - "091234567" (9位) → ErrInvalidPhoneNumberFormat
// - "0812345678" (不是09開頭) → ErrInvalidPhoneNumberFormat
// - "0912-345-678" (包含連字號) → ErrInvalidPhoneNumberFormat
func NewPhoneNumber(value string) (PhoneNumber, error) {
	// 1. 驗證格式
	if !taiwanMobilePattern.MatchString(value) {
		return PhoneNumber{}, ErrInvalidPhoneNumberFormat.WithContext(
			"phone", value,
			"reason", "must be 10 digits starting with 09",
		)
	}

	// 2. 創建值對象
	return PhoneNumber{value: value}, nil
}

// String 返回手機號碼字串表示
//
// 返回：原始手機號碼字串（例如："0912345678"）
func (p PhoneNumber) String() string {
	return p.value
}

// Equals 比較兩個手機號碼是否相等
//
// 參數：
// - other: 另一個手機號碼值對象
//
// 返回：
// - true: 手機號碼相同
// - false: 手機號碼不同
//
// 設計原則：
// - 值對象相等性基於內容，而非引用
// - 使用字串比較，而非指針比較
func (p PhoneNumber) Equals(other PhoneNumber) bool {
	return p.value == other.value
}

// IsZero 檢查是否為零值
//
// 返回：
// - true: 手機號碼為空（零值）
// - false: 手機號碼已設定
//
// 使用場景：
// - 檢查會員是否已綁定手機號碼
// - 驗證必填欄位
func (p PhoneNumber) IsZero() bool {
	return p.value == ""
}
