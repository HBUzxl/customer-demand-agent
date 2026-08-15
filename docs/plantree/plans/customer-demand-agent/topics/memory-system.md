# 记忆系统设计

> 整个 Agent 的脊梁骨。设计目标：在正确的时刻把正确的知识以正确的密度喂给 LLM。

## 架构概览

```text
┌──────────────────────────────────────────────────┐
│                  Memory System                     │
├──────────────────────────────────────────────────┤
│                                                   │
│  ┌────────────────────┐  ┌──────────────────┐    │
│  │  Long-Term Memory   │  │ Short-Term Memory │    │
│  │  (Wiki 适配器)       │  │ (Checkpoint 链)    │    │
│  └────────┬───────────┘  └────────┬─────────┘    │
│           │                       │               │
│           └──────────┬────────────┘               │
│                      ▼                            │
│           ┌──────────────────┐                    │
│           │ Prompt Assembler │                    │
│           └──────────────────┘                    │
│                                                   │
└──────────────────────────────────────────────────┘
```

## 长期记忆：Wiki 层

> Wiki 层的本质是**符号化知识**——每条知识有明确的标签和路径，检索是**确定性的**（关键词匹配），不依赖向量相似度。
>
> 维护者：**人 + AI 都可以写**。Agent 在分析过程中发现新威胁模式、新行业特征，主动调 `wiki_ensure_page` 写入 Wiki。Wiki 不是静态知识库，而是 AI 自主维护的活知识。

### 三层记忆

```text
Wiki 层
├── 产品记忆（Product Memory）
│   └── 每款产品：能力清单 / 适用场景 / 能力边界 / 竞品对比
│
├── 行业记忆（Industry Memory）
│   ├── 威胁类型：SQL注入 / CC攻击 / 越权 / 信息泄露 / 勒索软件...
│   ├── 合规条款：等保二级/三级 / 数据安全法 / 关基条例...
│   └── 行业场景：电商 / 金融 / 政务 / 医疗 / 制造业...
│
└── 用户记忆（User Memory）
    ├── 客户画像：行业 / 规模 / 技术栈 / 已有安全能力 / 采购偏好
    └── 使用者画像：销售姓名 / 经验等级 / 擅长领域 / 历史准确率
```

### 为什么不用 RAG 做产品匹配

产品知识需要**精确匹配**——"CC 攻击"必须命中的是"雷池的 CC 防护"，不是 cos 相似度最高的某个片段。Wiki 的标签/关键词检索是 100% 确定性的，不会漏也不会错。

### 产品记忆

**定位**：结构化程度最高的知识，确定性最强。"雷池能不能防 CC 攻击"——答案是是或否，不存在模糊空间。

**存储方式**：每个产品一个 Wiki 页面，frontmatter 打标签，Markdown 章节对应结构体字段。

**检索方式**：用户输入 → 分词 → 关键词提取 → 标签/能力名精确匹配 → 返回产品列表（按置信度排序）。

```go
type Product struct {
    Name         string       `json:"name"`         // 雷池
    FullName     string       `json:"full_name"`    // 长亭雷池下一代 Web 应用防火墙
    Aliases      []string     `json:"aliases"`      // ["WAF", "SafeLine"]
    Category     string       `json:"category"`     // 边界安全
    Capabilities []Capability `json:"capabilities"`  // 结构化能力
    Scenarios    []string     `json:"scenarios"`    // ["网站被扫描", "遭受 DDoS"]
    Limitations  []string     `json:"limitations"`  // ["不覆盖主机层威胁"]
    Competitors  []Competitor `json:"competitors"`
    Tags         []string     `json:"tags"`         // ["WAF", "Web安全", "CC防护"]
}

type Capability struct {
    Name        string  `json:"name"`         // "CC 攻击防护"
    Description string  `json:"description"`  // 详细说明
    Confidence  float64 `json:"confidence"`   // 1.0=核心能力, 0.5=边缘能力
    Keywords    []string `json:"keywords"`    // ["CC", "DDoS", "流量清洗"]
}
```

### 行业记忆

**定位**：标准知识——威胁类型的定义、合规条款的要求、行业的典型痛点。这些是"字典"性质的知识，不跟具体客户绑定。

**存储方式**：按分类建 Wiki 页面目录结构：`行业记忆/威胁类型/CC攻击.md`、`行业记忆/合规/等保二级.md`、`行业记忆/行业场景/电商.md`。

**检索方式**：用户输入中识别到行业/威胁/合规关键词 → 精确查找对应页面 → 注入 prompt。

```go
type ThreatType struct {
    Name           string   `json:"name"`            // "CC 攻击"
    Aliases        []string `json:"aliases"`         // ["CC", "HTTP Flood"]
    Description    string   `json:"description"`     // 攻击原理
    TypicalSigns   []string `json:"typical_signs"`   // 典型表现（"网站变慢", "大量异常请求"）
    RelatedProducts []string `json:"related_products"` // ["雷池"]
}

type ComplianceRequirement struct {
    Name           string   `json:"name"`            // "等保二级"
    Requirements   []string `json:"requirements"`    // ["边界防护", "漏洞扫描", "日志审计"]
    RelatedProducts []string `json:"related_products"` // ["雷池", "洞鉴", "万象"]
}

type IndustryScenario struct {
    Name          string   `json:"name"`             // "电商"
    TypicalPains  []string `json:"typical_pains"`    // ["高并发CC攻击", "API滥用", "支付信息泄露"]
    CommonNeeds   []string `json:"common_needs"`     // ["Web防护", "API安全", "数据加密"]
    Keywords      []string `json:"keywords"`         // ["电商", "商城", "在线交易"]
}
```

### 用户记忆

**定位**：与人相关的记忆，分两块——客户画像（被分析的对象）和使用者画像（使用 Agent 的销售）。

**与产品记忆/行业记忆的区别**：产品记忆是"雷池能做什么"，行业记忆是"CC 攻击是什么"——都是通用知识。用户记忆是"这个具体客户是什么情况"、"这个销售擅长什么"——是实例，不是类型。

#### 客户画像

每个客户一个 Wiki 页面，由 Agent 在分析过程中自动创建/更新。

```go
type CustomerProfile struct {
    Name         string   `json:"name"`          // 客户名称（脱敏）
    Industry     string   `json:"industry"`      // 行业
    Scale        string   `json:"scale"`         // 规模（小型/中型/大型）
    TechStack    []string `json:"tech_stack"`    // 技术栈
    ExistingSecurity []string `json:"existing_security"` // 已有安全能力
    PainPoints   []string `json:"pain_points"`   // 已知痛点
    ProcurementPref string `json:"procurement_pref"` // 采购偏好
    UpdatedAt    string   `json:"updated_at"`
}
```

#### 使用者画像

当前销售的基本信息，影响 Agent 的输出风格和深度。

```go
type UserProfile struct {
    Name         string   `json:"name"`           // 销售姓名
    Level        string   `json:"level"`          // 初级/中级/高级
    Expertise    []string `json:"expertise"`      // 擅长领域
    Accuracy     float64  `json:"accuracy"`       // 历史判断准确率
}
```

**分级输出**：

- 初级销售 → 给直接结论（"推荐雷池，因为..."），少用术语
- 高级销售 → 给分析过程 + 竞品话术（"客户可能说华为WAF更便宜，你可以这样回..."）

### 检索流程

```text
用户输入: "我们做跨境电商，最近过等保二级"
    │
    ├── 行业记忆/行业场景 "电商" → 匹配 → 注入电商痛点
    ├── 行业记忆/合规 "等保二级" → 匹配 → 注入合规要求
    ├── 产品记忆 → 电商+等保二关键词 → 雷池 + 洞鉴 + 万象
    ├── 用户记忆/客户画像 → 查已有客户 → 如果是老客户，追加历史上下文
    └── 用户记忆/使用者画像 → 判断销售等级 → 调整输出风格
```

### Wiki 初始化

服务启动时：

```go
func (w *WikiStore) Load(ctx context.Context) error {
    // 1. 扫描 wiki 目录
    pages := w.scanWikiDir()
    // 2. 解析每个页面
    for _, p := range pages {
        switch p.Category {
        case "产品":
            w.products[p.Name] = parseProduct(p)
        case "威胁类型", "合规", "行业场景":
            w.industries[p.Category][p.Name] = parseIndustry(p)
        case "客户":
            w.customers[p.Name] = parseCustomer(p)
        case "使用者":
            w.users[p.Name] = parseUser(p)
        }
    }
    // 3. 构建索引
    w.buildIndex()
    return nil
}
```

## 短期记忆：Checkpoint 链

> 设计灵感来自小米 MiMo Code 的记忆架构。核心原则：**维护结构化状态快照，而非对话文本缓冲**。

### 为什么不用对话缓冲

这个 Agent 不是聊天机器人——它是一次深度分析 + 少量追问。对话缓冲模式有三个问题：

1. **token 浪费**：追问时不需要重放 3000 字原始文档，上一次的分析结果就够
2. **信息稀释**：对话历史里混着原始输入、分析结果、追问、回答，模型提取关键信息越来越难
3. **压缩不可逆**：摘要一旦生成，原文丢了就回不来

### Checkpoint 链模型

参考 MiMo Code：主 Agent 只负责分析，记忆维护通过**独立的结构化快照**完成。不同于 MiMo 有独立 writer subagent，我们同步完成（场景简单，不需要异步）。

```text
Session
├── checkpoints/
│   ├── 001_initial.json        ← 首次分析后的状态
│   │   ├── raw_document         原始客户文本
│   │   ├── demand_summary       需求理解
│   │   ├── matched_products[]   匹配产品 + 理由 + 置信度
│   │   ├── feasibility          可行性判断
│   │   └── missing_info[]       待追问问题
│   │
│   ├── 002_followup.json       ← 追问后的状态
│   │   ├── prev → 001           链式引用
│   │   ├── question             销售问了什么
│   │   └── answer               Agent 答了什么
│   │
│   └── 003_reanalysis.json     ← 新文档重新分析
│       ├── prev → 002
│       ├── new_document         新客户文本
│       └── new_analysis         新分析结果
│
├── notes.md                    ← 临时便签（类似 MiMo notes）
├── current_id                  ← 指向最新 checkpoint
└── session_meta
```

### 三种 Checkpoint 类型

| 类型 | 触发场景 | 内容 |
|------|----------|------|
| `initial` | 销售粘贴新文档 → Agent 分析 | 原始文档 + 完整分析结果 |
| `followup` | 销售追问（"雷池和洞鉴一起推怎么报价？"） | 引用前一个 checkpoint + 追问内容 + 回答 |
| `reanalysis` | 销售换了一个客户重新分析 | 新文档 + 新分析结果，仍链在前一个后面 |

### Checkpoint 链的作用

1. **追问时**：只需要读 `current checkpoint`，不需要重放整个对话
2. **重新分析时**：旧分析保留在链上，可以对比两次分析的变化
3. **Prompt 拼装时**：取 `current checkpoint` + 前 1 个 checkpoint（如果需要上下文），结构体直接序列化注入
4. **审计**：每一步有明确快照，知道 Agent 判断了什么、为什么

### ~~Notes 机制~~（已删——P2 裁决 2026-08-15）

原设计（已废弃）：主 Agent 零散观察 append 到 notes，checkpoint 创建时消费。现由 memory_observe 工具直接写 Wiki 观察，Notes 通道全链删除（Manager 字段/方法/Checkpoint 字段）。

## Prompt 拼装器

### 职责

把长期记忆、短期记忆（checkpoint）、system instruction 拼成最终发给 LLM 的 messages 数组。

### 拼装逻辑

根据当前操作类型，拼装不同的 context：

**首次分析**（无历史 checkpoint）：

```text
[
  { role: "system", content: system_instruction + product_knowledge },
  { role: "user",   content: raw_document }
]
```

**追问**（有 current checkpoint）：

```text
[
  { role: "system", content: system_instruction + product_knowledge },
  { role: "system", content: "上一次分析结果：\n" + checkpoint_json },
  { role: "user",   content: "客户原始描述：\n" + checkpoint.document_summary },
  { role: "user",   content: "追问：" + question }
]
```

**重新分析**（已有 checkpoint，但换文档了）：

```text
[
  { role: "system", content: system_instruction + product_knowledge },
  { role: "system", content: "之前的分析上下文：\n" + prev_checkpoint_json },
  { role: "user",   content: "新的客户描述：\n" + new_document }
]
```

关键原则：**不重放对话原文，只注入结构化状态**。Checkpoint 里已有上一次分析的完整结论，追问时 LLM 不需要重新"看懂"3000 字客户原文。

### System Instruction 模板要点

1. 角色定位：长亭科技售前分析助手
2. 核心任务：需求翻译 → 产品匹配 → 可行性判断
3. 输出格式：JSON Schema
4. 约束：
   - 仅推荐已知产品，禁止编造
   - 不确定时标注"需进一步确认"
   - 主动识别客户未说明的信息

## 接口定义

```go
// ── 长期记忆：Wiki 层 ───────────────────────────

type LongTermMemory interface {
    // 启动时加载所有 Wiki 页面
    Load(ctx context.Context) error

    // ── 产品记忆 ──
    SearchProducts(ctx context.Context, query string) ([]Product, error)
    GetProduct(name string) (*Product, error)
    AllProducts() []Product

    // ── 行业记忆 ──
    SearchThreats(keywords []string) ([]ThreatType, error)
    SearchCompliance(keywords []string) ([]ComplianceRequirement, error)
    SearchIndustryScenario(keywords []string) ([]IndustryScenario, error)

    // ── 用户记忆 ──
    GetCustomerProfile(name string) (*CustomerProfile, error)
    UpsertCustomerProfile(profile *CustomerProfile) error  // AI 可写
    GetUserProfile(name string) (*UserProfile, error)

    // 全量注入（用于 system prompt）
    AllKnowledge() *KnowledgeBundle
}

type KnowledgeBundle struct {
    Products    []Product
    Threats     []ThreatType
    Compliances []ComplianceRequirement
    Industries  []IndustryScenario
}

// ── 短期记忆（Checkpoint 链）─────────────────────

type ShortTermMemory interface {
    // Checkpoint 操作
    CreateInitialCheckpoint(document string, analysis *AnalysisResult) *Checkpoint
    CreateFollowupCheckpoint(question, answer string) *Checkpoint
    CreateReanalysisCheckpoint(document string, analysis *AnalysisResult) *Checkpoint
    CurrentCheckpoint() *Checkpoint
    CheckpointChain() []*Checkpoint

    // 为 Prompt 拼装器提供上下文
    BuildContext(op CheckpointOp) *SessionContext
}

type Checkpoint struct {
    ID        string          `json:"id"`
    Type      CheckpointType  `json:"type"`
    PrevID    string          `json:"prev_id,omitempty"`
    Document  string          `json:"document,omitempty"`   // 原始文档（initial/reanalysis）
    Analysis  *AnalysisResult `json:"analysis,omitempty"`   // 结构化分析
    Question  string          `json:"question,omitempty"`   // 追问内容（followup）
    Answer    string          `json:"answer,omitempty"`     // 回答内容（followup）
    Notes     []string        `json:"notes,omitempty"`
    CreatedAt time.Time       `json:"created_at"`
}

type CheckpointType string
const (
    CheckpointInitial    CheckpointType = "initial"
    CheckpointFollowup   CheckpointType = "followup"
    CheckpointReanalysis CheckpointType = "reanalysis"
)

type CheckpointOp string
const (
    OpInitial    CheckpointOp = "initial"
    OpFollowup   CheckpointOp = "followup"
    OpReanalysis CheckpointOp = "reanalysis"
)

// ── Prompt 拼装器 ───────────────────────────────

type PromptAssembler interface {
    Assemble(op CheckpointOp, knowledge []Product, ctx *SessionContext) []Message
}
```
