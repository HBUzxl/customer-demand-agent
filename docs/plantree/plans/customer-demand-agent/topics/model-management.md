# 模型管理系统

> 运行时模型调度：注册、路由、回退、调用封装。一期所有模型默认支持 function call，不做 FC 能力标签。

## 职责

```text
模型管理系统
├── 模型注册     有哪些模型？endpoint / key / 参数
├── 模型路由     什么任务用哪个模型
├── Fallback     模型挂了降级到备选
└── 调用封装     统一 LLM 客户端（OpenAI 兼容 + 重试 + 超时）
```

## 一、模型注册

前端页面配置，存配置文件/数据库。后端启动时加载。

```go
type ModelConfig struct {
    Name        string        `json:"name"`         // 标识，如 "deepseek-v4-pro"
    Endpoint    string        `json:"endpoint"`     // API 网关地址
    APIKey      string        `json:"api_key"`      // API key（不展示，前端只存不读回）
    Model       string        `json:"model"`        // 实际模型名（网关用）
    Temperature float64       `json:"temperature"`  // 默认 0.3
    MaxTokens   int           `json:"max_tokens"`   // 最大输出 token
    Timeout     time.Duration `json:"timeout"`      // 默认 60s
    // 注意：没有 SupportsFunctionCall 字段——默认都支持
}
```

**要点**：

- 一个 API 网关可以注册多个模型名（同一 endpoint 不同 model）
- 也可以只注册一个模型，所有任务都走它
- 模型数量无所谓，前端想加几个加几个

## 二、模型路由

不同任务类型可以指定不同模型。一期最简单的形态：**全部任务走默认模型**。

```go
type TaskType string

const (
    TaskAnalysis   TaskType = "analysis"    // 主分析（需求理解/产品匹配/可行性）
    TaskToolCall   TaskType = "tool_call"   // 工具调用（记忆读写）
    TaskBackground TaskType = "background"  // 后台任务（后续可能需要）
)

type Router struct {
    // 任务类型 → 模型名。未配置的走 Default
    routes map[TaskType]string
    Default string
}
```

**为什么路由可以很简单**：用户可以把不同能力分到一个模型，也可以分到多个。一期先"一个模型打天下"，路由表留空全走 default，等真有需要再细分。

## 三、Fallback 机制

```go
type FallbackPolicy struct {
    MaxRetries  int           `json:"max_retries"`  // 每个模型重试次数
    BackoffBase time.Duration `json:"backoff_base"` // 退避基数，默认 200ms
    Chain       []string      `json:"chain"`        // 回退链：["primary", "backup"]
}
```

**流程**：

```text
调用 primary 模型
    │
    ├── 成功 → 返回
    │
    └── 失败（超时/5xx/限流）
         │
         ├── 重试 primary（指数退避，最多 N 次）
         │
         └── 仍失败 → 切到 backup 模型
              │
              ├── 成功 → 返回（标记降级）
              └── 失败 → 返回错误
```

**重试判定**（不是所有错误都重试）：

- 重试：网络超时、5xx、429 限流
- 不重试：4xx 参数错误、鉴权失败、内容违规

## 四、调用封装

统一的 LLM 客户端，所有模型调用走这里：

```go
type LLMClient interface {
    // Chat 普通对话，返回文本
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    // ChatWithTools 带 function calling 的对话
    ChatWithTools(ctx context.Context, req *ChatRequest, tools []Tool) (*ChatResponse, error)
}

type ChatRequest struct {
    TaskType TaskType  // 决定路由到哪个模型
    Messages []Message
    Tools    []Tool    // function calling 工具定义（可选）
    Temperature *float64 // 覆盖默认温度
}
```

**结构化输出**：主分析要求 JSON 输出。两种方式：

1. `response_format: json_object`（OpenAI 兼容）
2. 带 JSON Schema 的 function calling（更严格）

一期先用 `json_object` + prompt 里写清楚格式，跑通了再考虑 JSON Schema。

## 五、前端配置页

| 配置项 | 说明 |
|--------|------|
| API 网关地址 | endpoint |
| API Key | 密钥（密文存储，前端不读回） |
| 模型列表 | 名称 + 参数（temperature/max_tokens/timeout） |
| 路由配置 | 任务类型 → 模型（可选，默认走 default） |
| Fallback 链 | 回退顺序 + 重试次数 |

**后台功能支持**：Agent 主流程之外的、需要调 API 的后台功能（如果以后有），也在这个配置页配置，不单独搞一套。

## 与记忆系统的关系

模型管理系统是**被调用的基础设施**，记忆系统是**被管理的数据**。

```text
Agent 编排
    │
    ├── 记忆系统（数据层）——存什么、查什么
    └── 模型管理系统（执行层）——用什么模型跑
```

记忆系统的 `memory_recall` 语义搜索（如果将来做）可能需要 embedding 模型——这个也归模型管理系统管，配置在模型列表里。
