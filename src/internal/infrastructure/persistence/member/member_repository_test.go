package member

import (
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/member"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ===========================
// MemberRepository Integration Tests
// ===========================

// setupTestDB 創建測試資料庫（in-memory SQLite）
func setupTestDB(t *testing.T) *gorm.DB {
	// 1. 使用 in-memory SQLite
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to test database")

	// 2. 自動遷移
	err = db.AutoMigrate(&MemberGORM{})
	require.NoError(t, err, "failed to migrate database schema")

	return db
}

// createTestMember 創建測試用會員
func createTestMember(t *testing.T) *member.Member {
	lineUserID, err := member.NewLineUserID("U1234567890abcdef1234567890abcdef")
	require.NoError(t, err)

	m, err := member.NewMember(lineUserID, "Test User")
	require.NoError(t, err)

	return m
}

// Test 1: Save new member successfully
func TestMemberRepository_Save_NewMember_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)

	// Act
	err := repo.Save(nil, m)

	// Assert
	require.NoError(t, err)

	// Verify in database
	var gormModel MemberGORM
	result := db.First(&gormModel, "member_id = ?", m.MemberID().String())
	require.NoError(t, result.Error)
	assert.Equal(t, m.MemberID().String(), gormModel.MemberID)
	assert.Equal(t, m.LineUserID().String(), gormModel.LineUserID)
	assert.Equal(t, m.DisplayName(), gormModel.DisplayName)
	assert.Nil(t, gormModel.PhoneNumber, "new member should not have phone number")
}

// Test 2: Save member with phone number
func TestMemberRepository_Save_WithPhoneNumber_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)

	phoneNumber, _ := member.NewPhoneNumber("0912345678")
	m.BindPhoneNumber(phoneNumber)

	// Act
	err := repo.Save(nil, m)

	// Assert
	require.NoError(t, err)

	// Verify phone number is saved
	var gormModel MemberGORM
	db.First(&gormModel, "member_id = ?", m.MemberID().String())
	require.NotNil(t, gormModel.PhoneNumber)
	assert.Equal(t, "0912345678", *gormModel.PhoneNumber)
}

// Test 3: Update existing member (Upsert)
func TestMemberRepository_Save_UpdateExisting_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)

	// 先保存
	repo.Save(nil, m)

	// Act - 綁定手機號碼後再保存
	phoneNumber, _ := member.NewPhoneNumber("0987654321")
	m.BindPhoneNumber(phoneNumber)
	err := repo.Save(nil, m)

	// Assert
	require.NoError(t, err)

	// Verify updated
	var gormModel MemberGORM
	db.First(&gormModel, "member_id = ?", m.MemberID().String())
	require.NotNil(t, gormModel.PhoneNumber)
	assert.Equal(t, "0987654321", *gormModel.PhoneNumber)

	// Verify only one record exists
	var count int64
	db.Model(&MemberGORM{}).Count(&count)
	assert.Equal(t, int64(1), count, "should only have one record (update, not insert)")
}

// Test 4: Save fails with duplicate phone number
func TestMemberRepository_Save_DuplicatePhoneNumber_ReturnsError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)

	// 創建第一個會員並綁定手機號碼
	lineUserID1, _ := member.NewLineUserID("U1234567890abcdef1234567890abcdef")
	member1, _ := member.NewMember(lineUserID1, "User 1")
	phoneNumber, _ := member.NewPhoneNumber("0912345678")
	member1.BindPhoneNumber(phoneNumber)
	repo.Save(nil, member1)

	// 創建第二個會員並綁定相同手機號碼
	lineUserID2, _ := member.NewLineUserID("Uabcdefabcdefabcdefabcdefabcdefab")
	member2, _ := member.NewMember(lineUserID2, "User 2")
	member2.BindPhoneNumber(phoneNumber)

	// Act
	err := repo.Save(nil, member2)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, member.ErrPhoneNumberAlreadyBound)
}

// Test 5: FindByMemberID success
func TestMemberRepository_FindByMemberID_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)
	repo.Save(nil, m)

	// Act
	found, err := repo.FindByMemberID(nil,m.MemberID())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.True(t, found.MemberID().Equals(m.MemberID()))
	assert.True(t, found.LineUserID().Equals(m.LineUserID()))
	assert.Equal(t, m.DisplayName(), found.DisplayName())
}

// Test 6: FindByMemberID not found
func TestMemberRepository_FindByMemberID_NotFound_ReturnsError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	nonExistentID := member.NewMemberID()

	// Act
	found, err := repo.FindByMemberID(nil,nonExistentID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, member.ErrMemberNotFound)
	assert.Nil(t, found)
}

// Test 7: FindByLineUserID success
func TestMemberRepository_FindByLineUserID_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)
	repo.Save(nil, m)

	// Act
	found, err := repo.FindByLineUserID(nil,m.LineUserID())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.True(t, found.MemberID().Equals(m.MemberID()))
	assert.True(t, found.LineUserID().Equals(m.LineUserID()))
}

// Test 8: FindByLineUserID not found
func TestMemberRepository_FindByLineUserID_NotFound_ReturnsError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	nonExistentLineUserID, _ := member.NewLineUserID("Uffffffffffffffffffffffffffffffff")

	// Act
	found, err := repo.FindByLineUserID(nil,nonExistentLineUserID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, member.ErrMemberNotFound)
	assert.Nil(t, found)
}

// Test 9: ExistsByPhoneNumber returns true when exists
func TestMemberRepository_ExistsByPhoneNumber_Exists_ReturnsTrue(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)

	phoneNumber, _ := member.NewPhoneNumber("0912345678")
	m.BindPhoneNumber(phoneNumber)
	repo.Save(nil, m)

	// Act
	exists, err := repo.ExistsByPhoneNumber(nil,phoneNumber)

	// Assert
	require.NoError(t, err)
	assert.True(t, exists, "phone number should exist")
}

// Test 10: ExistsByPhoneNumber returns false when not exists
func TestMemberRepository_ExistsByPhoneNumber_NotExists_ReturnsFalse(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	phoneNumber, _ := member.NewPhoneNumber("0987654321")

	// Act
	exists, err := repo.ExistsByPhoneNumber(nil,phoneNumber)

	// Assert
	require.NoError(t, err)
	assert.False(t, exists, "phone number should not exist")
}

// Test 11: ExistsByLineUserID returns true when exists
func TestMemberRepository_ExistsByLineUserID_Exists_ReturnsTrue(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)
	repo.Save(nil, m)

	// Act
	exists, err := repo.ExistsByLineUserID(nil,m.LineUserID())

	// Assert
	require.NoError(t, err)
	assert.True(t, exists, "LINE UserID should exist")
}

// Test 12: ExistsByLineUserID returns false when not exists
func TestMemberRepository_ExistsByLineUserID_NotExists_ReturnsFalse(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	nonExistentLineUserID, _ := member.NewLineUserID("Uffffffffffffffffffffffffffffffff")

	// Act
	exists, err := repo.ExistsByLineUserID(nil,nonExistentLineUserID)

	// Assert
	require.NoError(t, err)
	assert.False(t, exists, "LINE UserID should not exist")
}

// Test 13: Round-trip conversion preserves all fields
func TestMemberRepository_RoundTrip_PreservesAllFields(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)

	phoneNumber, _ := member.NewPhoneNumber("0923456789")
	m.BindPhoneNumber(phoneNumber)

	// Act - Save and retrieve
	repo.Save(nil, m)
	retrieved, err := repo.FindByMemberID(nil,m.MemberID())

	// Assert
	require.NoError(t, err)
	assert.True(t, retrieved.MemberID().Equals(m.MemberID()))
	assert.True(t, retrieved.LineUserID().Equals(m.LineUserID()))
	assert.Equal(t, m.DisplayName(), retrieved.DisplayName())
	assert.True(t, retrieved.PhoneNumber().Equals(m.PhoneNumber()))
	assert.True(t, retrieved.HasPhoneNumber())
	assert.Equal(t, m.CreatedAt().Unix(), retrieved.CreatedAt().Unix()) // Compare Unix timestamps (ignore nanoseconds)
	assert.Equal(t, m.UpdatedAt().Unix(), retrieved.UpdatedAt().Unix())
}

// Test 14: Save and retrieve member without phone number
func TestMemberRepository_SaveAndRetrieve_NoPhoneNumber_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewMemberRepository(db)
	m := createTestMember(t)

	// Act
	repo.Save(nil, m)
	retrieved, err := repo.FindByMemberID(nil,m.MemberID())

	// Assert
	require.NoError(t, err)
	assert.False(t, retrieved.HasPhoneNumber(), "retrieved member should not have phone number")
	assert.True(t, retrieved.PhoneNumber().IsZero(), "phone number should be zero value")
}
