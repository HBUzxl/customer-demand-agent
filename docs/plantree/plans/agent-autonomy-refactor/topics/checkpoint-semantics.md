# Checkpoint 在自主模式下的语义

## 现状

Checkpoint 类型由 DetermineOp（文本启发式）决定：OpInitial / OpFollowup /
OpReanalysis，同时决定 prompt 分支与输出格式。AnalyzeStream 结束时无条件建
checkpoint（initial/reanalysis），ChatStream 建 followup。

## 自主模式下的新语义（2026-08-14 按用户裁决更新）

DetermineOp **降级为 checkpoint 类型提示**（不再影响 prompt/输出）。

**关键裁决（ADR-015 + Q1）**：记忆系统覆盖**所有前台轮次**——纯聊天也建
checkpoint（轻量 followup：Question/Answer 照记，Analysis 空），保证对话链
完整、断点续传不断裂。

| 轮次结局 | checkpoint 类型 | 说明 |
|----------|-----------------|------|
| 调用了 analysis_submit（无前次分析 / Agent 判定新文档） | initial | Document + Analysis 都来自工具结果 |
| 调用了 analysis_submit（Agent 判定需求实质变化） | reanalysis | 同上 |
| 未调用（纯聊天/追问回答） | followup（轻量） | Question/Answer 记对话，Analysis 空 |

判定"首次 vs 实质变化"：信任 Agent 在 analysis_submit 参数里带可选
`is_reanalysis` 标记（决策还给 LLM），DetermineOp 启发式做兜底。

## 已定项

- Q1：纯聊天建 checkpoint ✅（见上表）
- DetermineOp：保留为 checkpoint 类型提示，不影响 Agent 行为 ✅

## 不变的部分

- Checkpoint 链结构、Restore、断点续传、SQLite 持久化——全不动
- 多租户作用域（本轮已加固）——不动
