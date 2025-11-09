# **部署與運維指南 (Deployment Guide): 餐廳會員管理 Line Bot**

*   **版本**: 3.1
*   **最後更新**: 2025-01-08
*   **維護者**: DevOps Team

> **文件目的**: 本文檔提供系統的部署、配置管理、依賴注入和運維監控指南。
>
> **相關文檔**:
> - [PRD.md](../product/PRD.md) - 產品需求文件
> - [SYSTEM_DESIGN.md](../architecture/SYSTEM_DESIGN.md) - 系統架構總覽
> - [DOMAIN_MODEL.md](../architecture/DOMAIN_MODEL.md) - 領域模型與業務邏輯
> - [DATABASE_DESIGN.md](../architecture/DATABASE_DESIGN.md) - 資料庫設計與 ORM

---

## **目錄**

1. [FX 依賴注入配置](#1-fx-依賴注入配置)
2. [生產環境配置管理](#2-生產環境配置管理)
3. [部署流程](#3-部署流程)
4. [監控和故障排除](#4-監控和故障排除)

---

## **1. FX 依賴注入配置**

### **1.1 模組載入順序**

系統使用 **Uber FX** 進行依賴注入，模組載入順序遵循依賴關係：

```go
fx.New(
    // Register all modules
    providers.LoggerModule,                    // 1. 日誌系統 (無相依)
    providers.ConfigModule,                    // 2. 配置管理 (依賴: Logger)
    providers.DatabaseModule,                  // 3. 資料庫連接 (依賴: Config, Logger)
    providers.RepositoryModule,                // 4. 資料存取層 (依賴: Database)
    providers.RedisModule,                     // 5. Redis 快取 (依賴: Config, Logger)
    providers.RegistrationModule,              // 6. 用戶註冊 (依賴: Repository, Logger)
    providers.PointsModule,                    // 7. 積分服務 (依賴: Repository, Logger)
    providers.LineBotModule,                   // 8. LINE Bot (依賴: Config, Logger)
    providers.AdminModule,                     // 9. 管理後台 (依賴: Repository, Config)
    providers.LineUserModule,                  // 10. LINE 用戶管理 (依賴: Repository)
    providers.TransactionModule,               // 11. 交易管理 (依賴: Repository)
    providers.IChefImportModule,               // 12. iChef 匯入 (依賴: Repository)
    providers.SurveyModule,                    // 13. 問卷系統 (依賴: Repository)
    providers.PointsConversionRuleModule,      // 14. 積分規則 (依賴: Repository)
    providers.HandlerModule,                   // 15. HTTP 處理器 (依賴: 所有業務服務)
    providers.ServerModule,                    // 16. HTTP 伺服器 (依賴: Handler, Config)
)
```

### **1.2 相依關係圖**

```
LoggerModule (無相依)
    ↓
ConfigModule (Logger)
    ↓
DatabaseModule (Config, Logger)
    ↓
RepositoryModule (Database)
    ├─→ RedisModule (Config, Logger)
    ↓
業務邏輯層 (Business Layer)
    ├─→ RegistrationModule (Repository, Logger)
    ├─→ PointsModule (Repository, Logger)
    ├─→ LineBotModule (Config, Logger)
    ├─→ AdminModule (Repository, Config)
    ├─→ LineUserModule (Repository)
    ├─→ TransactionModule (Repository)
    ├─→ IChefImportModule (Repository)
    ├─→ SurveyModule (Repository)
    └─→ PointsConversionRuleModule (Repository)
    ↓
介面層 (Interface Layer)
    ├─→ HandlerModule (所有業務服務)
    └─→ ServerModule (Handler, Config)
```

### **1.3 關鍵模組說明**

#### **1. LoggerModule (基礎層)**
- **提供**: `*zap.Logger`
- **相依**: 無
- **被依賴**: 所有其他模組
- **職責**: 提供結構化日誌記錄

#### **2. ConfigModule (基礎層)**
- **提供**: `*AppConfig`
- **相依**: `*zap.Logger`
- **被依賴**: DatabaseModule, LineBotModule, AdminModule, RedisModule, ServerModule
- **職責**: 環境變數驗證和配置管理

#### **3. DatabaseModule (資料層)**
- **提供**: `*gorm.DB`
- **相依**: `*AppConfig`, `*zap.Logger`
- **被依賴**: RepositoryModule
- **職責**: 資料庫連接池管理
- **特性**: 可選模組（無 DB 配置時自動跳過）

#### **4. RepositoryModule (資料存取層)**
- **提供**: 所有 Repository 介面實作
  - `UserRepository`
  - `UserPointsSummaryRepository`
  - `TransactionRepository`
  - `SurveyRepository`
  - `PointsConversionRuleRepository`
  - `IChefImportHistoryRepository`
  - `IChefInvoiceRecordRepository`
- **相依**: `*gorm.DB` (可選)
- **被依賴**: 所有業務邏輯模組
- **職責**: 提供資料存取抽象
- **特性**: 無資料庫時自動使用 Mock Repository

#### **5. RedisModule (快取層)**
- **提供**: `RedisService` 介面
- **相依**: `*AppConfig`, `*zap.Logger`
- **被依賴**: 業務邏輯層（用於會話狀態管理）
- **職責**: 快取和會話管理
- **特性**: 可選模組（無 Redis 配置時 fallback 到 in-memory）

#### **6-14. 業務邏輯模組**
- **RegistrationModule**: 用戶註冊和手機號碼綁定
- **PointsModule**: 積分查詢和計算
- **LineBotModule**: LINE Bot SDK 封裝
- **AdminModule**: Google OAuth2 認證和 JWT 管理
- **LineUserModule**: LINE 用戶管理
- **TransactionModule**: 發票交易管理
- **IChefImportModule**: iChef POS 系統整合
- **SurveyModule**: 問卷調查系統
- **PointsConversionRuleModule**: 積分轉換規則管理

#### **15. HandlerModule (介面層)**
- **提供**: `handler.LinebotHandler`, `handler.AdminHandler`
- **相依**: 所有業務服務介面
- **被依賴**: ServerModule
- **職責**: HTTP 請求處理和路由

#### **16. ServerModule (基礎設施層)**
- **提供**: HTTP 伺服器生命週期管理
- **相依**: Handlers, `*AppConfig`
- **被依賴**: 無
- **職責**: Gin 伺服器啟動和優雅關閉

### **1.4 模組載入順序驗證**

#### **正確順序 (當前)**
✅ **LoggerModule** → ConfigModule → DatabaseModule → RepositoryModule → RedisModule → 業務邏輯層 → 介面層

#### **錯誤順序範例**
❌ ConfigModule → LoggerModule (ConfigModule 需要 Logger)
❌ PointsModule → RepositoryModule (PointsModule 需要 Repository)
❌ DatabaseModule → ConfigModule (Database 需要 Config)

### **1.5 FX 相依性檢查**

#### **啟動時檢查**
FX 會在啟動時自動進行相依性檢查：
1. **循環相依檢測** - 檢查是否存在相依循環
2. **缺少相依檢測** - 檢查是否有未滿足的相依
3. **類型匹配檢查** - 驗證相依類型是否正確

#### **常見相依問題**

1. **循環相依**
   ```
   錯誤: [Fx] SUPPLY MISSING: missing type: *service.PointsService
   原因: PointsModule 和某個模組形成循環相依
   解決: 檢查模組間的依賴關係，確保單向依賴
   ```

2. **缺少相依**
   ```
   錯誤: [Fx] PROVIDE MISSING: *gorm.DB not provided
   原因: RepositoryModule 載入在 DatabaseModule 之前
   解決: 調整模組載入順序
   ```

3. **重複提供**
   ```
   錯誤: [Fx] PROVIDE DUPLICATE: *zap.Logger provided twice
   原因: 多個模組重複提供同一類型
   解決: 確保每個類型只被一個模組提供
   ```

### **1.6 測試和驗證**

#### **單元測試模組相依**
```go
func TestModuleDependencies(t *testing.T) {
    app := fx.New(
        // 按正確順序載入模組
        providers.LoggerModule,
        providers.ConfigModule,
        providers.DatabaseModule,
        // ... 其他模組

        // 測試模式：不啟動伺服器
        fx.NopLogger,
    )

    require.NoError(t, app.Err())

    ctx := context.Background()
    require.NoError(t, app.Start(ctx))
    require.NoError(t, app.Stop(ctx))
}
```

#### **相依注入驗證**
```go
func TestDependencyInjection(t *testing.T) {
    var pointsService service.PointsServiceInterface

    app := fx.New(
        // 載入所有必要模組
        providers.AllModules...,

        // 提取服務進行驗證
        fx.Populate(&pointsService),
        fx.NopLogger,
    )

    require.NoError(t, app.Err())
    require.NotNil(t, pointsService)
}
```

### **1.7 最佳實踐**

1. ✅ **明確模組邊界** - 每個模組職責清晰，單一職責
2. ✅ **使用介面抽象** - 減少模組間耦合
3. ✅ **可選相依設計** - 支援功能開關和測試
4. ✅ **啟動時驗證** - 確保所有相依正確滿足
5. ✅ **監控和日誌** - 追蹤啟動性能和問題
6. ✅ **測試覆蓋** - 驗證相依注入正確性

#### **介面抽象範例**
```go
// ✅ 好的做法：依賴介面
type PointsHandler struct {
    pointsService service.PointsServiceInterface  // 介面
    linebotService service.LineBotServiceInterface // 介面
}

// ❌ 避免：依賴具體類型
type PointsHandler struct {
    pointsService *service.PointsService  // 具體類型
}
```

#### **可選相依範例**
```go
// 資料庫是可選的
func NewUserRepository(db *gorm.DB, logger *zap.Logger) repository.UserRepository {
    if db == nil {
        logger.Warn("Database not configured, using mock repository")
        return repository.NewMockUserRepository()
    }
    return repository.NewGormUserRepository(db)
}
```

---

## **2. 生產環境配置管理**

### **2.1 配置架構概述**

#### **配置層級**
1. **應用預設值** - 程式碼中的預設配置
2. **環境變數** - 生產環境特定配置
3. **驗證層** - 啟動時配置驗證
4. **運行時檢查** - 動態配置健康檢查

#### **配置類型**
- **必填配置** - 系統無法運行時缺失的配置
- **可選配置** - 有預設值的配置
- **敏感配置** - 密碼、密鑰等安全敏感配置
- **功能開關** - 啟用/禁用特定功能的配置

### **2.2 必填配置項目**

#### **LINE Bot 核心配置**
```bash
# LINE Developer Console 提供的認證資訊
CHANNEL_SECRET=your_line_bot_channel_secret
CHANNEL_TOKEN=your_line_bot_access_token
```

**驗證規則**:
- `CHANNEL_SECRET`: 必填，非空字符串
- `CHANNEL_TOKEN`: 必填，非空字符串

**取得方式**:
1. 登入 [LINE Developers Console](https://developers.line.biz/console/)
2. 選擇你的 Messaging API Channel
3. 在 "Basic settings" 找到 `Channel secret`
4. 在 "Messaging API" 找到 `Channel access token`

#### **應用服務配置**
```bash
# HTTP 服務配置
PORT=8080                    # 服務埠號 (1024-65535)
GIN_MODE=release            # 運行模式 (release/debug/test)
```

**驗證規則**:
- `PORT`: 1024-65535 範圍內的整數
- `GIN_MODE`: 必須是 release、debug 或 test

### **2.3 可選配置項目**

#### **資料庫配置 (可選)**
```bash
# PostgreSQL 連接配置
DB_HOST=localhost
DB_USER=linebot_user
DB_PASSWORD=secure_password
DB_NAME=linebot_bar
DB_PORT=5432
DB_SSL_MODE=disable         # disable/require/verify-ca/verify-full
```

**驗證規則**:
- 如果 `DB_HOST` 存在，則其他資料庫配置為必填
- `DB_PORT`: 1-65535 範圍內的整數
- `DB_SSL_MODE`: 必須是允許的SSL模式之一

**系統行為**:
- ✅ **有 DB 配置**: 使用 PostgreSQL + GORM Repository
- ⚠️ **無 DB 配置**: 自動 fallback 到 Mock Repository（僅供開發測試）

#### **Redis 配置 (可選)**
```bash
# Redis 連接配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

**系統行為**:
- ✅ **有 Redis 配置**: 使用 Redis 儲存會話狀態
- ⚠️ **無 Redis 配置**: 自動 fallback 到 in-memory storage（會話不持久化）

#### **Admin 後台配置 (可選)**
```bash
# Google OAuth2 配置
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:5173/callback

# JWT 配置
JWT_SECRET=your_jwt_secret_key
JWT_ISSUER=linebot-bar

# 預設管理員
DEFAULT_ADMIN_EMAIL=admin@example.com
```

**系統行為**:
- ✅ **有 Admin 配置**: 啟用管理後台功能
- ⚠️ **無 Admin 配置**: Admin API 端點返回 503 Service Unavailable

#### **前端配置**
```bash
# 前端 API 基礎 URL
VITE_API_URL=http://localhost:8080/api

# 問卷連結基礎 URL
FRONTEND_URL=http://localhost:5173
```

#### **日誌配置**
```bash
# 日誌級別控制
LOG_LEVEL=info              # debug/info/warn/error
```

### **2.4 配置驗證機制**

#### **啟動時驗證**
系統啟動時自動執行配置驗證：

```go
func ValidateProductionConfig() error {
    // 1. 檢查所有必填配置
    // 2. 驗證配置格式和範圍
    // 3. 測試可選配置的可用性
    // 4. 返回詳細的驗證錯誤
}
```

#### **驗證錯誤範例**
```
配置驗證失敗:
[CHANNEL_SECRET]: 必填環境變數 - LINE Bot 頻道密鑰 (值: <空值>)
[DB_PORT]: 整數必須在 1 到 65535 之間 - 資料庫埠號 (值: 70000)
[GIN_MODE]: 必須是 release、debug 或 test - Gin 運行模式 (值: production)
```

### **2.5 部署環境配置**

#### **開發環境 (.env.development)**
```bash
# 開發環境配置 - 寬鬆的驗證和調試設定
GIN_MODE=debug
LOG_LEVEL=debug
PORT=8080

# 測試用的LINE Bot憑證
CHANNEL_SECRET=test_channel_secret
CHANNEL_TOKEN=test_channel_token

# 本地資料庫（可選）
DB_HOST=localhost
DB_USER=dev_user
DB_PASSWORD=dev_password
DB_NAME=linebot_bar_dev
DB_PORT=5432
DB_SSL_MODE=disable

# 本地 Redis（可選）
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# 本地 Admin 後台
GOOGLE_CLIENT_ID=dev_google_client_id
GOOGLE_CLIENT_SECRET=dev_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:5173/callback
JWT_SECRET=dev_jwt_secret_key_change_in_production
JWT_ISSUER=linebot-bar-dev
DEFAULT_ADMIN_EMAIL=dev@example.com

# 前端配置
VITE_API_URL=http://localhost:8080/api
FRONTEND_URL=http://localhost:5173
```

#### **測試環境 (.env.staging)**
```bash
# 測試環境配置 - 接近生產的設定
GIN_MODE=release
LOG_LEVEL=info
PORT=8080

# 測試環境LINE Bot憑證
CHANNEL_SECRET=staging_channel_secret
CHANNEL_TOKEN=staging_channel_token

# 測試資料庫
DB_HOST=staging-db.company.com
DB_USER=staging_user
DB_PASSWORD=staging_secure_password
DB_NAME=linebot_bar_staging
DB_PORT=5432
DB_SSL_MODE=require

# 測試 Redis
REDIS_HOST=staging-redis.company.com
REDIS_PORT=6379
REDIS_PASSWORD=staging_redis_password
REDIS_DB=0

# 測試 Admin 後台
GOOGLE_CLIENT_ID=staging_google_client_id
GOOGLE_CLIENT_SECRET=staging_google_client_secret
GOOGLE_REDIRECT_URL=https://staging-admin.company.com/callback
JWT_SECRET=staging_jwt_secret_key
JWT_ISSUER=linebot-bar-staging
DEFAULT_ADMIN_EMAIL=staging-admin@company.com

# 前端配置
VITE_API_URL=https://staging-api.company.com/api
FRONTEND_URL=https://staging-admin.company.com
```

#### **生產環境 (.env.production)**
```bash
# 生產環境配置 - 最嚴格的安全設定
GIN_MODE=release
LOG_LEVEL=warn
PORT=8080

# 生產LINE Bot憑證 (從外部密鑰管理系統注入)
CHANNEL_SECRET=${LINE_CHANNEL_SECRET}
CHANNEL_TOKEN=${LINE_CHANNEL_TOKEN}

# 生產資料庫配置
DB_HOST=prod-db.company.com
DB_USER=${DB_PROD_USER}
DB_PASSWORD=${DB_PROD_PASSWORD}
DB_NAME=linebot_bar_prod
DB_PORT=5432
DB_SSL_MODE=verify-full

# 生產 Redis 配置
REDIS_HOST=prod-redis.company.com
REDIS_PORT=6379
REDIS_PASSWORD=${REDIS_PROD_PASSWORD}
REDIS_DB=0

# 生產 Admin 後台
GOOGLE_CLIENT_ID=${GOOGLE_PROD_CLIENT_ID}
GOOGLE_CLIENT_SECRET=${GOOGLE_PROD_CLIENT_SECRET}
GOOGLE_REDIRECT_URL=https://admin.company.com/callback
JWT_SECRET=${JWT_PROD_SECRET}
JWT_ISSUER=linebot-bar-production
DEFAULT_ADMIN_EMAIL=admin@company.com

# 前端配置
VITE_API_URL=https://api.company.com/api
FRONTEND_URL=https://admin.company.com
```

### **2.6 安全配置管理**

#### **密鑰輪換策略**

1. **LINE Bot 憑證**
   - **定期輪換頻率**: 每 6 個月
   - **緊急輪換**: 懷疑洩露時立即執行
   - **輪換程序**:
     1. LINE Developer Console 生成新 token
     2. 更新環境變數/密鑰管理系統
     3. 滾動重啟應用實例
     4. 驗證新憑證有效
     5. 撤銷舊憑證

2. **資料庫憑證**
   - **定期輪換頻率**: 每個月
   - **最小權限原則**: 僅授予必要的資料庫權限
   - **連接加密**: 強制使用 SSL/TLS (`DB_SSL_MODE=verify-full`)

3. **JWT Secret**
   - **定期輪換頻率**: 每季度
   - **影響**: 輪換後所有現有 token 失效，用戶需重新登入
   - **建議**: 使用雙 secret 過渡期策略

#### **密鑰管理最佳實踐**

**方案 A: AWS Secrets Manager**
```bash
# 從 AWS Secrets Manager 取得密鑰
export CHANNEL_SECRET=$(aws secretsmanager get-secret-value \
  --secret-id linebot/channel-secret \
  --query SecretString \
  --output text)

export CHANNEL_TOKEN=$(aws secretsmanager get-secret-value \
  --secret-id linebot/channel-token \
  --query SecretString \
  --output text)
```

**方案 B: HashiCorp Vault**
```bash
# 從 Vault 取得密鑰
export CHANNEL_SECRET=$(vault kv get -field=value secret/linebot/channel-secret)
export CHANNEL_TOKEN=$(vault kv get -field=value secret/linebot/channel-token)
```

**方案 C: Kubernetes Secrets**
```bash
# 創建 Kubernetes Secret
kubectl create secret generic linebot-secrets \
  --from-literal=channel-secret=$CHANNEL_SECRET \
  --from-literal=channel-token=$CHANNEL_TOKEN \
  --from-literal=db-password=$DB_PASSWORD
```

```yaml
# 在 Deployment 中使用
apiVersion: apps/v1
kind: Deployment
metadata:
  name: linebot-app
spec:
  template:
    spec:
      containers:
      - name: app
        env:
        - name: CHANNEL_SECRET
          valueFrom:
            secretKeyRef:
              name: linebot-secrets
              key: channel-secret
        - name: CHANNEL_TOKEN
          valueFrom:
            secretKeyRef:
              name: linebot-secrets
              key: channel-token
```

#### **配置安全檢查清單**

- [ ] ❌ 絕對不要將 `.env` 檔案提交到版本控制
- [ ] ✅ 使用 `.gitignore` 排除所有 `.env*` 檔案
- [ ] ✅ 使用密鑰管理系統（AWS/Vault/K8s Secrets）
- [ ] ✅ 生產環境強制使用 `DB_SSL_MODE=verify-full`
- [ ] ✅ 定期輪換所有敏感憑證
- [ ] ✅ 最小權限原則（資料庫用戶、IAM 角色）
- [ ] ✅ 啟用審計日誌記錄所有密鑰存取

---

## **3. 部署流程**

### **3.1 本地開發部署**

#### **方式 A: 直接運行 Go**
```bash
# 1. 設置環境變數
export CHANNEL_SECRET=test_secret
export CHANNEL_TOKEN=test_token

# 2. 運行應用
make dev
# 或
go run cmd/app/main.go
```

#### **方式 B: Docker Compose**
```bash
# 啟動所有服務（App + PostgreSQL + Redis）
make start

# 查看日誌
make logs

# 停止服務
make down
```

### **3.2 生產環境部署**

#### **方式 A: Docker**

**步驟 1: 構建 Docker 映像**
```bash
# 構建映像
make build IMAGE_TAG=v1.0.0

# 或直接使用 docker build
docker build -t linebot-bar:v1.0.0 .
```

**步驟 2: 運行容器**
```bash
# 使用環境變數檔案
docker run -d \
  --name linebot-app \
  --env-file .env.production \
  -p 8080:8080 \
  linebot-bar:v1.0.0

# 或從密鑰管理系統注入
docker run -d \
  --name linebot-app \
  -e CHANNEL_SECRET=$(vault kv get -field=value secret/linebot/channel-secret) \
  -e CHANNEL_TOKEN=$(vault kv get -field=value secret/linebot/channel-token) \
  -p 8080:8080 \
  linebot-bar:v1.0.0
```

#### **方式 B: Kubernetes**

**步驟 1: 創建 ConfigMap 和 Secret**
```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: linebot-config
data:
  GIN_MODE: "release"
  LOG_LEVEL: "info"
  PORT: "8080"
  DB_HOST: "postgres-service"
  DB_PORT: "5432"
  DB_NAME: "linebot_bar"
  REDIS_HOST: "redis-service"
  REDIS_PORT: "6379"
---
# secret.yaml (從 Vault 或外部系統生成)
apiVersion: v1
kind: Secret
metadata:
  name: linebot-secrets
type: Opaque
stringData:
  CHANNEL_SECRET: "your_channel_secret"
  CHANNEL_TOKEN: "your_channel_token"
  DB_USER: "linebot_user"
  DB_PASSWORD: "secure_password"
  REDIS_PASSWORD: "redis_password"
  JWT_SECRET: "jwt_secret"
```

**步驟 2: 創建 Deployment**
```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: linebot-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: linebot
  template:
    metadata:
      labels:
        app: linebot
    spec:
      containers:
      - name: app
        image: linebot-bar:v1.0.0
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: linebot-config
        - secretRef:
            name: linebot-secrets
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: linebot-service
spec:
  selector:
    app: linebot
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

**步驟 3: 部署到集群**
```bash
# 應用配置
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f deployment.yaml

# 檢查狀態
kubectl get pods -l app=linebot
kubectl logs -l app=linebot -f

# 取得服務 URL
kubectl get service linebot-service
```

#### **方式 C: 快速部署腳本**

使用專案提供的 `quick-deploy.sh`：

```bash
# 部署到遠端伺服器
./quick-deploy.sh

# 腳本會自動執行：
# 1. 檢查遠端伺服器連線
# 2. 同步程式碼到伺服器
# 3. 構建 Docker 映像
# 4. 停止舊容器
# 5. 啟動新容器
# 6. 驗證健康狀態
```

### **3.3 前端部署**

#### **開發環境**
```bash
# 安裝依賴
make fe-install

# 啟動開發伺服器
make fe-dev
# 訪問 http://localhost:5173
```

#### **生產環境**
```bash
# 構建生產版本
make fe-build

# 靜態檔案輸出到 frontend/dist/
# 部署到 Nginx/CDN
```

**Nginx 配置範例**:
```nginx
server {
    listen 80;
    server_name admin.company.com;

    root /var/www/linebot-admin/dist;
    index index.html;

    # SPA 路由支援
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 反向代理
    location /api {
        proxy_pass http://linebot-backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### **3.4 資料庫遷移**

#### **自動遷移（開發/測試環境）**
```bash
# 使用 GORM AutoMigrate
make migrate

# 或設置環境變數自動遷移
AUTO_MIGRATE=true go run cmd/app/main.go
```

#### **手動遷移（生產環境）**
```bash
# 1. 備份資料庫
./scripts/backup-db.sh

# 2. 執行遷移
make migrate

# 3. 驗證遷移
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "\dt"

# 4. 如需回滾
./scripts/restore-local-db.sh ~/linebot_backups/backup_20250108.sql
```

---

## **4. 監控和故障排除**

### **4.1 健康檢查端點**

系統提供以下健康檢查端點：

```bash
# 基本健康檢查
curl http://localhost:8080/health

# 回應範例
{
  "status": "healthy",
  "timestamp": "2025-01-08T10:30:00Z",
  "version": "3.1.0",
  "components": {
    "database": "healthy",
    "redis": "healthy",
    "linebot": "healthy"
  }
}
```

### **4.2 配置監控指標**

#### **關鍵指標**
1. **配置載入狀態** - 啟動時配置驗證成功/失敗
2. **模組初始化時間** - 各 FX 模組初始化耗時
3. **依賴可用性** - 資料庫、Redis、LINE API 連線狀態
4. **憑證到期監控** - 距離憑證到期的天數

#### **監控警報規則**
```yaml
alerts:
  - name: configuration_validation_failed
    condition: config_validation_errors > 0
    severity: critical
    message: "生產配置驗證失敗，系統可能無法正常啟動"

  - name: database_connection_lost
    condition: database_status != "healthy"
    severity: critical
    message: "資料庫連線中斷"

  - name: redis_connection_lost
    condition: redis_status != "healthy"
    severity: warning
    message: "Redis 連線中斷，會話狀態不持久化"

  - name: module_init_timeout
    condition: module_init_duration > 30s
    severity: warning
    message: "模組初始化超時，檢查依賴服務"
```

### **4.3 日誌管理**

#### **結構化日誌格式**
系統使用 Zap 輸出 JSON 格式日誌：

```json
{
  "level": "info",
  "ts": 1704700800.123456,
  "caller": "service/linebot.go:69",
  "msg": "Input received",
  "input": "Hello",
  "userID": "U1234567890"
}
```

#### **日誌級別**
- **debug**: 詳細的調試資訊（僅開發環境）
- **info**: 一般操作資訊
- **warn**: 警告訊息（不影響正常運行）
- **error**: 錯誤訊息（需要關注）

#### **日誌查詢範例**
```bash
# 查詢特定用戶的日誌
cat app.log | jq 'select(.userID == "U1234567890")'

# 查詢錯誤日誌
cat app.log | jq 'select(.level == "error")'

# 統計各級別日誌數量
cat app.log | jq -r '.level' | sort | uniq -c
```

### **4.4 常見問題排查**

#### **問題 1: 啟動失敗 - 配置驗證錯誤**

**錯誤訊息**:
```
配置驗證失敗: [CHANNEL_SECRET]: 必填環境變數
```

**排查步驟**:
```bash
# 1. 檢查環境變數是否設置
env | grep CHANNEL

# 2. 檢查拼寫錯誤
# 正確: CHANNEL_SECRET
# 錯誤: CHANNEL_SECRE (少一個 T)

# 3. 檢查 .env 檔案是否被讀取
cat .env | grep CHANNEL

# 4. 手動設置環境變數測試
export CHANNEL_SECRET=test_secret
export CHANNEL_TOKEN=test_token
go run cmd/app/main.go
```

#### **問題 2: FX 模組初始化失敗**

**錯誤訊息**:
```
[Fx] ERROR		Failed to initialize	fx.Option
```

**排查步驟**:
```bash
# 1. 啟用 FX 詳細日誌
export FX_LOG_LEVEL=debug
go run cmd/app/main.go

# 2. 檢查依賴順序
# 確認 cmd/app/main.go 中的模組順序

# 3. 檢查缺失的依賴
# 錯誤通常會指出缺少哪個類型
```

#### **問題 3: 資料庫連接失敗**

**錯誤訊息**:
```
database connection timeout
```

**排查步驟**:
```bash
# 1. 檢查資料庫服務是否運行
docker ps | grep postgres
# 或
pg_isready -h localhost -p 5432

# 2. 測試資料庫連線
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT 1;"

# 3. 檢查防火牆規則
telnet $DB_HOST $DB_PORT

# 4. 檢查 SSL 模式設置
# 如果資料庫不支援 SSL，設置 DB_SSL_MODE=disable
```

#### **問題 4: LINE Bot 無回應**

**排查步驟**:
```bash
# 1. 檢查 Webhook URL 設置
# LINE Developers Console → Messaging API → Webhook URL
# 應該是: https://your-domain.com/callback

# 2. 驗證 CHANNEL_SECRET 和 CHANNEL_TOKEN
# 確認與 LINE Developer Console 一致

# 3. 檢查伺服器是否接收到請求
tail -f app.log | grep "/callback"

# 4. 測試 Webhook 簽章驗證
curl -X POST http://localhost:8080/callback \
  -H "Content-Type: application/json" \
  -H "X-Line-Signature: test" \
  -d '{"events":[]}'
```

#### **問題 5: Redis 連線失敗（會話狀態丟失）**

**錯誤訊息**:
```
Redis connection failed, fallback to in-memory storage
```

**影響**:
- 用戶註冊流程的會話狀態不會持久化
- 應用重啟後會話狀態丟失

**解決方案**:
```bash
# 方案 A: 啟動 Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine

# 方案 B: 設置 Redis 環境變數
export REDIS_HOST=localhost
export REDIS_PORT=6379

# 方案 C: 接受 fallback（開發環境可接受）
# 系統會自動使用 in-memory storage
```

### **4.5 性能監控**

#### **啟動時間監控**
```go
// 在 cmd/app/main.go 中添加
func measureStartupTime() fx.Option {
    return fx.Invoke(func(lc fx.Lifecycle) {
        start := time.Now()
        lc.Append(fx.Hook{
            OnStart: func(ctx context.Context) error {
                duration := time.Since(start)
                log.Printf("應用啟動時間: %v", duration)
                return nil
            },
        })
    })
}
```

#### **模組初始化時間追蹤**
```bash
# 啟用詳細日誌
export FX_LOG_LEVEL=debug

# 查看各模組初始化時間
go run cmd/app/main.go 2>&1 | grep "PROVIDE"
```

#### **建議基準**
- 總啟動時間: < 5 秒
- 資料庫連線: < 1 秒
- Redis 連線: < 500ms
- FX 模組初始化: < 2 秒

### **4.6 調試工具**

#### **配置模板生成**
```bash
# 生成包含所有配置項的模板
go run tools/config-generator.go > .env.template

# 使用模板
cp .env.template .env
vim .env  # 填寫實際值
```

#### **配置驗證工具**
```bash
# 僅驗證配置，不啟動伺服器
go run cmd/app/main.go --validate-config-only
```

#### **依賴圖生成**
```bash
# 生成 FX 依賴關係圖
go run tools/fx-graph.go > dependency-graph.dot
dot -Tpng dependency-graph.dot -o dependency-graph.png
```

---

## **附錄**

### **A. 部署檢查清單**

**部署前檢查**:
- [ ] ✅ 所有必填環境變數已設置
- [ ] ✅ 資料庫遷移已執行
- [ ] ✅ 資料庫已備份（生產環境）
- [ ] ✅ 密鑰已從密鑰管理系統取得
- [ ] ✅ SSL 憑證有效且未過期
- [ ] ✅ 健康檢查端點可訪問
- [ ] ✅ 日誌系統正常運作
- [ ] ✅ 監控警報已配置

**部署後驗證**:
- [ ] ✅ 健康檢查返回 healthy
- [ ] ✅ LINE Bot 回應正常
- [ ] ✅ 管理後台可登入
- [ ] ✅ 資料庫連線正常
- [ ] ✅ Redis 連線正常（如有配置）
- [ ] ✅ 日誌正常輸出
- [ ] ✅ 無錯誤警報

### **B. 回滾計畫**

**Docker 部署回滾**:
```bash
# 1. 停止當前版本
docker stop linebot-app

# 2. 啟動前一版本
docker run -d \
  --name linebot-app \
  --env-file .env.production \
  -p 8080:8080 \
  linebot-bar:v0.9.0  # 前一個穩定版本

# 3. 驗證健康狀態
curl http://localhost:8080/health
```

**Kubernetes 回滾**:
```bash
# 查看部署歷史
kubectl rollout history deployment/linebot-app

# 回滾到前一版本
kubectl rollout undo deployment/linebot-app

# 回滾到特定版本
kubectl rollout undo deployment/linebot-app --to-revision=2
```

### **C. 相關文檔**

- [PRD.md](../product/PRD.md) - 產品需求文件
- [SYSTEM_DESIGN.md](../architecture/SYSTEM_DESIGN.md) - 系統架構設計
- [CLAUDE.md](../CLAUDE.md) - 開發指引
- [README.md](../README.md) - 快速上手指南

---

**文件版本**: 3.1
**最後更新**: 2025-01-08
**維護者**: DevOps Team
