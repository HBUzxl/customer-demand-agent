#!/usr/bin/env bash
# render-ppt.sh —— 用 headless Chrome 逐页渲染 PPT 到 PNG（可复现验证）。
# 用法：./scripts/render-ppt.sh [页数] [输出目录]
set -euo pipefail
cd "$(dirname "$0")/.."
CHROME="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
if [ ! -x "$CHROME" ]; then
  echo "Chrome 不在默认路径，请安装或改脚本里的 CHROME 变量"; exit 1
fi
N="${1:-}"            # 空 = 渲染全部
OUT="${2:-/tmp/ppt-render}"
mkdir -p "$OUT"
TOTAL=$(grep -c '<section ' HTMLPPT/index.html)
if [ -z "$N" ]; then N="$TOTAL"; fi
for i in $(seq 1 "$N"); do
  "$CHROME" --headless --disable-gpu --screenshot="$OUT/slide-$i.png" \
    --window-size=1280,720 --virtual-time-budget=3000 \
    "file://$(pwd)/HTMLPPT/index.html#/$i" 2>/dev/null
done
echo "已渲染 $N/$TOTAL 页到 $OUT/"
