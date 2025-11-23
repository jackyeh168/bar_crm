package member

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================
// Member Aggregate Tests (TDD)
// ===========================

// Test 1: Create new member successfully
func TestNewMember_ValidInput_Success(t *testing.T) {
	// Arrange
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	displayName := "John Doe"

	// Act
	member, err := NewMember(lineUserID, displayName)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, member)
	assert.False(t, member.MemberID().IsEmpty())
	assert.True(t, member.LineUserID().Equals(lineUserID))
	assert.Equal(t, displayName, member.DisplayName())
	assert.False(t, member.HasPhoneNumber(), "new member should not have phone number")
	assert.False(t, member.CreatedAt().IsZero())
	assert.False(t, member.UpdatedAt().IsZero())
}

// Test 2: Empty display name should return error
func TestNewMember_EmptyDisplayName_ReturnsError(t *testing.T) {
	// Arrange
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")

	// Act
	member, err := NewMember(lineUserID, "")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, member)
}

// Test 3: Bind phone number successfully
func TestMember_BindPhoneNumber_Success(t *testing.T) {
	// Arrange
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	member, _ := NewMember(lineUserID, "John Doe")
	phoneNumber, _ := NewPhoneNumber("0912345678")

	// Act
	err := member.BindPhoneNumber(phoneNumber)

	// Assert
	require.NoError(t, err)
	assert.True(t, member.HasPhoneNumber())
	assert.True(t, member.PhoneNumber().Equals(phoneNumber))
}

// Test 4: Cannot bind phone number twice
func TestMember_BindPhoneNumber_AlreadyBound_ReturnsError(t *testing.T) {
	// Arrange
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	member, _ := NewMember(lineUserID, "John Doe")
	phoneNumber1, _ := NewPhoneNumber("0912345678")
	phoneNumber2, _ := NewPhoneNumber("0987654321")

	// 先綁定第一個號碼
	member.BindPhoneNumber(phoneNumber1)

	// Act - 嘗試綁定第二個號碼
	err := member.BindPhoneNumber(phoneNumber2)

	// Assert
	assert.Error(t, err, "should not allow binding second phone number")
	assert.True(t, member.PhoneNumber().Equals(phoneNumber1), "should keep first phone number")
}

// Test 5: UpdatedAt changes when binding phone number
func TestMember_BindPhoneNumber_UpdatesTimestamp(t *testing.T) {
	// Arrange
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	member, _ := NewMember(lineUserID, "John Doe")
	originalUpdatedAt := member.UpdatedAt()

	// 等待一小段時間確保時間戳會不同
	time.Sleep(1 * time.Millisecond)

	phoneNumber, _ := NewPhoneNumber("0912345678")

	// Act
	member.BindPhoneNumber(phoneNumber)

	// Assert
	assert.True(t, member.UpdatedAt().After(originalUpdatedAt), "UpdatedAt should be updated")
}

// Test 6: Reconstruct member from database
func TestReconstructMember_ValidData_Success(t *testing.T) {
	// Arrange
	memberID := NewMemberID()
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	phoneNumber, _ := NewPhoneNumber("0912345678")
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	// Act
	member, err := ReconstructMember(
		memberID,
		lineUserID,
		"John Doe",
		phoneNumber,
		createdAt,
		updatedAt,
		1, // version
	)

	// Assert
	require.NoError(t, err)
	assert.True(t, member.MemberID().Equals(memberID))
	assert.True(t, member.LineUserID().Equals(lineUserID))
	assert.Equal(t, "John Doe", member.DisplayName())
	assert.True(t, member.PhoneNumber().Equals(phoneNumber))
	assert.True(t, member.HasPhoneNumber())
	assert.Equal(t, createdAt, member.CreatedAt())
	assert.Equal(t, updatedAt, member.UpdatedAt())
}

// Test 7: Reconstruct member without phone number
func TestReconstructMember_NoPhoneNumber_Success(t *testing.T) {
	// Arrange
	memberID := NewMemberID()
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	var zeroPhoneNumber PhoneNumber // 零值
	createdAt := time.Now()
	updatedAt := time.Now()

	// Act
	member, err := ReconstructMember(
		memberID,
		lineUserID,
		"John Doe",
		zeroPhoneNumber,
		createdAt,
		updatedAt,
		1, // version
	)

	// Assert
	require.NoError(t, err)
	assert.False(t, member.HasPhoneNumber(), "member should not have phone number")
	assert.True(t, member.PhoneNumber().IsZero())
}

// Test 8: Reconstruct member with empty display name fails
func TestReconstructMember_EmptyDisplayName_ReturnsError(t *testing.T) {
	// Arrange
	memberID := NewMemberID()
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	var zeroPhoneNumber PhoneNumber
	createdAt := time.Now()
	updatedAt := time.Now()

	// Act
	member, err := ReconstructMember(
		memberID,
		lineUserID,
		"", // 空的顯示名稱
		zeroPhoneNumber,
		createdAt,
		updatedAt,
		1, // version
	)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, member)
}

// Test 9: CreatedAt is immutable
func TestMember_CreatedAt_IsImmutable(t *testing.T) {
	// Arrange
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	member, _ := NewMember(lineUserID, "John Doe")
	originalCreatedAt := member.CreatedAt()

	// Act - 綁定手機號碼（觸發 UpdatedAt 變更）
	phoneNumber, _ := NewPhoneNumber("0912345678")
	member.BindPhoneNumber(phoneNumber)

	// Assert
	assert.Equal(t, originalCreatedAt, member.CreatedAt(), "CreatedAt should never change")
}

// Test 10: Getters return correct values
func TestMember_Getters_ReturnCorrectValues(t *testing.T) {
	// Arrange
	lineUserID, _ := NewLineUserID("U1234567890abcdef1234567890abcdef")
	displayName := "John Doe"
	member, _ := NewMember(lineUserID, displayName)
	phoneNumber, _ := NewPhoneNumber("0912345678")
	member.BindPhoneNumber(phoneNumber)

	// Assert - MemberID
	assert.False(t, member.MemberID().IsEmpty())

	// Assert - LineUserID
	assert.True(t, member.LineUserID().Equals(lineUserID))

	// Assert - DisplayName
	assert.Equal(t, displayName, member.DisplayName())

	// Assert - PhoneNumber
	assert.True(t, member.PhoneNumber().Equals(phoneNumber))
	assert.True(t, member.HasPhoneNumber())

	// Assert - Timestamps
	assert.False(t, member.CreatedAt().IsZero())
	assert.False(t, member.UpdatedAt().IsZero())
}
