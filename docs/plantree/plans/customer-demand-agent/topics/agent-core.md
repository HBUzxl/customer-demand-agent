# Agent 核心编排

## 职责

`internal/agent/agent.go` 是整个系统的编排层，不包含业务逻辑，只做串联。

## 接口

```go
type Agent interface {
    // Analyze 分析客户需求，返回结构化结果
    Analyze(ctx context.Context, sessionID, input string) (*AnalysisResult, error)
}

type AnalysisResult struct {
    DemandAnalysis  string            `json:"demand_analysis"`  // 需求理解
    MatchedProducts []MatchedProduct  `json:"matched_products"` // 匹配产品
    Feasibility     Feasibility       `json:"feasibility"`      // 可行性判断
    MissingInfo     []string          `json:"missing_info"`     // 待追问信息
}

type MatchedProduct struct {
    Name       string  `json:"name"`        // 产品名
    Confidence float64 `json:"confidence"`  // 置信度 0-1
    Reason     string  `json:"reason"`      // 匹配理由
    Suggestion string  `json:"suggestion"`  // 推荐话术
}

type Feasibility string

const (
    FeasibilityDirect  Feasibility = "direct"  // 现有产品直接覆盖
    FeasibilityCustom  Feasibility = "custom"  // 基础上定制
    FeasibilityPartner Feasibility = "partner" // 需整合外部资源
    FeasibilityReject  Feasibility = "reject"  // 做不了
)
```

## 编排流程（自主循环，非固定 workflow）

> 见 [[agent-autonomy.md]]。框架不预设"先查产品再匹配"的顺序，LLM 自主决定调用哪些工具。

```go
func (a *agent) Analyze(ctx context.Context, sessionID, input string) (*AnalysisResult, error) {
    // 1. 用户输入 → 加入 messages
    a.messages = append(a.messages, Message{Role: "user", Content: input})

    // 2. 自主循环：调 LLM → 工具调用? → 执行 → 回填 → 再调
    for {
        resp, err := a.llm.Chat(ctx, a.messages, a.tools)
        if err != nil {
            return nil, fmt.Errorf("llm call: %w", err)
        }

        a.messages = append(a.messages, resp.Message)

        // 没有工具调用 → LLM 给出了最终答案
        if len(resp.ToolCalls) == 0 {
            return a.parseResult(resp.Message.Content)
        }

        // 执行工具调用
        for _, tc := range resp.ToolCalls {
            result := a.executeTool(ctx, tc)
            a.messages = append(a.messages, Message{
                Role:       "tool",
                ToolCallID: tc.ID,
                Content:    result,
            })
        }
        // 回到循环，让 LLM 看工具结果继续推理
    }
}
```

**关键点**：框架只提供循环和工具执行，不决定"先做什么后做什么"。LLM 看到客户输入后，自主决定：查产品？查客户？直接给结论？记录画像？

## 输出解析

LLM 返回 JSON：

```json
{
  "demand_analysis": "客户描述的是 Web 应用层面的安全威胁...",
  "matched_products": [
    {
      "name": "雷池",
      "confidence": 0.95,
      "reason": "客户描述的扫描行为是典型的 Web 攻击前兆",
      "suggestion": "推荐雷池 WAF，部署在业务前端进行流量清洗"
    }
  ],
  "feasibility": "direct",
  "feasibility_detail": "雷池 WAF 可直接满足客户需求，7 层防护全覆盖",
  "missing_info": ["攻击频次和规模", "是否有合规要求", "业务系统技术栈"]
}
```

解析时先尝试 JSON unmarshal，失败则 retry（最多 2 次），仍失败则返回原始文本作为 demand_analysis。

## 注意：分析结果 vs 自主输出

`AnalysisResult` 结构体是**期望的最终输出**（写在 system prompt 里让 LLM 产出），不是 workflow 步骤。LLM 自主推理后，给出符合这个结构的答案。中间过程（查了什么、记了什么）由 LLM 自由决定，框架不管。
