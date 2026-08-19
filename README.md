# 客户需求分析智能体

> 面向销售与售前团队的 AI 需求分析助手。粘贴客户沟通文本，Agent **自主**理解
> 安全需求、匹配长亭产品、判断可行性，并给出待追问信息。

自主 Agent（非固定 workflow）：LLM 按当前租户角色获得最小权限记忆工具集，
自主决策检索、精读、分析与追问路径；删除类工具不交给模型。
底层循环不会外发模型原始思维链或未经工具核验的中间草稿；结构化分析通过后
自动进入无工具最终回答，并受整轮超时、工具预算、只读缓存和流式重试保护。
长期记忆用确定性关键词检索的 Wiki（不做 RAG），短期记忆用 Checkpoint 链。

## 架构

```text

数据位置（P9）：三级优先 `CDA_DATA_DIR` 环境变量 > `config.json data_dir` > XDG 默认（`~/.local/share/customer-demand-agent/`）；`wiki_dir`/`history_db` 未显式配置时自动落数据根。项目内旧 `./data` 自动就地兼容。Docker：`ENV CDA_DATA_DIR=/var/lib/cda` + `VOLUME /var/lib/cda`，镜像无状态。
React+Vite 前端 ──HTTP──→ channel/http → agent（自主循环）
                              ↓
                    ┌─────────┼─────────┐
                    │         │         │
              memory(长+短期)  llm+model  history(SQLite)
              tools(6+ FC)   review      后台任务
```

| 子系统     | 包                                       | 说明                                                              |
| ---------- | ---------------------------------------- | ----------------------------------------------------------------- |
| Agent 核心 | `internal/agent`                         | 自主循环：LLM → tool_calls → 执行 → 回填 → 再调（ADR-009）        |
| 长期记忆   | `internal/memory/longterm`               | Wiki 适配器，确定性关键词检索（ADR-002/004）                      |
| 短期记忆   | `internal/memory/shortterm`              | Checkpoint 链，结构化状态快照（ADR-003）                          |
| 记忆工具   | `internal/memory/tools`                  | 7 个 Function Calling + 权限矩阵（ADR-005；含 memory_get 读正文） |
| 拼装器     | `internal/memory/assembler`              | system prompt + 知识 + checkpoint 注入                            |
| LLM 客户端 | `internal/llm`                           | OpenAI 兼容，Chat/ChatWithTools                                   |
| 模型管理   | `internal/model`                         | 注册/路由/回退/重试（ADR-007）                                    |
| 历史记录   | `internal/history`                       | SQLite 完整轨迹持久化（ADR-010）                                  |
| 审核系统   | `internal/review`                        | 待审核记忆队列（ADR-005）                                         |
| Channel    | `internal/channel`                       | HTTP 实现 + 钉钉预留口子（ADR-012）                               |
| 租户       | `internal/{identity,auth,tenancy,audit}` | 多租户 V1：认证控制面 + 租户独立数据面（设计文档 §6）             |

## 快速开始

### 1. 配置（业务配置走 config.json；数据根可用 CDA_DATA_DIR 环境变量指定）

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
首次启动自动从 `wiki/` 播种到数据根（默认 `~/.local/share/customer-demand-agent/wiki`，可用 `CDA_DATA_DIR` 覆盖；老 `./data` 首次启动自动迁入数据根、原位保留可回退）。

**多租户**：所有业务接口需登录（未登录 401）。`/register` 注册工作空间并成为 owner；owner/admin 可在成员页添加已有账号或新建成员，同一账号可加入多个工作空间并从用户菜单切换。platform_admin 可在平台页完成用户、租户和目标租户成员 CRUD，但不会获得目标租户业务数据权限。存量单租户数据首次启动自动迁入 Legacy 租户，可用 `-setup` 交互式创建平台管理员（`go run ./cmd/agent --config config.json -setup`，密码不回显）。

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
bash scripts/e2e-multitenant.sh       # 多租户端到端冒烟（自建 Mock LLM，双租户隔离 + 跨租户 404）
bash scripts/e2e-legacy-migration.sh  # 存量数据自动迁入 Legacy 租户 + -setup 引导冒烟
```

## 核心能力

**自主分析**——粘贴客户文本，Agent 自主调用记忆工具，输出：

- 需求理解（业务语言 → 安全需求）
- 匹配产品（长亭产品 + 置信度 + 推荐话术）
- 可行性判断（直接覆盖 / 需定制 / 外部整合 / 不建议接）
- 待追问信息

**记忆管理**——注册表包含 7 个记忆工具（`memory_search/get/ensure/observe/delete/recall/list`）；
实际注入 Agent 的工具按角色最小化，`memory_delete` 仅保留给显式人工管理接口。

| 类型           | AI 读 |        AI 写        |     AI 删      |
| -------------- | :---: | :-----------------: | :------------: |
| 产品           |  ✅   |     ❌（人工）      |       ❌       |
| 威胁/合规/行业 |  ✅   | owner/admin，待审核 |       ❌       |
| 客户           |  ✅   |     owner/admin     | ❌（人工管理） |
| 使用者         |  ✅   |    ❌（管理员）     |       ❌       |

**待审核机制（写入门禁）**——AI 写入的威胁/合规/行业记忆标记 `pending_review`，**批准前对 Agent 不可见**
（不检索、不注入 prompt），人工在对话内批准后立即生效、拒绝则删除。

记忆库人工权限按登录身份执行：产品为平台只读基线；普通成员只能查看产品、
威胁、合规和行业知识；owner/admin 可维护租户知识覆盖层。每个租户成员会自动
生成 `user_id` 绑定的使用者画像，本人可维护自己的画像，管理员可维护本租户画像；
不再设置全局“当前销售”。

**历史回放**——每条消息、每次工具调用、每个 checkpoint 都持久化到 SQLite，可回放审计。

## API 速览

| 方法            | 路径                                            | 说明                                                            |
| --------------- | ----------------------------------------------- | --------------------------------------------------------------- |
| POST            | `/api/message`                                  | 统一消息入口（body: `{text, session_id?, customer?, branch?}`） |
| POST            | `/api/analyze` `/api/chat`                      | 旧入口兼容 shim，建议新客户端使用 `/api/message`                |
| GET             | `/api/memory/search?q=&type=&limit=`            | 记忆检索                                                        |
| GET             | `/api/memory/list?type=`                        | 列出某类型                                                      |
| GET/POST/DELETE | `/api/memory/**`                                | 记忆查询与管理（写操作受租户 RBAC 约束）                       |
| GET             | `/api/review/pending`                           | 待审核队列                                                      |
| POST            | `/api/review/{type}/{title}/approve|reject`      | 审核/驳回                                                       |
| GET             | `/api/config`                                   | 租户安全配置视图                                                |
| GET             | `/api/sessions` `GET/DELETE /api/sessions/{id}` | 会话历史                                                        |
| GET/PATCH       | `/api/tenant`                                   | 当前租户资料（owner/admin）                                     |
| GET/POST/PATCH/DELETE | `/api/members/**`                         | 当前租户成员管理（owner/admin）                                 |
| POST            | `/api/auth/switch-tenant`                       | 校验 Membership 后切换工作空间                                 |
| CRUD            | `/api/platform/users/**`                        | 平台用户管理（platform_admin）                                  |
| CRUD            | `/api/platform/tenants/**`                      | 平台租户及目标成员管理（platform_admin）                        |

## 测试

```bash
go test ./...                       # 11+ 包测试（含 mock LLM 全链路 + 固化 HTTP 端到端 + 跨租户负向）
bash scripts/e2e-multitenant.sh     # 多租户端到端冒烟
bash scripts/e2e-legacy-migration.sh # 存量数据迁移冒烟
```

集成测试用 httptest 模拟 LLM 网关，验证完整自主循环（工具调用 → 最终 JSON），
**无需真实 API Key**。

## 项目结构

```text
cmd/agent/          入口
internal/
  domain/           共享类型（避免循环依赖的叶子包）
  config/  api/     配置 / 渠道消息边界
  memory/{longterm,shortterm,tools,assembler}/
  llm/  model/  agent/  history/  review/
  channel/{http,dingtalk}/
frontend/           React + Vite + TS（5 页面）
wiki/               种子知识库（提交版，首次运行播种到数据根 wiki）
scripts/            run.sh / e2e-multitenant.sh / e2e-legacy-migration.sh
docs/plantree/      设计文档与 ADR（ADR-001 ~ ADR-012）
```

## 设计决策

当前优化与实施状态统一见 `docs/unified-optimization-and-bugfix-plan.md`；详细多租户设计见
`docs/multi-tenant-design.md`。历史决策见
`docs/plantree/plans/customer-demand-agent/decisions/README.md`（ADR-001 ~ ADR-012），
核心：Go 自研轻量底座 / 不做 RAG / Checkpoint 链 / Wiki 三层分类 / AI 自主写入 + 待审核 /
前后端分离 / 默认支持 FC / 自主 Agent 非 workflow / SQLite 历史 / 服务端 Run 任务模型 / 钉钉口子。

交互与可观测（一期收尾）：ask_user 选项卡 / 内联审核卡（F4，弃独立审核页主路径）/ 编辑重发 / 观测台（/observe：后台任务+日志 tail+LLM 审计）/ 配置中心（设置页只读：prompt 分层/回退链/参数/记忆统计）/ prompt 模板外置（prompts/system.md 编辑重启生效+版本快照）/ 事实时效（覆盖归档留痕）/ Docker 化（VOLUME /var/lib/cda）。

## 延期项

本阶段明确暂缓：文件上传、录音/ASR、CRM 对接和 RAG。后续先完成证据 Harness、
客户事实账本、多租户配额/备份恢复和可选邀请确认，再分别重新评审扩展能力。
