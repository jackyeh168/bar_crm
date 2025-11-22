package points

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackyeh168/bar_crm/src/internal/domain/points"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
)

// ===========================
// CreatePointsAccount Use Case
// ===========================

// CreatePointsAccountCommand 創建積分帳戶的命令
//
// 輸入：
// - MemberID: 會員 ID（UUID 字串）
//
// 驗證：
// - MemberID 必須是有效的 UUID 格式
// - MemberID 不能已經有積分帳戶
type CreatePointsAccountCommand struct {
	MemberID string
}

// CreatePointsAccountResult 創建積分帳戶的結果
//
// 輸出：
// - AccountID: 新創建的帳戶 ID
// - MemberID: 會員 ID
// - InitialBalance: 初始餘額（永遠為 0）
// - CreatedAt: 創建時間
type CreatePointsAccountResult struct {
	AccountID      string
	MemberID       string
	InitialBalance int
	CreatedAt      time.Time
}

// CreatePointsAccountUseCase 創建積分帳戶 Use Case
//
// 職責：
// 1. 驗證輸入（MemberID 格式）
// 2. 創建新的積分帳戶
// 3. 保存到 Repository（在事務中）
// 4. 返回結果
//
// 設計原則：
// - 單一職責：只負責協調創建帳戶的流程
// - 依賴倒置：依賴 Repository 介面和 TransactionManager 介面
// - 事務管理：Use Case 管理事務（不依賴調用者）
// - 並發安全：依賴資料庫唯一約束，而非 check-then-insert
type CreatePointsAccountUseCase struct {
	accountRepo points.PointsAccountRepository
	txManager   shared.TransactionManager
}

// NewCreatePointsAccountUseCase 創建 Use Case 實例
func NewCreatePointsAccountUseCase(
	repo points.PointsAccountRepository,
	txManager shared.TransactionManager,
) *CreatePointsAccountUseCase {
	return &CreatePointsAccountUseCase{
		accountRepo: repo,
		txManager:   txManager,
	}
}

// Execute 執行創建積分帳戶
//
// 執行流程：
// 1. 驗證 MemberID 格式
// 2. 創建新帳戶（Domain 聚合）
// 3. 在事務中保存到 Repository
// 4. 返回結果
//
// 錯誤處理：
// - ErrInvalidMemberID: MemberID 格式無效
// - ErrAccountAlreadyExists: 會員已有積分帳戶（由資料庫唯一約束保證）
// - 其他 Repository 錯誤：添加上下文後返回
//
// 並發安全：
// - 不使用 check-then-insert 模式（避免競爭條件）
// - 依賴資料庫 UNIQUE 約束保證唯一性
func (uc *CreatePointsAccountUseCase) Execute(cmd CreatePointsAccountCommand) (*CreatePointsAccountResult, error) {
	// 1. 驗證並轉換 MemberID
	memberID, err := points.MemberIDFromString(cmd.MemberID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse member ID: %w", err)
	}

	// 2. 創建新的積分帳戶（Domain Layer）
	account, err := points.NewPointsAccount(memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to create points account: %w", err)
	}

	// 3. 在事務中保存到 Repository
	var result *CreatePointsAccountResult
	err = uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
		// 保存帳戶
		if err := uc.accountRepo.Save(ctx, account); err != nil {
			// 如果是唯一約束違反，返回更友好的錯誤訊息
			if errors.Is(err, points.ErrAccountAlreadyExists) {
				return fmt.Errorf("member already has an account: %w", err)
			}
			return fmt.Errorf("failed to save account: %w", err)
		}

		// 構建結果
		result = &CreatePointsAccountResult{
			AccountID:      account.AccountID().String(),
			MemberID:       account.MemberID().String(),
			InitialBalance: 0,
			CreatedAt:      account.CreatedAt(),
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// ExecuteWithContext 在已有事務上下文中執行創建帳戶
//
// 使用場景：
// - 當此 Use Case 需要與其他操作組合在同一個事務中
// - 調用者已經開啟了事務，傳入 TransactionContext
//
// 參數：
// - ctx: 事務上下文（由調用者的 TransactionManager 提供）
// - cmd: 創建帳戶命令
//
// 返回：
// - result: 創建結果
// - error: 錯誤（如果有）
//
// 注意：
// - 此方法不會開啟新事務（使用調用者提供的 ctx）
// - 錯誤時不會自動回滾（由調用者的 TransactionManager 處理）
func (uc *CreatePointsAccountUseCase) ExecuteWithContext(
	ctx shared.TransactionContext,
	cmd CreatePointsAccountCommand,
) (*CreatePointsAccountResult, error) {
	// 1. 驗證並轉換 MemberID
	memberID, err := points.MemberIDFromString(cmd.MemberID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse member ID: %w", err)
	}

	// 2. 創建新的積分帳戶
	account, err := points.NewPointsAccount(memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to create points account: %w", err)
	}

	// 3. 保存到 Repository（使用調用者提供的事務上下文）
	if err := uc.accountRepo.Save(ctx, account); err != nil {
		// 如果是唯一約束違反，返回更友好的錯誤訊息
		if errors.Is(err, points.ErrAccountAlreadyExists) {
			return nil, fmt.Errorf("member already has an account: %w", err)
		}
		return nil, fmt.Errorf("failed to save account: %w", err)
	}

	// 4. 返回結果
	return &CreatePointsAccountResult{
		AccountID:      account.AccountID().String(),
		MemberID:       account.MemberID().String(),
		InitialBalance: 0,
		CreatedAt:      account.CreatedAt(),
	}, nil
}
