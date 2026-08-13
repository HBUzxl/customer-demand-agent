#!/usr/bin/env bash
# 客户需求分析智能体 —— 端到端 API 冒烟测试
#
# 用法：
#   1. 先启动后端：  ./scripts/run.sh   （或 go run ./cmd/agent）
#   2. 运行本脚本：  ./scripts/e2e.sh
#
# 覆盖：健康检查 / 配置 / 记忆检索 / 记忆写入+审核流 / 会话历史
# 注意：/api/analyze 需要 LLM_API_KEY，否则返回 502（见 README）。

set -uo pipefail

BASE="${BASE:-http://localhost:8080}"
TENANT="${TENANT_ID:-default}"
# 用绝对路径 curl，避免被 shell 别名/RTK 拦截
CURL="${CURL:-/usr/bin/curl}"
PASS=0
FAIL=0

ok()   { echo "  [PASS] $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }

# get <path> —— 期望 200
get() {
  "$CURL" -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: ${TENANT}" "$BASE$1"
}
# post <path> <json>
post() {
  "$CURL" -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: ${TENANT}" \
    -H "Content-Type: application/json" -d "$2" "$BASE$1"
}
delete() {
  "$CURL" -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: ${TENANT}" -X DELETE "$BASE$1"
}

echo "=== E2E (base=${BASE} tenant=${TENANT}) ==="

echo "-- 1. 健康检查 --"
[ "$(get /api/health)" = "200" ] && ok "GET /api/health" || { fail "GET /api/health (后端是否启动？)"; exit 1; }

echo "-- 2. 配置 --"
[ "$(get /api/config)" = "200" ] && ok "GET /api/config" || fail "GET /api/config"

echo "-- 3. 记忆检索 --"
[ "$(get '/api/memory/list?type=product')" = "200" ] && ok "GET /api/memory/list" || fail "memory list"
[ "$(get '/api/memory/search?q=CC&type=product')" = "200" ] && ok "GET /api/memory/search (CC)" || fail "memory search"
[ "$(get '/api/memory/product/%E9%9B%B7%E6%B1%A0')" = "200" ] && ok "GET /api/memory/product/雷池" || fail "memory get 雷池"

echo "-- 4. 记忆写入 + 审核流 (threat 待审核 -> 批准 -> verified) --"
[ "$(post /api/memory '{"type":"threat","title":"E2E测试威胁","content":"自动化测试","status":"pending_review"}')" = "200" ] && ok "POST /api/memory (threat, pending)" || fail "memory upsert"

PENDING="$("$CURL" -s "$BASE/api/review/pending")"
echo "$PENDING" | grep -q "E2E测试威胁" && ok "GET /api/review/pending (出现待审核)" || fail "review pending 未出现"

[ "$(post /api/review/threat/E2E%E6%B5%8B%E8%AF%95%E5%A8%81%E8%83%81/approve '{}')" = "200" ] && ok "POST /api/review/.../approve" || fail "review approve"

echo "-- 5. 会话历史 --"
[ "$(get /api/sessions)" = "200" ] && ok "GET /api/sessions" || fail "sessions list"

echo "-- 6. 清理测试数据 --"
[ "$(delete '/api/memory/threat/E2E%E6%B5%8B%E8%AF%95%E5%A8%81%E8%83%81?archive=false')" = "200" ] && ok "DELETE /api/memory (清理)" || fail "memory delete"

echo ""
echo "=== 结果: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -gt 0 ] && exit 1

cat <<'NOTE'

-- 7. 端到端分析 (需 LLM_API_KEY) --
若已配置 LLM_API_KEY，可测试完整分析链路：

  curl -s -X POST localhost:8080/api/analyze \
    -H 'Content-Type: application/json' \
    -d '{"text":"我们电商网站大促时被CC攻击，正在过等保二级"}' | jq

预期返回 analysis: { demand_analysis, matched_products:[雷池...], feasibility:"direct", missing_info }
NOTE
