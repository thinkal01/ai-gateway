# AI Gateway MVP — 脚手架与核心代码文档

> 版本: 1.0
> 本文档记录阶段三~五（开发准备 → 接口先行 → 模块实现）的全部产出，包括项目结构、每个文件的职责、关键设计决策。

---

## 1. 最终文件结构

```
ai-gateway/
├── cmd/
│   └── server/
│       └── main.go                  # 应用入口
├── internal/
│   ├── config/
│   │   └── config.go                # 环境变量配置加载
│   ├── model/
│   │   ├── tenant.go                # 租户模型 + 请求/响应 DTO
│   │   ├── apikey.go                # API Key 模型 + 请求/响应 DTO
│   │   └── usage.go                 # 用量记录模型
│   ├── store/
│   │   ├── db.go                    # SQLite 初始化 + 自动迁移
│   │   ├── tenant_store.go          # 租户 CRUD 接口与实现
│   │   ├── apikey_store.go          # API Key CRUD 接口与实现
│   │   └── usage_store.go           # 用量记录存储接口与实现
│   ├── auth/
│   │   ├── scope.go                 # 权限类型定义与匹配逻辑
│   │   └── service.go               # Key 生成、认证服务
│   ├── proxy/
│   │   ├── provider.go              # Provider 抽象接口
│   │   └── mock.go                  # Mock Provider 实现
│   ├── usage/
│   │   └── recorder.go              # 异步批量用量记录器
│   ├── middleware/
│   │   └── auth.go                  # 认证/日志/CORS 中间件
│   ├── handler/
│   │   ├── response.go              # writeJSON / writeError 工具
│   │   ├── health.go                # GET /health
│   │   ├── tenant.go                # 管理 API: /api/tenants CRUD
│   │   ├── apikey.go                # 管理 API: /api/keys CRUD
│   │   └── notimplemented.go        # 占位 501 handler
│   └── router/
│       └── router.go                # chi 路由注册
├── scripts/
│   └── mock-provider.conf           # nginx 模拟上游配置
├── api/                             # OpenAPI 规范（暂未创建）
├── web/dashboard/                   # 管理面板（暂未创建）
├── test/                            # 集成测试（暂未创建）
├── .gitignore
├── .env.example
├── Makefile
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── go.sum
```

---

## 2. 各模块详细说明

### 2.1 config — 配置加载

**文件:** `internal/config/config.go`

- 从环境变量加载全部配置，无配置文件依赖
- 扁平结构，直接定义字段而非嵌套结构
- 提供 `getEnv`、`getEnvInt`、`getEnvDuration` 三个辅助函数

**配置项:**

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `DATABASE_PATH` | `./data/gateway.db` | SQLite 文件路径 |
| `PORT` | `8080` | HTTP 监听端口 |
| `LOG_LEVEL` | `info` | 日志级别 |
| `VERSION` | `0.1.0` | 应用版本 |
| `MOCK_PROVIDER_BASE_URL` | `http://localhost:9090` | Mock Provider 地址 |
| `API_KEY_PREFIX_LENGTH` | `8` | Key 前缀字节数（base62 编码） |
| `API_KEY_HASH_ALGORITHM` | `sha256` | Key 哈希算法 |
| `API_KEY_SECRET_LENGTH` | `32` | Key 密钥部分字节数（base62 编码） |
| `USAGE_FLUSH_INTERVAL` | `5s` | 用量记录刷盘间隔 |
| `USAGE_FLUSH_BATCH_SIZE` | `100` | 批量刷盘阈值 |
| `CORS_ALLOWED_ORIGINS` | `*` | CORS 允许来源 |

---

### 2.2 model — 数据模型

**三个文件：`tenant.go`、`apikey.go`、`usage.go`**

每个文件包含三类结构体：

1. **持久化模型** — 对应数据库表字段，包含 `ID`、`CreatedAt`、`UpdatedAt`
2. **创建请求** (`CreateXxxRequest`) — POST body 定义
3. **更新请求** (`UpdateXxxRequest`) — PATCH body 定义，字段均为指针类型
4. **输出响应** (`XxxResponse`) — API 返回结构

**ApiKey 特殊处理（相关代码在 `auth/service.go`）：**

- 生成时返回完整 `sk-{prefix}{secret}`，此值仅创建时可见
- 数据库中只存储 `key_prefix` 和 `key_hash`
- `key_hash = sha256(full_key)`，不可逆

---

### 2.3 store — 数据持久层

**四个文件：**

| 文件 | 接口 | 实现 | 表名 |
|------|------|------|------|
| `db.go` | — | `NewDB()` | 自动建 3 表 + 索引 |
| `tenant_store.go` | `TenantStore` | `tenantStore` | `tenants` |
| `apikey_store.go` | `ApiKeyStore` | `apiKeyStore` | `api_keys` |
| `usage_store.go` | `UsageStore` | `usageStore` | `usage_records` |

**设计要点：**

- 使用 `modernc.org/sqlite`（纯 Go 实现，无需 CGO）
- 日期字段以 `RFC3339` 字符串存储（SQLite 无原生时间类型）
- `api_keys` 表中 `scopes` 存储为逗号分隔字符串（如 `"chat:write,model:list"`）
- `tenants` 在删除时级联删除关联的 `api_keys`（外键约束）
- 所有 store 接口应返回 `nil, nil` 而非 `nil, sql.ErrNoRows`

---

### 2.4 auth — 认证模块

| 文件 | 内容 |
|------|------|
| `scope.go` | `Scope` 类型常量 + `ScopeMatch(required, have string) bool` |
| `service.go` | `GenerateKey()` + `AuthService.Authenticate()` |

**Scope 匹配规则：**

```
精确匹配: "chat:write" == "chat:write"         → true
通配匹配: "chat:*" matches "chat:write"        → true
前缀匹配: "chat:" matches "chat:write"          → true
拒绝匹配: "chat:write" matches "-chat:write"    → false
范围包含: "*" matches "chat:write"              → true
```

**Key 格式：`sk-{prefix18}{secret22}`（base62，共 40 位）**

- prefix=8 字节 → base62 编码约 11 字符
- secret=16 字节 → base62 编码约 22 字符
- 前缀用于快速查找，哈希用于验证

---

### 2.5 proxy — 上游代理

| 文件 | 内容 |
|------|------|
| `provider.go` | `Provider` 接口定义 |
| `mock.go` | `MockProvider` 实现 |

`Provider` 接口：

```go
type Provider interface {
    GetName() string
    ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error)
    ListModels(ctx context.Context) (*model.ModelListResponse, error)
}
```

**MockProvider** 返回固定响应，用于本地开发和测试，无需真实 API Key。

---

### 2.6 usage — 用量记录

**文件:** `usage/recorder.go`

- 提供 `Recorder` 结构体，内部维护一个带锁的 `[]*model.UsageRecord` 缓冲
- 提供 `Record()` 方法将记录加入缓冲
- 后台 goroutine 定时（`flushInterval`）或在缓冲满（`batchSize`）时批量写入 store
- `Shutdown()` 确保关闭前完成最后一次 flush

---

### 2.7 middleware — HTTP 中间件

| 中间件 | 说明 |
|--------|------|
| `Logging` | 记录请求方法、路径、状态码、耗时 |
| `CORS(origins)` | 添加跨域头 |
| `Authenticate(authService)` | 从 `Authorization: Bearer sk-...` 提取 Key 并验证 |

**认证流程：**

```
请求头提取前缀 → 按 prefix 查找 → 不存在则 401
    ↓ 存在
sha256(full_key) 比对 → 不匹配则 401
    ↓ 匹配
校验 is_active && !expired → 无效则 403
    ↓ 有效
检查 scope 权限 → 不满足则 403
    ↓ 满足
将 tenant_id 注入 context → 放行
```

---

### 2.8 handler — HTTP 处理层

| 文件 | 路由 | 说明 |
|------|------|------|
| `response.go` | — | 统一的 `writeJSON` / `writeError` 工具 |
| `health.go` | `GET /health` | 返回 `{"status":"ok","version":"..."}` |
| `tenant.go` | `/api/tenants` | 完整的 CRUD 五方法 |
| `apikey.go` | `/api/keys` | 完整的 CRUD 五方法 |
| `notimplemented.go` | `POST /v1/chat/completions` 等 | 501 占位 |

**RESTful 设计：**

```
POST   /api/tenants          → 创建租户（201）
GET    /api/tenants          → 列出全部（200）
GET    /api/tenants/{id}     → 查询单个（200/404）
PUT    /api/tenants/{id}     → 更新（200）
DELETE /api/tenants/{id}     → 删除（204）

POST   /api/keys             → 创建 Key（201，返回完整 Key）
GET    /api/keys             → 列出全部（200）
GET    /api/keys/{id}        → 查询单个（200/404）
PUT    /api/keys/{id}        → 更新（200）
DELETE /api/keys/{id}        → 删除（204）
```

---

### 2.9 router — 路由注册

**文件:** `router/router.go`

- 外层：`GET /health`（无认证）
- 认证组：`POST /v1/chat/completions`、`GET /v1/models`（均返回 501）
- 管理组：`/api/tenants/*`、`/api/keys/*`

---

### 2.10 cmd/server/main — 入口

```go
main()
  ├── config.Load()           // 加载配置
  ├── store.NewDB()           // 初始化 SQLite
  ├── router.New(cfg, db)     // 组装路由
  └── http.ListenAndServe()   // 启动 + 优雅关闭
```

---

## 3. 关于 "阶段拆分" 的说明

在实际开发过程中，阶段三（开发准备）、阶段四（接口先行）、阶段五（模块实现）一次性完成，原因如下：

| 理由 | 说明 |
|------|------|
| 项目规模较小 | Go 项目约 20 个文件，定义接口后立即实现更高效 |
| 接口稳定 | 需求分析和设计文档已覆盖了 90% 的接口边界，无需先写空接口再填充 |
| 减少上下文切换 | 一次性完成可保持对整体架构的连续记忆 |

如果后续功能膨胀，再拆分为独立的接口定义 → 实现循环。

---

## 4. 未实现项（后续阶段补齐）

- [ ] 认证中间件的 scope 校验（当前仅认证，未校验 scope）
- [ ] 代理请求转发（`/v1/chat/completions` 的 Provider 调用）
- [ ] 用量记录的 Recorder 启动（`Start()` 尚未接入 main）
- [ ] 所有测试文件
- [ ] OpenAPI 规范文档
- [ ] 管理 Web 面板

---

## 文档版本记录

| 版本 | 日期 | 变更内容 |
|------|------|---------|
| 1.0 | 2026-07-07 | 初始版本 |