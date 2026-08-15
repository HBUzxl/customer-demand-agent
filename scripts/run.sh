#!/usr/bin/env bash
# 启动后端服务（开发模式）。业务配置来自 config.json；数据根可用 CDA_DATA_DIR 指定（P9）。
set -euo pipefail
cd "$(dirname "$0")/.."

CONFIG="${CONFIG:-./config.json}"

# 播种已内置服务端（P9：数据根 wiki 为空时自动播种；老 ./data 自动迁入、原位保留）。

echo "启动客户需求分析智能体（配置：${CONFIG}）…"
exec go run ./cmd/agent --config "$CONFIG"
