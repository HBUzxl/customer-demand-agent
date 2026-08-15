#!/usr/bin/env bash
# 启动后端服务（开发模式）。配置完全来自 config.json，不读环境变量。
set -euo pipefail
cd "$(dirname "$0")/.."

CONFIG="${CONFIG:-./config.json}"

# 播种已内置服务端（P9：数据根 wiki 为空时自动从 ./wiki 复制）
fi

# 首次运行：服务端自动播种 wiki/ 到数据根（P9；老 ./data 自动迁入、原位保留）
WIKI_DIR="$(grep -o '\"wiki_dir\"[^,]*' "$CONFIG" | head -1 | sed 's/.*: *\"//;s/\".*//' || true)"
WIKI_DIR="${WIKI_DIR:-./wiki}"
if [ ! -d "$WIKI_DIR" ] || [ -z "$(ls -A "$WIKI_DIR" 2>/dev/null)" ]; then
  if [ -d ./wiki ]; then
    mkdir -p "$WIKI_DIR"
    cp -r ./wiki/* "$WIKI_DIR/" 2>/dev/null || true
    echo "  已从 ./wiki 播种知识库到 $WIKI_DIR"
  fi
fi

echo "启动客户需求分析智能体（配置：$CONFIG）…"
exec go run ./cmd/agent --config "$CONFIG"
