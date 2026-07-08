#!/bin/bash
# =============================================================================
# AI Gateway MVP 集成验证脚本
# 使用方式：
#   1. docker compose up -d
#   2. ./scripts/verify.sh
# =============================================================================

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() {
    PASS=$((PASS + 1))
    echo -e "  ${GREEN}✓${NC} $1"
}

fail() {
    FAIL=$((FAIL + 1))
    echo -e "  ${RED}✗${NC} $1"
    if [ -n "${2:-}" ]; then
        echo -e "    ${YELLOW}reason:${NC} $2"
    fi
}

check_status() {
    local expected=$1
    local actual=$2
    local label=$3
    if [ "$actual" -eq "$expected" ]; then
        pass "$label (HTTP $actual)"
    else
        fail "$label (expected HTTP $expected, got $actual)"
    fi
}

echo ""
echo "============================================"
echo " AI Gateway MVP 集成验证"
echo "============================================"
echo ""

# -------------------------------------------------------------------------
# 1. 健康检查
# -------------------------------------------------------------------------
echo "━━━ 1. 健康检查 ━━━"

RESP=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
check_status 200 "$RESP" "健康检查返回 200"

BODY=$(curl -s "$BASE_URL/health")
if echo "$BODY" | grep -q '"status":"ok"'; then
    pass "健康检查 body 包含 status: ok"
else
    fail "健康检查 body 缺少 status: ok" "$BODY"
fi

# -------------------------------------------------------------------------
# 2. 管理 API — 创建租户
# -------------------------------------------------------------------------
echo ""
echo "━━━ 2. 管理 API — 创建租户 ━━━"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/tenants" \
    -H "Content-Type: application/json" \
    -d '{"name":"integration-test-tenant","description":"Created by verify.sh"}')
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check_status 201 "$HTTP_CODE" "创建租户返回 201"

TENANT_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$TENANT_ID" ]; then
    pass "租户 ID 非空: $TENANT_ID"
else
    fail "未能提取租户 ID" "$BODY"
fi

# -------------------------------------------------------------------------
# 3. 管理 API — 创建 API Key
# -------------------------------------------------------------------------
echo ""
echo "━━━ 3. 管理 API — 创建 API Key ━━━"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/keys" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"test-key\",\"tenant_id\":\"$TENANT_ID\",\"scopes\":\"gpt-4,gpt-3.5-turbo\"}")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check_status 201 "$HTTP_CODE" "创建 API Key 返回 201"

API_KEY=$(echo "$BODY" | grep -o '"full_key":"[^"]*"' | head -1 | cut -d'"' -f4)
KEY_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$API_KEY" ] && [ -n "$KEY_ID" ]; then
    pass "API Key 完整返回 (key=$API_KEY, id=$KEY_ID)"
else
    fail "API Key 返回值不完整" "$BODY"
fi

# -------------------------------------------------------------------------
# 4. 管理 API — 列出租户
# -------------------------------------------------------------------------
echo ""
echo "━━━ 4. 管理 API — 列出租户 ━━━"

RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/tenants")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check_status 200 "$HTTP_CODE" "列出租户返回 200"
if echo "$BODY" | grep -q "$TENANT_ID"; then
    pass "租户列表包含刚创建的租户"
else
    fail "租户列表缺少新创建的租户" "$BODY"
fi

# -------------------------------------------------------------------------
# 5. 管理 API — 列出 Key
# -------------------------------------------------------------------------
echo ""
echo "━━━ 5. 管理 API — 列出 Key ━━━"

RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/keys")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
check_status 200 "$HTTP_CODE" "列出 Key 返回 200"

# -------------------------------------------------------------------------
# 6. 代理 API — 聊天补全（需认证）
# -------------------------------------------------------------------------
echo ""
echo "━━━ 6. 代理 API — 聊天补全 ━━━"

# 目前返回 501（未实现），验证认证通过
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/chat/completions" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}')
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
if [ "$HTTP_CODE" -eq 501 ]; then
    pass "代理聊天补全返回 501（未实现，预期行为）"
    if echo "$BODY" | grep -q '"error"'; then
        pass "响应体包含 error 字段"
    fi
elif [ "$HTTP_CODE" -eq 200 ]; then
    pass "代理聊天补全返回 200（已实现）"
else
    fail "代理聊天补全返回异常" "HTTP $HTTP_CODE: $BODY"
fi

# -------------------------------------------------------------------------
# 7. 代理 API — 模型列表（需认证）
# -------------------------------------------------------------------------
echo ""
echo "━━━ 7. 代理 API — 模型列表 ━━━"

RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/v1/models" \
    -H "Authorization: Bearer $API_KEY")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)
if [ "$HTTP_CODE" -eq 501 ]; then
    pass "模型列表返回 501（未实现，预期行为）"
elif [ "$HTTP_CODE" -eq 200 ]; then
    pass "模型列表返回 200（已实现）"
else
    fail "模型列表返回异常" "HTTP $HTTP_CODE: $BODY"
fi

# -------------------------------------------------------------------------
# 8. 错误路径 — 401 无认证
# -------------------------------------------------------------------------
echo ""
echo "━━━ 8. 错误路径 — 401 无认证 ━━━"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d '{"model":"gpt-4"}')
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status 401 "$HTTP_CODE" "无认证请求返回 401"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/chat/completions" \
    -H "Authorization: Bearer sk-invalid-key" \
    -H "Content-Type: application/json" \
    -d '{"model":"gpt-4"}')
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status 401 "$HTTP_CODE" "无效 Key 请求返回 401"

# -------------------------------------------------------------------------
# 9. 错误路径 — 400 非法请求体
# -------------------------------------------------------------------------
echo ""
echo "━━━ 9. 错误路径 — 400 非法请求体 ━━━"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/tenants" \
    -H "Content-Type: application/json" \
    -d 'not-json')
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status 400 "$HTTP_CODE" "非法 JSON 请求返回 400"

RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/keys" \
    -H "Content-Type: application/json" \
    -d '{"name":"test"}')
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status 400 "$HTTP_CODE" "缺少必填字段的 Key 请求返回 400"

# -------------------------------------------------------------------------
# 10. 管理 API — 删除租户
# -------------------------------------------------------------------------
echo ""
echo "━━━ 10. 管理 API — 删除租户 ━━━"

RESP=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/api/tenants/$TENANT_ID")
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status 204 "$HTTP_CODE" "删除租户返回 204"

# 验证已删除
RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/tenants/$TENANT_ID")
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status 404 "$HTTP_CODE" "删除后再查询返回 404"

# -------------------------------------------------------------------------
# 11. 管理 API — 删除 Key
# -------------------------------------------------------------------------
echo ""
echo "━━━ 11. 管理 API — 删除 Key ━━━"

RESP=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/api/keys/$KEY_ID")
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status 204 "$HTTP_CODE" "删除 Key 返回 204"

# -------------------------------------------------------------------------
# 12. 404 处理
# -------------------------------------------------------------------------
echo ""
echo "━━━ 12. 404 处理 ━━━"

RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/nonexistent")
HTTP_CODE=$(echo "$RESP" | tail -1)
check_status 404 "$HTTP_CODE" "不存在路径返回 404"

# -------------------------------------------------------------------------
# 汇总
# -------------------------------------------------------------------------
echo ""
echo "============================================"
echo -e " 结果: ${GREEN}$PASS 通过${NC} / ${RED}$FAIL 失败${NC}"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi