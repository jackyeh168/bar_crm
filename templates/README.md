# 代碼模板（Code Templates）

## 目的

提供符合架構約束的標準代碼模板，確保實作一致性並避免常見錯誤。

## 使用方式

1. 根據實作內容選擇對應的模板
2. 複製模板代碼
3. 替換模板變量（如 `{{.Package}}`, `{{.TypeName}}`）
4. 根據業務需求調整細節

## 可用模板

| 模板文件 | 用途 | 使用時機 |
|---------|------|---------|
| `domain_error.go.template` | 領域錯誤定義 | 創建新的 bounded context 時 |
| `value_object.go.template` | 值對象實作 | 實作新的值對象時 |
| `aggregate.go.template` | 聚合根實作 | 實作新的聚合根時 |
| `repository.go.template` | Repository 接口 | 定義 Repository 接口時 |
| `use_case.go.template` | Use Case 實作 | 實作新的用例時 |

## 模板變量說明

模板中使用 `{{.VariableName}}` 表示需要替換的變量：

| 變量 | 說明 | 示例 |
|------|------|------|
| `{{.Package}}` | 包名 | `points`, `member`, `invoice` |
| `{{.TypeName}}` | 類型名稱 | `PointsAmount`, `MemberID` |
| `{{.FieldName}}` | 字段名稱 | `value`, `amount` |
| `{{.MethodName}}` | 方法名稱 | `Add`, `Subtract` |
| `{{.ErrorCode}}` | 錯誤代碼 | `POINTS_NEGATIVE`, `MEMBER_NOT_FOUND` |

## 快速開始示例

### 示例 1: 創建新的值對象

**需求**：創建 `ConversionRate` 值對象

**步驟**：

1. 複製 `value_object.go.template`
2. 替換變量：
   - `{{.Package}}` → `points`
   - `{{.TypeName}}` → `ConversionRate`
   - `{{.FieldType}}` → `int`
   - `{{.FieldName}}` → `value`
3. 添加業務驗證邏輯（如範圍檢查 1-1000）
4. 添加業務方法（如 `CalculatePoints`）

### 示例 2: 創建新的錯誤定義

**需求**：為 `invoice` bounded context 定義錯誤

**步驟**：

1. 複製 `domain_error.go.template`
2. 替換變量：
   - `{{.Package}}` → `invoice`
3. 定義具體錯誤：
   ```go
   const (
       ErrCodeInvoiceNotFound ErrorCode = "INVOICE_NOT_FOUND"
       ErrCodeInvoiceDuplicate ErrorCode = "INVOICE_DUPLICATE"
   )
   ```

## 注意事項

⚠️ **模板只是起點，不是終點**

- ✅ 模板提供符合架構約束的基礎結構
- ✅ 根據業務需求調整和擴展
- ❌ 不要盲目複製，要理解每一行代碼的作用

⚠️ **使用模板前先閱讀架構文檔**

- 模板無法涵蓋所有情況
- 複雜場景需參考架構文檔
- 參考 `docs/implementation-checklist.md`

## 相關文檔

- **實作檢查清單**: `docs/implementation-checklist.md`
- **核心架構約束**: `CLAUDE.md`
- **DDD 架構指南**: `docs/architecture/ddd/README.md`
