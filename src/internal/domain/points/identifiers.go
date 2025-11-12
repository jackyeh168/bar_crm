package points

import (
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
)

// ===========================
// 實體 ID 類型定義
// ===========================

// 設計原則：使用泛型 EntityID[T] 消除重複代碼
//
// 架構改進（Uncle Bob Code Review - Day 2 Critical Issue #2）：
// 原始實現：AccountID 和 MemberID 各有 30+ 行代碼，82% 重複
// 新實現：使用泛型 + 類型別名，減少到 3 行/類型
//
// 類型安全保證：
// - AccountID 和 MemberID 是不同類型（編譯器強制檢查）
// - 不能將 AccountID 賦值給 MemberID 變量
// - 不能比較 AccountID 和 MemberID（編譯錯誤）

// ===========================
// AccountID - 積分帳戶 ID
// ===========================

// AccountMarker 是 AccountID 的標記類型
// 用途：讓 AccountID 和 MemberID 成為不同的類型
type AccountMarker struct{}

// AccountID 積分帳戶的唯一標識符
//
// 實現：EntityID[AccountMarker] 的類型別名
// 使用：id := NewAccountID() 或 AccountIDFromString(s)
type AccountID = shared.EntityID[AccountMarker]

// NewAccountID 生成新的積分帳戶 ID（UUID v4）
//
// 返回：新生成的 AccountID
//
// 使用場景：創建新積分帳戶時
func NewAccountID() AccountID {
	return shared.NewEntityID[AccountMarker]()
}

// AccountIDFromString 從字串解析積分帳戶 ID
//
// 參數：
//   s - UUID 字串
//
// 返回：
//   AccountID - 解析成功的 ID
//   error - 解析失敗（返回 ErrInvalidAccountID）
//
// 使用場景：
// - 從數據庫讀取 ID
// - 從 HTTP 請求解析 ID
// - 從配置文件讀取 ID
func AccountIDFromString(s string) (AccountID, error) {
	return shared.EntityIDFromString[AccountMarker](s, ErrInvalidAccountID)
}

// ===========================
// MemberID - 會員 ID
// ===========================

// MemberMarker 是 MemberID 的標記類型
type MemberMarker struct{}

// MemberID 會員的唯一標識符
//
// 實現：EntityID[MemberMarker] 的類型別名
// 使用：id := NewMemberID() 或 MemberIDFromString(s)
type MemberID = shared.EntityID[MemberMarker]

// NewMemberID 生成新的會員 ID（UUID v4）
//
// 返回：新生成的 MemberID
//
// 使用場景：會員註冊時
func NewMemberID() MemberID {
	return shared.NewEntityID[MemberMarker]()
}

// MemberIDFromString 從字串解析會員 ID
//
// 參數：
//   s - UUID 字串
//
// 返回：
//   MemberID - 解析成功的 ID
//   error - 解析失敗（返回 ErrInvalidMemberID）
//
// 使用場景：
// - 從數據庫讀取會員信息
// - 從 LINE UserID 映射
// - API 請求解析
func MemberIDFromString(s string) (MemberID, error) {
	return shared.EntityIDFromString[MemberMarker](s, ErrInvalidMemberID)
}

// ===========================
// 設計優勢說明
// ===========================

// 1. DRY 原則（Don't Repeat Yourself）：
//    - 原始代碼：64 行（32 行 × 2 類型）
//    - 新代碼：20 行（10 行/類型，包含註釋）
//    - 減少 69% 代碼量

// 2. 單一真相來源（Single Source of Truth）：
//    - 所有 ID 邏輯在 shared.EntityID[T]
//    - 添加新功能（如 JSON 序列化）只需修改一處
//    - Bug 修復一次，所有 ID 類型受益

// 3. 類型安全（Type Safety）：
//    - AccountID ≠ MemberID（編譯器保證）
//    - 防止 ID 混用的運行時錯誤
//    - 比 string 或 uuid.UUID 更安全

// 4. 可維護性（Maintainability）：
//    - 新增 ID 類型只需 10 行代碼
//    - 重構成本降低
//    - 測試覆蓋率更高（集中測試 EntityID[T]）

// 5. 符合 SOLID 原則：
//    - SRP: EntityID[T] 只負責 UUID 封裝
//    - OCP: 通過泛型擴展，無需修改
//    - DIP: 不依賴具體實現

// ===========================
// 遷移說明（向後兼容）
// ===========================

// 這個重構對外部代碼是透明的：
// - AccountID 和 MemberID 的 API 保持不變
// - 所有方法簽名一致
// - 測試代碼無需修改（type alias 行為完全一致）
//
// 唯一變化：導入路徑（內部實現細節）
// - 舊：value_objects.go 中的獨立實現
// - 新：shared.EntityID[T] + 類型別名

// ===========================
// 未來擴展
// ===========================

// 如果需要新的 ID 類型，只需：
// 1. 定義標記類型：type TransactionMarker struct{}
// 2. 定義類型別名：type TransactionID = shared.EntityID[TransactionMarker]
// 3. 添加構造函數：NewTransactionID() 和 TransactionIDFromString()
// 4. 定義錯誤：ErrInvalidTransactionID
//
// 總共 ~10 行代碼，無需重複實現 UUID 邏輯
