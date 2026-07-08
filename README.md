# AI Gateway

统一 API 网关，用于管理和代理 AI 模型提供商（如 OpenAI）的请求。提供租户管理、API Key 认证、基于 Scope 的权限控制、用量记录与查询等功能。

---

## 目录

- [架构说明](#架构说明)
- [快速开始](#快速开始)
- [运行步骤](#运行步骤)
- [API 参考](#api-参考)
- [配置](#配置)
- [开发指南](#开发指南)
- [设计决策](#设计决策)
- [已知限制](#已知限制)

---

## 架构说明

```
┌──────────────┐      ┌──────────────────────────────────────┐
│   Client     │─────▶│           AI Gateway                 │
│ (curl/App)   │      │                                      │
└──────────────┘      │  ┌──────────┐  ┌──────────────────┐  │
                      │  │ Router   │──▶  Middleware       │  │
                      │  │ (chi)    │  │  - RequestID     │  │
                      │  │          │  │  - Logging       │  │
                      │  │          │  │  - CORS          │  │
                      │  │          │  │  - Recoverer     │  │
                      │  │          │  │  - Authenticate  │  │
                      │  └────┬─────┘  └────────┬─────────┘  │
                      │       │                 │             │
                      │       ▼                 ▼             │
                      │  ┌──────────┐  ┌──────────────────┐  │
                      │  │ Handler  │  │  Auth Service    │  │
                      │  │ - health │  │  - Key Validate │  │
                      │  │ - tenant │  │  - Scope Match  │  │
                      │  │ - apikey │  └──────────────────┘  │
                      │  │ - proxy  │                         │
                      │  │ - usage  │  ┌──────────────────┐  │
                      │  └────┬─────┘  │  Usage Recorder  │  │
                      │       │        │  - Async Batch   │  │
                      │       ▼        │  - Flush on Timer│  │
                      │  ┌──────────┐  └──────────────────┘  │
                      │  │  Store   │                         │
                      │  │ (SQLite) │  ┌──────────────────┐  │
                      │  └──────────┘  │  Provider        │  │
                      │                │  (Mock/OpenAI)   │  │
                      │                └────────┬─────────┘  │
                      └─────────────────────────┼────────────┘
                                                │
                                                ▼
                                        ┌──────────────┐
                                        │   Upstream   │
                                        │  (AI Model)  │
                                        └──────────────┘
```

### 组件分层

| 层 | 包 | 职责 |
|---|---|---|
| **入口** | `cmd/server` | HTTP 服务启动、优雅关闭 |
| **路由** | `internal/router` | 路由注册、中间件组装 |
| **中间件** | `internal/middleware` | 认证、日志、CORS |
| **处理器** | `internal/handler` | HTTP 请求处理、参数校验、响应序列化 |
| **认证** | `internal/auth` | API Key 生成/验证、Scope 匹配 |
| **代理** | `internal/proxy` | AI 提供商客户端（通过 Provider 接口抽象） |
| **存储** | `internal/store` | SQLite 数据访问层（Tenant/ApiKey/Usage） |
| **用量** | `internal/usage` | Token 用量异步记录与批量写入 |
| **配置** | `internal/config` | 环境变量配置加载 |
| **模型** | `internal/model` | 数据模型定义 |

### 请求生命周期

```
Client ──POST /v1/chat/completions──▶
  1. Router 匹配路由
  2. Authenticate 中间件解析 Bearer Token
  3. Auth Service 验证 Key（前缀查找 → 哈希比对 → Scope 校验）
  4. Proxy Handler 调用 Provider
  5. Provider 转发请求到上游 AI 服务
  6. 响应回写 Client
  7. Usage Recorder 异步记录 Token 用量
```

---

## 快速开始

### 前置要求

- Go 1.23+

### 项目结构

```
ai-gateway/
├── api/                  # OpenAPI 规范
├── cmd/server/           # 入口
├── internal/
│   ├── auth/             # 认证逻辑
│   ├── config/           # 配置加载
│   ├── handler/          # HTTP Handler
│   ├── middleware/       # HTTP 中间件
│   ├── model/            # 数据模型
│   ├── proxy/            # 提供商代理
│   ├── router/           # 路由组装
│   ├── store/            # 数据存储
│   └── usage/            # 用量记录
├── scripts/              # 工具脚本
├── docs/                 # 文档
└── docker-compose.yml    # 开发环境编排
```

### 常用命令

```bash
make run        # 本地启动（需先启动 mock-provider）
make test       # 运行所有测试（51 个测试用例全部通过）
make build      # 编译二进制
make lint       # 代码检查
make deps       # 整理依赖
make dev        # 开发模式（热重载）
make docker-up  # Docker 启动完整环境
make verify     # 集成验证
```

### 测试覆盖

```
internal/auth       93.2%
internal/config    100.0%  ✅
internal/handler    94.6%
internal/middleware 100.0%  ✅
internal/proxy      81.2%
internal/router     100.0%  ✅
internal/store      72.8%
internal/usage      92.9%
```

测试策略详见 [`docs/03-testing-strategy.md`](docs/03-testing-strategy.md)。

### 前端页面测试

| 概览 | 租户管理 | API Key 管理 | 用量监控 |
|------|---------|-------------|---------|
| ![概览](docs/10-概览.png) | ![租户管理](docs/11-租户管理.png) | ![API Key管理](docs/12-API%20KEY管理.png) | ![用量管理](docs/13-用量管理.png) |

管理面板是一个嵌入 Go binary 的纯客户端 SPA（`/dashboard`），测试分三个层面：

#### 1. Go 侧 HTML 结构验证（已有）

`internal/handler/handler_test.go` 中的 `TestDashboard` 测试验证：
- HTTP 200 和正确的 Content-Type（`text/html; charset=utf-8`）
- HTML 关键结构元素（导航、页面容器、模态框、Toast 通知）
- JS 函数定义（`loadOverview`, `loadTenants`, `loadKeys`, `loadUsageSummary` 等）
- API_BASE 配置、复制 Key 功能等交互特性

```bash
make test  # 全部测试，或
go test ./internal/handler/... -v -count=1 -run TestDashboard
```

#### 2. 手动浏览器端到端验证

```bash
# 终端 1：启动 mock provider
go run ./tools/mock-server/

# 终端 2：启动网关
make run
```

> ⚠️ `/dashboard` 页面**嵌入在 Go 二进制中**，不能直接双击 HTML 文件或拖到浏览器打开。必须先启动服务（如上两步），然后通过 **http://localhost:8080/dashboard** 访问。

打开浏览器访问 **http://localhost:8080/dashboard**，按以下步骤验证：

**步骤 1 — 概览页**
- 页面自动加载，左侧显示 **AI Gateway** 标题和 4 个导航链接（概览、租户管理、API Key 管理、用量监控）
- 中部显示 5 张统计卡片：租户总数、API Key 总数、活跃 Key、总请求次数、总 Token 消耗
- 首次打开时各卡片显示 `-`（尚无数据），底部「最近用量记录」显示「暂无用量记录」

**步骤 2 — 创建租户**
- 点击左侧「🏢 租户管理」→ 页面显示「租户管理」标题和「+ 新建租户」按钮
- 点击「+ 新建租户」→ 弹出模态框
- 输入名称（例如 `test-org`）→ 点击「保存」
- 模态框关闭，表格中显示新记录（ID、名称、状态「活跃」、创建时间）
- 右侧弹出绿色 Toast 提示「租户已创建」

**步骤 3 — 编辑租户**
- 点击租户行的「编辑」按钮 → 模态框标题变为「编辑租户」，名称输入框已填入原值
- 修改名称 → 点击「更新」→ 表格内容更新，Toast 提示「租户已更新」

**步骤 4 — 删除租户**
- 点击租户行的「删除」按钮 → 浏览器弹出确认对话框 `确定删除此租户？`
- 点击「确定」→ 表格中该行消失，Toast 提示「租户已删除」

**步骤 5 — 创建 API Key**
- 点击左侧「🔑 API Key 管理」→ 页面自动加载租户列表到下拉框
- 点击「+ 新建 Key」→ 弹出模态框
- 选择所属租户、输入名称（例如 `dev-key`）、权限模型默认 `gpt-4,gpt-3.5-turbo`
- 可选设置过期时间 → 点击「保存」
- 模态框内显示绿色卡片，包含完整 Key 值（例如 `sk-a1b2c3d4...`）和「📋 复制 Key」按钮
- 点击「📋 复制 Key」→ Key 复制到剪贴板，Toast 提示「已复制到剪贴板」
- 关闭模态框，Key 表格中出现新记录（前缀、名称、租户、权限、状态、过期时间）

**步骤 6 — 用量监控**
- 先通过后端 API 或 curl 发起一次代理请求，产生用量记录：
  ```bash
  curl -X POST http://localhost:8080/v1/chat/completions \
    -H "Authorization: Bearer <上一步创建的完整Key>" \
    -H "Content-Type: application/json" \
    -d '{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}'
  ```
- 点击左侧「📈 用量监控」→ 页面显示日期范围控件（默认最近 7 天）
- 选择租户和日期范围 → 点击「查询」
- 上方 4 张卡片显示：请求次数、Prompt Tokens、Completion Tokens、总 Tokens
- 下方表格显示详细用量记录（模型、各 Token 数、时间）

**步骤 7 — 侧边栏导航**
- 点击各个导航链接，页面正确切换，URL 不变（纯前端 SPA）
- 切回概览页，统计卡片数据已更新（反映刚才创建的租户和 Key 数量）

#### 3. 集成验证脚本

项目已提供端到端 CLI 验证脚本，涵盖后端 API 完整流程：

```bash
# 启动服务后执行
make verify
# 或
./scripts/verify.sh
```

> 如需自动化前端 UI 测试，可考虑引入 [Playwright](https://playwright.dev/) 或 [Cypress](https://www.cypress.io/) 对 `/dashboard` 页面编写浏览器测试脚本。

---

## 运行步骤

### 纯 Go 方式启动（无需 Docker）

本机已安装 Go，不需要 Docker 即可完整运行：

```bash
# 1. 复制环境变量
cp .env.example .env

# 2. 在终端 1 — 启动 mock provider（Go 编写，纯 HTTP 服务）
go run ./tools/mock-server/

# 3. 在终端 2 — 启动网关
make run

# 4. 验证健康检查
curl http://localhost:8080/health
```

> 如果只有一台终端，可用 `start` 命令在后台启动 mock provider：
> ```bash
> start /B go run ./tools/mock-server/
> make run
> ```

### 方式二：Docker Compose（可选，需安装 Docker）

```bash
# 一键启动
docker compose up -d
```

### 端到端验证

以下 curl 命令展示完整的租户创建 → API Key 生成 → 代理调用 → 用量查询流程。

#### 1. 健康检查

```bash
curl http://localhost:8080/health
```

```json
{"status":"ok","version":"0.1.0"}
```

#### 2. 创建租户

```bash
curl -X POST http://localhost:8080/api/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "my-org"}'
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "my-org",
  "is_active": true,
  "created_at": "2026-07-07T10:00:00Z",
  "updated_at": "2026-07-07T10:00:00Z"
}
```

#### 3. 创建 API Key

```bash
# 替换 {tenant_id} 为上一步返回的 id
curl -X POST http://localhost:8080/api/keys \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "550e8400-e29b-41d4-a716-446655440000", "name": "dev-key", "scopes": "gpt-4,gpt-3.5-turbo"}'
```

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "dev-key",
  "key_prefix": "sk-a1b2c3d4",
  "full_key": "sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "scopes": "gpt-4,gpt-3.5-turbo",
  "is_active": true,
  "created_at": "2026-07-07T10:00:00Z",
  "updated_at": "2026-07-07T10:00:00Z"
}
```

> ⚠️ `full_key` 仅在创建时返回一次，后续无法再次获取。

#### 4. 调用代理 API

```bash
# 聊天补全
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}'

# 模型列表
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
```

#### 5. 查询用量

```bash
# 查询所有用量记录
curl http://localhost:8080/api/usage

# 按租户和模型过滤
curl "http://localhost:8080/api/usage?tenant_id={tenant_id}&model=gpt-4"

# 总用量汇总
curl "http://localhost:8080/api/usage/summary?tenant_id={tenant_id}"

# 按时间范围过滤
curl "http://localhost:8080/api/usage?start=2026-07-01T00:00:00Z&end=2026-07-31T23:59:59Z"

# 分页
curl "http://localhost:8080/api/usage?page=1&page_size=20"
```

#### 6. 管理租户和 Key

```bash
# 列出租户
curl http://localhost:8080/api/tenants

# 获取租户详情
curl http://localhost:8080/api/tenants/{id}

# 更新租户
curl -X PUT http://localhost:8080/api/tenants/{id} \
  -H "Content-Type: application/json" \
  -d '{"name": "my-org-renamed", "is_active": true}'

# 删除租户
curl -X DELETE http://localhost:8080/api/tenants/{id}

# 列出所有 API Key
curl http://localhost:8080/api/keys

# 获取 Key 详情
curl http://localhost:8080/api/keys/{id}

# 更新 Key（名称/权限/状态/过期时间）
curl -X PUT http://localhost:8080/api/keys/{id} \
  -H "Content-Type: application/json" \
  -d '{"name": "prod-key", "scopes": "gpt-4,*", "is_active": true}'

# 删除 Key
curl -X DELETE http://localhost:8080/api/keys/{id}
```

---

## API 参考

完整 OpenAPI 规范见 [`api/openapi.yaml`](api/openapi.yaml)。

### 健康检查

```
GET /health
```

### 代理 API（需 Bearer Token 认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/chat/completions` | 聊天补全 |
| GET | `/v1/models` | 模型列表 |

### 管理 API（内部使用，无独立认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/tenants` | 创建租户 |
| GET | `/api/tenants` | 列出租户 |
| GET | `/api/tenants/{id}` | 获取租户详情 |
| PUT | `/api/tenants/{id}` | 更新租户 |
| DELETE | `/api/tenants/{id}` | 删除租户 |
| POST | `/api/keys` | 创建 API Key |
| GET | `/api/keys` | 列出 API Key |
| GET | `/api/keys/{id}` | 获取 Key 详情 |
| PUT | `/api/keys/{id}` | 更新 Key |
| DELETE | `/api/keys/{id}` | 删除 Key |
| GET | `/api/usage` | 查询用量记录 |
| GET | `/api/usage/summary` | 用量汇总 |

### 错误响应格式

所有错误响应遵循统一格式：

```json
{
  "error": {
    "message": "invalid request body",
    "type": "request_error",
    "code": 400
  }
}
```

| 状态码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未认证或认证失败 |
| 403 | 权限不足（Scope 不匹配） |
| 404 | 资源不存在 |
| 501 | 功能未实现 |
| 502 | 上游服务错误 |

---

## 配置

所有配置通过环境变量注入，详见 [`.env.example`](.env.example)。

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DATABASE_PATH` | `./data/gateway.db` | SQLite 数据库路径 |
| `PORT` | `8080` | 服务端口 |
| `LOG_LEVEL` | `info` | 日志级别 |
| `MOCK_PROVIDER_BASE_URL` | `http://localhost:9090` | Mock Provider 地址 |
| `API_KEY_PREFIX_LENGTH` | `8` | Key 前缀长度 |
| `API_KEY_HASH_ALGORITHM` | `sha256` | 哈希算法 |
| `API_KEY_SECRET_LENGTH` | `32` | Key 密钥长度（字节） |
| `USAGE_FLUSH_INTERVAL` | `5s` | 用量批量写入间隔 |
| `USAGE_FLUSH_BATCH_SIZE` | `100` | 每批最大条数 |
| `CORS_ALLOWED_ORIGINS` | `*` | CORS 允许的域名 |

---

## 开发指南

开发流程详见 [`docs/00-development-workflow.md`](docs/00-development-workflow.md)。

### 快速迭代

```bash
# 启动开发模式（需安装 air，自动重载）
make dev

# 或手动启动
go run ./cmd/server/
```

### 运行测试

```bash
# 全部测试
make test

# 单一包
go test ./internal/handler/... -v -count=1

# 带覆盖率
go test ./... -cover -count=1
```

### 代码检查

```bash
make lint
```

---

## 设计决策

### 为什么使用 SQLite 而非 PostgreSQL？

| 因素 | SQLite | PostgreSQL |
|------|--------|------------|
| 运维 | 零运维，单文件 | 需要独立数据库服务 |
| 部署 | 单二进制，无需额外服务 | 依赖外部数据库 |
| 并发 | 单写者模式，适合 < 100 QPS | 高并发 |
| 迁移 | 数据文件可备份/迁移 | 需 pg_dump |

**决策**：MVP 阶段使用 SQLite。存储层已抽象为 `Store` 接口，后续可平滑替换为 PostgreSQL。

### 为什么使用 chi 路由？

- **轻量**：零外部依赖，核心逻辑与标准库 `net/http` 完全兼容
- **兼容**：Handler 签名即 `http.HandlerFunc`，不绑定框架
- **链式中间件**：天然支持 `Use` 和 `Group`，路由组织清晰

### API Key 安全设计

```
生成：sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
       │  └──────── 前缀 ────────┘│
       │  存储 → 前缀索引(查找)    │
       │                          │
       └── sk- 前缀 ──────────────┘
           完整 Key → SHA256 哈希 → 存储哈希值(验证)
```

- 完整 Key 仅创建时返回一次，数据库不存储明文
- 使用 `key_prefix`（前 N 位）做前缀索引，加速查找
- 哈希比对验证 Key 真实性
- 支持过期时间、禁用状态、Scope 细粒度权限控制

### Provider 接口设计

```go
type Provider interface {
    ChatCompletion(body io.Reader) (*http.Response, error)
    ListModels() (*http.Response, error)
}
```

通过接口抽象 AI 提供商，当前绑定 Mock Provider（nginx）。后续接入真实 API（OpenAI、Claude 等）只需实现该接口，无需修改 handler 和 router 层代码。

### 为什么没有项目管理 API 认证？

- MVP 阶段管理 API 仅限内部网络使用
- 后续可通过 `Master API Key` 或独立管理认证中间件补充

---

## 已知限制

### MVP 阶段限制

| 限制 | 说明 | 影响 |
|------|------|------|
| **单实例部署** | 不支持水平扩展，SQLite 单写者模式 | 高并发场景受限 |
| **无 Rate Limiting** | 未做频率限制 | 恶意调用无防护 |
| **Mock Provider** | 当前仅集成 Mock Provider | 无法调用真实 AI 服务 |
| **无 TLS** | 演示环境使用 HTTP | 生产需配置 HTTPS |
| **无监控** | 仅基础日志，无 metrics、tracing | 无法观测服务状态 |
| **管理 API 无独立认证** | `/api/*` 路由未加密 | 需在内网使用 |
| **未实现流式响应** | 暂不支持 SSE/Stream | 大 Token 响应延迟高 |

### 可扩展方向

- **存储层**：Store 接口已抽象，可替换为 PostgreSQL（需修改 `store.NewDB` 及 driver import）
- **Provider**：实现 `OpenAIProvider` / `ClaudeProvider`，配置真实 API Key 即可
- **Scope 体系**：当前为逗号分隔字符串，可扩展为结构化 RBAC
- **用量统计**：当前原始记录查询，可扩展为小时/日/月聚合报表
- **流式响应**：Handler 层已预留 stream 参数，需在 Provider 层实现 SSE 转发

---

## 许可

MIT License