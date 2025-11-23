package member

import (
	"github.com/jackyeh168/bar_crm/src/internal/domain/member"
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
)

// ===========================
// UC-001: RegisterMember Use Case
// ===========================

// RegisterMemberCommand 註冊會員指令（Input DTO）
//
// 設計原則：
// - 只包含外部輸入數據（不包含內部邏輯）
// - 使用原始類型（string），由 Use Case 轉換為 Value Object
// - 不依賴 Domain Layer（避免循環依賴）
type RegisterMemberCommand struct {
	LineUserID  string // LINE Platform User ID (33字符，以 U 開頭)
	DisplayName string // LINE 顯示名稱
	PhoneNumber string // 手機號碼（10位數字，以 09 開頭）
}

// RegisterMemberResult 註冊會員結果（Output DTO）
//
// 設計原則：
// - 只包含外部需要的數據
// - 使用原始類型（避免暴露 Domain 對象）
type RegisterMemberResult struct {
	MemberID   string // 會員 ID (UUID)
	LineUserID string // LINE UserID
}

// RegisterMemberUseCase 註冊會員 Use Case 接口
//
// 設計原則：
// - 定義在 Application Layer（業務流程編排）
// - 依賴 Domain Layer 的 Repository 接口（依賴反轉）
// - 使用 TransactionManager 保證原子性
//
// 業務規則：
// 1. LINE UserID 不能重複（一個 LINE 帳號只能註冊一次）
// 2. PhoneNumber 不能重複（一個手機號碼只能綁定一個會員）
// 3. DisplayName 不能為空
// 4. 成功註冊後返回會員 ID
//
// 使用場景：
// - LINE Bot Webhook 接收用戶註冊請求
// - Admin Portal 手動創建會員
type RegisterMemberUseCase interface {
	Execute(cmd RegisterMemberCommand) (*RegisterMemberResult, error)
}

// ===========================
// RegisterMemberUseCaseImpl
// ===========================

// RegisterMemberUseCaseImpl 註冊會員 Use Case 實作
//
// 設計原則：
// - 實作 RegisterMemberUseCase 接口
// - 依賴注入 MemberRepository 和 TransactionManager
// - 業務流程編排（orchestration），不包含業務邏輯
// - 業務邏輯在 Domain Layer（Member 聚合）
//
// 職責：
// 1. 驗證輸入（轉換為 Value Object）
// 2. 檢查業務規則（重複性）
// 3. 調用 Domain 對象執行邏輯
// 4. 協調事務（使用 TransactionManager）
type RegisterMemberUseCaseImpl struct {
	memberRepo member.MemberRepository
	txManager  shared.TransactionManager
}

// NewRegisterMemberUseCase 創建 RegisterMemberUseCase 實例
//
// 參數：
// - memberRepo: 會員倉儲接口
// - txManager: 事務管理器
//
// 返回：
// - RegisterMemberUseCase: Use Case 接口實例
func NewRegisterMemberUseCase(
	memberRepo member.MemberRepository,
	txManager shared.TransactionManager,
) RegisterMemberUseCase {
	return &RegisterMemberUseCaseImpl{
		memberRepo: memberRepo,
		txManager:  txManager,
	}
}

// Execute 執行註冊會員 Use Case
//
// 業務流程：
// 1. 驗證輸入並轉換為 Value Object
// 2. 在事務中執行：
//    a. 檢查 LINE UserID 是否已註冊
//    b. 檢查 PhoneNumber 是否已被綁定
//    c. 創建 Member 聚合
//    d. 綁定 PhoneNumber（如果提供）
//    e. 保存到資料庫
// 3. 返回結果
//
// 錯誤處理：
// - 輸入驗證失敗 → 返回 Domain 錯誤
// - LINE UserID 已存在 → member.ErrMemberAlreadyExists
// - PhoneNumber 已綁定 → member.ErrPhoneNumberAlreadyBound
// - 資料庫錯誤 → 返回原始錯誤
//
// 事務保證：
// - 所有操作在同一事務中執行
// - 任一步驟失敗，整個操作回滾
func (uc *RegisterMemberUseCaseImpl) Execute(cmd RegisterMemberCommand) (*RegisterMemberResult, error) {
	// Step 1: 驗證輸入並轉換為 Value Object
	lineUserID, err := member.NewLineUserID(cmd.LineUserID)
	if err != nil {
		return nil, err
	}

	phoneNumber, err := member.NewPhoneNumber(cmd.PhoneNumber)
	if err != nil {
		return nil, err
	}

	// Step 2: 在事務中執行業務邏輯
	var newMember *member.Member

	err = uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
		// 2a. 檢查 LINE UserID 是否已註冊
		exists, err := uc.memberRepo.ExistsByLineUserID(ctx, lineUserID)
		if err != nil {
			return err
		}
		if exists {
			return member.ErrMemberAlreadyExists.WithContext(
				"line_user_id", cmd.LineUserID,
			)
		}

		// 2b. 檢查 PhoneNumber 是否已被綁定
		exists, err = uc.memberRepo.ExistsByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			return err
		}
		if exists {
			return member.ErrPhoneNumberAlreadyBound.WithContext(
				"phone_number", cmd.PhoneNumber,
			)
		}

		// 2c. 創建 Member 聚合（業務邏輯在 Domain Layer）
		newMember, err = member.NewMember(lineUserID, cmd.DisplayName)
		if err != nil {
			return err
		}

		// 2d. 綁定 PhoneNumber
		err = newMember.BindPhoneNumber(phoneNumber)
		if err != nil {
			return err
		}

		// 2e. 保存到資料庫
		return uc.memberRepo.Save(ctx, newMember)
	})

	if err != nil {
		return nil, err
	}

	// Step 3: 返回結果（DTO 轉換）
	return &RegisterMemberResult{
		MemberID:   newMember.MemberID().String(),
		LineUserID: newMember.LineUserID().String(),
	}, nil
}
