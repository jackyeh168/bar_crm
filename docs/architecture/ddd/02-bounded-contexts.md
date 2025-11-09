# 限界上下文劃分

> **版本**: 1.0
> **最後更新**: 2025-01-08

---

## **3.1 上下文概覽**

本系統劃分為 **8 個限界上下文**，遵循中粒度原則：

```
┌─────────────────────────────────────────────────────────┐
│                     餐廳會員管理系統                        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  會員管理    │  │  積分管理 ⭐  │  │  發票處理     │  │
│  │  (支撐域)    │  │  (核心域)    │  │  (核心域)     │  │
│  └─────────────┘  └──────────────┘  └──────────────┘  │
│                                                         │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  問卷管理    │  │  外部系統整合 │  │  身份與訪問   │  │
│  │  (支撐域)    │  │  (支撐域)    │  │  (通用域)     │  │
│  └─────────────┘  └──────────────┘  └──────────────┘  │
│                                                         │
│  ┌─────────────┐  ┌──────────────┐                    │
│  │  通知服務    │  │  稽核追蹤    │                    │
│  │  (通用域)    │  │  (支撐域)    │                    │
│  └─────────────┘  └──────────────┘                    │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## **3.2 會員管理上下文 (Member Management Context)**

**領域類型**: 支撐域
**職責**: 管理會員註冊、手機號碼綁定、基本資料維護

### **統一語言 (Ubiquitous Language)**
- **會員 (Member)**: 已完成註冊並綁定手機號碼的 LINE 用戶
- **LINE 用戶 ID**: LINE Platform 分配的唯一識別碼
- **手機號碼綁定**: 將 LINE 帳號與台灣手機號碼關聯的過程
- **註冊狀態**: 會員的註冊完成狀態 (已註冊/未註冊)

### **聚合設計**

**聚合: Member** (聚合根)
```
Member (Aggregate Root)
├── MemberID (Entity ID - Value Object)
├── LineUserID (Value Object) - 唯一
├── DisplayName (Value Object)
├── PhoneNumber (Value Object) - 唯一
├── RegistrationDate (Value Object)
└── IsActive (Value Object)
```

**值對象**:
- `MemberID`: UUID 格式的會員識別碼
- `LineUserID`: LINE Platform 的用戶 ID (以 "U" 開頭)
- `PhoneNumber`: 台灣手機號碼 (10 位數，09 開頭)
- `DisplayName`: LINE 用戶的顯示名稱

### **聚合方法（接口定義）**

**Member 聚合根方法**:
```go
// 構造方法
NewMember(lineUserID LineUserID, displayName DisplayName) (*Member, error)

// 命令方法
BindPhoneNumber(phoneNumber PhoneNumber) error
  // 前置條件: phoneNumber 已通過唯一性檢查
  // 後置條件: PhoneNumber 已綁定，IsActive = true
  // 不變性保護: 每個 Member 只能綁定一個 PhoneNumber

UpdateDisplayName(displayName DisplayName) error
  // 更新顯示名稱

Deactivate() error
  // 停用會員帳號
  // 後置條件: IsActive = false

// 查詢方法
IsRegistered() bool
  // 返回: PhoneNumber 是否已綁定

GetLineUserID() LineUserID
GetPhoneNumber() PhoneNumber
GetDisplayName() DisplayName
```

### **領域服務**
- `MemberRegistrationService`: 處理會員註冊流程，驗證手機號碼唯一性
- `PhoneNumberBindingService`: 處理手機號碼綁定邏輯

### **倉儲接口**

**MemberRepository** (會員聚合持久化接口):
```go
Create(member *Member) error
  // 返回: ErrMemberAlreadyExists 如果 LineUserID 重複

Update(member *Member) error
  // 返回: ErrMemberNotFound 如果會員不存在

FindByID(memberID MemberID) (*Member, error)
  // 返回: ErrMemberNotFound 如果不存在

FindByLineUserID(lineUserID LineUserID) (*Member, error)
  // 返回: ErrMemberNotFound 如果不存在

FindByPhoneNumber(phoneNumber PhoneNumber) (*Member, error)
  // 返回: ErrMemberNotFound 如果不存在

ExistsByLineUserID(lineUserID LineUserID) (bool, error)
ExistsByPhoneNumber(phoneNumber PhoneNumber) (bool, error)

Delete(memberID MemberID) error
  // 軟刪除: IsActive = false
  // 返回: ErrMemberNotFound 如果不存在
```

### **領域事件**
- `MemberRegistered`: 會員完成註冊
- `PhoneNumberBound`: 手機號碼綁定成功
- `MemberProfileUpdated`: 會員資料更新

### **領域錯誤定義**

```go
// 會員管理上下文錯誤
var (
    ErrMemberNotFound           error = "Member not found"
    ErrMemberAlreadyExists      error = "Member with this LineUserID already exists"
    ErrPhoneNumberAlreadyBound  error = "This phone number is already bound to another member"
    ErrInvalidPhoneNumberFormat error = "Phone number must be 10 digits starting with 09"
    ErrInvalidLineUserID        error = "LineUserID must start with 'U'"
    ErrMemberNotRegistered      error = "Member has not completed registration"
    ErrMemberDeactivated        error = "Member account is deactivated"
)
```

---

## **3.3 積分管理上下文 (Points Management Context)** ⭐

**領域類型**: 核心域
**職責**: 管理積分賺取、查詢、使用、轉換規則、積分重算

### **統一語言 (Ubiquitous Language)**
- **積分帳戶 (Points Account)**: 會員的積分錢包，記錄累積積分與可用積分
- **累積積分 (Earned Points)**: 會員所有已驗證交易獲得的積分總和
- **可用積分 (Available Points)**: 累積積分 - 已使用積分
- **轉換率 (Conversion Rate)**: 消費金額轉換為積分的比率 (例: 100 元 = 1 點)
- **轉換規則 (Conversion Rule)**: 定義特定日期範圍內的轉換率
- **基礎積分 (Base Points)**: 根據消費金額與轉換率計算的積分
- **問卷獎勵積分 (Survey Bonus Points)**: 完成問卷獲得的額外積分 (+1 點)
- **積分交易 (Points Transaction)**: 積分變動記錄 (賺取/扣除)

### **聚合設計**

**聚合 1: PointsAccount** (聚合根 - 輕量級設計)

**❌ 錯誤設計 - God Aggregate (避免此設計)**:
```go
// 反模式：包含無界集合導致性能災難
type PointsAccount struct {
    accountID    AccountID
    memberID     MemberID
    earnedPoints PointsAmount
    usedPoints   PointsAmount
    transactions []*PointsTransaction // ❌ 可能有 10,000+ 筆交易
}

// 問題:
// - 每次加載聚合都要加載所有交易 → 內存爆炸
// - 並發沖突頻繁（多個操作爭奪同一聚合鎖）
// - 無法支持分頁查詢交易歷史
```

**✅ 正確設計 - 輕量級聚合**:
```go
// 聚合只保留必須強一致性的狀態
type PointsAccount struct {
    accountID    AccountID
    memberID     MemberID
    earnedPoints PointsAmount  // 累積積分（權威數據源）
    usedPoints   PointsAmount  // 已使用積分（權威數據源）
    lastUpdatedAt time.Time
    version      int           // 樂觀鎖版本號
    events       []DomainEvent // 領域事件（待發布）
    // transactions 不在聚合內 ✅
}

// 計算屬性
func (a *PointsAccount) GetAvailablePoints() PointsAmount {
    return a.earnedPoints - a.usedPoints
}

// 優勢:
// ✅ 快速加載（只包含必要狀態）
// ✅ 低內存開銷（~100 bytes per aggregate）
// ✅ 支持高效批量操作（積分重算）
// ✅ 並發沖突少（狀態更新頻率低）
```

**獨立實體: PointsTransaction** (不屬於任何聚合)

```go
// PointsTransaction 作為獨立實體存儲
type PointsTransaction struct {
    transactionID TransactionID
    accountID     AccountID     // 引用，非所有權
    txType        TransactionType // Earned/Deducted
    amount        PointsAmount
    source        PointsSource  // Invoice/Survey/Redemption/etc.
    sourceID      string        // 來源記錄 ID
    description   string
    createdAt     time.Time
}

// 通過獨立倉儲管理
type PointsTransactionRepository interface {
    Create(ctx Context, tx *PointsTransaction) error
    FindByAccountID(ctx Context, accountID AccountID, pagination Pagination) ([]*PointsTransaction, error)
    FindByDateRange(ctx Context, accountID AccountID, start, end time.Time) ([]*PointsTransaction, error)
}

// 優勢:
// ✅ 分頁查詢（不影響聚合加載）
// ✅ 獨立擴展（可選 Event Sourcing 存儲）
// ✅ 審計日誌（不可變記錄）
```

**設計原則總結**:
- ✅ **聚合邊界 = 事務邊界**: 一個事務只修改一個聚合
- ✅ **小型聚合**: 只包含必須在一個事務內保持一致的數據
- ✅ **按需查詢**: 交易歷史通過 Repository 獨立查詢
- ✅ **並發控制**: 使用樂觀鎖 (version) 處理並發更新

**聚合 2: ConversionRule** (聚合根 - 狀態機設計)
```
ConversionRule (Aggregate Root)
├── RuleID (Entity ID - Value Object)
├── DateRange (Value Object)
│   ├── StartDate
│   └── EndDate
├── ConversionRate (Value Object) - 轉換率 (1-1000)
├── Status (Value Object) - 明確的狀態機
│   ├── Draft (草稿)
│   ├── Active (啟用)
│   └── Inactive (停用)
├── CreatedAt (Value Object)
└── UpdatedAt (Value Object)

設計原則:
- ✅ 明確的狀態轉換規則（狀態機模式）
- ✅ 防止無效的狀態轉換
- ✅ 冪等性操作（可重複調用）
```

**值對象**:
- `PointsAmount`: 積分數量，必須 >= 0
- `ConversionRate`: 轉換率，範圍 1-1000
- `DateRange`: 日期範圍，StartDate <= EndDate，不可與其他規則重疊
- `PointsTransactionType`: 枚舉 (Earned, Deducted)
- `PointsSource`: 枚舉 (Invoice, Survey, Redemption, Expiration, Transfer)
- `RuleStatus`: 枚舉值對象，明確的狀態機
  ```go
  type RuleStatus int

  const (
      RuleStatusDraft    RuleStatus = iota  // 草稿狀態
      RuleStatusActive                      // 啟用狀態
      RuleStatusInactive                    // 停用狀態
  )

  // 狀態轉換規則:
  // Draft    → Active   ✅ (啟用規則)
  // Draft    → Inactive ❌ (草稿不能直接停用)
  // Active   → Inactive ✅ (停用規則)
  // Active   → Draft    ❌ (已啟用不能回到草稿)
  // Inactive → Active   ✅ (重新啟用)
  // Inactive → Draft    ❌ (已停用不能回到草稿)
  ```

**跨上下文數據傳遞（見 Application Layer DTO）**:

積分計算需要來自 Invoice Context 的交易數據，但不應直接依賴 Invoice 實體。
解決方案：Application Layer 使用 DTO（Data Transfer Object）作為數據載體。

DTO 設計原則：
- **定義位置**: Application Layer（`internal/application/dto/`）
- **用途**: 跨上下文數據傳輸，避免聚合間直接引用
- **特性**: 無業務邏輯，僅包含數據字段
- **轉換責任**: Application Layer 負責 Entity → DTO → Entity 轉換

相關設計模式詳見：
- Chapter 7 (7.4.1): DTO 模式完整說明
- Chapter 11 (11.4): 依賴規則與 DTO 使用

### **聚合方法（接口定義）**

**PointsAccount 聚合根方法**:
```go
// 構造方法
NewPointsAccount(memberID MemberID) (*PointsAccount, error)

// 命令方法
EarnPoints(amount PointsAmount, source PointsSource, sourceID string, description string) error
DeductPoints(amount PointsAmount, reason string) error
RecalculatePoints(transactions []VerifiedTransactionDTO, calculator PointsCalculationService) error

// 查詢方法
GetAvailablePoints() PointsAmount  // EarnedPoints - UsedPoints
GetEarnedPoints() PointsAmount
GetUsedPoints() PointsAmount
GetAccountID() AccountID
GetMemberID() MemberID
GetLastUpdatedAt() time.Time
```

**ConversionRule 聚合根方法**:
```go
// 構造方法
NewConversionRule(dateRange DateRange, conversionRate ConversionRate) (*ConversionRule, error)

// 命令方法（狀態轉換 - 詳見狀態機圖）
Activate() error    // Draft/Inactive → Active
Deactivate() error  // Active → Inactive

// 命令方法（資料修改 - 僅允許 Draft 狀態）
UpdateDateRange(newDateRange DateRange) error
UpdateConversionRate(newRate ConversionRate) error

// 查詢方法
IsActiveForDate(date Date) bool
GetDateRange() DateRange
GetConversionRate() ConversionRate
GetStatus() RuleStatus
IsActive() bool
IsDraft() bool
IsInactive() bool
```

**狀態機圖**:
```
        +-------+
        | Draft |
        +-------+
            |
            | Activate()
            v
        +--------+        Deactivate()        +----------+
        | Active | <------------------------- | Inactive |
        +--------+                            +----------+
            |                                      ^
            | Deactivate()                         |
            +--------------------------------------+

            (冪等操作: Active → Active, Inactive → Inactive)
```

**設計優勢**:
- ✅ **明確的狀態轉換**: 使用 switch 語句清楚定義每種轉換
- ✅ **防止無效操作**: 草稿不能直接停用，已啟用不能修改
- ✅ **冪等性**: `Activate()` 和 `Deactivate()` 可重複調用
- ✅ **不變性保護**: 啟用後的規則不可修改（需先停用才能改）
- ✅ **明確的錯誤訊息**: `ErrCannotDeactivateDraftRule`, `ErrCannotModifyActiveRule`
```

### **領域服務（接口定義）**

**PointsCalculationService** (核心業務邏輯 - Strategy Pattern):
```go
// 主接口 - 計算單筆交易的總積分
CalculateForTransaction(dto VerifiedTransactionDTO, ruleService ConversionRuleService) PointsAmount
  // 計算單筆交易的總積分
  // 參數:
  //   - dto: 已驗證交易的 DTO（包含 Amount, InvoiceDate, SurveySubmitted）
  //   - ruleService: 用於查詢轉換規則
  // 返回: 該交易的總積分
  // 設計: 使用 Composite Pattern 組合多個計算策略
  //   - BasePointsCalculator: 計算基礎積分 (floor(amount / rate))
  //   - SurveyBonusCalculator: 計算問卷獎勵 (+1 或 0)
  //   - [V3.4+] TierBonusCalculator: 會員等級加成
  //   - [V4.0+] PromotionalBonusCalculator: 促銷活動加成
  // 注意: 使用 DTO 作為參數，避免依賴其他上下文的實體
```

**PointsCalculationStrategy** (策略接口 - Open/Closed Principle):
```go
// 個別積分計算策略
Calculate(dto VerifiedTransactionDTO, ruleService ConversionRuleService) PointsAmount
  // 計算特定類型的積分
  // 參數: dto - 交易數據傳輸對象
  // 實現類別:
  //   - BasePointsCalculator: 基礎積分
  //   - SurveyBonusCalculator: 問卷獎勵
  //   - [未來擴展] 新增策略無需修改現有代碼
```

**CompositePointsCalculator** (組合實現):
```go
// 實現 PointsCalculationService
type CompositePointsCalculator struct {
    strategies []PointsCalculationStrategy
}

func (c *CompositePointsCalculator) CalculateForTransaction(
    dto VerifiedTransactionDTO,
    ruleService ConversionRuleService,
) PointsAmount {
    total := 0
    for _, strategy := range c.strategies {
        total += strategy.Calculate(dto, ruleService)
    }
    return total
}

// V3.1 配置: strategies = [BasePointsCalculator, SurveyBonusCalculator]
// V3.4 配置: strategies = [..., TierBonusCalculator] // 新增策略，無需修改現有代碼
```

**PointsRecalculationService** (積分重算服務):
```go
RecalculateAllAccounts(db Transaction) error
  // 重算所有會員的累積積分
  // 前置條件: 必須在資料庫事務中執行
  // 流程:
  //   1. 對每個會員查詢所有已驗證交易
  //   2. 對每筆交易重新計算積分（根據 InvoiceDate 查詢規則）
  //   3. 更新 PointsAccount.EarnedPoints
  // 性能: 鎖表 30-60 秒
  // 返回: 錯誤時回滾整個事務

RecalculateMemberAccount(memberID MemberID, db Transaction) error
  // 重算單一會員的累積積分
  // 前置條件: 必須在資料庫事務中執行
  // 流程: 同上，僅針對單一會員
```

**ConversionRuleValidationService**:
```go
ValidateRuleDateRange(dateRange DateRange, excludeRuleID RuleID) error
ValidateConversionRate(rate ConversionRate) error
GetRuleForDate(date Date) (*ConversionRule, error)
```

### **倉儲接口**

**PointsAccountRepository**:
```go
Create(account *PointsAccount) error
Update(account *PointsAccount) error
FindByID(accountID AccountID) (*PointsAccount, error)
FindByMemberID(memberID MemberID) (*PointsAccount, error)
FindAll() ([]*PointsAccount, error)
```

**PointsTransactionRepository**:
```go
Create(transaction *PointsTransaction) error
CreateBatch(transactions []*PointsTransaction) error
FindByAccountID(accountID AccountID, limit int, offset int) ([]*PointsTransaction, error)
FindByAccountIDAndDateRange(accountID AccountID, startDate Date, endDate Date) ([]*PointsTransaction, error)
FindBySource(accountID AccountID, source PointsSource) ([]*PointsTransaction, error)
CountByAccountID(accountID AccountID) (int, error)
DeleteByAccountID(accountID AccountID) error
```

**ConversionRuleRepository**:
```go
Create(rule *ConversionRule) error
Update(rule *ConversionRule) error
FindByID(ruleID RuleID) (*ConversionRule, error)
FindByDate(date Date) (*ConversionRule, error)
FindOverlapping(dateRange DateRange, excludeRuleID RuleID) ([]*ConversionRule, error)
FindAll() ([]*ConversionRule, error)
FindActive() ([]*ConversionRule, error)
Delete(ruleID RuleID) error
```

### **領域事件**
- `PointsEarned`: 積分已獲得 (交易驗證或問卷完成)
- `PointsDeducted`: 積分已扣除 (V3.2+ 兌換)
- `ConversionRuleCreated`: 轉換規則已創建
- `ConversionRuleUpdated`: 轉換規則已更新
- `ConversionRuleDeleted`: 轉換規則已刪除
- `PointsRecalculationRequested`: 積分重算已請求 (管理員觸發)
- `PointsRecalculated`: 積分已重算 (系統完成)

### **領域錯誤定義**

```go
// 積分管理上下文錯誤（核心域 ⭐）
var (
    // PointsAccount 相關錯誤
    ErrAccountNotFound         error = "Points account not found"
    ErrAccountAlreadyExists    error = "Points account already exists for this member"
    ErrInsufficientPoints      error = "Insufficient points for this operation"
    ErrNegativePointsAmount    error = "Points amount cannot be negative"
    ErrInvalidPointsSource     error = "Invalid points source"

    // ConversionRule 相關錯誤
    ErrRuleNotFound             error = "Conversion rule not found"
    ErrRuleAlreadyExists        error = "Conversion rule already exists"
    ErrNoConversionRuleForDate  error = "No conversion rule found for the specified date"
    ErrDateRangeOverlap         error = "Date range overlaps with existing conversion rule"
    ErrInvalidDateRange         error = "Invalid date range: start date must be before or equal to end date"
    ErrInvalidConversionRate    error = "Conversion rate must be between 1 and 1000"

    // ConversionRule 狀態機錯誤
    ErrInvalidStatusTransition   error = "Invalid rule status transition"
    ErrCannotDeactivateDraftRule error = "Cannot deactivate a draft rule: activate it first"
    ErrCannotModifyActiveRule    error = "Cannot modify an active rule: deactivate it first"
    ErrRuleAlreadyActive         error = "Rule is already active"
    ErrRuleAlreadyInactive       error = "Rule is already inactive"

    // 積分計算相關錯誤
    ErrInvalidAmount            error = "Amount must be greater than 0"
    ErrRecalculationFailed      error = "Points recalculation failed"
    ErrRecalculationInProgress  error = "Points recalculation is already in progress"
)
```

---

## **3.4 發票處理上下文 (Invoice Processing Context)**

**領域類型**: 核心域
**職責**: 處理發票掃描、解析、驗證、交易記錄管理

### **統一語言 (Ubiquitous Language)**
- **發票 (Invoice)**: 台灣電子發票，包含發票號碼、日期、金額
- **QR Code 掃描 (QR Code Scan)**: 用戶上傳發票 QR Code 照片的過程
- **發票驗證 (Invoice Validation)**: 檢查發票有效期、重複性、格式
- **交易 (Transaction)**: 會員掃描發票後創建的消費記錄
- **交易狀態**: imported (待驗證) / verified (已驗證) / failed (已失敗)
- **發票號碼**: 10 位英數字組合
- **發票日期**: 發票開立日期 (ROC 民國年格式: yyyMMdd)
- **有效期**: 發票開立日期起 60 天內

### **聚合設計**

**聚合: InvoiceTransaction** (聚合根)
```
InvoiceTransaction (Aggregate Root)
├── TransactionID (Entity ID - Value Object)
├── MemberID (Reference to Member Context)
├── Invoice (Value Object)
│   ├── InvoiceNumber (Value Object) - 唯一
│   ├── InvoiceDate (Value Object)
│   ├── Amount (Money Value Object)
│   └── TaxID (Value Object - optional)
├── Status (Value Object) - imported/verified/failed
├── QRCodeData (Value Object)
├── SurveySubmitted (Value Object) - boolean
├── SurveyID (Reference to Survey Context - optional)
├── VerifiedAt (Value Object - optional)
├── CreatedAt (Value Object)
└── UpdatedAt (Value Object)
```

**值對象**:
- `InvoiceNumber`: 10 位英數字，格式驗證
- `InvoiceDate`: 日期，格式 yyyMMdd (ROC 年份)
- `Money`: 金額，必須 > 0
- `TransactionStatus`: 枚舉 (Imported, Verified, Failed)
- `QRCodeData`: 發票 QR Code 原始資料

### **聚合方法（接口定義）**

**InvoiceTransaction 聚合根方法**:
```go
// 構造方法
NewInvoiceTransaction(memberID MemberID, invoice Invoice, qrCodeData QRCodeData) (*InvoiceTransaction, error)
  // 創建新交易（初始狀態: imported）
  // 前置條件: invoice 已通過驗證（有效期、格式）
  // 前置條件: invoice.InvoiceNumber 唯一性已檢查
  // 後置條件: Status = Imported, CreatedAt = 現在時間

// 命令方法（狀態變更）
VerifyTransaction(verifiedAt time.Time) error
  // 驗證交易（iChef 匹配成功）
  // 前置條件: Status == Imported
  // 後置條件: Status = Verified, VerifiedAt = 指定時間
  // 副作用: 發布 TransactionVerified 事件 → 觸發積分計算
  // 返回: ErrInvalidStatusTransition 如果當前狀態不是 Imported

FailTransaction(reason string) error
  // 標記交易為失敗
  // 前置條件: Status == Imported
  // 後置條件: Status = Failed
  // 返回: ErrInvalidStatusTransition 如果當前狀態不是 Imported

LinkSurvey(surveyID SurveyID) error
  // 關聯問卷到交易
  // 前置條件: SurveyID 存在且啟用
  // 後置條件: SurveyID 已設置

MarkSurveySubmitted() error
  // 標記問卷已提交
  // 前置條件: SurveyID 不為 nil
  // 後置條件: SurveySubmitted = true
  // 副作用: 如果 Status == Verified，發布 SurveyRewardGranted 事件 → 觸發積分重算
  // 返回: ErrSurveyNotLinked 如果 SurveyID 為 nil

VoidInvoice(reason string) error
  // 作廢發票（V3.2+）
  // 前置條件: Status == Verified
  // 後置條件: Status = Failed
  // 副作用: 需要扣回已獲得的積分

// 查詢方法
GetInvoice() Invoice
GetStatus() TransactionStatus
GetMemberID() MemberID
IsSurveySubmitted() bool
IsVerified() bool
  // 返回: Status == Verified

CanVerify() bool
  // 返回: Status == Imported

GetSurveyID() (SurveyID, bool)
  // 返回: SurveyID 和是否存在
```

**InvoiceParsingService** (發票解析服務):
```go
ParseQRCode(qrCodeData string) (*Invoice, error)
  // 解析台灣電子發票 QR Code
  // QR Code 格式: 發票號碼(10碼)|日期(7碼 yyyMMdd)|金額|...
  // 返回: ErrInvalidQRCodeFormat 如果格式錯誤
  // 返回: ErrInvoiceParsingFailed 如果解析失敗

ExtractInvoiceNumber(qrCodeData string) (InvoiceNumber, error)
ExtractInvoiceDate(qrCodeData string) (InvoiceDate, error)
ExtractAmount(qrCodeData string) (Money, error)
```

**InvoiceValidationService** (發票驗證服務):
```go
ValidateInvoice(invoice Invoice) error
  // 綜合驗證發票
  // 檢查: 格式、有效期、重複性
  // 返回: ErrInvoiceDuplicate, ErrInvoiceExpired, ErrInvalidInvoiceFormat

CheckDuplicate(invoiceNumber InvoiceNumber) (bool, error)
  // 檢查發票號碼是否已存在
  // 返回: true 如果重複

CheckExpiry(invoiceDate InvoiceDate) bool
  // 檢查發票是否在有效期內（60 天）
  // 返回: true 如果已過期

ValidateInvoiceNumber(invoiceNumber InvoiceNumber) error
  // 驗證發票號碼格式（10 位英數字）
  // 返回: ErrInvalidInvoiceNumber

ValidateInvoiceDate(invoiceDate InvoiceDate) error
  // 驗證發票日期格式（yyyMMdd）
  // 返回: ErrInvalidInvoiceDate
```

**TransactionVerificationService** (交易驗證服務):
```go
VerifyTransaction(transactionID TransactionID) error
  // 驗證交易（iChef 匹配後調用）
  // 流程:
  //   1. 查詢交易
  //   2. 檢查狀態是否為 Imported
  //   3. 調用 transaction.VerifyTransaction()
  //   4. 保存交易
  //   5. 發布 TransactionVerified 事件
  // 返回: ErrTransactionNotFound, ErrInvalidStatusTransition
```

### **倉儲接口**

**InvoiceTransactionRepository**:
```go
Create(transaction *InvoiceTransaction) error
  // 返回: ErrTransactionAlreadyExists, ErrInvoiceDuplicate

Update(transaction *InvoiceTransaction) error
  // 返回: ErrTransactionNotFound

FindByID(transactionID TransactionID) (*InvoiceTransaction, error)
  // 返回: ErrTransactionNotFound

FindByInvoiceNumber(invoiceNumber InvoiceNumber) (*InvoiceTransaction, error)
  // 返回: ErrTransactionNotFound

FindByMemberID(memberID MemberID) ([]*InvoiceTransaction, error)

FindVerifiedByMemberID(memberID MemberID) ([]*InvoiceTransaction, error)
  // 條件: Status == Verified

FindByStatus(status TransactionStatus) ([]*InvoiceTransaction, error)

ExistsByInvoiceNumber(invoiceNumber InvoiceNumber) (bool, error)

FindByInvoiceNumbers(invoiceNumbers []InvoiceNumber) ([]*InvoiceTransaction, error)
```

### **領域事件**
- `InvoiceQRCodeScanned`: 發票 QR Code 已掃描
- `InvoiceParsed`: 發票已解析
- `InvoiceValidated`: 發票已驗證
- `InvoiceRejected`: 發票已拒絕 (重複/過期/無效)
- `TransactionCreated`: 交易已創建
- `TransactionVerified`: 交易已驗證
- `TransactionFailed`: 交易已失敗
- `InvoiceVoided`: 發票已作廢

### **領域錯誤定義**

```go
// 發票處理上下文錯誤（核心域）
var (
    // Transaction 相關錯誤
    ErrTransactionNotFound          error = "Transaction not found"
    ErrTransactionAlreadyExists     error = "Transaction already exists"
    ErrInvalidStatusTransition      error = "Invalid transaction status transition"

    // Invoice 相關錯誤
    ErrInvoiceDuplicate            error = "Invoice with this number already exists"
    ErrInvoiceExpired              error = "Invoice has expired (older than 60 days)"
    ErrInvalidInvoiceNumber        error = "Invalid invoice number format (must be 10 alphanumeric characters)"
    ErrInvalidInvoiceDate          error = "Invalid invoice date format (must be yyyMMdd in ROC calendar)"
    ErrInvalidInvoiceFormat        error = "Invalid invoice format"
    ErrInvalidAmount               error = "Amount must be greater than 0"

    // QR Code 相關錯誤
    ErrInvalidQRCodeFormat         error = "Invalid QR code format"
    ErrQRCodeParsingFailed         error = "Failed to parse QR code data"

    // Survey 相關錯誤
    ErrSurveyNotLinked             error = "Survey is not linked to this transaction"
    ErrSurveyAlreadySubmitted      error = "Survey has already been submitted for this transaction"
    ErrInvalidSurveyID             error = "Invalid survey ID"

    // Verification 相關錯誤
    ErrCannotVerifyTransaction     error = "Cannot verify transaction: status must be 'imported'"
    ErrCannotVoidTransaction       error = "Cannot void transaction: status must be 'verified'"
)
```

---

## **3.5 外部系統整合上下文 (External Integration Context)**

**領域類型**: 支撐域
**職責**: 管理 iChef POS 系統批次匯入、發票匹配、資料同步

### **統一語言 (Ubiquitous Language)**
- **批次匯入 (Batch Import)**: 從 iChef POS 系統匯入發票資料的過程
- **發票匹配 (Invoice Matching)**: 將 iChef 發票與會員掃描記錄配對
- **匹配條件**: 發票號碼、日期、金額三者完全一致
- **匯入狀態**: 處理中 (Processing) / 已完成 (Completed) / 失敗 (Failed)
- **匹配結果**: 已匹配 (Matched) / 未匹配 (Unmatched) / 跳過 (Skipped) / 重複 (Duplicate)

### **聚合設計**

**設計原則**: 遵循 SRP (Single Responsibility Principle)，將批次生命周期、統計計算、記錄處理分離

**聚合 1: ImportBatch** (聚合根 - 批次生命周期管理)
```
ImportBatch (Aggregate Root) - 職責：管理批次狀態和生命周期
├── BatchID (Entity ID - Value Object)
├── FileName (Value Object)
├── ImportedBy (AdminUserID Reference)
├── Status (Value Object) - Processing/Completed/Failed
├── StartedAt (Value Object)
├── CompletedAt (Value Object - nullable)
└── ErrorMessage (Value Object - nullable) - 失敗時的錯誤訊息
```

**值對象: ImportStatistics** (統計計算結果 - 職責：存儲統計數據)
```
ImportStatistics (Value Object) - 不可變
├── BatchID (Reference) - 關聯到批次
├── TotalRows (int)
├── MatchedCount (int) - 成功匹配的數量
├── UnmatchedCount (int) - 未匹配的數量
├── SkippedCount (int) - 跳過的數量（格式錯誤）
└── DuplicateCount (int) - 重複的數量

說明：
- 由 Domain Service 計算後創建
- 不屬於 ImportBatch 聚合內部（避免 God Object）
- 單獨持久化，可獨立查詢
```

**實體: ImportedInvoiceRecord** (獨立實體 - 職責：追蹤單筆發票處理結果)
```
ImportedInvoiceRecord (Entity) - 不屬於 ImportBatch 聚合
├── RecordID (Entity ID - Value Object)
├── BatchID (Reference) - 引用批次（不是聚合內部）
├── InvoiceNumber (Value Object)
├── InvoiceDate (Value Object)
├── Amount (Money Value Object)
├── MatchStatus (Value Object) - Matched/Unmatched/Skipped/Duplicate
├── MatchedTransactionID (TransactionID Reference - nullable)
├── SkipReason (Value Object - nullable) - 跳過原因
└── CreatedAt (Value Object)

說明：
- 與 ImportBatch 是弱關聯（通過 BatchID 引用）
- 可以獨立查詢、分頁加載（避免加載整個集合）
- 職責單一：記錄單筆發票的處理結果
```

**值對象**:
- `ImportStatistics`: 匯入統計資訊（已在上方定義）
- `MatchStatus`: 枚舉 (Matched, Unmatched, Skipped, Duplicate)
- `FileName`: 檔案名稱
- `SkipReason`: 跳過原因（格式錯誤、缺少必填欄位等）
- `ImportStatus`: 枚舉 (Processing, Completed, Failed)

### **領域服務**
- `IChefImportService`: iChef 匯入邏輯
  - `ImportExcelFile(file io.Reader, adminUserID AdminUserID) (*ImportBatch, error)`
- `InvoiceMatchingService`: 發票匹配邏輯
  - `MatchInvoice(invoiceRecord ImportedInvoiceRecord) (TransactionID, bool, error)`
- `DuplicateDetectionService`: 重複檢測邏輯
  - `IsDuplicate(invoiceNumber, date, amount) bool`

### **倉儲接口**
- `ImportBatchRepository`:
  - `FindByID(batchID BatchID) (*ImportBatch, error)`
  - `Save(batch *ImportBatch) error`
  - `FindByDateRange(startDate, endDate Date) ([]*ImportBatch, error)`

### **領域事件**
- `BatchImportStarted`: 批次匯入已開始
- `BatchImportCompleted`: 批次匯入已完成
- `InvoiceMatched`: 發票已匹配 (觸發交易驗證)
- `InvoiceUnmatched`: 發票未匹配

---

## **3.6 問卷管理上下文 (Survey Management Context)**

**領域類型**: 支撐域
**職責**: 管理問卷設計、回應收集、獎勵機制

### **統一語言 (Ubiquitous Language)**
- **問卷 (Survey)**: 餐廳滿意度調查表
- **問題 (Question)**: 問卷中的單一問題
- **問題類型**: 文字題 (Text) / 選擇題 (MultipleChoice) / 評分題 (Rating)
- **問卷回應 (Survey Response)**: 會員填寫問卷的答案集合
- **啟用狀態**: 同時只能有一個問卷處於啟用狀態
- **問卷獎勵**: 完成問卷獲得 +1 點積分

### **聚合設計**

**聚合 1: Survey** (聚合根)
```
Survey (Aggregate Root)
├── SurveyID (Entity ID - Value Object)
├── Title (Value Object)
├── Description (Value Object)
├── IsActive (Value Object)
├── Questions (Entity Collection)
│   └── SurveyQuestion (Entity)
│       ├── QuestionID
│       ├── QuestionText
│       ├── QuestionType (Text/MultipleChoice/Rating)
│       ├── IsRequired
│       ├── DisplayOrder
│       └── Options (for MultipleChoice)
├── CreatedAt (Value Object)
└── UpdatedAt (Value Object)
```

**聚合 2: SurveyResponse** (聚合根)
```
SurveyResponse (Aggregate Root)
├── ResponseID (Entity ID - Value Object)
├── SurveyID (Reference)
├── TransactionID (Reference to Invoice Context)
├── MemberID (Reference to Member Context)
├── Answers (Entity Collection)
│   └── Answer (Entity)
│       ├── AnswerID
│       ├── QuestionID (Reference)
│       ├── AnswerText
│       └── SelectedOptions
├── SubmittedAt (Value Object)
└── RewardGranted (Value Object) - boolean
```

**值對象**:
- `QuestionType`: 枚舉 (Text, MultipleChoice, Rating)
- `RatingScore`: 1-5 星評分

### **領域服務**
- `SurveyActivationService`: 問卷啟用邏輯 (確保單一啟用)
- `SurveyResponseValidationService`: 驗證回應完整性
- `SurveyRewardService`: 發放問卷獎勵

### **倉儲接口**
- `SurveyRepository`:
  - `FindActiveSurvey() (*Survey, error)`
  - `Save(survey *Survey) error`
- `SurveyResponseRepository`:
  - `FindByTransactionID(transactionID TransactionID) (*SurveyResponse, error)`
  - `Save(response *SurveyResponse) error`
  - `ExistsByTransactionID(transactionID TransactionID) bool`

### **領域事件**
- `SurveyCreated`: 問卷已創建
- `SurveyActivated`: 問卷已啟用
- `SurveyDeactivated`: 問卷已停用
- `SurveyResponseSubmitted`: 問卷回應已提交
- `SurveyRewardGranted`: 問卷獎勵已發放 (觸發積分增加)

---

## **3.7 身份與訪問上下文 (Identity & Access Context)**

**領域類型**: 通用域
**職責**: 管理後台認證、授權、RBAC

### **統一語言 (Ubiquitous Language)**
- **管理員 (Admin User)**: 使用管理後台的用戶
- **角色 (Role)**: Admin / User / Guest
- **權限 (Permission)**: 對特定資源的操作權限
- **訪問令牌 (Access Token)**: JWT 令牌

### **聚合設計**

**聚合: AdminUser** (聚合根)
```
AdminUser (Aggregate Root)
├── UserID (Entity ID - Value Object)
├── Email (Value Object) - 唯一
├── Name (Value Object)
├── Role (Value Object) - Admin/User/Guest
├── IsActive (Value Object)
├── LastLoginAt (Value Object)
├── CreatedAt (Value Object)
└── UpdatedAt (Value Object)
```

---

## **3.8 通知服務上下文 (Notification Context)**

**領域類型**: 通用域
**職責**: LINE Bot 訊息推送、Webhook 處理

### **統一語言 (Ubiquitous Language)**
- **通知 (Notification)**: 系統主動推送給用戶的訊息
- **Webhook 事件**: LINE Platform 推送的事件
- **訊息模板 (Message Template)**: 預定義的訊息格式

### **聚合設計**

**聚合: Notification** (聚合根)
```
Notification (Aggregate Root)
├── NotificationID (Entity ID)
├── MemberID (Reference)
├── Type (Welcome/PointsEarned/SurveyLink/etc.)
├── MessageContent (Value Object)
├── SentAt (Value Object)
└── Status (Pending/Sent/Failed)
```

---

## **3.9 稽核追蹤上下文 (Audit Context)**

**領域類型**: 支撐域
**職責**: 記錄所有資料變更歷史、提供稽核追蹤、支援合規檢查、GDPR 資料處理記錄

### **統一語言 (Ubiquitous Language)**
- **稽核日誌 (Audit Log)**: 不可變的資料變更記錄
- **操作者 (Actor)**: 執行操作的主體（會員、管理員、系統）
- **目標資源 (Target)**: 被操作的業務實體（會員、交易、積分帳戶等）
- **變更追蹤 (Change Tracking)**: 記錄資料變更前後的狀態對比
- **稽核事件 (Audit Event)**: 可稽核的業務操作類型
- **保存期限 (Retention Period)**: 依資料類型保存 3-7 年或永久
- **事務一致性 (Transactional Consistency)**: 稽核日誌與業務操作在同一事務中提交

### **聚合設計**

**聚合: AuditLog** (聚合根 - 不可變聚合)
```
AuditLog (Aggregate Root) - 設計原則：完全不可變
├── AuditID (Entity ID - Value Object) - "AUD-20250109-143045-ABC123"
├── Timestamp (Value Object) - 操作時間
├── EventType (Value Object) - 事件類型枚舉
├── Actor (Value Object) - 操作者資訊
│   ├── ActorType (Enum: MEMBER/ADMIN/SYSTEM)
│   ├── ActorID (string) - M123, A456, SYSTEM
│   ├── ActorName (string) - 小陳, 王姐, Auto Recalculation
│   ├── IPAddress (string) - 192.168.1.* (部分遮罩)
│   └── UserAgent (string) - LINE/10.0.0, Chrome/120.0
├── Target (Value Object) - 目標資源資訊
│   ├── TargetType (Enum: MEMBER/TRANSACTION/POINTS_ACCOUNT/SURVEY/RULE)
│   ├── TargetID (string) - M123, TX456, PA789
│   └── Description (string) - 會員小陳, 發票 AB12345678
├── Action (Value Object) - CREATE/UPDATE/DELETE
├── Changes (Value Object) - 變更內容
│   ├── Before (map[string]interface{}) - 原始狀態
│   ├── After (map[string]interface{}) - 新狀態
│   └── Diff (map[string]string) - 人類可讀的差異描述
├── Metadata (Value Object) - 額外元數據
│   ├── Reason (string) - 操作原因
│   ├── BatchID (string) - 批次 ID（如適用）
│   ├── RelatedTransactionID (string) - 相關交易 ID
│   └── CustomData (map[string]interface{}) - 自訂欄位
└── Result (Value Object) - 執行結果
    ├── Status (Enum: SUCCESS/FAILURE)
    └── ErrorMessage (string) - 錯誤訊息（如失敗）

設計原則：
- ✅ 完全不可變：創建後不允許任何修改或刪除
- ✅ 事務一致性：必須與業務操作在同一資料庫事務中創建
- ✅ 原子性保證：稽核日誌寫入失敗 → 業務操作自動回滾
- ✅ 敏感資料保護：IP 位址、手機號碼部分遮罩
- ✅ 輕量級聚合：不包含關聯實體，只存儲快照數據
```

**值對象**:
- `AuditID`: "AUD-{timestamp}-{random}" 格式，全局唯一
- `EventType`: 枚舉，包含所有可稽核事件類型
  - Member Events: `MEMBER_CREATED`, `MEMBER_PHONE_UPDATED`, `MEMBER_DELETED`
  - Points Events: `POINTS_EARNED`, `POINTS_DEDUCTED`, `POINTS_RECALCULATED`
  - Transaction Events: `TRANSACTION_CREATED`, `TRANSACTION_STATUS_CHANGED`, `TRANSACTION_MATCHED`
  - Survey Events: `SURVEY_CREATED`, `SURVEY_ACTIVATED`, `SURVEY_RESPONSE_CREATED`
  - Rule Events: `CONVERSION_RULE_CREATED`, `CONVERSION_RULE_UPDATED`, `CONVERSION_RULE_DELETED`
  - Import Events: `IMPORT_BATCH_CREATED`, `IMPORT_BATCH_COMPLETED`
  - Admin Events: `ADMIN_LOGIN`, `ADMIN_ROLE_CHANGED`, `ADMIN_SENSITIVE_OPERATION`
- `ActorType`: 枚舉 (MEMBER, ADMIN, SYSTEM)
- `TargetType`: 枚舉 (MEMBER, TRANSACTION, POINTS_ACCOUNT, SURVEY, CONVERSION_RULE, IMPORT_BATCH)
- `ActionType`: 枚舉 (CREATE, UPDATE, DELETE)
- `Changes`: 變更前後對比值對象，不可變
- `Metadata`: 元數據值對象，靈活擴展

### **聚合方法（接口定義）**

**AuditLog 聚合根方法**:
```go
// 構造方法（唯一創建途徑）
NewAuditLog(
    eventType EventType,
    actor Actor,
    target Target,
    action ActionType,
    changes Changes,
    metadata Metadata,
) (*AuditLog, error)
  // 創建稽核日誌（不可變）
  // 前置條件: 所有必填欄位已提供
  // 後置條件: AuditID 已生成，Timestamp = 現在，Result.Status = SUCCESS
  // 不變性保護: 創建後無任何修改方法
  // 設計原則:
  //   - 只有構造函數，無狀態變更方法
  //   - 完全不可變（Immutable Aggregate）
  //   - 符合 Event Sourcing 模式

// 驗證方法
Validate() error
  // 驗證所有必填欄位
  // 返回: ErrMissingRequiredField, ErrInvalidEventType, etc.

// 查詢方法（只讀）
GetAuditID() AuditID
GetTimestamp() time.Time
GetEventType() EventType
GetActor() Actor
GetTarget() Target
GetAction() ActionType
GetChanges() Changes
GetMetadata() Metadata
GetResult() Result

// 敏感資料遮罩
GetMaskedActor() Actor
  // 返回: Actor with masked IP (192.168.1.*) and phone numbers
  // 用途: 前端顯示時保護隱私

// 導出方法
ToJSON() ([]byte, error)
  // 轉換為 JSON 格式（匯出用）

ToPDF() ([]byte, error)
  // 轉換為 PDF 格式（合規報告）
```

### **領域服務（接口定義）**

**AuditLogRecordingService** (稽核日誌記錄服務):
```go
// Application Service - 簡化稽核日誌記錄流程
type AuditLogRecordingService struct {
    writer AuditLogWriter // 只依賴寫入接口
}

// 主接口 - 記錄稽核日誌（必須在業務事務中調用）
RecordAuditLog(ctx Context, log *AuditLog) error
  // 在同一事務中記錄稽核日誌
  // 參數: ctx - 包含資料庫事務的上下文
  // 設計原則:
  //   - 必須使用與業務操作相同的資料庫連接/事務
  //   - 如果稽核日誌寫入失敗，事務必須回滾
  //   - 保證 100% 記錄完整性（不允許業務操作成功但稽核日誌丟失）
  // 實現: 委託給 AuditLogWriter.Create()
  // 返回: ErrAuditLogWriteFailed 如果寫入失敗

RecordBatchAuditLogs(ctx Context, logs []*AuditLog) error
  // 批次記錄稽核日誌（用於批量操作）
  // 如: 積分重算、iChef 匯入等
  // 實現: 委託給 AuditLogWriter.CreateBatch()
```

**AuditLogQueryService** (稽核日誌查詢服務):
```go
// Application Service - 使用 AuditLogReader 和 AuditLogStatistics 接口
type AuditLogQueryService struct {
    reader     AuditLogReader     // 只依賴查詢接口
    statistics AuditLogStatistics // 只依賴統計接口
}

// 按篩選條件查詢
QueryAuditLogs(filter AuditLogFilter, pagination Pagination) ([]*AuditLog, int, error)
  // 複合查詢：支援多條件篩選
  // 參數:
  //   - filter: 篩選條件（時間範圍、事件類型、操作者、目標資源等）
  //   - pagination: 分頁參數（頁碼、每頁筆數、排序）
  // 返回:
  //   - []*AuditLog: 稽核日誌列表
  //   - int: 總筆數（用於分頁計算）
  //   - error: 錯誤（如有）
  // 實現: 委託給 AuditLogReader.FindAll()
  // 性能要求: < 3 秒（中位數）

// 按目標資源查詢
QueryByTarget(targetType TargetType, targetID string, pagination Pagination) ([]*AuditLog, int, error)
  // 查詢特定資源的所有變更歷史
  // 用途: 會員查詢自己的操作歷史、管理員追蹤問題
  // 實現: 委託給 AuditLogReader.FindByTarget()

// 按操作者查詢
QueryByActor(actorType ActorType, actorID string, pagination Pagination) ([]*AuditLog, int, error)
  // 查詢特定用戶的所有操作
  // 用途: 管理員行為稽核、異常操作偵測
  // 實現: 委託給 AuditLogReader.FindByActor()

// 統計查詢
CountByEventType(eventType EventType) (int, error)
  // 統計特定事件類型的總數
  // 用途: 稽核報表、合規統計
  // 實現: 委託給 AuditLogStatistics.CountByEventType()

DetectAnomalies(threshold AnomalyThreshold) ([]*AuditLog, error)
  // 偵測異常操作
  // 例如:
  //   - 單日積分變動超過 1000 點
  //   - 短時間多次登入失敗
  //   - 批量操作影響超過 100 筆記錄
  // 實現: 組合使用 reader 和 statistics 接口
  // 返回: 異常操作的稽核日誌列表
```

**AuditLogExportService** (稽核日誌匯出服務 - Application Service):
```go
// Application Service - 負責匯出邏輯，不是 Repository 職責
type AuditLogExportService struct {
    reader       AuditLogReader // 只依賴查詢接口
    csvFormatter CSVFormatter   // 格式化器（Infrastructure）
    pdfFormatter PDFFormatter   // 格式化器（Infrastructure）
}

// 匯出為 CSV
ExportToCSV(filter AuditLogFilter) ([]byte, error)
  // 匯出稽核日誌為 CSV 格式
  // 流程:
  //   1. 使用 reader.FindAll() 查詢日誌（最多 10,000 筆）
  //   2. 使用 csvFormatter.Format() 轉換為 CSV
  // 限制: 單次最多匯出 10,000 筆記錄
  // 用途: Excel 分析、合規報告

// 匯出為 PDF
ExportToPDF(filter AuditLogFilter, reportTemplate string) ([]byte, error)
  // 匯出稽核日誌為 PDF 報告
  // 流程:
  //   1. 使用 reader.FindAll() 查詢日誌
  //   2. 使用 pdfFormatter.Format() 轉換為 PDF
  // 參數:
  //   - reportTemplate: 報告模板（GDPR、內部稽核、監管報告等）
  // 用途: 正式合規報告、監管單位提交

// GDPR 資料匯出
ExportGDPRReport(memberID string) ([]byte, error)
  // 匯出特定會員的所有資料處理記錄
  // 流程:
  //   1. 查詢會員相關的所有稽核日誌
  //   2. 生成 JSON（機器可讀）+ PDF（人類可讀）
  // 符合 GDPR Right to Data Portability 要求

// 設計優勢:
// ✅ Export 是應用層關注點，不是持久化層
// ✅ Repository 只負責查詢，Formatter 負責格式轉換
// ✅ 易於擴展新格式（XML, Excel, etc.）
```

**AuditLogArchiveService** (稽核日誌歸檔服務 - Application Service):
```go
// Application Service - 使用 AuditLogArchiver 接口
type AuditLogArchiveService struct {
    archiver AuditLogArchiver // 只依賴歸檔接口
}

// 歸檔舊資料
ArchiveOldLogs(olderThan time.Time, targetStorage string) error
  // 將舊稽核日誌移至歸檔儲存
  // 參數:
  //   - olderThan: 歸檔閾值（例如 1 年前）
  //   - targetStorage: 目標儲存（warm storage, cold storage）
  // 策略:
  //   - 熱資料（0-1 年）: 主資料庫，快速查詢
  //   - 溫資料（1-3 年）: 歸檔資料庫，查詢稍慢
  //   - 冷資料（3 年以上）: 壓縮備份，按需恢復
  // 實現: 委託給 AuditLogArchiver.ArchiveOldLogs()
  // 不變性保護: 永不刪除（會員、積分、規則相關日誌永久保存）

// 恢復歸檔資料
RestoreArchivedLogs(dateRange DateRange) error
  // 從歸檔儲存恢復特定時間範圍的日誌至主資料庫
  // 用途: 歷史調查、合規檢查
  // 實現: 委託給 AuditLogArchiver.RestoreArchivedLogs()
```

### **倉儲接口**

**設計原則**: 遵循接口隔離原則 (ISP)，按客戶端使用場景拆分接口

**AuditLogWriter** (寫入接口 - 業務操作使用):
```go
type AuditLogWriter interface {
    Create(ctx Context, log *AuditLog) error
      // 前置條件: ctx 包含活躍的資料庫事務
      // 後置條件: 稽核日誌已寫入，與業務操作一同提交
      // 不變性保護:
      //   - 如果寫入失敗，返回錯誤並觸發事務回滾
      //   - 一旦成功寫入，永不允許修改或刪除
      // 返回: ErrAuditLogWriteFailed 如果寫入失敗

    CreateBatch(ctx Context, logs []*AuditLog) error
      // 用途: 積分重算、iChef 匯入等批次操作
      // 事務保證: 全部成功或全部回滾
}
```

**AuditLogReader** (查詢接口 - 管理後台使用):
```go
type AuditLogReader interface {
    FindAll(filter AuditLogFilter, pagination Pagination) ([]*AuditLog, int, error)
      // 索引優化:
      //   - idx_audit_timestamp (timestamp DESC)
      //   - idx_audit_event_type (event_type)
      //   - idx_audit_target (target_type, target_id, timestamp DESC)
      //   - idx_audit_actor (actor_type, actor_id)
      // 分頁: 每頁 100 筆，最多 500 筆
      // 返回: (日誌列表, 總筆數, 錯誤)

    FindByTarget(targetType TargetType, targetID string, pagination Pagination) ([]*AuditLog, int, error)
      // 排序: timestamp DESC（最新在前）

    FindByActor(actorType ActorType, actorID string, pagination Pagination) ([]*AuditLog, int, error)

    FindByEventType(eventType EventType, pagination Pagination) ([]*AuditLog, int, error)

    FindByDateRange(startDate time.Time, endDate time.Time, pagination Pagination) ([]*AuditLog, int, error)
}
```

**AuditLogStatistics** (統計接口 - 報表與分析使用):
```go
type AuditLogStatistics interface {
    CountByEventType(eventType EventType) (int, error)

    CountByActor(actorType ActorType, actorID string) (int, error)

    CountByDateRange(startDate time.Time, endDate time.Time) (int, error)

    GetDailySummary(date time.Time) (*DailySummary, error)
      // 返回: 各類型事件的統計數據
}
```

**AuditLogArchiver** (歸檔接口 - 系統維護使用):
```go
type AuditLogArchiver interface {
    ArchiveOldLogs(olderThan time.Time, targetStorage string) error
      // 參數:
      //   - olderThan: 歸檔閾值（例如 1 年前）
      //   - targetStorage: 目標儲存（warm/cold）
      // 策略:
      //   - 熱資料（0-1 年）: 主資料庫
      //   - 溫資料（1-3 年）: 歸檔資料庫
      //   - 冷資料（3 年以上）: 壓縮備份

    RestoreArchivedLogs(dateRange DateRange) error

    GetArchiveStatus() (*ArchiveStatus, error)
}
```

**實現範例** (Infrastructure Layer):
```go
// PostgreSQL 實現所有接口
type PostgresAuditLogRepository struct {
    db *sql.DB
}

// 實現 AuditLogWriter
func (r *PostgresAuditLogRepository) Create(ctx Context, log *AuditLog) error {
    // Implementation
}

func (r *PostgresAuditLogRepository) CreateBatch(ctx Context, logs []*AuditLog) error {
    // Implementation
}

// 實現 AuditLogReader
func (r *PostgresAuditLogRepository) FindAll(filter AuditLogFilter, pagination Pagination) ([]*AuditLog, int, error) {
    // Implementation
}
// ... 實現其他查詢方法

// 實現 AuditLogStatistics
func (r *PostgresAuditLogRepository) CountByEventType(eventType EventType) (int, error) {
    // Implementation
}
// ... 實現其他統計方法

// 實現 AuditLogArchiver
func (r *PostgresAuditLogRepository) ArchiveOldLogs(olderThan time.Time, targetStorage string) error {
    // Implementation
}
// ... 實現其他歸檔方法

// Application Layer 依賴注入
type SomeBusinessUseCase struct {
    auditWriter AuditLogWriter // 只依賴寫入接口
}

type AuditLogQueryUseCase struct {
    auditReader AuditLogReader // 只依賴查詢接口
}

type ArchiveMaintenanceJob struct {
    auditArchiver AuditLogArchiver // 只依賴歸檔接口
}
```

**設計優勢**:
- ✅ **接口隔離**: 業務操作只依賴 `AuditLogWriter`（2 方法），不依賴查詢/統計/歸檔
- ✅ **職責分離**: 每個接口服務特定客戶端（業務操作、管理後台、系統維護）
- ✅ **易於測試**: Mock 小接口比 Mock 12 方法的 God Interface 簡單
- ✅ **依賴最小化**: 減少不必要的耦合
- ✅ **符合 ISP**: "Clients should not be forced to depend on methods they do not use"

**注意**: 不提供 Update 和 Delete 方法（不可變聚合設計）
```

**AuditLogFilter** (查詢篩選值對象):
```go
type AuditLogFilter struct {
    EventTypes   []EventType  // 事件類型篩選（多選）
    ActorTypes   []ActorType  // 操作者類型篩選
    TargetTypes  []TargetType // 目標資源類型篩選
    StartDate    *time.Time   // 開始時間（可選）
    EndDate      *time.Time   // 結束時間（可選）
    ActorID      string       // 操作者 ID（精確匹配）
    TargetID     string       // 目標資源 ID（精確匹配）
    SearchText   string       // 全文搜索（描述、元數據）
    ActionTypes  []ActionType // 操作類型篩選（CREATE/UPDATE/DELETE）
}
```

**Pagination** (分頁值對象):
```go
type Pagination struct {
    Page      int    // 頁碼（1-based）
    PageSize  int    // 每頁筆數（預設 100，最大 500）
    SortBy    string // 排序欄位（預設 "timestamp"）
    SortOrder string // 排序方向（"asc" 或 "desc"，預設 "desc"）
}
```

### **領域事件**

**重要設計決策**: Audit Context 不發布領域事件，只監聽其他 Context 的事件

稽核上下文作為「觀察者」角色：
- ✅ **監聽所有業務事件**: `MemberRegistered`, `PointsEarned`, `TransactionVerified`, etc.
- ✅ **被動記錄**: 收到事件後創建 AuditLog，但不發布新事件
- ✅ **避免循環依賴**: 防止「稽核事件觸發稽核事件」的無限遞迴
- ❌ **不發布事件**: 稽核日誌的創建不應觸發其他業務邏輯

**監聽的事件列表**:
- `MemberRegistered` → 記錄 `MEMBER_CREATED`
- `PhoneNumberBound` → 記錄 `MEMBER_PHONE_UPDATED`
- `MemberProfileUpdated` → 記錄 `MEMBER_UPDATED`
- `PointsEarned` → 記錄 `POINTS_EARNED`
- `PointsDeducted` → 記錄 `POINTS_DEDUCTED`
- `PointsRecalculated` → 記錄 `POINTS_RECALCULATED`
- `TransactionCreated` → 記錄 `TRANSACTION_CREATED`
- `TransactionVerified` → 記錄 `TRANSACTION_STATUS_CHANGED`
- `SurveyCreated` → 記錄 `SURVEY_CREATED`
- `SurveyActivated` → 記錄 `SURVEY_ACTIVATED`
- `SurveyResponseSubmitted` → 記錄 `SURVEY_RESPONSE_CREATED`
- `ConversionRuleCreated` → 記錄 `CONVERSION_RULE_CREATED`
- `ConversionRuleUpdated` → 記錄 `CONVERSION_RULE_UPDATED`
- `ImportBatchCompleted` → 記錄 `IMPORT_BATCH_COMPLETED`
- `AdminLogin` → 記錄 `ADMIN_LOGIN`
- `AdminRoleChanged` → 記錄 `ADMIN_ROLE_CHANGED`

### **領域錯誤定義**

```go
// 稽核追蹤上下文錯誤
var (
    // AuditLog 相關錯誤
    ErrAuditLogWriteFailed      error = "Failed to write audit log (transaction will rollback)"
    ErrMissingRequiredField     error = "Audit log is missing required field"
    ErrInvalidEventType         error = "Invalid audit event type"
    ErrInvalidActorType         error = "Invalid actor type"
    ErrInvalidTargetType        error = "Invalid target type"
    ErrInvalidActionType        error = "Invalid action type"

    // 查詢相關錯誤
    ErrAuditLogNotFound         error = "Audit log not found"
    ErrInvalidDateRange         error = "Invalid date range: start date must be before end date"
    ErrExceedExportLimit        error = "Export limit exceeded (max 10,000 records)"
    ErrInvalidPagination        error = "Invalid pagination parameters"

    // 歸檔相關錯誤
    ErrArchiveFailed            error = "Failed to archive audit logs"
    ErrRestoreFailed            error = "Failed to restore archived audit logs"
    ErrCannotDeleteAuditLog     error = "Audit logs cannot be deleted (immutable)"

    // 權限相關錯誤
    ErrUnauthorizedAuditAccess  error = "Unauthorized access to audit logs"
    ErrGuestCannotViewAudit     error = "Guest role cannot view audit logs"
)
```

### **資料保存策略**

**分層儲存架構**:

| 資料層級 | 時間範圍 | 儲存位置 | 查詢性能 | 保存期限 |
|---------|---------|---------|---------|---------|
| **熱資料** | 最近 1 年 | PostgreSQL 主資料庫 | < 1 秒 | 1 年 |
| **溫資料** | 1-3 年 | PostgreSQL 歸檔資料庫 | < 5 秒 | 3 年 |
| **冷資料** | 3 年以上 | S3/備份儲存（壓縮） | 按需恢復 | 永久/7 年 |

**保存期限規則** (符合 PRD 要求):

| 資料類型 | 保存期限 | 法規依據 |
|---------|---------|---------|
| 會員資料變更 | 永久 | 個資法、GDPR |
| 積分變動 | 永久 | 商業記錄 |
| 交易記錄 | 7 年 | 稅務法規 |
| 問卷相關 | 5 年 | 統計法 |
| 規則變更 | 永久 | 內部稽核 |
| iChef 匯入 | 7 年 | 財務合規 |
| 管理員登入 | 3 年 | 資安政策 |
| 管理員敏感操作 | 永久 | 內部稽核 |

**GDPR 合規**:
- ✅ **Right to Access**: 會員可查詢自己的資料處理記錄
- ✅ **Right to Data Portability**: 支援匯出 JSON/PDF 格式
- ✅ **Right to be Forgotten**: 資料刪除操作記錄稽核日誌
- ✅ **Data Processing Record**: 完整記錄所有個資處理操作
- ✅ **Breach Notification**: 異常操作即時告警

---
