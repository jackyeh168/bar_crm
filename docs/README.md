# 📚 專案文件導航 (Project Documentation)

本目錄包含「餐廳會員管理 Line Bot」專案的所有核心文件。文件已按照 **受眾** 和 **關注點** 進行模組化組織，遵循 **Single Responsibility Principle (SRP)**。

---

## 🗂️ 文件結構

```
docs/
├── README.md                          # 📖 本文件 - 文件導航
├── architecture/                      # 🏗️  架構設計文件
│   ├── SYSTEM_DESIGN.md              #    系統架構總覽
│   ├── DOMAIN_MODEL.md               #    領域模型設計
│   └── DATABASE_DESIGN.md            #    資料庫設計
├── operations/                        # 🚀 運維部署文件
│   └── DEPLOYMENT.md                 #    部署與配置管理
├── product/                           # 📋 產品相關文件
│   ├── PRD.md                        #    產品需求文件
│   ├── stories/                      #    用戶故事
│   └── ui-ux/                        #    UI/UX 設計
└── qa/                                # 🧪 QA 測試文件
```

---

## 📖 文件總覽

### 1️⃣ **產品與業務文件** (給 PM/業務/QA)

**[product/PRD.md](./product/PRD.md)** - 產品需求文件
- **受眾**: Product Manager、業務人員、QA 工程師
- **內容**: 使用者故事、功能需求、驗收標準、發布計畫
- **篇幅**: ~540 行
- **特點**: 無程式碼、面向業務需求

**[product/stories/](./product/stories/)** - 用戶故事
- 開發任務拆解、技術實作細節

**[product/ui-ux/](./product/ui-ux/)** - UI/UX 設計
- 使用者介面設計、互動流程

---

### 2️⃣ **技術架構文件** (給架構師/技術主管)

**[architecture/SYSTEM_DESIGN.md](./architecture/SYSTEM_DESIGN.md)** - 系統架構總覽
- **受眾**: 架構師、技術主管、新進團隊成員
- **內容**: Clean Architecture 分層、技術棧選型、錯誤處理策略
- **篇幅**: ~492 行
- **特點**: 高層次架構視角、適合快速了解系統全貌

---

### 3️⃣ **開發實作文件** (給後端工程師)

**[architecture/DOMAIN_MODEL.md](./architecture/DOMAIN_MODEL.md)** - 領域模型設計
- **受眾**: 後端開發工程師、業務邏輯實作者
- **內容**:
  - § 1: 核心實體 (Entities + Value Objects)
  - § 2: Repository 介面
  - § 3: Use Case 介面
  - § 4: 積分計算邏輯
- **篇幅**: ~1016 行
- **特點**: 業務邏輯核心、獨立於框架和資料庫

**[architecture/DATABASE_DESIGN.md](./architecture/DATABASE_DESIGN.md)** - 資料庫設計
- **受眾**: DBA、基礎設施工程師、後端開發工程師
- **內容**:
  - § 1: 資料庫表結構 (SQL Schema)
  - § 2: ORM 模型定義 (GORM Models)
  - § 3: 映射層 (Entity ↔ ORM Mapper)
- **篇幅**: ~539 行
- **特點**: 資料層設計、索引優化、ORM 映射

---

### 4️⃣ **部署運維文件** (給 DevOps/SRE)

**[operations/DEPLOYMENT.md](./operations/DEPLOYMENT.md)** - 部署與配置管理
- **受眾**: DevOps、SRE、基礎設施工程師
- **內容**:
  - § 1: FX 依賴注入配置
  - § 2: 生產環境配置管理
  - § 3: 部署流程 (Docker、Kubernetes)
  - § 4: 監控和故障排除
- **篇幅**: ~1130 行
- **特點**: 運維實戰、配置管理、監控告警

---

### 5️⃣ **測試文件** (給 QA)

**[qa/](./qa/)** - QA 測試文件
- 測試計畫、測試案例、測試報告

---

## 🔗 文件關聯圖

```
                    ┌─────────────┐
                    │  PRD.md     │
                    │ (業務需求)   │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌───────────────┐  ┌──────────────┐  ┌──────────────┐
│SYSTEM_DESIGN  │  │DOMAIN_MODEL  │  │  DATABASE    │
│   .md         │  │   .md        │  │  _DESIGN.md  │
│ (架構總覽)    │  │ (領域模型)   │  │ (資料庫設計) │
└───────┬───────┘  └──────┬───────┘  └──────┬───────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │ DEPLOYMENT   │
                   │    .md       │
                   │  (部署運維)  │
                   └──────────────┘
```

---

## 📚 建議閱讀順序

### **新進團隊成員**
1. [../README.md](../README.md) - 專案快速上手
2. [product/PRD.md](./product/PRD.md) - 了解產品需求
3. [architecture/SYSTEM_DESIGN.md](./architecture/SYSTEM_DESIGN.md) - 理解架構設計
4. [architecture/DOMAIN_MODEL.md](./architecture/DOMAIN_MODEL.md) - 學習業務邏輯
5. [operations/DEPLOYMENT.md](./operations/DEPLOYMENT.md) - 熟悉部署流程

### **後端開發工程師**
1. [architecture/DOMAIN_MODEL.md](./architecture/DOMAIN_MODEL.md) - 業務邏輯實作
2. [architecture/DATABASE_DESIGN.md](./architecture/DATABASE_DESIGN.md) - 資料層實作
3. [architecture/SYSTEM_DESIGN.md](./architecture/SYSTEM_DESIGN.md) - 架構原則
4. [../CLAUDE.md](../CLAUDE.md) - 開發工具使用

### **DBA / 基礎設施工程師**
1. [architecture/DATABASE_DESIGN.md](./architecture/DATABASE_DESIGN.md) - 資料庫設計
2. [operations/DEPLOYMENT.md](./operations/DEPLOYMENT.md) - 部署配置
3. [architecture/SYSTEM_DESIGN.md](./architecture/SYSTEM_DESIGN.md) - 技術棧選型

### **DevOps / SRE**
1. [operations/DEPLOYMENT.md](./operations/DEPLOYMENT.md) - 部署運維
2. [architecture/SYSTEM_DESIGN.md](./architecture/SYSTEM_DESIGN.md) - 架構概覽
3. [architecture/DATABASE_DESIGN.md](./architecture/DATABASE_DESIGN.md) - 資料庫配置

### **Product Manager**
1. [product/PRD.md](./product/PRD.md) - 產品需求文件
2. [product/stories/](./product/stories/) - 開發任務
3. [product/ui-ux/](./product/ui-ux/) - UI/UX 設計

---

## ✅ 設計原則

所有文件拆分遵循以下原則：

1. **Single Responsibility Principle (SRP)** - 每個文件只關注一個面向
2. **按受眾分離** - 不同角色閱讀不同文件，降低認知負荷
3. **互相參照** - 文件間相互連結，便於深入學習
4. **可維護性** - 變更影響範圍小，易於更新
5. **模組化組織** - 按資料夾分類，易於查找

---

## 📝 文件維護指南

### **更新文件時的注意事項**

1. **架構變更**: 更新 `architecture/` 下的相關文件
2. **部署流程變更**: 更新 `operations/DEPLOYMENT.md`
3. **產品需求變更**: 更新 `product/PRD.md`
4. **新增功能**: 同時更新 PRD 和對應的技術文件
5. **跨文件參照**: 確保更新所有相關連結

### **版本控制**

所有技術文件都標註版本號和最後更新日期：
- **當前版本**: V3.1 (Clean Architecture Compliant)
- **最後更新**: 2025-01-08

---

## 🔍 快速查找

**想要查找...**

- 📋 **產品功能需求** → [product/PRD.md](./product/PRD.md)
- 🏗️ **系統架構設計** → [architecture/SYSTEM_DESIGN.md](./architecture/SYSTEM_DESIGN.md)
- 💾 **資料庫表結構** → [architecture/DATABASE_DESIGN.md](./architecture/DATABASE_DESIGN.md)
- 🔧 **業務邏輯實作** → [architecture/DOMAIN_MODEL.md](./architecture/DOMAIN_MODEL.md)
- 🚀 **部署配置步驟** → [operations/DEPLOYMENT.md](./operations/DEPLOYMENT.md)
- 🧪 **測試案例** → [qa/](./qa/)
- 📖 **用戶故事** → [product/stories/](./product/stories/)

---

**文件版本**: V3.1 (Clean Architecture Compliant)
**最後更新**: 2025-01-08
