# 客户需求分析智能体

> 面向长亭科技销售与售前团队的 AI 需求分析助手。粘贴客户沟通文本，Agent **自主**理解
> 安全需求、匹配长亭产品、判断可行性，并给出待追问信息。

自主 Agent（非固定 workflow）：LLM 通过 6 个常驻记忆工具自主决策检索与记录路径。
长期记忆用确定性关键词检索的 Wiki（不做 RAG），短期记忆用 Checkpoint 链。

## 架构

```

数据位置（P9）：三级优先 `CDA_DATA_DIR` 环境变量 > `config.json data_dir` > XDG 默认（`~/.local/share/customer-demand-agent/`）；`wiki_dir`/`history_db` 未显式配置时自动落数据根。项目内旧 `./data` 自动就地兼容。Docker：`ENV CDA_DATA_DIR=/var/lib/cda` + `VOLUME /var/lib/cda`，镜像无状态。
React+Vite 前端 ──HTTP──→ channel/http → agent（自主循环）
                              ↓
                    ┌─────────┼─────────┐
                    │         │         │
              memory(长+短期)  llm+model  history(SQLite)
              tools(6 FC)    review      多租户
```

| 子系统 | 包 | 说明 |
|--------|-----|------|
| Agent 核心 | `internal/agent` | 自主循环：LLM → tool_calls → 执行 → 回填 → 再调（ADR-009）|
| 长期记忆 | `internal/memory/longterm` | Wiki 适配器，确定性关键词检索（ADR-002/004）|
| 短期记忆 | `internal/memory/shortterm` | Checkpoint 链，结构化状态快照（ADR-003）|
| 记忆工具 | `internal/memory/tools` | 6 个 Function Calling + 权限矩阵（ADR-005）|
| 拼装器 | `internal/memory/assembler` | system prompt + 知识 + checkpoint 注入 |
| LLM 客户端 | `internal/llm` | OpenAI 兼容，Chat/ChatWithTools |
| 模型管理 | `internal/model` | 注册/路由/回退/重试（ADR-007）|
| 历史记录 | `internal/history` | SQLite 完整轨迹持久化（ADR-010）|
| 审核系统 | `internal/review` | 待审核记忆队列（ADR-005）|
| Channel | `internal/channel` | HTTP 实现 + 钉钉预留口子（ADR-012）|
| 租户 | — | 单租户（de-tenancy，X-Tenant-ID 忽略，列保留；ADR-011 废止）|

## 快速开始

### 1. 配置（config.json，不用环境变量）

首次运行若 `config.json` 不存在会自动从模板创建；或手动复制：

```bash
cp config.example.json config.json
# 编辑 config.json，填入 models[].api_key / endpoint / model
```

API key、模型列表、路由、回退策略全部集中在 `config.json` 一个文件里。
支持任意 OpenAI 兼容网关（DeepSeek / 通义 / 公司内部网关 / OpenAI）。
运行时也可通过前端「设置」页或 `PUT /api/config` 修改，改动自动回写文件。

### 2. 启动后端

```bash
./scripts/run.sh          # 自动从 config.example.json 创建 config.json，从 wiki/ 播种知识库
# 或：go run ./cmd/agent --config config.json
```

服务监听 `:8080`。Wiki 种子知识库（6 产品 + 6 威胁 + 3 合规 + 3 行业 + 示例画像）
首次启动自动从 `wiki/` 复制到 `data/wiki/`。

### 3. 启动前端（开发模式）

```bash
cd frontend && npm install && npm run dev
# 打开 http://localhost:5173（/api 代理到 :8080）
```

或构建后由后端托管：

```bash
cd frontend && npm run build
FRONTEND_DIST=./frontend/dist ../scripts/run.sh
# 打开 http://localhost:8080
```

### 4. 验证

```bash
./scripts/e2e.sh           # 端到端 API 冒烟测试（10 项，无需 LLM）
```

## 核心能力

**自主分析**——粘贴客户文本，Agent 自主调用记忆工具，输出：
- 需求理解（业务语言 → 安全需求）
- 匹配产品（长亭产品 + 置信度 + 推荐话术）
- 可行性判断（直接覆盖 / 需定制 / 外部整合 / 不建议接）
- 待追问信息

**记忆管理**——6 个常驻工具（`memory_search/ensure/observe/delete/recall/list`），

| 类型 | AI 读 | AI 写 | AI 删 |
|------|:---:|:---:|:---:|
| 产品 | ✅ | ❌（人工）| ❌ |
| 威胁/合规/行业 | ✅ | ✅ 待审核 | ❌ |
| 客户 | ✅ | ✅ | ✅ |
| 使用者 | ✅ | ❌（管理员）| ❌ |

**待审核机制**——AI 写入的威胁/合规/行业记忆标记 `pending_review`，可用于分析（降权）
并出现在审核队列，人工批准转为正式知识、拒绝则删除。

**历史回放**——每条消息、每次工具调用、每个 checkpoint 都持久化到 SQLite，可回放审计。

## API 速览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/analyze` | 分析客户文本（body: `{text, session_id?}`）|
| POST | `/api/chat` | 追问（body: `{session_id, question}`）|
| GET | `/api/memory/search?q=&type=&limit=` | 记忆检索 |
| GET | `/api/memory/list?type=` | 列出某类型 |
| GET/POST/DELETE | `/api/memory/{type}/{title}` | 记忆 CRUD |
| GET | `/api/review/pending` | 待审核队列 |
| POST | `/api/review/{type}/{title}/approve|reject` | 审核 |
| GET/PUT | `/api/config` | 模型与路由配置 |
| GET | `/api/sessions` `GET/DELETE /api/sessions/{id}` | 会话历史 |



## 测试

```bash
go test ./...              # 24 个单测 + 集成测试（含 mock LLM 全链路）
./scripts/e2e.sh           # HTTP API 端到端冒烟
```

集成测试用 httptest 模拟 LLM 网关，验证完整自主循环（工具调用 → 最终 JSON），
**无需真实 API Key**。

## 项目结构

```
cmd/agent/          入口
internal/
  domain/           共享类型（避免循环依赖的叶子包）
  config/  api/     配置 / 渠道消息边界
  memory/{longterm,shortterm,tools,assembler}/
  llm/  model/  agent/  history/  review/
  channel/{http,dingtalk}/
frontend/           React + Vite + TS（5 页面）
wiki/               种子知识库（提交版，首次运行复制到 data/wiki/）
scripts/            run.sh / e2e.sh
docs/plantree/      设计文档与 ADR（ADR-001 ~ ADR-012）
```

## 设计决策

见 `docs/plantree/plans/customer-demand-agent/decisions/README.md`（ADR-001 ~ ADR-012），
核心：Go 自研轻量底座 / 不做 RAG / Checkpoint 链 / Wiki 三层分类 / AI 自主写入 + 待审核 /
前后端分离 / 默认支持 FC / 自主 Agent 非 workflow / SQLite 历史 / 多租户 / 钉钉口子。

## 延期项

钉钉 webhook 对接、登录认证与权限（RBAC）、方案建议书生成、竞品话术、反馈闭环。
