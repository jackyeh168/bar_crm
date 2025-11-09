# Use Case 定義（應用層規格）

> **版本**: 1.0
> **最後更新**: 2025-01-08

本章節定義關鍵 Use Case 的輸入輸出契約和執行流程，這些規格指導 Application Service 的實現。

---

## **10.1 會員管理 Use Cases**

### **UC-001: RegisterMember (會員註冊)**

**輸入 (Command)**:
```go
type RegisterMemberCommand struct {
    LineUserID  string  // LINE Platform 用戶 ID
    DisplayName string  // LINE 顯示名稱
    PhoneNumber string  // 手機號碼 (原始輸入，未驗證)
}
```

**輸出 (Result)**:
```go
type RegisterMemberResult struct {
    MemberID       string    // 新創建的會員 ID
    RegisteredAt   time.Time // 註冊時間
}
```

**執行流程**:
1. 驗證 `LineUserID` 格式（必須以 "U" 開頭）
2. 創建 `PhoneNumber` 值對象（自動驗證格式：10 位數，09 開頭）
3. 檢查 `PhoneNumber` 唯一性：`MemberRepository.ExistsByPhoneNumber()`
   - 如果已存在：返回 `ErrPhoneNumberAlreadyBound`
4. 檢查 `LineUserID` 唯一性：`MemberRepository.ExistsByLineUserID()`
   - 如果已存在：返回 `ErrMemberAlreadyExists`
5. 創建 `Member` 聚合：`Member.NewMember(lineUserID, displayName)`
6. 綁定手機號碼：`member.BindPhoneNumber(phoneNumber)`
7. 保存到倉儲：`MemberRepository.Create(member)`
8. 發布領域事件：`MemberRegistered`
9. 返回 `RegisterMemberResult`

**事務邊界**: 單一資料庫事務

**授權**: 公開（LINE Bot 用戶）

**錯誤處理**:
- `ErrPhoneNumberAlreadyBound` → HTTP 400
- `ErrMemberAlreadyExists` → HTTP 409
- `ErrInvalidPhoneNumberFormat` → HTTP 400
- `ErrInvalidLineUserID` → HTTP 400

---

## **10.2 積分管理 Use Cases**

### **UC-002: EarnPointsFromTransaction (從交易獲得積分)**

**輸入 (Event)**:
```go
type TransactionVerifiedEvent struct {
    TransactionID   string    // 交易 ID
    MemberID        string    // 會員 ID
    Amount          decimal   // 消費金額
    InvoiceDate     string    // 發票日期 (yyyMMdd)
    SurveySubmitted bool      // 是否已完成問卷
}
```

**輸出**:
```go
// 無返回值（事件處理器）
```

**執行流程** (輕量級聚合設計):
1. 查詢積分帳戶：`PointsAccountRepository.FindByMemberID(memberID)`
   - 如果不存在：創建新帳戶 `NewPointsAccount(memberID)`
2. 查詢轉換規則：`ConversionRuleQueryService.GetRuleForDate(invoiceDate)`
   - 如果找不到：返回 `ErrNoConversionRuleForDate`
3. 計算積分：使用 DTO + Strategy Pattern
   ```go
   dto := VerifiedTransactionDTO{
       TransactionID: transactionID,
       Amount: amount,
       InvoiceDate: invoiceDate,
       SurveySubmitted: surveySubmitted,
   }
   totalPoints := calculator.CalculateForTransaction(dto, ruleService)
   ```
4. 更新帳戶狀態：`account.EarnPoints(totalPoints, SourceInvoice, transactionID, description)`
5. 保存帳戶狀態：`PointsAccountRepository.Update(account)`
6. 創建交易記錄（審計日誌）：
   ```go
   transaction := NewPointsTransaction(
       accountID: account.AccountID,
       type: TypeEarned,
       amount: totalPoints,
       source: SourceInvoice,
       sourceID: transactionID,
       description: "從發票獲得積分",
   )
   PointsTransactionRepository.Create(transaction)
   ```
7. 觸發通知：`NotificationService.SendPointsEarnedNotification(memberID, totalPoints)`

**事務邊界**: 單一資料庫事務（帳戶狀態 + 交易記錄）

**設計優勢**:
- ✅ 聚合輕量級：PointsAccount 只包含狀態，加載快速
- ✅ 交易歷史獨立：PointsTransaction 單獨管理，支持分頁查詢
- ✅ 審計完整性：交易記錄永久保存，不可變
- ✅ 性能優化：批量加載帳戶不受交易數量影響

**授權**: 系統內部（事件處理器）

**錯誤處理**:
- `ErrNoConversionRuleForDate` → 記錄警告日誌，跳過積分計算
- `ErrAccountNotFound` → 自動創建新帳戶
- 其他錯誤 → 記錄錯誤，重試機制（消息隊列）

---

### **UC-003: RecalculateAllPoints (重算所有積分)**

**輸入 (Command)**:
```go
type RecalculateAllPointsCommand struct {
    AdminUserID string  // 觸發重算的管理員 ID
    Force       bool    // 是否強制執行（跳過確認）
}
```

**輸出 (Result)**:
```go
type RecalculateAllPointsResult struct {
    TotalMembers       int           // 處理的會員數
    TotalTransactions  int           // 處理的交易數
    Duration           time.Duration // 執行時間
    UpdatedAt          time.Time     // 完成時間
}
```

**執行流程** (Application Layer - 協調器模式):
1. 檢查是否有重算正在進行：`CheckRecalculationInProgress()`
   - 如果進行中：返回 `ErrRecalculationInProgress`
2. 使用 TransactionManager 開啟事務
3. 在事務中執行：
   a. 查詢所有積分帳戶：`accountRepo.FindAll(ctx)`
   b. 對每個會員：
      - 查詢所有已驗證交易：`txRepo.FindVerifiedByMemberID(ctx, memberID)`
      - **在 Application Layer 計算積分**（協調業務邏輯）
      - **調用聚合方法設置狀態**：`account.SetEarnedPoints(newTotal)`
      - 保存帳戶：`accountRepo.Update(ctx, account)`
   c. 如果任何錯誤：自動回滾事務
   d. 否則：提交事務
4. 發布領域事件：`PointsRecalculated`（由聚合方法觸發）
5. 記錄審計日誌：`AuditLog.Record("RecalculateAllPoints", adminUserID, result)`
6. 返回 `RecalculateAllPointsResult`

**關鍵設計原則**:
- ✅ **Use Case 協調業務流程**: 獲取數據 → 計算積分 → 更新聚合 → 保存結果
- ✅ **聚合負責不變性保護**: `SetEarnedPoints()` 拒絕負數，發布事件
- ✅ **符合單一職責原則 (SRP)**: 聚合不協調工作流，只管理狀態
- ✅ **事務管理在 Application Layer**: 使用 TransactionManager，不污染 Domain 接口
- ✅ **Strategy Pattern 支持擴展**: 計算邏輯使用 Composite Strategy

**代碼示例** (修正後的設計):
```go
func (uc *RecalculateAllPointsUseCase) Execute(cmd RecalculateAllPointsCommand) error {
    return uc.txManager.InTransaction(func(ctx Context) error {
        accounts := uc.accountRepo.FindAll(ctx)

        for _, account := range accounts {
            // 1. 從 Invoice Context 查詢已驗證交易（實體）
            invoiceTransactions := uc.invoiceTxRepo.FindVerifiedByMemberID(ctx, account.MemberID())

            // 2. Application Layer 負責 DTO 轉換（避免跨聚合實體引用）
            transactionDTOs := make([]VerifiedTransactionDTO, len(invoiceTransactions))
            for i, tx := range invoiceTransactions {
                transactionDTOs[i] = VerifiedTransactionDTO{
                    TransactionID:   tx.ID(),
                    Amount:          tx.Amount(),
                    InvoiceDate:     tx.InvoiceDate(),
                    SurveySubmitted: tx.IsSurveySubmitted(),
                }
            }

            // 3. 調用 Domain Service 計算總積分（業務邏輯在 Domain Layer）
            totalPoints := uc.calculator.CalculateTotalPoints(transactionDTOs, uc.ruleService)

            // 4. 聚合只負責狀態更新和不變性保護
            err := account.SetEarnedPoints(PointsAmount(totalPoints))
            if err != nil {
                return err // ErrNegativePointsAmount
            }

            // 5. 保存狀態
            uc.accountRepo.Update(ctx, account)

            // 注意: 積分重算不創建新的 PointsTransaction 記錄
            // 原因: 交易歷史已存在（來自原始的 EarnPoints 操作）
            // 重算只更新帳戶狀態（EarnedPoints），不重複記錄交易
        }
        return nil
    })
}
```

**Domain Service 實現** (PointsCalculationService):

```go
// Domain Layer - internal/domain/points/service.go
package points

import "myapp/internal/application/dto"

type PointsCalculationService struct {
    // 無狀態服務，可注入配置或策略
}

// CalculateTotalPoints 計算多筆交易的總積分（業務邏輯）
// ✅ 業務規則封裝在 Domain Service，而非 Use Case
func (s *PointsCalculationService) CalculateTotalPoints(
    transactions []dto.VerifiedTransactionDTO,
    ruleService ConversionRuleService,
) int {
    totalPoints := 0

    for _, tx := range transactions {
        points := s.CalculateForTransaction(tx, ruleService)
        totalPoints += points
    }

    return totalPoints
}

// CalculateForTransaction 計算單筆交易的積分
func (s *PointsCalculationService) CalculateForTransaction(
    tx dto.VerifiedTransactionDTO,
    ruleService ConversionRuleService,
) int {
    // 1. 查詢轉換規則
    rule := ruleService.GetRuleForDate(tx.InvoiceDate)

    // 2. 計算基礎積分（向下取整）
    basePoints := int(tx.Amount.Div(rule.ConversionRate).IntPart())

    // 3. 問卷加成
    surveyBonus := 0
    if tx.SurveySubmitted {
        surveyBonus = 1
    }

    return basePoints + surveyBonus
}
```

**關鍵設計原則**:
- ✅ **業務邏輯在 Domain Service**: 累加邏輯、計算規則都在 Domain Layer
- ✅ **Use Case 只做編排**: 獲取數據 → 調用服務 → 保存結果
- ✅ **Domain Service 無狀態**: 可安全地被多個 Use Case 共用
- ✅ **符合 Single Responsibility Principle**: Use Case 負責流程，Service 負責計算

**設計優勢**:
- ✅ **DTO 轉換在 Application Layer**: 將 Invoice 實體轉換為 DTO
- ✅ **業務邏輯在 Domain Service**: 點數計算、累加邏輯封裝在 Service
- ✅ **Use Case 純編排**: 獲取數據 → 調用服務 → 更新狀態 → 保存
- ✅ **狀態管理在 Aggregate**: 聚合負責不變性和事件發布
- ✅ **聚合邊界清晰**: Points Context 和 Invoice Context 通過 DTO 解耦
- ✅ **防止跨聚合引用**: 避免一個聚合直接持有另一個聚合的實體
- ✅ **易於測試**: Domain Service 可獨立測試，無需 Mock Repository
- ✅ **輕量級聚合**: FindAll() 高效加載，不含交易歷史（性能優勢）

**事務邊界**: 單一大型資料庫事務（鎖表 30-60 秒）

**授權**: 僅管理員（Admin 角色）

**性能考量**:
- 鎖表時間：30-60 秒
- 執行時機：離峰時段（凌晨 3:00-5:00）
- 需要用戶確認（CLI 提示）

**錯誤處理**:
- 任何錯誤：回滾整個事務
- `ErrRecalculationInProgress` → 提示管理員稍後再試
- 事務超時 → 記錄錯誤，發送告警

---

## **10.3 發票處理 Use Cases**

### **UC-004: ScanInvoiceQRCode (掃描發票 QR Code)**

**輸入 (Command)**:
```go
type ScanInvoiceQRCodeCommand struct {
    MemberID    string  // 會員 ID
    QRCodeData  string  // QR Code 原始資料
}
```

**輸出 (Result)**:
```go
type ScanInvoiceQRCodeResult struct {
    TransactionID      string    // 交易 ID
    EstimatedPoints    int       // 預估積分
    SurveyLink         string    // 問卷連結（可選）
    CreatedAt          time.Time // 創建時間
}
```

**執行流程**:
1. 解析 QR Code：`InvoiceParsingService.ParseQRCode(qrCodeData)`
   - 返回：`Invoice` (InvoiceNumber, InvoiceDate, Amount)
   - 錯誤：`ErrInvalidQRCodeFormat`, `ErrQRCodeParsingFailed`
2. 驗證發票：`InvoiceValidationService.ValidateInvoice(invoice)`
   - 檢查格式：`ValidateInvoiceNumber()`, `ValidateInvoiceDate()`
   - 檢查有效期：`CheckExpiry()` （60 天內）
   - 檢查重複：`CheckDuplicate()` （發票號碼唯一）
   - 錯誤：`ErrInvoiceExpired`, `ErrInvoiceDuplicate`, `ErrInvalidInvoiceFormat`
3. 查詢會員：`MemberRepository.FindByID(memberID)`
   - 錯誤：`ErrMemberNotFound`
4. 創建交易：`InvoiceTransaction.NewInvoiceTransaction(memberID, invoice, qrCodeData)`
   - 初始狀態：`Status = Imported`
5. 查詢啟用的問卷：`SurveyRepository.FindActiveSurvey()`
   - 如果存在：`transaction.LinkSurvey(surveyID)`
6. 保存交易：`InvoiceTransactionRepository.Create(transaction)`
7. 發布領域事件：`TransactionCreated`
8. 計算預估積分（僅供顯示）：
   - 查詢當前轉換規則：`ConversionRuleValidationService.GetRuleForDate(invoiceDate)`
   - 計算：`estimatedPoints = floor(amount / conversionRate)`
9. 如果有問卷，生成問卷連結：`SurveyService.GenerateSurveyLink(surveyID, transactionID)`
10. 返回 `ScanInvoiceQRCodeResult`

**事務邊界**: 單一資料庫事務

**授權**: 已註冊會員

**錯誤處理**:
- `ErrInvoiceDuplicate` → HTTP 409, 顯示「此發票已登錄」
- `ErrInvoiceExpired` → HTTP 400, 顯示「發票已過期（超過 60 天）」
- `ErrMemberNotFound` → HTTP 404, 提示先註冊
- `ErrInvalidQRCodeFormat` → HTTP 400, 顯示「無效的 QR Code 格式」

---

### **UC-005: ImportIChefBatch (iChef 批次匯入)**

**輸入 (Command)**:
```go
type ImportIChefBatchCommand struct {
    ExcelFile   io.Reader // Excel 檔案流
    AdminUserID string    // 管理員 ID
}
```

**輸出 (Result)**:
```go
type ImportIChefBatchResult struct {
    BatchID         string    // 批次 ID
    TotalRows       int       // 總行數
    MatchedCount    int       // 匹配數
    UnmatchedCount  int       // 未匹配數
    SkippedCount    int       // 跳過數
    DuplicateCount  int       // 重複數
    CompletedAt     time.Time // 完成時間
}
```

**執行流程**:
1. 創建批次記錄：`ImportBatch.NewImportBatch(fileName, adminUserID)`
   - 初始狀態：`Status = Processing`
2. 保存批次：`ImportBatchRepository.Create(batch)`
3. 解析 Excel 檔案：`ExcelParser.ParseIChefFile(excelFile)`
   - 返回：`[]IChefInvoiceDTO`
4. 對每筆 iChef 發票記錄：
   a. 正規化發票號碼、日期、金額
   b. 檢查重複：`ImportBatchRepository.IsDuplicate(invoiceNumber, date, amount)`
      - 如果重複：標記為 `Duplicate`, 跳過
   c. 創建 `ImportedInvoiceRecord` 實體
   d. 查詢匹配的交易：`InvoiceTransactionRepository.FindByInvoiceNumber(invoiceNumber)`
      - 如果找到 + 金額日期匹配：
        - 標記為 `Matched`
        - 驗證交易：`TransactionVerificationService.VerifyTransaction(transactionID)`
        - 發布 `TransactionVerified` 事件 → 觸發積分計算
      - 如果找不到：標記為 `Unmatched`
   e. 添加到 `batch.InvoiceRecords`
5. 批次保存所有 `ImportedInvoiceRecord`：`ImportBatchRepository.BulkCreateRecords(records)`
6. 更新批次狀態：`batch.Complete(statistics)`
   - 狀態：`Status = Completed`
7. 保存批次：`ImportBatchRepository.Update(batch)`
8. 發布領域事件：`BatchImportCompleted`
9. 返回 `ImportIChefBatchResult`

**事務邊界**: 單一資料庫事務（批次操作）

**授權**: 僅管理員（Admin 角色）

**性能考量**:
- 批次插入（每 1000 筆提交一次）
- 重複檢查使用 unique constraint
- 異步處理大檔案（超過 10000 筆）

**錯誤處理**:
- Excel 解析錯誤 → 更新批次狀態為 `Failed`
- 部分記錄失敗 → 記錄錯誤但繼續處理
- 事務失敗 → 回滾整個批次

---

## **10.4 稽核管理 Use Cases**

### **UC-006: RecordAuditLog (記錄稽核日誌)**

**重要**: 此 Use Case 不是獨立調用的，而是作為所有業務操作的一部分，在同一事務中執行。

**輸入 (Command)**:
```go
type RecordAuditLogCommand struct {
    EventType   EventType  // 事件類型（如 POINTS_EARNED, MEMBER_CREATED）
    Actor       Actor      // 操作者資訊
    Target      Target     // 目標資源資訊
    Action      ActionType // 操作類型（CREATE/UPDATE/DELETE）
    Changes     Changes    // 變更內容（前後對比）
    Metadata    Metadata   // 額外元數據
}
```

**輸出**:
```go
// 無返回值（內部操作）
// 錯誤返回時會觸發事務回滾
```

**執行流程** (在業務事務中調用):
1. 創建 `AuditLog` 聚合：`NewAuditLog(eventType, actor, target, action, changes, metadata)`
   - 自動生成: `AuditID`, `Timestamp`, `Result.Status = SUCCESS`
2. 驗證：`auditLog.Validate()`
   - 檢查所有必填欄位
   - 返回錯誤：`ErrMissingRequiredField`, `ErrInvalidEventType`
3. 敏感資料遮罩（如需要）:
   - IP 位址遮罩: `192.168.1.*`
   - 手機號碼遮罩: `0912****678`
4. 保存到倉儲：`AuditLogRepository.Create(ctx, auditLog)`
   - **重要**: ctx 必須包含業務操作的資料庫事務
   - 如果寫入失敗：返回 `ErrAuditLogWriteFailed`，觸發業務事務回滾
5. 不發布領域事件（避免循環依賴）

**事務邊界**: **必須在業務事務中調用**（強一致性要求）

**典型使用場景**:
```go
// Application Layer - 任何業務 Use Case
func (uc *SomeBusinessUseCase) Execute(cmd Command) error {
    return uc.txManager.InTransaction(func(ctx Context) error {
        // 1. 執行業務操作
        oldState := someEntity.GetState()
        someEntity.DoSomething()
        uc.someRepo.Update(ctx, someEntity)

        // 2. 記錄稽核日誌（同一事務）
        auditCmd := RecordAuditLogCommand{
            EventType: SOME_EVENT_TYPE,
            Actor:     Actor{Type: MEMBER, ID: cmd.ActorID},
            Target:    Target{Type: SOME_TARGET, ID: someEntity.ID()},
            Action:    UPDATE,
            Changes:   Changes{Before: oldState, After: someEntity.GetState()},
            Metadata:  Metadata{Reason: "業務原因"},
        }
        uc.recordAuditLogUseCase.Execute(ctx, auditCmd) // 失敗 → 回滾

        return nil // 同時提交
    })
}
```

**授權**: 系統內部（所有業務操作自動調用）

**錯誤處理**:
- `ErrAuditLogWriteFailed` → 業務事務回滾，業務操作也取消
- `ErrMissingRequiredField` → 開發時錯誤，應在測試階段修正
- 資料庫連接失敗 → 業務事務回滾

---

### **UC-007: QueryAuditLogs (查詢稽核日誌)**

**輸入 (Query)**:
```go
type QueryAuditLogsQuery struct {
    Filter     AuditLogFilter  // 篩選條件
    Pagination Pagination      // 分頁參數
    UserRole   string          // 查詢者角色（權限檢查）
}

type AuditLogFilter struct {
    EventTypes   []EventType  // 事件類型篩選（多選）
    ActorTypes   []ActorType  // 操作者類型篩選
    TargetTypes  []TargetType // 目標資源類型篩選
    StartDate    *time.Time   // 開始時間
    EndDate      *time.Time   // 結束時間
    ActorID      string       // 操作者 ID（精確匹配）
    TargetID     string       // 目標資源 ID（精確匹配）
    SearchText   string       // 全文搜索（描述、元數據）
    ActionTypes  []ActionType // 操作類型篩選
}

type Pagination struct {
    Page      int    // 頁碼（1-based）
    PageSize  int    // 每頁筆數（預設 100，最大 500）
    SortBy    string // 排序欄位（預設 "timestamp"）
    SortOrder string // 排序方向（"asc" 或 "desc"，預設 "desc"）
}
```

**輸出 (Result)**:
```go
type QueryAuditLogsResult struct {
    Logs       []*AuditLog // 稽核日誌列表
    TotalCount int         // 總筆數（用於分頁）
    PageCount  int         // 總頁數
    Page       int         // 當前頁碼
    PageSize   int         // 每頁筆數
}
```

**執行流程**:
1. 權限檢查：`AuthorizationService.CheckAuditLogAccess(userRole)`
   - **Admin**: 可查詢所有稽核日誌
   - **User**: 可查詢會員、交易、問卷相關日誌（唯讀）
   - **Guest**: 無權限 → 返回 `ErrGuestCannotViewAudit`
2. 驗證篩選條件：`ValidateFilter(filter)`
   - 檢查日期範圍：`StartDate <= EndDate`
   - 檢查分頁參數：`Page >= 1, PageSize <= 500`
   - 返回錯誤：`ErrInvalidDateRange`, `ErrInvalidPagination`
3. 應用權限篩選（如 User 角色）:
   - User 角色: 自動添加 `TargetTypes = [MEMBER, TRANSACTION, SURVEY]`
4. 查詢稽核日誌：`AuditLogRepository.FindAll(filter, pagination)`
   - 使用索引優化查詢性能
   - 性能目標: < 3 秒（中位數）
5. 敏感資料遮罩（前端顯示）:
   - 對每筆日誌調用: `log.GetMaskedActor()`
   - IP 位址: `192.168.1.*`
   - 手機號碼: `0912****678`
6. 計算分頁資訊：`PageCount = ceil(TotalCount / PageSize)`
7. 返回 `QueryAuditLogsResult`

**事務邊界**: 只讀操作（不需要事務）

**授權**: Admin, User（Guest 無權限）

**性能考量**:
- 使用資料庫索引優化查詢
- 複雜查詢應在 5 秒內完成
- 分頁每頁最多 500 筆
- 建議預設每頁 100 筆

**錯誤處理**:
- `ErrGuestCannotViewAudit` → HTTP 403
- `ErrInvalidDateRange` → HTTP 400
- `ErrInvalidPagination` → HTTP 400
- 資料庫查詢超時 → HTTP 504, 建議縮小查詢範圍

---

### **UC-008: QueryAuditLogsByTarget (查詢特定資源的稽核日誌)**

**輸入 (Query)**:
```go
type QueryAuditLogsByTargetQuery struct {
    TargetType TargetType  // 目標資源類型（MEMBER, TRANSACTION, etc.）
    TargetID   string      // 目標資源 ID
    Pagination Pagination  // 分頁參數
    UserRole   string      // 查詢者角色
    UserID     string      // 查詢者 ID（用於權限檢查）
}
```

**輸出 (Result)**:
```go
type QueryAuditLogsByTargetResult struct {
    Logs       []*AuditLog // 稽核日誌列表（時間倒序）
    TotalCount int         // 總筆數
    PageCount  int         // 總頁數
    Target     Target      // 目標資源資訊
}
```

**執行流程**:
1. 權限檢查：`AuthorizationService.CheckTargetAccess(userRole, userID, targetType, targetID)`
   - **Admin**: 可查詢任何資源
   - **User**: 可查詢會員、交易、問卷
   - **Member**: 只能查詢自己的資料（TargetType=MEMBER && TargetID=自己的 MemberID）
   - 返回錯誤：`ErrUnauthorizedAuditAccess`
2. 查詢稽核日誌：`AuditLogRepository.FindByTarget(targetType, targetID, pagination)`
   - 自動按 `timestamp DESC` 排序（最新在前）
3. 敏感資料遮罩（前端顯示）
4. 返回 `QueryAuditLogsByTargetResult`

**典型使用場景**:
- 會員查詢自己的操作歷史: `TargetType=MEMBER, TargetID=M123`
- 管理員追蹤特定交易的所有變更: `TargetType=TRANSACTION, TargetID=TX456`
- 稽核人員查詢特定問卷的修改歷史: `TargetType=SURVEY, TargetID=S789`

**事務邊界**: 只讀操作

**授權**: Admin, User, Member（依權限範圍）

**錯誤處理**:
- `ErrUnauthorizedAuditAccess` → HTTP 403
- `ErrInvalidPagination` → HTTP 400
- 目標資源不存在 → 返回空列表（不報錯）

---

### **UC-009: ExportAuditReport (匯出稽核報告)**

**輸入 (Command)**:
```go
type ExportAuditReportCommand struct {
    Filter         AuditLogFilter  // 篩選條件
    Format         ExportFormat    // 匯出格式（CSV/PDF）
    ReportTemplate string          // 報告模板（GDPR/內部稽核/監管報告）
    AdminUserID    string          // 匯出操作者 ID
}

type ExportFormat string
const (
    ExportFormatCSV ExportFormat = "CSV"
    ExportFormatPDF ExportFormat = "PDF"
)
```

**輸出 (Result)**:
```go
type ExportAuditReportResult struct {
    FileContent  []byte    // 檔案內容
    FileName     string    // 檔案名稱（含副檔名）
    ContentType  string    // MIME 類型
    RecordCount  int       // 匯出記錄數
    ExportedAt   time.Time // 匯出時間
}
```

**執行流程**:
1. 權限檢查：`AuthorizationService.CheckExportPermission(adminUserID)`
   - 僅 Admin 角色可匯出
   - 返回錯誤：`ErrUnauthorizedAuditAccess`
2. 驗證篩選條件：`ValidateFilter(filter)`
   - 檢查匯出筆數限制: 最多 10,000 筆
   - 返回錯誤：`ErrExceedExportLimit`
3. 查詢稽核日誌：`AuditLogRepository.FindAll(filter, Pagination{PageSize: 10000})`
   - 如果超過 10,000 筆: 返回 `ErrExceedExportLimit`
4. 根據格式匯出：
   - **CSV**: `AuditLogExportService.ExportToCSV(logs)`
     - 欄位: AuditID, Timestamp, EventType, ActorType, ActorID, TargetType, TargetID, Action, Changes, Metadata
     - 編碼: UTF-8 with BOM（Excel 相容）
   - **PDF**: `AuditLogExportService.ExportToPDF(logs, reportTemplate)`
     - 包含: 報告標題、匯出時間、篩選條件摘要、稽核日誌表格
     - 模板選項:
       - `GDPR`: GDPR 合規報告（包含個資處理記錄）
       - `Internal`: 內部稽核報告
       - `Regulatory`: 監管單位報告
5. 記錄匯出操作（稽核日誌）:
   ```go
   auditLog := NewAuditLog(
       eventType: AUDIT_REPORT_EXPORTED,
       actor:     Actor{Type: ADMIN, ID: adminUserID},
       target:    Target{Type: AUDIT_LOG, ID: "REPORT"},
       action:    CREATE,
       metadata:  Metadata{
           Format:      format,
           RecordCount: len(logs),
           FilterApplied: filter,
       },
   )
   ```
6. 生成檔案名稱：`audit_report_{timestamp}_{format}.{ext}`
   - 例如: `audit_report_20250109_143045_CSV.csv`
7. 返回 `ExportAuditReportResult`

**事務邊界**: 只讀操作（匯出操作本身也記錄稽核日誌）

**授權**: 僅 Admin 角色

**性能考量**:
- 限制單次匯出 10,000 筆記錄
- CSV 生成: < 10 秒 / 1000 筆
- PDF 生成: < 20 秒 / 1000 筆
- 大檔案建議使用異步處理 + 郵件通知

**錯誤處理**:
- `ErrExceedExportLimit` → HTTP 400, 提示「請縮小查詢範圍（最多 10,000 筆）」
- `ErrUnauthorizedAuditAccess` → HTTP 403
- 檔案生成失敗 → HTTP 500, 記錄錯誤日誌

---

### **UC-010: ExportGDPRReport (匯出 GDPR 個資處理報告)**

**輸入 (Command)**:
```go
type ExportGDPRReportCommand struct {
    MemberID    string  // 會員 ID
    RequestID   string  // GDPR 請求 ID（追蹤用）
    AdminUserID string  // 處理請求的管理員 ID
}
```

**輸出 (Result)**:
```go
type ExportGDPRReportResult struct {
    JSONReport   []byte    // JSON 格式（機器可讀）
    PDFReport    []byte    // PDF 格式（人類可讀）
    RecordCount  int       // 資料處理記錄數
    MemberInfo   MemberInfo // 會員基本資訊
    ExportedAt   time.Time // 匯出時間
}

type MemberInfo struct {
    MemberID    string
    PhoneNumber string // 部分遮罩
    DisplayName string
}
```

**執行流程**:
1. 權限檢查：`AuthorizationService.CheckGDPRPermission(adminUserID)`
2. 查詢會員基本資訊：`MemberRepository.FindByID(memberID)`
   - 如果不存在：返回 `ErrMemberNotFound`
3. 查詢會員的所有資料處理記錄：
   ```go
   filter := AuditLogFilter{
       TargetTypes: []TargetType{MEMBER, TRANSACTION, POINTS_ACCOUNT, SURVEY_RESPONSE},
       TargetID:    memberID,
   }
   logs := AuditLogRepository.FindAll(filter, Pagination{PageSize: 100000})
   ```
4. 生成 JSON 報告：`AuditLogExportService.ExportGDPRReportJSON(memberInfo, logs)`
   - 符合 GDPR Article 20（Right to Data Portability）
   - 機器可讀格式，便於資料遷移
5. 生成 PDF 報告：`AuditLogExportService.ExportGDPRReportPDF(memberInfo, logs)`
   - 包含:
     - 會員基本資訊
     - 資料處理活動摘要
     - 完整的操作歷史（時間、類型、變更內容）
     - GDPR 聲明（資料主體權利說明）
6. 記錄 GDPR 報告匯出操作（稽核日誌）:
   ```go
   auditLog := NewAuditLog(
       eventType: GDPR_REPORT_EXPORTED,
       actor:     Actor{Type: ADMIN, ID: adminUserID},
       target:    Target{Type: MEMBER, ID: memberID},
       action:    CREATE,
       metadata:  Metadata{
           RequestID:   requestID,
           RecordCount: len(logs),
       },
   )
   ```
7. 返回 `ExportGDPRReportResult`

**事務邊界**: 只讀操作

**授權**: 僅 Admin 角色（處理 GDPR 請求）

**典型使用場景**:
- 會員行使 GDPR Right to Access（查詢個資處理記錄）
- 會員行使 Right to Data Portability（資料可攜權）
- 監管單位要求提供個資處理證明

**GDPR 合規要求**:
- ✅ 必須在 30 天內回應 GDPR 請求
- ✅ 提供機器可讀格式（JSON）
- ✅ 提供人類可讀格式（PDF）
- ✅ 完整記錄所有個資處理活動
- ✅ 記錄 GDPR 報告匯出操作（稽核追蹤）

**錯誤處理**:
- `ErrMemberNotFound` → HTTP 404
- `ErrUnauthorizedAuditAccess` → HTTP 403
- 報告生成失敗 → HTTP 500, 記錄錯誤

---
