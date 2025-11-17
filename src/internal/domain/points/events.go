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
