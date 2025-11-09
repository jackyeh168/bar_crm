# 📱 UI/UX 設計文件

本目錄包含「餐廳會員管理 Line Bot」專案的所有 UI/UX 設計文件，涵蓋 LINE Bot 對話介面、管理後台網頁、問卷填寫頁面等。

---

## 📂 文件結構

```
docs/product/ui-ux/
├── README.md                           # 📖 本文件 - UI/UX 導航
├── linebot-conversation-flows.md      # 💬 LINE Bot 對話流程設計
├── admin-portal-design.md             # 🖥️  管理後台介面設計
├── survey-page-design.md              # 📋 問卷頁面設計
├── audit-log-design.md                # 🔍 稽核日誌系統設計
└── wireframes/                         # 🎨 線框圖與設計資源
    ├── README.md                       #    線框圖說明
    ├── registration-flow.png           #    註冊流程示意圖
    ├── qr-scanning-flow.png            #    QR Code 掃描流程
    ├── admin-dashboard.png             #    管理後台儀表板
    ├── survey-form.png                 #    問卷表單設計
    └── audit-log-list.png              #    稽核日誌列表頁
```

---

## 🎯 設計原則

### **1. 簡單易用 (Simplicity)**
- **LINE Bot**: 減少操作步驟，一次對話完成一個任務
- **管理後台**: 清晰的導航結構，常用功能優先顯示
- **問卷頁面**: 簡潔的表單設計，避免過多資訊干擾

### **2. 一致性 (Consistency)**
- **視覺風格**: 統一的色彩、字體、間距
- **互動模式**: 相同操作使用相同的互動方式
- **訊息格式**: 錯誤提示、成功訊息使用一致的格式

### **3. 即時反饋 (Feedback)**
- **載入狀態**: 顯示處理中動畫（如 QR Code 解析）
- **操作結果**: 明確的成功/失敗訊息
- **進度指示**: 多步驟流程顯示當前進度

### **4. 容錯設計 (Error Prevention)**
- **輸入驗證**: 即時驗證格式錯誤（如手機號碼）
- **確認機制**: 重要操作（如刪除）需要確認
- **清楚的錯誤訊息**: 告知用戶問題和解決方法

### **5. 無障礙設計 (Accessibility)**
- **對比度**: 符合 WCAG 2.1 AA 標準
- **字體大小**: 可讀性優先（最小 14px）
- **觸控目標**: 按鈕最小 44x44px（符合 iOS/Android 指南）

---

## 📱 使用者介面總覽

### **會員端 (LINE Bot)**

| 功能 | 入口 | 設計文件 |
|------|------|---------|
| 帳號註冊 | 加入好友 | [LINE Bot 對話流程](./linebot-conversation-flows.md#註冊流程) |
| QR Code 掃描 | 上傳圖片 | [LINE Bot 對話流程](./linebot-conversation-flows.md#qr-code-掃描) |
| 積分查詢 | 輸入「積分」 | [LINE Bot 對話流程](./linebot-conversation-flows.md#積分查詢) |
| 問卷填寫 | 點擊連結 | [問卷頁面設計](./survey-page-design.md) |

### **管理端 (Web Portal)**

| 功能模組 | 主要頁面 | 設計文件 |
|---------|---------|---------|
| 認證 | 登入頁 | [管理後台設計](./admin-portal-design.md#登入頁面) |
| 會員管理 | 會員列表、會員詳情 | [管理後台設計](./admin-portal-design.md#會員管理) |
| 交易管理 | 交易列表、交易編輯 | [管理後台設計](./admin-portal-design.md#交易管理) |
| 問卷管理 | 問卷列表、問卷編輯、統計 | [管理後台設計](./admin-portal-design.md#問卷管理) |
| 積分規則 | 規則列表、規則設定 | [管理後台設計](./admin-portal-design.md#積分規則管理) |
| iChef 匯入 | 檔案上傳、匯入歷史 | [管理後台設計](./admin-portal-design.md#ichef-匯入) |
| 稽核日誌 | 日誌列表、詳情、匯出、異常監控 | [稽核日誌設計](./audit-log-design.md) |

---

## 🎨 視覺設計規範

### **色彩系統**

```
主色調 (Primary):
  - Primary: #1890ff (Ant Design Blue)
  - Primary Hover: #40a9ff
  - Primary Active: #096dd9

輔助色 (Secondary):
  - Success: #52c41a (綠色 - 成功狀態)
  - Warning: #faad14 (橘色 - 警告)
  - Error: #ff4d4f (紅色 - 錯誤)
  - Info: #1890ff (藍色 - 資訊)

中性色 (Neutral):
  - Text Primary: rgba(0, 0, 0, 0.85)
  - Text Secondary: rgba(0, 0, 0, 0.65)
  - Text Disabled: rgba(0, 0, 0, 0.25)
  - Border: #d9d9d9
  - Background: #f0f2f5
```

### **字體系統**

```
字體家族:
  - 中文: "PingFang TC", "Microsoft JhengHei", sans-serif
  - 英文: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif

字體大小:
  - H1: 24px / 1.35 (頁面標題)
  - H2: 20px / 1.4 (區塊標題)
  - H3: 16px / 1.5 (小標題)
  - Body: 14px / 1.5715 (內文)
  - Small: 12px / 1.66 (輔助文字)

字重:
  - Regular: 400 (一般文字)
  - Medium: 500 (強調文字)
  - Semibold: 600 (標題)
```

### **間距系統**

```
基礎單位: 8px

間距規範:
  - xs: 8px
  - sm: 12px
  - md: 16px
  - lg: 24px
  - xl: 32px
  - xxl: 48px
```

### **圓角系統**

```
  - None: 0px (表格、分隔線)
  - Small: 2px (標籤、徽章)
  - Base: 4px (按鈕、輸入框)
  - Large: 8px (卡片、對話框)
```

### **陰影系統**

```
  - Level 1: 0 2px 8px rgba(0, 0, 0, 0.15) (卡片)
  - Level 2: 0 4px 16px rgba(0, 0, 0, 0.15) (懸浮選單)
  - Level 3: 0 8px 24px rgba(0, 0, 0, 0.15) (對話框)
```

---

## 📐 佈局規範

### **響應式斷點**

```
xs: < 576px   (手機直向)
sm: ≥ 576px   (手機橫向)
md: ≥ 768px   (平板)
lg: ≥ 992px   (桌面)
xl: ≥ 1200px  (大螢幕)
xxl: ≥ 1600px (超大螢幕)
```

### **Grid 系統**

```
- 24 欄網格系統 (Ant Design Grid)
- Gutter: 16px (預設間距)
- Container Max Width: 1200px
```

---

## 💬 LINE Bot 對話設計重點

### **訊息格式規範**

1. **歡迎訊息**
   - 使用友善的問候語
   - 清楚說明下一步操作
   - 包含 emoji 增加親和力（適度使用）

2. **錯誤訊息**
   - 明確指出問題
   - 提供解決方法
   - 使用 ❌ 或 ⚠️ emoji

3. **成功訊息**
   - 簡短確認操作結果
   - 顯示關鍵資訊（如積分、金額）
   - 使用 ✅ emoji

### **對話流程設計原則**

1. **線性流程**: 一次完成一個任務，避免分支過多
2. **狀態管理**: 使用 Redis 記錄會話狀態（如註冊進度）
3. **錯誤恢復**: 提供重試機制，避免用戶重新開始
4. **超時處理**: 5 分鐘無回應則清除狀態

---

## 🖥️ 管理後台設計重點

### **導航結構**

```
側邊欄 (Sidebar):
├── 📊 儀表板 (Dashboard)
├── 👥 會員管理 (Users)
├── 💳 交易管理 (Transactions)
├── 📋 問卷管理 (Surveys)
├── 🎯 積分規則 (Points Rules)
├── 📥 iChef 匯入 (iChef Import)
├── 🔍 稽核日誌 (Audit Logs)
└── ⚙️  系統設定 (Settings)
```

### **頁面佈局模式**

1. **列表頁 (List Page)**
   - 搜尋列 + 篩選器（頂部）
   - 操作按鈕（新增、匯出）
   - 資料表格（可排序、分頁）
   - 批次操作（可選）

2. **詳情頁 (Detail Page)**
   - 頁面標題 + 操作按鈕
   - 資訊卡片（分組顯示）
   - 相關資料（標籤頁）

3. **編輯頁 (Edit Page)**
   - 表單（分組、驗證）
   - 儲存 / 取消按鈕
   - 即時驗證提示

---

## 📋 問卷頁面設計重點

### **表單設計原則**

1. **簡潔性**: 一頁顯示所有問題（避免多頁切換）
2. **必填標示**: 使用 * 標示必填題
3. **即時驗證**: 提交前驗證完整性
4. **進度提示**: 顯示已填寫 / 總題數

### **題型設計**

1. **文字題**: 單行或多行輸入框
2. **選擇題**: Radio 按鈕（單選）
3. **評分題**: 星級評分（1-5 星，支援半星）

---

## 🔗 相關文件連結

### **User Stories**
- [US-001: 帳號綁定](../stories/US-001-account-binding.md)
- [US-002: QR Code 掃描](../stories/US-002-qr-code-scanning-points.md)
- [US-003: 積分查詢](../stories/US-003-points-balance-query.md)
- [US-004: 問卷系統](../stories/US-004-survey-system.md)
- [US-005: iChef 整合](../stories/US-005-ichef-integration.md)
- [US-006: 管理後台](../stories/US-006-admin-portal.md)
- [US-007: 稽核日誌系統](../stories/US-007-audit-log-system.md)

### **技術文件**
- [PRD.md](../PRD.md) - 產品需求文件
- [SYSTEM_DESIGN.md](../../architecture/SYSTEM_DESIGN.md) - 系統架構設計
- [FRONTEND_INTEGRATION.md](../../../FRONTEND_INTEGRATION.md) - 前端整合指南

### **設計資源**
- [Ant Design 5](https://ant.design/) - UI 元件庫
- [LINE Design System](https://designsystem.line.me/) - LINE 設計規範
- [Refine v5](https://refine.dev/) - 管理後台框架

---

## 🎨 設計工具與資源

### **推薦設計工具**
- **Figma**: 協作設計、原型製作
- **Sketch**: Mac 平台設計工具
- **Adobe XD**: 跨平台設計工具

### **圖示資源**
- [Ant Design Icons](https://ant.design/components/icon/) - 官方圖示庫
- [Heroicons](https://heroicons.com/) - 簡潔線性圖示
- [Feather Icons](https://feathericons.com/) - 輕量圖示集

### **插畫資源**
- [unDraw](https://undraw.co/) - 免費 SVG 插畫
- [Illustrations](https://illlustrations.co/) - 開源插畫庫

---

## ✅ UI/UX 檢查清單

### **開發前檢查**
- [ ] 所有流程圖已完成
- [ ] 線框圖已審查並通過
- [ ] 視覺設計規範已確認
- [ ] 互動原型已測試
- [ ] 稽核日誌介面設計已完成

### **開發中檢查**
- [ ] 元件實作符合設計規範
- [ ] 響應式佈局正確
- [ ] 色彩對比度符合標準
- [ ] 字體大小可讀
- [ ] 敏感資料遮罩正確顯示

### **上線前檢查**
- [ ] 所有使用者流程已測試
- [ ] 錯誤狀態顯示正確
- [ ] 載入狀態顯示正確
- [ ] 多裝置相容性測試通過
- [ ] 稽核日誌匯出功能測試通過
- [ ] GDPR 資料刪除流程測試通過

---

## 📞 設計團隊聯絡

如有設計相關問題或建議，請聯繫：

- **UI/UX Designer**: [待指派]
- **Product Manager**: Sarah
- **Frontend Lead**: Frontend Team

---

**文件版本**: V3.1
**最後更新**: 2025-01-08
**維護者**: Product Team & Design Team
