package points

import (
	"fmt"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
)

// GetPointsBalanceQuery 查詢積分餘額的查詢
type GetPointsBalanceQuery struct {
	MemberID string
}

// GetPointsBalanceResult 查詢積分餘額的結果
type GetPointsBalanceResult struct {
	AccountID       string
	MemberID        string
	EarnedPoints    int
	UsedPoints      int
	AvailablePoints int
}

// GetPointsBalanceUseCase 查詢積分餘額 Use Case
type GetPointsBalanceUseCase struct {
	accountRepo points.PointsAccountRepository
}

// NewGetPointsBalanceUseCase 創建 Use Case 實例
func NewGetPointsBalanceUseCase(repo points.PointsAccountRepository) *GetPointsBalanceUseCase {
	return &GetPointsBalanceUseCase{
		accountRepo: repo,
	}
}

// Execute 執行查詢積分餘額
//
// 執行流程：
// 1. 驗證並轉換 MemberID
// 2. 查詢積分帳戶
// 3. 返回結果
//
// 錯誤處理：
// - ErrInvalidMemberID: MemberID 格式無效
// - ErrAccountNotFound: 帳戶不存在
// - 其他錯誤：添加上下文後返回
func (uc *GetPointsBalanceUseCase) Execute(query GetPointsBalanceQuery) (*GetPointsBalanceResult, error) {
	return uc.ExecuteWithContext(nil, query)
}

// ExecuteWithContext 在事務上下文中執行查詢
//
// 使用場景：
// - 在已有事務中查詢餘額（與其他操作組合）
// - 獨立查詢時可傳入 nil（不需要事務）
//
// 參數：
// - ctx: 事務上下文（可為 nil）
// - query: 查詢參數
//
// 返回：
// - result: 查詢結果
// - error: 錯誤（如果有）
func (uc *GetPointsBalanceUseCase) ExecuteWithContext(
	ctx shared.TransactionContext,
	query GetPointsBalanceQuery,
) (*GetPointsBalanceResult, error) {
	// 1. 驗證並轉換 MemberID
	memberID, err := points.MemberIDFromString(query.MemberID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse member ID: %w", err)
	}

	// 2. 查詢積分帳戶
	account, err := uc.accountRepo.FindByMemberID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	// 3. 返回結果
	return &GetPointsBalanceResult{
		AccountID:       account.AccountID().String(),
		MemberID:        account.MemberID().String(),
		EarnedPoints:    account.EarnedPoints().Value(),
		UsedPoints:      account.UsedPoints().Value(),
		AvailablePoints: account.GetAvailablePoints().Value(),
	}, nil
}
