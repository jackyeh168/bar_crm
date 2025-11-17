package points_test

import (
	"testing"
	"time"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Test 46: EarnPoints 累積積分到帳戶
func TestPointsAccount_EarnPoints_AccumulatesPointsToAccount(t *testing.T) {
	// Arrange
	account := createCleanAccount(t)
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
	account := createCleanAccount(t)
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

// Test 55: DeductPoints 從可用餘額扣減積分
func TestDeductPoints_DeductsFromAvailableBalance(t *testing.T) {
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

// Test 56: DeductPoints 餘額不足時拒絕扣減
func TestDeductPoints_RejectsWhenBalanceInsufficient(t *testing.T) {
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

// ===========================
// Mock 物件（用於 RecalculatePoints 測試）
// ===========================

// MockTransaction 實作 PointsCalculableTransaction 介面
// ISP 優化：只實作必要方法（GetAmount），不包含未使用的方法
type MockTransaction struct {
	amount int // 交易金額（TWD元）
}

func (m MockTransaction) GetAmount() int {
	return m.amount
}

// ===========================
// 測試輔助函數
// ===========================

// createCleanAccount 創建一個乾淨的帳戶（已清除創建事件）
// 用於簡化測試準備階段，讓測試更專注於業務場景
func createCleanAccount(t *testing.T) *points.PointsAccount {
	t.Helper() // 標記為測試輔助函數，錯誤會指向調用處
	account, err := points.NewPointsAccount(points.NewMemberID())
	require.NoError(t, err, "創建測試帳戶不應失敗")
	account.PullEvents() // 清除創建事件
	return account
}

// ===========================
// RecalculatePoints 命令測試
// ===========================

// Test 61: RecalculatePoints 根據交易歷史重算累積積分
func TestPointsAccount_RecalculatePoints_RecomputesEarnedFromTransactions(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// 準備測試數據
	calculator := points.NewPointsCalculationService()
	rate, _ := points.NewConversionRate(100) // 100 TWD = 1 點

	transactions := []points.PointsCalculableTransaction{
		MockTransaction{amount: 350}, // 3 點
		MockTransaction{amount: 250}, // 2 點
	}

	// Act
	err := account.RecalculatePoints(transactions, calculator, rate, "test_scenario")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 5, account.EarnedPoints().Value(), "350/100 + 250/100 = 3 + 2 = 5")
	assert.Equal(t, 0, account.UsedPoints().Value())
}

// Test 62: RecalculatePoints 偵測並拒絕導致資料不一致的重算
func TestPointsAccount_RecalculatePoints_RejectsRecalculationCausingDataCorruption(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// 先獲得 100 點並扣除 80 點
	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "test")

	deduct, _ := points.NewPointsAmount(80)
	account.DeductPoints(deduct, "兌換商品")

	// 準備重算：只有 50 點（< usedPoints 80）
	calculator := points.NewPointsCalculationService()
	rate, _ := points.NewConversionRate(100)

	transactions := []points.PointsCalculableTransaction{
		MockTransaction{amount: 5000}, // 只有 50 點
	}

	// Act
	err := account.RecalculatePoints(transactions, calculator, rate, "data_correction")

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, points.ErrInsufficientEarnedPoints)
	assert.Equal(t, 100, account.EarnedPoints().Value(), "重算失敗，積分應保持不變")
}

// Test 63: RecalculatePoints 發布 PointsRecalculated 事件
func TestPointsAccount_RecalculatePoints_PublishesEvent(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	// 先獲得一些積分
	earned, _ := points.NewPointsAmount(100)
	account.EarnPoints(earned, points.PointsSourceInvoice, "inv-1", "test")
	account.PullEvents() // 清空事件

	// 準備重算
	calculator := points.NewPointsCalculationService()
	rate, _ := points.NewConversionRate(100)

	transactions := []points.PointsCalculableTransaction{
		MockTransaction{amount: 15000}, // 150 點
	}

	// Act
	err := account.RecalculatePoints(transactions, calculator, rate, "rule_change")

	// Assert
	assert.NoError(t, err)
	events := account.PullEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "points.recalculated", events[0].EventType())
}

// Test 64: RecalculatePoints 空交易列表
func TestPointsAccount_RecalculatePoints_EmptyTransactions_Success(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	calculator := points.NewPointsCalculationService()
	rate, _ := points.NewConversionRate(100)

	var transactions []points.PointsCalculableTransaction

	// Act
	err := account.RecalculatePoints(transactions, calculator, rate, "migration")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0, account.EarnedPoints().Value())
}

// Test 65: RecalculatePoints 檢測整數溢位
func TestPointsAccount_RecalculatePoints_DetectsIntegerOverflow(t *testing.T) {
	// Arrange
	memberID := points.NewMemberID()
	account, _ := points.NewPointsAccount(memberID)

	calculator := points.NewPointsCalculationService()
	rate, _ := points.NewConversionRate(1) // 1 TWD = 1 點（最大轉換）

	// 構造會導致溢位的交易列表
	// 使用接近 math.MaxInt 的金額
	const maxInt = int(^uint(0) >> 1)
	transactions := []points.PointsCalculableTransaction{
		MockTransaction{amount: maxInt - 1000},
		MockTransaction{amount: 2000}, // 總和會溢位
	}

	// Act
	err := account.RecalculatePoints(transactions, calculator, rate, "overflow_test")

	// Assert
	// 溢位會導致 newEarnedTotal 變成負數，被 NewPointsAmount() 拒絕
	assert.Error(t, err, "整數溢位應被檢測")
	assert.ErrorIs(t, err, points.ErrNegativePointsAmount,
		"溢位導致負數，應返回 ErrNegativePointsAmount")
}

// ===========================
// ReconstructPointsAccount 測試
// ===========================

// Test 65: ReconstructPointsAccount 有效資料
func TestReconstructPointsAccount_ValidData_Success(t *testing.T) {
	// Arrange
	accountID := points.NewAccountID()
	memberID := points.NewMemberID()
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	// Act
	account, err := points.ReconstructPointsAccount(
		accountID,
		memberID,
		150, // earnedPoints
		50,  // usedPoints
		createdAt,
		updatedAt,
	)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, accountID, account.AccountID())
	assert.Equal(t, memberID, account.MemberID())
	assert.Equal(t, 150, account.EarnedPoints().Value())
	assert.Equal(t, 50, account.UsedPoints().Value())
	assert.Equal(t, 100, account.GetAvailablePoints().Value())
	assert.Len(t, account.PullEvents(), 0, "重建時不應包含事件")
}

// Test 66-70: ReconstructPointsAccount 無效輸入（表格驅動測試）
func TestReconstructPointsAccount_InvalidInputs(t *testing.T) {
	// Arrange - 準備有效的預設值
	validAccountID := points.NewAccountID()
	validMemberID := points.NewMemberID()
	now := time.Now()

	tests := []struct {
		name        string
		accountID   points.AccountID
		memberID    points.MemberID
		earned      int
		used        int
		expectedErr error
		description string
	}{
		{
			name:        "negative_earned_points",
			accountID:   validAccountID,
			memberID:    validMemberID,
			earned:      -100,
			used:        0,
			expectedErr: points.ErrCorruptedEarnedPoints,
			description: "負數累積積分應被拒絕（資料損壞）",
		},
		{
			name:        "negative_used_points",
			accountID:   validAccountID,
			memberID:    validMemberID,
			earned:      100,
			used:        -50,
			expectedErr: points.ErrCorruptedUsedPoints,
			description: "負數已使用積分應被拒絕（資料損壞）",
		},
		{
			name:        "invariant_violation_data_corruption",
			accountID:   validAccountID,
			memberID:    validMemberID,
			earned:      50,
			used:        100, // usedPoints > earnedPoints
			expectedErr: points.ErrInvariantViolation,
			description: "資料損壞（已使用 > 累積）應被檢測",
		},
		{
			name:        "empty_account_id",
			accountID:   points.AccountID{}, // 空 ID
			memberID:    validMemberID,
			earned:      100,
			used:        50,
			expectedErr: points.ErrInvalidAccountID,
			description: "空 AccountID 應被拒絕",
		},
		{
			name:        "empty_member_id",
			accountID:   validAccountID,
			memberID:    points.MemberID{}, // 空 ID
			earned:      100,
			used:        50,
			expectedErr: points.ErrInvalidMemberID,
			description: "空 MemberID 應被拒絕",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			account, err := points.ReconstructPointsAccount(
				tt.accountID,
				tt.memberID,
				tt.earned,
				tt.used,
				now,
				now,
			)

			// Assert
			assert.Error(t, err, tt.description)
			assert.Nil(t, account, "失敗時不應返回 account")
			assert.ErrorIs(t, err, tt.expectedErr, tt.description)
		})
	}
}
