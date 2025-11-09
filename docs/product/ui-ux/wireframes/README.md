# 🎨 線框圖與設計資源

本目錄包含「餐廳會員管理 Line Bot」專案的所有線框圖、原型和設計資源。

---

## 📂 目錄結構

```
wireframes/
├── README.md                      # 📖 本文件 - 線框圖說明
├── registration-flow.png          # 會員註冊流程圖
├── qr-scanning-flow.png           # QR Code 掃描流程圖
├── points-query-flow.png          # 積分查詢流程圖
├── admin-dashboard.png            # 管理後台儀表板
├── admin-users-list.png           # 會員列表頁面
├── admin-transactions-list.png    # 交易列表頁面
├── admin-surveys-list.png         # 問卷列表頁面
├── survey-form.png                # 問卷表單設計
└── interactive-prototype/         # 互動原型（Figma/Adobe XD 連結）
```

---

## 🎯 線框圖清單

### **LINE Bot 對話流程**

#### 1. **會員註冊流程** (`registration-flow.png`)

**說明**:
- 用戶加入好友到完成註冊的完整流程
- 包含歡迎訊息、輸入手機號碼、驗證、成功/失敗回應

**涵蓋場景**:
- ✅ 首次註冊成功
- ❌ 手機號碼格式錯誤
- ❌ 手機號碼已被使用

**相關文件**: [LINE Bot 對話流程 - 註冊流程](../linebot-conversation-flows.md#註冊流程)

**預覽**:
```
(此處應放置實際的流程圖圖片)

建議工具: Figma, Draw.io, Lucidchart
尺寸: 1920x1080px 或 SVG 格式
```

---

#### 2. **QR Code 掃描流程** (`qr-scanning-flow.png`)

**說明**:
- 用戶上傳發票照片到獲得積分的完整流程
- 包含解析中狀態、成功/失敗回應、問卷連結

**涵蓋場景**:
- ✅ 成功解析並創建交易（待驗證）
- ✅ 發票已驗證（雙向驗證場景）
- ❌ 無法辨識 QR Code
- ❌ 發票已過期
- ❌ 重複發票

**相關文件**: [LINE Bot 對話流程 - QR Code 掃描](../linebot-conversation-flows.md#qr-code-掃描流程)

---

#### 3. **積分查詢流程** (`points-query-flow.png`)

**說明**:
- 用戶輸入關鍵字到顯示積分資訊的流程
- 包含詳細統計、新用戶提示

**涵蓋場景**:
- ✅ 有積分用戶查詢
- ℹ️  新用戶（無積分）查詢

**相關文件**: [LINE Bot 對話流程 - 積分查詢](../linebot-conversation-flows.md#積分查詢流程)

---

### **管理後台介面**

#### 4. **管理後台儀表板** (`admin-dashboard.png`)

**說明**:
- 管理後台首頁佈局
- 包含側邊欄導航、關鍵指標卡片、趨勢圖表、待處理事項

**元件清單**:
- 🧩 側邊欄導航
- 📊 關鍵指標卡片（4 個）
- 📈 本週趨勢圖
- ⚠️  待處理事項列表

**相關文件**: [管理後台設計 - 儀表板](../admin-portal-design.md#儀表板)

**參考尺寸**:
- 桌面: 1440x900px
- 平板: 1024x768px

---

#### 5. **會員列表頁面** (`admin-users-list.png`)

**說明**:
- 會員管理列表頁面佈局
- 包含搜尋列、篩選器、資料表格、分頁控制

**元件清單**:
- 🔍 搜尋框
- 🗂️  篩選下拉選單
- 📊 資料表格
- 📄 分頁控制
- 🔘 操作按鈕

**相關文件**: [管理後台設計 - 會員管理](../admin-portal-design.md#會員管理)

---

#### 6. **交易列表頁面** (`admin-transactions-list.png`)

**說明**:
- 交易管理列表頁面佈局
- 包含狀態篩選、日期範圍選擇、批次操作

**元件清單**:
- 🔍 搜尋框（發票號碼）
- 🏷️  狀態篩選（全部/已驗證/待驗證/作廢）
- 📅 日期範圍選擇器
- 📊 資料表格
- ✏️  編輯對話框

**相關文件**: [管理後台設計 - 交易管理](../admin-portal-design.md#交易管理)

---

#### 7. **問卷列表頁面** (`admin-surveys-list.png`)

**說明**:
- 問卷管理列表頁面佈局
- 包含問卷列表、啟用/停用狀態、統計預覽

**元件清單**:
- ➕ 建立新問卷按鈕
- 📊 問卷列表（標題、題數、回應數、完成率）
- 🟢 啟用狀態標示
- 📈 查看統計按鈕

**相關文件**: [管理後台設計 - 問卷管理](../admin-portal-design.md#問卷管理)

---

### **問卷填寫頁面**

#### 8. **問卷表單設計** (`survey-form.png`)

**說明**:
- 公開問卷填寫頁面佈局
- 包含三種題型（評分、選擇、文字）、進度指示器

**涵蓋題型**:
- ⭐ 評分題（1-5 星）
- ○ 選擇題（單選）
- ✍️  文字題（單行/多行）

**響應式版本**:
- 🖥️  桌面版（720px 最大寬度，置中）
- 📱 手機版（全寬，觸控優化）

**相關文件**: [問卷頁面設計](../survey-page-design.md)

---

## 🎨 設計規範

### **檔案命名規範**

```
格式: {模組}-{頁面類型}-{狀態}.png

範例:
- linebot-registration-success.png
- linebot-qr-scanning-error-expired.png
- admin-users-list-desktop.png
- admin-users-list-mobile.png
- survey-form-rating-question.png
```

### **尺寸規範**

| 平台 | 寬度 | 高度 | 格式 |
|------|------|------|------|
| LINE Bot 訊息 | 1040px | Auto | PNG |
| 管理後台（桌面） | 1440px | 900px | PNG/SVG |
| 管理後台（平板） | 1024px | 768px | PNG/SVG |
| 管理後台（手機） | 375px | 812px | PNG/SVG |
| 問卷頁面（桌面） | 720px | Auto | PNG/SVG |
| 問卷頁面（手機） | 375px | Auto | PNG/SVG |

### **匯出設定**

**Figma**:
```
匯出設定:
- 格式: PNG (2x)
- 背景: 透明（元件）或白色（完整頁面）
- 壓縮: 啟用
```

**Sketch**:
```
匯出設定:
- 格式: PNG @2x
- 色彩: sRGB
- 壓縮: 啟用
```

**Adobe XD**:
```
匯出設定:
- 格式: PNG
- 解析度: 2x (Retina)
- 背景: 白色
```

---

## 🔗 互動原型

### **Figma 原型連結**

建議使用 Figma 建立互動原型，包含：

1. **LINE Bot 對話流程**
   - 連結: `https://www.figma.com/proto/[project-id]...`
   - 描述: 完整的對話流程互動原型
   - 包含: 註冊、掃描、查詢三大流程

2. **管理後台原型**
   - 連結: `https://www.figma.com/proto/[project-id]...`
   - 描述: 管理後台所有頁面互動
   - 包含: 儀表板、會員、交易、問卷、積分規則、iChef 匯入

3. **問卷填寫流程**
   - 連結: `https://www.figma.com/proto/[project-id]...`
   - 描述: 問卷填寫完整體驗
   - 包含: 載入、填寫、驗證、提交、成功/錯誤狀態

---

## 📐 設計工具建議

### **推薦工具**

1. **Figma** ⭐⭐⭐⭐⭐
   - **優點**: 雲端協作、強大的原型功能、元件庫
   - **適用**: 所有設計階段
   - **連結**: https://www.figma.com/

2. **Sketch** ⭐⭐⭐⭐
   - **優點**: Mac 原生、豐富插件、精確控制
   - **適用**: UI 詳細設計
   - **連結**: https://www.sketch.com/

3. **Draw.io** ⭐⭐⭐⭐
   - **優點**: 免費、開源、流程圖專用
   - **適用**: 流程圖、架構圖
   - **連結**: https://app.diagrams.net/

4. **Lucidchart** ⭐⭐⭐
   - **優點**: 雲端、協作、豐富模板
   - **適用**: 流程圖、使用者旅程圖
   - **連結**: https://www.lucidchart.com/

---

## 📋 待建立的線框圖

以下線框圖尚待設計：

### **高優先級**

- [ ] **registration-flow.png** - 會員註冊流程圖
- [ ] **qr-scanning-flow.png** - QR Code 掃描流程圖
- [ ] **admin-dashboard.png** - 管理後台儀表板
- [ ] **survey-form.png** - 問卷表單設計

### **中優先級**

- [ ] **points-query-flow.png** - 積分查詢流程圖
- [ ] **admin-users-list.png** - 會員列表頁面
- [ ] **admin-transactions-list.png** - 交易列表頁面
- [ ] **admin-surveys-list.png** - 問卷列表頁面

### **低優先級**

- [ ] **admin-users-detail.png** - 會員詳情頁面
- [ ] **admin-surveys-create.png** - 建立問卷頁面
- [ ] **admin-surveys-statistics.png** - 問卷統計頁面
- [ ] **admin-ichef-import.png** - iChef 匯入頁面
- [ ] **admin-points-rules.png** - 積分規則管理頁面

---

## 🎨 元件庫

建議建立共用元件庫，包含：

### **基礎元件**

- 🔘 **按鈕**: Primary, Secondary, Text, Icon
- 📝 **輸入框**: Text, Number, TextArea, Date Picker
- ☑️  **選擇**: Checkbox, Radio, Select, Switch
- 📊 **表格**: Data Table, Pagination
- 🏷️  **標籤**: Badge, Tag, Status
- 📈 **圖表**: Line Chart, Bar Chart, Pie Chart
- 💬 **對話框**: Modal, Drawer, Popover, Tooltip
- ⚠️  **提示**: Alert, Notification, Toast
- 🔄 **載入**: Spinner, Skeleton, Progress Bar

### **複合元件**

- 🔍 **搜尋列**: Search Box + Filters
- 📊 **指標卡片**: Statistic Card with Trend
- 📋 **表單群組**: Form Section with Validation
- 🗂️  **資料表**: Table with Sorting & Pagination
- 🎯 **星級評分**: Rating Stars (Interactive)

---

## 📞 設計協作

### **設計審查流程**

1. **初稿設計**
   - 設計師建立線框圖
   - 上傳至 Figma/Sketch Cloud

2. **團隊審查**
   - Product Manager 審查業務邏輯
   - Frontend Developer 審查技術可行性
   - UX Designer 審查使用者體驗

3. **迭代修改**
   - 根據反饋調整設計
   - 更新 Figma 文件

4. **最終確認**
   - 匯出高解析度圖檔
   - 更新本 README 文件

### **版本控制**

- 所有線框圖使用 Git LFS 管理
- 檔案命名包含版本號: `admin-dashboard-v1.2.png`
- 重大變更在 Figma 中建立新版本

---

## 🔗 相關文件

- [UI/UX 設計導航](../README.md)
- [LINE Bot 對話流程設計](../linebot-conversation-flows.md)
- [管理後台介面設計](../admin-portal-design.md)
- [問卷頁面設計](../survey-page-design.md)
- [User Stories](../../stories/README.md)

---

## 📝 維護說明

### **新增線框圖**

1. 設計完成後匯出圖檔
2. 放入 `wireframes/` 目錄
3. 更新本 README 文件（新增描述）
4. 提交 Git commit

### **更新線框圖**

1. 在 Figma 中修改設計
2. 重新匯出圖檔（覆蓋舊檔）
3. 版本號 +1（如 v1.1 → v1.2）
4. 更新相關文件說明

---

**文件版本**: V3.1
**最後更新**: 2025-01-08
**維護者**: Product Team & UI/UX Designer

**注意**: 所有線框圖檔案尚未建立，本文件提供設計規範和建立指引。建議使用 Figma 進行設計，完成後匯出高解析度 PNG 圖檔放入本目錄。
