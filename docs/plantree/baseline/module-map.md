# 模块地图

> 前后端分离：后端 Go（HTTP API），前端 React + Vite。

## 整体架构

```text
┌──────────────┐    HTTP/JSON     ┌──────────────┐
│   React+Vite  │ ───────────────→ │   Go 后端     │
│   前端        │                  │  (HTTP API)   │
│              │ ←─────────────── │              │
│  分析对话     │                  │  memory/     │
│  记忆管理     │                  │  agent/      │
│  审核队列     │                  │  llm/        │
│              │                  │  handler/    │
└──────────────┘                  └──────┬───────┘
                                         │ 只读
                                         ▼
                                    LLM Wiki
```

## 后端模块

```text
cmd/
└── agent/              # 入口：HTTP 服务启动
    └── main.go

internal/
├── memory/              # 🔴 记忆系统（一等公民）
│   ├── longterm/        #   长期记忆：Wiki 适配器
│   │   ├── wiki.go      #     Wiki 页面加载/解析
│   │   ├── products.go  #     产品知识结构体
│   │   ├── industries.go #    行业记忆结构体
│   │   ├── profiles.go  #     客户/使用者画像
│   │   ├── tools.go     #     Function Calling 工具实现
│   │   └── retriever.go #     关键词检索
│   ├── shortterm/       #   短期记忆：Checkpoint 链
│   │   ├── checkpoint.go #    Checkpoint 结构体
│   │   ├── chain.go      #    Checkpoint 链管理
│   │   ├── notes.go      #    Notes 便签
│   │   └── manager.go    #    上下文管理器
│   └── assembler/       #   拼装层
│       └── assembler.go #     Prompt 拼装器
├── llm/                 # LLM 客户端
│   └── client.go        #   OpenAI 兼容 API 调用
├── model/               # 模型管理系统
│   ├── registry.go      #   模型注册表
│   ├── router.go        #   任务→模型路由
│   ├── fallback.go      #   回退机制
│   └── config.go        #   模型配置结构体
├── agent/               # Agent 核心编排
│   └── agent.go         #   串联记忆→LLM→解析→工具调用
├── review/              # 审核系统（待审核记忆队列）
│   └── review.go        #   pending → approved/rejected
├── history/             # 历史记录（SQLite 持久化）
│   └── store.go         #   session / message / tool_call 轨迹
├── channel/             # 🔌 消息入口抽象（为钉钉留口子）
│   ├── channel.go       #   Channel 接口（Receive/Reply）
│   ├── http/            #   HTTP 实现（一期）
│   │   ├── chat.go      #     /api/analyze, /api/chat
│   │   ├── memory.go    #     /api/memory/* (CRUD)
│   │   └── review.go    #     /api/review/* (审核队列)
│   └── dingtalk/        #   钉钉实现（未来）
│       └── webhook.go   #     钉钉机器人 webhook + 主动推送
└── api/                 # 统一消息类型（channel 与核心的边界）
    └── message.go       #   InboundMessage / OutboundMessage
```

## 前端模块

```text
frontend/
├── src/
│   ├── pages/
│   │   ├── Analyze.tsx        # 分析对话页（粘贴文本 → 看结果）
│   │   ├── Memory.tsx         # 记忆管理页（浏览/编辑/删除）
│   │   ├── Review.tsx         # 审核队列页（pending → 批准/拒绝）
│   │   └── Settings.tsx       # 配置页（LLM 参数等）
│   ├── components/
│   │   ├── ProductCard.tsx    # 产品卡片
│   │   ├── MemoryTable.tsx    # 记忆条目表格
│   │   ├── ReviewQueue.tsx    # 审核队列组件
│   │   └── AnalysisResult.tsx # 分析结果展示
│   └── api/
│       └── client.ts          # 后端 API 封装
├── index.html
├── package.json
└── vite.config.ts
```

## 前后端接口约定

| 前端页面 | 调用的后端 API |
|----------|---------------|
| Analyze | `POST /api/analyze`（分析）、`POST /api/chat`（追问） |
| Memory | `GET /api/memory/search`、`GET /api/memory/list`、`POST /api/memory`、`DELETE /api/memory/{type}/{title}` |
| Review | `GET /api/review/pending`、`POST /api/review/{id}/approve`、`POST /api/review/{id}/reject` |
| Settings | `GET /api/config`、`PUT /api/config` |

## 依赖方向

```text
前端（React）──HTTP──→ channel/http → agent → memory/assembler → memory/longterm
                                        → memory/shortterm
                                        → llm
                                        → model
                                        → review
                                        → history

钉钉（未来）──webhook──→ channel/dingtalk → agent（同一核心，不重复实现）

memory/longterm  ←→  LLM Wiki
history          ←→  SQLite
```

- `channel/*` 依赖 `agent`、`review`、`memory/longterm`、`api`（统一消息类型）
- `agent` 依赖 `memory/*`、`llm`、`model`、`history`、`api`
- `agent` 核心**不依赖** channel，只处理 `api.InboundMessage` / `api.OutboundMessage`
- `review` 依赖 `memory/longterm`（操作待审核队列）
- `memory/longterm` 读写 LLM Wiki
- `history` 读写 SQLite
- 新渠道（钉钉）只新增一个 channel 实现，不动 agent 核心
