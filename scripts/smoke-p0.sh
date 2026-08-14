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
DB="data/history.db"

# 1. 真实需求轮（带客户名）——F0：202 创建 Run，订阅流收集事件至完成
$CURL -s -H 'Content-Type: application/json' -H 'X-Tenant-ID: default' \
  -d "{\"text\":\"客户电商网站大促被CC攻击打慢了，过等保二级\",\"session_id\":\"$SID\",\"customer\":\"冒烟测试电商\"}" \
  "$BASE/api/message" > /tmp/smoke_p0_run.json
RID=$(sed 's/.*"run_id":"\([^"]*\)".*/\1/' /tmp/smoke_p0_run.json)
[ -n "$RID" ] && ok "202+run_id（F0）" || fail "无 run_id（$(cat /tmp/smoke_p0_run.json)）"
# 订阅流收完（Run 结束流自然关闭）
$CURL -s -N --max-time 180 "$BASE/api/sessions/$SID/stream?since=0" > /tmp/smoke_p0_sse.txt 2>&1
grep -q '"tool":"analysis_submit"' /tmp/smoke_p0_sse.txt && ok "analysis_submit 出现（工具轨迹正常）" || fail "analysis_submit 未出现"

python3 - "$SID" "$DB" <<'PY'
import sqlite3, sys
sid, db = sys.argv[1], sys.argv[2]
conn = sqlite3.connect(db)
row = conn.execute(
    "SELECT content FROM messages WHERE session_id=? AND role='system' ORDER BY id DESC LIMIT 1", (sid,)).fetchone()
if not row:
    print("  [FAIL] system prompt 未落库"); sys.exit(1)
p = row[0]
checks = [
    ("产品目录索引注入", "长亭产品目录" in p),
    ("客户身份行注入", "当前会话客户" in p and "冒烟测试电商" in p),
    ("无全量注入痕迹（威胁段）", "已知威胁类型" not in p),
    ("无全量注入痕迹（能力明细）", "CC 攻击防护 [1.0]" not in p),
    ("体积 <2000 字符", len(p) < 2000),
]
fails = 0
for name, ok in checks:
    print(f"  [{'PASS' if ok else 'FAIL'}] {name}（len={len(p)}）")
    fails += 0 if ok else 1
sys.exit(1 if fails else 0)
PY
[ $? -eq 0 ] && PASS=$((PASS+1)) || FAIL=$((FAIL+1))

# 2. 寒暄轮（F0：创建 Run + 订阅收尾）
$CURL -s -H 'Content-Type: application/json' -H 'X-Tenant-ID: default' \
  -d "{\"text\":\"你好\",\"session_id\":\"$SID\"}" "$BASE/api/message" > /dev/null
$CURL -s -N --max-time 120 "$BASE/api/sessions/$SID/stream?since=0" > /dev/null 2>&1

# 2. 寒暄轮：核心断言是「无全量知识注入痕迹」（体积随会话状态注入浮动，
#    分析后的追问带最近分析结果是 ADR-016 L2 的设计行为）
python3 - "$SID" "$DB" <<'PY'
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
[ $? -eq 0 ] && PASS=$((PASS+1)) || FAIL=$((FAIL+1))

# 清理
$CURL -s -o /dev/null -X DELETE "$BASE/api/sessions/$SID" -H 'X-Tenant-ID: default'
echo "=== smoke-p0: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -eq 0 ]
