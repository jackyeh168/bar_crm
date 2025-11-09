# 戰略設計總覽 (Strategic Design Overview)

> **版本**: 1.0
> **最後更新**: 2025-01-08

---

## 1. 設計概述

### **1.1 設計目標**

本文檔提供餐廳會員管理 LINE Bot 系統的 Domain-Driven Design (DDD) 架構設計,目標是:

- **清晰的業務邊界**: 透過限界上下文劃分,建立清晰的業務能力邊界
- **核心域聚焦**: 識別積分管理為核心域,集中資源進行設計與優化
- **未來擴展性**: 考慮 V3.2+ 的積分兌換、優惠券等功能的架構彈性
- **技術實現指引**: 提供從領域概念到技術實現的清晰映射

### **1.2 設計原則**

- **限界上下文粒度**: 中粒度,根據業務能力劃分,平衡內聚性與獨立性
- **事件驅動**: 使用領域事件實現上下文間的鬆耦合集成
- **不變性保護**: 透過聚合根保護業務不變性規則
- **充血模型**: 領域邏輯封裝在領域對象內,而非分散在服務層

### **1.3 領域劃分策略**

基於 PRD 分析,本系統劃分為:

| 類型 | 上下文 | 業務價值 | 投資優先級 |
|------|--------|----------|-----------|
| **核心域** | 積分管理 | 核心競爭力,差異化業務邏輯 | 最高 ⭐⭐⭐ |
| **核心域** | 發票處理 | 關鍵業務流程,影響用戶體驗 | 高 ⭐⭐ |
| **支撐域** | 會員管理 | 支撐核心業務,標準化流程 | 中 ⭐ |
| **支撐域** | 問卷管理 | 增強用戶參與度 | 中 ⭐ |
| **支撐域** | 外部系統整合 | 資料驗證與同步 | 中 ⭐ |
| **通用域** | 身份與訪問 | 通用功能,可採用現成方案 | 低 |
| **通用域** | 通知服務 | 通用功能,LINE SDK 處理 | 低 |

---

## 2. 戰略設計

### **2.1 領域事件識別**

透過 Event Storming 方法,識別系統中的關鍵領域事件:

#### **會員管理領域事件**
- `MemberRegistered` - 會員已註冊
- `PhoneNumberBound` - 手機號碼已綁定
- `MemberProfileUpdated` - 會員資料已更新

#### **積分管理領域事件** (核心域 ⭐)
- `PointsEarned` - 積分已獲得 (交易驗證或問卷完成)
- `PointsDeducted` - 積分已扣除 (V3.2+ 兌換)
- `PointsExpired` - 積分已過期 (V3.3+)
- `PointsTransferred` - 積分已轉讓 (V4.0+)
- `ConversionRuleCreated` - 轉換規則已創建
- `ConversionRuleUpdated` - 轉換規則已更新
- `ConversionRuleDeleted` - 轉換規則已刪除
- `PointsRecalculationRequested` - 積分重算已請求
- `PointsRecalculated` - 積分已重算

#### **發票處理領域事件**
- `InvoiceQRCodeScanned` - 發票 QR Code 已掃描
- `InvoiceParsed` - 發票已解析
- `InvoiceValidated` - 發票已驗證 (通過所有驗證規則)
- `InvoiceRejected` - 發票已拒絕 (重複/過期/無效)
- `TransactionCreated` - 交易已創建 (狀態: imported)
- `TransactionVerified` - 交易已驗證 (狀態: verified)
- `TransactionFailed` - 交易已失敗 (狀態: failed)
- `InvoiceVoided` - 發票已作廢

#### **外部系統整合領域事件**
- `BatchImportStarted` - 批次匯入已開始
- `BatchImportCompleted` - 批次匯入已完成
- `InvoiceMatched` - 發票已匹配 (iChef 與會員掃描記錄)
- `InvoiceUnmatched` - 發票未匹配

#### **問卷管理領域事件**
- `SurveyCreated` - 問卷已創建
- `SurveyActivated` - 問卷已啟用
- `SurveyDeactivated` - 問卷已停用
- `SurveyResponseSubmitted` - 問卷回應已提交
- `SurveyRewardGranted` - 問卷獎勵已發放

### **2.2 命令識別**

| 上下文 | 命令 | 觸發者 | 產生事件 |
|--------|------|--------|----------|
| 會員管理 | `RegisterMember` | LINE Bot 用戶 | `MemberRegistered` |
| 會員管理 | `BindPhoneNumber` | LINE Bot 用戶 | `PhoneNumberBound` |
| 積分管理 | `EarnPoints` | 系統 | `PointsEarned` |
| 積分管理 | `RedeemPoints` | LINE Bot 用戶 | `PointsDeducted` |
| 積分管理 | `CreateConversionRule` | 管理員 | `ConversionRuleCreated` |
| 積分管理 | `RecalculateAllPoints` | 管理員 | `PointsRecalculated` |
| 發票處理 | `ScanInvoiceQRCode` | LINE Bot 用戶 | `InvoiceQRCodeScanned`, `TransactionCreated` |
| 發票處理 | `VerifyTransaction` | 系統 | `TransactionVerified` |
| 外部系統整合 | `ImportIChefBatch` | 管理員 | `BatchImportStarted`, `InvoiceMatched` |
| 問卷管理 | `SubmitSurveyResponse` | LINE Bot 用戶 | `SurveyResponseSubmitted`, `SurveyRewardGranted` |

### **2.3 業務不變性規則**

#### **會員管理**
- 每個 LINE 帳號只能綁定一個手機號碼
- 每個手機號碼只能綁定一個 LINE 帳號
- 手機號碼格式必須為 10 位數字,09 開頭

#### **積分管理** (核心業務規則 ⭐)
- 積分計算公式: `基礎積分 = floor(消費金額 / 轉換率)`
- 問卷獎勵: 每筆交易完成問卷額外 +1 點
- 累積積分 = Σ(所有已驗證交易的基礎積分 + 問卷獎勵)
- 轉換率日期範圍不可重疊
- 轉換率必須在 1-1000 之間
- 積分不可為負數
- 只有已驗證交易才計入累積積分

#### **發票處理**
- 發票有效期: 開立日期起 60 天內
- 同一發票號碼只能登錄一次 (唯一性約束)
- 發票必須通過解析才能創建交易
- 交易狀態流轉: `imported` → `verified` 或 `failed`

#### **問卷管理**
- 同一時間只能有一個啟用的問卷
- 同一筆交易的問卷只能填寫一次
- 問卷獎勵只有在交易驗證後才計入積分
