package shared

import "time"

// DomainEvent 領域事件基礎介面
type DomainEvent interface {
	EventID() string        // 事件唯一標識
	EventType() string      // 事件類型
	OccurredAt() time.Time  // 發生時間
	AggregateID() string    // 聚合根 ID
}

// EventPublisher 事件發布器介面
// 設計原則：介面定義在 Domain Layer（使用者），由 Infrastructure 實作
type EventPublisher interface {
	Publish(event DomainEvent) error
	PublishBatch(events []DomainEvent) error
}

// EventSubscriber 事件訂閱器介面
type EventSubscriber interface {
	Subscribe(eventType string, handler EventHandler) error
}

// EventHandler 事件處理器介面
type EventHandler interface {
	Handle(event DomainEvent) error
	EventType() string
}
