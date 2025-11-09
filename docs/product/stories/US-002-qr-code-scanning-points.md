# User Story 002: QR Code 掃描與積分計算 (QR Code Scanning & Points Calculation)

**Story ID**: US-002
**Priority**: P0 (Must Have)
**Sprint**: Phase 1 - MVP Core Features
**Status**: ✅ Completed (Enhanced in Phase 2 with Dynamic Conversion Rules)
**Estimated Effort**: 13 Story Points

---

## 📖 User Story

> **身為** 一位會員 (小陳)，
> **我想要** 上傳包含消費金額 QR Code 的照片來自動計算並獲得積分，
> **以便** 我能根據消費金額輕鬆累積相對應的積分獎勵，並收到問卷連結。

---

## 💎 Points Calculation Rules

### **基本公式**

```
單筆交易基礎積分 = 消費金額 ÷ 轉換率（無條件捨去小數）
問卷獎勵積分 = 1 點（完成問卷後獲得）
累積總積分 = 所有已驗證交易的積分總和
```

### **積分轉換率**

- **預設轉換率**: 100 元 = 1 點
- **動態轉換率**: 管理員可設定不同時期的轉換率（如促銷期間 50 元 = 1 點）
- **自動套用**: 系統根據發票日期自動套用對應時期的轉換率

### **計算範例**

| 消費金額 | 轉換率 | 問卷狀態 | 基礎積分 | 問卷獎勵 | 總積分 |
|---------|-------|---------|---------|---------|--------|
| $250 | 100 | 未完成 | 2 點 | 0 點 | **2 點** |
| $250 | 100 | 已完成 | 2 點 | 1 點 | **3 點** |
| $180 | 50 | 未完成 | 3 點 | 0 點 | **3 點** |
| $99 | 100 | 已完成 | 0 點 | 1 點 | **1 點** |

---

## ✅ Acceptance Criteria

### **成功場景 1：掃描有效發票**

**Given** 會員上傳發票 QR Code 照片
**When** 系統解析 QR Code
**Then** 提取發票號碼、日期、金額並顯示確認訊息

**Given** 發票未重複且在 60 天有效期內
**When** 系統處理發票
**Then**
- 記錄交易（狀態：imported - 待驗證）
- 計算預估積分並顯示
- 若有啟用問卷，提供問卷連結

**Given** 發票已通過 iChef 系統驗證
**When** 管理員匯入 iChef 資料
**Then** 交易狀態更新為「verified - 已驗證」並自動計入累積積分

---

### **失敗場景 1：QR Code 無效**

**Given** 會員上傳的照片無法辨識 QR Code
**When** 系統處理
**Then** 顯示「未找到 QR Code，請重新上傳清晰的照片」

---

### **失敗場景 2：發票已過期**

**Given** 發票日期超過 60 天
**When** 系統驗證
**Then** 顯示「發票已超過有效期限（60天），無法獲得積分」

---

### **失敗場景 3：重複發票**

**Given** 發票已被登錄過
**When** 系統檢查
**Then** 顯示「此發票已被登錄，無法重複獲得積分」

---

## 📋 Business Rules

| Rule ID | Description |
|---------|-------------|
| BR-002-01 | 發票有效期：開立日期起 60 天內 |
| BR-002-02 | 重複檢測：同一發票號碼只能登錄一次 |
| BR-002-03 | 積分計算：使用發票日期對應的轉換率規則 |
| BR-002-04 | 積分狀態：「待驗證」交易不計入累積積分，需等待 iChef 系統驗證 |
| BR-002-05 | 問卷獎勵：每筆交易最多獲得 1 點問卷獎勵 |
| BR-002-06 | 無條件捨去：積分計算結果無條件捨去小數 |
| BR-002-07 | 積分重算：交易狀態變更時自動觸發積分重新計算 |

---

## 🔧 Technical Implementation Notes

### **Entity & Value Object**

```go
// Transaction Entity
type Transaction struct {
    ID             TransactionID
    User           *User
    InvoiceNo      InvoiceNumber  // Value Object with validation
    InvoiceDate    InvoiceDate    // Value Object with date validation
    Amount         Money          // Value Object with currency handling
    Status         TransactionStatus
    SurveySubmitted bool
    CreatedAt      time.Time
}

// Money Value Object
type Money struct {
    Amount   int // 單位：元（整數）
    Currency string // "TWD"
}

// InvoiceDate Value Object
type InvoiceDate struct {
    Date time.Time
}

func (d InvoiceDate) IsValid() bool {
    // 檢查是否在 60 天內
    return time.Since(d.Date) <= 60*24*time.Hour
}

// PointsCalculator Entity
type PointsCalculator struct {
    ConversionRules []PointsConversionRule
}

func (pc *PointsCalculator) CalculatePoints(amount int, invoiceDate time.Time, surveySubmitted bool) int {
    rule := pc.FindApplicableRule(invoiceDate)
    basePoints := amount / rule.Multiplier  // Floor division
    surveyBonus := 0
    if surveySubmitted {
        surveyBonus = 1
    }
    return basePoints + surveyBonus
}
```

### **Use Case Interface**

```go
// internal/service/qrcode_service.go
type QRCodeService interface {
    ProcessInvoiceQRCode(lineUserID string, imageData []byte) (*Transaction, error)
    ParseInvoiceQRCode(qrCodeData string) (*InvoiceInfo, error)
}

// internal/service/transaction_service.go
type TransactionService interface {
    CreateTransaction(tx *Transaction) error
    UpdateTransactionStatus(txID string, newStatus TransactionStatus) error
    RecalculateUserPoints(userID string) error
}
```

### **Repository Interface**

```go
// internal/repository/transaction_repository.go
type TransactionRepository interface {
    Create(tx *Transaction) error
    FindByInvoiceNo(invoiceNo string) (*Transaction, error)
    FindByUserID(userID string) ([]*Transaction, error)
    FindVerifiedTransactionsByUserID(userID string) ([]*Transaction, error)
    UpdateStatus(txID string, status TransactionStatus) error
}
```

### **Database Schema**

```sql
-- Table: transactions
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    invoice_no VARCHAR(10) UNIQUE NOT NULL,
    invoice_date DATE NOT NULL,
    amount INTEGER NOT NULL,
    status VARCHAR(20) DEFAULT 'imported',
    survey_submitted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_invoice_no ON transactions(invoice_no);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_invoice_date ON transactions(invoice_date);

-- Table: points_conversion_rules
CREATE TABLE points_conversion_rules (
    id SERIAL PRIMARY KEY,
    start_date DATE NOT NULL,
    end_date DATE,
    multiplier INTEGER NOT NULL CHECK (multiplier >= 1 AND multiplier <= 1000),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_conversion_rules_dates ON points_conversion_rules(start_date, end_date);
```

### **Transaction Status Flow**

```
imported (待驗證)
    ↓
verified (已驗證) ← 觸發積分重算
    ↓
failed (作廢) ← 觸發積分重算（扣除）
```

### **Error Handling**

- `ErrQRCodeNotFound`: 未找到 QR Code
- `ErrInvalidQRCodeFormat`: QR Code 格式錯誤
- `ErrInvoiceExpired`: 發票已過期
- `ErrDuplicateInvoice`: 發票已登錄
- `ErrInvoiceDateInvalid`: 發票日期無效

---

## 🧪 Test Cases

### **Unit Tests**

- ✅ `TestParseInvoiceQRCode_ValidFormat`: 解析有效的 QR Code
- ✅ `TestParseInvoiceQRCode_InvalidFormat`: 解析無效的 QR Code
- ✅ `TestInvoiceDate_IsValid`: 驗證發票有效期（60 天）
- ✅ `TestPointsCalculator_CalculatePoints`: 積分計算邏輯
  - 測試案例：$250 → 2 點（轉換率 100）
  - 測試案例：$250 + 問卷 → 3 點
  - 測試案例：$180 → 3 點（轉換率 50）
  - 測試案例：$99 + 問卷 → 1 點（0 基礎 + 1 問卷）
- ✅ `TestCreateTransaction_Success`: 成功建立交易
- ✅ `TestCreateTransaction_DuplicateInvoice`: 重複發票失敗
- ✅ `TestUpdateTransactionStatus_TriggersPointsRecalculation`: 狀態變更觸發積分重算

### **Integration Tests**

- ✅ `TestQRCodeScanning_EndToEnd`: 完整掃描流程
- ✅ `TestDynamicConversionRules_Application`: 動態轉換率套用
- ✅ `TestPointsRecalculation_OnStatusChange`: 狀態變更時積分重算

---

## 📦 Dependencies

### **Internal Dependencies**

- **US-001**: 用戶必須先完成帳號綁定才能掃描 QR Code
- **US-004**: 問卷系統（可選，影響問卷獎勵積分）

### **External Dependencies**

- LINE Bot SDK: 接收圖片訊息
- QR Code Library: 解析 QR Code（如 `github.com/makiuchi-d/gozxing`）
- PostgreSQL: 儲存交易記錄

### **Service Dependencies**

- `QRCodeService`: QR Code 解析與發票處理
- `TransactionService`: 交易管理與積分計算
- `PointsService`: 積分查詢與統計
- `SurveyService`: 問卷連結生成（可選）

---

## 📊 Success Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| QR Code 辨識成功率 | > 95% | 成功解析數 / 總上傳數 |
| QR Code 掃描參與度 | > 60% | 每月掃描會員數 / 總會員數 |
| 重複發票錯誤率 | < 2% | 重複發票數 / 總掃描數 |
| 平均處理時間 | < 5 秒 | 上傳到顯示結果的時間 |
| 積分計算準確率 | 100% | 正確計算筆數 / 總筆數 |

---

## 🎯 User Personas

**Primary Persona**: 常客小陳（忠誠顧客）
- 25-35 歲上班族
- 每週至少光顧 2 次
- 喜歡簡單快速累積積分
- 願意參與問卷調查

**Secondary Persona**: 新客小美（潛在顧客）
- 20-30 歲學生或社會新鮮人
- 對科技體驗感興趣
- 希望立即看到積分累積效果

---

## 📝 UI/UX Flow

### **Conversation Flow**

```
用戶: [上傳發票 QR Code 照片]
Bot:
  ┌─────────────────────────────────┐
  │ 🔍 正在解析 QR Code...          │
  └─────────────────────────────────┘

Bot:
  ┌─────────────────────────────────┐
  │ ✅ 發票資訊確認                 │
  │                                  │
  │ 發票號碼: AB12345678            │
  │ 消費日期: 2025-01-05            │
  │ 消費金額: $250                  │
  │ 預估積分: 2 點                  │
  │                                  │
  │ 狀態: 待驗證                    │
  │ （待店家匯入後自動驗證）        │
  │                                  │
  │ 📋 填寫問卷可再得 1 點：        │
  │ [問卷連結]                      │
  └─────────────────────────────────┘
```

---

## 🚀 Performance Considerations

### **Optimization Strategies**

1. **QR Code 解析優化**
   - 使用高效的 QR Code 解析庫
   - 圖片預處理（調整大小、對比度）

2. **積分計算優化**
   - PointsCalculator Entity 封裝計算邏輯
   - 轉換規則緩存（Redis）

3. **資料庫查詢優化**
   - invoice_no 唯一索引（防止重複）
   - user_id, status, invoice_date 複合索引

4. **並發控制**
   - 資料庫事務隔離級別：READ COMMITTED
   - 樂觀鎖或悲觀鎖防止重複掃描

---

## 🔗 Related Documents

- [PRD.md](../PRD.md) - 完整產品需求文件（§ 2.2）
- [DOMAIN_MODEL.md](../../architecture/DOMAIN_MODEL.md) - Transaction Entity 和 PointsCalculator 設計
- [DATABASE_DESIGN.md](../../architecture/DATABASE_DESIGN.md) - transactions 表結構設計
- [POINTS_SYSTEM_DESIGN.md](../../POINTS_SYSTEM_DESIGN.md) - 積分系統性能優化設計
- [US-004](./US-004-survey-system.md) - 問卷系統（影響問卷獎勵積分）
- [US-005](./US-005-ichef-integration.md) - iChef 整合（發票驗證）

---

**Story Created**: 2025-01-08
**Last Updated**: 2025-01-08
**Story Owner**: Product Team
**Technical Owner**: Backend Team
