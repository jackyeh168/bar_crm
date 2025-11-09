# User Story 004: 問卷調查系統 (Survey System)

**Story ID**: US-004
**Priority**: P1 (Should Have)
**Sprint**: Phase 2 - Advanced Features
**Status**: ✅ Completed
**Estimated Effort**: 13 Story Points

---

## 📖 User Story

> **身為** 一位會員 (小陳)，
> **我想要** 填寫餐廳滿意度問卷，
> **以便** 我能分享我的意見，並獲得額外 1 點積分獎勵。

---

## ✅ Acceptance Criteria

### **成功場景 1：完成問卷**

**Given** 會員掃描發票後收到問卷連結
**When** 點擊連結
**Then** 開啟問卷填寫頁面（無需登入，Token 認證）

**Given** 會員填寫所有必填題並提交
**When** 系統驗證完整性
**Then**
- 顯示「感謝您的回饋」訊息
- 若交易已驗證，立即增加 1 點積分
- 若交易未驗證，待驗證後自動增加 1 點積分

---

### **失敗場景 1：重複填寫**

**Given** 會員已填寫過此交易的問卷
**When** 再次嘗試填寫
**Then** 顯示「您已填寫過此問卷」

---

### **失敗場景 2：Token 過期或無效**

**Given** 會員使用過期或無效的 Token
**When** 嘗試訪問問卷
**Then** 顯示「問卷連結已失效」

---

## 📋 Business Rules

| Rule ID | Description |
|---------|-------------|
| BR-004-01 | 問卷題型：文字題、選擇題、評分題（1-5 星） |
| BR-004-02 | 獎勵規則：每筆交易問卷獎勵 1 點 |
| BR-004-03 | 唯一性：同一筆交易的問卷只能填寫一次 |
| BR-004-04 | 啟用控制：管理員可設定問卷是否啟用（同時只能有一個啟用問卷） |
| BR-004-05 | 無登入填寫：使用 Token 認證，無需 LINE 登入 |
| BR-004-06 | Token 有效期：生成後 30 天內有效 |
| BR-004-07 | 積分發放：問卷提交後，若交易已驗證則立即發放，否則待驗證後發放 |

---

## 🔧 Technical Implementation Notes

### **Entity & Value Object**

```go
// Survey Entity
type Survey struct {
    ID          SurveyID
    Title       string
    Description string
    Questions   []SurveyQuestion
    IsActive    bool
    CreatedAt   time.Time
}

// SurveyQuestion Entity
type SurveyQuestion struct {
    ID              QuestionID
    SurveyID        SurveyID
    QuestionText    string
    QuestionType    QuestionType // text, multiple_choice, rating
    Options         []string     // For multiple_choice
    IsRequired      bool
    OrderIndex      int
}

// QuestionType Value Object
type QuestionType string

const (
    QuestionTypeText           QuestionType = "text"
    QuestionTypeMultipleChoice QuestionType = "multiple_choice"
    QuestionTypeRating         QuestionType = "rating"  // 1-5 stars
)

// SurveyResponse Entity
type SurveyResponse struct {
    ID            ResponseID
    Survey        *Survey
    Transaction   *Transaction
    Answers       []Answer
    SubmittedAt   time.Time
}

// Answer Value Object
type Answer struct {
    QuestionID QuestionID
    AnswerText string     // For text/multiple_choice
    Rating     int        // For rating (1-5)
}

// SurveyToken Value Object
type SurveyToken struct {
    Token         string
    SurveyID      SurveyID
    TransactionID TransactionID
    ExpiresAt     time.Time
}

func (t SurveyToken) IsValid() bool {
    return time.Now().Before(t.ExpiresAt)
}
```

### **Use Case Interface**

```go
// internal/service/survey_service.go
type SurveyService interface {
    // Admin operations
    CreateSurvey(survey *Survey) error
    ActivateSurvey(surveyID int) error
    DeactivateSurvey(surveyID int) error
    GetActiveSurvey() (*Survey, error)

    // User operations
    GenerateSurveyToken(surveyID int, transactionID int) (string, error)
    ValidateSurveyToken(token string) (*SurveyContext, error)
    SubmitResponse(response *SurveyResponse) error
    GetSurveyResponses(surveyID int) ([]*SurveyResponse, error)
}

// SurveyContext contains all info needed for filling survey
type SurveyContext struct {
    Survey      *Survey
    Transaction *Transaction
    User        *User
}
```

### **Repository Interface**

```go
// internal/repository/survey_repository.go
type SurveyRepository interface {
    Create(survey *Survey) error
    FindByID(id int) (*Survey, error)
    FindActiveSurvey() (*Survey, error)
    UpdateActiveStatus(surveyID int, isActive bool) error
}

// internal/repository/survey_response_repository.go
type SurveyResponseRepository interface {
    Create(response *SurveyResponse) error
    FindBySurveyID(surveyID int) ([]*SurveyResponse, error)
    FindByTransactionID(txID int) (*SurveyResponse, error)
    CheckIfSubmitted(surveyID int, txID int) (bool, error)
}
```

### **Database Schema**

```sql
-- Table: surveys
CREATE TABLE surveys (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 確保同時只有一個啟用的問卷
CREATE UNIQUE INDEX idx_surveys_active ON surveys(is_active) WHERE is_active = TRUE;

-- Table: survey_questions
CREATE TABLE survey_questions (
    id SERIAL PRIMARY KEY,
    survey_id INTEGER REFERENCES surveys(id) ON DELETE CASCADE,
    question_text TEXT NOT NULL,
    question_type VARCHAR(20) NOT NULL,
    options JSONB,
    is_required BOOLEAN DEFAULT TRUE,
    order_index INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_survey_questions_survey_id ON survey_questions(survey_id);

-- Table: survey_responses
CREATE TABLE survey_responses (
    id SERIAL PRIMARY KEY,
    survey_id INTEGER REFERENCES surveys(id),
    transaction_id INTEGER REFERENCES transactions(id),
    submitted_at TIMESTAMP DEFAULT NOW()
);

-- 確保同一筆交易只能填寫一次問卷
CREATE UNIQUE INDEX idx_survey_responses_unique ON survey_responses(survey_id, transaction_id);

CREATE INDEX idx_survey_responses_survey_id ON survey_responses(survey_id);
CREATE INDEX idx_survey_responses_transaction_id ON survey_responses(transaction_id);

-- Table: survey_answers
CREATE TABLE survey_answers (
    id SERIAL PRIMARY KEY,
    response_id INTEGER REFERENCES survey_responses(id) ON DELETE CASCADE,
    question_id INTEGER REFERENCES survey_questions(id),
    answer_text TEXT,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_survey_answers_response_id ON survey_answers(response_id);
```

### **Survey Token Generation**

```go
// Generate JWT token for survey access
func GenerateSurveyToken(surveyID int, transactionID int) (string, error) {
    claims := jwt.MapClaims{
        "survey_id": surveyID,
        "transaction_id": transactionID,
        "exp": time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// Survey URL format
// {FRONTEND_URL}/public-survey?token={survey_token}
```

### **Points Reward Flow**

```
1. User submits survey
   ↓
2. Set transaction.SurveySubmitted = true
   ↓
3. Check transaction status:
   - If verified → Immediately recalculate user points (+1 bonus)
   - If imported → Wait for verification, bonus applied when verified
```

### **Error Handling**

- `ErrSurveyNotFound`: 問卷不存在
- `ErrSurveyNotActive`: 問卷未啟用
- `ErrSurveyAlreadySubmitted`: 已填寫過此問卷
- `ErrInvalidSurveyToken`: Token 無效或過期
- `ErrMissingRequiredAnswer`: 必填題未填寫
- `ErrInvalidRating`: 評分不在 1-5 範圍

---

## 🧪 Test Cases

### **Unit Tests**

- ✅ `TestCreateSurvey_Success`: 成功建立問卷
- ✅ `TestActivateSurvey_OnlyOneActive`: 啟用問卷時自動停用其他問卷
- ✅ `TestGenerateSurveyToken_ValidFormat`: 生成有效的 Token
- ✅ `TestValidateSurveyToken_Expired`: 驗證過期 Token 失敗
- ✅ `TestSubmitResponse_Success`: 成功提交問卷
- ✅ `TestSubmitResponse_DuplicateSubmission`: 重複提交失敗
- ✅ `TestSubmitResponse_MissingRequiredAnswer`: 必填題未填寫失敗
- ✅ `TestSubmitResponse_TriggersPointsRecalculation`: 提交後觸發積分重算

### **Integration Tests**

- ✅ `TestSurveyFlow_EndToEnd`: 完整問卷流程（生成 Token → 填寫 → 提交 → 獲得積分）
- ✅ `TestSurveyFlow_PointsReward`: 驗證問卷獎勵積分正確發放

---

## 📦 Dependencies

### **Internal Dependencies**

- **US-002**: QR Code 掃描創建交易，生成問卷連結
- **US-003**: 問卷提交後影響積分查詢結果

### **External Dependencies**

- LINE Bot SDK: 發送問卷連結訊息
- JWT Library: 生成和驗證 Token（如 `github.com/golang-jwt/jwt`）
- PostgreSQL: 儲存問卷和回應資料

### **Service Dependencies**

- `SurveyService`: 問卷管理與回應處理
- `TransactionService`: 更新 `SurveySubmitted` 狀態，觸發積分重算
- `PointsService`: 積分重新計算

---

## 📊 Success Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| 問卷完成率 | > 30% | 完成問卷數 / 收到問卷連結數 |
| 問卷連結點擊率 | > 50% | 點擊連結數 / 收到連結數 |
| 平均填寫時間 | < 3 分鐘 | 開啟到提交的平均時間 |
| 問卷提交成功率 | > 95% | 成功提交數 / 嘗試提交數 |
| 積分獎勵發放準確率 | 100% | 正確發放數 / 總提交數 |

---

## 🎯 User Personas

**Primary Persona**: 常客小陳（忠誠顧客）
- 25-35 歲上班族
- 每週至少光顧 2 次
- 願意填寫問卷提供意見
- 期望獲得額外積分獎勵

**Secondary Persona**: 店長王姐（營運管理者）
- 35-45 歲餐廳店長
- 希望收集顧客意見
- 期望查看問卷統計數據
- 需要簡單的問卷建立介面

---

## 📝 UI/UX Flow

### **Member Side: Conversation Flow**

```
用戶: [上傳發票 QR Code]
Bot:
  ┌─────────────────────────────────┐
  │ ✅ 發票資訊確認                 │
  │                                  │
  │ 發票號碼: AB12345678            │
  │ 消費金額: $250                  │
  │ 預估積分: 2 點                  │
  │                                  │
  │ 📋 填寫問卷可再得 1 點：        │
  │ https://example.com/survey?...  │
  └─────────────────────────────────┘

用戶: [點擊問卷連結]
瀏覽器:
  ┌─────────────────────────────────┐
  │ 📋 餐廳滿意度問卷               │
  │                                  │
  │ 1. 您對今日的用餐體驗滿意嗎？  │
  │    ★★★★★                        │
  │                                  │
  │ 2. 您最喜歡哪道菜？             │
  │    [ 文字輸入框 ]               │
  │                                  │
  │ 3. 您會推薦給朋友嗎？           │
  │    ○ 會                         │
  │    ○ 不會                       │
  │    ○ 不確定                     │
  │                                  │
  │ [提交]                          │
  └─────────────────────────────────┘

用戶: [提交問卷]
瀏覽器:
  ┌─────────────────────────────────┐
  │ ✅ 感謝您的回饋！               │
  │                                  │
  │ 您已獲得 1 點問卷獎勵積分       │
  │ （若發票已驗證）                │
  │                                  │
  │ 歡迎再次光臨！                  │
  └─────────────────────────────────┘
```

### **Admin Side: Survey Management**

**參考**: `frontend/src/pages/surveys/`
- `list.tsx` - 問卷列表
- `create.tsx` - 建立問卷
- `edit.tsx` - 編輯問卷
- `show.tsx` - 查看問卷詳情和統計

---

## 🚀 Performance Considerations

### **Optimization Strategies**

1. **資料庫查詢優化**
   - `survey_responses(survey_id, transaction_id)` 唯一索引防止重複提交
   - `survey_questions.survey_id` 索引加速問題查詢

2. **Token 驗證優化**
   - JWT 無狀態驗證，無需資料庫查詢
   - Token 包含過期時間，自動處理失效

3. **問卷回應儲存**
   - 使用資料庫事務確保原子性：
     1. 創建 survey_response
     2. 批次插入 survey_answers
     3. 更新 transaction.SurveySubmitted
     4. 觸發積分重算

4. **統計查詢優化**
   - 問卷統計使用 GROUP BY 聚合
   - 考慮使用 Materialized View 快取統計結果

---

## 🔗 Related Documents

- [PRD.md](../PRD.md) - 完整產品需求文件（§ 2.4）
- [DOMAIN_MODEL.md](../../architecture/DOMAIN_MODEL.md) - Survey Entity 和 Answer Value Object 設計
- [DATABASE_DESIGN.md](../../architecture/DATABASE_DESIGN.md) - surveys 相關表結構設計
- [FRONTEND_INTEGRATION.md](../../FRONTEND_INTEGRATION.md) - 前端問卷頁面整合
- [US-002](./US-002-qr-code-scanning-points.md) - QR Code 掃描（生成問卷連結）
- [US-003](./US-003-points-balance-query.md) - 積分查詢（顯示問卷獎勵）

---

## 📋 Future Enhancements (V3.5+)

### **V3.5: 進階問卷功能**
- 條件式問題（依前一題答案顯示）
- 多頁問卷支援
- 圖片上傳題型

### **V3.6: 問卷分析**
- 情感分析（文字回應）
- 趨勢分析（評分變化）
- 匯出報表（Excel/PDF）

### **V3.7: 個人化問卷**
- 根據消費歷史推送不同問卷
- A/B 測試支援

---

**Story Created**: 2025-01-08
**Last Updated**: 2025-01-08
**Story Owner**: Product Team
**Technical Owner**: Backend Team & Frontend Team
