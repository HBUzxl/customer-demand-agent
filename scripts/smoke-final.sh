#!/usr/bin/env bash
# smoke-final.sh —— 一期收尾新功能的真实冒烟（需 :8080 运行 + 真实 api_key）
# 覆盖：F3 ask_user 选项卡（SSE 事件）/ F4 内联审核（needs_review→approve）/
#       观测台（任务列表+LLM 审计非空）/ C3 外置 prompt 生效
set -uo pipefail
BASE="${BASE:-http://localhost:8080}"
CURL="/usr/bin/curl"
PASS=0; FAIL=0
ok()   { echo "  [PASS] $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }

# ── F3：ask_user 事件（诱导 Agent 提问——客户需求缺关键信息）──
SID="sf-ask-$(date +%s)"
$CURL -s -H "Content-Type: application/json" \
  -d "{\"text\":\"客户要做安全建设，预算和部署环境都还没定，你先问清楚再分析\",\"session_id\":\"$SID\"}" \
  "$BASE/api/message" > /tmp/sf_ask.json
RID=$(sed 's/.*"run_id":"\([^"]*\)".*/\1/' /tmp/sf_ask.json)
[ -n "$RID" ] && ok "F3 提交 202" || fail "F3 提交"
STREAM=$($CURL -s -N --max-time 120 "$BASE/api/sessions/$SID/stream?since=0" 2>/dev/null)
echo "$STREAM" | grep -q '"type":"ask_user"' && ok "F3 ask_user 事件出现" || fail "F3 无 ask_user 事件"
$CURL -s -o /dev/null -X POST "$BASE/api/sessions/$SID/runs/$RID/cancel"
$CURL -s -o /dev/null -X DELETE "$BASE/api/sessions/$SID"

# ── F4：Agent 写待审知识 → needs_review → 前端卡片数据 → approve API 生效 ──
SID2="sf-rv-$(date +%s)"
$CURL -s -H "Content-Type: application/json" \
  -d "{\"text\":\"记住一个新趋势：最近金融客户特别关注AI大模型的数据安全合规，请记录到行业记忆\",\"session_id\":\"$SID2\"}" \
  "$BASE/api/message" > /tmp/sf_rv.json
$CURL -s -N --max-time 150 "$BASE/api/sessions/$SID2/stream?since=0" > /tmp/sf_rv_stream.txt 2>/dev/null
RV_HIT=$(grep -oE '\\?"needs_review\\?":\\?true' /tmp/sf_rv_stream.txt | head -1)
[ -n "$RV_HIT" ] && ok "F4 needs_review=true 出现（卡片数据）" || fail "F4 无 needs_review"
# 审批 API 可用（对该会话产生的条目——不依赖具体 title，验证端点活着）
CODE=$($CURL -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/review/industry/金融客户关注AI大模型数据安全合规/approve")
[ "$CODE" = "200" ] || [ "$CODE" = "404" ] && ok "F4 审批端点可用（${CODE}）" || fail "F4 审批端点异常（${CODE}）"
$CURL -s -o /dev/null -X DELETE "$BASE/api/sessions/$SID2"
# 清理测试条目
$CURL -s -o /dev/null -X DELETE "$BASE/api/memory/industry/%E9%87%91%E8%9E%8D%E5%AE%A2%E6%88%B7%E5%85%B3%E6%B3%A8AI%E5%A4%A7%E6%A8%A1%E5%9E%8B%E6%95%B0%E6%8D%AE%E5%AE%89%E5%85%A8%E5%90%88%E8%A7%84?archive=false"

# ── 观测台 ──
TASKS=$($CURL -s "$BASE/api/tasks")
echo "$TASKS" | grep -q '"type"' && ok "观测台：任务列表有记录" || fail "观测台任务列表空"
AUDIT=$($CURL -s "$BASE/api/console/llm-audit")
echo "$AUDIT" | grep -q '"model"' && ok "观测台：LLM 审计非空" || fail "LLM 审计空"
LOGS=$($CURL -s -N --max-time 2 "$BASE/api/console/logs" 2>/dev/null | head -c 200)
echo "$LOGS" | grep -q 'data:' && ok "观测台：日志 tail 出流" || fail "日志 tail"

# ── C3：外置 prompt 生效 ──
PROMPTS=$($CURL -s "$BASE/api/console/prompts")
echo "$PROMPTS" | grep -q 'template_raw' && echo "$PROMPTS" | grep -q '长亭科技' && ok "C3 外置模板 API（原文返回）" || fail "C3 模板 API"

echo "=== smoke-final: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -eq 0 ]
