#!/usr/bin/env bash
# smoke-f0.sh — F0+F1（服务端 Run 任务模型）可复现冒烟
#
# 验证五件事（需 :8080 运行 + config.json 已配 api_key 即真实 LLM）：
#   1. 202+run_id：POST /api/message 立即返回
#   2. 断连运行继续：POST 后不订阅，稍后 running=true（执行仍在后台）
#   3. replay 续传：订阅 since=0 拿到已发生事件（session 等）
#   4. cancel 生效：cancel 后 running 解除（5s 内）
#   5. truncate 生效：截断后消息清空、可重新发消息
#
# 用法：./scripts/smoke-f0.sh   （默认 BASE=http://localhost:8080）
set -uo pipefail
BASE="${BASE:-http://localhost:8080}"
CURL="/usr/bin/curl"
SID="smoke-f0-$(date +%s)"
PASS=0; FAIL=0
ok()   { echo "  [PASS] $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }

# 1. 202 + run_id（立即返回，不等执行；显式校验 HTTP 状态码）
CODE=$("$CURL" -s -o /tmp/smoke_f0_resp.json -w "%{http_code}" -H "Content-Type: application/json" \
  -d "{\"text\":\"客户电商网站被CC攻击，过等保二级，预算50万，请给出完整的产品组合方案和理由\",\"session_id\":\"$SID\",\"customer\":\"冒烟F0客户\"}" \
  "$BASE/api/message")
[ "$CODE" = "202" ] && ok "① HTTP 202" || fail "① 状态码 ${CODE} 非 202"
RESP=$(cat /tmp/smoke_f0_resp.json)
RID=$(echo "$RESP" | sed 's/.*"run_id":"\([^"]*\)".*/\1/')
[ -n "$RID" ] && [ "$RID" != "$RESP" ] && ok "① run_id（${RID}）" || fail "① 无 run_id（${RESP}）"

# 2. 断连运行继续：POST 已返回（连接已关）。轮询 running（最多 10s）——
#    在 Run 生命周期内观察到一次 true 即证明断连后执行仍在；若首轮即 false
#    则 Run 已结束（超快完成），改验 replay 中有执行事件作为降级证明。
SAW_RUNNING=0
for i in $(seq 1 20); do
  R=$("$CURL" -s "$BASE/api/sessions/$SID/running" | grep -o '"running":[a-z]*' | head -1)
  echo "$R" | grep -q true && { SAW_RUNNING=1; break; }
  echo "$R" | grep -q false && break
  sleep 0.5
done
if [ "$SAW_RUNNING" = "1" ]; then
  ok "② 断连后运行继续（观察到 running=true）"
else
  STUB=$("$CURL" -s -N --max-time 2 "$BASE/api/sessions/$SID/stream?since=0" | head -c 300)
  echo "$STUB" | grep -qE '"type":"(round|tool_call|reasoning)"' \
    && ok "② Run 极速完成（降级验证：replay 有执行事件）" \
    || fail "② 既未观察到 running 也无执行事件"
fi

# 3. replay 续传：订阅 since=0（读 2 秒），应见 session/工具事件
STREAM=$("$CURL" -s -N --max-time 2 "$BASE/api/sessions/$SID/stream?since=0" 2>/dev/null | head -c 600)
echo "$STREAM" | grep -q '"type":"session"' && ok "③ replay 含 session 事件" || fail "③ replay 无 session（${STREAM:0:60}）"
echo "$STREAM" | grep -qE '"type":"(round|reasoning|tool_call|content)"' && ok "③ replay 含执行事件" || fail "③ replay 无执行事件"
echo "$STREAM" | grep -q '^id:' && ok "③ SSE 带 id: 游标（可续传）" || fail "③ SSE 无 id: 游标"
# 非零游标增量续传：取已见最大 seq，since=它 → 只收更大 id
MAXSEQ=$(echo "$STREAM" | grep '^id:' | tail -1 | tr -dc '0-9')
if [ -n "$MAXSEQ" ]; then
  INC=$("$CURL" -s -N --max-time 2 "$BASE/api/sessions/$SID/stream?since=$MAXSEQ" 2>/dev/null | head -c 400)
  DUP=$(echo "$INC" | grep '^id:' | awk -F': *' -v m="$MAXSEQ" '$2 <= m' | head -1)
  [ -z "$DUP" ] && ok "③ since=${MAXSEQ} 增量续传无重复" || fail "③ 续传收到重复事件（${DUP}）"
else
  fail "③ 无法提取游标"
fi

# 4. cancel 生效
"$CURL" -s -o /dev/null -X POST "$BASE/api/sessions/$SID/runs/$RID/cancel"
CANCELLED=1
for i in $(seq 1 10); do
  R=$("$CURL" -s "$BASE/api/sessions/$SID/running" | grep -o '"running":[a-z]*' | head -1)
  echo "$R" | grep -q false && { CANCELLED=0; break; }
  sleep 0.5
done
[ "$CANCELLED" = "0" ] && ok "④ cancel 后 5s 内 running 解除" || fail "④ cancel 未解除运行态"

# 5. truncate 生效：等 Run 完全退出（cancel 后 goroutine 收尾）再截断
IDLE=1
for i in $(seq 1 10); do
  R=$("$CURL" -s "$BASE/api/sessions/$SID/running" | grep -o '"running":[a-z]*' | head -1)
  echo "$R" | grep -q false && { IDLE=0; break; }
  sleep 0.5
done
sleep 0.5
TRESP=$("$CURL" -s -X DELETE "$BASE/api/sessions/$SID/messages/after?seq=1")
echo "$TRESP" | grep -q '"status":"ok"' && ok "⑤ truncate 返回 ok" || fail "⑤ truncate（${TRESP}）"
MRESP=$("$CURL" -s "$BASE/api/sessions/$SID")
MCOUNT=$(echo "$MRESP" | python3 -c "import json,sys;print(len(json.load(sys.stdin).get('messages') or []))" 2>/dev/null || echo -1)
[ "$MCOUNT" = "0" ] && ok "⑤ 截断后消息清空（0 条）" || fail "⑤ 截断后仍有 $MCOUNT 条"
# 截断后可重新发消息（Run 解除占用）
R2=$("$CURL" -s -H "Content-Type: application/json" -d "{\"text\":\"你好\",\"session_id\":\"$SID\"}" "$BASE/api/message")
echo "$R2" | grep -q run_id && ok "⑤ 截断后可重发（新 Run）" || fail "⑤ 重发失败（${R2}）"
R2ID=$(echo "$R2" | sed 's/.*"run_id":"\([^"]*\)".*/\1/')
[ -n "$R2ID" ] && "$CURL" -s -o /dev/null -X POST "$BASE/api/sessions/$SID/runs/$R2ID/cancel"

# 清理
"$CURL" -s -o /dev/null -X DELETE "$BASE/api/sessions/$SID"
echo "=== smoke-f0: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -eq 0 ]
