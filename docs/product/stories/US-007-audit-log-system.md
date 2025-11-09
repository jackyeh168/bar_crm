# User Story 007: 稽核日誌系統 (Audit Log System)

**Story ID**: US-007
**Priority**: P1 (Should Have - Security & Compliance)
**Sprint**: Phase 3 - System Optimization
**Status**: ✅ Completed
**Estimated Effort**: 21 Story Points

---

## 📖 User Story

> **身為** 系統管理員或稽核人員 (Linda)，
> **我想要** 查詢系統中所有資料變更的完整歷史記錄，
> **以便** 進行安全稽核、問題追蹤、合規檢查和資料恢復。

---

## ✅ Acceptance Criteria

### **AC-1: 查詢稽核日誌**

**Given** 管理員訪問稽核日誌頁面
**When** 查詢特定會員的操作歷史
**Then** 顯示該會員所有資料變更記錄（時間倒序）

**Given** 管理員篩選特定時間範圍
**When** 選擇「2025-01-01 至 2025-01-31」
**Then** 僅顯示該期間的稽核記錄

**Given** 管理員篩選特定操作類型
**When** 選擇「積分變動」
**Then** 顯示所有積分賺取/扣除/重算記錄

### **AC-2: 追蹤問題**

**Given** 會員回報積分異常
**When** 管理員查詢該會員的積分變動日誌
**Then** 顯示完整的積分增減歷史和觸發原因

**Given** 發現重複交易
**When** 管理員查詢該交易的稽核日誌
**Then** 顯示交易創建、狀態變更、iChef 匹配的完整歷史

### **AC-3: 合規檢查**

**Given** 監管單位要求提供特定會員的資料處理記錄
**When** 管理員匯出該會員的稽核日誌
**Then** 生成包含所有資料變更的 CSV/PDF 報告

**Given** GDPR 資料刪除請求
**When** 管理員刪除會員資料
**Then** 系統記錄刪除操作（包含請求 ID、刪除時間、操作者）

### **AC-4: 事務一致性**

**Given** 業務操作需要記錄稽核日誌
**When** 稽核日誌寫入失敗
**Then** 業務操作自動回滾，確保 100% 記錄完整性

**Given** 業務操作成功
**When** 提交事務
**Then** 稽核日誌與業務資料同時提交，保證同步

---

## 📋 Business Rules

### **記錄時機**

| Rule ID | Description |
|---------|-------------|
| BR-007-01 | **同步記錄**: 資料變更操作與稽核日誌創建必須在同一資料庫事務中 |
| BR-007-02 | **原子性**: 如果稽核日誌寫入失敗，原操作必須回滾 |
| BR-007-03 | **不可變性**: 稽核日誌一旦寫入，不允許修改或刪除 |
| BR-007-04 | **完整性**: 所有 CREATE、UPDATE、DELETE 操作都必須記錄 |

### **稽核範圍**

| 資料類型 | CREATE | UPDATE | DELETE | 保存期限 |
|---------|--------|--------|--------|---------|
| 會員資料 | ✅ | ✅ | ✅ | 永久 |
| 積分帳戶 | ✅ | ✅ | ❌ | 永久 |
| 交易記錄 | ✅ | ✅ | ❌ | 7 年 |
| 問卷 | ✅ | ✅ | ✅ | 5 年 |
| 問卷回應 | ✅ | ❌ | ❌ | 5 年 |
| 積分規則 | ✅ | ✅ | ✅ | 永久 |
| iChef 匯入 | ✅ | ✅ | ❌ | 7 年 |
| 管理員操作 | ✅ | ✅ | ✅ | 3年-永久 |

### **查詢權限**

| Rule ID | Description |
|---------|-------------|
| BR-007-05 | **Admin**: 可查詢所有稽核日誌 |
| BR-007-06 | **User**: 可查詢會員、交易、問卷相關日誌（唯讀） |
| BR-007-07 | **Guest**: 無權限訪問稽核日誌 |
| BR-007-08 | **系統審計員**: 特殊角色，僅查詢權限（未來 V3.2+） |

### **保存策略**

| Rule ID | Description |
|---------|-------------|
| BR-007-09 | **熱資料** (最近 1 年): 主資料庫，支援快速查詢 |
| BR-007-10 | **溫資料** (1-3 年): 歸檔資料庫，查詢稍慢 |
| BR-007-11 | **冷資料** (3 年以上): 壓縮備份，按需恢復查詢 |
| BR-007-12 | **永久保存**: 會員、積分、規則相關日誌永不刪除 |

### **敏感資料保護**

| Rule ID | Description |
|---------|-------------|
| BR-007-13 | 手機號碼部分遮罩顯示（如 `0912****678`） |
| BR-007-14 | IP 位址僅記錄前 3 段（如 `192.168.1.*`） |
| BR-007-15 | 問卷回應內容需加密存儲 |

### **效能要求**

| Rule ID | Description |
|---------|-------------|
| BR-007-16 | 稽核日誌寫入不應顯著影響業務操作效能（< 50ms overhead） |
| BR-007-17 | 稽核日誌查詢應支援分頁（每頁 100 筆） |
| BR-007-18 | 複雜查詢（多條件篩選）應在 5 秒內返回結果 |

---

## 🔧 Technical Implementation Notes

### **Audit Log Schema**

```go
// Domain Entity
type AuditLog struct {
    AuditID     AuditID       // Value Object - "AUD-20250109-123456-ABC123"
    Timestamp   time.Time
    EventType   EventType     // Value Object - MEMBER_CREATED, POINTS_EARNED, etc.
    Actor       Actor         // Value Object - Who performed the action
    Target      Target        // Value Object - What was affected
    Action      ActionType    // Value Object - CREATE, UPDATE, DELETE
    Changes     Changes       // Value Object - Before/After comparison
    Metadata    Metadata      // Value Object - Additional context
    Result      Result        // Value Object - SUCCESS or FAILURE
}

// Actor Value Object
type Actor struct {
    Type        ActorType     // MEMBER, ADMIN, SYSTEM
    ID          string        // M123, A456, SYSTEM
    Name        string        // 小陳, 王姐, Auto Recalculation
    IPAddress   string        // 192.168.1.100 (masked to 192.168.1.*)
    UserAgent   string        // LINE/10.0.0, Chrome/120.0
}

// Target Value Object
type Target struct {
    Type        TargetType    // MEMBER, TRANSACTION, POINTS_ACCOUNT, SURVEY, etc.
    ID          string        // M123, TX456, PA789
    Description string        // 會員小陳, 發票 AB12345678
}

// Changes Value Object
type Changes struct {
    Before map[string]interface{}  // Original state
    After  map[string]interface{}  // New state
    Diff   map[string]string       // Human-readable diff
}

// Metadata Value Object
type Metadata struct {
    Reason              string  // 原因
    BatchID             string  // 批次 ID（如果適用）
    RelatedTransactionID string // 相關交易 ID
    SurveyCompleted     bool    // 問卷是否完成（如果適用）
    CustomData          map[string]interface{}  // 其他自訂欄位
}
```

### **Event Types (EventType Value Object)**

```go
type EventType string

const (
    // Member Events
    EventMemberCreated        EventType = "MEMBER_CREATED"
    EventMemberPhoneUpdated   EventType = "MEMBER_PHONE_UPDATED"
    EventMemberDeleted        EventType = "MEMBER_DELETED"

    // Points Account Events
    EventPointsEarned         EventType = "POINTS_EARNED"
    EventPointsDeducted       EventType = "POINTS_DEDUCTED"
    EventPointsRecalculated   EventType = "POINTS_RECALCULATED"

    // Transaction Events
    EventTransactionCreated   EventType = "TRANSACTION_CREATED"
    EventTransactionStatusChanged EventType = "TRANSACTION_STATUS_CHANGED"
    EventTransactionMatched   EventType = "TRANSACTION_MATCHED"

    // Survey Events
    EventSurveyCreated        EventType = "SURVEY_CREATED"
    EventSurveyActivated      EventType = "SURVEY_ACTIVATED"
    EventSurveyDeactivated    EventType = "SURVEY_DEACTIVATED"
    EventSurveyDeleted        EventType = "SURVEY_DELETED"
    EventSurveyResponseCreated EventType = "SURVEY_RESPONSE_CREATED"

    // Conversion Rule Events
    EventConversionRuleCreated EventType = "CONVERSION_RULE_CREATED"
    EventConversionRuleUpdated EventType = "CONVERSION_RULE_UPDATED"
    EventConversionRuleDeleted EventType = "CONVERSION_RULE_DELETED"

    // Import Batch Events
    EventImportBatchCreated   EventType = "IMPORT_BATCH_CREATED"
    EventImportBatchCompleted EventType = "IMPORT_BATCH_COMPLETED"

    // Admin Events
    EventAdminLogin           EventType = "ADMIN_LOGIN"
    EventAdminLogout          EventType = "ADMIN_LOGOUT"
    EventAdminRoleChanged     EventType = "ADMIN_ROLE_CHANGED"
    EventAdminSensitiveOperation EventType = "ADMIN_SENSITIVE_OPERATION"
)
```

### **Repository Interface**

```go
// Domain Layer
package repository

type AuditLogRepository interface {
    // Create audit log (must be called within same transaction)
    Create(log *AuditLog) error
    CreateBatch(logs []*AuditLog) error

    // Query audit logs
    FindAll(filter AuditLogFilter, pagination Pagination) ([]*AuditLog, int, error)
    FindByTarget(targetType TargetType, targetID string, pagination Pagination) ([]*AuditLog, int, error)
    FindByActor(actorType ActorType, actorID string, pagination Pagination) ([]*AuditLog, int, error)
    FindByEventType(eventType EventType, pagination Pagination) ([]*AuditLog, int, error)
    FindByDateRange(startDate time.Time, endDate time.Time, pagination Pagination) ([]*AuditLog, int, error)

    // Count for statistics
    CountByEventType(eventType EventType) (int, error)
    CountByActor(actorType ActorType, actorID string) (int, error)

    // Export for compliance
    ExportToCSV(filter AuditLogFilter) ([]byte, error)
    ExportToPDF(filter AuditLogFilter) ([]byte, error)

    // Archive management
    ArchiveOldLogs(olderThan time.Time) error  // Move to warm storage
    DeleteArchivedLogs(olderThan time.Time) error  // Delete from cold storage (only if permitted)
}

// Filter Value Object
type AuditLogFilter struct {
    EventTypes   []EventType
    ActorTypes   []ActorType
    TargetTypes  []TargetType
    StartDate    *time.Time
    EndDate      *time.Time
    ActorID      string
    TargetID     string
    SearchText   string  // Search in description, metadata
}

// Pagination Value Object
type Pagination struct {
    Page     int  // 1-based
    PageSize int  // Default 100, max 500
    SortBy   string  // Default "timestamp"
    SortOrder string  // "asc" or "desc", default "desc"
}
```

### **Use Case: Record Audit Log**

```go
// Application Layer
package usecases

type RecordAuditLogUseCase struct {
    auditLogRepo repository.AuditLogRepository
}

// Must be called within the same transaction as the business operation
func (uc *RecordAuditLogUseCase) Execute(cmd RecordAuditLogCommand) error {
    // Create audit log entity
    log := &AuditLog{
        AuditID:   GenerateAuditID(),
        Timestamp: time.Now(),
        EventType: cmd.EventType,
        Actor:     cmd.Actor,
        Target:    cmd.Target,
        Action:    cmd.Action,
        Changes:   cmd.Changes,
        Metadata:  cmd.Metadata,
        Result:    Result{Status: "SUCCESS"},
    }

    // Validate
    if err := log.Validate(); err != nil {
        return err
    }

    // Save (within same transaction)
    if err := uc.auditLogRepo.Create(log); err != nil {
        // If audit log fails, the entire transaction will rollback
        return fmt.Errorf("failed to record audit log: %w", err)
    }

    return nil
}
```

### **Database Schema**

```sql
CREATE TABLE audit_logs (
    audit_id VARCHAR(50) PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type VARCHAR(50) NOT NULL,

    -- Actor
    actor_type VARCHAR(20) NOT NULL,  -- MEMBER, ADMIN, SYSTEM
    actor_id VARCHAR(50) NOT NULL,
    actor_name VARCHAR(100),
    actor_ip VARCHAR(15),  -- Masked: 192.168.1.*
    actor_user_agent TEXT,

    -- Target
    target_type VARCHAR(50) NOT NULL,  -- MEMBER, TRANSACTION, SURVEY, etc.
    target_id VARCHAR(50) NOT NULL,
    target_description VARCHAR(255),

    -- Action
    action VARCHAR(10) NOT NULL,  -- CREATE, UPDATE, DELETE

    -- Changes (JSONB for PostgreSQL)
    changes_before JSONB,
    changes_after JSONB,
    changes_diff JSONB,

    -- Metadata
    metadata JSONB,

    -- Result
    result_status VARCHAR(20) NOT NULL,  -- SUCCESS, FAILURE
    result_error_message TEXT,

    -- Indexes
    INDEX idx_audit_timestamp (timestamp DESC),
    INDEX idx_audit_event_type (event_type),
    INDEX idx_audit_actor (actor_type, actor_id),
    INDEX idx_audit_target (target_type, target_id),
    INDEX idx_audit_action (action),

    -- Composite indexes for common queries
    INDEX idx_audit_target_timestamp (target_type, target_id, timestamp DESC),
    INDEX idx_audit_event_timestamp (event_type, timestamp DESC)
);

-- Partition by month for better performance (PostgreSQL 10+)
CREATE TABLE audit_logs_2025_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

### **Transaction Pattern Example**

```go
// Application Layer - EarnPointsUseCase
func (uc *EarnPointsUseCase) Execute(cmd EarnPointsCommand) error {
    return uc.txManager.InTransaction(func(ctx context.Context) error {
        // 1. Business operation: Update points account
        account, err := uc.accountRepo.FindByMemberID(ctx, cmd.MemberID)
        if err != nil {
            return err
        }

        oldPoints := account.EarnedPoints()

        err = account.EarnPoints(cmd.Amount, cmd.Source, cmd.SourceID, cmd.Description)
        if err != nil {
            return err
        }

        err = uc.accountRepo.Update(ctx, account)
        if err != nil {
            return err
        }

        // 2. Audit log: Record in same transaction
        auditCmd := RecordAuditLogCommand{
            EventType: EventPointsEarned,
            Actor: Actor{
                Type: ActorTypeMember,
                ID:   cmd.MemberID,
                Name: account.MemberName(),
            },
            Target: Target{
                Type: TargetTypePointsAccount,
                ID:   account.AccountID(),
                Description: fmt.Sprintf("積分帳戶 - 會員 %s", account.MemberName()),
            },
            Action: ActionUpdate,
            Changes: Changes{
                Before: map[string]interface{}{"earned_points": oldPoints},
                After:  map[string]interface{}{"earned_points": account.EarnedPoints()},
                Diff:   map[string]string{"earned_points": fmt.Sprintf("+%d", cmd.Amount)},
            },
            Metadata: Metadata{
                Reason:              "從交易獲得積分",
                RelatedTransactionID: cmd.SourceID,
            },
        }

        err = uc.recordAuditLogUseCase.Execute(ctx, auditCmd)
        if err != nil {
            // If audit log fails, entire transaction rollbacks
            return fmt.Errorf("audit log failed: %w", err)
        }

        // Both operations commit together
        return nil
    })
}
```

---

## 🧪 Test Cases

### **Unit Tests**

```go
func TestAuditLog_Create(t *testing.T) {
    // Test audit log creation with all required fields
}

func TestAuditLog_Validate(t *testing.T) {
    // Test validation of audit log fields
}

func TestAuditLog_Immutability(t *testing.T) {
    // Test that audit logs cannot be modified after creation
}

func TestAuditLogRepository_TransactionConsistency(t *testing.T) {
    // Test that audit log and business operation are in same transaction
    // If audit log fails, business operation should rollback
}

func TestAuditLogRepository_Query(t *testing.T) {
    // Test various query filters
}

func TestAuditLogRepository_Pagination(t *testing.T) {
    // Test pagination with large datasets
}

func TestAuditLog_SensitiveDataMasking(t *testing.T) {
    // Test phone number and IP masking
}
```

### **Integration Tests**

```go
func TestAuditLog_EndToEnd(t *testing.T) {
    // 1. Perform business operation (e.g., EarnPoints)
    // 2. Verify audit log was created
    // 3. Verify both are in database
    // 4. Verify audit log contains correct before/after values
}

func TestAuditLog_TransactionRollback(t *testing.T) {
    // 1. Mock audit log repository to fail
    // 2. Attempt business operation
    // 3. Verify business operation was rolled back
    // 4. Verify no data was persisted
}

func TestAuditLog_Export(t *testing.T) {
    // Test CSV and PDF export functionality
}
```

### **Performance Tests**

```go
func BenchmarkAuditLog_Write(b *testing.B) {
    // Verify < 50ms overhead for audit log writing
}

func BenchmarkAuditLog_Query(b *testing.B) {
    // Verify < 3s for complex queries
}
```

---

## 📦 Dependencies

### **Internal Dependencies**

- **US-001 to US-006**: All user stories generate audit logs
- **TransactionManager**: For ensuring audit logs are in same transaction

### **External Dependencies**

- PostgreSQL: Audit log storage with JSONB support
- Time-series database (optional): For high-volume audit log analytics

### **Service Dependencies**

- `AuditLogRepository`: 稽核日誌資料存取
- `TransactionManager`: 事務管理
- `MaskingService`: 敏感資料遮罩

---

## 📊 Success Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| **稽核日誌完整性** | 100% | 成功記錄的操作 / 總操作數 |
| **稽核日誌寫入開銷** | < 50 ms | 平均稽核日誌寫入時間 |
| **稽核日誌查詢性能** | < 3 秒 | 查詢回應時間（中位數） |
| **事務一致性** | 100% | 稽核日誌與業務資料同步率 |
| **資料不可變性** | 100% | 稽核日誌修改/刪除嘗試阻止率 |
| **匯出成功率** | > 99% | 成功匯出 / 總匯出請求 |

---

## 🎯 User Personas

**Primary Persona**: 稽核專員 Linda
- 30-40 歲內部稽核人員或外部監管單位
- 負責系統稽核與合規檢查
- 需要追蹤所有資料變更歷史
- 期望完整的稽核日誌系統，可以快速查詢特定會員或操作的歷史記錄
- 支援匯出符合監管要求的報告
- 能偵測異常操作

**Secondary Persona**: Admin 管理員
- 25-35 歲技術人員
- 負責問題追蹤與調查
- 需要查詢歷史操作以解決用戶問題
- 期望快速定位問題原因

---

## 📝 UI/UX Flow

### **Audit Log List Page**

```
┌─────────────────────────────────────────────────────────┐
│ 🔍 稽核日誌查詢                                         │
├─────────────────────────────────────────────────────────┤
│ 篩選條件:                                               │
│ [時間範圍: 最近 7 天 ▼] [操作類型: 全部 ▼]             │
│ [操作者: _______] [目標資源: _______] [搜尋]           │
├─────────────────────────────────────────────────────────┤
│ 時間              操作類型       操作者    目標資源      │
├─────────────────────────────────────────────────────────┤
│ 2025-01-09 14:30  積分賺取       小陳      積分帳戶  [詳情]│
│ 2025-01-09 14:25  交易狀態變更   王姐      交易 TX123 [詳情]│
│ 2025-01-09 14:20  會員註冊       小美      會員 M456  [詳情]│
│ ...                                                     │
├─────────────────────────────────────────────────────────┤
│ 第 1 頁，共 50 頁   [上一頁][下一頁]    [匯出 CSV/PDF]  │
└─────────────────────────────────────────────────────────┘
```

### **Audit Log Detail Page**

```
┌─────────────────────────────────────────────────────────┐
│ 📋 稽核日誌詳情                                         │
├─────────────────────────────────────────────────────────┤
│ 稽核 ID: AUD-20250109-143045-ABC123                     │
│ 時間: 2025-01-09 14:30:45                               │
│ 事件類型: 積分賺取 (POINTS_EARNED)                      │
│                                                         │
│ 操作者:                                                 │
│  - 類型: 會員 (MEMBER)                                  │
│  - ID: M123                                             │
│  - 名稱: 小陳                                           │
│  - IP 位址: 192.168.1.*                                 │
│  - User Agent: LINE/10.0.0                              │
│                                                         │
│ 目標資源:                                               │
│  - 類型: 積分帳戶 (POINTS_ACCOUNT)                      │
│  - ID: PA789                                            │
│  - 描述: 積分帳戶 - 會員小陳                            │
│                                                         │
│ 操作: 更新 (UPDATE)                                     │
│                                                         │
│ 變更內容:                                               │
│  ┌─────────────────────────────────────────┐           │
│  │ 欄位         變更前    變更後    差異    │           │
│  ├─────────────────────────────────────────┤           │
│  │ earned_points  100      103      +3     │           │
│  └─────────────────────────────────────────┘           │
│                                                         │
│ 元數據:                                                 │
│  - 原因: 從交易獲得積分                                 │
│  - 相關交易 ID: TX456                                   │
│  - 問卷完成: 是                                         │
│                                                         │
│ 執行結果: 成功 (SUCCESS)                                │
│                                                         │
│ [返回列表]                                              │
└─────────────────────────────────────────────────────────┘
```

---

## 🔗 Related Documents

- [PRD.md](../PRD.md) - 完整產品需求文件（§ 2.7: 稽核日誌系統）
- [DOMAIN_MODEL.md](../../architecture/DOMAIN_MODEL.md) - AuditLog Entity 設計
- [DATABASE_DESIGN.md](../../architecture/DATABASE_DESIGN.md) - audit_logs 表結構設計
- [DDD Architecture](../../architecture/ddd/02-bounded-contexts.md) - Audit Context 設計
- [SECURITY.md](../../operations/SECURITY.md) - 安全性與合規文件

---

## 📋 GDPR Compliance Checklist

### **Right to be Informed**

- [x] 系統記錄所有個人資料處理活動
- [x] 稽核日誌包含處理目的和法律依據

### **Right of Access**

- [x] 使用者可透過管理員請求查看其個人資料處理歷史
- [x] 支援匯出個人資料處理報告（CSV/PDF）

### **Right to Erasure (Right to be Forgotten)**

- [x] 會員資料刪除操作完整記錄在稽核日誌
- [x] 刪除請求包含請求 ID、原因、操作者
- [x] 刪除操作永久保存在稽核日誌中

### **Right to Data Portability**

- [x] 支援匯出個人資料及處理歷史
- [x] 匯出格式: CSV, PDF

### **Data Breach Notification**

- [x] 異常操作監控（單日積分變動 > 1000 點）
- [x] 異常登入行為監控（短時間多次失敗登入）
- [x] 批量操作監控（單次影響 > 100 筆記錄）
- [x] 異常操作即時通知管理員

---

## 📋 Future Enhancements (V3.2+)

### **V3.2: 進階稽核功能**
- 稽核日誌異常偵測 (Machine Learning)
- 即時稽核告警 (Real-time Alerts)
- 稽核儀表板 (Audit Dashboard)
- 稽核日誌視覺化 (Timeline View)

### **V3.3: 合規報告**
- 自動生成監管報告
- 合規檢查清單
- 定期稽核報告排程

### **V3.4: Event Sourcing**
- 完整事件溯源架構
- 時間旅行查詢（查詢任意時間點的系統狀態）
- 事件重播 (Event Replay)
- CQRS 架構整合

---

**Story Created**: 2025-01-09
**Last Updated**: 2025-01-09
**Story Owner**: Product Team & Security Team
**Technical Owner**: Backend Team
