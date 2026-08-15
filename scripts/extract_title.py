#!/usr/bin/env python3
"""从 SSE 流文件提取 memory_ensure/observe 的条目 title（嵌套 JSON 转义形态）。"""
import re
import sys

sys.stdout.reconfigure(encoding="utf-8")  # macOS shell locale 非 UTF-8 时防坏字节

raw = open(sys.argv[1], encoding="utf-8", errors="replace").read()
m = re.search(r'title\\?"\\?:\\?"([^"\\]+)', raw)
if not m:
    m = re.search(r'title\\?"\s*:\s*\\?"([^"\\]+)', raw)
print(m.group(1) if m else "")
