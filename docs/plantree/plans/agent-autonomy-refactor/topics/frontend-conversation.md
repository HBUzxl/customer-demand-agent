# 前端统一对话与渲染

## 现状问题（Conversation.tsx）

- `send(text, kind)` 按 `messages.length` 硬分 analyze/chat——workflow 思维
- assistant 文本 `<p>{m.text}</p>` 纯文本，`**加粗**`、列表、链接全是字面符
- handleEvent 里 content 只在 `kind==="chat"` 时累积——分析模式的 content 流
  被丢弃（用户看不到 Agent 的自然语言说明！）

## 改动

### 1. 统一消息流

- client.ts：`analyzeStream`/`chatStream` → `messageStream(text, sessionId, onEvent)`
- `send()` 单一入口；ChatMsg.kind 字段删除
- handleEvent：content **始终累积**（这是"过程透明"的关键修复）

### 2. Markdown 渲染

抽公共组件 `MarkdownView`（从 MemoryDetail 提取）：
`react-markdown + remark-gfm + rehype-raw + rehype-sanitize + mdComponents`
（mermaid 代码块、img 断链隐藏）。assistant 文本气泡用它渲染。
**必须保持 rehype-sanitize 在 rehype-raw 之后**（安全基线，勿回退）。

### 3. 结果卡片条件渲染

```text
<AssistantMsg>
  {m.analysis && <ResultCard r={m.analysis} />}   // 上方：结构化卡片
  <MarkdownView>{m.text}</MarkdownView>            // 下方：自然语言总结
</AssistantMsg>
```

- analysis 在 analysis_submit 的 **tool_result 到达时即渲染**（用户裁决交实现者
  定，采用过程透明优先）；done 时若再次提交则覆盖
- 纯聊天轮次：只有 MarkdownView，无卡片

### 4. 回放（reconstruct，不做老数据兼容）

**用户裁决**：代码不兼容老数据；老数据会被处理适配新系统。

- assistant content 一律视为自然语言
- analysis 从该会话 tool_calls 里的 `analysis_submit` params_json 还原
- 不需要 content-JSON 探测 fallback

## 验收场景（人工）

1. 发"你好" → markdown 聊天回复，无卡片，`**` 正常加粗
2. 发客户需求原文 → 思考流（自动展开）→ 工具调用 → 结果卡片 + 自然语言总结
3. 追问 → 基于上下文回答
4. 新系统产出的会话回放正常
