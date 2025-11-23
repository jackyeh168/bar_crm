package member

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================
// PhoneNumber Value Object Tests (TDD Red Phase)
// ===========================

// Test 1: Valid Taiwan mobile number (09xxxxxxxx)
func TestNewPhoneNumber_ValidTaiwanMobile_Success(t *testing.T) {
	// Arrange
	validNumbers := []string{
		"0912345678", // 中華電信
		"0987654321", // 台灣大哥大
		"0923456789", // 遠傳電信
		"0900000000", // 邊界值
		"0999999999", // 邊界值
	}

	for _, number := range validNumbers {
		t.Run(number, func(t *testing.T) {
			// Act
			phoneNumber, err := NewPhoneNumber(number)

			// Assert
			require.NoError(t, err, "valid Taiwan mobile should be accepted: %s", number)
			assert.Equal(t, number, phoneNumber.String())
		})
	}
}

// Test 2: Invalid format - not 10 digits
func TestNewPhoneNumber_InvalidLength_ReturnsError(t *testing.T) {
	// Arrange
	invalidNumbers := []string{
		"091234567",   // 9 digits
		"09123456789", // 11 digits
		"",            // empty
		"123",         // too short
	}

	for _, number := range invalidNumbers {
		t.Run(number, func(t *testing.T) {
			// Act
			_, err := NewPhoneNumber(number)

			// Assert
			assert.Error(t, err, "invalid length should be rejected: %s", number)
			assert.ErrorIs(t, err, ErrInvalidPhoneNumberFormat)
		})
	}
}

// Test 3: Invalid format - doesn't start with 09
func TestNewPhoneNumber_NotStartWith09_ReturnsError(t *testing.T) {
	// Arrange
	invalidNumbers := []string{
		"0812345678", // 08開頭
		"1012345678", // 10開頭
		"0212345678", // 市話區碼
		"8912345678", // 沒有0開頭
	}

	for _, number := range invalidNumbers {
		t.Run(number, func(t *testing.T) {
			// Act
			_, err := NewPhoneNumber(number)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidPhoneNumberFormat)
		})
	}
}

// Test 4: Invalid format - contains non-numeric characters
func TestNewPhoneNumber_NonNumeric_ReturnsError(t *testing.T) {
	// Arrange
	invalidNumbers := []string{
		"091234567a",     // 字母
		"0912-345-678",   // 連字號
		"0912 345 678",   // 空格
		"0912.345.678",   // 點
		"+886912345678",  // 國際格式
		"(09)12345678",   // 括號
	}

	for _, number := range invalidNumbers {
		t.Run(number, func(t *testing.T) {
			// Act
			_, err := NewPhoneNumber(number)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidPhoneNumberFormat)
		})
	}
}

// Test 5: Value object immutability
func TestPhoneNumber_IsImmutable(t *testing.T) {
	// Arrange
	number1, _ := NewPhoneNumber("0912345678")
	number2, _ := NewPhoneNumber("0912345678")

	// Assert - different instances
	assert.NotSame(t, &number1, &number2, "should be different instances")

	// Assert - same value
	assert.Equal(t, number1.String(), number2.String())
}

// Test 6: Equality comparison
func TestPhoneNumber_Equals_ComparesValue(t *testing.T) {
	// Arrange
	number1, _ := NewPhoneNumber("0912345678")
	number2, _ := NewPhoneNumber("0912345678")
	number3, _ := NewPhoneNumber("0987654321")

	// Assert
	assert.True(t, number1.Equals(number2), "same phone numbers should be equal")
	assert.False(t, number1.Equals(number3), "different phone numbers should not be equal")
}
