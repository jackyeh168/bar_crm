package member

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================
// MemberID Value Object Tests (TDD)
// ===========================

// Test 1: Generate new MemberID
func TestNewMemberID_GeneratesValidUUID(t *testing.T) {
	// Act
	id1 := NewMemberID()
	id2 := NewMemberID()

	// Assert
	assert.NotEqual(t, id1.String(), id2.String(), "each generated ID should be unique")
	assert.False(t, id1.IsEmpty(), "generated ID should not be zero")
	assert.False(t, id2.IsEmpty(), "generated ID should not be zero")

	// Verify UUID v4 format
	_, err := uuid.Parse(id1.String())
	assert.NoError(t, err, "generated ID should be valid UUID")
}

// Test 2: Parse valid UUID string
func TestMemberIDFromString_ValidUUID_Success(t *testing.T) {
	// Arrange
	validUUIDs := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
	}

	for _, uuidStr := range validUUIDs {
		t.Run(uuidStr, func(t *testing.T) {
			// Act
			memberID, err := MemberIDFromString(uuidStr)

			// Assert
			require.NoError(t, err, "valid UUID should be accepted")
			assert.Equal(t, uuidStr, memberID.String())
		})
	}
}

// Test 3: Invalid UUID string
func TestMemberIDFromString_InvalidUUID_ReturnsError(t *testing.T) {
	// Arrange
	invalidUUIDs := []string{
		"",                                      // 空字串
		"invalid-uuid",                          // 無效格式
		"550e8400-e29b-41d4-a716",               // 太短
		"550e8400-e29b-41d4-a716-446655440000-extra", // 太長
		"not-a-uuid-at-all",                     // 完全無效
		"12345678-1234-1234-1234-12345678901z", // 包含非十六進制字符
	}

	for _, uuidStr := range invalidUUIDs {
		t.Run(uuidStr, func(t *testing.T) {
			// Act
			_, err := MemberIDFromString(uuidStr)

			// Assert
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidMemberID)
		})
	}
}

// Test 4: String representation
func TestMemberID_String_ReturnsUUIDString(t *testing.T) {
	// Arrange
	expectedUUID := "550e8400-e29b-41d4-a716-446655440000"
	memberID, _ := MemberIDFromString(expectedUUID)

	// Act
	result := memberID.String()

	// Assert
	assert.Equal(t, expectedUUID, result)
}

// Test 5: Equality comparison
func TestMemberID_Equals_ComparesValue(t *testing.T) {
	// Arrange
	uuid1 := "550e8400-e29b-41d4-a716-446655440000"
	uuid2 := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	id1a, _ := MemberIDFromString(uuid1)
	id1b, _ := MemberIDFromString(uuid1)
	id2, _ := MemberIDFromString(uuid2)

	// Assert
	assert.True(t, id1a.Equals(id1b), "same UUIDs should be equal")
	assert.False(t, id1a.Equals(id2), "different UUIDs should not be equal")
}

// Test 6: Zero value check
func TestMemberID_IsZero_ChecksNilUUID(t *testing.T) {
	// Arrange
	var zeroID MemberID
	validID := NewMemberID()

	// Assert
	assert.True(t, zeroID.IsEmpty(), "zero value should return true")
	assert.False(t, validID.IsEmpty(), "valid ID should return false")
}

// Test 7: Value object immutability
func TestMemberID_IsImmutable(t *testing.T) {
	// Arrange
	id1 := NewMemberID()
	id2 := NewMemberID()

	// Assert - different instances
	assert.NotSame(t, &id1, &id2, "should be different instances")

	// Assert - each ID is unique
	assert.NotEqual(t, id1.String(), id2.String(), "each instance should have unique ID")
}

// Test 8: Round-trip conversion
func TestMemberID_RoundTrip_PreservesValue(t *testing.T) {
	// Arrange
	originalID := NewMemberID()
	uuidString := originalID.String()

	// Act
	parsedID, err := MemberIDFromString(uuidString)

	// Assert
	require.NoError(t, err)
	assert.True(t, originalID.Equals(parsedID), "round-trip should preserve value")
	assert.Equal(t, originalID.String(), parsedID.String())
}
