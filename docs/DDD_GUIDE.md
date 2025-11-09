# Domain-Driven Design (DDD) 完整指南

**版本**: 1.0
**最後更新**: 2025-01-08
**目標受眾**: 後端開發工程師、架構設計師、技術主管

> **文件目的**: 本文檔提供 Domain-Driven Design 的完整介紹，包含核心概念、設計方法、實踐指南和 Go 語言實現範例。

---

## 目錄

1. [DDD 是什麼？](#1-ddd-是什麼)
2. [為什麼需要 DDD？](#2-為什麼需要-ddd)
3. [DDD 核心概念](#3-ddd-核心概念)
4. [戰略設計（Strategic Design）](#4-戰略設計strategic-design)
5. [戰術設計（Tactical Design）](#5-戰術設計tactical-design)
6. [DDD 設計流程](#6-ddd-設計流程)
7. [實踐指南](#7-實踐指南)
8. [常見誤解與陷阱](#8-常見誤解與陷阱)
9. [DDD 與 Clean Architecture](#9-ddd-與-clean-architecture)
10. [Go 語言實現範例](#10-go-語言實現範例)

---

## 1. DDD 是什麼？

### 1.1 定義

**Domain-Driven Design（領域驅動設計）** 是一套軟體開發的思想和方法論，由 Eric Evans 在 2003 年提出。

**核心思想**：
> 軟體的複雜度來自於業務領域（Domain）的複雜度，而不是技術本身。因此，軟體設計應該圍繞業務領域展開，用業務語言建模，讓代碼反映業務邏輯。

### 1.2 DDD 的三大支柱

| 支柱 | 說明 | 實踐方式 |
|------|------|---------|
| **統一語言（Ubiquitous Language）** | 開發團隊和業務專家使用相同的術語 | 代碼中的類名、方法名與業務術語一致 |
| **限界上下文（Bounded Context）** | 明確劃分模型的適用範圍 | 不同業務領域使用獨立的模型 |
| **戰略設計 + 戰術設計** | 從宏觀到微觀的設計方法 | 先劃分業務領域，再設計具體實現 |

### 1.3 DDD 不是什麼

❌ **DDD 不是**：
- 不是框架或庫
- 不是特定的編程語言技術
- 不是銀彈，不能解決所有問題
- 不適合簡單的 CRUD 系統

✅ **DDD 適合**：
- 業務邏輯複雜的系統
- 需要長期維護和演化的系統
- 業務規則頻繁變更的系統
- 團隊規模中大（4+ 人）的項目

---

## 2. 為什麼需要 DDD？

### 2.1 傳統開發的問題

#### 問題 1：業務邏輯散落在各處

```go
// ❌ 傳統做法：業務邏輯在 Controller
func CreateOrder(c *gin.Context) {
    var req OrderRequest
    c.BindJSON(&req)

    // 業務邏輯混在 Controller
    if req.Quantity <= 0 {
        c.JSON(400, "數量必須大於 0")
        return
    }

    totalAmount := req.Price * req.Quantity
    if totalAmount < 100 {
        c.JSON(400, "訂單金額不能小於 100 元")
        return
    }

    // 直接操作數據庫
    db.Create(&Order{...})
}
```

**問題**：
- 業務規則（數量驗證、金額限制）散落在 Controller、Service、Repository
- 難以測試業務邏輯（必須啟動 HTTP 服務）
- 業務規則變更需要改多處代碼

#### 問題 2：模型貧血（Anemic Domain Model）

```go
// ❌ 貧血模型：只有數據，沒有行為
type Order struct {
    ID          int
    Status      string
    TotalAmount int
}

// Service 包含所有業務邏輯
type OrderService struct {}

func (s *OrderService) ConfirmOrder(orderID int) error {
    order := s.repo.FindByID(orderID)
    if order.Status != "draft" {
        return errors.New("只能確認草稿訂單")
    }
    order.Status = "confirmed"
    s.repo.Update(order)
}
```

**問題**：
- Order 只是數據容器，沒有業務行為
- 業務邏輯集中在 Service，違反了面向對象設計
- 難以保證業務規則一致性

#### 問題 3：技術語言 vs 業務語言

```go
// ❌ 代碼使用技術術語
type UserDTO struct {
    ID     int    `json:"id"`
    Field1 string `json:"field1"`  // 業務專家不知道這是什麼
    Field2 int    `json:"field2"`
}

func ProcessData(data UserDTO) {
    // 複雜的業務邏輯，但代碼看不出來在做什麼
    if data.Field2 > 100 {
        // ...
    }
}
```

**問題**：
- 業務專家看不懂代碼
- 開發人員不理解業務含義
- 溝通成本高，容易產生誤解

### 2.2 DDD 如何解決這些問題

#### 解決方案 1：業務邏輯在領域模型

```go
// ✅ DDD 做法：業務邏輯在 Entity
type Order struct {
    ID          int
    Status      OrderStatus
    TotalAmount Money
}

// 業務邏輯封裝在 Entity 內
func (o *Order) Confirm() error {
    if o.Status != StatusDraft {
        return errors.New("只能確認草稿訂單")
    }

    if o.TotalAmount.LessThan(Money{Amount: 100}) {
        return errors.New("訂單金額不能小於 100 元")
    }

    o.Status = StatusConfirmed
    o.ConfirmedAt = time.Now()
    return nil
}
```

**優點**：
- 業務規則集中在 Entity
- 易於測試（不需要數據庫或 HTTP）
- 變更業務規則只需改一處

#### 解決方案 2：統一語言（Ubiquitous Language）

```go
// ✅ 代碼使用業務術語
type MembershipAccount struct {
    LineUserID   string
    DisplayName  string
    EarnedPoints int      // 業務術語：累積賺取積分
    UsedPoints   int      // 業務術語：已使用積分
}

// 方法名使用業務語言
func (m *MembershipAccount) AvailablePoints() int {
    return m.EarnedPoints - m.UsedPoints
}

func (m *MembershipAccount) RedeemReward(rewardCost int) error {
    if !m.HasSufficientPoints(rewardCost) {
        return ErrInsufficientPoints
    }
    m.UsedPoints += rewardCost
    return nil
}
```

**優點**：
- 業務專家能看懂代碼
- 減少溝通成本
- 業務知識直接體現在代碼中

---

## 3. DDD 核心概念

### 3.1 概念地圖

```
DDD
├── 戰略設計（Strategic Design）
│   ├── 領域（Domain）
│   ├── 子領域（Subdomain）
│   │   ├── 核心域（Core Domain）
│   │   ├── 支撐域（Supporting Subdomain）
│   │   └── 通用域（Generic Subdomain）
│   ├── 限界上下文（Bounded Context）
│   └── 上下文映射（Context Mapping）
│
└── 戰術設計（Tactical Design）
    ├── 實體（Entity）
    ├── 值對象（Value Object）
    ├── 聚合（Aggregate）
    │   └── 聚合根（Aggregate Root）
    ├── 領域服務（Domain Service）
    ├── 領域事件（Domain Event）
    ├── 倉儲（Repository）
    └── 工廠（Factory）
```

### 3.2 統一語言（Ubiquitous Language）

**定義**：開發團隊和業務專家共同使用的語言，貫穿整個項目。

**實踐原則**：

| 原則 | 說明 | 例子 |
|------|------|------|
| **一詞一義** | 同一術語在整個項目中只有一個含義 | "會員" = MembershipAccount，不是 User |
| **代碼即文檔** | 代碼中的命名與業務術語一致 | `RedeemReward()` 而不是 `UsePoints()` |
| **避免技術術語** | 不用技術黑話，用業務語言 | `InvoiceDate` 而不是 `Timestamp` |
| **持續演化** | 隨著業務理解加深，語言也會演化 | 發現 "積分" 有多種含義，細化為 "累積積分" 和 "可用積分" |

**例子：電商系統的統一語言**

| 業務術語 | 代碼中的體現 | 錯誤示範 |
|---------|-------------|---------|
| 商品 | `Product` | ~~`Item`~~, ~~`Goods`~~ |
| 下單 | `PlaceOrder()` | ~~`CreateOrder()`~~ |
| 庫存 | `Inventory` | ~~`Stock`~~ |
| 會員等級 | `MembershipTier` | ~~`UserLevel`~~ |

---

## 4. 戰略設計（Strategic Design）

**目標**：從宏觀角度劃分系統邊界，識別核心業務領域。

### 4.1 領域（Domain）與子領域（Subdomain）

#### 4.1.1 定義

| 概念 | 定義 | 例子 |
|------|------|------|
| **領域（Domain）** | 業務活動的範圍 | 電商系統、會員管理系統 |
| **核心域（Core Domain）** | 公司的核心競爭力 | 推薦算法、定價策略 |
| **支撐域（Supporting Subdomain）** | 支持核心業務的領域 | 訂單處理、庫存管理 |
| **通用域（Generic Subdomain）** | 可購買現成方案的領域 | 支付、物流、身份驗證 |

#### 4.1.2 本系統的領域劃分

**餐廳會員管理系統**

```
核心域（Core Domain）：
  ├── 會員積分系統（核心競爭力）
  │   ├── 積分計算規則
  │   ├── 積分使用策略
  │   └── 會員等級制度（未來）

支撐域（Supporting Subdomain）：
  ├── 發票驗證系統
  │   ├── QR Code 解析
  │   └── 發票有效期檢查
  ├── 問卷系統
  │   ├── 問卷設計
  │   └── 回應收集
  └── iChef 發票匯入
      ├── Excel 解析
      └── 發票匹配

通用域（Generic Subdomain）：
  ├── LINE Bot 整合（使用 LINE SDK）
  ├── Google OAuth（使用 Google 服務）
  └── 數據庫存取（使用 GORM）
```

**設計決策指南**：

| 領域類型 | 投資策略 | 技術選擇 |
|---------|---------|---------|
| **核心域** | 自主開發，精心設計 | 使用 DDD 戰術設計 |
| **支撐域** | 簡單實現，快速交付 | 可用 CRUD 架構 |
| **通用域** | 購買或使用開源方案 | 整合第三方服務 |

### 4.2 限界上下文（Bounded Context）

#### 4.2.1 定義

**限界上下文 = 模型的適用邊界**

> 同一個詞在不同的上下文中有不同的含義和職責。

#### 4.2.2 為什麼需要限界上下文？

**例子：「顧客」在不同上下文的含義**

```go
// 銷售上下文（Sales Context）
type Customer struct {
    CustomerID      string
    LoyaltyPoints   int           // 關注：購買行為
    PurchaseHistory []Order
    PreferredProducts []string
}

func (c *Customer) CanGetDiscount() bool {
    return c.LoyaltyPoints > 1000
}

// 技術支援上下文（Support Context）
type Customer struct {
    CustomerID        string
    TicketHistory     []SupportTicket  // 關注：服務問題
    SatisfactionScore float64
    PreferredChannel  string // email, phone, chat
}

func (c *Customer) NeedsFollowUp() bool {
    return c.SatisfactionScore < 3.0
}

// 會計上下文（Accounting Context）
type Customer struct {
    CustomerID     string
    CreditLimit    int               // 關注：財務狀況
    OutstandingBalance int
    PaymentHistory []Payment
}

func (c *Customer) CanPlaceOrder(amount int) bool {
    return c.OutstandingBalance + amount <= c.CreditLimit
}
```

**關鍵洞察**：
- 同樣是 Customer，但在不同上下文中有完全不同的屬性和行為
- 強行合併成一個 Customer 會導致：
  - 類變得臃腫（God Object）
  - 職責不清晰
  - 不同團隊互相干擾

#### 4.2.3 本系統的限界上下文

```
餐廳會員管理系統
│
├── 會員積分上下文（Membership Context）← 核心域
│   ├── MembershipAccount（會員帳戶）
│   ├── PointsTransaction（積分交易）
│   └── PointsConversionRule（積分轉換規則）
│
├── 發票驗證上下文（Invoice Context）← 支撐域
│   ├── Invoice（發票）
│   ├── InvoiceQRCode（發票 QR Code）
│   └── IChefImportRecord（iChef 匯入記錄）
│
├── 問卷上下文（Survey Context）← 支撐域
│   ├── Survey（問卷）
│   ├── SurveyQuestion（問卷題目）
│   └── SurveyResponse（問卷回應）
│
└── 身份認證上下文（Identity Context）← 通用域
    ├── LineUser（LINE 用戶）
    └── AdminUser（管理員用戶）
```

### 4.3 上下文映射（Context Mapping）

#### 4.3.1 定義不同上下文之間的關係

| 關係類型 | 說明 | 適用場景 |
|---------|------|---------|
| **Shared Kernel** | 共享核心模型（緊密耦合） | 兩個團隊需要共享部分模型 |
| **Customer-Supplier** | 客戶-供應商關係 | 上游團隊提供 API 給下游 |
| **Conformist** | 遵從者（下游完全依賴上游） | 必須遵從外部系統的模型 |
| **Anti-Corruption Layer** | 防腐層（保護自己） | 包裝外部 API，避免污染內部模型 |
| **Separate Ways** | 各走各路（完全獨立） | 兩個上下文無業務交集 |
| **Open Host Service** | 開放主機服務 | 提供標準化 API 供多個下游使用 |
| **Published Language** | 發布語言 | 使用標準格式（如 JSON Schema） |

#### 4.3.2 本系統的上下文映射

```
┌─────────────────────────────────────────────────────────┐
│ 會員積分上下文（Membership Context）                      │
│ ├── MembershipAccount                                   │
│ └── PointsTransaction                                   │
└─────────────────────────────────────────────────────────┘
    ↑ Customer-Supplier（供應商）
    │ 提供：GetUserPoints(), RedeemReward()
    │
┌─────────────────────────────────────────────────────────┐
│ 發票驗證上下文（Invoice Context）                        │
│ ├── Invoice                                             │
│ └── 當發票驗證通過 → 更新會員積分                         │
└─────────────────────────────────────────────────────────┘
    ↑ Anti-Corruption Layer（防腐層）
    │ 使用 Adapter 包裝 LINE API
    │
┌─────────────────────────────────────────────────────────┐
│ LINE Platform（外部系統）                                │
│ ├── Conformist 關係（我們必須遵從 LINE 的模型）            │
│ └── 提供：用戶資料、Webhook 事件                          │
└─────────────────────────────────────────────────────────┘
```

**防腐層（ACL）範例**：

```go
// ❌ 直接依賴 LINE SDK 的模型（污染內部模型）
type User struct {
    LineUser *linebot.UserProfileResponse  // 緊密耦合
}

// ✅ 使用防腐層（ACL）
type LineUserAdapter struct {
    linebotClient *linebot.Client
}

func (a *LineUserAdapter) GetUserProfile(lineUserID string) (*domain.MembershipAccount, error) {
    // 調用 LINE API
    profile, err := a.linebotClient.GetProfile(lineUserID).Do()
    if err != nil {
        return nil, err
    }

    // 轉換為內部領域模型（隔離外部變化）
    return &domain.MembershipAccount{
        LineUserID:  profile.UserID,
        DisplayName: profile.DisplayName,
        // LINE 的模型變化不會影響內部
    }, nil
}
```

---

## 5. 戰術設計（Tactical Design）

**目標**：在限界上下文內部設計具體的模型和交互。

### 5.1 實體（Entity）

#### 5.1.1 定義

**實體 = 有唯一標識（ID），可以追蹤其生命週期的對象**

**關鍵特徵**：
1. 有唯一標識（ID）
2. 屬性可以變化
3. 可以追蹤生命週期（創建、修改、刪除）
4. 相等性由 ID 決定，不是屬性

#### 5.1.2 Entity vs 數據庫記錄

| 概念 | Entity（領域模型） | Database Record（數據層） |
|------|-------------------|--------------------------|
| **職責** | 業務邏輯和規則 | 數據持久化 |
| **方法** | 包含業務行為 | 只有 getter/setter |
| **獨立性** | 獨立於數據庫 | 依賴於數據庫 Schema |

#### 5.1.3 範例

```go
// ✅ Entity：包含業務邏輯
type MembershipAccount struct {
    // 唯一標識
    LineUserID  string

    // 屬性
    DisplayName  string
    Phone        string
    EarnedPoints int
    UsedPoints   int

    // 時間戳
    CreatedAt time.Time
    UpdatedAt time.Time
}

// 業務邏輯：計算可用積分
func (m *MembershipAccount) AvailablePoints() int {
    return m.EarnedPoints - m.UsedPoints
}

// 業務邏輯：驗證是否可兌換獎品
func (m *MembershipAccount) CanRedeemReward(rewardCost int) bool {
    return m.AvailablePoints() >= rewardCost
}

// 業務邏輯：兌換獎品
func (m *MembershipAccount) RedeemReward(rewardCost int) error {
    if !m.CanRedeemReward(rewardCost) {
        return ErrInsufficientPoints
    }

    m.UsedPoints += rewardCost
    m.UpdatedAt = time.Now()
    return nil
}

// 業務邏輯：驗證實體有效性
func (m *MembershipAccount) Validate() error {
    if m.LineUserID == "" {
        return errors.New("line_user_id_required")
    }
    if m.UsedPoints > m.EarnedPoints {
        return errors.New("used_points_exceeds_earned_points")
    }
    return nil
}

// 相等性判斷：基於 ID，不是屬性
func (m *MembershipAccount) Equals(other *MembershipAccount) bool {
    return m.LineUserID == other.LineUserID
}
```

**關鍵設計原則：Tell, Don't Ask**

```go
// ❌ 錯誤：外部代碼詢問數據然後操作（Ask）
if account.AvailablePoints() >= rewardCost {
    account.UsedPoints += rewardCost
}

// ✅ 正確：告訴對象做什麼（Tell）
err := account.RedeemReward(rewardCost)
```

### 5.2 值對象（Value Object）

#### 5.2.1 定義

**值對象 = 沒有唯一標識，只關心屬性值的對象**

**關鍵特徵**：
1. 沒有 ID
2. 不可變（Immutable）
3. 相等性由所有屬性決定
4. 可以被替換（兩個值相同的對象可以互換）

#### 5.2.2 為什麼需要值對象？

**問題：使用原始類型（Primitive Obsession）**

```go
// ❌ 使用原始類型
type Order struct {
    Amount   int     // 金額是多少？元？分？美元？台幣？
    Currency string  // 沒有驗證，可能寫成 "TWDD"（打錯字）
}

func (o *Order) Total() int {
    return o.Amount  // 沒有單位，容易出錯
}
```

**解決方案：使用值對象**

```go
// ✅ 值對象：封裝業務概念
type Money struct {
    amount   int     // 私有：只能通過方法訪問
    currency string
}

// 工廠方法：保證創建時的有效性
func NewMoney(amount int, currency string) (Money, error) {
    if amount < 0 {
        return Money{}, errors.New("金額不能為負")
    }
    if currency != "TWD" && currency != "USD" {
        return Money{}, errors.New("不支持的貨幣")
    }
    return Money{amount: amount, currency: currency}, nil
}

// 值對象是不可變的：操作返回新對象
func (m Money) Add(other Money) (Money, error) {
    if m.currency != other.currency {
        return Money{}, errors.New("不同貨幣不能相加")
    }
    return Money{
        amount:   m.amount + other.amount,
        currency: m.currency,
    }, nil
}

// 相等性：基於所有屬性
func (m Money) Equals(other Money) bool {
    return m.amount == other.amount && m.currency == other.currency
}

// Getter 方法（不提供 Setter，保證不可變性）
func (m Money) Amount() int {
    return m.amount
}

func (m Money) Currency() string {
    return m.currency
}
```

#### 5.2.3 使用值對象的優勢

```go
// ✅ 使用值對象
type Order struct {
    TotalAmount Money  // 清晰：包含金額和貨幣
}

// 業務邏輯清晰可讀
func (o *Order) CanApplyDiscount() bool {
    minimumAmount, _ := NewMoney(1000, "TWD")
    return o.TotalAmount.GreaterThan(minimumAmount)
}

// 類型安全：編譯時就能發現錯誤
func (o *Order) AddItem(price Money, quantity int) error {
    itemTotal, err := price.Multiply(quantity)
    if err != nil {
        return err
    }

    o.TotalAmount, err = o.TotalAmount.Add(itemTotal)
    return err
}
```

#### 5.2.4 常見的值對象

| 業務概念 | 值對象類型 | 封裝的邏輯 |
|---------|-----------|-----------|
| 金額 | `Money` | 金額驗證、貨幣轉換、加減運算 |
| 電話號碼 | `PhoneNumber` | 格式驗證、國碼處理 |
| 電子郵件 | `Email` | 格式驗證 |
| 地址 | `Address` | 郵遞區號、國家、城市組合 |
| 日期範圍 | `DateRange` | 開始/結束日期、重疊檢查 |
| 發票號碼 | `InvoiceNumber` | 格式驗證、校驗碼檢查 |

**本系統範例：**

```go
// 台灣手機號碼值對象
type PhoneNumber struct {
    number string  // 格式：09XXXXXXXX
}

func NewPhoneNumber(number string) (PhoneNumber, error) {
    // 移除空格和連字號
    cleaned := strings.ReplaceAll(number, " ", "")
    cleaned = strings.ReplaceAll(cleaned, "-", "")

    // 驗證格式
    if len(cleaned) != 10 {
        return PhoneNumber{}, errors.New("手機號碼必須是 10 位數")
    }
    if !strings.HasPrefix(cleaned, "09") {
        return PhoneNumber{}, errors.New("手機號碼必須以 09 開頭")
    }

    return PhoneNumber{number: cleaned}, nil
}

func (p PhoneNumber) String() string {
    return p.number
}

func (p PhoneNumber) FormattedString() string {
    // 格式化為：0912-345-678
    return fmt.Sprintf("%s-%s-%s",
        p.number[0:4],
        p.number[4:7],
        p.number[7:10])
}
```

### 5.3 聚合（Aggregate）與聚合根（Aggregate Root）

#### 5.3.1 定義

**聚合 = 一組必須保持一致性的對象集合**

**聚合根 = 聚合的入口點，外部只能通過聚合根訪問聚合內的對象**

#### 5.3.2 聚合設計的核心原則

| 原則 | 說明 | 違反後果 |
|------|------|---------|
| **聚合邊界 = 事務邊界** | 一個事務只修改一個聚合 | 性能問題、死鎖 |
| **只能通過聚合根修改** | 外部不能直接訪問聚合內的子實體 | 無法保證不變量 |
| **聚合之間用 ID 引用** | 不直接持有其他聚合的對象引用 | 強耦合、無法獨立演化 |
| **聚合應該小** | 一個聚合包含盡可能少的實體 | 鎖競爭、性能下降 |

#### 5.3.3 設計聚合的步驟

**步驟 1：識別不變量（Invariant）**

不變量 = 必須始終為真的業務規則

**例子：訂單聚合的不變量**

```
不變量 1：訂單總金額 = 所有訂單項的金額總和
不變量 2：訂單狀態只能按特定順序轉換（draft → confirmed → shipped → delivered）
不變量 3：已確認的訂單不能修改訂單項
不變量 4：訂單至少要有一個訂單項
```

**步驟 2：確定聚合邊界**

**判斷標準：需要在同一事務中保證一致性的對象，屬於同一個聚合**

```go
// ✅ 正確的聚合設計
type Order struct {  // ← Aggregate Root
    orderID    string
    customerID string  // ← 通過 ID 引用 Customer Aggregate
    status     OrderStatus
    orderItems []OrderItem  // ← 聚合內的子實體
    totalAmount Money
}

type OrderItem struct {  // ← 不是 Aggregate Root，只能通過 Order 訪問
    productID string  // ← 通過 ID 引用 Product Aggregate
    quantity  int
    price     Money
}

// ✅ 通過聚合根修改，保證不變量
func (o *Order) AddItem(productID string, quantity int, price Money) error {
    // 驗證業務規則
    if o.status != OrderStatusDraft {
        return errors.New("已確認的訂單不能修改")
    }

    // 添加訂單項
    item := OrderItem{
        productID: productID,
        quantity:  quantity,
        price:     price,
    }
    o.orderItems = append(o.orderItems, item)

    // 保證不變量：重新計算總金額
    o.recalculateTotalAmount()
    return nil
}

func (o *Order) recalculateTotalAmount() {
    total := Money{amount: 0, currency: "TWD"}
    for _, item := range o.orderItems {
        itemTotal := item.price.Multiply(item.quantity)
        total = total.Add(itemTotal)
    }
    o.totalAmount = total
}

// ✅ 狀態轉換邏輯封裝在聚合根
func (o *Order) Confirm() error {
    if o.status != OrderStatusDraft {
        return errors.New("只能確認草稿訂單")
    }

    if len(o.orderItems) == 0 {
        return errors.New("訂單至少要有一個訂單項")
    }

    o.status = OrderStatusConfirmed
    o.confirmedAt = time.Now()
    return nil
}
```

**步驟 3：判斷是否應該分離聚合**

**決策矩陣：**

| 問題 | 如果答案是「是」 | 如果答案是「否」 |
|------|----------------|----------------|
| A 和 B 可以在不同事務中修改嗎？ | 分離聚合 | 同一聚合 |
| A 和 B 有不同的生命週期嗎？ | 分離聚合 | 同一聚合 |
| A 不存在時，B 仍然有意義嗎？ | 分離聚合 | 同一聚合 |
| A 和 B 由不同的業務團隊管理嗎？ | 分離聚合 | 同一聚合 |
| A 和 B 的修改頻率差異很大嗎？ | 考慮分離 | 同一聚合 |

#### 5.3.4 本系統的聚合設計

**聚合 1：MembershipAccount（會員帳戶）**

```go
type MembershipAccount struct {  // ← Aggregate Root
    lineUserID   string  // 唯一標識
    displayName  string
    phone        PhoneNumber  // ← 值對象
    earnedPoints int
    usedPoints   int
    createdAt    time.Time
    updatedAt    time.Time
}

// 不變量：
// 1. EarnedPoints >= 0
// 2. UsedPoints >= 0
// 3. UsedPoints <= EarnedPoints

func (m *MembershipAccount) AvailablePoints() int {
    return m.earnedPoints - m.usedPoints
}

func (m *MembershipAccount) RedeemReward(cost int) error {
    if cost <= 0 {
        return errors.New("兌換金額必須大於 0")
    }
    if m.AvailablePoints() < cost {
        return ErrInsufficientPoints
    }

    m.usedPoints += cost  // 保證不變量
    m.updatedAt = time.Now()
    return nil
}

func (m *MembershipAccount) AddEarnedPoints(points int) error {
    if points <= 0 {
        return errors.New("增加的積分必須大於 0")
    }

    m.earnedPoints += points
    m.updatedAt = time.Now()
    return nil
}
```

**聚合 2：Invoice（發票）**

```go
type Invoice struct {  // ← Aggregate Root
    invoiceNo   string  // 唯一標識
    invoiceDate time.Time
    amount      Money    // ← 值對象
    status      InvoiceStatus
    userID      string   // ← 通過 ID 引用 MembershipAccount Aggregate
    verifiedAt  *time.Time
}

// 不變量：
// 1. 發票號碼唯一
// 2. 狀態轉換合法：imported → verified / failed
// 3. 已驗證的發票不能改為 failed

func (i *Invoice) Verify() error {
    if i.status != InvoiceStatusImported {
        return errors.New("只能驗證已匯入的發票")
    }

    if !i.IsWithinValidPeriod() {
        return errors.New("發票已超過有效期限")
    }

    i.status = InvoiceStatusVerified
    now := time.Now()
    i.verifiedAt = &now
    return nil
}

func (i *Invoice) IsWithinValidPeriod() bool {
    daysSinceInvoice := time.Since(i.invoiceDate).Hours() / 24
    return daysSinceInvoice <= 60
}
```

**聚合 3：Survey（問卷）**

```go
type Survey struct {  // ← Aggregate Root
    surveyID    string
    title       string
    description string
    active      bool
    questions   []SurveyQuestion  // ← 聚合內的子實體
    createdAt   time.Time
}

type SurveyQuestion struct {  // ← 不是 Aggregate Root
    questionID   string
    questionText string
    questionType QuestionType
    options      []string
    required     bool
    order        int
}

// 不變量：
// 1. 問卷至少要有一個問題
// 2. 同時只能有一個問卷處於 active 狀態
// 3. 刪除問卷會級聯刪除所有問題

// ✅ 通過聚合根修改問題
func (s *Survey) AddQuestion(text string, qType QuestionType, required bool) error {
    if s.active {
        return errors.New("啟用中的問卷不能修改")
    }

    question := SurveyQuestion{
        questionID:   uuid.New().String(),
        questionText: text,
        questionType: qType,
        required:     required,
        order:        len(s.questions) + 1,
    }

    s.questions = append(s.questions, question)
    return nil
}

func (s *Survey) Activate() error {
    if len(s.questions) == 0 {
        return errors.New("問卷至少要有一個問題才能啟用")
    }

    s.active = true
    return nil
}

// ❌ 錯誤：外部不應該直接修改問題
// survey.Questions[0].QuestionText = "new text"  // 繞過了驗證邏輯

// ✅ 正確：通過聚合根修改
// survey.UpdateQuestion(questionID, "new text")
```

#### 5.3.5 常見錯誤

**錯誤 1：聚合太大**

```go
// ❌ 聚合太大：包含太多實體
type User struct {
    UserID   string
    Orders   []Order        // ← 不應該在聚合內
    Addresses []Address     // ← 不應該在聚合內
    CreditCards []CreditCard  // ← 不應該在聚合內
}

// ✅ 正確：聚合應該小
type User struct {
    UserID   string
    // 只包含用戶身份相關的屬性
}

// Order 是獨立的聚合
type Order struct {
    OrderID  string
    UserID   string  // ← 通過 ID 引用
}
```

**錯誤 2：跨聚合修改**

```go
// ❌ 錯誤：在一個事務中修改多個聚合
func TransferPoints(fromUserID, toUserID string, points int) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // 修改第一個聚合
        fromUser := userRepo.FindByID(fromUserID)
        fromUser.UsedPoints += points
        userRepo.Update(fromUser)

        // 修改第二個聚合（違反原則）
        toUser := userRepo.FindByID(toUserID)
        toUser.EarnedPoints += points
        userRepo.Update(toUser)

        return nil
    })
}

// ✅ 正確：使用領域事件實現跨聚合操作
func TransferPoints(fromUserID string, points int) error {
    // 只修改一個聚合
    fromUser := userRepo.FindByID(fromUserID)
    err := fromUser.TransferPointsOut(points)
    if err != nil {
        return err
    }
    userRepo.Update(fromUser)

    // 發布領域事件
    eventBus.Publish(PointsTransferredOut{
        FromUserID: fromUserID,
        Points:     points,
    })

    return nil
}

// 事件處理器：異步處理第二個聚合
func OnPointsTransferredOut(event PointsTransferredOut) {
    toUser := userRepo.FindByID(event.ToUserID)
    toUser.ReceivePoints(event.Points)
    userRepo.Update(toUser)
}
```

### 5.4 領域服務（Domain Service）

#### 5.4.1 定義

**領域服務 = 不屬於任何 Entity 或 Value Object 的業務邏輯**

#### 5.4.2 何時使用領域服務？

| 場景 | 是否使用領域服務 | 原因 |
|------|----------------|------|
| 計算單個對象的屬性 | ❌ 否 | 應該在 Entity/Value Object 內 |
| 跨多個聚合的業務邏輯 | ✅ 是 | 不屬於任何單一聚合 |
| 需要訪問外部系統 | ✅ 是 | Entity 不應該有外部依賴 |
| 需要使用 Repository | ✅ 是 | Entity 不應該依賴 Repository |

#### 5.4.3 範例

**場景：計算用戶積分（需要查詢所有交易記錄）**

```go
// ❌ 錯誤：在 Entity 內訪問 Repository
type MembershipAccount struct {
    earnedPoints int
    txRepo TransactionRepository  // ← Entity 不應該依賴 Repository
}

func (m *MembershipAccount) RecalculatePoints() error {
    // Entity 不應該知道如何查詢數據庫
    transactions := m.txRepo.FindByUserID(m.LineUserID)
    // ...
}

// ✅ 正確：使用領域服務
type PointsCalculationService struct {
    transactionRepo TransactionRepository
    ruleRepo        PointsConversionRuleRepository
}

func (s *PointsCalculationService) CalculateUserPoints(userID string) (int, error) {
    // 查詢所有已驗證的交易
    transactions, err := s.transactionRepo.FindVerifiedByUserID(userID)
    if err != nil {
        return 0, err
    }

    // 獲取積分轉換規則
    rule, err := s.ruleRepo.FindActiveRule()
    if err != nil {
        return 0, err
    }

    // 計算總積分
    totalPoints := 0
    for _, tx := range transactions {
        // 委託給 Entity 計算單筆交易的積分
        points := tx.CalculatePoints(rule.AmountPerPoint)
        totalPoints += points
    }

    return totalPoints, nil
}
```

**本系統的領域服務：**

```go
// 1. 積分計算服務
type PointsCalculationService struct {
    transactionRepo        TransactionRepository
    surveyResponseRepo     SurveyResponseRepository
    pointsConversionRepo   PointsConversionRuleRepository
}

func (s *PointsCalculationService) RecalculateUserPoints(userID string, date time.Time) (int, error) {
    // 跨聚合的業務邏輯
    transactions, _ := s.transactionRepo.FindVerifiedByUserID(userID)
    rule, _ := s.pointsConversionRepo.FindByDate(date)

    totalPoints := 0
    for _, tx := range transactions {
        // 基礎積分
        basePoints := tx.CalculateBasePoints(rule.AmountPerPoint)
        totalPoints += basePoints

        // 問卷獎勵（跨聚合查詢）
        hasSurvey, _ := s.surveyResponseRepo.HasResponseForTransaction(tx.ID)
        if hasSurvey {
            totalPoints += 1
        }
    }

    return totalPoints, nil
}

// 2. 發票匹配服務
type InvoiceMatchingService struct {
    invoiceRepo       InvoiceRepository
    ichefRecordRepo   IChefRecordRepository
}

func (s *InvoiceMatchingService) MatchInvoices(ichefRecords []IChefInvoiceRecord) (*MatchResult, error) {
    // 跨聚合的匹配邏輯
    matched := 0
    unmatched := 0

    for _, record := range ichefRecords {
        invoice, err := s.invoiceRepo.FindByInvoiceNoDateAmount(
            record.InvoiceNo,
            record.Date,
            record.Amount,
        )

        if err == nil && invoice != nil {
            // 找到匹配：更新發票狀態
            invoice.Verify()
            s.invoiceRepo.Update(invoice)
            matched++
        } else {
            unmatched++
        }
    }

    return &MatchResult{
        Matched:   matched,
        Unmatched: unmatched,
    }, nil
}
```

### 5.5 領域事件（Domain Event）

#### 5.5.1 定義

**領域事件 = 領域中發生的重要業務事實**

**特徵**：
- 過去式命名（已經發生的事情）：`OrderConfirmed`, `InvoiceVerified`
- 不可變（Immutable）
- 包含事件發生的時間戳
- 包含足夠的信息讓訂閱者處理

#### 5.5.2 為什麼需要領域事件？

**問題：跨聚合的業務邏輯如何處理？**

**場景：發票驗證後，需要重新計算用戶積分**

```go
// ❌ 方案 1：直接調用（緊耦合）
func VerifyInvoice(invoiceID string) error {
    invoice := invoiceRepo.FindByID(invoiceID)
    invoice.Verify()
    invoiceRepo.Update(invoice)

    // 緊耦合：Invoice 聚合直接依賴 MembershipAccount 聚合
    account := accountRepo.FindByUserID(invoice.UserID)
    pointsService.RecalculatePoints(account)
    accountRepo.Update(account)
}

// ❌ 方案 2：在同一事務中修改兩個聚合（違反 DDD 原則）
func VerifyInvoice(invoiceID string) error {
    return db.Transaction(func(tx *gorm.DB) error {
        invoice := invoiceRepo.FindByID(invoiceID)
        invoice.Verify()
        invoiceRepo.Update(invoice)

        account := accountRepo.FindByUserID(invoice.UserID)
        account.RecalculatePoints()
        accountRepo.Update(account)  // 違反原則：修改了兩個聚合
    })
}

// ✅ 方案 3：使用領域事件（鬆耦合 + 最終一致性）
func VerifyInvoice(invoiceID string) error {
    invoice := invoiceRepo.FindByID(invoiceID)
    invoice.Verify()
    invoiceRepo.Update(invoice)

    // 發布領域事件
    eventBus.Publish(InvoiceVerified{
        InvoiceID: invoiceID,
        UserID:    invoice.UserID,
        Amount:    invoice.Amount,
        VerifiedAt: time.Now(),
    })

    return nil
}

// 事件處理器：異步處理
func OnInvoiceVerified(event InvoiceVerified) {
    // 在獨立的事務中處理
    account := accountRepo.FindByUserID(event.UserID)
    pointsService.RecalculatePoints(account)
    accountRepo.Update(account)
}
```

#### 5.5.3 領域事件的實現

**定義事件：**

```go
// 領域事件接口
type DomainEvent interface {
    OccurredAt() time.Time
    EventName() string
}

// 具體事件
type InvoiceVerified struct {
    invoiceID  string
    userID     string
    amount     Money
    verifiedAt time.Time
}

func (e InvoiceVerified) OccurredAt() time.Time {
    return e.verifiedAt
}

func (e InvoiceVerified) EventName() string {
    return "InvoiceVerified"
}
```

**發布事件：**

```go
// 簡單的內存事件總線
type EventBus struct {
    handlers map[string][]EventHandler
}

type EventHandler func(event DomainEvent) error

func (bus *EventBus) Subscribe(eventName string, handler EventHandler) {
    bus.handlers[eventName] = append(bus.handlers[eventName], handler)
}

func (bus *EventBus) Publish(event DomainEvent) {
    handlers := bus.handlers[event.EventName()]
    for _, handler := range handlers {
        // 異步處理（可以用 goroutine 或消息隊列）
        go handler(event)
    }
}
```

**本系統的領域事件：**

```go
// 事件 1：發票已驗證
type InvoiceVerified struct {
    InvoiceID  string
    UserID     string
    Amount     int
    VerifiedAt time.Time
}

// 事件 2：問卷已提交
type SurveySubmitted struct {
    SurveyID      string
    TransactionID string
    UserID        string
    SubmittedAt   time.Time
}

// 事件 3：積分轉換規則已變更
type PointsConversionRuleChanged struct {
    RuleID    string
    StartDate time.Time
    EndDate   time.Time
    ChangedAt time.Time
}

// 事件處理器
func OnInvoiceVerified(event InvoiceVerified) error {
    // 重新計算用戶積分
    return pointsService.RecalculateUserPoints(event.UserID)
}

func OnSurveySubmitted(event SurveySubmitted) error {
    // 標記交易已提交問卷，重新計算積分
    return pointsService.RecalculateUserPoints(event.UserID)
}

func OnPointsRuleChanged(event PointsConversionRuleChanged) error {
    // 記錄審計日誌，通知管理員需要手動重算積分
    logger.Warn("積分規則已變更，請運行 make recalculate-points")
    return nil
}
```

### 5.6 倉儲（Repository）

#### 5.6.1 定義

**Repository = 聚合的持久化和查詢接口**

**關鍵原則**：
- 每個聚合根有一個 Repository
- Repository 接口屬於領域層（Domain Layer）
- Repository 實現屬於基礎設施層（Infrastructure Layer）

#### 5.6.2 Repository 設計

```go
// ✅ 領域層定義接口
type MembershipAccountRepository interface {
    // 基本 CRUD
    Save(account *MembershipAccount) error
    FindByLineUserID(lineUserID string) (*MembershipAccount, error)
    Delete(lineUserID string) error

    // 業務查詢
    FindByPhone(phone PhoneNumber) (*MembershipAccount, error)
    FindTopPointsHolders(limit int) ([]*MembershipAccount, error)
}

// ✅ 基礎設施層實現
type GormMembershipAccountRepository struct {
    db *gorm.DB
}

func (r *GormMembershipAccountRepository) Save(account *MembershipAccount) error {
    // 轉換為 GORM 模型
    model := &MembershipAccountModel{
        LineUserID:   account.LineUserID,
        DisplayName:  account.DisplayName,
        Phone:        account.Phone.String(),
        EarnedPoints: account.EarnedPoints,
        UsedPoints:   account.UsedPoints,
    }

    return r.db.Save(model).Error
}

func (r *GormMembershipAccountRepository) FindByLineUserID(lineUserID string) (*MembershipAccount, error) {
    var model MembershipAccountModel
    err := r.db.Where("line_user_id = ?", lineUserID).First(&model).Error
    if err != nil {
        return nil, err
    }

    // 轉換為領域模型
    phone, _ := NewPhoneNumber(model.Phone)
    return &MembershipAccount{
        LineUserID:   model.LineUserID,
        DisplayName:  model.DisplayName,
        Phone:        phone,
        EarnedPoints: model.EarnedPoints,
        UsedPoints:   model.UsedPoints,
    }, nil
}
```

#### 5.6.3 Repository 的職責劃分（ISP 原則）

**問題：單一 Repository 接口太大**

```go
// ❌ 違反 ISP：接口太大
type UserRepository interface {
    // 寫入操作
    Create(user *User) error
    Update(user *User) error
    Delete(userID string) error

    // 讀取操作
    FindByID(userID string) (*User, error)
    FindByEmail(email string) (*User, error)

    // 複雜查詢
    List(offset, limit int, filters UserFilters) ([]*User, error)
    Count(filters UserFilters) (int, error)
    Search(keyword string) ([]*User, error)
}
```

**解決方案：按職責拆分**

```go
// ✅ 寫入接口
type UserWriter interface {
    Create(user *User) error
    Update(user *User) error
    Delete(userID string) error
}

// ✅ 讀取接口
type UserReader interface {
    FindByID(userID string) (*User, error)
    FindByEmail(email string) (*User, error)
}

// ✅ 查詢服務接口
type UserQueryService interface {
    List(offset, limit int, filters UserFilters) ([]*User, error)
    Count(filters UserFilters) (int, error)
    Search(keyword string) ([]*User, error)
}

// Use Case 只依賴需要的接口
type RegisterUserUseCase struct {
    writer UserWriter  // 只需要寫入
    reader UserReader  // 只需要讀取
}
```

---

## 6. DDD 設計流程

### 6.1 完整流程圖

```
階段 1：戰略設計（Strategic Design）
├── 步驟 1：與業務專家溝通，識別領域
│   └── 輸出：領域邊界、核心業務活動
├── 步驟 2：劃分限界上下文（Bounded Context）
│   └── 輸出：上下文邊界圖、統一語言詞彙表
├── 步驟 3：定義上下文映射（Context Mapping）
│   └── 輸出：上下文關係圖
└── 步驟 4：識別核心域、支撐域、通用域
    └── 輸出：投資優先級決策

階段 2：戰術設計（Tactical Design）
├── 步驟 5：在每個 Bounded Context 內識別實體和值對象
│   └── 輸出：領域模型草圖
├── 步驟 6：識別不變量（Invariant）
│   └── 輸出：業務規則清單
├── 步驟 7：根據不變量劃分聚合邊界
│   └── 輸出：聚合設計圖
├── 步驟 8：設計 Repository 接口
│   └── 輸出：Repository 接口定義
├── 步驟 9：識別需要領域服務的業務邏輯
│   └── 輸出：領域服務列表
└── 步驟 10：定義領域事件
    └── 輸出：領域事件清單
```

### 6.2 實戰範例：從需求到設計

#### 需求

> 開發一個餐廳會員管理系統，用戶通過 LINE Bot 掃描發票 QR Code，系統驗證發票並計算積分。用戶可以用積分兌換獎品。

#### 步驟 1：識別領域

**與業務專家對話：**

```
開發者：「這個系統的核心業務是什麼？」
業務專家：「是會員積分管理，我們希望通過積分獎勵吸引客戶重複消費。」

開發者：「積分怎麼計算？」
業務專家：「基本上是 100 元 1 點，但我們會定期調整這個比例，比如促銷期間可能是 50 元 1 點。」

開發者：「發票驗證的規則是什麼？」
業務專家：「發票必須是 60 天內的，而且不能重複使用。我們每個月會從 iChef 系統匯入實際的交易記錄來驗證。」
```

**輸出：領域邊界**

```
核心域：會員積分系統
  - 積分計算
  - 積分兌換
  - 會員管理

支撐域：
  - 發票驗證
  - iChef 整合
  - 問卷系統

通用域：
  - LINE Bot 整合
  - 身份認證
```

#### 步驟 2：劃分限界上下文

**識別統一語言：**

| 業務術語 | 含義 | 代碼命名 |
|---------|------|---------|
| 會員 | 註冊用戶 | `MembershipAccount` |
| 累積積分 | 用戶賺取的總積分 | `EarnedPoints` |
| 可用積分 | 可以使用的積分 | `AvailablePoints` |
| 兌換獎品 | 使用積分換取獎品 | `RedeemReward()` |
| 發票驗證 | 確認發票真實性 | `VerifyInvoice()` |

**輸出：上下文劃分**

```
會員積分上下文（Membership Context）
  - 關注：積分計算、兌換
  - 模型：MembershipAccount, PointsConversionRule

發票驗證上下文（Invoice Context）
  - 關注：發票真偽、有效期
  - 模型：Invoice, IChefRecord

問卷上下文（Survey Context）
  - 關注：問卷設計、回應收集
  - 模型：Survey, SurveyResponse
```

#### 步驟 3：定義上下文映射

```
會員積分上下文 (Customer)
    ↑
    | 提供：GetUserPoints(), AddPoints()
    |
發票驗證上下文 (Supplier)
    ↑
    | Anti-Corruption Layer（防腐層）
    |
LINE Platform (External)
```

#### 步驟 4-7：戰術設計（在會員積分上下文內）

**識別實體和值對象：**

```go
// 實體
type MembershipAccount struct {
    LineUserID   string  // 唯一標識
    DisplayName  string
    Phone        PhoneNumber  // 值對象
    EarnedPoints int
    UsedPoints   int
}

// 值對象
type PhoneNumber struct {
    number string
}

func NewPhoneNumber(number string) (PhoneNumber, error) {
    // 驗證邏輯
}
```

**識別不變量：**

```
不變量 1：EarnedPoints >= 0
不變量 2：UsedPoints >= 0
不變量 3：UsedPoints <= EarnedPoints
不變量 4：AvailablePoints = EarnedPoints - UsedPoints
```

**劃分聚合：**

```
聚合 1：MembershipAccount
  - 包含：EarnedPoints, UsedPoints
  - 理由：必須在同一事務中保證不變量

聚合 2：Invoice
  - 獨立聚合
  - 通過 UserID 引用 MembershipAccount
```

**設計 Repository：**

```go
type MembershipAccountRepository interface {
    Save(account *MembershipAccount) error
    FindByLineUserID(lineUserID string) (*MembershipAccount, error)
}
```

---

## 7. 實踐指南

### 7.1 如何開始應用 DDD

#### 階段 1：建立統一語言（1-2 週）

**行動清單：**
- ✅ 與業務專家開會，列出核心業務術語
- ✅ 創建術語詞彙表（Glossary）
- ✅ 在代碼中統一使用業務術語
- ✅ Code Review 時檢查術語一致性

**產出物：**
```markdown
# 統一語言詞彙表

| 業務術語 | 英文 | 代碼命名 | 說明 |
|---------|------|---------|------|
| 會員 | Member | MembershipAccount | 註冊用戶 |
| 累積積分 | Earned Points | EarnedPoints | 用戶賺取的總積分 |
| 可用積分 | Available Points | AvailablePoints() | 可以使用的積分 |
```

#### 階段 2：識別聚合（2-4 週）

**行動清單：**
- ✅ 列出所有業務規則（不變量）
- ✅ 根據事務邊界劃分聚合
- ✅ 繪製聚合關係圖
- ✅ 在團隊內評審聚合設計

**工具：**
- 白板或 Miro
- Markdown 文檔
- PlantUML / Mermaid 圖表

#### 階段 3：重構現有代碼（持續進行）

**重構策略：**

```
優先級 1：核心域
  - 先重構最複雜、變更最頻繁的業務邏輯
  - 使用 DDD 戰術設計（Entity, Aggregate, Domain Service）

優先級 2：支撐域
  - 保持簡單，快速交付
  - 可以繼續使用 CRUD 架構

優先級 3：通用域
  - 使用第三方服務或開源方案
  - 不投入太多開發資源
```

### 7.2 團隊協作

#### 7.2.1 角色分工

| 角色 | 職責 | DDD 中的作用 |
|------|------|-------------|
| **業務專家** | 定義業務規則 | 提供領域知識，定義統一語言 |
| **架構師** | 設計系統架構 | 劃分限界上下文，定義上下文映射 |
| **開發人員** | 實現代碼 | 實現戰術設計（Entity, Aggregate, Repository） |
| **測試人員** | 驗證業務規則 | 測試不變量、業務規則 |

#### 7.2.2 協作流程

```
1. 業務專家講解業務規則
   ↓
2. 團隊共同建模（Event Storming）
   ↓
3. 開發人員設計聚合和實體
   ↓
4. Code Review 檢查統一語言
   ↓
5. 測試人員驗證業務規則
```

### 7.3 代碼組織結構

**推薦的目錄結構（Go 語言）：**

```
project/
├── cmd/
│   ├── app/              # 應用程式入口
│   └── migrate/          # 數據庫遷移工具
│
├── internal/
│   ├── domain/           # 領域層（DDD 核心）
│   │   ├── membership/   # 會員積分上下文
│   │   │   ├── account.go           # Entity
│   │   │   ├── phone_number.go      # Value Object
│   │   │   ├── points_calculator.go # Domain Service
│   │   │   ├── repository.go        # Repository 接口
│   │   │   └── events.go            # Domain Events
│   │   │
│   │   ├── invoice/      # 發票驗證上下文
│   │   │   ├── invoice.go
│   │   │   ├── invoice_status.go
│   │   │   └── repository.go
│   │   │
│   │   └── survey/       # 問卷上下文
│   │       ├── survey.go
│   │       ├── survey_response.go
│   │       └── repository.go
│   │
│   ├── application/      # 應用層（Use Cases）
│   │   ├── membership/
│   │   │   ├── register_user.go
│   │   │   ├── redeem_reward.go
│   │   │   └── calculate_points.go
│   │   └── invoice/
│   │       ├── verify_invoice.go
│   │       └── import_ichef.go
│   │
│   ├── infrastructure/   # 基礎設施層
│   │   ├── persistence/  # Repository 實現
│   │   │   ├── gorm_account_repo.go
│   │   │   └── gorm_invoice_repo.go
│   │   ├── messaging/    # 事件總線
│   │   └── external/     # 外部服務適配器
│   │       └── line_adapter.go  # Anti-Corruption Layer
│   │
│   └── interfaces/       # 接口層
│       ├── http/         # HTTP API
│       │   └── gin_handler.go
│       └── linebot/      # LINE Bot Webhook
│           └── webhook_handler.go
│
└── docs/
    └── architecture/
        ├── DDD_GUIDE.md
        └── DOMAIN_MODEL.md
```

### 7.4 測試策略

#### 7.4.1 單元測試：測試領域模型

```go
func TestMembershipAccount_RedeemReward(t *testing.T) {
    tests := []struct {
        name          string
        earnedPoints  int
        usedPoints    int
        rewardCost    int
        expectError   bool
        expectedUsed  int
    }{
        {
            name:         "成功兌換",
            earnedPoints: 100,
            usedPoints:   0,
            rewardCost:   50,
            expectError:  false,
            expectedUsed: 50,
        },
        {
            name:         "積分不足",
            earnedPoints: 100,
            usedPoints:   80,
            rewardCost:   50,
            expectError:  true,
            expectedUsed: 80,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            account := &MembershipAccount{
                LineUserID:   "U1234567890",
                EarnedPoints: tt.earnedPoints,
                UsedPoints:   tt.usedPoints,
            }

            err := account.RedeemReward(tt.rewardCost)

            if tt.expectError && err == nil {
                t.Errorf("期望錯誤但沒有錯誤")
            }
            if !tt.expectError && err != nil {
                t.Errorf("不期望錯誤但發生錯誤: %v", err)
            }
            if account.UsedPoints != tt.expectedUsed {
                t.Errorf("UsedPoints = %d, 期望 %d", account.UsedPoints, tt.expectedUsed)
            }
        })
    }
}
```

#### 7.4.2 集成測試：測試 Repository

```go
func TestGormAccountRepository_Save(t *testing.T) {
    // 使用測試數據庫
    db := setupTestDB(t)
    defer db.Close()

    repo := NewGormAccountRepository(db)

    phone, _ := NewPhoneNumber("0912345678")
    account := &MembershipAccount{
        LineUserID:   "U1234567890",
        DisplayName:  "測試用戶",
        Phone:        phone,
        EarnedPoints: 100,
        UsedPoints:   0,
    }

    err := repo.Save(account)
    assert.NoError(t, err)

    // 驗證保存成功
    found, err := repo.FindByLineUserID("U1234567890")
    assert.NoError(t, err)
    assert.Equal(t, "測試用戶", found.DisplayName)
    assert.Equal(t, 100, found.EarnedPoints)
}
```

#### 7.4.3 端到端測試：測試 Use Case

```go
func TestRedeemRewardUseCase_Execute(t *testing.T) {
    // 準備測試數據
    db := setupTestDB(t)
    accountRepo := NewGormAccountRepository(db)
    rewardRepo := NewGormRewardRepository(db)

    // 創建測試用戶
    phone, _ := NewPhoneNumber("0912345678")
    account := &MembershipAccount{
        LineUserID:   "U1234567890",
        DisplayName:  "測試用戶",
        Phone:        phone,
        EarnedPoints: 1000,
        UsedPoints:   0,
    }
    accountRepo.Save(account)

    // 創建測試獎品
    reward := &Reward{
        RewardID: "R001",
        Name:     "咖啡券",
        Cost:     500,
    }
    rewardRepo.Save(reward)

    // 執行 Use Case
    useCase := NewRedeemRewardUseCase(accountRepo, rewardRepo)
    err := useCase.Execute(context.Background(), "U1234567890", "R001")

    assert.NoError(t, err)

    // 驗證積分扣減
    updatedAccount, _ := accountRepo.FindByLineUserID("U1234567890")
    assert.Equal(t, 500, updatedAccount.UsedPoints)
    assert.Equal(t, 500, updatedAccount.AvailablePoints())
}
```

---

## 8. 常見誤解與陷阱

### 8.1 誤解 1：「DDD = 微服務」

❌ **錯誤認知**：
> 「我們要用 DDD，所以每個聚合都應該是一個微服務。」

✅ **正確理解**：
- DDD 和微服務是兩個獨立的概念
- DDD 的限界上下文可以幫助劃分微服務邊界
- 但一個限界上下文不一定對應一個微服務
- 小型系統用單體架構 + DDD 完全可行

**建議**：
```
用戶規模 < 10 萬：單體應用 + DDD
用戶規模 10-100 萬：模組化單體 + DDD
用戶規模 > 100 萬：考慮微服務 + DDD
```

### 8.2 誤解 2：「所有項目都應該用 DDD」

❌ **錯誤認知**：
> 「DDD 是最佳實踐，所有項目都應該用 DDD。」

✅ **正確理解**：
- DDD 適合業務邏輯複雜的系統
- CRUD 系統不需要 DDD（過度設計）
- 團隊需要有 DDD 經驗，否則會適得其反

**決策矩陣：**

| 項目特徵 | 是否使用 DDD |
|---------|------------|
| 簡單的 CRUD 系統 | ❌ 不需要 |
| 複雜的業務規則 | ✅ 需要 |
| 業務規則頻繁變更 | ✅ 需要 |
| 團隊缺乏 DDD 經驗 | ❌ 先學習再用 |
| 項目週期 < 3 個月 | ❌ 時間不夠 |

### 8.3 誤解 3：「Entity 不能有行為」

❌ **錯誤認知**：
> 「Entity 應該是貧血模型，只有屬性沒有方法，所有邏輯放在 Service。」

✅ **正確理解**：
- DDD 反對貧血模型
- Entity 應該包含業務邏輯
- Service 只處理跨聚合的邏輯

**對比：**

```go
// ❌ 貧血模型
type Order struct {
    ID          int
    Status      string
    TotalAmount int
}

type OrderService struct {}

func (s *OrderService) ConfirmOrder(order *Order) error {
    if order.Status != "draft" {
        return errors.New("只能確認草稿訂單")
    }
    order.Status = "confirmed"
}

// ✅ 充血模型
type Order struct {
    ID          int
    Status      OrderStatus
    TotalAmount Money
}

func (o *Order) Confirm() error {
    if o.Status != OrderStatusDraft {
        return errors.New("只能確認草稿訂單")
    }
    o.Status = OrderStatusConfirmed
    return nil
}
```

### 8.4 陷阱 1：過度拆分聚合

❌ **錯誤做法**：
```go
// 把每個 Entity 都當作獨立聚合
type User struct {
    UserID string
}

type UserProfile struct {
    ProfileID string
    UserID    string  // 引用
}

type UserSettings struct {
    SettingsID string
    UserID     string  // 引用
}
```

✅ **正確做法**：
```go
// User + Profile + Settings 是一個聚合
type User struct {
    UserID   string
    Profile  UserProfile   // 組合，不是引用
    Settings UserSettings  // 組合，不是引用
}
```

**判斷標準：如果必須在同一事務中修改，就應該在同一個聚合**

### 8.5 陷阱 2：用技術術語而非業務術語

❌ **錯誤做法**：
```go
type DataProcessor struct {}

func (p *DataProcessor) ProcessData(input DataDTO) (OutputDTO, error) {
    // 業務邏輯，但看不出在做什麼
}
```

✅ **正確做法**：
```go
type PointsCalculationService struct {}

func (s *PointsCalculationService) RecalculateUserPoints(userID string) (int, error) {
    // 清晰的業務語言
}
```

### 8.6 陷阱 3：讓性能問題驅動領域模型設計

❌ **錯誤思維**：
> 「因為更新積分很頻繁，會鎖住 User 表，所以應該把 PointsAccount 拆分為獨立聚合。」

✅ **正確思維**：
> 「性能問題應該在基礎設施層解決（樂觀鎖、讀寫分離、批次處理），不應該改變領域模型。」

**記住 Clean Architecture 原則：業務規則應該獨立於數據庫**

---

## 9. DDD 與 Clean Architecture

### 9.1 兩者的關係

**DDD** 關注「設計什麼」（業務建模）
**Clean Architecture** 關注「如何組織」（代碼架構）

它們是互補的：
- DDD 告訴你如何劃分業務領域、設計聚合
- Clean Architecture 告訴你如何分層、如何管理依賴

### 9.2 層次對應

| Clean Architecture 層 | DDD 概念 | 職責 |
|----------------------|---------|------|
| **Entities** | Entity, Value Object, Aggregate | 業務規則 |
| **Use Cases** | Application Service, Domain Service | 業務流程 |
| **Interface Adapters** | Repository Implementation, HTTP Handler | 適配外部世界 |
| **Frameworks & Drivers** | GORM, Gin, LINE SDK | 框架和工具 |

### 9.3 依賴方向

```
┌─────────────────────────────────────────────────┐
│ Frameworks & Drivers（最外層）                   │
│ ├── GORM                                        │
│ ├── Gin HTTP Framework                         │
│ └── LINE Bot SDK                               │
└─────────────────────────────────────────────────┘
    ↑ 依賴方向（向內）
┌─────────────────────────────────────────────────┐
│ Interface Adapters（接口適配層）                 │
│ ├── GormUserRepository（實現 Repository 接口）   │
│ ├── GinHTTPHandler                             │
│ └── LineWebhookAdapter                         │
└─────────────────────────────────────────────────┘
    ↑ 依賴方向（向內）
┌─────────────────────────────────────────────────┐
│ Use Cases（應用層）                              │
│ ├── RegisterUserUseCase                        │
│ ├── RedeemRewardUseCase                        │
│ └── CalculatePointsUseCase                     │
└─────────────────────────────────────────────────┘
    ↑ 依賴方向（向內）
┌─────────────────────────────────────────────────┐
│ Entities（領域層 - 核心）                        │
│ ├── MembershipAccount (Entity)                │
│ ├── PhoneNumber (Value Object)                │
│ ├── PointsCalculationService (Domain Service) │
│ └── UserRepository (Interface)                │
└─────────────────────────────────────────────────┘
```

**關鍵原則：內層不依賴外層，外層依賴內層**

---

## 10. Go 語言實現範例

### 10.1 完整範例：會員積分系統

#### 領域層（Domain Layer）

```go
// domain/membership/account.go
package membership

import (
    "errors"
    "time"
)

// MembershipAccount - 會員帳戶 Aggregate Root
type MembershipAccount struct {
    lineUserID   string
    displayName  string
    phone        PhoneNumber
    earnedPoints int
    usedPoints   int
    createdAt    time.Time
    updatedAt    time.Time
}

// 工廠方法
func NewMembershipAccount(lineUserID, displayName string, phone PhoneNumber) (*MembershipAccount, error) {
    if lineUserID == "" {
        return nil, errors.New("line_user_id_required")
    }

    return &MembershipAccount{
        lineUserID:   lineUserID,
        displayName:  displayName,
        phone:        phone,
        earnedPoints: 0,
        usedPoints:   0,
        createdAt:    time.Now(),
        updatedAt:    time.Now(),
    }, nil
}

// Getter 方法（封裝私有字段）
func (m *MembershipAccount) LineUserID() string {
    return m.lineUserID
}

func (m *MembershipAccount) EarnedPoints() int {
    return m.earnedPoints
}

func (m *MembershipAccount) UsedPoints() int {
    return m.usedPoints
}

// 業務邏輯：計算可用積分
func (m *MembershipAccount) AvailablePoints() int {
    return m.earnedPoints - m.usedPoints
}

// 業務邏輯：驗證是否可兌換獎品
func (m *MembershipAccount) CanRedeemReward(rewardCost int) bool {
    return m.AvailablePoints() >= rewardCost
}

// 業務邏輯：兌換獎品
func (m *MembershipAccount) RedeemReward(rewardCost int) error {
    if rewardCost <= 0 {
        return errors.New("兌換金額必須大於 0")
    }

    if !m.CanRedeemReward(rewardCost) {
        return ErrInsufficientPoints
    }

    m.usedPoints += rewardCost
    m.updatedAt = time.Now()
    return nil
}

// 業務邏輯：增加累積積分
func (m *MembershipAccount) AddEarnedPoints(points int) error {
    if points <= 0 {
        return errors.New("增加的積分必須大於 0")
    }

    m.earnedPoints += points
    m.updatedAt = time.Now()
    return nil
}

// 業務邏輯：驗證實體有效性
func (m *MembershipAccount) Validate() error {
    if m.lineUserID == "" {
        return errors.New("line_user_id_required")
    }
    if m.usedPoints > m.earnedPoints {
        return errors.New("used_points_exceeds_earned_points")
    }
    return nil
}

var ErrInsufficientPoints = errors.New("積分不足")
```

```go
// domain/membership/phone_number.go
package membership

import (
    "errors"
    "fmt"
    "strings"
)

// PhoneNumber - 台灣手機號碼值對象
type PhoneNumber struct {
    number string
}

// 工廠方法
func NewPhoneNumber(number string) (PhoneNumber, error) {
    // 移除空格和連字號
    cleaned := strings.ReplaceAll(number, " ", "")
    cleaned = strings.ReplaceAll(cleaned, "-", "")

    // 驗證格式
    if len(cleaned) != 10 {
        return PhoneNumber{}, errors.New("手機號碼必須是 10 位數")
    }
    if !strings.HasPrefix(cleaned, "09") {
        return PhoneNumber{}, errors.New("手機號碼必須以 09 開頭")
    }

    return PhoneNumber{number: cleaned}, nil
}

// String 方法
func (p PhoneNumber) String() string {
    return p.number
}

// FormattedString - 格式化顯示
func (p PhoneNumber) FormattedString() string {
    return fmt.Sprintf("%s-%s-%s",
        p.number[0:4],
        p.number[4:7],
        p.number[7:10])
}

// Equals - 相等性判斷
func (p PhoneNumber) Equals(other PhoneNumber) bool {
    return p.number == other.number
}
```

```go
// domain/membership/repository.go
package membership

import "context"

// Repository 接口（領域層定義，基礎設施層實現）
type AccountRepository interface {
    Save(ctx context.Context, account *MembershipAccount) error
    FindByLineUserID(ctx context.Context, lineUserID string) (*MembershipAccount, error)
    FindByPhone(ctx context.Context, phone PhoneNumber) (*MembershipAccount, error)
}
```

#### 應用層（Application Layer）

```go
// application/membership/register_user.go
package membership

import (
    "context"
    "errors"

    "your-project/domain/membership"
)

// RegisterUserUseCase - 註冊用戶
type RegisterUserUseCase struct {
    accountRepo membership.AccountRepository
}

func NewRegisterUserUseCase(accountRepo membership.AccountRepository) *RegisterUserUseCase {
    return &RegisterUserUseCase{
        accountRepo: accountRepo,
    }
}

// Execute - 執行用例
func (uc *RegisterUserUseCase) Execute(ctx context.Context, req RegisterUserRequest) error {
    // 驗證手機號碼格式
    phone, err := membership.NewPhoneNumber(req.Phone)
    if err != nil {
        return err
    }

    // 檢查手機號碼是否已註冊
    existing, _ := uc.accountRepo.FindByPhone(ctx, phone)
    if existing != nil {
        return errors.New("手機號碼已註冊")
    }

    // 創建新用戶（使用工廠方法）
    account, err := membership.NewMembershipAccount(req.LineUserID, req.DisplayName, phone)
    if err != nil {
        return err
    }

    // 保存到數據庫
    return uc.accountRepo.Save(ctx, account)
}

type RegisterUserRequest struct {
    LineUserID  string
    DisplayName string
    Phone       string
}
```

#### 基礎設施層（Infrastructure Layer）

```go
// infrastructure/persistence/gorm_account_repository.go
package persistence

import (
    "context"

    "gorm.io/gorm"
    "your-project/domain/membership"
)

// GORM 模型（數據層）
type MembershipAccountModel struct {
    LineUserID   string `gorm:"primaryKey;column:line_user_id"`
    DisplayName  string `gorm:"column:display_name"`
    Phone        string `gorm:"column:phone;uniqueIndex"`
    EarnedPoints int    `gorm:"column:earned_points;default:0"`
    UsedPoints   int    `gorm:"column:used_points;default:0"`
    CreatedAt    time.Time `gorm:"column:created_at"`
    UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (MembershipAccountModel) TableName() string {
    return "users"
}

// Repository 實現
type GormAccountRepository struct {
    db *gorm.DB
}

func NewGormAccountRepository(db *gorm.DB) *GormAccountRepository {
    return &GormAccountRepository{db: db}
}

// Save - 保存用戶
func (r *GormAccountRepository) Save(ctx context.Context, account *membership.MembershipAccount) error {
    model := &MembershipAccountModel{
        LineUserID:   account.LineUserID(),
        DisplayName:  account.DisplayName(),
        Phone:        account.Phone().String(),
        EarnedPoints: account.EarnedPoints(),
        UsedPoints:   account.UsedPoints(),
        CreatedAt:    account.CreatedAt(),
        UpdatedAt:    account.UpdatedAt(),
    }

    return r.db.WithContext(ctx).Save(model).Error
}

// FindByLineUserID - 根據 LINE User ID 查找用戶
func (r *GormAccountRepository) FindByLineUserID(ctx context.Context, lineUserID string) (*membership.MembershipAccount, error) {
    var model MembershipAccountModel
    err := r.db.WithContext(ctx).Where("line_user_id = ?", lineUserID).First(&model).Error
    if err != nil {
        return nil, err
    }

    // 轉換為領域模型
    phone, _ := membership.NewPhoneNumber(model.Phone)
    account, _ := membership.NewMembershipAccount(
        model.LineUserID,
        model.DisplayName,
        phone,
    )

    // 恢復狀態（使用反射或提供 Setter 方法）
    account.SetEarnedPoints(model.EarnedPoints)
    account.SetUsedPoints(model.UsedPoints)

    return account, nil
}
```

#### 接口層（Interface Layer）

```go
// interfaces/http/gin_handler.go
package http

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "your-project/application/membership"
)

type MembershipHandler struct {
    registerUseCase *membership.RegisterUserUseCase
}

func NewMembershipHandler(registerUseCase *membership.RegisterUserUseCase) *MembershipHandler {
    return &MembershipHandler{
        registerUseCase: registerUseCase,
    }
}

// RegisterUser - 註冊用戶 API
func (h *MembershipHandler) RegisterUser(c *gin.Context) {
    var req RegisterUserHTTPRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 調用 Use Case
    err := h.registerUseCase.Execute(c.Request.Context(), membership.RegisterUserRequest{
        LineUserID:  req.LineUserID,
        DisplayName: req.DisplayName,
        Phone:       req.Phone,
    })

    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "註冊成功"})
}

type RegisterUserHTTPRequest struct {
    LineUserID  string `json:"line_user_id" binding:"required"`
    DisplayName string `json:"display_name" binding:"required"`
    Phone       string `json:"phone" binding:"required"`
}
```

---

## 總結

DDD 是一套強大的軟體設計方法論，但需要正確理解和應用：

**適合使用 DDD 的場景：**
- ✅ 業務邏輯複雜
- ✅ 業務規則頻繁變更
- ✅ 需要長期維護和演化
- ✅ 團隊有一定規模（4+ 人）

**DDD 的核心價值：**
1. 統一語言（Ubiquitous Language）- 降低溝通成本
2. 限界上下文（Bounded Context）- 控制複雜度
3. 聚合（Aggregate）- 保證業務規則一致性
4. 領域事件（Domain Event）- 實現鬆耦合

**記住：**
> DDD 不是銀彈。簡單的 CRUD 系統不需要 DDD。但對於複雜的業務系統，DDD 能幫助你建立清晰的模型，讓代碼更容易理解和維護。

**參考資料：**
- Eric Evans - "Domain-Driven Design: Tackling Complexity in the Heart of Software"
- Vaughn Vernon - "Implementing Domain-Driven Design"
- Martin Fowler - "Patterns of Enterprise Application Architecture"
