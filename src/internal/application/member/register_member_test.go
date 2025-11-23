package member

import (
	"errors"
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/member"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ===========================
// Mocks
// ===========================

// MockMemberRepository mock implementation of MemberRepository
type MockMemberRepository struct {
	mock.Mock
}

func (m *MockMemberRepository) Save(ctx shared.TransactionContext, mem *member.Member) error {
	args := m.Called(ctx, mem)
	return args.Error(0)
}

func (m *MockMemberRepository) FindByMemberID(ctx shared.TransactionContext, id member.MemberID) (*member.Member, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*member.Member), args.Error(1)
}

func (m *MockMemberRepository) FindByLineUserID(ctx shared.TransactionContext, lineUserID member.LineUserID) (*member.Member, error) {
	args := m.Called(ctx, lineUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*member.Member), args.Error(1)
}

func (m *MockMemberRepository) ExistsByPhoneNumber(ctx shared.TransactionContext, phoneNumber member.PhoneNumber) (bool, error) {
	args := m.Called(ctx, phoneNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockMemberRepository) ExistsByLineUserID(ctx shared.TransactionContext, lineUserID member.LineUserID) (bool, error) {
	args := m.Called(ctx, lineUserID)
	return args.Bool(0), args.Error(1)
}

// MockTransactionManager mock implementation of TransactionManager
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) InTransaction(fn func(ctx shared.TransactionContext) error) error {
	// Directly execute the function with nil context (for unit tests)
	return fn(nil)
}

// ===========================
// RegisterMemberUseCase Tests
// ===========================

// Test 1: Register member successfully
func TestRegisterMemberUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "John Doe",
		PhoneNumber: "0912345678",
	}

	// Mock: LINE UserID does not exist
	mockRepo.On("ExistsByLineUserID", mock.Anything, mock.Anything).Return(false, nil)

	// Mock: PhoneNumber does not exist
	mockRepo.On("ExistsByPhoneNumber", mock.Anything, mock.Anything).Return(false, nil)

	// Mock: Save succeeds
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.MemberID, "MemberID should be generated")
	assert.Equal(t, cmd.LineUserID, result.LineUserID)

	mockRepo.AssertExpectations(t)
}

// Test 2: Invalid LINE UserID format
func TestRegisterMemberUseCase_Execute_InvalidLineUserID_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "INVALID",
		DisplayName: "John Doe",
		PhoneNumber: "0912345678",
	}

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, member.ErrInvalidLineUserID)
	assert.Nil(t, result)

	// No repository calls should be made
	mockRepo.AssertNotCalled(t, "ExistsByLineUserID")
	mockRepo.AssertNotCalled(t, "Save")
}

// Test 3: Invalid phone number format
func TestRegisterMemberUseCase_Execute_InvalidPhoneNumber_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "John Doe",
		PhoneNumber: "12345", // Invalid format
	}

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, member.ErrInvalidPhoneNumberFormat)
	assert.Nil(t, result)

	// No repository calls should be made
	mockRepo.AssertNotCalled(t, "ExistsByLineUserID")
	mockRepo.AssertNotCalled(t, "Save")
}

// Test 4: LINE UserID already exists
func TestRegisterMemberUseCase_Execute_LineUserIDExists_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "John Doe",
		PhoneNumber: "0912345678",
	}

	// Mock: LINE UserID already exists
	mockRepo.On("ExistsByLineUserID", mock.Anything, mock.Anything).Return(true, nil)

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, member.ErrMemberAlreadyExists)
	assert.Nil(t, result)

	// Verify no Save was called
	mockRepo.AssertNotCalled(t, "Save")
}

// Test 5: Phone number already bound
func TestRegisterMemberUseCase_Execute_PhoneNumberExists_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "John Doe",
		PhoneNumber: "0912345678",
	}

	// Mock: LINE UserID does not exist
	mockRepo.On("ExistsByLineUserID", mock.Anything, mock.Anything).Return(false, nil)

	// Mock: PhoneNumber already exists
	mockRepo.On("ExistsByPhoneNumber", mock.Anything, mock.Anything).Return(true, nil)

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, member.ErrPhoneNumberAlreadyBound)
	assert.Nil(t, result)

	// Verify no Save was called
	mockRepo.AssertNotCalled(t, "Save")
}

// Test 6: Empty display name
func TestRegisterMemberUseCase_Execute_EmptyDisplayName_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "", // Empty
		PhoneNumber: "0912345678",
	}

	// Mock: LINE UserID does not exist
	mockRepo.On("ExistsByLineUserID", mock.Anything, mock.Anything).Return(false, nil)

	// Mock: PhoneNumber does not exist
	mockRepo.On("ExistsByPhoneNumber", mock.Anything, mock.Anything).Return(false, nil)

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "顯示名稱不能為空")
	assert.Nil(t, result)

	// Verify no Save was called
	mockRepo.AssertNotCalled(t, "Save")
}

// Test 7: Repository ExistsByLineUserID fails
func TestRegisterMemberUseCase_Execute_ExistsByLineUserIDFails_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "John Doe",
		PhoneNumber: "0912345678",
	}

	dbError := errors.New("database connection failed")

	// Mock: ExistsByLineUserID fails
	mockRepo.On("ExistsByLineUserID", mock.Anything, mock.Anything).Return(false, dbError)

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, dbError, err)
	assert.Nil(t, result)

	// Verify no Save was called
	mockRepo.AssertNotCalled(t, "Save")
}

// Test 8: Repository ExistsByPhoneNumber fails
func TestRegisterMemberUseCase_Execute_ExistsByPhoneNumberFails_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "John Doe",
		PhoneNumber: "0912345678",
	}

	dbError := errors.New("database connection failed")

	// Mock: LINE UserID does not exist
	mockRepo.On("ExistsByLineUserID", mock.Anything, mock.Anything).Return(false, nil)

	// Mock: ExistsByPhoneNumber fails
	mockRepo.On("ExistsByPhoneNumber", mock.Anything, mock.Anything).Return(false, dbError)

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, dbError, err)
	assert.Nil(t, result)

	// Verify no Save was called
	mockRepo.AssertNotCalled(t, "Save")
}

// Test 9: Repository Save fails
func TestRegisterMemberUseCase_Execute_SaveFails_ReturnsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)
	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "John Doe",
		PhoneNumber: "0912345678",
	}

	dbError := errors.New("database write failed")

	// Mock: LINE UserID does not exist
	mockRepo.On("ExistsByLineUserID", mock.Anything, mock.Anything).Return(false, nil)

	// Mock: PhoneNumber does not exist
	mockRepo.On("ExistsByPhoneNumber", mock.Anything, mock.Anything).Return(false, nil)

	// Mock: Save fails
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(dbError)

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, dbError, err)
	assert.Nil(t, result)

	mockRepo.AssertExpectations(t)
}

// Test 10: Verify error propagation from transaction
func TestRegisterMemberUseCase_Execute_ErrorPropagation_FromTransaction(t *testing.T) {
	// Arrange
	mockRepo := new(MockMemberRepository)
	mockTxManager := new(MockTransactionManager)

	useCase := NewRegisterMemberUseCase(mockRepo, mockTxManager)

	cmd := RegisterMemberCommand{
		LineUserID:  "U1234567890abcdef1234567890abcdef",
		DisplayName: "John Doe",
		PhoneNumber: "0912345678",
	}

	dbError := errors.New("database error during transaction")

	// Mock: LINE UserID does not exist
	mockRepo.On("ExistsByLineUserID", mock.Anything, mock.Anything).Return(false, nil)

	// Mock: ExistsByPhoneNumber fails (error during transaction)
	mockRepo.On("ExistsByPhoneNumber", mock.Anything, mock.Anything).Return(false, dbError)

	// Act
	result, err := useCase.Execute(cmd)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, dbError, err, "error should be propagated from transaction")
	assert.Nil(t, result)

	// Verify Save was not called (error occurred before Save)
	mockRepo.AssertNotCalled(t, "Save")
}
