package points

import (
	"errors"
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================
// CreatePointsAccount Use Case 測試（TDD Red Phase）
// ===========================

// Test 1: 成功創建積分帳戶
func TestCreatePointsAccountUseCase_Success(t *testing.T) {
	// Arrange
	mockRepo := NewMockPointsAccountRepository()
	mockTxManager := NewMockTransactionManager()
	useCase := NewCreatePointsAccountUseCase(mockRepo, mockTxManager)

	memberID := points.NewMemberID()
	cmd := CreatePointsAccountCommand{
		MemberID: memberID.String(),
	}

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.AccountID)
	assert.Equal(t, memberID.String(), result.MemberID)
	assert.Equal(t, 0, result.InitialBalance)
	assert.False(t, result.CreatedAt.IsZero())

	// 驗證 Repository 被調用
	assert.Equal(t, 1, mockRepo.SaveCallCount)
	// 驗證 TransactionManager 被調用
	assert.Equal(t, 1, mockTxManager.InTransactionCallCount)
}

// Test 2: MemberID 已存在，返回錯誤
func TestCreatePointsAccountUseCase_MemberAlreadyHasAccount_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := NewMockPointsAccountRepository()
	mockTxManager := NewMockTransactionManager()
	useCase := NewCreatePointsAccountUseCase(mockRepo, mockTxManager)

	memberID := points.NewMemberID()

	// 預先創建一個帳戶（模擬資料庫中已存在）
	existingAccount, _ := points.NewPointsAccount(memberID)
	mockRepo.accounts[memberID.String()] = existingAccount

	cmd := CreatePointsAccountCommand{
		MemberID: memberID.String(),
	}

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	// 錯誤應該包含 ErrAccountAlreadyExists（被 fmt.Errorf 包裝）
	assert.True(t, errors.Is(err, points.ErrAccountAlreadyExists), "error should wrap ErrAccountAlreadyExists")

	// 驗證 Save 被調用了（但返回錯誤）
	assert.Equal(t, 1, mockRepo.SaveCallCount)
	// 驗證 TransactionManager 被調用
	assert.Equal(t, 1, mockTxManager.InTransactionCallCount)
}

// Test 3: 無效的 MemberID 格式，返回錯誤
func TestCreatePointsAccountUseCase_InvalidMemberID_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := NewMockPointsAccountRepository()
	mockTxManager := NewMockTransactionManager()
	useCase := NewCreatePointsAccountUseCase(mockRepo, mockTxManager)

	cmd := CreatePointsAccountCommand{
		MemberID: "invalid-id", // 無效 UUID
	}

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	// 錯誤應該包含 ErrInvalidMemberID（被 fmt.Errorf 包裝）
	assert.True(t, errors.Is(err, points.ErrInvalidMemberID), "error should wrap ErrInvalidMemberID")

	// 驗證 Save 沒有被調用（MemberID 驗證失敗，提前返回）
	assert.Equal(t, 0, mockRepo.SaveCallCount)
	// 驗證 TransactionManager 沒有被調用
	assert.Equal(t, 0, mockTxManager.InTransactionCallCount)
}

// Test 4: 空 MemberID，返回錯誤
func TestCreatePointsAccountUseCase_EmptyMemberID_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := NewMockPointsAccountRepository()
	mockTxManager := NewMockTransactionManager()
	useCase := NewCreatePointsAccountUseCase(mockRepo, mockTxManager)

	cmd := CreatePointsAccountCommand{
		MemberID: "", // 空字串
	}

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	// 錯誤應該包含 ErrInvalidMemberID（被 fmt.Errorf 包裝）
	assert.True(t, errors.Is(err, points.ErrInvalidMemberID), "error should wrap ErrInvalidMemberID")

	// 驗證 TransactionManager 沒有被調用
	assert.Equal(t, 0, mockTxManager.InTransactionCallCount)
}

// ===========================
// Mock Repository
// ===========================

type MockPointsAccountRepository struct {
	accounts      map[string]*points.PointsAccount
	SaveCallCount int
}

func NewMockPointsAccountRepository() *MockPointsAccountRepository {
	return &MockPointsAccountRepository{
		accounts: make(map[string]*points.PointsAccount),
	}
}

func (m *MockPointsAccountRepository) Save(ctx shared.TransactionContext, account *points.PointsAccount) error {
	m.SaveCallCount++ // 無論成功或失敗，都計數

	// 檢查是否已存在
	if _, exists := m.accounts[account.MemberID().String()]; exists {
		return points.ErrAccountAlreadyExists
	}

	m.accounts[account.MemberID().String()] = account
	return nil
}

func (m *MockPointsAccountRepository) FindByID(ctx shared.TransactionContext, accountID points.AccountID) (*points.PointsAccount, error) {
	for _, account := range m.accounts {
		if account.AccountID().String() == accountID.String() {
			return account, nil
		}
	}
	return nil, points.ErrAccountNotFound
}

func (m *MockPointsAccountRepository) FindByMemberID(ctx shared.TransactionContext, memberID points.MemberID) (*points.PointsAccount, error) {
	if account, exists := m.accounts[memberID.String()]; exists {
		return account, nil
	}
	return nil, points.ErrAccountNotFound
}

func (m *MockPointsAccountRepository) Update(ctx shared.TransactionContext, account *points.PointsAccount) error {
	m.accounts[account.MemberID().String()] = account
	return nil
}

// ===========================
// Mock TransactionManager
// ===========================

type MockTransactionManager struct {
	InTransactionCallCount int
	ShouldFail             bool
	FailError              error
}

func NewMockTransactionManager() *MockTransactionManager {
	return &MockTransactionManager{}
}

func (m *MockTransactionManager) InTransaction(fn func(ctx shared.TransactionContext) error) error {
	m.InTransactionCallCount++

	// 如果設置為失敗，返回錯誤
	if m.ShouldFail {
		return m.FailError
	}

	// 創建一個 nil context（對於 mock 來說足夠）
	// 或者可以創建一個 mock context
	var ctx shared.TransactionContext = nil

	// 執行函數
	return fn(ctx)
}
