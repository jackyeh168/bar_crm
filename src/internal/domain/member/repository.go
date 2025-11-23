package member

import (
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
)

// ===========================
// MemberRepository Interface
// ===========================

// MemberRepository 會員倉儲接口
//
// 設計原則：
// - 接口定義在 Domain Layer（依賴反轉原則）
// - 具體實現在 Infrastructure Layer
// - 返回 Domain 對象，不暴露資料庫細節
// - 使用 TransactionContext 支持事務管理
//
// 事務管理策略（Transaction Management Strategy）：
//
// Write Operations (寫操作) - ctx 必須 non-nil (強制事務)：
//   - Save(): 創建或更新會員（必須在事務中保證原子性）
//
// Read Operations (讀操作) - ctx 可為 nil (可選事務參與)：
//   - FindByMemberID(): 根據 ID 查詢
//   - FindByLineUserID(): 根據 LINE UserID 查詢
//   - ExistsByPhoneNumber(): 檢查手機號碼是否存在（效能優化：COUNT 查詢）
//   - ExistsByLineUserID(): 檢查 LINE UserID 是否存在（效能優化：COUNT 查詢）
//
// 如果 ctx != nil，讀操作參與當前事務（保證一致性）
// 如果 ctx == nil，讀操作使用獨立連接（提升性能）
//
// 使用場景範例：
//
// 1. 註冊流程（所有操作在同一事務中）：
//    txManager.InTransaction(func(ctx shared.TransactionContext) error {
//        // 檢查重複性（在事務中）
//        exists, _ := repo.ExistsByLineUserID(ctx, lineUserID)
//        if exists {
//            return ErrMemberAlreadyExists
//        }
//
//        // 保存會員（在事務中）
//        return repo.Save(ctx, member)
//    })
//
// 2. 獨立查詢（不需要事務）：
//    member, err := repo.FindByLineUserID(nil, lineUserID)
//
// 注意事項：
// - Save() 用於新增和更新（Upsert 模式，基於 MemberID）
// - FindByXXX() 找不到時返回 ErrMemberNotFound
// - ExistsByXXX() 用於檢查重複性（效能優化，只執行 COUNT）
//
// Application Layer 通過此接口操作會員資料
// Infrastructure Layer 提供 GORM 實現
type MemberRepository interface {
	// Save 保存會員（新增或更新）
	//
	// 參數：
	// - ctx: 事務上下文（必須 non-nil，寫操作需要事務）
	// - member: 會員聚合
	//
	// 返回：
	// - error: 保存失敗時返回錯誤
	//
	// 業務規則：
	// - 如果 MemberID 已存在，執行更新
	// - 如果 MemberID 不存在，執行新增
	// - PhoneNumber 唯一性由資料庫約束保證
	Save(ctx shared.TransactionContext, member *Member) error

	// FindByMemberID 根據會員 ID 查找會員
	//
	// 參數：
	// - ctx: 事務上下文（可為 nil，讀操作可選事務參與）
	// - id: 會員 ID
	//
	// 返回：
	// - *Member: 找到的會員聚合
	// - error: 找不到時返回 ErrMemberNotFound
	FindByMemberID(ctx shared.TransactionContext, id MemberID) (*Member, error)

	// FindByLineUserID 根據 LINE UserID 查找會員
	//
	// 參數：
	// - ctx: 事務上下文（可為 nil，讀操作可選事務參與）
	// - lineUserID: LINE Platform 用戶 ID
	//
	// 返回：
	// - *Member: 找到的會員聚合
	// - error: 找不到時返回 ErrMemberNotFound
	//
	// 使用場景：
	// - LINE Webhook 接收事件時查找會員
	// - 防止同一個 LINE 帳號重複註冊
	FindByLineUserID(ctx shared.TransactionContext, lineUserID LineUserID) (*Member, error)

	// ExistsByPhoneNumber 檢查手機號碼是否已被註冊
	//
	// 參數：
	// - ctx: 事務上下文（可為 nil）
	// - phoneNumber: 手機號碼
	//
	// 返回：
	// - bool: true 表示已存在，false 表示未註冊
	// - error: 查詢失敗時返回錯誤
	//
	// 使用場景：
	// - 註冊前檢查手機號碼是否重複
	// - 效能優化：比 Find 更輕量（只需 COUNT）
	ExistsByPhoneNumber(ctx shared.TransactionContext, phoneNumber PhoneNumber) (bool, error)

	// ExistsByLineUserID 檢查 LINE UserID 是否已註冊
	//
	// 參數：
	// - ctx: 事務上下文（可為 nil）
	// - lineUserID: LINE UserID
	//
	// 返回：
	// - bool: true 表示已存在，false 表示未註冊
	// - error: 查詢失敗時返回錯誤
	//
	// 使用場景：
	// - 註冊前檢查是否重複註冊
	// - 效能優化：比 Find 更輕量
	ExistsByLineUserID(ctx shared.TransactionContext, lineUserID LineUserID) (bool, error)
}
