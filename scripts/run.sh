#!/usr/bin/env bash
# 启动后端服务（开发模式）。配置完全来自 config.json，不读环境变量。
set -euo pipefail
cd "$(dirname "$0")/.."

CONFIG="${CONFIG:-./config.json}"

# 首次运行：若 config.json 不存在，从模板复制一份
if [ ! -f "$CONFIG" ]; then
  if [ -f ./config.example.json ]; then
    cp ./config.example.json "$CONFIG"
    echo "  已从 config.example.json 创建 $CONFIG，请编辑后填入 api_key"
  fi
fi

# 首次运行：将提交的种子知识库（wiki/）复制到运行时目录（data/wiki/）
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
