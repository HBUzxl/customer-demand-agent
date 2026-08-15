#!/usr/bin/env bash
# verify-all.sh —— 全套门禁一键复现（goal 终态验证的机械入口）。
# 用法：./scripts/verify-all.sh [--with-smoke]
#   --with-smoke 附加四脚本（需 :8080 运行实例 + 真实 api_key）
# 退出码：0=全绿；1=任何失败。
set -uo pipefail
cd "$(dirname "$0")/.."
ROOT="$(pwd)"
PASS=0; FAIL=0
ok()   { echo "  [PASS] $1"; PASS=$((PASS+1)); }
fail() { echo "  [FAIL] $1"; FAIL=$((FAIL+1)); }
run()  { # run <名称> <目录> <命令> [args...]——命令拆散传
  local name="$1" dir="$2" cmd="$3"; shift 3
  if (cd "$dir" && "$cmd" "$@" > /tmp/verify-$$-out.txt 2>&1); then
    ok "$name"
  else
    fail "${name}（尾 3 行：$(tail -3 /tmp/verify-$$-out.txt | tr '\n' ' ' | cut -c1-120)）"
  fi
}

echo "── Go 门禁（目录：仓库根）──"
run "go build"               "$ROOT" go build ./...
run "go vet"                 "$ROOT" go vet ./...
run "go test"                "$ROOT" go test ./...
run "golangci-lint（0 issue）" "$ROOT" golangci-lint run ./...
GOFMT_N=$(gofmt -l cmd/ internal/ | wc -l | tr -d ' ')
[ "$GOFMT_N" = "0" ] && ok "gofmt -l 空（$GOFMT_N 文件）" || fail "gofmt 未格式化 $GOFMT_N 文件"

echo "── 前端门禁（目录：frontend/）──"
run "tsc --noEmit"           "$ROOT/frontend" npx tsc --noEmit
run "npm run lint（0 errors）" "$ROOT/frontend" npm run lint
run "npm run format:check"  "$ROOT/frontend" npm run format:check
run "npm run build"         "$ROOT/frontend" npm run build

echo "── 冒烟脚本（需 :8080 实例；--with-smoke 启用）──"
if [ "${1:-}" = "--with-smoke" ]; then
  for s in e2e smoke-f0 smoke-p0 smoke-final; do
    if ./scripts/$s.sh > /tmp/verify-$$-$s.txt 2>&1; then
      ok "${s}（$(grep -oE '[0-9]+ passed, [0-9]+ failed' /tmp/verify-$$-$s.txt | tail -1)）"
    else
      fail "${s}（$(grep FAIL /tmp/verify-$$-$s.txt | head -1)）"
    fi
  done
else
  echo "  （跳过——加 --with-smoke 参数运行；需 :8080 实例（./cda-agent --config config.json）+ 真实 api_key）"
fi

echo "=== verify-all: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -eq 0 ]
