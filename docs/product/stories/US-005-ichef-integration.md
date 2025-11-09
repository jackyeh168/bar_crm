# User Story 005: iChef POS 系統整合 (iChef POS System Integration)

**Story ID**: US-005
**Priority**: P1 (Should Have)
**Sprint**: Phase 2 - Advanced Features
**Status**: ✅ Completed (Enhanced with Bidirectional Verification & Status Tracking)
**Estimated Effort**: 21 Story Points

---

## 📖 User Story

> **身為** 店長 (王姐)，
> **我想要** 批次匯入 iChef POS 系統的發票資料，
> **以便** 自動驗證會員掃描的發票，並自動計算正確的積分。

---

## ✅ Acceptance Criteria

### **成功場景 1：匯入 iChef 資料**

**Given** 店長從 iChef 系統匯出 Excel 發票資料
**When** 上傳到管理後台
**Then** 系統顯示匯入摘要：
- 總筆數
- 成功匹配筆數（與會員掃描記錄匹配）
- 未匹配筆數（無對應掃描記錄）
- 重複筆數（已匯入過）
- 跳過筆數（無效格式）

**Given** 匯入的發票與會員掃描記錄匹配
**When** 系統驗證
**Then**
- 交易狀態從「imported」更新為「verified」
- 自動計入會員累積積分

---

### **成功場景 2：雙向驗證（會員先掃描）**

**Given** 會員先上傳發票 QR Code（status = imported）
**When** 店長後續匯入 iChef 資料
**Then**
- 系統自動比對發票號碼、日期、金額
- 匹配成功後狀態更新為 verified
- 自動觸發積分重算

---

### **成功場景 3：雙向驗證（店家先匯入）**

**Given** 店長先匯入 iChef 資料（無對應會員掃描記錄）
**When** 會員後續掃描該發票 QR Code
**Then**
- 系統檢測到已存在 iChef 記錄
- 自動創建交易並設定 status = verified
- 立即計入積分（無需等待驗證）

---

### **成功場景 4：發票作廢處理**

**Given** 已驗證的交易（status = verified）
**When** iChef 資料顯示發票作廢（status_change = 作廢）
**Then**
- 交易狀態更新為 failed
- 自動扣除已累積的積分
- 記錄狀態變更歷史

---

## 📋 Business Rules

| Rule ID | Description |
|---------|-------------|
| BR-005-01 | 匹配條件：發票號碼、日期、金額三者完全一致 |
| BR-005-02 | 重複檢測：相同發票（號碼+日期+金額）只能匯入一次 |
| BR-005-03 | 狀態流轉：imported → verified（匹配成功）或 verified → failed（發票作廢） |
| BR-005-04 | 積分重算：狀態變更後自動觸發積分重新計算 |
| BR-005-05 | 雙向驗證：會員可先掃描，店家後續驗證；或店家先匯入，會員掃描時自動驗證 |
| BR-005-06 | 發票正規化：統一發票號碼格式（大寫、移除空白） |
| BR-005-07 | 狀態追蹤：記錄發票狀態變更歷史（imported → verified → failed） |

---

## 🔧 Technical Implementation Notes

### **Entity & Value Object**

```go
// IChefImportHistory Entity
type IChefImportHistory struct {
    ID              ImportHistoryID
    FileName        string
    TotalRows       int
    MatchedCount    int
    UnmatchedCount  int
    SkippedCount    int
    DuplicateCount  int
    ImportedAt      time.Time
    ImportedBy      *AdminUser
}

// IChefInvoiceRecord Entity
type IChefInvoiceRecord struct {
    ID                    RecordID
    ImportHistoryID       ImportHistoryID
    InvoiceNo             InvoiceNumber  // Normalized
    InvoiceDate           InvoiceDate
    Amount                Money
    MatchStatus           MatchStatus
    MatchedTransactionID  *TransactionID
    StatusChange          StatusChange   // 作廢, 正常, etc.
    CreatedAt             time.Time
}

// MatchStatus Value Object
type MatchStatus string

const (
    MatchStatusMatched   MatchStatus = "matched"
    MatchStatusUnmatched MatchStatus = "unmatched"
    MatchStatusSkipped   MatchStatus = "skipped"
)

// StatusChange Value Object
type StatusChange string

const (
    StatusChangeNormal StatusChange = "正常"
    StatusChangeVoid   StatusChange = "作廢"
)

// InvoiceNumber Value Object (with normalization)
type InvoiceNumber struct {
    Value string
}

func NewInvoiceNumber(raw string) InvoiceNumber {
    normalized := strings.ToUpper(strings.TrimSpace(raw))
    return InvoiceNumber{Value: normalized}
}
```

### **Use Case Interface**

```go
// internal/service/ichef_import_service.go
type IChefImportService interface {
    ProcessExcelImport(file io.Reader, fileName string, adminUserID int) (*IChefImportHistory, error)
    GetImportHistory(historyID int) (*IChefImportHistory, error)
    GetImportHistoryList(page int, pageSize int) ([]*IChefImportHistory, int, error)
    GetInvoiceRecords(historyID int) ([]*IChefInvoiceRecord, error)
}

// Import Processing Steps:
// 1. Parse Excel file (read rows)
// 2. Normalize invoice numbers
// 3. Check for duplicates (batch query)
// 4. Match with existing transactions
// 5. Update transaction status
// 6. Create import history and records
// 7. Trigger points recalculation
```

### **Repository Interface**

```go
// internal/repository/ichef_import_repository.go
type IChefImportHistoryRepository interface {
    Create(history *IChefImportHistory) error
    FindByID(id int) (*IChefImportHistory, error)
    List(offset int, limit int) ([]*IChefImportHistory, int, error)
}

// internal/repository/ichef_invoice_record_repository.go
type IChefInvoiceRecordRepository interface {
    BulkCreate(records []*IChefInvoiceRecord) error
    BulkCreateNonExisting(records []*IChefInvoiceRecord) error // Skip duplicates
    FindByHistoryID(historyID int) ([]*IChefInvoiceRecord, error)
    FindByInvoiceKey(invoiceNo string, invoiceDate time.Time, amount int) (*IChefInvoiceRecord, error)
    CheckDuplicates(records []*IChefInvoiceRecord) ([]bool, error)
}
```

### **Database Schema**

```sql
-- Table: ichef_import_history
CREATE TABLE ichef_import_history (
    id SERIAL PRIMARY KEY,
    file_name VARCHAR(255) NOT NULL,
    total_rows INTEGER NOT NULL,
    matched_count INTEGER DEFAULT 0,
    unmatched_count INTEGER DEFAULT 0,
    skipped_count INTEGER DEFAULT 0,
    duplicate_count INTEGER DEFAULT 0,
    imported_at TIMESTAMP DEFAULT NOW(),
    imported_by INTEGER REFERENCES admin_users(id)
);

CREATE INDEX idx_ichef_import_history_imported_at ON ichef_import_history(imported_at);

-- Table: ichef_invoice_records
CREATE TABLE ichef_invoice_records (
    id SERIAL PRIMARY KEY,
    import_history_id INTEGER REFERENCES ichef_import_history(id) ON DELETE CASCADE,
    invoice_no_normalized VARCHAR(10) NOT NULL,
    invoice_date DATE NOT NULL,
    amount INTEGER NOT NULL,
    match_status VARCHAR(20) DEFAULT 'unmatched',
    matched_transaction_id INTEGER REFERENCES transactions(id),
    status_change VARCHAR(10),
    created_at TIMESTAMP DEFAULT NOW()
);

-- 防止重複匯入：相同發票（號碼+日期+金額）唯一性約束
CREATE UNIQUE INDEX idx_ichef_invoice_unique ON ichef_invoice_records(
    invoice_no_normalized,
    invoice_date,
    amount
);

CREATE INDEX idx_ichef_invoice_import_history ON ichef_invoice_records(import_history_id);
CREATE INDEX idx_ichef_invoice_match_status ON ichef_invoice_records(match_status);
```

### **Matching Algorithm**

```go
// Matching Logic
func MatchInvoice(record *IChefInvoiceRecord, transactions []*Transaction) (*Transaction, bool) {
    for _, tx := range transactions {
        if tx.InvoiceNo.Value == record.InvoiceNo.Value &&
           tx.InvoiceDate.Date.Equal(record.InvoiceDate.Date) &&
           tx.Amount.Amount == record.Amount.Amount {
            return tx, true
        }
    }
    return nil, false
}

// Bidirectional Verification
func ProcessImport(records []*IChefInvoiceRecord) {
    for _, record := range records {
        // Check if transaction exists
        tx := FindTransactionByInvoiceKey(record.InvoiceNo, record.InvoiceDate, record.Amount)

        if tx != nil {
            // Scenario 1: Member scanned first
            tx.Status = TransactionStatusVerified
            record.MatchStatus = MatchStatusMatched
            record.MatchedTransactionID = &tx.ID

            // Handle status change (void invoice)
            if record.StatusChange == StatusChangeVoid {
                tx.Status = TransactionStatusFailed
            }

            // Trigger points recalculation
            RecalculateUserPoints(tx.UserID)
        } else {
            // Scenario 2: Store imported first (no member scan yet)
            record.MatchStatus = MatchStatusUnmatched
        }
    }
}
```

### **Duplicate Detection Strategy**

**Database-Level**:
- Unique constraint on `(invoice_no_normalized, invoice_date, amount)`
- Automatic duplicate rejection

**Application-Level**:
- Batch duplicate check before insert: `BulkCreateNonExisting()`
- Filter out duplicates, insert only new records
- Count duplicates in import statistics

### **Performance Optimization**

1. **Batch Processing**
   - Read Excel file in chunks (1000 rows per batch)
   - Batch query existing transactions (IN clause)
   - Batch insert iChef records (bulk insert)

2. **Duplicate Check Optimization**
   - Use `SELECT invoice_no_normalized, invoice_date, amount FROM ichef_invoice_records WHERE ...`
   - Build hash set for O(1) lookup
   - Filter duplicates before insert

3. **Points Recalculation**
   - Collect all affected user IDs
   - Batch recalculate (single query per user)
   - Use database transaction to ensure atomicity

### **Error Handling**

- `ErrInvalidExcelFormat`: Excel 格式錯誤
- `ErrMissingRequiredColumn`: 缺少必要欄位
- `ErrInvalidInvoiceNumber`: 發票號碼格式錯誤
- `ErrInvalidInvoiceDate`: 發票日期格式錯誤
- `ErrInvalidAmount`: 金額格式錯誤
- `ErrDuplicateInvoice`: 發票已匯入（批次去重時使用）

---

## 🧪 Test Cases

### **Unit Tests**

- ✅ `TestNormalizeInvoiceNumber`: 發票號碼正規化
- ✅ `TestMatchInvoice_Success`: 成功匹配發票
- ✅ `TestMatchInvoice_NoMatch`: 無匹配記錄
- ✅ `TestProcessImport_MemberScannedFirst`: 會員先掃描場景
- ✅ `TestProcessImport_StoreImportedFirst`: 店家先匯入場景
- ✅ `TestProcessImport_VoidInvoice`: 發票作廢處理
- ✅ `TestBulkCreateNonExisting_SkipDuplicates`: 批次去重

### **Integration Tests**

- ✅ `TestExcelImport_EndToEnd`: 完整匯入流程
- ✅ `TestExcelImport_DuplicateDetection`: 重複檢測
- ✅ `TestExcelImport_PointsRecalculation`: 積分重算觸發
- ✅ `TestBidirectionalVerification_MemberFirst`: 雙向驗證（會員先掃）
- ✅ `TestBidirectionalVerification_StoreFirst`: 雙向驗證（店家先匯）

### **Performance Tests**

- ✅ `TestImport_1000Rows_Performance`: 1000 筆資料匯入性能
- ✅ `TestBulkCreateNonExisting_Performance`: 批次去重性能

---

## 📦 Dependencies

### **Internal Dependencies**

- **US-002**: QR Code 掃描創建交易記錄（匹配來源）
- **US-003**: 積分查詢（顯示驗證後的積分）
- **US-006**: 管理後台（提供匯入介面）

### **External Dependencies**

- Excel Library: 解析 Excel 檔案（如 `github.com/xuri/excelize/v2`）
- PostgreSQL: 儲存匯入歷史和發票記錄

### **Service Dependencies**

- `IChefImportService`: 匯入處理邏輯
- `TransactionService`: 交易狀態更新
- `PointsService`: 積分重算
- `IChefImportHistoryRepository`: 匯入歷史存取
- `IChefInvoiceRecordRepository`: 發票記錄存取

---

## 📊 Success Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| iChef 驗證率 | > 85% | 匹配成功數 / QR 掃描數 |
| 匯入處理時間 | < 30 秒 / 1000 筆 | Excel 上傳到完成的時間 |
| 重複發票率 | < 5% | 重複數 / 總匯入數 |
| 積分重算準確率 | 100% | 正確重算數 / 總匹配數 |
| 雙向驗證成功率 | > 95% | 自動驗證成功數 / 總記錄數 |

---

## 🎯 User Personas

**Primary Persona**: 店長王姐（營運管理者）
- 35-45 歲餐廳店長
- 負責日常營運與會員管理
- 每週匯入 1-2 次 iChef 資料
- 期望簡單快速的匯入流程

**Secondary Persona**: 會員小陳（忠誠顧客）
- 25-35 歲上班族
- 希望掃描的發票能快速驗證
- 期望積分自動更新

---

## 📝 UI/UX Flow

### **Admin Side: Import Flow**

```
店長: [登入管理後台]
     ↓
店長: [進入 iChef 匯入頁面]
     ↓
系統: 顯示上傳介面
     ┌─────────────────────────────────┐
     │ iChef 發票資料匯入              │
     │                                  │
     │ 選擇檔案: [選擇 Excel 檔案]     │
     │                                  │
     │ [開始匯入]                      │
     └─────────────────────────────────┘

店長: [選擇 Excel 檔案並上傳]
     ↓
系統: 處理匯入並顯示結果
     ┌─────────────────────────────────┐
     │ ✅ 匯入完成                     │
     │                                  │
     │ 總筆數: 523                     │
     │ 成功匹配: 487 (93%)             │
     │ 未匹配: 30 (6%)                 │
     │ 重複: 5 (1%)                    │
     │ 跳過: 1 (0%)                    │
     │                                  │
     │ [查看詳細記錄]                  │
     └─────────────────────────────────┘
```

### **Member Side: Automatic Verification**

```
會員: [上傳發票 QR Code]
     ↓
系統: 檢查是否已有 iChef 記錄
     ↓
Case 1: 已有 iChef 記錄
Bot:
  ┌─────────────────────────────────┐
  │ ✅ 發票已驗證                   │
  │                                  │
  │ 發票號碼: AB12345678            │
  │ 消費金額: $250                  │
  │ 獲得積分: 2 點 ✓               │
  │                                  │
  │ （已自動驗證，積分已入帳）      │
  └─────────────────────────────────┘

Case 2: 無 iChef 記錄
Bot:
  ┌─────────────────────────────────┐
  │ ✅ 發票資訊確認                 │
  │                                  │
  │ 發票號碼: AB12345678            │
  │ 消費金額: $250                  │
  │ 預估積分: 2 點                  │
  │                                  │
  │ 狀態: 待驗證                    │
  │ （待店家匯入後自動驗證）        │
  └─────────────────────────────────┘
```

---

## 🚀 Performance Considerations

### **Excel Parsing Optimization**

```go
// Use streaming reader for large files
func ParseExcel(file io.Reader) ([]InvoiceRow, error) {
    f, err := excelize.OpenReader(file)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    rows, err := f.GetRows("Sheet1")
    if err != nil {
        return nil, err
    }

    // Parse in chunks
    const chunkSize = 1000
    var invoices []InvoiceRow

    for i := 1; i < len(rows); i++ { // Skip header
        invoice := parseRow(rows[i])
        invoices = append(invoices, invoice)

        if len(invoices) >= chunkSize {
            // Process chunk
            processChunk(invoices)
            invoices = invoices[:0] // Clear
        }
    }

    // Process remaining
    if len(invoices) > 0 {
        processChunk(invoices)
    }

    return invoices, nil
}
```

### **Batch Duplicate Check**

```go
// Optimized duplicate check
func (r *IChefInvoiceRecordRepository) BulkCreateNonExisting(records []*IChefInvoiceRecord) error {
    // Build query with IN clause
    var keys []string
    for _, record := range records {
        key := fmt.Sprintf("('%s', '%s', %d)",
            record.InvoiceNo,
            record.InvoiceDate.Format("2006-01-02"),
            record.Amount)
        keys = append(keys, key)
    }

    // Single query to find existing records
    query := fmt.Sprintf(`
        SELECT invoice_no_normalized, invoice_date, amount
        FROM ichef_invoice_records
        WHERE (invoice_no_normalized, invoice_date, amount) IN (%s)
    `, strings.Join(keys, ","))

    existing := make(map[string]bool)
    rows, _ := r.db.Query(query)
    for rows.Next() {
        var no string
        var date time.Time
        var amt int
        rows.Scan(&no, &date, &amt)
        key := fmt.Sprintf("%s-%s-%d", no, date.Format("2006-01-02"), amt)
        existing[key] = true
    }

    // Filter new records
    var newRecords []*IChefInvoiceRecord
    for _, record := range records {
        key := fmt.Sprintf("%s-%s-%d",
            record.InvoiceNo,
            record.InvoiceDate.Format("2006-01-02"),
            record.Amount)
        if !existing[key] {
            newRecords = append(newRecords, record)
        }
    }

    // Bulk insert
    return r.bulkInsert(newRecords)
}
```

---

## 🔗 Related Documents

- [PRD.md](../PRD.md) - 完整產品需求文件（§ 2.5）
- [DOMAIN_MODEL.md](../../architecture/DOMAIN_MODEL.md) - IChefImportHistory Entity 設計
- [DATABASE_DESIGN.md](../../architecture/DATABASE_DESIGN.md) - ichef 相關表結構設計
- [ICHEF_IMPORT_README.md](../../ICHEF_IMPORT_README.md) - iChef 匯入詳細文件
- [US-002](./US-002-qr-code-scanning-points.md) - QR Code 掃描（匹配來源）
- [US-006](./US-006-admin-portal.md) - 管理後台（匯入介面）

---

## 📋 Future Enhancements (V3.5+)

### **V3.5: 進階匯入功能**
- 支援多種檔案格式（CSV, JSON）
- 自動排程匯入（每日自動從 iChef API 拉取）
- 差異比對（僅匯入新增/變更的發票）

### **V3.6: 智能匹配**
- 模糊匹配（容忍小額差異，如找零誤差）
- 機器學習輔助匹配
- 未匹配發票建議

### **V3.7: 進階報表**
- 匯入歷史趨勢分析
- 未匹配發票根因分析
- 自動異常檢測（如大量作廢）

---

**Story Created**: 2025-01-08
**Last Updated**: 2025-01-08
**Story Owner**: Product Team
**Technical Owner**: Backend Team
