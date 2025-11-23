package shared

import (
	"time"

	"github.com/shopspring/decimal"
)

// PointsCalculableTransaction 可計算積分的交易介面
// 設計原則：介面隔離，只暴露積分計算所需的方法
type PointsCalculableTransaction interface {
	GetTransactionAmount() decimal.Decimal
	GetTransactionDate() time.Time
}

// TransactionContext 事務上下文介面
//
// 設計決策：可選事務參與模式（Optional Transaction Participation）
//
// 行為約定：
// - ctx != nil: 在調用者的事務中執行（事務傳播）
// - ctx == nil: 使用 auto-commit 模式（適用於單一讀操作）
//
// 使用場景：
//
// 1. 寫操作：必須在事務中（通過 TransactionManager.InTransaction）
//    - 保證原子性（Atomicity）
//    - 支援回滾（Rollback on error）
//    - 例如：創建帳戶、賺取積分、使用積分
//
// 2. 讀操作：可選事務參與
//    - 獨立查詢：傳入 nil（性能優先，auto-commit 模式）
//    - 在事務中讀取：傳入調用者的 ctx（保證一致性）
//    - 例如：查詢餘額（獨立）vs 轉帳前查詢餘額（在事務中）
//
// Repository 方法約束指南：
//
// ✅ ctx 必須為 non-nil（寫操作需要事務保證）：
//    - Save()   - 創建新記錄
//    - Update() - 更新現有記錄
//    - Delete() - 刪除記錄
//
// ✅ ctx 可為 nil（讀操作可選事務參與）：
//    - FindByID()       - 根據 ID 查詢
//    - FindByMemberID() - 根據 MemberID 查詢
//    - FindAll()        - 批次查詢
//
// 原則：修改狀態的操作必須在事務中，查詢操作可選擇是否參與事務
//
// 範例：
//
// 寫操作（必須在事務中）：
//   txManager.InTransaction(func(ctx TransactionContext) error {
//       account, _ := repo.FindByID(ctx, accountID)
//       account.EarnPoints(amount, source, sourceID, description)
//       return repo.Update(ctx, account)  // ctx != nil
//   })
//
// 讀操作（獨立查詢，不需要事務）：
//   balance, _ := repo.FindByMemberID(nil, memberID)  // ctx == nil, auto-commit
//
// 讀操作（在事務中，保證一致性）：
//   txManager.InTransaction(func(ctx TransactionContext) error {
//       account1, _ := repo.FindByID(ctx, accountID1)  // ctx != nil
//       account2, _ := repo.FindByID(ctx, accountID2)  // ctx != nil
//       // 兩次查詢在同一事務中，保證一致性
//       return processTransfer(account1, account2)
//   })
//
// 架構原則：
// - 這是一個標記介面（Marker Interface），不暴露任何方法
// - Infrastructure Layer 負責實作具體的事務封裝（如 GORM, SQL）
// - Domain Layer 和 Application Layer 只依賴此介面，不依賴具體實作
// - 保持依賴方向：Infrastructure → Domain（依賴倒置原則）
type TransactionContext interface {
	// 標記介面：僅用於傳遞上下文，不暴露方法
}

// TransactionManager 事務管理器介面
type TransactionManager interface {
	InTransaction(fn func(ctx TransactionContext) error) error
}
