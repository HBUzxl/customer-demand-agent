# 路线图

## Done

### Phase 1：后端 Agent 工具化（核心）— 2026-08-14

- ✅ `analysis_submit` 工具（internal/agent/submit.go）：schema 对齐
      domain.AnalysisResult + 可选 is_reanalysis；注册进 runStreaming 的工具表
      （agent 层业务工具，executeTool 拦截挂轮次状态）
- ✅ `runStreaming` 移除 jsonOutput 强制；`parseAnalysis`/stripCodeFence/
      extractJSON 文本兜底全删
- ✅ system prompt 自主化重写（「你是自主的」四类意图指引 + 会话状态只述事实）
- ✅ mock LLM 集成测试三分支全过（寒暄/需求/追问）

### Phase 2：统一入口 + SSE 语义 — 2026-08-14

- ✅ `POST /api/message`（handleMessage/messageCore 分层）
- ✅ `/api/analyze`、`/api/chat` 为 shim（Deprecation 头 + question→text 映射）
- ✅ SSE done：content 恒有、analysis 仅 submit 过才有
- ✅ DetermineOp 收窄为 checkpoint 类型提示
- ✅ e2e 新增 /api/message + analyze shim 用例（18/0 全过）

### Phase 3：前端统一对话 + 渲染 — 2026-08-14

- ✅ `messageStream` 单一入口（analyzeStream/chatStream 删除）
- ✅ content 事件始终累积（分析模式文本流可见）
- ✅ `MarkdownView` 公共组件（sanitize 在 raw 后；Conversation + MemoryDetail 共用，
      ** `**` 渲染修复）
- ✅ ResultCard 在 analysis_submit tool_result 时即时渲染、done 定稿
- ✅ reconstruct 从 tool_calls 还原 analysis（不做老数据兼容——用户裁决）

### Phase 4：全轮次 checkpoint — 2026-08-14

- ✅ 纯聊天轮次建轻量 followup checkpoint（ADR-015/Q1，TestAgentGreetingChat）
- ✅ analysis_submit → initial/reanalysis（is_reanalysis 优先、DetermineOp 兜底，
      TestAgentReanalysisCheckpoint）
- ✅ 断点续传含纯聊天 checkpoint（TestAgentCheckpointRestore 模拟重启 restore）

## In Progress

（无——四 Phase 全部落地，2026-08-14 全套验证通过：go build/vet/test 8 包 +
golangci 0 issue + 前端 tsc/lint/format/build + e2e 18/0 + SSE 手动双场景验证）

## Next

（无）

## Deferred

- 旧端点 `/api/analyze`、`/api/chat` 的删除（等钉钉 Channel 或外部调用方确认后）
- 更多业务工具化候选：方案建议书生成（generate_proposal，ADR-009 已预留此模式）
