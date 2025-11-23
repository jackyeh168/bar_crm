package member

import (
	"strings"
)

// ===========================
// LineUserID Value Object
// ===========================

// LineUserID LINE Platform 用戶 ID 值對象
//
// 業務規則：
// 1. 必須符合 LINE Platform 的 User ID 格式
// 2. 必須以 "U" 開頭
// 3. 長度為 33 個字符（U + 32位十六進制字符）
//
// 設計原則：
// - 不可變性（Immutability）
// - 自我驗證（Self-validation）
// - 值相等（Value Equality）
//
// LINE User ID 格式範例：
//   "U1234567890abcdef1234567890abcdef"
//
// 參考：
// https://developers.line.biz/en/docs/messaging-api/user-ids/
type LineUserID struct {
	value string
}

// NewLineUserID 創建新的 LINE UserID 值對象（Checked Constructor）
//
// 參數：
// - value: LINE Platform 提供的 User ID
//
// 返回：
// - LineUserID: 驗證通過的 LINE UserID 值對象
// - error: 驗證失敗時返回 ErrInvalidLineUserID
//
// 驗證規則：
// 1. 不能為空
// 2. 必須以 "U" 開頭
// 3. 長度必須為 33 個字符
//
// 錯誤範例：
// - "" (空字串) → ErrInvalidLineUserID
// - "X1234..." (不是U開頭) → ErrInvalidLineUserID
// - "U123" (長度不足) → ErrInvalidLineUserID
func NewLineUserID(value string) (LineUserID, error) {
	// 1. 檢查是否為空
	if value == "" {
		return LineUserID{}, ErrInvalidLineUserID.WithContext(
			"line_user_id", value,
			"reason", "cannot be empty",
		)
	}

	// 2. 檢查是否以 "U" 開頭
	if !strings.HasPrefix(value, "U") {
		return LineUserID{}, ErrInvalidLineUserID.WithContext(
			"line_user_id", value,
			"reason", "must start with U",
		)
	}

	// 3. 檢查長度（LINE User ID 標準長度為 33）
	if len(value) != 33 {
		return LineUserID{}, ErrInvalidLineUserID.WithContext(
			"line_user_id", value,
			"reason", "must be 33 characters long",
		)
	}

	// 4. 創建值對象
	return LineUserID{value: value}, nil
}

// String 返回 LINE UserID 字串表示
func (l LineUserID) String() string {
	return l.value
}

// Equals 比較兩個 LINE UserID 是否相等
func (l LineUserID) Equals(other LineUserID) bool {
	return l.value == other.value
}

// IsZero 檢查是否為零值
func (l LineUserID) IsZero() bool {
	return l.value == ""
}
