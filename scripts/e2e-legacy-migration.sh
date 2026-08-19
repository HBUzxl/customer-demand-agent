#!/usr/bin/env bash
# 存量数据自动迁入 Legacy 租户 + -setup 引导端到端冒烟（scripts/e2e-legacy-migration.sh）
#
# 用法：bash scripts/e2e-legacy-migration.sh
#   - 自建临时 CDA_DATA_DIR + 伪造旧单租户数据（history.db + wiki），不依赖外部服务；
#   - 验证设计文档 §16 验收「旧 ./data 首次启动自动迁入 Legacy 租户，
#     -setup 建管理员后可查旧会话」的完整链路：
#       旧库启动 → 自动建 Legacy 租户 + 迁会话/客户记忆 → 产品记忆进系统基线；
#       -setup（pty 驱动）建平台管理员 + owner membership；
#       管理员登录 → /api/auth/me 返回 Legacy 租户 + owner/platform_admin 角色 →
#       会话列表与详情可见旧会话与旧消息；平台路由可访问。
#   - 数据目录结构断言（§6）：control/identity.db、tenants/<uuid>/history.db +
#     tenant.json + 租户 wiki（客户记忆）、system/wiki（产品记忆）。
#
# 环境变量：APP_PORT（默认 18190）。Mock LLM 不必要（不发起分析请求）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d /tmp/cda-legacy-e2e.XXXXXX)"
APP_PORT="${APP_PORT:-18190}"
BASE="http://127.0.0.1:${APP_PORT}"
BIN="${WORK}/agent"
CFG="${WORK}/config.json"
DATA="${WORK}/data"
JAR="${WORK}/jar.txt"
EMAIL="admin@mt.test"
PASS="Password-123"
FIXTURE="${ROOT}/.tmp_legacy_fixture" # 模块内临时 Go 工具（构造旧库，用后即删）

PASS_N=0
FAIL_N=0
ok()   { echo "  [PASS] $1"; PASS_N=$((PASS_N+1)); }
fail() { echo "  [FAIL] $1"; FAIL_N=$((FAIL_N+1)); }

AGENT_PID=""
cleanup() {
  [ -n "$AGENT_PID" ] && kill "$AGENT_PID" 2>/dev/null || true
  rm -rf "$WORK" "$FIXTURE"
}
trap cleanup EXIT

echo "=== 存量数据迁移 + -setup 引导 E2E（base=${BASE}）==="

echo "-- 0. 构建 + 伪造旧单租户数据（history.db + wiki）--"
( cd "$ROOT" && go build -o "$BIN" ./cmd/agent )

mkdir -p "$ROOT/.tmp_legacy_fixture"
cat > "$ROOT/.tmp_legacy_fixture/main.go" <<'GO'
// 临时工具：构造旧单租户数据（history.db + wiki），供 e2e-legacy-migration.sh 冒烟。
// 用后即删（脚本 cleanup）。构建方式与 Go 单测一致：history.Open 建真实 SQLite。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"customer-demand-agent/internal/history"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: fixture <historyDB> <wikiDir>")
		os.Exit(2)
	}
	dbPath, wikiDir := os.Args[1], os.Args[2]
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		panic(err)
	}
	h, err := history.Open(dbPath)
	if err != nil {
		panic(err)
	}
	if err := h.EnsureSession("", "sess_legacy", "旧会话标题", ""); err != nil {
		panic(err)
	}
	if _, err := h.AppendMessage("sess_legacy", "user", "这是一条旧会话消息", ""); err != nil {
		panic(err)
	}
	if err := h.Close(); err != nil {
		panic(err)
	}
	write := func(rel, content string) {
		p := filepath.Join(wikiDir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			panic(err)
		}
	}
	write("产品记忆/安全平台/雷池.md", "---\ntype: product\ntitle: 雷池\n---\n\n雷池 WAF 产品正文")
	write("用户记忆/客户/某集团.md", "---\ntype: customer\ntitle: 某集团\n---\n\n某集团客户画像")
	fmt.Println("fixture ok")
}
GO
( cd "$ROOT" && go run ./.tmp_legacy_fixture "$DATA/history.db" "$DATA/wiki" )

cat > "$CFG" <<EOF
{
  "server": {"addr": "127.0.0.1:${APP_PORT}", "frontend_dist": ""},
  "data_dir": "${DATA}",
  "models": [
    {"name":"mock","endpoint":"http://127.0.0.1:19999/v1/chat/completions","api_key":"test-key","protocol":"openai-chat","model":"mock"}
  ],
  "router": {"default":"mock","routes":{},"fallback":{"max_retries":1,"backoff_base_ms":50,"chain":[]}},
  "agent_max_iterations": 3,
  "registration_enabled": true,
  "secure_cookies": false,
  "jiying": {"enabled": false},
  "lead_manager": {"enabled": false}
}
EOF
[ -f "$DATA/history.db" ] && [ -f "$DATA/wiki/用户记忆/客户/某集团.md" ] \
  && ok "伪造旧库就绪（history.db + wiki 客户记忆）" || fail "旧库伪造失败"

echo "-- 1. -setup 引导（pty 驱动：term.ReadPassword 需 tty）--"
# -setup 用 golang.org/x/term 不回显读密码（非 tty 直接报错），故用 pty 驱动：
# 先写邮箱、再写密码（间隔让 bufio 不吞两行）。在 $WORK 下运行保证种子隔离。
LOG="$WORK/setup1.log"
LOG2="$WORK/setup2.log"
setup_pty() { # setup_pty <outlog> —— pty 驱动跑一次 -setup，输出落 <outlog>
  local out="$1"
  ( cd "$WORK" && python3 - "$BIN" "$CFG" "$EMAIL" "$PASS" > "$out" 2>&1 <<'PY'
import os, pty, subprocess, time, select, sys
bin_path, cfg, email, pw = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
master, slave = pty.openpty()
p = subprocess.Popen([bin_path, "-config", cfg, "-setup"],
                     stdin=slave, stdout=slave, stderr=slave, close_fds=True)
os.close(slave)
def write(s):
    os.write(master, s.encode())
    time.sleep(0.5)
def drain(timeout=8):
    out = b""; end = time.time() + timeout
    while time.time() < end:
        r, _, _ = select.select([master], [], [], 0.2)
        if r:
            try:
                d = os.read(master, 4096)
            except OSError:
                break
            if not d:
                break
            out += d
        if p.poll() is not None:
            break
    return out.decode(errors="replace")
write(email + "\n")
write(pw + "\n")
sys.stdout.write(drain())
sys.exit(p.wait())
PY
  )
}
setup_pty "$LOG"
grep -q "已创建管理员账号" "$LOG" && ok "-setup 创建平台管理员 ${EMAIL}" || fail "-setup 建管理员"
grep -q "已加入 Legacy 租户" "$LOG" && ok "-setup 管理员加入 Legacy 租户（owner）" || fail "-setup 成员关系"
grep -q "存量数据已迁入 Legacy 租户" "$LOG" && ok "旧库自动迁入 Legacy 租户（启动引导）" || fail "自动迁移"
# 幂等：重跑复用账号、不覆盖密码、rc=0
setup_pty "$LOG2"
grep -q "复用已有管理员账号" "$LOG2" && ok "-setup 幂等（重跑复用账号不覆盖密码）" || fail "-setup 幂等"

echo "-- 2. 数据目录结构断言（§6）--"
[ -f "$DATA/control/identity.db" ] && ok "中央身份库 control/identity.db" || fail "identity.db"
TID=""
for d in "$DATA/tenants"/*/; do TID="$(basename "$d")"; done
[ -n "$TID" ] && [ -f "$DATA/tenants/$TID/history.db" ] && ok "租户历史库 tenants/${TID}/history.db" || fail "租户 history.db"
[ -f "$DATA/tenants/$TID/tenant.json" ] && ok "tenants/${TID}/tenant.json" || fail "tenant.json"
[ -f "$DATA/tenants/$TID/wiki/用户记忆/客户/某集团.md" ] && ok "客户记忆迁入租户 wiki" || fail "客户记忆迁移"
grep -q "某集团客户画像" "$DATA/tenants/$TID/wiki/用户记忆/客户/某集团.md" && ok "客户画像内容保留" || fail "客户内容"
[ -f "$DATA/system/wiki/产品记忆/安全平台/雷池.md" ] && ok "产品记忆进系统基线 system/wiki" || fail "产品记忆基线"
[ ! -e "$DATA/system/wiki/用户记忆/客户/某集团.md" ] && ok "系统层不携带客户记忆（租户私密）" || fail "系统层泄漏客户记忆"

echo "-- 3. 启动 + 管理员登录 --"
( cd "$WORK" && CDA_DATA_DIR="$DATA" "$BIN" -config "$CFG" > "$WORK/server.log" 2>&1 ) &
AGENT_PID=$!
for _ in $(seq 1 60); do
  curl -s -o /dev/null "$BASE/api/health" && break
  sleep 0.25
done
[ "$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/health")" = "200" ] && ok "健康检查（公开）" || { fail "健康检查未就绪"; cat "$WORK/server.log"; exit 1; }

LOGIN=$(curl -s -c "$JAR" -b "$JAR" -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASS}\"}" "$BASE/api/auth/login")
echo "$LOGIN" | grep -q '"slug":"legacy"' && ok "登录返回 Legacy 租户（slug=legacy）" || fail "登录 slug"
echo "$LOGIN" | grep -q '"roles":\["owner","platform_admin"\]' && ok "登录角色 owner + platform_admin" || fail "登录角色"
ME=$(curl -s -b "$JAR" -c "$JAR" "$BASE/api/auth/me")
echo "$ME" | grep -q '"tenant_name":"Legacy 存量工作空间"' && ok "me 返回 Legacy 存量工作空间" || fail "me 租户名"
[ "$(curl -s -b "$JAR" "$BASE/api/platform/config" -o /dev/null -w "%{http_code}")" = "200" ] && ok "平台管理员访问 /api/platform/config → 200" || fail "平台 config"

echo "-- 4. 旧会话可查（§16 验收）--"
SESS=$(curl -s -b "$JAR" "$BASE/api/sessions")
echo "$SESS" | grep -q "sess_legacy" && ok "会话列表含旧会话 sess_legacy" || fail "会话列表"
echo "$SESS" | grep -q "旧会话标题" && ok "旧会话标题保留" || fail "旧标题"
DET=$(curl -s -b "$JAR" "$BASE/api/sessions/sess_legacy")
echo "$DET" | grep -q "这是一条旧会话消息" && ok "旧会话消息可读" || fail "旧消息"

echo ""
echo "=== 结果: ${PASS_N} passed, ${FAIL_N} failed ==="
if [ "$FAIL_N" -gt 0 ]; then
  exit 1
fi
exit 0
