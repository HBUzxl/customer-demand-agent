# 历史记录与多租户（租户部分已废止）

> 会话级基础设施：完整轨迹的持久化 + 使用者隔离。

## 一、历史记录

### 与短期记忆的区分

| | 短期记忆（Checkpoint 链） | 历史记录（History） |
|---|---|---|
| 本质 | 工作记忆，给 LLM 用 | 持久记录，给人看 + 审计 |
| 存储 | 进程内存 | SQLite |
| 生命周期 | 会话期间 | 会话结束后仍可查 |
| 内容 | 结构化分析状态 | 完整轨迹（每条消息+工具调用） |
| 参考 | MiMo Session memory | MiMo History 层 |

### 记录什么

```text
Session (session_id, created_at)  # tenant_id 列保留但代码不读写（de-tenancy）
├── Message       每条消息：role / content / timestamp
├── ToolCall      每次工具调用：tool_name / params / result / timestamp
├── Checkpoint    每个 checkpoint 快照（同步落一份到磁盘）
└── SessionMeta   会话元数据：标题、客户名、创建时间
```

### 用途

- **会话列表页**：看历史会话，按时间/客户检索（tenant 维度已废止）
- **会话详情页**：回放对话 + 工具调用轨迹 + checkpoint 快照
- **审计**：Agent 为什么这么判断？查它的工具调用记录
- **断点续传**：会话中断后从 checkpoint 恢复

### 存储方案（de-tenancy 后已不再按 tenant 隔离；现状 schema 见 `internal/history/store.go` 与 `data/wiki/`）

<details><summary>废止前的设计存档（点击展开）</summary>

SQLite（`modernc.org/sqlite`，纯 Go 无 cgo）：

```sql
CREATE TABLE sessions (
    session_id  TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    title       TEXT,
    customer    TEXT,
    created_at  TIMESTAMP
);

CREATE TABLE messages (
    id          INTEGER PRIMARY KEY,
    session_id  TEXT,
    role        TEXT,       -- user / assistant / tool
    content     TEXT,
    tool_call_id TEXT,      -- 工具调用关联
    created_at  TIMESTAMP
);

CREATE TABLE tool_calls (
    id          INTEGER PRIMARY KEY,
    session_id  TEXT,
    tool_name   TEXT,
    params_json TEXT,
    result_json TEXT,
    created_at  TIMESTAMP
);
```

</details>

## 二、多租户（已废止——ADR-011 废止 2026-08-14，de-tenancy 已落地）

> 以下租户设计整体作废：单租户内部工具；tenant_id 列保留但代码不读写；
> X-Tenant-ID 忽略。仅存档设计过程，勿按此理解现状。

<details><summary>废止前的设计存档（点击展开）</summary>

### 隔离范围

```text
隔离（每个 tenant 独立）           共享（组织级知识）
─────────────────────            ─────────────────
├── 会话 session                   ├── 产品记忆
├── 历史记录 history               ├── 行业记忆
├── 使用者画像 user profile        └── 客户画像
└── （后续：权限、配额）
```

**逻辑**：产品和行业知识是全公司共用的，不因使用者不同而变。客户的画像也应该是组织级资产（多个销售可能服务同一客户）。只有"谁在用什么、说过什么"是个人级的。

### 一期实现（租户部分已废止——见顶部说明）

- `tenant_id` 作为请求上下文，贯穿所有 handler
- 来源：登录态 / HTTP header / 前端选择，具体方式后续定
- 无 RBAC、无权限校验——只做数据隔离
- 每个使用者一个 tenant

### 后续（Deferred）

- 登录认证（账号密码/SSO/钉钉）
- 权限体系（RBAC：谁能看谁的会话、谁能审谁的内容）
- 配额（每 tenant 的 token/调用限制）

</details>

## 三、页面功能（先不做业务，只看页面）

| 页面 | 功能 | 数据 |
|------|------|------|
| 分析对话 | 跟 Agent 对话 | 当前 session（内存） |
| 会话列表 | 看历史会话，~~按 tenant 过滤~~（已废止） | SQLite sessions |
| 会话详情 | 回放对话 + 工具调用轨迹 | SQLite messages + tool_calls |
| 记忆管理 | 浏览/编辑/删除记忆 | Wiki（共享） |
| 审核队列 | 待审核记忆审批 | 待审核队列 |
| 设置 | LLM 参数、模型、fallback | 配置 |

~~多租户体现在：会话列表/详情只显示当前 tenant 自己的数据。~~（已废止——单租户，全部会话可见）
