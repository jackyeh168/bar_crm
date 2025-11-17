package points

import (
	"time"

	"github.com/google/uuid"
)

// ===========================
// PointsAccount 領域事件
// ===========================

// PointsAccountCreatedEvent 積分帳戶創建事件
type PointsAccountCreatedEvent struct {
	eventID     string
	accountID   AccountID
	memberID    MemberID
	occurredAt  time.Time
}

// NewPointsAccountCreatedEvent 創建帳戶創建事件
func NewPointsAccountCreatedEvent(accountID AccountID, memberID MemberID) *PointsAccountCreatedEvent {
	return &PointsAccountCreatedEvent{
		eventID:    uuid.New().String(),
		accountID:  accountID,
		memberID:   memberID,
		occurredAt: time.Now(),
	}
}

// EventID 實現 DomainEvent 介面
func (e *PointsAccountCreatedEvent) EventID() string {
	return e.eventID
}

// EventType 實現 DomainEvent 介面
func (e *PointsAccountCreatedEvent) EventType() string {
	return "points.account_created"
}

// OccurredAt 實現 DomainEvent 介面
func (e *PointsAccountCreatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID 實現 DomainEvent 介面
func (e *PointsAccountCreatedEvent) AggregateID() string {
	return e.accountID.String()
}

// AccountID 獲取帳戶 ID
func (e *PointsAccountCreatedEvent) AccountID() AccountID {
	return e.accountID
}

// MemberID 獲取會員 ID
func (e *PointsAccountCreatedEvent) MemberID() MemberID {
	return e.memberID
}

// ===========================
// PointsEarned 領域事件
// ===========================

// PointsEarnedEvent 積分已獲得事件
type PointsEarnedEvent struct {
	eventID     string
	accountID   AccountID
	amount      PointsAmount
	source      PointsSource
	sourceID    string
	description string
	occurredAt  time.Time
}

// NewPointsEarnedEvent 創建積分已獲得事件
func NewPointsEarnedEvent(
	accountID AccountID,
	amount PointsAmount,
	source PointsSource,
	sourceID string,
	description string,
) *PointsEarnedEvent {
	return &PointsEarnedEvent{
		eventID:     uuid.New().String(),
		accountID:   accountID,
		amount:      amount,
		source:      source,
		sourceID:    sourceID,
		description: description,
		occurredAt:  time.Now(),
	}
}

// EventID 實現 DomainEvent 介面
func (e *PointsEarnedEvent) EventID() string {
	return e.eventID
}

// EventType 實現 DomainEvent 介面
func (e *PointsEarnedEvent) EventType() string {
	return "points.earned"
}

// OccurredAt 實現 DomainEvent 介面
func (e *PointsEarnedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID 實現 DomainEvent 介面
func (e *PointsEarnedEvent) AggregateID() string {
	return e.accountID.String()
}

// AccountID 獲取帳戶 ID
func (e *PointsEarnedEvent) AccountID() AccountID {
	return e.accountID
}

// Amount 獲取積分數量
func (e *PointsEarnedEvent) Amount() PointsAmount {
	return e.amount
}

// Source 獲取積分來源
func (e *PointsEarnedEvent) Source() PointsSource {
	return e.source
}

// SourceID 獲取來源 ID
func (e *PointsEarnedEvent) SourceID() string {
	return e.sourceID
}

// Description 獲取描述
func (e *PointsEarnedEvent) Description() string {
	return e.description
}

// ===========================
// PointsDeducted 領域事件
// ===========================

// PointsDeductedEvent 積分已扣減事件
type PointsDeductedEvent struct {
	eventID    string
	accountID  AccountID
	amount     PointsAmount
	reason     string
	occurredAt time.Time
}

// NewPointsDeductedEvent 創建積分已扣減事件
func NewPointsDeductedEvent(
	accountID AccountID,
	amount PointsAmount,
	reason string,
) *PointsDeductedEvent {
	return &PointsDeductedEvent{
		eventID:    uuid.New().String(),
		accountID:  accountID,
		amount:     amount,
		reason:     reason,
		occurredAt: time.Now(),
	}
}

// EventID 實現 DomainEvent 介面
func (e *PointsDeductedEvent) EventID() string {
	return e.eventID
}

// EventType 實現 DomainEvent 介面
func (e *PointsDeductedEvent) EventType() string {
	return "points.deducted"
}

// OccurredAt 實現 DomainEvent 介面
func (e *PointsDeductedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID 實現 DomainEvent 介面
func (e *PointsDeductedEvent) AggregateID() string {
	return e.accountID.String()
}

// AccountID 獲取帳戶 ID
func (e *PointsDeductedEvent) AccountID() AccountID {
	return e.accountID
}

// Amount 獲取積分數量
func (e *PointsDeductedEvent) Amount() PointsAmount {
	return e.amount
}

// Reason 獲取扣減原因
func (e *PointsDeductedEvent) Reason() string {
	return e.reason
}

// ===========================
// PointsRecalculated 領域事件
// ===========================

// PointsRecalculatedEvent 積分已重算事件
// 審計增強：包含完整的重算上下文信息
type PointsRecalculatedEvent struct {
	eventID        string
	accountID      AccountID
	oldPoints      int
	newPoints      int
	reason         string // 重算原因（如："rule_change"、"data_correction"、"migration"）
	conversionRate int    // 使用的轉換率（TWD per point）
	triggeredBy    string // 觸發者（管理員 ID，可選）
	occurredAt     time.Time
}

// NewPointsRecalculatedEvent 創建積分已重算事件
// 參數：
//   - accountID: 帳戶 ID
//   - oldPoints: 重算前積分
//   - newPoints: 重算後積分
//   - reason: 重算原因（業務上下文）
//   - conversionRate: 使用的轉換率
//   - triggeredBy: 觸發者 ID（可為空字串）
func NewPointsRecalculatedEvent(
	accountID AccountID,
	oldPoints int,
	newPoints int,
	reason string,
	conversionRate int,
	triggeredBy string,
) *PointsRecalculatedEvent {
	return &PointsRecalculatedEvent{
		eventID:        uuid.New().String(),
		accountID:      accountID,
		oldPoints:      oldPoints,
		newPoints:      newPoints,
		reason:         reason,
		conversionRate: conversionRate,
		triggeredBy:    triggeredBy,
		occurredAt:     time.Now(),
	}
}

// EventID 實現 DomainEvent 介面
func (e *PointsRecalculatedEvent) EventID() string {
	return e.eventID
}

// EventType 實現 DomainEvent 介面
func (e *PointsRecalculatedEvent) EventType() string {
	return "points.recalculated"
}

// OccurredAt 實現 DomainEvent 介面
func (e *PointsRecalculatedEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID 實現 DomainEvent 介面
func (e *PointsRecalculatedEvent) AggregateID() string {
	return e.accountID.String()
}

// AccountID 獲取帳戶 ID
func (e *PointsRecalculatedEvent) AccountID() AccountID {
	return e.accountID
}

// OldPoints 獲取舊積分值
func (e *PointsRecalculatedEvent) OldPoints() int {
	return e.oldPoints
}

// NewPoints 獲取新積分值
func (e *PointsRecalculatedEvent) NewPoints() int {
	return e.newPoints
}

// Reason 獲取重算原因
func (e *PointsRecalculatedEvent) Reason() string {
	return e.reason
}

// ConversionRate 獲取使用的轉換率
func (e *PointsRecalculatedEvent) ConversionRate() int {
	return e.conversionRate
}

// TriggeredBy 獲取觸發者 ID
func (e *PointsRecalculatedEvent) TriggeredBy() string {
	return e.triggeredBy
}
