package points

import (
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ===========================
// PointsAccountRepository Integration Tests
// ===========================

// setupTestDB 創建測試資料庫（in-memory SQLite）
func setupTestDB(t *testing.T) *gorm.DB {
	// 1. 使用 in-memory SQLite
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to connect to test database")

	// 2. 自動遷移
	err = db.AutoMigrate(&PointsAccountGORM{})
	require.NoError(t, err, "failed to migrate database schema")

	return db
}

// createTestAccount 創建測試用積分帳戶
func createTestAccount(t *testing.T) *points.PointsAccount {
	memberID := points.NewMemberID()

	account, err := points.NewPointsAccount(memberID)
	require.NoError(t, err)

	return account
}

// Test 1: Save new account successfully
func TestPointsAccountRepository_Save_NewAccount_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)

	// Act
	err := repo.Save(nil, account)

	// Assert
	require.NoError(t, err)

	// Verify in database
	var gormModel PointsAccountGORM
	result := db.First(&gormModel, "account_id = ?", account.AccountID().String())
	require.NoError(t, result.Error)
	assert.Equal(t, account.AccountID().String(), gormModel.AccountID)
	assert.Equal(t, account.MemberID().String(), gormModel.MemberID)
	assert.Equal(t, 0, gormModel.EarnedPoints, "new account should have 0 earned points")
	assert.Equal(t, 0, gormModel.UsedPoints, "new account should have 0 used points")
}

// Test 2: Save account with points
func TestPointsAccountRepository_Save_WithPoints_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)

	// Earn some points
	amount, _ := points.NewPointsAmount(100)
	account.EarnPoints(amount, points.PointsSourceInvoice, "TX001", "Test transaction")

	// Act
	err := repo.Save(nil, account)

	// Assert
	require.NoError(t, err)

	// Verify points are saved
	var gormModel PointsAccountGORM
	db.First(&gormModel, "account_id = ?", account.AccountID().String())
	assert.Equal(t, 100, gormModel.EarnedPoints)
	assert.Equal(t, 0, gormModel.UsedPoints)
}

// Test 3: Save fails with duplicate member_id
func TestPointsAccountRepository_Save_DuplicateMemberID_ReturnsError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)

	// Create first account
	memberID := points.NewMemberID()
	account1, _ := points.NewPointsAccount(memberID)
	repo.Save(nil, account1)

	// Create second account with same memberID
	account2, _ := points.NewPointsAccount(memberID)

	// Act
	err := repo.Save(nil, account2)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrAccountAlreadyExists)
}

// Test 4: FindByID success
func TestPointsAccountRepository_FindByID_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)
	repo.Save(nil, account)

	// Act
	found, err := repo.FindByID(nil, account.AccountID())

	// Assert
	require.NoError(t, err)
	assert.True(t, found.AccountID().Equals(account.AccountID()))
	assert.True(t, found.MemberID().Equals(account.MemberID()))
	assert.Equal(t, 0, found.EarnedPoints().Value())
	assert.Equal(t, 0, found.UsedPoints().Value())
}

// Test 5: FindByID not found
func TestPointsAccountRepository_FindByID_NotFound_ReturnsError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	nonExistentID := points.NewAccountID()

	// Act
	_, err := repo.FindByID(nil, nonExistentID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrAccountNotFound)
}

// Test 6: FindByMemberID success
func TestPointsAccountRepository_FindByMemberID_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)
	repo.Save(nil, account)

	// Act
	found, err := repo.FindByMemberID(nil, account.MemberID())

	// Assert
	require.NoError(t, err)
	assert.True(t, found.AccountID().Equals(account.AccountID()))
	assert.True(t, found.MemberID().Equals(account.MemberID()))
}

// Test 7: FindByMemberID not found
func TestPointsAccountRepository_FindByMemberID_NotFound_ReturnsError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	nonExistentMemberID := points.NewMemberID()

	// Act
	_, err := repo.FindByMemberID(nil, nonExistentMemberID)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrAccountNotFound)
}

// Test 8: Update existing account
func TestPointsAccountRepository_Update_ExistingAccount_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)

	// Save initial state
	repo.Save(nil, account)

	// Modify account
	amount, _ := points.NewPointsAmount(150)
	account.EarnPoints(amount, points.PointsSourceInvoice, "TX002", "Another transaction")

	// Act
	err := repo.Update(nil, account)

	// Assert
	require.NoError(t, err)

	// Verify updated in database
	var gormModel PointsAccountGORM
	db.First(&gormModel, "account_id = ?", account.AccountID().String())
	assert.Equal(t, 150, gormModel.EarnedPoints)
	assert.Equal(t, 0, gormModel.UsedPoints)

	// Verify only one record exists
	var count int64
	db.Model(&PointsAccountGORM{}).Count(&count)
	assert.Equal(t, int64(1), count, "should only have one record (update, not insert)")
}

// Test 9: Update with deducted points
func TestPointsAccountRepository_Update_WithDeductedPoints_Success(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)

	// Earn and deduct points
	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "TX003", "Earn points")

	deducted, _ := points.NewPointsAmount(30)
	account.DeductPoints(deducted, "Redeem item")

	// Act
	err := repo.Save(nil, account)
	require.NoError(t, err)

	// Assert
	var gormModel PointsAccountGORM
	db.First(&gormModel, "account_id = ?", account.AccountID().String())
	assert.Equal(t, 100, gormModel.EarnedPoints)
	assert.Equal(t, 30, gormModel.UsedPoints)

	// Verify available points calculation
	found, _ := repo.FindByID(nil, account.AccountID())
	assert.Equal(t, 70, found.GetAvailablePoints().Value(), "available = earned - used")
}

// Test 10: Update non-existent account returns error
func TestPointsAccountRepository_Update_NonExistentAccount_ReturnsError(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)

	// Act - Update without Save (account doesn't exist in DB)
	err := repo.Update(nil, account)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrAccountNotFound)
}

// Test 11: Mapper round-trip (Domain → GORM → Domain)
func TestPointsAccountRepository_MapperRoundTrip_PreservesData(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	original := createTestAccount(t)

	// Add some points
	amount, _ := points.NewPointsAmount(200)
	original.EarnPoints(amount, points.PointsSourceSurvey, "SV001", "Survey bonus")

	// Act - Save and retrieve
	repo.Save(nil, original)
	retrieved, err := repo.FindByID(nil, original.AccountID())

	// Assert
	require.NoError(t, err)
	assert.True(t, original.AccountID().Equals(retrieved.AccountID()))
	assert.True(t, original.MemberID().Equals(retrieved.MemberID()))
	assert.Equal(t, original.EarnedPoints().Value(), retrieved.EarnedPoints().Value())
	assert.Equal(t, original.UsedPoints().Value(), retrieved.UsedPoints().Value())
	assert.Equal(t, original.GetAvailablePoints().Value(), retrieved.GetAvailablePoints().Value())
}

// Test 12: Zero points can be stored and retrieved correctly
func TestPointsAccountRepository_ZeroPoints_CorrectlyHandled(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)

	// Save account with zero points
	repo.Save(nil, account)

	// Act
	found, err := repo.FindByID(nil, account.AccountID())

	// Assert
	require.NoError(t, err)
	assert.True(t, found.EarnedPoints().IsZero(), "earned points should be zero")
	assert.True(t, found.UsedPoints().IsZero(), "used points should be zero")
	assert.True(t, found.GetAvailablePoints().IsZero(), "available points should be zero")
}

// Test 13: Timestamps are preserved
func TestPointsAccountRepository_Timestamps_PreservedCorrectly(t *testing.T) {
	// Arrange
	db := setupTestDB(t)
	repo := NewPointsAccountRepository(db)
	account := createTestAccount(t)
	originalCreatedAt := account.CreatedAt()
	originalUpdatedAt := account.UpdatedAt()

	// Act - Save and retrieve
	repo.Save(nil, account)
	found, err := repo.FindByID(nil, account.AccountID())

	// Assert
	require.NoError(t, err)
	// Timestamps should be close (within 1 second due to possible precision differences)
	assert.WithinDuration(t, originalCreatedAt, found.CreatedAt(), 1000000000) // 1 second in nanoseconds
	assert.WithinDuration(t, originalUpdatedAt, found.UpdatedAt(), 1000000000)
}
