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

// Test 48: EarnPoints 多次累加
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

// ===========================
// GetAvailablePoints 查詢測試
// ===========================

// Test 52: GetAvailablePoints 新帳戶返回零
func TestGetAvailablePoints_NewAccount_ReturnsZero(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// Act
	available := account.GetAvailablePoints()

	// Assert
	assert.Equal(t, 0, available.Value(), "新帳戶可用積分應為 0")
}

// Test 53: GetAvailablePoints 獲得積分後返回正確數量
func TestGetAvailablePoints_AfterEarning_ReturnsCorrectAmount(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "test")

	// Act
	available := account.GetAvailablePoints()

	// Assert
	assert.Equal(t, 100, available.Value(), "獲得 100 積分後，可用積分應為 100")
}

// Test 54: GetAvailablePoints 是派生值（不存儲）
func TestGetAvailablePoints_IsDerivedValue(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "test")

	// Act - 多次調用應該返回相同結果
	available1 := account.GetAvailablePoints()
	available2 := account.GetAvailablePoints()

	// Assert - 派生值每次計算應該一致
	assert.Equal(t, available1.Value(), available2.Value())
}

// ===========================
// DeductPoints 命令測試
// ===========================

// Test 55: DeductPoints 成功扣減積分
func TestDeductPoints_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// 先獲得 100 積分
	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "消費")

	// 清除事件
	account.PullEvents()

	// 準備扣減
	deduct, _ := points.NewPointsAmount(30)

	// Act
	err := account.DeductPoints(deduct, "兌換商品")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, account.EarnedPoints().Value(), "累積獲得不變")
	assert.Equal(t, 30, account.UsedPoints().Value(), "已使用應為 30")
	assert.Equal(t, 70, account.GetAvailablePoints().Value(), "可用應為 70")
}

// Test 56: DeductPoints 餘額不足返回錯誤
func TestDeductPoints_InsufficientPoints_ReturnsError(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// 只有 50 積分
	earned, _ := points.NewPointsAmount(50)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "消費")

	// 嘗試扣減 100 積分
	deduct, _ := points.NewPointsAmount(100)

	// Act
	err := account.DeductPoints(deduct, "兌換商品")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrInsufficientPoints)
	assert.Equal(t, 0, account.UsedPoints().Value(), "扣減失敗，已使用應保持為 0")
	assert.Equal(t, 50, account.GetAvailablePoints().Value(), "可用積分不變")
}

// Test 57: DeductPoints 剛好扣完所有積分
func TestDeductPoints_ExactAmount_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "消費")

	deduct, _ := points.NewPointsAmount(100)

	// Act
	err := account.DeductPoints(deduct, "兌換商品")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, account.UsedPoints().Value())
	assert.Equal(t, 0, account.GetAvailablePoints().Value(), "應該剛好扣完")
}

// Test 58: DeductPoints 零積分也可接受
func TestDeductPoints_ZeroAmount_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "消費")

	zeroAmount, _ := points.NewPointsAmount(0)

	// Act
	err := account.DeductPoints(zeroAmount, "測試零扣減")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0, account.UsedPoints().Value())
}

// Test 59: DeductPoints 發布事件
func TestDeductPoints_PublishesEvent(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "消費")
	account.PullEvents() // 清除之前的事件

	deduct, _ := points.NewPointsAmount(30)

	// Act
	account.DeductPoints(deduct, "兌換商品")

	// Assert
	events := account.PullEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.deducted", events[0].EventType())
}

// Test 60: DeductPoints 維護不變條件
func TestDeductPoints_MaintainsInvariant(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "消費")

	deduct, _ := points.NewPointsAmount(30)

	// Act
	account.DeductPoints(deduct, "兌換商品")

	// Assert - 不變條件：UsedPoints <= EarnedPoints 應該成立
	assert.True(t, account.UsedPoints().Value() <= account.EarnedPoints().Value(),
		"不變條件：已使用積分應該 <= 累積獲得積分")
}
