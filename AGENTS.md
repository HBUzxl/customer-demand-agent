# AGENTS.md — 客户需求分析智能体 操作契约

> 面向未来 Agent（和人类）的 operating contract。只记**真规矩与硬约束**，领域细节看
> [README.md](./README.md)、设计决策看 [docs/plantree/](./docs/plantree/)（ADR-001~012）。

## 这是什么

面向长亭科技销售/售前的 AI 需求分析助手。粘贴客户沟通文本 → **自主 Agent**（非固定 workflow）理解安全需求、匹配长亭产品、判断可行性、给待追问信息。长期记忆用**确定性关键词检索的 Wiki**（不做 RAG），短期记忆用 Checkpoint 链。

## 技术栈

| 层 | 栈 | 版本锚点 |
|----|----|---------|
| 后端 | Go（net/http 标准库，无 Web 框架） | `go 1.26`（见 go.mod） |
| 前端 | React + Vite + TypeScript | react 18 / vite 5 / ts 5 |
| 存储 | SQLite（modernc.org/sqlite，纯 Go） | `data/history.db` |
| 知识 | 文件型 Wiki（Markdown/JSON 页面） | `wiki/`（种子）→ `data/wiki/`（运行副本） |
| LLM | OpenAI 兼容网关（deepseek / 通义 / 自建网关） | 配置在 config.json |

**关键：自研轻量 Agent 底座，不引入 Agent 框架**（ADR-008）。核心循环 < 500 行。

## 权威命令（canonical，先跑这些）

```bash
# 后端
go build ./...            # 构建
go vet ./...              # 静态检查（门禁，当前 clean）
go test ./...             # 单测 + 集成测试（mock LLM，无需 API key）
golangci-lint run ./...   # lint（见下方「已知基线」，非阻塞）

# 前端（在 frontend/ 下）
npm install
npm run dev               # 开发服 :5173，/api 代理到 :8080
npm run build             # tsc -b && vite build（门禁）
npx tsc --noEmit          # 类型检查（门禁）
npm run lint              # eslint（已配，见「已知基线」）
npm run format:check      # prettier（已配，见「已知基线」）

# 跑起来
./scripts/run.sh          # 启动后端（自动建 config.json + 播种 wiki）
./scripts/e2e.sh          # HTTP API 端到端冒烟（需后端先起，无需 LLM）
```

## 配置约定（重要）

- **配置走 `config.json`，不用环境变量**。API key / 模型列表 / 路由 / 回退全在这个文件。
- `config.json` 被 `.gitignore` 忽略（含 api_key，**绝不提交**）。首次运行自动从 `config.example.json` 生成。
- 运行时改配置：前端「设置」页或 `PUT /api/config`，改动**回写文件**。
- 没配 api_key 时 `/api/analyze` 返回 502，其余接口正常。

## 架构约束（改动前必读）

模块地图权威在 [docs/plantree/baseline/module-map.md](./docs/plantree/baseline/module-map.md)。硬约束：

- **依赖方向**：`channel/*` → `agent` → `{memory/*, llm, model, history}`；`agent` 核心**不依赖 channel**，只处理 `api.InboundMessage`/`OutboundMessage`（ADR-012，为钉钉留口子）。
- **`internal/domain` 是叶子包**——共享类型放这里，禁止反向依赖业务包（防循环依赖）。
- **新渠道 = 新增一个 channel adapter，不动 agent 核心**。
- **深模块纪律**：别把单文件扩成大杂烩；按 memory/agent/channel/llm/model/history/review 的领域边界组织。
- Go 1.22+ ServeMux 方法路由（`mux.HandleFunc("POST /api/analyze", ...)`），不用第三方路由库。

## Agent 自主化（ADR-013/014/015，2026-08-14 落地）

- **业务能力工具化**：需求分析不是固定输出格式，是 `analysis_submit` 工具（agent 层业务工具，`internal/agent/submit.go`，schema=AnalysisResult+is_reanalysis）。done 事件 content 恒有（自然语言），analysis 仅 submit 过才有。
- **统一对话入口**：`POST /api/message`（{text, session_id?}）走 `agent.Message()` 单一自主循环；`/api/analyze`、`/api/chat` 是 deprecated shim。
- **前后台域划分**：前台（与 Agent 的对话）= 记忆系统成套且**全程在线**——聊天轮次涉及事实也要 memory_search 查证、聊出线索要 memory_observe 记录（不是裸 chat）；后台（TaskBackground，一期未实现）= 无记忆依赖的一次性调用。
- **checkpoint 全轮次**：提交分析→initial/reanalysis（is_reanalysis 优先）；纯聊天/追问→轻量 followup（Question/Answer 照记）。
- **历史数据契约**：assistant content 存自然语言；结构化分析在 tool_calls 的 analysis_submit params 里（前端回放从 tool_calls 还原）。不做老数据兼容（用户裁决：老数据适配新系统）。

## 错误处理（已落地，照此延续）

LLM 调用层有显式错误分类（`internal/llm/client.go` + `internal/model/manager.go`），新代码沿用：

- **`TransportError`**（网络层，可重试）vs **`APIError{StatusCode, Body, Retryable}`**（API 非 2xx）。
- **`IsRetryable(err)`** 谓词 + `isRetryableStatus`（429 或 ≥500 才重试）。
- **`%w` 包裹**保留 cause chain（`fmt.Errorf("构造请求: %w", err)`）。
- **重试只在「可重试 + 幂等」上做**，指数退避 + **全抖动**（`base * 2^(n-1)`，封顶 30s，`[0,d)` 抖动），`MaxRetries` 默认 2，`BackoffMs` 默认 200。
- **流式调用只在「尚未推送首字节」时才回退**（推过就撤不回了）。
- **HTTP 边界统一** `writeJSON` / `writeError(w, status, ...)` → `{"error":"..."}`；不往外抛栈。
- 不吞错；验证失败用早返回 + 明确中文消息。

## 日志

- stdlib `log.Printf`。请求日志：`%s %s %s %v`（method, path, tenant, duration），在 `logging` 中间件。
- 启动日志用 `[ok]` / `[warn]` 前缀。**不记 api_key / 敏感配置**。

## 多租户

请求头 `X-Tenant-ID`（缺省用 `config.default_tenant`）。tenant_id 贯穿 session / history / 使用者画像；产品/行业/客户记忆是组织级共享（ADR-011）。**一期无鉴权**。

## 前后端连接

开发：Vite dev server `:5173`，`/api` 代理到后端 `:8080`（见 `frontend/vite.config.ts`）。
生产：后端可选托管前端 `dist/`（`config.server.frontend_dist`），SPA fallback 到 `index.html`，`/api` 与 `/assets/` 走专属路径。

## 测试

- **`go test ./...`**：当前 agent / channel/http / memory×4 共 7 个包有测试，**含 mock LLM 全链路集成测试**（无需真实 key）。
- **无测试的包**：config / domain / history / llm / model / review / api —— 新行为补测试时优先这些。
- **`./scripts/e2e.sh`**：真实 HTTP 端到端冒烟（健康/配置/记忆 CRUD/审核流/会话），需后端先起。
- 新逻辑**必须带验证行为的测试**（不是空断言）。

## 初始化状态（本次 /init，2026-08-13）

已落地（additive，未动现有源码）：
- ✅ Baseline 验证：`go build` / `go vet` / `go test ./...` 全过；前端 `tsc --noEmit` / `npm run build` 过。
- ✅ CI：`.github/workflows/ci.yml`（后端 build/vet/test + golangci-lint + govulncheck 全门禁；前端 tsc/lint/format:check/build 全门禁；npm audit informational）。
- ✅ Go lint：`.golangci.yml`（v2，gofmt 走 formatters）+ golangci-lint 已装，**0 issue**。
- ✅ 前端 lint/format：eslint flat config + prettier，`npm run lint` / `format` / `format:check` 可用，**lint 0 errors**。
- ✅ 依赖审计基线已测（见下）。
- ✅ 本文件（AGENTS.md）。

### 已知基线（2026-08-13 已清一轮）

| 项 | 基线 | 处理 |
|----|------|------|
| golangci-lint | **0 issue**（原 24：errcheck/staticcheck/unused 全清，4 个死函数 joinID/toOutbound/jsonEncode/newID 已删） | CI 阻塞门禁 |
| govulncheck | **0 漏洞** | CI 门禁（阻塞） |
| npm audit | 4 漏洞（1 high：vite→esbuild；3 moderate：react-router） | react-router 可 `npm audit fix`；vite 升 8 是 breaking，**需用户决策**。CI 暂 informational |
| eslint | **0 errors**（原 42→0：`no-explicit-any` 全替类型化、Math.random→useId、img node 不透传；剩 ~13 warnings：react-refresh 路由/工具文件结构性 + exhaustive-deps 标准 load 模式） | CI 门禁（lint）；warnings 不阻塞 |
| prettier | **全部合规**（`npm run format` 已采纳） | CI 门禁（format:check） |
| 测试覆盖 | history/llm/http 已有测试；config/domain/model/review 仍无 | 按需补 |

## 工作约束（给 Agent）

- **先读后改**。改架构前读 module-map + 相关 ADR。
- **不引入新依赖**除非用户要（ADR-008：底座保持轻量）。
- **不改产品记忆的 AI 写权限**（ADR-005：产品记忆只人工维护）。
- **不做 RAG**（ADR-002，明确 deferred）。
- **config.json / api_key 绝不提交、绝不打进日志、绝不回显给客户端**。
- commit 前跑 canonical 命令；改了 Go 跑 `go build && go vet && go test ./...`，改了前端跑 `npm run build`。

## 延期项（Deferred）

钉钉 webhook 对接、登录鉴权/RBAC、方案建议书生成、竞品话术、反馈闭环（见 README「延期项」与 ADR）。
