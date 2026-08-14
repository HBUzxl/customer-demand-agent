# Agent 自主化重构（analysis-toolization）

> 把「需求分析工作流」重构为「真正的自主 Agent」：需求分析、可行性判定等业务能力
> 从固定的输出格式/流程，变成 Agent 按需调用的**工具**。这是 ADR-009 的真正落地。

## 为什么重构

ADR-009 说「顶层是自主 Agent，不是 workflow」，但当前实现走样了：

| 走样点 | 现状 | 后果 |
|--------|------|------|
| 强制 JSON 输出 | `runStreaming(..., jsonOutput=true)` + prompt 框死「永远输出分析 JSON」 | 发「你好」也被硬塞成 `demand_analysis`，回复变成说明书 |
| 双入口分裂 | `/api/analyze`（JSON）vs `/api/chat`（自然语言），前端按 `messages.length` 硬分模式 | Agent 没有自主判断输入意图的空间 |
| 解析失败兜底 | `parseAnalysis` 失败 → 整段文本塞进 `demand_analysis` | 寒暄文本被伪装成"分析结果"渲染成卡片 |
| 前端纯文本渲染 | 聊天回复 `<p>{m.text}</p>`，不渲染 markdown | `**加粗**` 显示成字面星号 |

## 目标形态

用户输入（任意文本）→ **单一自主循环**：

- 寒暄/闲聊/简单问题 → Agent 直接自然语言回答（markdown）
- 真实客户需求 → Agent 自主检索记忆（memory_* 工具）→ 调用 `analysis_submit` 工具提交结构化分析 → 前端渲染结果卡片
- 追问 → 基于上下文回答，可再次提交分析（如果判断需求变了）

**决策者是 LLM，不是代码。** 需求分析/可行性判定是 Agent 的工具（能力），不是流程步骤。

## 范围

- **在**：agent 工具化、prompt 重写、API 统一、SSE done 语义、前端统一对话 + markdown 渲染、checkpoint 语义适配、测试更新
- **不在**：记忆系统（Wiki/Checkpoint 内部实现不动）、模型管理、审核流、多租户、钉钉、鉴权

## 文件地图

- [roadmap.md](./roadmap.md) — 四个阶段的状态与验收
- [decisions/README.md](./decisions/README.md) — ADR-013（业务能力工具化）、ADR-014（统一对话入口）
- [open-questions.md](./open-questions.md) — 未决问题
- topics/
  - [tool-contract.md](./topics/tool-contract.md) — analysis_submit 工具契约（核心）
  - [prompt-rewrite.md](./topics/prompt-rewrite.md) — system prompt 自主化重写
  - [api-and-streaming.md](./topics/api-and-streaming.md) — 统一入口 + SSE 协议
  - [frontend-conversation.md](./topics/frontend-conversation.md) — 前端统一对话与渲染
  - [checkpoint-semantics.md](./topics/checkpoint-semantics.md) — checkpoint 在自主模式下的语义

## 阅读路径

1. 本文件（为什么 + 目标形态）
2. decisions/README.md（两个关键决策）
3. topics/tool-contract.md（核心契约）
4. roadmap.md（怎么做、做到哪了）
