# 戰術設計 (Tactical Design)

> **版本**: 1.0
> **最後更新**: 2025-01-08

---

## **5.1 核心域深度設計 - 積分管理** ⭐

### **5.1.1 積分計算完整流程**

```
┌─────────────────────────────────────────────────────────────┐
│            積分計算流程 (Points Calculation Flow)              │
└─────────────────────────────────────────────────────────────┘

1. 交易驗證觸發
   TransactionVerified Event
   ├── TransactionID
   ├── MemberID
   ├── Amount (消費金額)
   ├── InvoiceDate (發票日期)
   └── SurveySubmitted (是否完成問卷)

2. 查詢轉換規則
   ConversionRuleQueryService.GetRuleForDate(InvoiceDate)
   └── 返回: ConversionRule (轉換率)

3. 計算基礎積分
   BasePoints = floor(Amount / ConversionRate)
   例: 250 / 100 = 2 點

4. 計算問卷獎勵
   SurveyBonus = SurveySubmitted ? 1 : 0

5. 計算總積分
   TotalPoints = BasePoints + SurveyBonus
   例: 2 + 1 = 3 點

6. 更新積分帳戶（輕量級聚合）
   PointsAccount.EarnPoints(TotalPoints, SourceInvoice, TransactionID)
   ├── EarnedPoints += TotalPoints（狀態更新）
   └── 發布 PointsEarned Event（領域事件）

7. Application Layer 處理事件
   監聽 PointsEarned Event:
   ├── 創建 PointsTransaction 記錄（審計日誌）
   │   └── PointsTransactionRepository.Create(transaction)
   └── 觸發通知: "您獲得了 3 點積分"
```

### **5.1.2 積分重算機制**

```
┌─────────────────────────────────────────────────────────────┐
│          積分重算流程 (Points Recalculation Flow)             │
└─────────────────────────────────────────────────────────────┘

觸發條件:
1. 管理員創建/更新/刪除轉換規則
2. 管理員手動執行重算指令

重算步驟:
1. 鎖定所有會員的積分帳戶 (Database Transaction)
2. 對每個會員:
   a. 查詢所有已驗證交易 (Status = Verified)
   b. 對每筆交易:
      - 根據 InvoiceDate 查詢適用的 ConversionRule
      - 重新計算 BasePoints
      - 檢查 SurveySubmitted, 計算 SurveyBonus
   c. 加總所有交易的積分
   d. 更新 EarnedPoints
3. 提交事務
4. 發布 PointsRecalculated 事件

注意事項:
- 重算期間鎖表 30-60 秒
- 必須在離峰時段執行
- 使用事務保證原子性
- 需要管理員確認才執行
```

### **5.1.3 積分兌換擴展設計 (V3.2+)**

```
┌─────────────────────────────────────────────────────────────┐
│            未來: 積分兌換設計 (Points Redemption)              │
└─────────────────────────────────────────────────────────────┘

新增聚合: RedemptionCatalog (兌換目錄)
├── CatalogID
├── Items (Entity Collection)
│   └── RedemptionItem (Entity)
│       ├── ItemID
│       ├── Name
│       ├── PointsCost (兌換所需積分)
│       ├── StockQuantity (庫存)
│       ├── IsActive
│       └── ValidUntil
└── UpdatedAt

新增聚合: Redemption (兌換記錄)
├── RedemptionID
├── MemberID
├── ItemID
├── PointsDeducted (扣除積分)
├── Status (Pending/Confirmed/Cancelled)
├── RedeemedAt
└── CompletedAt

新增業務規則:
1. 可用積分 >= PointsCost 才能兌換
2. 兌換後立即扣除積分 (AvailablePoints -= PointsCost)
3. 兌換記錄可在 24 小時內取消
4. 取消後積分退回

新增領域事件:
- RedemptionRequested (兌換請求已提交)
- RedemptionConfirmed (兌換已確認)
- RedemptionCancelled (兌換已取消)
- PointsRefunded (積分已退回)
```
