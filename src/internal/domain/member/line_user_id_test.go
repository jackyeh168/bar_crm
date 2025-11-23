package member

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================
// LineUserID Value Object Tests (TDD)
// ===========================

// Test 1: Valid LINE User ID format
func TestNewLineUserID_ValidFormat_Success(t *testing.T) {
	// Arrange
	validIDs := []string{
		"U1234567890abcdef1234567890abcdef", // 標準格式 (33字符)
		"Uabcdefabcdefabcdefabcdefabcdefab", // 全小寫 (33字符)
		"UABCDEFABCDEFABCDEFABCDEFABCDEFAB", // 全大寫 (33字符)
		"U00000000000000000000000000000000", // 全0 (33字符)
		"Uffffffffffffffffffffffffffffffff", // 全f (33字符)
	}

	for _, id := range validIDs {
		t.Run(id, func(t *testing.T) {
			// Act
			lineUserID, err := NewLineUserID(id)

			// Assert
			require.NoError(t, err, "valid LINE User ID should be accepted: %s", id)
			assert.Equal(t, id, lineUserID.String())
		})
	}
}

// Test 2: Empty string
func TestNewLineUserID_Empty_ReturnsError(t *testing.T) {
	// Act
	_, err := NewLineUserID("")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidLineUserID)
}

// Test 3: Doesn't start with "U"
func TestNewLineUserID_NotStartWithU_ReturnsError(t *testing.T) {
	// Arrange
	invalidIDs := []string{
		"X1234567890abcdef1234567890abcdef", // X開頭
		"u1234567890abcdef1234567890abcdef", // 小寫u開頭
		"A1234567890abcdef1234567890abcdef", // A開頭
		"1234567890abcdef1234567890abcdef",  // 數字開頭
	}

	for _, id := range invalidIDs {
		t.Run(id, func(t *testing.T) {
			// Act
			_, err := NewLineUserID(id)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidLineUserID)
		})
	}
}

// Test 4: Invalid length
func TestNewLineUserID_InvalidLength_ReturnsError(t *testing.T) {
	// Arrange
	invalidIDs := []string{
		"U",                                  // 只有U
		"U123",                               // 太短
		"U1234567890abcdef",                  // 17字符（太短）
		"U1234567890abcdef1234567890abcdef1", // 34字符（太長）
		"U1234567890abcdef1234567890abcdefabcdef", // 38字符（太長）
	}

	for _, id := range invalidIDs {
		t.Run(id, func(t *testing.T) {
			// Act
			_, err := NewLineUserID(id)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidLineUserID)
		})
	}
}

// Test 5: Value object immutability
func TestLineUserID_IsImmutable(t *testing.T) {
	// Arrange
	id1, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	id2, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")

	// Assert - different instances
	assert.NotSame(t, &id1, &id2, "should be different instances")

	// Assert - same value
	assert.Equal(t, id1.String(), id2.String())
}

// Test 6: Equality comparison
func TestLineUserID_Equals_ComparesValue(t *testing.T) {
	// Arrange
	id1, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	id2, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	id3, _ := NewLineUserID("Uabcdefabcdefabcdefabcdefabcdefab")

	// Assert
	assert.True(t, id1.Equals(id2), "same LINE User IDs should be equal")
	assert.False(t, id1.Equals(id3), "different LINE User IDs should not be equal")
}

// Test 7: Zero value check
func TestLineUserID_IsZero_ChecksEmptyValue(t *testing.T) {
	// Arrange
	var zeroID LineUserID
	validID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")

	// Assert
	assert.True(t, zeroID.IsZero(), "zero value should return true")
	assert.False(t, validID.IsZero(), "valid ID should return false")
}
