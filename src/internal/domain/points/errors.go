package points

import "fmt"

// ErrNegativePointsAmount 積分數量不可為負數
var ErrNegativePointsAmount = fmt.Errorf("points amount cannot be negative")

// ErrInvalidPointsAmount 無效的積分數量
var ErrInvalidPointsAmount = fmt.Errorf("invalid points amount")

// ErrInsufficientPoints 積分不足
var ErrInsufficientPoints = fmt.Errorf("insufficient points")

// ErrInvalidAccountID 無效的帳戶 ID
var ErrInvalidAccountID = fmt.Errorf("invalid account ID")

// ErrInvalidMemberID 無效的會員 ID
var ErrInvalidMemberID = fmt.Errorf("invalid member ID")

// ErrInvalidConversionRate 無效的轉換率
var ErrInvalidConversionRate = fmt.Errorf("invalid conversion rate")
