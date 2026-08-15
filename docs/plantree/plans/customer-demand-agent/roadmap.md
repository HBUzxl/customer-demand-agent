# 路线图

## Done

### 一期全栈实现（后端 + 前端 + 种子知识库 + 测试）

- ✅ Go 后端全模块实现：domain/config/api/memory(长/短/工具/拼装)/llm/model/agent/history/review/channel
- ✅ 自主 Agent 循环（LLM + function calling，非 workflow）
- ✅ 长期记忆 Wiki 适配器（6 产品 + 6 威胁 + 3 合规 + 3 行业 + 示例画像种子）
- ✅ 短期记忆 Checkpoint 链（initial/followup/reanalysis + notes）
- ✅ 6 个记忆 FC 工具 + 权限矩阵 + 待审核机制
- ✅ SQLite 历史持久化（会话/消息/工具调用轨迹）+ ~~多租户隔离~~（ADR-011 已废止，de-tenancy 落地）
- ✅ Channel 抽象 + HTTP REST API + 钓鱼口子预留
- ✅ React+Vite+TS 前端（6 页面：分析/会话列表/详情/记忆/审核/设置）
- ✅ 24 个单测+集成测试（含 mock LLM 全链路）+ E2E 冒烟脚本（10 项）
- ✅ 后端可托管前端静态文件（SPA fallback）

### 设计阶段（ADR-001 ~ ADR-012）

- ✅ ADR-001 ~ ADR-012 全部落定（见 decisions/README.md）
- ✅ Plan tree 设计文档完整

## In Progress → 已全部完成（2026-08-13 一期 Landed）

> 以下条目在建设中标记进行中，随一期全部落地（见 Done 段与 registry）。

### 记忆系统接口设计 + 项目骨架

- ✅ 记忆系统架构设计完成（Checkpoint 链方案，参考 MiMo Code）
- ✅ ADR-003：短期记忆采用 Checkpoint 链
- ✅ ADR-004：Wiki 三层分类（产品/行业/用户）
- ✅ ADR-005：Wiki 支持 AI 自主写入
- ✅ ADR-006：前后端分离（Go + React/Vite）
- ✅ ADR-007：模型管理默认支持 Function Call
- ✅ ADR-008：自研轻量 Agent 底座
- ✅ ADR-009：顶层是自主 Agent，不是 workflow
- ✅ ADR-010：历史记录持久化到 SQLite
- ✅ ADR-011：~~多租户隔离~~ → **已废止**（2026-08-14 裁决单租户，见 de-tenancy plan）
- ✅ Memory Function Calling 工具设计（6 工具 + 权限矩阵）
- 定义 `memory/longterm` 接口
- 定义 `memory/shortterm` 接口（Checkpoint 链；~~Notes~~ 已删——observe 覆盖职责）
- 定义 `memory/assembler` 接口
- 定义 `memory/tools` 接口（6 个 Function Calling）
- 定义 `llm/client` 接口
- 定义 `agent` 编排接口（自主循环）
- 定义 `review` 审核接口
- 定义 `history` 历史记录接口（SQLite）
- 创建 Go module + 项目目录结构
- 写 `go.mod`、`main.go`、`.env.example`

## Next → 已全部完成或转列 Deferred

> 以下为当时排期项：Wiki 产品页/Checkpoint 链/长期记忆等已落地；
> 多租户已废止；其余延期项见 Deferred 段。

### Wiki 产品知识库

- 在 LLM Wiki 中创建产品页面：
  - 雷池（WAF）
  - 洞鉴（漏洞扫描）
  - 万象（SOC）
  - 谛听（威胁情报）
  - 云图（攻击面管理）
  - 守元（AI 安全）
- 每个页面需包含：能力清单、适用场景、能力边界、竞品参考
- 实现 Wiki 适配器：加载 → 解析 → 索引

### 短期记忆实现（Checkpoint 链）

- Checkpoint 结构体 + 三种类型（initial / followup / reanalysis）
- Checkpoint 链管理器（创建 / 引用 / 遍历）
- ~~Notes 便签机制~~（已删：memory_observe 覆盖该职责，P2 裁决）

### LLM Client

- OpenAI 兼容 API 调用
- 结构化输出（JSON Schema / json_object）
- Function Calling 支持（调用 memory 工具）
- 重试 + 超时

### Prompt 拼装器

- System prompt 模板（含 6 个工具定义）
- 知识注入（检索结果 → prompt section）
- 上下文注入（checkpoint → prompt section）
- 输出格式约束

### Agent 核心（自主循环）

- 自主循环：调 LLM → tool_calls? → 执行工具 → 回填 → 再调，直到最终答案
- 框架不预设业务顺序，LLM 自主决定调用哪些工具
- 结构化输出解析（demand_analysis, products[], feasibility, missing_info）
- 工具调用循环（LLM 返回工具调用 → 执行 → 回填结果）

### 历史记录（SQLite）

- 持久化：消息 + 工具调用 + checkpoint 轨迹
- 会话列表 / 会话详情 / 回放
- 断点续传（从 checkpoint 恢复）

### ~~多租户~~（已废止——ADR-011 废止 2026-08-14，de-tenancy 落地）

- ~~tenant_id 贯穿 session / history / 使用者画像~~（已移除）
- ~~一期只隔离，无 RBAC~~（单租户内部工具；列保留但代码不读写）

### 审核系统（Review）

- 待审核队列（pending → approved / rejected）
- AI 写入行业记忆 → 自动入队
- 审核 API（列表 / 批准 / 拒绝）

### HTTP Handler（后端 API）

- `POST /api/analyze` — 单次分析
- `POST /api/chat` — 追问
- `GET/POST/DELETE /api/memory/*` — 记忆 CRUD
- `GET/POST /api/review/*` — 审核队列
- 会话管理（创建/查询/删除）

### 前端（React + Vite）

- 项目脚手架（Vite + React + TypeScript）
- 分析对话页（粘贴文本 → 看结构化结果）
- 会话列表页 + 会话详情页（回放）
- 记忆管理页（浏览/编辑/删除记忆条目）
- 审核队列页（pending → 批准/拒绝）
- 设置页（LLM 参数等）

## Deferred

- 钉钉机器人对接（Channel 抽象已预留，实现 channel/dingtalk 即可）
- 反馈闭环（售前标注 → 增量学习）
- 方案建议书生成（初版技术方案文档）
- 多轮追问引导
- 竞品话术生成
- 登录认证（账号/SSO/钉钉）
- 权限体系（RBAC）
- 配额（每 tenant 限制）
- Docker 化部署
- CI/CD
- 单元测试 / 集成测试
- RAG（非结构化语义检索）——明确不做，等积累了足够的历史对话和案例数据后再评估
