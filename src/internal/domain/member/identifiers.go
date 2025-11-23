package member

import (
	"github.com/jackyeh168/bar_crm/src/internal/domain/shared"
)

// ===========================
// MemberID Value Object (Generic Pattern)
// ===========================

// MemberMarker 會員 ID 標記類型
//
// 設計原則：
// - 用於泛型類型區分（MemberID ≠ AccountID）
// - 空結構體，不佔用記憶體
// - 僅用於編譯時類型檢查
type MemberMarker struct{}

// MemberID 會員 ID 值對象（基於泛型 EntityID）
//
// 設計原則：
// - 使用 shared.EntityID[T] 泛型實現（DRY 原則）
// - 類型安全：MemberID ≠ AccountID（編譯時檢查）
// - 不可變性：繼承自 EntityID[T]
// - UUID v4 實現
//
// 優勢：
// - 消除代碼重複（從 94 行減少到 ~20 行）
// - 與其他實體 ID 共享測試和維護
// - 類型安全（不同實體的 ID 不能混用）
//
// 使用範例：
//   memberID := NewMemberID()                 // 生成新ID
//   memberID, err := MemberIDFromString(str)  // 從字串解析
type MemberID = shared.EntityID[MemberMarker]

// NewMemberID 生成新的會員 ID（Unchecked Constructor）
//
// 返回：新生成的會員 ID (UUID v4)
//
// 使用場景：
// - 創建新會員時
// - 不需要驗證（UUID 生成保證有效性）
func NewMemberID() MemberID {
	return shared.NewEntityID[MemberMarker]()
}

// MemberIDFromString 從字串解析會員 ID（Checked Constructor）
//
// 參數：
// - value: UUID 字串表示（例如：\"550e8400-e29b-41d4-a716-446655440000\"）
//
// 返回：
// - MemberID: 解析成功的會員 ID
// - error: 解析失敗時返回 ErrInvalidMemberID
//
// 驗證規則：
// - 必須是有效的 UUID 格式
// - 支援標準 UUID 字串格式（帶連字號）
//
// 使用場景：
// - 從資料庫載入會員
// - 從 API 請求解析會員 ID
// - 從事件載入會員 ID
func MemberIDFromString(value string) (MemberID, error) {
	return shared.EntityIDFromString[MemberMarker](value, ErrInvalidMemberID)
}
