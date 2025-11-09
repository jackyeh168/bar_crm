# User Story 001: 帳號綁定 (Account Binding)

**Story ID**: US-001
**Priority**: P0 (Must Have)
**Sprint**: Phase 1 - MVP Core Features
**Status**: ✅ Completed
**Estimated Effort**: 5 Story Points

---

## 📖 User Story

> **身為** 一位新顧客 (小美)，
> **我想要** 透過提供手機號碼來快速完成會員註冊，
> **以便** 我能開始使用所有會員功能。

---

## ✅ Acceptance Criteria

### **成功場景 1：首次註冊**

**Given** 新使用者加入 LINE Bot
**When** 點擊「加入好友」
**Then** 收到歡迎訊息並提示輸入手機號碼

**Given** 使用者輸入 10 位數手機號碼（09 開頭）
**When** 提交註冊
**Then** 顯示註冊成功訊息並說明如何使用積分功能

**Given** 註冊成功
**When** 查詢會員資料
**Then** 系統記錄 LINE ID、顯示名稱和手機號碼的綁定關係

---

### **失敗場景 1：手機號碼格式錯誤**

**Given** 使用者輸入非 10 位數字或非 09 開頭的號碼
**When** 提交註冊
**Then** 顯示錯誤訊息「手機號碼格式錯誤，請輸入 10 位數字，以 09 開頭」

---

### **失敗場景 2：手機號碼已被使用**

**Given** 使用者輸入已註冊的手機號碼
**When** 提交註冊
**Then** 顯示錯誤訊息「此手機號碼已被註冊」

---

## 📋 Business Rules

| Rule ID | Description |
|---------|-------------|
| BR-001-01 | 每個 LINE 帳號只能綁定一個手機號碼 |
| BR-001-02 | 每個手機號碼只能綁定一個 LINE 帳號 |
| BR-001-03 | 手機號碼格式：10 位數字，09 開頭（台灣手機號碼） |
| BR-001-04 | 註冊後無法自行解除綁定（需聯繫管理員） |

---

## 🔧 Technical Implementation Notes

### **Entity & Value Object**
- **User Entity**: 包含 `LineUserID`（唯一識別碼）和 `DisplayName`
- **Phone Value Object**: 封裝手機號碼驗證邏輯
  - 驗證規則：10 位數字，09 開頭
  - 不可變性：建立後無法修改

### **Use Case Interface**
```go
// internal/service/registration_service.go
type RegistrationService interface {
    RegisterUserWithPhone(lineUserID, displayName, phoneNumber string) error
    CheckUserRegistration(lineUserID string) (*User, error)
    ValidatePhoneNumber(phoneNumber string) error
}
```

### **Repository Interface**
```go
// internal/repository/user_repository.go
type UserRepository interface {
    Create(user *User) error
    FindByLineUserID(lineUserID string) (*User, error)
    FindByPhone(phone string) (*User, error)
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
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_line_user_id ON users(line_user_id);
CREATE INDEX idx_users_phone ON users(phone);
```

### **Error Handling**
- `ErrInvalidPhoneFormat`: 手機號碼格式錯誤
- `ErrPhoneAlreadyRegistered`: 手機號碼已被註冊
- `ErrUserAlreadyRegistered`: LINE 帳號已註冊

---

## 🧪 Test Cases

### **Unit Tests**
- ✅ `TestValidatePhoneNumber_ValidFormat`: 驗證正確格式的手機號碼
- ✅ `TestValidatePhoneNumber_InvalidFormat`: 驗證錯誤格式的手機號碼
- ✅ `TestRegisterUser_Success`: 成功註冊新用戶
- ✅ `TestRegisterUser_DuplicatePhone`: 重複手機號碼註冊失敗
- ✅ `TestRegisterUser_DuplicateLineUserID`: 重複 LINE ID 註冊失敗

### **Integration Tests**
- ✅ `TestRegistrationFlow_EndToEnd`: 完整註冊流程測試
- ✅ `TestRegistrationFlow_ConcurrentRegistration`: 並發註冊測試

---

## 📦 Dependencies

### **Internal Dependencies**
- None (MVP 核心功能，無內部依賴)

### **External Dependencies**
- LINE Bot SDK: 接收 Follow Event 和 Message Event
- PostgreSQL: 儲存用戶資料

### **Service Dependencies**
- `RegistrationService`: 處理註冊邏輯
- `UserRepository`: 資料存取層
- `LineBotService`: 發送訊息到 LINE Platform

---

## 📊 Success Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| 註冊成功率 | > 95% | 成功註冊數 / 嘗試註冊數 |
| 用戶註冊率 | > 40% | 完成綁定用戶數 / 關注 Bot 用戶數 |
| 註冊錯誤率 | < 5% | 格式錯誤 + 重複錯誤數 / 總註冊數 |
| 平均註冊時間 | < 60 秒 | 從加入好友到完成註冊的平均時間 |

---

## 🎯 User Personas

**Primary Persona**: 新客小美（潛在顧客）
- 20-30 歲學生或社會新鮮人
- 首次或第二次光顧
- 對科技體驗感興趣
- 期望簡單快速的註冊流程

**Secondary Persona**: 常客小陳（忠誠顧客）
- 25-35 歲上班族
- 每週至少光顧 2 次
- 希望快速開始使用積分功能

---

## 📝 UI/UX Flow

### **Wireframe Reference**
參考: `docs/product/ui-ux/registration-flow.png`（待建立）

### **Conversation Flow**

```
用戶: [加入 LINE Bot 好友]
Bot:
  ┌─────────────────────────────────┐
  │ 🎉 歡迎加入餐廳會員系統！       │
  │                                  │
  │ 請輸入您的手機號碼完成註冊：    │
  │ （格式：0912345678）            │
  └─────────────────────────────────┘

用戶: 0912345678
Bot:
  ┌─────────────────────────────────┐
  │ ✅ 註冊成功！                   │
  │                                  │
  │ 您現在可以：                     │
  │ 📸 上傳發票 QR Code 賺取積分    │
  │ 💰 輸入「積分」查詢餘額         │
  │ 📋 填寫問卷獲得額外獎勵         │
  └─────────────────────────────────┘
```

---

## 🔗 Related Documents

- [PRD.md](../PRD.md) - 完整產品需求文件
- [DOMAIN_MODEL.md](../../architecture/DOMAIN_MODEL.md) - User Entity 和 Phone Value Object 設計
- [DATABASE_DESIGN.md](../../architecture/DATABASE_DESIGN.md) - users 表結構設計
- [SYSTEM_DESIGN.md](../../architecture/SYSTEM_DESIGN.md) - 系統架構總覽

---

**Story Created**: 2025-01-08
**Last Updated**: 2025-01-08
**Story Owner**: Product Team
**Technical Owner**: Backend Team
