package member

import (
	"time"
)

// ===========================
// Member Aggregate Root
// ===========================

// Member 會員聚合根
//
// 聚合邊界：
// - 會員基本信息（ID, LINE UserID, DisplayName）
// - 手機號碼綁定（PhoneNumber）
// - 註冊狀態（CreatedAt, UpdatedAt）
//
// 不變量（Invariants）：
// 1. 會員必須有 LINE UserID（註冊來源）
// 2. 會員必須有顯示名稱
// 3. 手機號碼綁定後不可變更（業務規則：需管理員介入）
// 4. CreatedAt 不可變更
// 5. UpdatedAt 在每次狀態變更時更新
//
// 設計原則：
// - Tell, Don't Ask：通過方法封裝行為，而非暴露狀態
// - 聚合內一致性：所有狀態變更通過方法執行
// - 不可變性：所有欄位為 unexported
//
// 使用範例：
//   member, err := NewMember(lineUserID, displayName)
//   member.BindPhoneNumber(phoneNumber)
type Member struct {
	// 識別欄位
	memberID    MemberID
	lineUserID  LineUserID
	displayName string

	// 綁定信息
	phoneNumber PhoneNumber

	// 審計欄位
	createdAt time.Time
	updatedAt time.Time
	version   int // 樂觀鎖版本號（Optimistic Locking）
}

// NewMember 創建新會員（Checked Constructor）
//
// 參數：
// - lineUserID: LINE Platform 用戶 ID
// - displayName: LINE 顯示名稱
//
// 返回：
// - Member: 新創建的會員聚合
// - error: 驗證失敗時返回錯誤
//
// 業務規則：
// 1. LINE UserID 必須有效（已在 LineUserID VO 中驗證）
// 2. DisplayName 不能為空
// 3. 自動生成 MemberID（UUID）
// 4. 初始狀態：未綁定手機號碼
// 5. 設定 CreatedAt 和 UpdatedAt 為當前時間
//
// 錯誤範例：
// - displayName == "" → 錯誤（顯示名稱不能為空）
func NewMember(lineUserID LineUserID, displayName string) (*Member, error) {
	// 1. 驗證 DisplayName
	if displayName == "" {
		return nil, ErrInvalidDisplayName
	}

	// 2. 生成 MemberID
	memberID := NewMemberID()

	// 3. 設定時間戳
	now := time.Now()

	// 4. 創建聚合
	member := &Member{
		memberID:    memberID,
		lineUserID:  lineUserID,
		displayName: displayName,
		phoneNumber: PhoneNumber{}, // 零值，表示未綁定
		createdAt:   now,
		updatedAt:   now,
		version:     1, // 初始版本為 1
	}

	return member, nil
}

// ReconstructMember 重建會員聚合（用於從資料庫載入）
//
// 參數：
// - memberID: 會員 ID
// - lineUserID: LINE UserID
// - displayName: 顯示名稱
// - phoneNumber: 手機號碼（可能為零值）
// - createdAt: 創建時間
// - updatedAt: 更新時間
// - version: 樂觀鎖版本號
//
// 返回：
// - Member: 重建的會員聚合
// - error: 驗證失敗時返回錯誤
//
// 使用場景：
// - Repository 從資料庫載入會員
// - 不執行業務規則驗證（假設資料庫中的數據已驗證）
func ReconstructMember(
	memberID MemberID,
	lineUserID LineUserID,
	displayName string,
	phoneNumber PhoneNumber,
	createdAt time.Time,
	updatedAt time.Time,
	version int,
) (*Member, error) {
	// 基本驗證
	if displayName == "" {
		return nil, ErrInvalidDisplayName
	}

	return &Member{
		memberID:    memberID,
		lineUserID:  lineUserID,
		displayName: displayName,
		phoneNumber: phoneNumber,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}, nil
}

// ===========================
// Member Aggregate Behavior Methods
// ===========================

// BindPhoneNumber 綁定手機號碼
//
// 參數：
// - phoneNumber: 要綁定的手機號碼
//
// 業務規則：
// 1. 手機號碼必須有效（已在 PhoneNumber VO 中驗證）
// 2. 如果已綁定手機號碼，不允許修改（業務規則）
// 3. 綁定成功後更新 UpdatedAt
//
// 返回：
// - error: 如果已綁定手機號碼，返回錯誤
//
// 使用場景：
// - 會員首次綁定手機號碼
// - 註冊流程中的必要步驟
func (m *Member) BindPhoneNumber(phoneNumber PhoneNumber) error {
	// 1. 檢查是否已綁定
	if !m.phoneNumber.IsZero() {
		return ErrPhoneAlreadyBound.WithContext(
			"current_phone", m.phoneNumber.String(),
			"new_phone", phoneNumber.String(),
		)
	}

	// 2. 綁定手機號碼
	m.phoneNumber = phoneNumber

	// 3. 更新時間戳和版本號
	m.updatedAt = time.Now()
	m.version++

	return nil
}

// ===========================
// Member Aggregate Getters
// ===========================

// MemberID 返回會員 ID
func (m *Member) MemberID() MemberID {
	return m.memberID
}

// LineUserID 返回 LINE UserID
func (m *Member) LineUserID() LineUserID {
	return m.lineUserID
}

// DisplayName 返回顯示名稱
func (m *Member) DisplayName() string {
	return m.displayName
}

// PhoneNumber 返回手機號碼
func (m *Member) PhoneNumber() PhoneNumber {
	return m.phoneNumber
}

// HasPhoneNumber 檢查是否已綁定手機號碼
func (m *Member) HasPhoneNumber() bool {
	return !m.phoneNumber.IsZero()
}

// CreatedAt 返回創建時間
func (m *Member) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt 返回更新時間
func (m *Member) UpdatedAt() time.Time {
	return m.updatedAt
}

// Version 返回版本號（用於樂觀鎖）
func (m *Member) Version() int {
	return m.version
}
