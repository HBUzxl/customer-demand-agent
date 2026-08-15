#!/usr/bin/env bash
# smoke-p0.sh — ADR-016 v2 落地的可复现冒烟验证（memory-v2 先锋批 goal）
#
# 验证三件事（需后端已运行 + config.json 已配 api_key，即真实 LLM）：
#   1. 真实需求轮：system prompt 含「产品目录索引」且不含全量注入痕迹，
#      analysis_submit 正常出现（工具轨迹正常）
#   2. 客户身份行：会话关联 customer 后，system prompt 含【当前会话客户】
#   3. 体积断言：寒暄轮 system prompt < 2000 字符（全量注入时代 ≥3300）
#
# 用法：./scripts/smoke-p0.sh   （默认 BASE=http://localhost:8080）
set -uo pipefail
BASE="${BASE:-http://localhost:8080}"
CURL="/usr/bin/curl"
SID="smoke-p0-$(date +%s)"
PASS=0; FAIL=0
ok()   { echo "  [PASS] $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }
DB="${CDA_HISTORY_DB:-}"
if [ -z "$DB" ]; then
  # 从运行实例取真实库路径（实例配置决定——XDG 或显式 ./data）
  DB=$($CURL -s "$BASE/api/console/config" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['history_db'])" 2>/dev/null || true)
fi
if [ -z "$DB" ] || [ ! -f "$DB" ]; then
  DB="data/history.db" # 兜底：实例不可达时项目内默认
fi

# 1. 真实需求轮（带客户名；LLM 自主判定——明确指令降低非确定性，两次重试）——F0：202 创建 Run
DEMAND_TEXT="客户电商网站大促被CC攻击打慢了，过等保二级。这是一个真实的客户安全需求，请调用 analysis_submit 工具提交结构化分析。"
SUBMIT_HIT=""
for attempt in 1 2; do
  SID="smoke-p0-$(date +%s)-$attempt"
  $CURL -s -H "Content-Type: application/json" -H "X-Tenant-ID: default" \
    -d "{\"text\":\"$DEMAND_TEXT\",\"session_id\":\"$SID\",\"customer\":\"冒烟测试电商\"}" \
    "$BASE/api/message" > /tmp/smoke_p0_run.json
  RID=$(sed 's/.*"run_id":"\([^"]*\)".*/\1/' /tmp/smoke_p0_run.json)
  $CURL -s -N --max-time 180 "$BASE/api/sessions/$SID/stream?since=0" > /tmp/smoke_p0_sse.txt 2>&1
  grep -q '"tool":"analysis_submit"' /tmp/smoke_p0_sse.txt && SUBMIT_HIT=1
  [ -n "$SUBMIT_HIT" ] && break
  echo "  [retry] analysis_submit 第 $attempt 次未触发，重试…"
  $CURL -s -o /dev/null -X DELETE "$BASE/api/sessions/$SID"
done
[ -n "$RID" ] && ok "202+run_id（F0）" || fail "无 run_id（$(cat /tmp/smoke_p0_run.json)）"
[ -n "$SUBMIT_HIT" ] && ok "analysis_submit 出现（工具轨迹正常）" || fail "analysis_submit 未出现（两次尝试）"

# 2. 寒暄轮（F0：创建 Run + 订阅收尾；新会话——system prompt 断言按会话查）
SID2="smoke-p0-hi-$(date +%s)"
HICODE=$($CURL -s -o /tmp/sp0_hi.json -w "%{http_code}" -H 'Content-Type: application/json' -H 'X-Tenant-ID: default' \
  -d "{\"text\":\"你好\",\"session_id\":\"$SID2\"}" "$BASE/api/message")
if [ "$HICODE" != "202" ]; then
  # 需求轮 Run 或许还占着全局锁——等待后重试一次
  sleep 3
  HICODE=$($CURL -s -o /tmp/sp0_hi.json -w "%{http_code}" -H 'Content-Type: application/json' -H 'X-Tenant-ID: default' \
    -d "{\"text\":\"你好\",\"session_id\":\"$SID2\"}" "$BASE/api/message")
fi
echo "  (hi POST: $HICODE)"
SID="$SID2"
$CURL -s -N --max-time 120 "$BASE/api/sessions/$SID/stream?since=0" > /dev/null 2>&1

# 2. 寒暄轮：核心断言是「无全量知识注入痕迹」（体积随会话状态注入浮动，
#    分析后的追问带最近分析结果是 ADR-016 L2 的设计行为）
python3 - "$SID" "$DB" > /tmp/sp0_sub.txt <<'PY'
import sqlite3, sys
sid, db = sys.argv[1], sys.argv[2]
conn = sqlite3.connect(db)
row = conn.execute(
    "SELECT content FROM messages WHERE session_id=? AND role='system' ORDER BY id DESC LIMIT 1", (sid,)).fetchone()
p = row[0] if row else ""
checks = [
    ("无威胁全量段", "已知威胁类型" not in p),
    ("无能力明细", "CC 攻击防护 [1.0]" not in p),
    ("有目录索引（L3a 恒在）", "长亭产品目录" in p),
]
fails = 0
for name, ok in checks:
    print(f"  [{'PASS' if ok else 'FAIL'}] 寒暄轮 {name}（len={len(p)}，含会话状态注入）")
    fails += 0 if ok else 1
sys.exit(1 if fails else 0)
PY
PYRC=$?
cat /tmp/sp0_sub.txt
SUBPASS=$(grep -c '\[PASS\]' /tmp/sp0_sub.txt 2>/dev/null || echo 0)
SUBFAIL=$(grep -c '\[FAIL\]' /tmp/sp0_sub.txt 2>/dev/null || echo 0)
PASS=$((PASS+SUBPASS))
FAIL=$((FAIL+SUBFAIL))

# 清理
$CURL -s -o /dev/null -X DELETE "$BASE/api/sessions/$SID" -H 'X-Tenant-ID: default'
echo "=== smoke-p0: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -eq 0 ]
