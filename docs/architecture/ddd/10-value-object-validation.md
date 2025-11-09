# Value Object 驗證規則

> **版本**: 1.0
> **最後更新**: 2025-01-08

本章節定義所有 Value Object 的構造方法、驗證規則、正規化策略和錯誤訊息。

---

## **11.1 會員管理 Value Objects**

### **PhoneNumber (手機號碼)**

**構造方法**:
```go
NewPhoneNumber(raw string) (PhoneNumber, error)
```

**驗證規則**:
1. **正規化**: 移除所有空格、破折號、括號
   - 輸入: `"0912-345-678"`, `"0912 345 678"`, `"(0912) 345-678"`
   - 正規化: `"0912345678"`
2. **長度檢查**: 必須恰好 10 位數字
3. **前綴檢查**: 必須以 `"09"` 開頭
4. **字符檢查**: 只能包含數字 `[0-9]`

**儲存格式**: 正規化後的字串 (例: `"0912345678"`)

**錯誤處理**:
```go
// 錯誤: ErrInvalidPhoneNumberFormat
// 訊息: "手機號碼必須為 10 位數字，以 09 開頭"

// 範例:
"123456789"     → ErrInvalidPhoneNumberFormat (不足 10 位)
"0812345678"    → ErrInvalidPhoneNumberFormat (不是 09 開頭)
"091234567a"    → ErrInvalidPhoneNumberFormat (包含非數字)
"0912-345-678"  → OK (正規化為 "0912345678")
```

**相等性檢查**: 比較正規化後的字串

---

### **LineUserID (LINE 用戶 ID)**

**構造方法**:
```go
NewLineUserID(raw string) (LineUserID, error)
```

**驗證規則**:
1. **非空檢查**: 不能為空字串
2. **前綴檢查**: 必須以 `"U"` 開頭
3. **長度檢查**: 至少 2 個字符

**儲存格式**: 原始字串（不正規化）

**錯誤處理**:
```go
// 錯誤: ErrInvalidLineUserID
// 訊息: "LineUserID 必須以 'U' 開頭"

// 範例:
""           → ErrInvalidLineUserID (空字串)
"U"          → ErrInvalidLineUserID (長度不足)
"Uxxx"       → OK
"Abc123"     → ErrInvalidLineUserID (不是 U 開頭)
```

---

## **11.2 積分管理 Value Objects**

### **PointsAmount (積分數量)**

**構造方法**:
```go
NewPointsAmount(value int) (PointsAmount, error)
```

**驗證規則**:
1. **非負檢查**: 必須 >= 0
2. **整數類型**: 使用 `int` 類型

**儲存格式**: 整數

**錯誤處理**:
```go
// 錯誤: ErrNegativePointsAmount
// 訊息: "積分數量不能為負數"

// 範例:
-1   → ErrNegativePointsAmount
0    → OK
100  → OK
```

**算術運算**:
```go
func (p PointsAmount) Add(other PointsAmount) PointsAmount
func (p PointsAmount) Subtract(other PointsAmount) (PointsAmount, error)
  // 返回 ErrNegativePointsAmount 如果結果為負
```

---

### **ConversionRate (轉換率)**

**構造方法**:
```go
NewConversionRate(value int) (ConversionRate, error)
```

**驗證規則**:
1. **範圍檢查**: 必須在 1-1000 之間（含邊界）
2. **整數類型**: 使用 `int` 類型

**業務含義**: 每消費多少元獲得 1 點積分（例：100 元 = 1 點）

**儲存格式**: 整數

**錯誤處理**:
```go
// 錯誤: ErrInvalidConversionRate
// 訊息: "轉換率必須在 1-1000 之間"

// 範例:
0      → ErrInvalidConversionRate (低於最小值)
1      → OK
100    → OK
1000   → OK
1001   → ErrInvalidConversionRate (超過最大值)
```

---

### **DateRange (日期範圍)**

**構造方法**:
```go
NewDateRange(startDate, endDate time.Time) (DateRange, error)
```

**驗證規則**:
1. **順序檢查**: `startDate` <= `endDate`
2. **非零檢查**: 兩個日期都不能為零值
3. **重疊檢查**: 由 Domain Service 負責（需要查詢資料庫）

**儲存格式**: 兩個 `DATE` 類型欄位

**錯誤處理**:
```go
// 錯誤: ErrInvalidDateRange
// 訊息: "結束日期必須在開始日期之後或相同"

// 範例:
StartDate: 2025-01-01, EndDate: 2025-12-31  → OK
StartDate: 2025-12-31, EndDate: 2025-01-01  → ErrInvalidDateRange
StartDate: 2025-01-01, EndDate: 2025-01-01  → OK (單日規則)
```

**查詢方法**:
```go
func (dr DateRange) Contains(date time.Time) bool
  // 返回: startDate <= date <= endDate

func (dr DateRange) Overlaps(other DateRange) bool
  // 返回: 是否有重疊
```

---

## **11.3 發票處理 Value Objects**

### **InvoiceNumber (發票號碼)**

**構造方法**:
```go
NewInvoiceNumber(raw string) (InvoiceNumber, error)
```

**驗證規則**:
1. **正規化**: 轉換為大寫，移除空格
   - 輸入: `"ab12345678"`, `"AB 12345678"`
   - 正規化: `"AB12345678"`
2. **長度檢查**: 必須恰好 10 個字符
3. **格式檢查**: 前兩位為英文字母，後八位為數字
   - 正則表達式: `^[A-Z]{2}[0-9]{8}$`

**儲存格式**: 大寫正規化字串 (例: `"AB12345678"`)

**錯誤處理**:
```go
// 錯誤: ErrInvalidInvoiceNumber
// 訊息: "發票號碼必須為 10 位英數字 (兩位英文 + 八位數字)"

// 範例:
"ab12345678"     → OK (正規化為 "AB12345678")
"AB 12345678"    → OK (正規化為 "AB12345678")
"AB123456"       → ErrInvalidInvoiceNumber (長度不足)
"1212345678"     → ErrInvalidInvoiceNumber (前兩位不是英文)
"ABCD123456"     → ErrInvalidInvoiceNumber (長度錯誤)
```

---

### **InvoiceDate (發票日期)**

**構造方法**:
```go
NewInvoiceDate(rocDate string) (InvoiceDate, error)
```

**輸入格式**: ROC 民國年格式 `yyyMMdd` (例: `"1140108"` 表示民國 114 年 1 月 8 日)

**驗證規則**:
1. **長度檢查**: 必須為 7 位數字
2. **格式檢查**: 純數字字串
3. **日期有效性**: 解析後必須是有效日期
4. **年份檢查**: 民國年必須在合理範圍內（例: 80-200）

**轉換**: ROC 年 → 西元年（年份 + 1911）
- `"1140108"` → `2025-01-08`

**儲存格式**: `DATE` 類型（西元年）

**錯誤處理**:
```go
// 錯誤: ErrInvalidInvoiceDate
// 訊息: "發票日期格式錯誤，必須為 yyyMMdd (民國年)"

// 範例:
"1140108"  → OK (民國 114 年 1 月 8 日 → 2025-01-08)
"114010"   → ErrInvalidInvoiceDate (長度不足)
"1140132"  → ErrInvalidInvoiceDate (1 月無 32 日)
"114ab08"  → ErrInvalidInvoiceDate (包含非數字)
```

**有效期檢查**:
```go
func (id InvoiceDate) IsExpired() bool
  // 返回: 是否超過 60 天
  // 計算: today - invoiceDate > 60 days
```

---

### **Money (金額)**

**構造方法**:
```go
NewMoney(value decimal.Decimal) (Money, error)
// 或
NewMoneyFromInt(cents int) (Money, error)  // 以「分」為單位
```

**驗證規則**:
1. **正數檢查**: 必須 > 0
2. **精度**: 支持兩位小數（元.角分）
3. **類型**: 使用 `decimal.Decimal` 避免浮點誤差

**儲存格式**: `DECIMAL(10, 2)` 或以「分」為單位的整數

**錯誤處理**:
```go
// 錯誤: ErrInvalidAmount
// 訊息: "金額必須大於 0"

// 範例:
0.00    → ErrInvalidAmount
-10.50  → ErrInvalidAmount
10.50   → OK
100     → OK
```

**算術運算**:
```go
func (m Money) Add(other Money) Money
func (m Money) Subtract(other Money) (Money, error)
func (m Money) Divide(divisor int) Money  // 用於積分計算
```
