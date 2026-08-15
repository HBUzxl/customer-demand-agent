#!/usr/bin/env bash
# verify-plan-commits.sh —— 「每 plan 独立 commit」机械复核：
# 每个功能 plan 检索其主 commit 并验证 diff 涉及对应模块文件。
set -uo pipefail
cd "$(dirname "$0")/.."
PASS=0; FAIL=0
check() { # check <plan> <主commit> <diff须含的关键词>
  local plan="$1" commit="$2" kw="$3"
  local files
  files=$(git show --format= --name-only "$commit" 2>/dev/null; git show --format= "$commit" 2>/dev/null | grep -o "$kw" | head -1)
  if [ -n "$files" ] && echo "$files" | grep -q "$kw"; then
    echo "  [PASS] ${plan} → ${commit}（含 ${kw}）"
    PASS=$((PASS+1))
  else
    echo "  [FAIL] ${plan} → ${commit}（diff 未含 ${kw}）"
    FAIL=$((FAIL+1))
  fi
}
check ui-shared          3bd2c2b "ConfirmDialog"
check session-batch      583c233 "History.tsx"
check session-search     34608f1 "SearchSessions"
check customer-bind      b4b9c44 "session_bind_customer"
check memory-batch       4ef7de0 "MemoryList.tsx"
check memory-search-ia   61b9475 "search-meta"
check wiki-hygiene       79d2dc2 "existing_hint"
check sidebar-models-ux  b0c2293 "conv-dots"
check review-gating      dcc0641 "visibleToAgent"
check console-config     1433aae "AgentMaxIterations"
check checkpoint-tree    34b030c "BranchAfter"
check ui-copy-cleanup    59c834c "ResultCard"
check verify-close       8cdc157 "Done"
echo "=== verify-plan-commits: ${PASS} passed, ${FAIL} failed ==="
[ "$FAIL" -eq 0 ]
