package shared_test

import (
	"errors"
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
	"github.com/stretchr/testify/assert"
)

// 定義測試用的標記類型
type TestEntityAMarker struct{}
type TestEntityBMarker struct{}

// 類型別名用於測試
type TestEntityAID = shared.EntityID[TestEntityAMarker]
type TestEntityBID = shared.EntityID[TestEntityBMarker]

// 測試用錯誤（模擬 DomainError）
type MockDomainError struct {
	message string
	context map[string]interface{}
}

func (e *MockDomainError) Error() string {
	return e.message
}

func (e *MockDomainError) WithContext(keyValues ...interface{}) error {
	ctx := make(map[string]interface{})
	for i := 0; i < len(keyValues); i += 2 {
		if i+1 < len(keyValues) {
			key := keyValues[i].(string)
			ctx[key] = keyValues[i+1]
		}
	}
	return &MockDomainError{
		message: e.message,
		context: ctx,
	}
}

var ErrInvalidTestEntityA = &MockDomainError{message: "invalid test entity A ID"}
var ErrInvalidTestEntityB = &MockDomainError{message: "invalid test entity B ID"}

// ===== EntityID[T] 基礎測試 =====

// Test 1: NewEntityID 生成唯一 UUID
func TestNewEntityID_GeneratesUniqueUUIDs(t *testing.T) {
	// Act
	id1 := shared.NewEntityID[TestEntityAMarker]()
	id2 := shared.NewEntityID[TestEntityAMarker]()

	// Assert
	assert.NotEqual(t, "", id1.String())
	assert.NotEqual(t, "", id2.String())
	assert.NotEqual(t, id1.String(), id2.String(), "每次生成的 UUID 應該不同")
}

// Test 2: EntityIDFromString 解析有效 UUID
func TestEntityIDFromString_ValidUUID_Success(t *testing.T) {
	// Arrange
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	// Act
	id, err := shared.EntityIDFromString[TestEntityAMarker](validUUID, ErrInvalidTestEntityA)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, validUUID, id.String())
}

// Test 3: EntityIDFromString 解析無效 UUID 返回錯誤
func TestEntityIDFromString_InvalidUUID_ReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"空字串", ""},
		{"不是 UUID 格式", "not-a-uuid"},
		{"錯誤格式", "123-456-789"},
		{"部分 UUID", "550e8400-e29b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			id, err := shared.EntityIDFromString[TestEntityAMarker](tt.value, ErrInvalidTestEntityA)

			// Assert
			assert.Error(t, err)
			assert.True(t, id.IsEmpty(), "解析失敗應該返回空 ID")

			// 驗證錯誤是正確的類型
			var mockErr *MockDomainError
			assert.True(t, errors.As(err, &mockErr), "應該返回 MockDomainError")
			assert.Equal(t, "invalid test entity A ID", mockErr.message)
		})
	}
}

// Test 4: Equals 比較相同 UUID
func TestEntityID_Equals_SameUUID_ReturnsTrue(t *testing.T) {
	// Arrange
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	id1, _ := shared.EntityIDFromString[TestEntityAMarker](uuid, ErrInvalidTestEntityA)
	id2, _ := shared.EntityIDFromString[TestEntityAMarker](uuid, ErrInvalidTestEntityA)

	// Act & Assert
	assert.True(t, id1.Equals(id2))
}

// Test 5: Equals 比較不同 UUID
func TestEntityID_Equals_DifferentUUID_ReturnsFalse(t *testing.T) {
	// Arrange
	id1 := shared.NewEntityID[TestEntityAMarker]()
	id2 := shared.NewEntityID[TestEntityAMarker]()

	// Act & Assert
	assert.False(t, id1.Equals(id2))
}

// Test 6: IsEmpty 判斷空 ID
func TestEntityID_IsEmpty(t *testing.T) {
	// Arrange
	emptyID := TestEntityAID{} // 零值
	validID := shared.NewEntityID[TestEntityAMarker]()

	// Act & Assert
	assert.True(t, emptyID.IsEmpty(), "零值應該是空 ID")
	assert.False(t, validID.IsEmpty(), "生成的 ID 不應該為空")
}

// Test 7: String 轉換為小寫 UUID
func TestEntityID_String_ReturnsLowercaseUUID(t *testing.T) {
	// Arrange - 使用大寫 UUID 測試
	upperUUID := "550E8400-E29B-41D4-A716-446655440000"

	// Act
	id, _ := shared.EntityIDFromString[TestEntityAMarker](upperUUID, ErrInvalidTestEntityA)

	// Assert - uuid.Parse 會規範化為小寫
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id.String())
}

// ===== 類型安全測試 =====

// Test 8: 不同標記類型的 ID 是不同類型（編譯時保證）
func TestEntityID_TypeSafety_DifferentMarkers(t *testing.T) {
	// Arrange
	idA := shared.NewEntityID[TestEntityAMarker]()
	idB := shared.NewEntityID[TestEntityBMarker]()

	// Assert - 類型不同
	assert.IsType(t, TestEntityAID{}, idA)
	assert.IsType(t, TestEntityBID{}, idB)

	// 以下代碼無法編譯（類型不匹配）：
	// idA.Equals(idB) // ✗ 編譯錯誤

	// 這是類型安全的保證：AccountID 不能和 MemberID 比較
}

// Test 9: 類型別名保持類型安全
func TestEntityID_TypeAlias_PreservesTypeSafety(t *testing.T) {
	// Arrange
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	// Act - 使用類型別名
	idA, _ := shared.EntityIDFromString[TestEntityAMarker](validUUID, ErrInvalidTestEntityA)
	idB, _ := shared.EntityIDFromString[TestEntityBMarker](validUUID, ErrInvalidTestEntityB)

	// Assert - 雖然底層 UUID 相同，但類型不同
	assert.Equal(t, validUUID, idA.String())
	assert.Equal(t, validUUID, idB.String())

	// 類型安全：不能將 idA 賦值給 TestEntityBID 類型的變量
	// var wrongType TestEntityBID = idA // ✗ 編譯錯誤
}

// ===== 錯誤處理測試 =====

// Test 10: EntityIDFromString 使用正確的錯誤類型
func TestEntityIDFromString_UsesCorrectErrorType(t *testing.T) {
	// Arrange
	invalidUUID := "not-a-uuid"

	// Act - 使用不同的錯誤模板
	idA, errA := shared.EntityIDFromString[TestEntityAMarker](invalidUUID, ErrInvalidTestEntityA)
	idB, errB := shared.EntityIDFromString[TestEntityBMarker](invalidUUID, ErrInvalidTestEntityB)

	// Assert - 錯誤類型不同
	assert.Error(t, errA)
	assert.Error(t, errB)

	var mockErrA, mockErrB *MockDomainError
	assert.True(t, errors.As(errA, &mockErrA))
	assert.True(t, errors.As(errB, &mockErrB))

	assert.Equal(t, "invalid test entity A ID", mockErrA.message)
	assert.Equal(t, "invalid test entity B ID", mockErrB.message)

	assert.True(t, idA.IsEmpty())
	assert.True(t, idB.IsEmpty())
}

// Test 11: EntityIDFromString 添加上下文信息
func TestEntityIDFromString_AddsContextToError(t *testing.T) {
	// Arrange
	invalidUUID := "bad-uuid"

	// Act
	_, err := shared.EntityIDFromString[TestEntityAMarker](invalidUUID, ErrInvalidTestEntityA)

	// Assert
	assert.Error(t, err)

	var mockErr *MockDomainError
	assert.True(t, errors.As(err, &mockErr))

	// 驗證上下文包含輸入值
	assert.NotNil(t, mockErr.context)
	assert.Equal(t, "bad-uuid", mockErr.context["input"])
	assert.NotNil(t, mockErr.context["parse_error"])
}

// Test 12: EntityIDFromString 處理不支持 WithContext 的錯誤
func TestEntityIDFromString_HandlesErrorsWithoutWithContext(t *testing.T) {
	// Arrange
	invalidUUID := "not-a-uuid"
	simpleErr := errors.New("simple error")

	// Act
	id, err := shared.EntityIDFromString[TestEntityAMarker](invalidUUID, simpleErr)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, simpleErr, err, "應該直接返回原始錯誤")
	assert.True(t, id.IsEmpty())
}

// ===== 實際使用場景測試 =====

// Test 13: 模擬實際業務場景 - 創建和查找實體
func TestEntityID_RealWorldScenario_CreateAndLookup(t *testing.T) {
	// Scenario: 創建一個實體並保存其 ID，稍後通過 ID 查找
	t.Run("Create Entity", func(t *testing.T) {
		// Arrange - 創建新實體
		entityID := shared.NewEntityID[TestEntityAMarker]()

		// Act - 保存 ID（模擬）
		savedIDString := entityID.String()

		// Assert
		assert.NotEmpty(t, savedIDString)
		assert.False(t, entityID.IsEmpty())
	})

	t.Run("Lookup Entity", func(t *testing.T) {
		// Arrange - 從數據庫讀取 ID 字串（模擬）
		savedIDString := "550e8400-e29b-41d4-a716-446655440000"

		// Act - 解析 ID
		entityID, err := shared.EntityIDFromString[TestEntityAMarker](
			savedIDString,
			ErrInvalidTestEntityA,
		)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, savedIDString, entityID.String())
		assert.False(t, entityID.IsEmpty())
	})
}

// Test 14: 並發安全測試（EntityID 應該是值類型，天然並發安全）
func TestEntityID_ConcurrencySafe(t *testing.T) {
	// Arrange
	const goroutines = 100
	ids := make([]TestEntityAID, goroutines)
	done := make(chan bool)

	// Act - 並發生成 ID
	for i := 0; i < goroutines; i++ {
		go func(index int) {
			ids[index] = shared.NewEntityID[TestEntityAMarker]()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Assert - 所有 ID 應該唯一且有效
	uniqueIDs := make(map[string]bool)
	for _, id := range ids {
		assert.False(t, id.IsEmpty(), "生成的 ID 不應為空")
		idStr := id.String()
		assert.False(t, uniqueIDs[idStr], "ID 應該唯一: %s", idStr)
		uniqueIDs[idStr] = true
	}

	assert.Equal(t, goroutines, len(uniqueIDs), "應該生成 %d 個唯一 ID", goroutines)
}
