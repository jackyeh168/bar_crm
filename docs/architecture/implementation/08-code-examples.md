# 完整代碼範例

> **版本**: 1.0
> **最後更新**: 2025-01-10

## 1. 完整流程：掃描發票獲得積分

本範例展示從 HTTP 請求到數據庫的完整流程，涵蓋所有層級。

### 1.1 HTTP 請求

```bash
POST /api/v1/invoices/scan
Content-Type: application/json

{
  "member_id": "M123",
  "qr_code_data": "AB12345678|1130115|1000|...",
  "image_url": "https://example.com/invoice.jpg"
}
```

### 1.2 Presentation Layer

**文件**: `internal/presentation/http/handlers/invoice_handler.go`

```go
func (h *InvoiceHandler) HandleScanInvoice(c *gin.Context) {
    var req ScanInvoiceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        responses.Error(c, http.StatusBadRequest, "Invalid request", err)
        return
    }

    // 構造 Command
    cmd := usecases.ScanInvoiceCommand{
        MemberID:    req.MemberID,
        QRCodeData:  req.QRCodeData,
        ImageURL:    req.ImageURL,
    }

    // 執行 Use Case
    result, err := h.scanInvoiceUseCase.Execute(cmd)
    if err != nil {
        responses.Error(c, http.StatusInternalServerError, "Scan failed", err)
        return
    }

    responses.Success(c, result)
}
```

### 1.3 Application Layer

**文件**: `internal/application/usecases/invoice/scan_invoice.go`

```go
func (uc *ScanInvoiceUseCase) Execute(cmd ScanInvoiceCommand) (*ScanInvoiceResult, error) {
    var result *ScanInvoiceResult

    // 使用事務
    err := uc.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        // 1. 解析 QR Code (Domain Service)
        invoice, err := uc.parsingService.ParseQRCode(cmd.QRCodeData)
        if err != nil {
            return err
        }

        // 2. 驗證發票 (Domain Service)
        err = uc.validationService.ValidateInvoice(invoice)
        if err != nil {
            return err
        }

        // 3. 創建交易 (Domain Entity)
        memberID, _ := invoice.NewMemberID(cmd.MemberID)
        transaction, err := invoice.NewInvoiceTransaction(
            memberID,
            invoice,
            invoice.NewQRCodeData(cmd.QRCodeData),
        )
        if err != nil {
            return err
        }

        // 4. 保存交易
        err = uc.transactionRepo.Create(ctx, transaction)
        if err != nil {
            return err
        }

        // 5. 發布領域事件（在事務提交後發布）
        // 事件: TransactionCreated
        
        result = &ScanInvoiceResult{
            TransactionID: transaction.GetTransactionID().String(),
            Status:        "imported",
            Message:       "Invoice scanned successfully",
        }

        return nil
    })

    return result, err
}
```

### 1.4 Domain Layer

**聚合根**: `internal/domain/invoice/transaction.go`

```go
func NewInvoiceTransaction(
    memberID MemberID,
    invoice Invoice,
    qrCodeData QRCodeData,
) (*InvoiceTransaction, error) {
    // 驗證
    if memberID.IsEmpty() {
        return nil, ErrInvalidMemberID
    }

    // 創建聚合
    tx := &InvoiceTransaction{
        transactionID: NewTransactionID(),
        memberID:      memberID,
        invoice:       invoice,
        qrCodeData:    qrCodeData,
        status:        StatusImported,  // 初始狀態
        createdAt:     time.Now(),
        events:        []DomainEvent{},
    }

    // 發布事件
    tx.publishEvent(TransactionCreated{
        TransactionID: tx.transactionID,
        MemberID:      memberID,
        InvoiceNumber: invoice.InvoiceNumber(),
        Amount:        invoice.Amount(),
        OccurredAt:    time.Now(),
    })

    return tx, nil
}
```

**領域服務**: `internal/domain/invoice/validation_service.go`

```go
func (s *InvoiceValidationService) ValidateInvoice(invoice Invoice) error {
    // 1. 檢查格式
    if err := s.ValidateInvoiceNumber(invoice.InvoiceNumber()); err != nil {
        return err
    }

    // 2. 檢查有效期（60 天）
    if s.CheckExpiry(invoice.InvoiceDate()) {
        return ErrInvoiceExpired
    }

    // 3. 檢查重複性
    isDuplicate, err := s.CheckDuplicate(invoice.InvoiceNumber())
    if err != nil {
        return err
    }
    if isDuplicate {
        return ErrInvoiceDuplicate
    }

    return nil
}
```

### 1.5 Infrastructure Layer

**Repository 實現**: `internal/infrastructure/persistence/invoice/transaction_repository.go`

```go
func (r *GormInvoiceTransactionRepository) Create(
    ctx shared.TransactionContext,
    transaction *invoice.InvoiceTransaction,
) error {
    db := r.extractDB(ctx)
    model := r.toModel(transaction)

    // 檢查唯一性約束
    var count int64
    db.Model(&InvoiceTransactionModel{}).
        Where("invoice_number = ?", model.InvoiceNumber).
        Count(&count)
    
    if count > 0 {
        return repository.ErrInvoiceDuplicate
    }

    // 插入記錄
    return db.Create(model).Error
}

func (r *GormInvoiceTransactionRepository) toModel(
    tx *invoice.InvoiceTransaction,
) *InvoiceTransactionModel {
    return &InvoiceTransactionModel{
        TransactionID: tx.GetTransactionID().String(),
        MemberID:      tx.GetMemberID().String(),
        InvoiceNumber: tx.GetInvoice().InvoiceNumber().String(),
        InvoiceDate:   tx.GetInvoice().InvoiceDate(),
        Amount:        tx.GetInvoice().Amount().Value(),
        Status:        tx.GetStatus().String(),
        QRCodeData:    tx.GetQRCodeData().String(),
        CreatedAt:     tx.GetCreatedAt(),
    }
}
```

### 1.6 事件處理流程

**事件發布** (使用 Application Service):

```go
// ✅ 新方式：使用 Application Service 自動處理事件發布
func (uc *ScanInvoiceUseCase) Execute(cmd ScanInvoiceCommand) (*ScanInvoiceResult, error) {
    var result *ScanInvoiceResult

    // 使用 Application Service 執行業務邏輯
    err := uc.appService.ExecuteInTransaction(func(ctx shared.TransactionContext) ([]service.AggregateRoot, error) {
        // 1. 解析 QR Code
        invoice, err := uc.parsingService.ParseQRCode(cmd.QRCodeData)
        if err != nil {
            return nil, err
        }

        // 2. 驗證發票
        err = uc.validationService.ValidateInvoice(invoice)
        if err != nil {
            return nil, err
        }

        // 3. 創建交易
        memberID, _ := invoice.NewMemberID(cmd.MemberID)
        transaction, err := invoice.NewInvoiceTransaction(
            memberID,
            invoice,
            invoice.NewQRCodeData(cmd.QRCodeData),
        )
        if err != nil {
            return nil, err
        }

        // 4. 保存交易
        err = uc.transactionRepo.Create(ctx, transaction)
        if err != nil {
            return nil, err
        }

        result = &ScanInvoiceResult{
            TransactionID: transaction.GetTransactionID().String(),
            Status:        "imported",
            Message:       "Invoice scanned successfully",
        }

        // 5. 返回修改過的聚合（Application Service 會自動發布事件）
        return []service.AggregateRoot{transaction}, nil
    })

    return result, err
}
```

**Application Service 設計**:

```go
package service

// ApplicationService 統一處理事務和事件發布
type ApplicationService struct {
    txManager shared.TransactionManager
    eventBus  shared.EventPublisher
}

func (s *ApplicationService) ExecuteInTransaction(
    fn func(ctx shared.TransactionContext) ([]AggregateRoot, error),
) error {
    var aggregates []AggregateRoot

    // 在事務中執行業務邏輯
    err := s.txManager.InTransaction(func(ctx shared.TransactionContext) error {
        aggs, err := fn(ctx)
        if err != nil {
            return err
        }
        aggregates = aggs
        return nil
    })

    if err != nil {
        return err  // 事務失敗，不發布事件
    }

    // 事務成功後，自動收集並發布所有事件
    events := s.collectEvents(aggregates)
    if len(events) > 0 {
        s.eventBus.PublishBatch(events)
    }

    // 清空聚合事件
    s.clearEvents(aggregates)

    return nil
}
```

**事件處理器** (Application Layer):

```go
// internal/application/events/audit/transaction_event_handler.go
func (h *TransactionEventHandler) Handle(event invoice.TransactionCreated) error {
    // 記錄稽核日誌
    auditLog := audit.NewAuditLog(
        audit.EventTypeTransactionCreated,
        audit.Actor{ActorType: audit.ActorTypeMember, ActorID: event.MemberID().String()},
        audit.Target{TargetType: audit.TargetTypeTransaction, TargetID: event.TransactionID().String()},
        audit.ActionCreate,
        audit.Changes{After: map[string]interface{}{
            "invoice_number": event.InvoiceNumber().String(),
            "amount": event.Amount().Value(),
        }},
        audit.Metadata{},
    )

    return h.auditWriter.Create(auditLog)
}
```

## 2. 測試範例

### 2.1 Domain Layer 單元測試

```go
func TestPointsAccount_EarnPoints(t *testing.T) {
    // Arrange
    memberID, _ := points.NewMemberID("M123")
    account, _ := points.NewPointsAccount(memberID)

    amount := points.NewPointsAmount(10)

    // Act
    err := account.EarnPoints(
        amount,
        points.PointsSourceInvoice,
        "TX123",
        "Test transaction",
    )

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 10, account.GetEarnedPoints().Value())
    assert.Equal(t, 10, account.GetAvailablePoints().Value())
    
    // 檢查事件
    events := account.GetEvents()
    assert.Len(t, events, 1)
    assert.Equal(t, "points.earned", events[0].EventType())
}
```

### 2.2 Application Layer 集成測試

```go
func TestEarnPointsUseCase_Execute(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewMockPointsAccountRepository()
    mockCalculator := mocks.NewMockPointsCalculationService()
    mockTxManager := mocks.NewMockTransactionManager()

    useCase := usecases.NewEarnPointsUseCase(mockRepo, mockCalculator, mockTxManager)

    cmd := usecases.EarnPointsCommand{
        MemberID:    "M123",
        Amount:      decimal.NewFromInt(1000),
        InvoiceDate: time.Now(),
        Source:      "invoice",
        SourceID:    "TX123",
    }

    // Act
    result, err := useCase.Execute(cmd)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, 10, result.EarnedPoints)
}
```

### 2.3 端到端測試

```go
func TestE2E_ScanAndEarnPoints(t *testing.T) {
    // 1. 掃描發票
    resp := httptest.NewRequest("POST", "/api/v1/invoices/scan", bytes.NewBuffer([]byte(`{
        "member_id": "M123",
        "qr_code_data": "AB12345678|1130115|1000|..."
    }`)))

    // 2. 驗證發票創建
    var scanResult ScanInvoiceResponse
    json.Unmarshal(resp.Body.Bytes(), &scanResult)
    assert.Equal(t, "imported", scanResult.Status)

    // 3. iChef 匹配（模擬批次匯入）
    importResp := httptest.NewRequest("POST", "/api/v1/admin/import/ichef", ...)

    // 4. 驗證積分獲得
    pointsResp := httptest.NewRequest("GET", "/api/v1/points/balance/M123", nil)
    var pointsResult PointsBalanceResponse
    json.Unmarshal(pointsResp.Body.Bytes(), &pointsResult)
    assert.Equal(t, 10, pointsResult.AvailablePoints)
}
```

---

## 3. 常見錯誤與解決方案

### 錯誤 1: 循環依賴

**問題**: Domain 依賴 Application 的 DTO

**解決方案**: Domain 定義接口，DTO 實現接口

### 錯誤 2: Repository 洩漏 GORM 錯誤

**問題**: Repository 返回 `gorm.ErrRecordNotFound`

**解決方案**: 轉換為 Domain 錯誤 `repository.ErrAccountNotFound`

### 錯誤 3: 貧血領域模型

**問題**: 聚合只有 getter/setter，業務邏輯在 Service

**解決方案**: 業務邏輯封裝在聚合方法內，使用 Tell Don't Ask 原則

---

**完成**: 實現指南文檔集已完成！開始參考 [README.md](./README.md) 開始實現。
