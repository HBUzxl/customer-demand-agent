# 终态验证（二期收口）

> goal msu4rm2p-vyd39h 的第 10 项：全套门禁+四脚本+plan-tree 终态的机械可复现验证。
> 状态：**Done**（2026-08-15）。

## 验证入口（单命令）

```bash
./scripts/verify-all.sh --with-smoke   # 全部 13 项（Go 5+前端 4+冒烟 4）；需 :8080 实例
./scripts/verify-all.sh                # 仅构建门禁（无需实例/key）
./scripts/verify-plan-commits.sh       # 「每 plan 独立 commit」机械复核（13 plan ↔ 主 commit + diff 关键词）
```

## 2026-08-15 终态结果

- verify-all --with-smoke：**13 passed / 0 failed**（go build/vet/test/golangci 0/gofmt 0 + 前端 tsc 0/lint 0 errors/format/build + e2e 29/0、f0 11/0、p0 5/0、final 8/0）
- verify-plan-commits：**13 passed / 0 failed**

## 组成说明

goal 的「10 个 plan」= 9 个功能 plan（session-management / memory-management / wiki-hygiene / sidebar-models-ux / memory-search-ia / review-gating / checkpoint-tree / console-config / ui-copy-cleanup）+ 本终态验证 plan。
