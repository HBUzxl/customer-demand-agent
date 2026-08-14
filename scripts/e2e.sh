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

# 统一消息入口（ADR-014）：SSE 200 即过（LLM 不可用时不等 done）
# SSE 端点：max-time 到点断开属正常（响应头已到）；只取 -w 输出的状态码。
# 固定 session_id（e2e-msg / e2e-an），结束时清理——不留脏数据（客户端断开
# 不再写 [失败]，见 messageCore；老实例上跑会留痕，勿用生产库跑 e2e）。
MSTATUS=$("$CURL" -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: ${TENANT}" -H "Content-Type: application/json" -d '{"text":"你好","session_id":"e2e-msg"}' --max-time 3 "$BASE/api/message" 2>/dev/null)
case "$MSTATUS" in *"000"*) MSTATUS="${MSTATUS//000/}";; esac
[ "$MSTATUS" = "200" ] && ok "POST /api/message（统一入口 SSE）" || fail "POST /api/message（got ${MSTATUS}）"
DSTATUS=$("$CURL" -s -o /dev/null -w "%{http_code}" -H "Content-Type: application/json" -d '{"text":"hi","session_id":"e2e-an"}' --max-time 3 "$BASE/api/analyze" 2>/dev/null)
case "$DSTATUS" in *"000"*) DSTATUS="${DSTATUS//000/}";; esac
[ "$DSTATUS" = "200" ] && ok "POST /api/analyze（deprecated shim 仍可用）" || fail "analyze shim（got ${DSTATUS}）"

# 带租户头的请求辅助
postt() {    # postt <tenant> <path> <json>
  "$CURL" -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: $1" -H "Content-Type: application/json" -d "$3" "$BASE$2"
}
deleteAs() { # deleteAs <tenant> <path>
  "$CURL" -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: $1" -X DELETE "$BASE$2"
}

echo "-- 6. 安全（C1 SSRF / C2 凭据外泄 / 跨租户隔离） --"
# 注：以下均在 LLM 调用前被拒，无需 api_key（与 e2e 的无 LLM 前提一致）。
# C1 SSRF：探测端点拒绝云元数据 / 回环
[ "$(post /api/config/test '{"endpoint":"http://169.254.169.254/","api_key":"x","model":"m","protocol":"openai-chat"}')" = "400" ] && ok "C1 /api/config/test 拒绝云元数据" || fail "C1 SSRF config/test 未拒绝"
[ "$(post /api/models '{"endpoint":"http://127.0.0.1:80/","api_key":"x","protocol":"openai-chat"}')" = "400" ] && ok "C1 /api/models 拒绝回环" || fail "C1 SSRF models 未拒绝"
# C2 凭据外泄：name 命中已存模型但 endpoint 不同源、未传 key → 拒绝代填
[ "$(post /api/config/test '{"name":"default","endpoint":"http://e2e-credexfil.invalid/v1","protocol":"openai-chat","model":"m"}')" = "400" ] && ok "C2 不同源 endpoint 拒绝代填 key" || fail "C2 凭据外泄未堵"
# 跨租户：tenantA 认领 session（无 key 时 LLM 快速失败，但 EnsureSession 已建会话），tenantB 复用同一 session_id → 403
[ "$(postt tenantA /api/analyze '{"session_id":"e2e-sec","text":"x"}')" = "200" ] && ok "跨租户 tenantA 认领 session" || fail "跨租户 A 认领失败"
[ "$(postt tenantB /api/analyze '{"session_id":"e2e-sec","text":"x"}')" = "403" ] && ok "跨租户 tenantB 被拒（403）" || fail "跨租户 B 未拒绝（隔离泄漏）"
[ "$(deleteAs tenantA /api/sessions/e2e-sec)" = "200" ] && ok "清理 e2e-sec session" || fail "清理 e2e-sec"

echo "-- 7. 清理测试数据 --"
[ "$(delete '/api/memory/threat/E2E%E6%B5%8B%E8%AF%95%E5%A8%81%E8%83%81?archive=false')" = "200" ] && ok "DELETE /api/memory (清理)" || fail "memory delete"
[ "$(delete /api/sessions/e2e-msg)" = "200" ] && ok "清理 e2e-msg session" || fail "清理 e2e-msg"
[ "$(delete /api/sessions/e2e-an)" = "200" ] && ok "清理 e2e-an session" || fail "清理 e2e-an"

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
