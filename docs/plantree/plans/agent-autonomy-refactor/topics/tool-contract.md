# analysis_submit 工具契约

> 核心设计：结构化分析从「输出格式」变成「工具调用」。

## 工具定义

```json
{
  "name": "analysis_submit",
  "description": "提交一份完整的客户需求分析结果。当且仅当你判断用户输入包含真实的客户安全需求（而非寒暄、闲聊或简单提问），且你已完成必要的记忆检索与思考时调用。同一轮对话若判断深化，可再次调用覆盖之前的结果。",
  "parameters": {
    "type": "object",
    "properties": {
      "demand_analysis": {"type": "string", "description": "需求理解（中文，把业务语言翻译成安全需求）"},
      "matched_products": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name":       {"type": "string"},
            "confidence": {"type": "number", "description": "0-1"},
            "reason":     {"type": "string"},
            "suggestion": {"type": "string", "description": "销售可直接使用的推荐话术"}
          },
          "required": ["name", "confidence", "reason"]
        }
      },
      "feasibility":        {"type": "string", "enum": ["direct", "custom", "partner", "reject"]},
      "feasibility_detail": {"type": "string"},
      "missing_info":       {"type": "array", "items": {"type": "string"}}
    },
    "required": ["demand_analysis", "feasibility"]
  }
}
```

schema 与 `domain.AnalysisResult` 对齐——工具参数直接 unmarshal 成结构体，零转换。

## 工具归属层

- **memory 工具**（memory_search/ensure/...）：读写记忆，属于记忆域，权限矩阵管
- **analysis_submit**：**agent 层业务工具**——它不"检索"什么，而是提交结论。
  注册在 tools registry 但标记为业务工具（`Business: true`），执行即返回
  `{"received": true}`，副作用是把结果挂到本轮 Trace/Event 流

实现选择：复用 `tools.Registry`（加一个 flag 区分），不另起注册表——避免两套
工具分发机制。

## 事件流语义

```text
content 流        → Agent 的自然语言说明/总结（用户先看到）
tool_call         → {"tool":"analysis_submit","params":{...结构化结果...}}
tool_result       → {"received": true}
done              → {content: "最终自然语言答案", analysis: {...} | 无}
```

- `analysis` 来自本轮最后一次 analysis_submit 的 params
- 未调用 → done 无 analysis，前端纯文本气泡
- 调用过 → 前端文本气泡 + ResultCard

## 为什么不用"最终答案里嵌 JSON"

- 输出格式歧义：解析器猜"是聊天还是分析"，猜错整个渲染分支崩
- 流式体验差：用户要等整个 JSON 拼完才能看到内容
- 工具调用是结构化、无歧义的信号，schema 即契约（OpenAI FC 原生校验）

## 历史持久化

analysis_submit 的 params 就是结构化结果，随现有 tool_calls 表自然持久化
（params_json 列），回放时从 tool_call 还原 analysis——不需要新表。
