# 统一入口 + SSE 语义

## 新端点

`POST /api/message`，body：`{"text": string, "session_id": string?}`。
响应：SSE 流（与现在完全一致的事件类型）。

handler 逻辑 = 现 handleAnalyze 去掉"分析"假设：
tenant 解析 → sessionID → EnsureSession（跨租户校验）→ AppendMessage(user) →
SSE 头 → agent.Message(...) → 持久化 assistant 输出 + trace。

## 旧端点 shim

- `/api/analyze`、`/api/chat` → 内部直接调 handleMessage 同一函数（chat 端点
  body 的 question 映射为 text）。响应加 `Deprecation` 头。
- 删除时机：Deferred（见 roadmap）。

## SSE 事件（协议不变，语义增强）

| 事件 | 变化 |
|------|------|
| session / round / reasoning / content / tool_call / tool_result | 不变 |
| done | `content` 恒有（自然语言答案）；`analysis` **可选**（本轮调用过 analysis_submit 才有） |

前端不再需要根据 `messages.length` 或 kind 决定行为——统一消费。

## agent 侧接口变化

`AnalyzeStream` / `ChatStream` 合并为 `Message(ctx, tenantID, sessionID, text, emit)`：

- 内部仍是同一个自主循环（runStreaming）
- DetermineOp 保留，但输出只用于 checkpoint 类型选择
- Event 结构不动（新增 analysis 已在 done 里可选携带）

## 测试影响

- integration_test：mock LLM 脚本改为"寒暄→纯文本 done"、"需求→tool_call
  analysis_submit→done 带 analysis"两条路径
- e2e.sh：/api/message 冒烟（无 LLM 路径：session 建立 + 403 跨租户已在覆盖）；
  analyze/chat shim 回归
