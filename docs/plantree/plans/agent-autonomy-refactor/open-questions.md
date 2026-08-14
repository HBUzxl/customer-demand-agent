# 未决问题

> 2026-08-14：用户已裁决 4 项（Q1/Q3/Q4 + 新增前后台域划分，见 decisions/ADR-015）。
> 本文件不再有未决项；新问题出现时再加。

## 已裁决（存档）

- **Q1 纯聊天建 checkpoint？** ✅ 建。短期记忆系统覆盖整个项目，纯聊天轮次
  也建轻量 checkpoint（followup），保证对话链完整（断点续传/上下文不断裂）。
- **Q3 ResultCard 时机？** ✅ 交给实现者：采用 tool_result 到达即渲染、done 时
  若再次提交则覆盖（过程透明优先）。
- **Q4 老数据兼容？** ✅ 代码**不做**老数据兼容。方向反过来：老数据会被处理
  去适配新系统（用户侧负责）。reconstruct 不需要 content-JSON fallback 分支。
- **DetermineOp 去留？** ✅ 交给实现者：保留为 checkpoint 类型提示，不影响
  Agent 自主行为。
