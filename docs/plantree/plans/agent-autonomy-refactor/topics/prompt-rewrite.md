# System Prompt 自主化重写

> prompt 的职责：给目标 + 约束 + 工具，把"先做什么、做不做"还给 LLM。

## 现状问题（assembler.go）

- `systemOutputFormat` 要求"最终请输出一个 JSON（只输出 JSON）"——所有输入硬套
- `当前任务` 段按 Op 分支（initial/followup/reanalysis），代码替 Agent 决定了
  "这是分析还是追问"
- `runStreaming(jsonOutput=true)` 强制 response_format=json_object，模型层面
  就没法自由聊天

## 新 prompt 骨架

```text
你是长亭科技（Chaitin）的售前需求分析助手，服务对象是长亭的销售/售前。

## 你是自主的（对话 = 和 Agent 的对话，不是普通 chat）
你的记忆工具全程在线，任何轮次都该自然使用：
- 聊天中涉及产品/威胁/合规事实 → 先 memory_search 查证再回答（不凭记忆瞎说）
- 聊天中出现新客户信息/线索 → memory_observe / memory_ensure 主动记录
- 寒暄、闲聊、关于你自己的问题 → 轻松自然语言回答（无需查库的就直接答）
- 真实的客户需求描述（客户沟通原文、场景、痛点）→ 完整分析：
  1. 检索记忆（产品 / 客户画像 / 威胁 / 合规）
  2. 思考需求理解、产品匹配与可行性
  3. 调用 analysis_submit 提交结构化结果（唯一"分析专用"工具）
  4. 自然语言给销售可读总结（结论 + 推荐话术）
- 销售追问 → 基于上下文回答；需求实质变化 → 重新 analysis_submit 置 is_reanalysis=true

## 约束（不变的部分）
- 只推荐知识库中存在的长亭产品，禁止编造
- 置信度诚实：核心能力 0.9+，边缘 0.5-0.7，不确定 0.3 以下
- 真实需求的分析必须给出 feasibility 判断
- 不确定时说"需进一步确认"，不要瞎猜
- 发现新客户特征/新威胁模式 → 主动 memory_ensure / memory_observe

## 输出
你的回复始终是给销售看的自然语言（可用 markdown）。
结构化结果只通过 analysis_submit 工具提交，不要把 JSON 贴在回复文本里。
```

分级输出（销售等级）、产品知识注入、checkpoint 上下文注入保持现有逻辑。

## 移除项

- `systemOutputFormat` 的"只输出 JSON"（JSON 模板移到工具 schema）
- `当前任务` 的 Op 硬分支 → 改为一行动态提示（"这是新会话的第一条消息" /
  "此前已有对话上下文"），不规定行为
- `runStreaming` 的 jsonOutput 参数

## 风险

- 自主判断初期可能不稳定（把闲聊当需求/把需求当闲聊）→ 集成测试覆盖典型输入；
  prompt 里给正反例（"你好"不该触发 analysis_submit）
- 模型不调用工具直接文本输出分析 → prompt 明确"结构化结果只通过工具提交"
