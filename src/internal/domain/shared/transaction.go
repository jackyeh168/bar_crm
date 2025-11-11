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
// 這是一個標記介面，Infrastructure Layer 會實作具體的事務封裝
type TransactionContext interface {
	// 標記介面：僅用於傳遞上下文，不暴露方法
}

// TransactionManager 事務管理器介面
type TransactionManager interface {
	InTransaction(fn func(ctx TransactionContext) error) error
}
