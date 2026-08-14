#!/usr/bin/env python3
"""把 chaitin-products skill 的文档导入到本项目的运行时 wiki（data/wiki/产品记忆/）。

格式：每个 .md 转为一条 product 类型记忆，带 product 字段关联产品名。
"""
import os
import re
import sys
import unicodedata

try:
    import yaml
except ImportError:
    yaml = None

SKILL = os.path.expanduser("~/.pi/agent/skills/products/chaitin-products/template")
OUT = "data/wiki/产品记忆"

PRODUCTS = ["万象", "云图", "守元", "洞鉴"]
# 跳过 readme（文件夹索引页）与结构文件
SKIP_NAMES = {"readme.md", "_file_tree.json", "nav.yaml"}
SKIP_DIRS = {"归档", "PRD", "summary", "images", "assets", "图片附件"}


def split_frontmatter(raw):
    """返回 (frontmatter_str, body) 或 (None, None)。"""
    raw = raw.strip()
    if not raw.startswith("---"):
        return None, None
    rest = raw[3:].lstrip("\r\n")
    m = re.match(r"\n---", "\n" + rest)
    idx = rest.find("\n---")
    if idx < 0:
        return None, None
    fm = rest[:idx]
    body = rest[idx + 4:].lstrip("\r\n")
    return fm, body


def parse_fm(fm):
    """从 frontmatter 里抽 title/summary/tags/visibility（简易 YAML）。"""
    out = {"title": "", "summary": "", "tags": [], "visibility": "external"}
    title = re.search(r"^title:\s*(.+)$", fm, re.M)
    if title:
        out["title"] = title.group(1).strip().strip("'\"")
    summary = re.search(r"^summary:\s*(.+)$", fm, re.M)
    if summary:
        out["summary"] = summary.group(1).strip().strip("'\"")
    vis = re.search(r"^visibility:\s*(\w+)$", fm, re.M)
    if vis:
        out["visibility"] = vis.group(1)
    # tags: 列表形式
    in_tags = False
    for line in fm.splitlines():
        if line.strip() == "tags:":
            in_tags = True
            continue
        if in_tags:
            if re.match(r"^\s*-\s*", line):
                t = re.sub(r"^\s*-\s*", "", line).strip().strip("'\"")
                if t:
                    out["tags"].append(t)
            elif re.match(r"^\S", line):
                break
    # 去重保序
    seen = set()
    tags = []
    for t in out["tags"]:
        if t not in seen:
            seen.add(t)
            tags.append(t)
    out["tags"] = tags
    return out


def clean(s):
    """清洗标题：去编号前缀、危险字符。"""
    s = re.sub(r"^\d+[\.\、\-_\s]+", "", s)
    for ch in r'\\/:*?"<>|#':
        s = s.replace(ch, "-")
    s = re.sub(r"\s+", " ", s).strip()
    return s


def sanitize_filename(s):
    s = clean(s)
    # 控制字符
    return "".join(ch for ch in s if unicodedata.category(ch)[0] != "C")


def main():
    os.makedirs(OUT, exist_ok=True)
    total = 0
    skipped = 0
    for product in PRODUCTS:
        base = os.path.join(SKILL, product)
        if not os.path.isdir(base):
            continue
        for root, dirs, files in os.walk(base):
            # 跳过隐藏/归档目录
            dirs[:] = [d for d in dirs if d not in SKIP_DIRS and not d.startswith(".")]
            for f in files:
                if not f.endswith(".md"):
                    continue
                if f in SKIP_NAMES:
                    continue
                path = os.path.join(root, f)
                try:
                    raw = open(path, encoding="utf-8").read()
                except Exception:
                    skipped += 1
                    continue
                fm, body = split_frontmatter(raw)
                if fm is None:
                    skipped += 1
                    continue
                info = parse_fm(fm)
                title = info["title"] or clean(os.path.splitext(f)[0])
                if not title:
                    skipped += 1
                    continue

                # 相对路径 → 分类（只取第一级，避免过深分组）
                rel = os.path.relpath(root, base)
                parts = [clean(p) for p in rel.split(os.sep) if p and p not in SKIP_DIRS]
                # 第一级作为分类（如 产品手册 / 产品文档 / 产品技术手册）
                category = parts[0] if parts else ""

                # visibility → status（internal/engineering 标记待审，售前默认只显 external）
                status = "verified" if info["visibility"] == "external" else "pending_review"

                # 唯一标题：产品名-文档名
                unique_title = f"{product}-{title}"
                # 去掉图片引用行，保留纯文本
                content = re.sub(r"!\[[^\]]*\]\([^)]*\)", "", body).strip()

                entry = {
                    "type": "product",
                    "title": unique_title,
                    "product": product,
                    "category": category,
                    "tags": info["tags"],
                    "summary": info["summary"],
                    "status": status,
                }
                # 用 yaml 正确序列化（避免 summary/title 里的 * # : 等破坏 YAML）
                fm_lines = []
                if yaml is not None:
                    fm_lines.append(yaml.safe_dump(entry, allow_unicode=True, sort_keys=False).rstrip("\n"))
                else:
                    fm_lines.append(f"type: {entry['type']}")
                    fm_lines.append(f"title: {entry['title']}")
                    fm_lines.append(f"product: {product}")
                    if category:
                        fm_lines.append(f"category: {category}")
                    if entry["tags"]:
                        fm_lines.append("tags: [" + ", ".join(f'"{t}"' for t in entry["tags"]) + "]")
                    if entry["summary"]:
                        fm_lines.append(f"summary: {entry['summary']!r}")
                    fm_lines.append(f"status: {status}")
                lines = ["---", "\n".join(fm_lines), "---", "", content]
                out_path = os.path.join(OUT, sanitize_filename(unique_title) + ".md")
                try:
                    open(out_path, "w", encoding="utf-8").write("\n".join(lines))
                    total += 1
                except Exception as e:
                    skipped += 1
                    print(f"写失败 {out_path}: {e}", file=sys.stderr)

    print(f"导入完成：{total} 篇，跳过 {skipped} 篇")
    print(f"输出目录：{OUT}")


if __name__ == "__main__":
    main()
