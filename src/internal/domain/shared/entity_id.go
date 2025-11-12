package shared

import (
	"github.com/google/uuid"
)

// ===========================
// EntityID[T] 泛型實體 ID
// ===========================

// EntityID 是一個泛型實體 ID 值對象
//
// 設計原則：
// 1. 使用 Go 1.18+ 泛型消除重複代碼（DRY 原則）
// 2. 類型安全：不同實體的 ID 不能混用（AccountID ≠ MemberID）
// 3. 不可變性（unexported field）
// 4. 自我驗證（建構函數檢查）
//
// 泛型參數 T：
// - 用於類型區分的標記類型（marker type）
// - 例如：EntityID[AccountMarker] 和 EntityID[MemberMarker] 是不同類型
// - T 不需要有任何方法或字段，只用於編譯時類型檢查
//
// 使用範例：
//   // 定義標記類型
//   type AccountMarker struct{}
//   type AccountID = shared.EntityID[AccountMarker]
//
//   // 使用
//   id := shared.NewEntityID[AccountMarker]()
//   str, _ := shared.EntityIDFromString[AccountMarker]("uuid-string", ErrInvalidAccountID)
type EntityID[T any] struct {
	value uuid.UUID
}

// NewEntityID 生成新的實體 ID（使用 UUID v4）
//
// 泛型參數 T 用於類型區分：
//   accountID := NewEntityID[AccountMarker]()
//   memberID := NewEntityID[MemberMarker]()
//   // accountID 和 memberID 是不同類型，不能混用
func NewEntityID[T any]() EntityID[T] {
	return EntityID[T]{value: uuid.New()}
}

// EntityIDFromString 從字串解析實體 ID
//
// 參數：
//   s - UUID 字串（標準格式：xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx）
//   errType - 解析失敗時返回的錯誤類型（由調用者提供，保持錯誤類型一致性）
//
// 返回：
//   EntityID[T] - 解析成功的實體 ID
//   error - 解析失敗時返回 errType（附帶上下文信息）
//
// 設計決策：為什麼需要 errType 參數？
// - 不同實體的 ID 應該返回不同的錯誤（ErrInvalidAccountID vs ErrInvalidMemberID）
// - 錯誤類型定義在各自的 bounded context（points, membership, etc.）
// - shared 層不應依賴具體業務錯誤，保持通用性
//
// 使用範例：
//   // 在 points 包中
//   id, err := shared.EntityIDFromString[AccountMarker](s, points.ErrInvalidAccountID)
func EntityIDFromString[T any](s string, errTemplate error) (EntityID[T], error) {
	id, err := uuid.Parse(s)
	if err != nil {
		// 使用調用者提供的錯誤模板，並添加上下文
		// 假設錯誤類型支持 WithContext（如 DomainError）
		if domainErr, ok := errTemplate.(interface {
			WithContext(keyValues ...interface{}) error
		}); ok {
			return EntityID[T]{}, domainErr.WithContext(
				"input", s,
				"parse_error", err.Error(),
			)
		}
		// 如果錯誤類型不支持 WithContext，直接返回
		return EntityID[T]{}, errTemplate
	}
	return EntityID[T]{value: id}, nil
}

// String 轉換為字串表示（小寫 UUID）
//
// 返回格式：xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx（小寫）
func (e EntityID[T]) String() string {
	return e.value.String()
}

// Equals 比較兩個 EntityID 是否相等
//
// 注意：只能比較相同類型的 ID
//   accountID1.Equals(accountID2) ✓
//   accountID.Equals(memberID) ✗ 編譯錯誤（類型不匹配）
func (e EntityID[T]) Equals(other EntityID[T]) bool {
	return e.value == other.value
}

// IsEmpty 判斷是否為空 ID（零值）
//
// 空 ID 的場景：
// - 未初始化的結構體字段
// - 解析失敗後的零值返回
func (e EntityID[T]) IsEmpty() bool {
	return e.value == uuid.Nil
}

// ===========================
// 設計原則說明
// ===========================

// 1. DRY (Don't Repeat Yourself)：
//    - 單一 EntityID[T] 實現，所有實體 ID 共享邏輯
//    - 新增 ID 類型只需一行 type alias，無需重複實現

// 2. 類型安全（Type Safety）：
//    - EntityID[AccountMarker] 和 EntityID[MemberMarker] 是不同類型
//    - 編譯器強制類型檢查，防止 ID 混用
//    - 比 string 或 uuid.UUID 更安全

// 3. 單一職責原則（SRP）：
//    - EntityID[T] 只負責 UUID 封裝和驗證
//    - 不包含業務邏輯（如權限檢查）
//    - 業務邏輯在各自的實體/聚合中

// 4. 開放封閉原則（OCP）：
//    - 新增 ID 類型無需修改 EntityID[T]
//    - 通過泛型參數擴展，而非修改實現

// 5. 依賴倒置原則（DIP）：
//    - EntityID[T] 不依賴任何具體業務錯誤
//    - 通過 errTemplate 參數反轉依賴方向
//    - shared 包保持純粹，可被任何 bounded context 使用

// ===========================
// 測試指南
// ===========================

// EntityID[T] 的測試應該：
// 1. 測試泛型類型安全性（編譯時檢查）
// 2. 測試 UUID 生成唯一性
// 3. 測試字串解析正確性
// 4. 測試錯誤處理（無效 UUID）
// 5. 測試 Equals 和 IsEmpty 語義

// 各業務包的 ID 測試應該：
// 1. 測試類型別名正確性（AccountID 行為符合預期）
// 2. 測試與具體錯誤類型的集成（ErrInvalidAccountID）
// 3. 不需要重複測試 EntityID[T] 的基礎功能

// ===========================
// 未來擴展點
// ===========================

// 如果需要支持更多功能，可以添加：
// - MarshalJSON / UnmarshalJSON（REST API 序列化）
// - Scan / Value（GORM 數據庫映射）
// - Validate（額外的業務驗證）
//
// 但要保持謹慎：不要讓 shared 包變成"萬能工具包"
// 只添加所有 ID 都需要的通用功能
