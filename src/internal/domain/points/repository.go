package points

import "github.com/jackyeh168/bar_crm/src/internal/domain/shared"

// ===========================
// PointsAccount Repository 介面
// ===========================

// PointsAccountRepository 積分帳戶倉儲介面（核心操作）
//
// 設計原則：
// 1. 依賴倒置原則（DIP）：Domain Layer 定義介面，Infrastructure Layer 實作
// 2. 介面隔離原則（ISP）：只包含核心 CRUD 操作，不包含批次/管理操作
// 3. 聚合根持久化：每個聚合根一個 Repository
// 4. 事務支持：使用 TransactionContext 封裝事務，避免基礎設施洩漏
//
// 職責範圍：
// - 單筆記錄的 CRUD 操作
// - 業務用例所需的基本查詢
//
// 事務使用範例：
//   txManager.InTransaction(func(ctx shared.TransactionContext) error {
//       account, _ := repo.FindByID(ctx, accountID)
//       account.EarnPoints(amount, source, sourceID, description)
//       return repo.Update(ctx, account)
//   })
type PointsAccountRepository interface {
	// Save 保存新的積分帳戶
	// 前置條件：帳戶不存在（MemberID 唯一）
	// 後置條件：帳戶已持久化
	// 錯誤：ErrAccountAlreadyExists（如果 MemberID 已存在）
	Save(ctx shared.TransactionContext, account *PointsAccount) error

	// FindByID 根據帳戶 ID 查找積分帳戶
	// 返回：找到的帳戶，或 ErrAccountNotFound
	FindByID(ctx shared.TransactionContext, accountID AccountID) (*PointsAccount, error)

	// FindByMemberID 根據會員 ID 查找積分帳戶
	// 業務規則：一個會員對應一個積分帳戶（1:1 關係）
	// 返回：找到的帳戶，或 ErrAccountNotFound
	FindByMemberID(ctx shared.TransactionContext, memberID MemberID) (*PointsAccount, error)

	// Update 更新積分帳戶
	// 前置條件：帳戶已存在
	// 後置條件：帳戶狀態已更新
	// 錯誤：ErrAccountNotFound（如果帳戶不存在）
	Update(ctx shared.TransactionContext, account *PointsAccount) error
}

// PointsAccountQueryRepository 積分帳戶查詢介面（批次操作）
//
// 設計原則：
// 1. 介面隔離原則（ISP）：將批次查詢操作與核心 CRUD 分離
// 2. 嵌入核心介面：繼承所有核心操作
//
// 使用場景：
// - 管理員批次操作（積分重算）
// - 報表生成
// - 數據遷移
//
// 注意：
// - 批次操作必須使用分頁或迭代器模式
// - 禁止載入所有記錄到記憶體
//
// TODO: 添加以下方法
// - FindPaginated(ctx shared.TransactionContext, page, pageSize int) (PagedResult[*PointsAccount], error)
// - Iterate(ctx shared.TransactionContext, batchSize int) (AccountIterator, error)
type PointsAccountQueryRepository interface {
	PointsAccountRepository
	// 當前為空，等待實作 FindPaginated 和 Iterate
}

// PointsAccountAdminRepository 積分帳戶管理介面（管理操作）
//
// 設計原則：
// 1. 介面隔離原則（ISP）：將管理操作與核心操作分離
// 2. 顯式意圖：使破壞性操作難以誤用
//
// 使用場景：
// - 帳戶狀態管理（停用/重新啟用）
// - GDPR 用戶刪除請求
// - 數據修復
//
// 安全性：
// - 只有管理員服務應該依賴此介面
// - 所有操作需要審計日誌
//
// TODO: 添加以下方法
// - MarkInactive(ctx shared.TransactionContext, accountID AccountID, reason string) error
// - Reactivate(ctx shared.TransactionContext, accountID AccountID) error
// - PermanentlyDelete(ctx shared.TransactionContext, accountID AccountID, auditInfo AuditInfo) error
type PointsAccountAdminRepository interface {
	PointsAccountQueryRepository
	// 當前為空，等待實作狀態管理方法
}

// ===========================
// Repository 錯誤定義
// ===========================

// Repository 相關錯誤代碼
const (
	ErrCodeAccountNotFound      ErrorCode = "ACCOUNT_NOT_FOUND"
	ErrCodeAccountAlreadyExists ErrorCode = "ACCOUNT_ALREADY_EXISTS"
	ErrCodeRepositoryError      ErrorCode = "REPOSITORY_ERROR"
)

// Repository 錯誤實例
var (
	// ErrAccountNotFound 帳戶不存在
	ErrAccountNotFound = &DomainError{
		Code:    ErrCodeAccountNotFound,
		Message: "積分帳戶不存在",
	}

	// ErrAccountAlreadyExists 帳戶已存在
	ErrAccountAlreadyExists = &DomainError{
		Code:    ErrCodeAccountAlreadyExists,
		Message: "積分帳戶已存在",
	}

	// ErrRepositoryError 倉儲操作錯誤（通用）
	ErrRepositoryError = &DomainError{
		Code:    ErrCodeRepositoryError,
		Message: "倉儲操作失敗",
	}
)
