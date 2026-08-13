# 运行时流程

## 核心链路：客户文本 → 分析结果

```
POST /api/analyze  { text: "客户原话..." }
         │
         ▼
   ┌─────────────┐
   │  Handler     │  解析请求，创建/获取会话
   └──────┬──────┘
          │
          ▼
   ┌─────────────┐
   │  Agent       │  编排核心逻辑
   └──────┬──────┘
          │
          ├──→ memory/shortterm  记录用户输入到对话缓冲
          │
          ├──→ memory/longterm   检索相关产品知识
          │        │
          │        └──→ Wiki 适配器读页面 → 结构化解析 → 关键词匹配
          │
          ├──→ memory/assembler  拼装最终 prompt
          │        │
          │        ├── System instruction（角色+输出格式）
          │        ├── Long-term context（检索到的产品知识）
          │        ├── Short-term context（对话历史+摘要）
          │        └── User input（客户原文）
          │
          ├──→ llm/client        调用 LLM
          │
          ├──→ 解析结构化 JSON 输出
          │
          └──→ memory/shortterm  记录 assistant 回复
                   │
                   └──→ 检查 token 预算 → 必要时触发压缩
          │
          ▼
   ┌─────────────┐
   │  Response    │  JSON { demand_analysis, products[], feasibility, missing_info }
   └─────────────┘
```

## 压缩流程

```
对话缓冲 token 数 > 预算阈值
         │
         ▼
   取最早的 N 轮对话 → LLM 摘要压缩
         │
         ▼
   原文丢弃，保留摘要
         │
         ▼
   缓冲 = 摘要 + 最近 K 轮对话
```

## 知识加载流程

```
服务启动
    │
    ▼
Wiki 适配器初始化
    │
    ├── 扫描 wiki 产品页面目录
    ├── 解析 frontmatter + markdown 章节
    ├── 构建内存索引（产品名→关键词→场景标签）
    └── 就绪
```
