# User Story 006: 管理後台系統 (Admin Portal System)

**Story ID**: US-006
**Priority**: P1 (Should Have)
**Sprint**: Phase 2 - Advanced Features
**Status**: ✅ Completed
**Estimated Effort**: 34 Story Points

---

## 📖 User Story

> **身為** 店長 (王姐) 或管理員，
> **我想要** 透過網頁管理後台查看和管理會員資料、交易記錄、問卷回應，
> **以便** 我能進行營運分析與決策。

---

## 🎯 Sub-Stories Overview

此 User Story 包含以下子功能：

1. **US-006.1**: 認證與授權 (Authentication & Authorization)
2. **US-006.2**: 會員管理 (Member Management)
3. **US-006.3**: 交易管理 (Transaction Management)
4. **US-006.4**: 問卷管理 (Survey Management)
5. **US-006.5**: 積分轉換規則管理 (Points Conversion Rules Management)

---

## 📋 Sub-Story US-006.1: 認證與授權

### **User Story**

> **身為** 管理員，
> **我想要** 使用 Google 帳號安全登入管理後台並獲得適當的權限，
> **以便** 我能訪問授權範圍內的功能。

### **Acceptance Criteria**

**Given** 管理員使用 Google 帳號登入
**When** 驗證成功
**Then** 系統分配角色（Admin/User/Guest）並產生訪問權限

**Given** 首位登入的使用者 email 符合 DEFAULT_ADMIN_EMAIL
**When** 首次登入
**Then** 自動授予 Admin 角色

**Given** 非管理員使用者
**When** 嘗試訪問受限功能
**Then** 顯示「權限不足」錯誤

### **Business Rules**

| Rule ID | Description |
|---------|-------------|
| BR-006.1-01 | 使用 Google OAuth2 認證，無需自訂密碼 |
| BR-006.1-02 | 首次登入的 DEFAULT_ADMIN_EMAIL 自動授予 Admin 角色 |
| BR-006.1-03 | JWT Token 有效期：24 小時 |
| BR-006.1-04 | Refresh Token 有效期：7 天 |

### **Role-Based Access Control (RBAC)**

| Role | Permissions |
|------|-------------|
| **Admin** | 完整權限（CRUD 所有資源、設定積分規則、管理用戶角色） |
| **User** | 唯讀權限（查看會員、交易、問卷資料） |
| **Guest** | 受限唯讀（僅查看公開統計數據） |

### **Technical Notes**

```go
// Entity
type AdminUser struct {
    ID        AdminUserID
    Email     Email  // Value Object with validation
    Name      string
    Role      AdminRole
    IsActive  bool
    CreatedAt time.Time
}

// AdminRole Value Object
type AdminRole string

const (
    AdminRoleAdmin AdminRole = "admin"
    AdminRoleUser  AdminRole = "user"
    AdminRoleGuest AdminRole = "guest"
)

func (r AdminRole) CanCreate() bool {
    return r == AdminRoleAdmin
}

func (r AdminRole) CanUpdate() bool {
    return r == AdminRoleAdmin
}

func (r AdminRole) CanDelete() bool {
    return r == AdminRoleAdmin
}

func (r AdminRole) CanRead() bool {
    return r == AdminRoleAdmin || r == AdminRoleUser
}
```

---

## 📋 Sub-Story US-006.2: 會員管理

### **User Story**

> **身為** 管理員，
> **我想要** 查看、搜尋、管理會員資料，
> **以便** 我能了解會員狀態和消費行為。

### **Acceptance Criteria**

**Given** 管理員訪問會員列表
**When** 查詢
**Then** 顯示分頁列表（LINE ID、暱稱、手機號碼、累積積分）

**Given** 管理員搜尋特定會員
**When** 輸入手機號碼或暱稱
**Then** 過濾並顯示匹配結果

**Given** 管理員查看會員詳情
**When** 點擊會員
**Then** 顯示交易歷史和問卷回應記錄

### **Business Rules**

| Rule ID | Description |
|---------|-------------|
| BR-006.2-01 | 支援分頁查詢（預設 10 筆/頁） |
| BR-006.2-02 | 支援按暱稱、手機號碼搜尋（模糊比對） |
| BR-006.2-03 | 支援按累積積分排序（升序/降序） |
| BR-006.2-04 | 會員詳情顯示交易歷史和問卷回應 |

### **API Endpoints**

```
GET /api/admin/users
  Query Params:
    - page: int (default: 1)
    - page_size: int (default: 10)
    - sort_field: string (display_name, phone, earned_points)
    - sort_order: string (ascend, descend)
    - display_name: string (contains filter)
    - phone: string (contains filter)

GET /api/admin/users/:id
  Response:
    - User details
    - Transaction history
    - Survey responses
```

---

## 📋 Sub-Story US-006.3: 交易管理

### **User Story**

> **身為** 管理員，
> **我想要** 查看、搜尋、管理交易記錄，
> **以便** 我能驗證發票並處理異常狀況。

### **Acceptance Criteria**

**Given** 管理員查看交易列表
**When** 查詢
**Then** 顯示所有交易（發票號碼、日期、金額、狀態、積分）

**Given** 管理員篩選特定狀態
**When** 選擇「待驗證」
**Then** 僅顯示待驗證交易

**Given** 管理員手動更新交易狀態
**When** 將「待驗證」改為「已驗證」
**Then** 系統自動重算會員積分

### **Business Rules**

| Rule ID | Description |
|---------|-------------|
| BR-006.3-01 | 狀態流轉限制：imported → verified 或 failed；verified → failed |
| BR-006.3-02 | failed 狀態無法恢復 |
| BR-006.3-03 | 狀態變更後立即觸發積分重算 |
| BR-006.3-04 | 支援按發票號碼、日期範圍、狀態篩選 |

### **Transaction Status Flow**

```
imported (待驗證)
    ↓
    ├─→ verified (已驗證) ← 可觸發積分增加
    │       ↓
    │   failed (作廢) ← 可觸發積分扣除
    │
    └─→ failed (作廢) ← 直接作廢
```

### **API Endpoints**

```
GET /api/admin/transactions
  Query Params:
    - page: int
    - page_size: int
    - status: string (imported, verified, failed)
    - invoice_no: string (contains filter)
    - start_date: date
    - end_date: date
    - user_id: int

PUT /api/admin/transactions/:id
  Body:
    - status: string (verified, failed)
  Triggers: Points recalculation
```

---

## 📋 Sub-Story US-006.4: 問卷管理

### **User Story**

> **身為** 管理員，
> **我想要** 建立、編輯、啟用問卷並查看回應統計，
> **以便** 我能收集顧客意見並分析滿意度。

### **Acceptance Criteria**

**Given** 管理員創建新問卷
**When** 輸入標題、題目和選項
**Then** 系統保存並分配唯一 ID

**Given** 管理員啟用問卷
**When** 設定為「啟用」
**Then** 會員掃描發票後自動收到問卷連結（自動停用其他問卷）

**Given** 管理員查看問卷回應
**When** 查詢特定問卷
**Then** 顯示所有回應和統計摘要

### **Business Rules**

| Rule ID | Description |
|---------|-------------|
| BR-006.4-01 | 單一啟用：同時只能有一個問卷處於啟用狀態 |
| BR-006.4-02 | 題型支援：text, multiple_choice, rating (1-5) |
| BR-006.4-03 | 必填控制：可設定個別題目是否必填 |
| BR-006.4-04 | 問卷刪除：僅可刪除未啟用且無回應的問卷 |

### **Survey Statistics**

```go
type SurveyStatistics struct {
    TotalResponses   int
    CompletionRate   float64  // Responses / Links sent
    AverageRating    float64  // For rating questions
    AnswerDistribution map[string]int  // For multiple_choice
    CommonKeywords   []string // For text questions (optional)
}
```

### **API Endpoints**

```
POST /api/admin/surveys
  Body: Survey object with questions

GET /api/admin/surveys
  Response: List of surveys

PUT /api/admin/surveys/:id/activate
  Action: Activate survey (deactivate others)

GET /api/admin/surveys/:id/responses
  Response: List of responses with statistics

GET /api/admin/surveys/:id/statistics
  Response: SurveyStatistics object
```

---

## 📋 Sub-Story US-006.5: 積分轉換規則管理

### **User Story**

> **身為** 管理員，
> **我想要** 設定不同時期的積分轉換率，
> **以便** 我能執行促銷活動並靈活調整積分獎勵。

### **Acceptance Criteria**

**Given** 管理員設定促銷期轉換率
**When** 輸入日期範圍和轉換率（如 50 元 = 1 點）
**Then** 系統保存規則

**Given** 管理員變更規則後
**When** 保存
**Then** 系統顯示警告「需執行 make recalculate-points 重新計算所有積分」

**Given** 管理員查詢某日期的適用規則
**When** 輸入日期
**Then** 顯示該日期適用的轉換率

### **Business Rules**

| Rule ID | Description |
|---------|-------------|
| BR-006.5-01 | 日期範圍：不可與現有規則重疊 |
| BR-006.5-02 | 轉換率範圍：1-1000（防止異常設定） |
| BR-006.5-03 | 手動重算：規則變更後需手動執行 CLI 指令重算積分（避免高峰期鎖表） |
| BR-006.5-04 | 優先順序：多個規則匹配時，使用創建時間最新的規則 |
| BR-006.5-05 | 預設規則：multiplier = 100（100 元 = 1 點） |

### **API Endpoints**

```
POST /api/admin/points-conversion-rules
  Body:
    - start_date: date (ROC format: yyymmdd)
    - end_date: date (optional)
    - multiplier: int (1-1000)
  Response: 200 OK + Warning log

GET /api/admin/points-conversion-rules
  Response: List of rules

PUT /api/admin/points-conversion-rules/:id
  Body: Updated rule
  Response: 200 OK + Warning log

DELETE /api/admin/points-conversion-rules/:id
  Response: 200 OK + Warning log

GET /api/admin/points-conversion-rules/active?date=yyymmdd
  Response: Applicable rule for given date
```

### **Manual Recalculation**

```bash
# Run during off-peak hours (e.g., 3:00 AM)
make recalculate-points

# CLI tool displays:
# - Current statistics (total users, verified transactions)
# - Confirmation prompt
# - Progress bar
# - Duration and result
```

---

## 🔧 Technical Implementation Notes

### **Frontend Stack**

- **Framework**: React 18 + TypeScript
- **UI Library**: Refine v5 + Ant Design 5
- **Routing**: React Router 7
- **State Management**: React Query (via Refine)
- **i18n**: react-i18next (Traditional Chinese / English)

### **Frontend Structure**

```
frontend/src/
├── pages/
│   ├── login.tsx
│   ├── callback.tsx (Google OAuth callback)
│   ├── users/
│   │   ├── list.tsx
│   │   ├── show.tsx
│   │   └── edit.tsx
│   ├── transactions/
│   │   ├── list.tsx
│   │   ├── show.tsx
│   │   └── edit.tsx
│   ├── surveys/
│   │   ├── list.tsx
│   │   ├── create.tsx
│   │   ├── edit.tsx
│   │   └── show.tsx
│   ├── points-conversion-rules/
│   │   ├── list.tsx
│   │   ├── create.tsx
│   │   └── edit.tsx
│   ├── ichef-import/
│   │   ├── upload.tsx
│   │   └── history.tsx
│   └── public-survey.tsx (no login required)
├── lib/
│   ├── auth-provider.ts
│   ├── data-provider.ts
│   ├── access-control-provider.ts
│   └── logger.ts
├── locales/
│   ├── zh-TW/
│   └── en/
└── components/
    ├── QuestionEditor.tsx
    └── ResponseViewer.tsx
```

### **Backend API Response Format**

All admin API endpoints use Refine-compatible format:

```json
{
  "data": { /* single resource */ },
  "total": 1
}

// Or for lists:
{
  "data": [ /* array of resources */ ],
  "total": 100
}
```

### **CORS Configuration**

```go
// Allow frontend origin
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:5173", "https://admin.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    AllowCredentials: true,
}))
```

### **RBAC Middleware**

```go
// Middleware for role-based access control
func RBACMiddleware(requiredRole AdminRole) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := c.MustGet("user").(*AdminUser)

        if !user.Role.HasPermission(requiredRole) {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Insufficient permissions",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// Usage
router.POST("/api/admin/surveys", RBACMiddleware(AdminRoleAdmin), handler.CreateSurvey)
router.GET("/api/admin/surveys", RBACMiddleware(AdminRoleUser), handler.ListSurveys)
```

---

## 🧪 Test Cases

### **Unit Tests**

- ✅ `TestAdminRole_Permissions`: 驗證角色權限
- ✅ `TestAuthProvider_GoogleOAuth`: Google OAuth 認證流程
- ✅ `TestRBACMiddleware_AccessControl`: RBAC 中間件測試
- ✅ `TestUserList_Pagination`: 會員列表分頁
- ✅ `TestUserList_Search`: 會員搜尋
- ✅ `TestTransactionUpdate_StatusFlow`: 交易狀態流轉
- ✅ `TestSurveyActivation_OnlyOneActive`: 單一問卷啟用
- ✅ `TestPointsRuleValidation_DateOverlap`: 日期重疊驗證

### **Integration Tests**

- ✅ `TestAdminPortal_EndToEnd`: 完整管理後台流程
- ✅ `TestGoogleOAuth_Login`: Google 登入流程
- ✅ `TestRBAC_AccessControl`: 角色權限控制

### **E2E Tests (Frontend)**

- ✅ Cypress tests for all CRUD operations
- ✅ Cypress tests for authentication flow
- ✅ Cypress tests for RBAC enforcement

---

## 📦 Dependencies

### **Internal Dependencies**

- **US-001 to US-005**: All user stories provide data for admin portal

### **External Dependencies**

- Google OAuth2: 認證服務
- PostgreSQL: 資料儲存
- React + TypeScript: 前端框架
- Refine + Ant Design: UI 框架

### **Service Dependencies**

- `AdminAuthService`: Google OAuth 認證
- `AdminUserService`: 管理員用戶管理
- `UserRepository`: 會員資料存取
- `TransactionRepository`: 交易資料存取
- `SurveyRepository`: 問卷資料存取
- `PointsConversionRuleRepository`: 積分規則存取

---

## 📊 Success Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| 登入成功率 | > 99% | 成功登入數 / 嘗試登入數 |
| 頁面載入時間 | < 3 秒 | 平均頁面載入時間 |
| 操作成功率 | > 99% | 成功操作數 / 總操作數 |
| 管理員使用頻率 | > 3 次/週 | 平均登入次數 |
| RBAC 準確率 | 100% | 正確權限控制數 / 總訪問數 |

---

## 🎯 User Personas

**Primary Persona**: 店長王姐（營運管理者）
- 35-45 歲餐廳店長
- 負責日常營運與會員管理
- 需要簡單易用的管理介面
- 期望快速查看關鍵數據

**Secondary Persona**: Admin 管理員
- 25-35 歲技術人員
- 負責系統設定與維護
- 需要完整的 CRUD 權限
- 期望彈性的配置選項

---

## 📝 UI/UX Screenshots

### **Login Page**

參考: `frontend/src/pages/login.tsx`

```
┌─────────────────────────────────────┐
│                                     │
│        🍽️ 餐廳會員管理系統         │
│                                     │
│    ┌─────────────────────────┐     │
│    │  🔐 使用 Google 登入    │     │
│    └─────────────────────────┘     │
│                                     │
└─────────────────────────────────────┘
```

### **Dashboard**

```
┌─────────────────────────────────────┐
│ 📊 儀表板                           │
├─────────────────────────────────────┤
│                                     │
│  📈 本月統計                        │
│  • 新增會員: 128                    │
│  • 交易筆數: 1,523                  │
│  • 發放積分: 15,487                 │
│  • 問卷完成: 456 (30%)              │
│                                     │
│  📋 待處理事項                      │
│  • 待驗證交易: 23 筆                │
│  • 未匹配發票: 5 筆                 │
│                                     │
└─────────────────────────────────────┘
```

### **User List**

```
┌─────────────────────────────────────┐
│ 👥 會員管理                         │
├─────────────────────────────────────┤
│ 🔍 [搜尋: 手機號碼或暱稱]          │
├─────────────────────────────────────┤
│ 暱稱      手機       累積積分       │
├─────────────────────────────────────┤
│ 小陳      0912***  125 點  [查看]  │
│ 小美      0987***  87 點   [查看]  │
│ ...                                 │
├─────────────────────────────────────┤
│ 第 1 頁，共 10 頁   [上一頁][下一頁]│
└─────────────────────────────────────┘
```

---

## 🔗 Related Documents

- [PRD.md](../PRD.md) - 完整產品需求文件（§ 2.6）
- [ADMIN_API.md](../../ADMIN_API.md) - 完整的管理後台 API 文件
- [FRONTEND_INTEGRATION.md](../../FRONTEND_INTEGRATION.md) - 前端整合指南
- [DOMAIN_MODEL.md](../../architecture/DOMAIN_MODEL.md) - AdminUser Entity 設計
- [DATABASE_DESIGN.md](../../architecture/DATABASE_DESIGN.md) - admin_users 表結構設計

---

## 📋 Future Enhancements (V4.0+)

### **V4.0: 進階儀表板**
- 即時數據視覺化（圖表）
- 自訂儀表板佈局
- 匯出報表（Excel, PDF）

### **V4.1: 進階權限管理**
- 自訂角色（細粒度權限）
- 權限繼承
- 操作日誌審計

### **V4.2: 批次操作**
- 批次更新交易狀態
- 批次發送通知
- 批次匯出資料

### **V4.3: 通知系統**
- Email 通知（異常狀況）
- LINE Notify 整合
- 推播通知

---

**Story Created**: 2025-01-08
**Last Updated**: 2025-01-08
**Story Owner**: Product Team
**Technical Owner**: Full Stack Team (Backend + Frontend)
