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

echo "-- 4. 记忆写入语义 (P10: 人工通道一律 verified, 不接受 status 传参) --"
[ "$(post /api/memory '{"type":"threat","title":"E2E测试威胁","content":"自动化测试","status":"pending_review"}')" = "200" ] && ok "POST /api/memory (threat)" || fail "memory upsert"

# P10：人工写入即使传 status=pending_review 也落 verified → 不进审核队列
PENDING="$("$CURL" -s "$BASE/api/review/pending")"
echo "$PENDING" | grep -q "E2E测试威胁" && fail "P10 违规：人工 pending 传参生效" || ok "人工写入不进审核队列（P10 语义）"

# 人工写入的条目确为 verified（可被检索）
GETSTAT="$("$CURL" -s "$BASE/api/memory/threat/E2E%E6%B5%8B%E8%AF%95%E5%A8%81%E8%83%81")"
echo "$GETSTAT" | grep -q '"verified"' && ok "人工条目状态为 verified" || fail "人工条目应为 verified"

[ "$(delete '/api/memory/threat/E2E%E6%B5%8B%E8%AF%95%E5%A8%81%E8%83%81?archive=false')" = "200" ] && ok "清理测试威胁" || fail "清理威胁"

echo "-- 5. 会话历史 --"
[ "$(get /api/sessions)" = "200" ] && ok "GET /api/sessions" || fail "sessions list"

# 统一消息入口（ADR-014 + F0：202 + Run 任务，立即返回 run_id）
# 固定 session_id（e2e-msg / e2e-an），结束时清理；断开不影响执行（F0）。
MRESP=$("$CURL" -s -H "X-Tenant-ID: ${TENANT}" -H "Content-Type: application/json" -d '{"text":"你好","session_id":"e2e-msg"}' "$BASE/api/message" 2>/dev/null)
echo "$MRESP" | grep -q '"run_id"' && ok "POST /api/message（202+run_id，F0）" || fail "POST /api/message（got ${MRESP}）"
# 订阅流：replay 应有 session 事件（Run 在后台跑，与请求连接无关）
STREAM=$("$CURL" -s -N --max-time 2 "$BASE/api/sessions/e2e-msg/stream?since=0" 2>/dev/null | head -c 400)
echo "$STREAM" | grep -q '"type":"session"' && ok "GET /stream replay（session 事件）" || fail "stream replay（got ${STREAM:0:80}）"
# 取消该 Run（让会话可复用/清理）
RID=$(echo "$MRESP" | sed 's/.*"run_id":"\([^"]*\)".*/\1/')
[ -n "$RID" ] && "$CURL" -s -o /dev/null -X POST "$BASE/api/sessions/e2e-msg/runs/$RID/cancel" -H "X-Tenant-ID: ${TENANT}" && ok "POST cancel（停止运行）" || fail "cancel"

# P0 前端行为契约：新会话空状态 chip 填客户名后发首条消息——请求体带 customer，
# 会话落库可读回（首轮身份行注入的数据前提）。
MCUST=$("$CURL" -s -H "X-Tenant-ID: ${TENANT}" -H "Content-Type: application/json" -d '{"text":"你好","session_id":"e2e-cust","customer":"E2E测试客户"}' "$BASE/api/message" 2>/dev/null)
echo "$MCUST" | grep -q '"run_id"' && ok "POST /api/message 首轮带 customer（空状态 chip 契约）" || fail "首轮带 customer（got ${MCUST}）"
CRID=$(echo "$MCUST" | sed 's/.*"run_id":"\([^"]*\)".*/\1/')
[ -n "$CRID" ] && "$CURL" -s -o /dev/null -X POST "$BASE/api/sessions/e2e-cust/runs/$CRID/cancel" -H "X-Tenant-ID: ${TENANT}"
CUSTBACK=$("$CURL" -s -H "X-Tenant-ID: ${TENANT}" "$BASE/api/sessions/e2e-cust" | grep -o '"customer":"[^"]*"' | head -1)
[ "$CUSTBACK" = '"customer":"E2E测试客户"' ] && ok "首轮 customer 落库可读回" || fail "customer 落库读回（got ${CUSTBACK}）"
[ "$("$CURL" -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-Tenant-ID: ${TENANT}" "$BASE/api/sessions/e2e-cust")" = "200" ] && ok "清理 e2e-cust" || fail "清理 e2e-cust"
DRESP=$("$CURL" -s -H "Content-Type: application/json" -d '{"text":"hi","session_id":"e2e-an"}' "$BASE/api/analyze" 2>/dev/null)
echo "$DRESP" | grep -q '"run_id"' && ok "POST /api/analyze（deprecated shim→202）" || fail "analyze shim（got ${DRESP}）"
DRID=$(echo "$DRESP" | sed 's/.*"run_id":"\([^"]*\)".*/\1/')
[ -n "$DRID" ] && "$CURL" -s -o /dev/null -X POST "$BASE/api/sessions/e2e-an/runs/$DRID/cancel" -H "X-Tenant-ID: ${TENANT}"

# 带租户头的请求辅助
postt() {    # postt <tenant> <path> <json> → body（F0：202 响应体含 run_id）
  "$CURL" -s -H "X-Tenant-ID: $1" -H "Content-Type: application/json" -d "$3" "$BASE$2"
}
posttcode() { # posttcode <tenant> <path> <json> → http_code
  "$CURL" -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: $1" -H "Content-Type: application/json" -d "$3" "$BASE$2"
}
deleteAs() { # deleteAs <tenant> <path>
  "$CURL" -s -o /dev/null -w "%{http_code}" -H "X-Tenant-ID: $1" -X DELETE "$BASE$2"
}

echo "-- 6. 安全（C1 SSRF / C2 凭据外泄 / de-tenancy） --"
# 注：以下均在 LLM 调用前被拒，无需 api_key（与 e2e 的无 LLM 前提一致）。
# C1 SSRF：探测端点拒绝云元数据 / 回环
[ "$(post /api/config/test '{"endpoint":"http://169.254.169.254/","api_key":"x","model":"m","protocol":"openai-chat"}')" = "400" ] && ok "C1 /api/config/test 拒绝云元数据" || fail "C1 SSRF config/test 未拒绝"
# F0 失败语义：不存在的会话 cancel → 404（前端 ApiError.status 分支的契约）
[ "$("$CURL" -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/sessions/none-e2e/runs/run-x/cancel")" = "404" ] && ok "F0 cancel 不存在会话 → 404" || fail "F0 cancel 404 语义"
[ "$(post /api/models '{"endpoint":"http://127.0.0.1:80/","api_key":"x","protocol":"openai-chat"}')" = "400" ] && ok "C1 /api/models 拒绝回环" || fail "C1 SSRF models 未拒绝"
# C2 凭据外泄：name 命中已存模型但 endpoint 不同源、未传 key → 拒绝代填
[ "$(post /api/config/test '{"name":"default","endpoint":"http://e2e-credexfil.invalid/v1","protocol":"openai-chat","model":"m"}')" = "400" ] && ok "C2 不同源 endpoint 拒绝代填 key" || fail "C2 凭据外泄未堵"
# de-tenancy：X-Tenant-ID 头接受但忽略——任意头同会话可访问（无 403）
SECRESP=$(postt whatever /api/analyze '{"session_id":"e2e-sec","text":"x"}')
echo "$SECRESP" | grep -q run_id && ok "de-tenancy：带任意租户头建会话（202）" || fail "de-tenancy 建会话失败（got ${SECRESP}）"
SECRID=$(echo "$SECRESP" | sed 's/.*"run_id":"\([^"]*\)".*/\1/')
[ -n "$SECRID" ] && "$CURL" -s -o /dev/null -X POST "$BASE/api/sessions/e2e-sec/runs/$SECRID/cancel"
DT_CODE=$(posttcode other /api/analyze '{"session_id":"e2e-sec","text":"x"}')
[ "$DT_CODE" != "403" ] && ok "de-tenancy：不同租户头同会话不再被拒（${DT_CODE}，202/409 均为正常）" || fail "de-tenancy 跨头仍 403（租户隔离未砍净）"
[ "$(deleteAs whoever /api/sessions/e2e-sec)" = "200" ] && ok "清理 e2e-sec session" || fail "清理 e2e-sec"

echo "-- 7. 清理测试数据 --"
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
