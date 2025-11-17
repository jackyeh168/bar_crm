package points_test

import (
	"testing"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/stretchr/testify/assert"
)

// ===========================
// PointsAccount 建構測試
// ===========================

// Test 41: NewPointsAccount 成功建立
func TestNewPointsAccount_ValidMemberID_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()

	// Act
	account, err := points.NewPointsAccount(memberID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, memberID, account.MemberID())
	assert.False(t, account.AccountID().IsEmpty())
	assert.Equal(t, 0, account.EarnedPoints().Value())
	assert.Equal(t, 0, account.UsedPoints().Value())
	assert.Equal(t, 1, account.Version())
}

// Test 42: NewPointsAccount 無效 MemberID
func TestNewPointsAccount_EmptyMemberID_ReturnsError(t *testing.T) {
	// Arrange
	emptyMemberID := points.MemberID{}

	// Act
	account, err := points.NewPointsAccount(emptyMemberID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, points.ErrInvalidMemberID)
}

// Test 43: NewPointsAccount 產生唯一 AccountID
func TestNewPointsAccount_GeneratesUniqueAccountID(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()

	// Act
	account1, _ := points.NewPointsAccount(memberID)
	account2, _ := points.NewPointsAccount(memberID)

	// Assert
	assert.NotEqual(t, account1.AccountID(), account2.AccountID())
}

// Test 44: NewPointsAccount 發布 AccountCreated 事件
func TestNewPointsAccount_PublishesAccountCreatedEvent(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()

	// Act
	account, _ := points.NewPointsAccount(memberID)

	// Assert
	events := account.PullEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.account_created", events[0].EventType())
}

// Test 45: PullEvents 清空事件列表
func TestPointsAccount_PullEvents_ClearsEventList(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// Act
	events1 := account.PullEvents()
	events2 := account.PullEvents()

	// Assert
	assert.Len(t, events1, 1, "第一次拉取應該有 1 個事件")
	assert.Len(t, events2, 0, "第二次拉取應該為空（事件已被清空）")
}

// ===========================
// EarnPoints 命令測試
// ===========================

// Test 46: EarnPoints 成功獲得積分
func TestPointsAccount_EarnPoints_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)
	account.PullEvents() // 清除創建事件

	amount, _ := points.NewPointsAmount(100)

	// Act
	err := account.EarnPoints(
		amount,
		points.PointsSourceInvoice,
		"invoice-123",
		"購買商品",
	)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, account.EarnedPoints().Value())
	assert.Equal(t, 0, account.UsedPoints().Value())
	assert.Equal(t, 2, account.Version(), "版本號應該從 1 增加到 2")
}

// Test 47: EarnPoints 發布 PointsEarned 事件
func TestPointsAccount_EarnPoints_PublishesEvent(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)
	account.PullEvents() // 清除創建事件

	amount, _ := points.NewPointsAmount(100)

	// Act
	err := account.EarnPoints(
		amount,
		points.PointsSourceInvoice,
		"invoice-123",
		"購買商品",
	)

	// Assert
	assert.NoError(t, err)
	events := account.PullEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.earned", events[0].EventType())
}

// Test 48: EarnPoints 版本號遞增
func TestPointsAccount_EarnPoints_IncrementsVersion(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)
	initialVersion := account.Version()

	amount, _ := points.NewPointsAmount(100)

	// Act
	account.EarnPoints(amount, points.PointsSourceInvoice, "inv-1", "test")

	// Assert
	assert.Equal(t, initialVersion+1, account.Version())
}

// Test 49: EarnPoints 多次累加
func TestPointsAccount_EarnPoints_Accumulates(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	amount1, _ := points.NewPointsAmount(100)
	amount2, _ := points.NewPointsAmount(50)
	amount3, _ := points.NewPointsAmount(25)

	// Act
	account.EarnPoints(amount1, points.PointsSourceInvoice, "inv-1", "消費獲得")
	account.EarnPoints(amount2, points.PointsSourceInvoice, "inv-2", "消費獲得")
	account.EarnPoints(amount3, points.PointsSourceSurvey, "survey-1", "問卷獎勵")

	// Assert
	assert.Equal(t, 175, account.EarnedPoints().Value())
	assert.Equal(t, 4, account.Version(), "初始版本 1 + 3 次操作 = 4")
}

// Test 50: EarnPoints 零積分也可接受
func TestPointsAccount_EarnPoints_ZeroAmount_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	zeroAmount, _ := points.NewPointsAmount(0)

	// Act
	err := account.EarnPoints(
		zeroAmount,
		points.PointsSourceInvoice,
		"invoice-0",
		"零金額發票",
	)

	// Assert - 零積分應該可以接受（業務上可能存在）
	assert.NoError(t, err)
	assert.Equal(t, 0, account.EarnedPoints().Value())
}

// Test 51: EarnPoints 維護不變條件
func TestPointsAccount_EarnPoints_MaintainsInvariant(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	amount, _ := points.NewPointsAmount(100)

	// Act
	account.EarnPoints(amount, points.PointsSourceInvoice, "inv-1", "test")

	// Assert - 不變條件：UsedPoints <= EarnedPoints 應該成立
	assert.True(t, account.UsedPoints().Value() <= account.EarnedPoints().Value(),
		"不變條件：已使用積分應該 <= 累積獲得積分")
}
