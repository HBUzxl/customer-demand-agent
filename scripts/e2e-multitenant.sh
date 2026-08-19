#!/usr/bin/env bash
# 多租户端到端冒烟测试（scripts/e2e-multitenant.sh）
#
# 用法：bash scripts/e2e-multitenant.sh
#   - 自建临时 CDA_DATA_DIR + Mock LLM，不依赖外部服务；
#   - 双租户注册/登录 → 同名客户「某集团」互不可见 → 会话/SSE/取消/删除
#     跨租户 404 → 记忆/搜索隔离 → 任务列表按租户过滤 → 平台路由 403；
#   - 覆盖设计文档 §16 验收标准中「跨租户访问一律 404、同名客户互不可见、
#     未登录 401、越权 403」的 HTTP 形态。
#
# 环境变量：APP_PORT（默认 18180）、LLM_PORT（默认 18181）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d /tmp/cda-mt-e2e.XXXXXX)"
APP_PORT="${APP_PORT:-18180}"
LLM_PORT="${LLM_PORT:-18181}"
BASE="http://127.0.0.1:${APP_PORT}"
BIN="${WORK}/agent"
CFG="${WORK}/config.json"
DATA="${WORK}/data"
MOCK="${WORK}/mock_llm.py"
JAR_A="${WORK}/jar_a.txt"
JAR_B="${WORK}/jar_b.txt"

PASS=0
FAIL=0
ok()   { echo "  [PASS] $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }

cleanup() {
  [ -n "${AGENT_PID:-}" ] && kill "$AGENT_PID" 2>/dev/null || true
  [ -n "${LLM_PID:-}" ] && kill "$LLM_PID" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

# enc <str>：路径段 URL 编码（中文标题）
enc() { python3 -c 'import sys,urllib.parse;print(urllib.parse.quote(sys.argv[1]))' "$1"; }
# codeA/codeB <args…>：带各自 Cookie jar 的 HTTP 状态码（-b 发送 / -c 保存）
codeA() { curl -s -o /dev/null -w "%{http_code}" -b "$JAR_A" -c "$JAR_A" "$@"; }
codeB() { curl -s -o /dev/null -w "%{http_code}" -b "$JAR_B" -c "$JAR_B" "$@"; }
bodyA() { curl -s -b "$JAR_A" -c "$JAR_A" "$@"; }
bodyB() { curl -s -b "$JAR_B" -c "$JAR_B" "$@"; }
# csrf_of <body>：从 auth 响应取 csrf
csrf_of() { printf '%s' "$1" | sed -n 's/.*"csrf":"\([^"]*\)".*/\1/p'; }

echo "=== 多租户 E2E（base=${BASE}，mock-llm:${LLM_PORT}）==="

echo "-- 0. 构建 + 启动（Mock LLM + Agent）--"
( cd "$ROOT" && go build -o "$BIN" ./cmd/agent )

cat > "$MOCK" <<'PY'
import json, os, time
from http.server import BaseHTTPRequestHandler, HTTPServer
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        time.sleep(1.5)  # 保持 Run 活跃窗口，便于测试跨租户 cancel 隔离
        resp = {"choices":[{"finish_reason":"stop","message":{"role":"assistant",
                "content":"这是多租户端到端测试的模拟分析结论。"}}],
                "usage":{"total_tokens":20,"prompt_tokens":10,"completion_tokens":10}}
        data = json.dumps(resp).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
    def log_message(self, *a): pass
HTTPServer(("127.0.0.1", int(os.environ["LLM_PORT"])), H).serve_forever()
PY
LLM_PORT="$LLM_PORT" python3 "$MOCK" &
LLM_PID=$!
sleep 0.5

cat > "$CFG" <<EOF
{
  "server": {"addr": "127.0.0.1:${APP_PORT}", "frontend_dist": ""},
  "data_dir": "${DATA}",
  "models": [
    {"name":"mock","endpoint":"http://127.0.0.1:${LLM_PORT}/v1/chat/completions","api_key":"test-key","protocol":"openai-chat","model":"mock"}
  ],
  "router": {"default":"mock","routes":{},"fallback":{"max_retries":1,"backoff_base_ms":50,"chain":[]}},
  "agent_max_iterations": 3,
  "registration_enabled": true,
  "secure_cookies": false,
  "jiying": {"enabled": false},
  "lead_manager": {"enabled": false}
}
EOF
CDA_DATA_DIR="$DATA" "$BIN" -config "$CFG" > "$WORK/server.log" 2>&1 &
AGENT_PID=$!
for _ in $(seq 1 60); do
  curl -s -o /dev/null "$BASE/api/health" && break
  sleep 0.25
done
[ "$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/health")" = "200" ] && ok "健康检查（公开）" || { fail "健康检查未就绪"; cat "$WORK/server.log"; exit 1; }

echo "-- 1. 未登录 401 --"
[ "$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/sessions")" = "401" ] && ok "GET /api/sessions 未登录 → 401" || fail "未登录应 401"
[ "$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/memory/list?type=product")" = "401" ] && ok "GET /api/memory/list 未登录 → 401" || fail "记忆未登录应 401"

echo "-- 2. 注册双租户 A/B --"
PW="Password-123"
RA=$(curl -s -c "$JAR_A" -H "Content-Type: application/json" \
  -d "{\"org_name\":\"A公司\",\"display_name\":\"A用户\",\"email\":\"a@mt.test\",\"password\":\"${PW}\"}" \
  "$BASE/api/auth/register")
[ "$(codeA "$BASE/api/auth/me")" = "200" ] && ok "注册 A 后 me → 200" || fail "注册 A 失败"
RB=$(curl -s -c "$JAR_B" -H "Content-Type: application/json" \
  -d "{\"org_name\":\"B公司\",\"display_name\":\"B用户\",\"email\":\"b@mt.test\",\"password\":\"${PW}\"}" \
  "$BASE/api/auth/register")
[ "$(codeB "$BASE/api/auth/me")" = "200" ] && ok "注册 B 后 me → 200" || fail "注册 B 失败"

# /auth/me 返回同一会话的稳定 CSRF；统一从 me 读取后供后续修改请求使用。
MA=$(bodyA "$BASE/api/auth/me"); CA=$(csrf_of "$MA")
MB=$(bodyB "$BASE/api/auth/me"); CB=$(csrf_of "$MB")
[ -n "$CA" ] && [ -n "$CB" ] && ok "A/B 均取得 CSRF" || fail "CSRF 缺失"
echo "$MA" | grep -q '"tenant_name":"A公司"' && ok "A me 返回租户 A公司" || fail "A me 租户名"
echo "$MA" | grep -q '"owner"' && ok "A 角色含 owner" || fail "A owner 角色"
echo "$MB" | grep -q '"tenant_name":"B公司"' && ok "B me 返回租户 B公司" || fail "B me 租户名"

echo "-- 3. 越权 / CSRF --"
# 租户用户访问平台路由 → 403
[ "$(codeB "$BASE/api/platform/config")" = "403" ] && ok "B 访问 /api/platform/config → 403" || fail "平台 config 应 403"
[ "$(codeB "$BASE/api/platform/llm-audit")" = "403" ] && ok "B 访问 /api/platform/llm-audit → 403" || fail "平台审计应 403"
[ "$(codeB "$BASE/api/platform/logs")" = "403" ] && ok "B 访问 /api/platform/logs → 403" || fail "平台日志应 403"
# 租户安全视图：GET /api/config 200 且不含 data_dir
[ "$(codeA "$BASE/api/config")" = "200" ] && ok "A GET /api/config（安全视图）→ 200" || fail "config 安全视图"
SAFE=$(bodyA "$BASE/api/config")
echo "$SAFE" | grep -q '"data_dir"' && fail "安全视图泄露 data_dir" || ok "安全视图不含 data_dir"
# 错误 CSRF → 403
[ "$(codeA -H "Content-Type: application/json" -H "X-CSRF-Token: bogus" -X POST -d '{"type":"threat","title":"x","content":"y"}' "$BASE/api/memory")" = "403" ] && ok "错误 CSRF → 403" || fail "CSRF 未拦截"

echo "-- 4. 同名客户「某集团」互不可见 --"
# A/B 各自写入同名客户（人工作业 → verified，不互串）
[ "$(codeA -H "Content-Type: application/json" -H "X-CSRF-Token: ${CA}" -X POST -d '{"type":"customer","title":"某集团","content":"A公司 的内部客户档案","status":"verified"}' "$BASE/api/memory")" = "200" ] && ok "A 写入客户 某集团" || fail "A 写客户"
[ "$(codeB -H "Content-Type: application/json" -H "X-CSRF-Token: ${CB}" -X POST -d '{"type":"customer","title":"某集团","content":"B公司 的完全不同档案","status":"verified"}' "$BASE/api/memory")" = "200" ] && ok "B 写入客户 某集团" || fail "B 写客户"
GA=$(bodyA "$BASE/api/memory/customer/$(enc "某集团")")
GB=$(bodyB "$BASE/api/memory/customer/$(enc "某集团")")
echo "$GA" | grep -q "A公司 的内部客户档案" && ok "A 读到自己的某集团" || fail "A 读客户内容"
echo "$GA" | grep -q "B公司 的完全不同档案" && fail "A 看到 B 的客户内容（串租户）" || ok "A 读不到 B 的客户内容"
echo "$GB" | grep -q "B公司 的完全不同档案" && ok "B 读到自己的某集团" || fail "B 读客户内容"
echo "$GB" | grep -q "A公司 的内部客户档案" && fail "B 看到 A 的客户内容（串租户）" || ok "B 读不到 A 的客户内容"

echo "-- 5. 记忆搜索隔离 --"
SA=$(bodyA "$BASE/api/memory/search?q=$(enc "某集团")&type=customer")
SB=$(bodyB "$BASE/api/memory/search?q=$(enc "某集团")&type=customer")
echo "$SA" | grep -q "A公司 的内部客户档案" && ok "A 搜索命中自己的某集团" || fail "A 搜索"
echo "$SB" | grep -q "B公司 的完全不同档案" && ok "B 搜索命中自己的某集团" || fail "B 搜索"
echo "$SB" | grep -q "A公司 的内部客户档案" && fail "B 搜索串到 A（租户泄漏）" || ok "B 搜索不含 A 档案"

echo "-- 6. 会话：A 创建 → B 跨租户一律 404 --"
MRESP=$(bodyA -H "Content-Type: application/json" -H "X-CSRF-Token: ${CA}" \
  -X POST -d '{"text":"请分析某集团的安全需求","session_id":"e2e-sess-a","customer":"某集团"}' "$BASE/api/message")
echo "$MRESP" | grep -q '"run_id"' && ok "A POST /api/message → 202+run_id" || fail "A 发消息（got ${MRESP}）"
RID=$(printf '%s' "$MRESP" | sed -n 's/.*"run_id":"\([^"]*\)".*/\1/p')
SESS=$(bodyA "$BASE/api/sessions/e2e-sess-a")
echo "$SESS" | grep -q '"customer":"某集团"' && ok "A 会话 customer 落库" || fail "会话 customer"
[ "$(codeA "$BASE/api/sessions/e2e-sess-a")" = "200" ] && ok "A 读自己的会话 → 200" || fail "A 读会话"
[ "$(codeB "$BASE/api/sessions/e2e-sess-a")" = "404" ] && ok "B 读 A 的会话 → 404" || fail "B 读 A 会话应 404"
[ "$(codeB "$BASE/api/sessions/e2e-sess-a/stream?since=0")" = "404" ] && ok "B 订阅 A 的 SSE → 404" || fail "B SSE 应 404"
[ "$(codeB -H "X-CSRF-Token: ${CB}" -X POST "$BASE/api/sessions/e2e-sess-a/runs/${RID}/cancel")" = "404" ] && ok "B 取消 A 的 run → 404" || fail "B cancel 应 404"
[ "$(codeB -H "X-CSRF-Token: ${CB}" -X DELETE "$BASE/api/sessions/e2e-sess-a")" = "404" ] && ok "B 删除 A 的会话 → 404" || fail "B 删除应 404"
# A 自己的取消：Run 仍活跃（mock 1.5s）→ 200
[ "$(codeA -H "X-CSRF-Token: ${CA}" -X POST "$BASE/api/sessions/e2e-sess-a/runs/${RID}/cancel")" = "200" ] && ok "A 取消自己的 run → 200（run 仍活跃）" || fail "A cancel"
# 会话列表隔离：A 列表含 e2e-sess-a，B 列表不含
LA=$(bodyA "$BASE/api/sessions")
LB=$(bodyB "$BASE/api/sessions")
echo "$LA" | grep -q "e2e-sess-a" && ok "A 会话列表含 e2e-sess-a" || fail "A 列表"
echo "$LB" | grep -q "e2e-sess-a" && fail "B 会话列表串到 A" || ok "B 会话列表不含 A 会话"

echo "-- 7. 任务列表按租户过滤 --"
[ "$(codeA -H "Content-Type: application/json" -H "X-CSRF-Token: ${CA}" -X POST -d '{"type":"threat","title":"e2e-某集团威胁"}' "$BASE/api/tasks/consolidate")" = "202" ] && ok "A 提交 consolidate 任务（202）" || fail "A 提交任务"
sleep 1
TA=$(bodyA "$BASE/api/tasks")
TB=$(bodyB "$BASE/api/tasks")
echo "$TA" | grep -q '"type":"consolidate"' && ok "A 任务列表含 consolidate" || fail "A 任务列表"
echo "$TB" | grep -q "e2e-某集团威胁" && fail "B 任务列表串到 A" || ok "B 任务列表不含 A 任务"

echo "-- 8. 清理 --"
[ "$(codeA -H "X-CSRF-Token: ${CA}" -X DELETE "$BASE/api/sessions/e2e-sess-a")" = "200" ] && ok "A 删除自己的会话 → 200" || fail "A 清理会话"

echo ""
echo "=== 结果: ${PASS} passed, ${FAIL} failed ==="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
