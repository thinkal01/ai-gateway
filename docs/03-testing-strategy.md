# AI Gateway MVP 测试策略

> 版本: 1.0
> 本文档定义测试分层、命名规范、覆盖范围及运行方式。

---

## 1. 测试分层

```
┌──────────────────────────────────────────────┐
│              HTTP 集成测试 (handler)          │
│   httptest.NewServer + 完整请求/响应验证      │
├──────────────────────────────────────────────┤
│          中间件测试 (middleware)               │
│   httptest.NewRecorder + 模拟 Handler         │
├──────────────────────────────────────────────┤
│          Service 单元测试 (auth)              │
│   模拟 Store 接口 + 纯逻辑验证                │
├──────────────────────────────────────────────┤
│          Store 集成测试 (store)               │
│   :memory: SQLite + CRUD 全链路               │
├──────────────────────────────────────────────┤
│          基础单元测试 (proxy, scope)          │
│   纯函数测试，无外部依赖                      │
└──────────────────────────────────────────────┘
```

---

## 2. 测试统计

| 包 | 测试文件 | 测试用例数 | 类型 | 状态 |
|----|---------|-----------|------|------|
| `internal/auth` | `scope_test.go` | 7 | 单元测试 | ✅ 通过 |
| `internal/auth` | `service_test.go` | 13 | 单元测试 | ✅ 通过 |
| `internal/handler` | `handler_test.go` | 11 | HTTP 集成测试 | ✅ 通过 |
| `internal/middleware` | `auth_test.go` | 9 | HTTP 集成测试 | ✅ 通过 |
| `internal/proxy` | `mock_test.go` | 5 | 单元测试 | ✅ 通过 |
| `internal/store` | `tenant_store_test.go` | 8 | 集成测试 | ✅ 通过 |
| `internal/store` | `apikey_store_test.go` | 8 | 集成测试 | ✅ 通过 |
| `internal/store` | `usage_store_test.go` | 6 | 集成测试 | ✅ 通过 |
| **合计** | **9** | **69** | — | **全部通过** |

---

## 3. 测试覆盖矩阵

### 3.1 认证模块 (`internal/auth`)

| 测试用例 | 场景 | 验证点 |
|---------|------|-------|
| `TestMatchScope_ExactMatch` | 精确匹配 `gpt-4` | 返回 true |
| `TestMatchScope_ExactMismatch` | 不匹配 `gpt-3` vs `gpt-4` | 返回 false |
| `TestMatchScope_Wildcard` | 通配符 `*` | 返回 true |
| `TestMatchScope_MultipleScopes_MatchSecond` | 多 Scope 匹配第二个 | 返回 true |
| `TestMatchScope_MultipleScopes_NoMatch` | 多 Scope 均不匹配 | 返回 false |
| `TestMatchScope_EmptyScopes` | 空 Scope 列表 | 返回 false |
| `TestMatchScope_WhitespaceInScopes` | 带空格的 Scope 列表 | 正确解析 |
| `TestValidateKey_Valid` | 完整有效 Key | 返回 AuthResult |
| `TestValidateKey_InvalidPrefix` | 前缀太短 | 返回错误 |
| `TestValidateKey_HashMismatch` | Key 哈希不匹配 | 返回错误 |
| `TestValidateKey_DisabledKey` | Key 已禁用 | 返回错误 |
| `TestValidateKey_ExpiredKey` | Key 已过期 | 返回错误 |
| `TestValidateKey_InsufficientScope` | 权限不足 | 返回错误 |
| `TestExtractPrefix` | 5 个子场景：正常/太短/空/空白/无 sk- 前缀 | 正确提取 |
| `TestHashKey` | SHA256 一致性 | 哈希稳定 |
| `TestGenerateKey` | 格式和长度 | 正确格式 `sk-{prefix}{secret}` |

### 3.2 Store 模块 (`internal/store`)

| 测试用例 | 场景 | 验证点 |
|---------|------|-------|
| `TestTenantStore_Create` | 创建租户 | 返回带 ID 的记录 |
| `TestTenantStore_CreateDuplicateName` | 重复名称 | 约束错误 |
| `TestTenantStore_GetByID` | 按 ID 查询 | 返回正确记录 |
| `TestTenantStore_GetByID_NotFound` | 不存在的 ID | 返回 nil |
| `TestTenantStore_List` | 列出所有 | 返回全部 |
| `TestTenantStore_List_Empty` | 空表 | 返回空列表 |
| `TestTenantStore_Update` | 更新名称 | 返回更新后记录 |
| `TestTenantStore_Delete` | 删除 | 返回 nil |
| `TestApiKeyStore_Create` | 创建 Key | 正确存储 |
| `TestApiKeyStore_GetByID` | 按 ID 查询 | 返回正确记录 |
| `TestApiKeyStore_FindByPrefix` | 按前缀查找 | 返回正确 Key |
| `TestApiKeyStore_FindByPrefix_NotFound` | 前缀不存在 | 返回 nil |
| `TestApiKeyStore_ListByTenant` | 按租户列出 | 返回该租户所有 Key |
| `TestApiKeyStore_Update` | 更新 Key 属性 | 更新后一致 |
| `TestApiKeyStore_Delete` | 删除 Key | 返回 nil |
| `TestApiKeyStore_UpdateLastUsed` | 更新最后使用时间 | 时间戳更新 |
| `TestUsageStore_Create` | 创建用量记录 | 正确存储 |
| `TestUsageStore_Query` | 按租户查询 | 返回正确记录 |
| `TestUsageStore_Query_Pagination` | 分页查询 | 分页正确 |
| `TestUsageStore_Query_NotFound` | 不存在的租户 | 空结果 |
| `TestUsageStore_BatchCreate` | 批量创建 | 多条记录 |
| `TestUsageStore_GetSummary` | 汇总统计 | 合计正确 |

### 3.3 Handler 模块 (`internal/handler`)

| 测试用例 | 场景 | 验证点 |
|---------|------|-------|
| `TestHealthCheck` | 健康检查 | 200 + `status: ok` + version |
| `TestHealthCheck_ContentType` | Content-Type 头 | `application/json` |
| `TestNotImplemented` | 未实现端点 | 501 + 错误消息 |
| `TestTenantHandler_Create` | 创建租户 | 201 + 返回租户 |
| `TestTenantHandler_Create_InvalidBody` | 非法请求体 | 400 |
| `TestTenantHandler_List` | 列出租户 | 200 + 2 条 |
| `TestTenantHandler_Delete` | 删除租户 | 204 + 实际删除 |
| `TestApiKeyHandler_Create` | 创建 Key | 201 + 返回完整 Key |
| `TestApiKeyHandler_Create_InvalidBody` | 非法请求体 | 400 |
| `TestApiKeyHandler_List` | 列出 Key | 200 + 2 条 |
| `TestApiKeyHandler_Delete` | 删除 Key | 204 + 实际删除 |

### 3.4 中间件模块 (`internal/middleware`)

| 测试用例 | 场景 | 验证点 |
|---------|------|-------|
| `TestAuthenticate_ValidKey` | 有效 Bearer Token | 200 + 注入 TenantID/KeyID/Scopes |
| `TestAuthenticate_MissingHeader` | 无 Authorization 头 | 401 |
| `TestAuthenticate_InvalidFormat` | 非 Bearer 格式 | 401 |
| `TestAuthenticate_InvalidKey` | 无效 Key | 401 |
| `TestCORS` | OPTIONS 预检 | 200 + CORS 头 |
| `TestCORS_HeaderOnNormalRequest` | 正常请求带 CORS 头 | 200 + Origin 回显 |
| `TestGetTenantID_EmptyContext` | 空上下文取 TenantID | 空字符串 |
| `TestGetKeyID_EmptyContext` | 空上下文取 KeyID | 空字符串 |
| `TestGetScopes_EmptyContext` | 空上下文取 Scopes | 空字符串 |

### 3.5 Proxy 模块 (`internal/proxy`)

| 测试用例 | 场景 | 验证点 |
|---------|------|-------|
| `TestMockProvider_ChatCompletion` | 正常 Chat 请求 | 200 + 模拟响应格式 |
| `TestMockProvider_ChatCompletion_ServerError` | 服务端错误 | 500 状态码 |
| `TestMockProvider_ListModels` | 模型列表 | 返回模型列表 |
| `TestMockProvider_BaseURLTrim` | URL 末尾斜杠处理 | 正确拼接 |
| `TestMockProvider_ConnectionRefused` | 连接拒绝 | 502 错误 |

---

## 4. 测试运行

```bash
# 运行所有测试
go test ./... -v -count=1

# 运行单个包测试
go test ./internal/auth/... -v -count=1
go test ./internal/store/... -v -count=1
go test ./internal/handler/... -v -count=1
go test ./internal/middleware/... -v -count=1
go test ./internal/proxy/... -v -count=1

# 运行单个测试用例
go test ./internal/auth/... -run TestValidateKey_Valid -v -count=1

# 编译检查（不运行测试）
go build ./...
```

---

## 5. 测试编写规范

### 5.1 命名

```
Test{函数名}_{场景}
```

示例：
- `TestMatchScope_Wildcard`
- `TestTenantHandler_Create_InvalidBody`
- `TestAuthenticate_ValidKey`

### 5.2 辅助函数

- `setupHandlerDB(t)` — 创建 `:memory:` SQLite 数据库
- `withChiURLParam(ctx, key, value)` — 注入 chi URL 参数上下文

### 5.3 断言风格

- 使用标准库 `testing.T` 的 `Errorf`/`Fatalf`
- 错误信息格式：`"expected X, got Y"` 或 `"expected error for XXX, got nil"`
- 不引入第三方断言库

---

## 6. 测试数据

- 所有测试使用 `:memory:` SQLite 数据库，互不干扰
- 测试 Key 使用 `auth.GenerateKey()` 生成，通过 `hashKey()` 验证
- 租户/Key/用量数据在测试中直接创建，不依赖外部 fixture

---

## 7. 改进计划

| 优先级 | 改进项 | 说明 |
|--------|-------|------|
| P0 | 无 | 当前测试覆盖已满足 MVP 要求 |
| P1 | 添加并发测试 | 验证 SQLite 单连接限制下的并发安全 |
| P2 | 添加覆盖率门禁 | 配置 CI 确保覆盖率 > 70% |
| P3 | 添加模糊测试 | 对 Key 解析/Scope 匹配做 fuzz |
| P4 | 集成 E2E 测试 | 启动完整服务后做 curl 验证 |

---

## 文档版本记录

| 版本 | 日期 | 变更内容 |
|------|------|---------|
| 1.0 | 2026-07-07 | 初始版本 |