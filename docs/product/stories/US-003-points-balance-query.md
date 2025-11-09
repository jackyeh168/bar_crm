# User Story 003: 積分餘額查詢 (Points Balance Query)

**Story ID**: US-003
**Priority**: P0 (Must Have)
**Sprint**: Phase 1 - MVP Core Features
**Status**: ✅ Completed
**Estimated Effort**: 3 Story Points

---

## 📖 User Story

> **身為** 一位會員 (小陳)，
> **我想要** 查詢我目前的累積積分，
> **以便** 我能了解可用的點數數量。

---

## ✅ Acceptance Criteria

### **成功場景 1：查詢積分**

**Given** 會員發送「積分」關鍵字
**When** 系統處理查詢
**Then** 顯示以下資訊：
- 累積賺取積分（所有已驗證交易的積分總和）
- 已使用積分（預留欄位，V3.1 暫為 0）
- 可用積分餘額（累積賺取 - 已使用）

---

## 📋 Business Rules

| Rule ID | Description |
|---------|-------------|
| BR-003-01 | 顯示積分：累積賺取積分（所有已驗證交易的積分總和） |
| BR-003-02 | 可用積分 = 累積賺取積分 - 已使用積分 |
| BR-003-03 | 查詢無需費用，可隨時查詢 |
| BR-003-04 | 只計算「已驗證」狀態的交易積分 |
| BR-003-05 | 積分計算包含基礎積分 + 問卷獎勵積分 |

---

## 🔧 Technical Implementation Notes

### **Points Calculation Logic**

```
累積賺取積分 (EarnedPoints) = Σ (已驗證交易的基礎積分 + 問卷獎勵積分)

其中：
- 基礎積分 = transaction.Amount / ConversionRule.Multiplier (floor division)
- 問卷獎勵積分 = transaction.SurveySubmitted ? 1 : 0

可用積分 (Points) = EarnedPoints - UsedPoints
```

### **Entity & Value Object**

```go
// User Entity (simplified)
type User struct {
    ID          UserID
    LineUserID  string
    DisplayName string
    Phone       Phone
    Points      int  // 可用積分（EarnedPoints - UsedPoints）
    EarnedPoints int // 累積賺取積分（自動計算）
    UsedPoints  int  // 已使用積分（V3.1 = 0，V3.2+ 積分兌換功能）
}

// PointsSummary Value Object
type PointsSummary struct {
    EarnedPoints int
    UsedPoints   int
    AvailablePoints int
}

func (ps PointsSummary) String() string {
    return fmt.Sprintf(
        "累積賺取: %d 點\n已使用: %d 點\n可用餘額: %d 點",
        ps.EarnedPoints, ps.UsedPoints, ps.AvailablePoints,
    )
}
```

### **Use Case Interface**

```go
// internal/service/points_service.go
type PointsService interface {
    GetUserPoints(lineUserID string) (*PointsSummary, error)
    RecalculateUserPoints(userID int) error
}

// Implementation
func (s *PointsServiceImpl) GetUserPoints(lineUserID string) (*PointsSummary, error) {
    user, err := s.UserRepo.FindByLineUserID(lineUserID)
    if err != nil {
        return nil, ErrUserNotFound
    }

    return &PointsSummary{
        EarnedPoints: user.EarnedPoints,
        UsedPoints: user.UsedPoints,
        AvailablePoints: user.Points, // Points = EarnedPoints - UsedPoints
    }, nil
}
```

### **Repository Interface**

```go
// internal/repository/user_repository.go
type UserRepository interface {
    FindByLineUserID(lineUserID string) (*User, error)
    UpdatePoints(userID int, earnedPoints int, usedPoints int) error
}

// internal/repository/transaction_repository.go
type TransactionRepository interface {
    FindVerifiedTransactionsByUserID(userID int) ([]*Transaction, error)
}
```

### **Database Schema**

```sql
-- Table: users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    line_user_id VARCHAR(100) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    phone VARCHAR(10) UNIQUE NOT NULL,
    points INTEGER DEFAULT 0,           -- 可用積分（EarnedPoints - UsedPoints）
    earned_points INTEGER DEFAULT 0,    -- 累積賺取積分（自動計算）
    used_points INTEGER DEFAULT 0,      -- 已使用積分（V3.1 = 0）
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_line_user_id ON users(line_user_id);
```

### **Points Recalculation Trigger**

**自動觸發場景**:
1. 交易狀態從 `imported` → `verified` (via iChef 匯入或手動更新)
2. 問卷提交（若交易已驗證）
3. 交易狀態從 `verified` → `failed`（扣除積分）

**手動觸發場景**:
- 管理員執行 `make recalculate-points` CLI 指令

### **Error Handling**

- `ErrUserNotFound`: 使用者不存在（未註冊）
- `ErrUserNotRegistered`: 使用者未完成註冊（無手機號碼綁定）

---

## 🧪 Test Cases

### **Unit Tests**

- ✅ `TestGetUserPoints_Success`: 成功查詢積分
- ✅ `TestGetUserPoints_UserNotFound`: 使用者不存在
- ✅ `TestGetUserPoints_NotRegistered`: 使用者未註冊
- ✅ `TestRecalculatePoints_SingleTransaction`: 計算單筆交易積分
- ✅ `TestRecalculatePoints_MultipleTransactions`: 計算多筆交易積分
- ✅ `TestRecalculatePoints_WithSurveyBonus`: 計算問卷獎勵積分
- ✅ `TestRecalculatePoints_OnlyVerifiedTransactions`: 只計算已驗證交易

### **Integration Tests**

- ✅ `TestPointsQuery_EndToEnd`: 完整查詢流程
- ✅ `TestPointsRecalculation_AfterStatusChange`: 狀態變更後自動重算
- ✅ `TestPointsRecalculation_AfterSurveySubmission`: 問卷提交後自動重算

---

## 📦 Dependencies

### **Internal Dependencies**

- **US-001**: 用戶必須先完成帳號綁定才能查詢積分
- **US-002**: QR Code 掃描創建交易記錄，影響積分計算
- **US-004**: 問卷系統（影響問卷獎勵積分）
- **US-005**: iChef 整合（驗證交易，觸發積分計算）

### **External Dependencies**

- LINE Bot SDK: 接收訊息和發送回應
- PostgreSQL: 儲存用戶和交易資料

### **Service Dependencies**

- `PointsService`: 積分查詢與計算
- `UserRepository`: 用戶資料存取
- `TransactionRepository`: 交易資料存取

---

## 📊 Success Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| 積分查詢活躍度 | > 80% | 每月查詢會員數 / 總會員數 |
| 查詢成功率 | > 99% | 成功查詢數 / 總查詢數 |
| 平均回應時間 | < 1 秒 | 查詢到返回結果的時間 |
| 積分計算準確率 | 100% | 正確計算數 / 總查詢數 |

---

## 🎯 User Personas

**Primary Persona**: 常客小陳（忠誠顧客）
- 25-35 歲上班族
- 每週至少光顧 2 次
- 經常查詢積分了解累積進度
- 關心可用餘額

**Secondary Persona**: 新客小美（潛在顧客）
- 20-30 歲學生或社會新鮮人
- 希望隨時了解積分狀態
- 期望即時反饋

---

## 📝 UI/UX Flow

### **Conversation Flow**

```
用戶: 積分
Bot:
  ┌─────────────────────────────────┐
  │ 💰 您的積分資訊                 │
  │                                  │
  │ 累積賺取: 125 點                │
  │ 已使用: 0 點                    │
  │ 可用餘額: 125 點                │
  │                                  │
  │ 📊 統計資訊:                    │
  │ • 已驗證交易: 48 筆             │
  │ • 待驗證交易: 2 筆              │
  │ • 已完成問卷: 35 筆             │
  └─────────────────────────────────┘
```

**Alternative Keywords**:
- "積分"
- "查詢積分"
- "點數"
- "餘額"

---

## 🚀 Performance Considerations

### **Optimization Strategies**

1. **資料庫查詢優化**
   - 積分資訊直接存儲在 `users.earned_points` 欄位（預計算）
   - 避免每次查詢時重新計算所有交易

2. **緩存策略**
   - Redis 緩存積分查詢結果（TTL: 5 分鐘）
   - 積分變更時清除緩存

3. **索引優化**
   - `users.line_user_id` 唯一索引
   - `transactions(user_id, status)` 複合索引

4. **並發控制**
   - 積分重算使用資料庫事務（READ COMMITTED）
   - 防止查詢與重算的競爭條件

---

## 🔗 Related Documents

- [PRD.md](../PRD.md) - 完整產品需求文件（§ 2.3）
- [DOMAIN_MODEL.md](../../architecture/DOMAIN_MODEL.md) - User Entity 和 PointsSummary 設計
- [DATABASE_DESIGN.md](../../architecture/DATABASE_DESIGN.md) - users 表結構設計
- [POINTS_SYSTEM_DESIGN.md](../../POINTS_SYSTEM_DESIGN.md) - 積分系統性能優化設計
- [US-002](./US-002-qr-code-scanning-points.md) - QR Code 掃描（創建交易）
- [US-004](./US-004-survey-system.md) - 問卷系統（影響積分計算）
- [US-005](./US-005-ichef-integration.md) - iChef 整合（驗證交易）

---

## 📋 Future Enhancements (V3.2+)

### **V3.2: 積分兌換功能**
- 顯示可兌換的商品清單
- 積分使用歷史記錄
- `UsedPoints` 欄位開始使用

### **V3.3: 積分明細查詢**
- 顯示每筆交易的詳細積分來源
- 積分增減歷史

### **V3.4: 積分預測**
- 根據消費習慣預測下月可獲得積分
- 個人化積分目標設定

---

**Story Created**: 2025-01-08
**Last Updated**: 2025-01-08
**Story Owner**: Product Team
**Technical Owner**: Backend Team
