#!/usr/bin/env bash
# smoke-final.sh —— 一期收尾新功能的真实冒烟（需 :8080 运行 + 真实 api_key）
# 覆盖：F3 ask_user 选项卡（SSE 事件）/ F4 内联审核（needs_review→approve）/
#       观测台（任务列表+LLM 审计非空）/ C3 外置 prompt 生效
set -euo pipefail
BASE="${BASE:-http://localhost:8080}"
CURL="/usr/bin/curl"
PASS=0; FAIL=0
ok()   { echo "  [PASS] $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }

# ── F3：ask_user 事件（诱导 Agent 提问——客户需求缺关键信息；LLM 非确定性→三次重试）──
ASK_HIT=""
for attempt in 1 2 3; do
  SID="sf-ask-$(date +%s)-$attempt"
  $CURL -s -H "Content-Type: application/json" \
    -d "{\"text\":\"客户要做安全建设，预算和部署环境都还没定，你必须先用 ask_user 工具向我提问这两个信息，拿到答案后才能继续分析。\",\"session_id\":\"$SID\"}" \
    "$BASE/api/message" > /tmp/sf_ask.json
  RID=$(sed 's/.*"run_id":"\([^"]*\)".*/\1/' /tmp/sf_ask.json || true)
  STREAM=$($CURL -s -N --max-time 120 "$BASE/api/sessions/$SID/stream?since=0" 2>/dev/null || true)
  if echo "$STREAM" | grep -q '"type":"ask_user"'; then ASK_HIT=1; fi
  $CURL -s -o /dev/null -X POST "$BASE/api/sessions/$SID/runs/$RID/cancel"
  $CURL -s -o /dev/null -X DELETE "$BASE/api/sessions/$SID"
  [ -n "$ASK_HIT" ] && break
  echo "  [retry] F3 第 $attempt 次未触发 ask_user，重试…"
done
[ -n "$ASK_HIT" ] && ok "F3 ask_user 事件出现" || fail "F3 无 ask_user 事件（三次尝试）"

# ── F4：固定条目走完 needs_review→approve 全链（200 硬断言）+ Agent 事件冒烟 ──
SID2="sf-rv-$(date +%s)"
RV_HIT=""
for attempt in 1 2 3; do
  SID2="sf-rv-$(date +%s)-$attempt"
  $CURL -s -H "Content-Type: application/json" \
    -d "{\"text\":\"请用 memory_observe 把这个新趋势记到行业记忆（industry）：最近金融客户特别关注AI大模型的数据安全合规，银行问过训练数据出境问题。直接调用工具记录，不要只口头说记了。\",\"session_id\":\"$SID2\"}" \
    "$BASE/api/message" > /tmp/sf_rv.json
  { $CURL -s -N --max-time 150 "$BASE/api/sessions/$SID2/stream?since=0" > /tmp/sf_rv_stream.txt 2>/dev/null || true; }
  RV_HIT=$(grep -oE '\\?"needs_review\\?":\\?true' /tmp/sf_rv_stream.txt | head -1 || true)
  $CURL -s -o /dev/null -X DELETE "$BASE/api/sessions/$SID2"
  [ -n "$RV_HIT" ] && break
  echo "  [retry] F4 第 $attempt 次未触发工具调用，重试…"
done
[ -n "$RV_HIT" ] && ok "F4 Agent 写入 needs_review=true 事件出现" || fail "F4 无 needs_review（三次尝试）"

# 审批链：抓 Agent 真实创建的条目 title → approve 硬断言 200 → 状态 verified → 清理
RV_TITLE=$(python3 "$(dirname "$0")/extract_title.py" /tmp/sf_rv_stream.txt)
if [ -z "$RV_TITLE" ]; then RV_TITLE="金融客户关注AI大模型数据安全合规"; fi
CODE=$($CURL -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/review/industry/$(python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))" "$RV_TITLE")/approve")
[ "$CODE" = "200" ] && ok "F4 审批 approve → 200（条目: ${RV_TITLE}）" || fail "F4 审批应 200，got $CODE"
ST=$($CURL -s "$BASE/api/memory/industry/$(python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))" "$RV_TITLE")" | grep -o '"status":"[a-z_]*"' | head -1)
echo "$ST" | grep -q 'verified' && ok "F4 审批后状态 verified" || fail "F4 审批后状态 $ST"
$CURL -s -o /dev/null -X DELETE "$BASE/api/memory/industry/$(python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))" "$RV_TITLE")?archive=false"

# ── 观测台 ──
TASKS=$($CURL -s "$BASE/api/tasks")
echo "$TASKS" | grep -q '"type"' && ok "观测台：任务列表有记录" || fail "观测台任务列表空"
AUDIT=$($CURL -s "$BASE/api/platform/llm-audit")
echo "$AUDIT" | grep -q '"model"' && ok "观测台：LLM 审计非空" || fail "LLM 审计空"
LOGS=$($CURL -s -N --max-time 2 "$BASE/api/platform/logs" 2>/dev/null | head -c 200 || true)
echo "$LOGS" | grep -q 'data:' && ok "观测台：日志 tail 出流" || fail "日志 tail"

# ── C3：外置 prompt 生效 ──
PROMPTS=$($CURL -s "$BASE/api/console/prompts")
echo "$PROMPTS" | grep -q 'template_raw' && echo "$PROMPTS" | grep -q '长亭科技' && ok "C3 外置模板 API（原文返回）" || fail "C3 模板 API"

echo "=== smoke-final: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -eq 0 ]
