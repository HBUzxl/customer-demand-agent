# 审核机制重构：写入即审批

> 审核从「事后标记」改成「写入门禁」。状态：**Done**（2026-08-15 二期 goal 已落地：
> visibleToAgent 门禁 + 对话内审批弹条 + ADR-005 修订，均有集成测试覆盖）。

## 用户裁决（2026-08-15 原话）

> 「如果他要往记忆库里填东西，是需要审核的对吧？按理说，应该是我通过
> 之后，内容才会进入记忆库。但是现在我看，好像是直接填进记忆库里了，
> 根本没管审不审核。」
>
> 「你们直接在他往里填东西的时候，在输入框上面弹出一个像权限一样的
> 东西，让用户选择不就好了吗？你为什么非得要出那个页面？那个页面有屁
> 用啊！他现在要写东西，我还得跳转到另一个页面，点击批准，然后再回到
> 那个对话里。」

## 现状核实（2026-08-15，实现前盘点——以下缺陷已随本期修复）

| 环节 | 现状 | 证据 |
|---|---|---|
| AI 写入（threat/compliance/industry） | 标 pending_review **立即落库** | tools.go:117 needsReview → StatusPendingReview |
| 检索 | **pending 立刻可检索**（只排除 archived） | retriever.go:157 `e.Status != StatusArchived` |
| prompt 注入 | **pending 也进 L2/L3a**（不筛 status） | assembler GetCustomerProfile/AllKnowledge |
| 降权 | **未实现**（ADR-005 说「可用但降权」，没有降权代码） | 全库无 relevance 惩罚逻辑 |
| /review 页 | 只是状态翻转器 | approve→verified / reject→删 |
| F4 内联审核卡 | 已存在但藏在工具时间线里，不弹不挡 | ReviewCard 在 ToolTimeline 内 |

**结论（实现前）：审核不是门禁，是事后标记——AI 写入内容在等待审批期间已被检索/注入/影响分析。
✅ 本期已修复：`visibleToAgent` 谓词使 pending 对 Agent 完全不可见（SearchEntry/ListEntry/GetCustomerProfile 全排除），批准即生效；`TestReviewGatingVisibleToAgent` + `TestReviewGatingCustomerProfilePending` 集成测试覆盖；对话内 sticky 审批弹条替代 /review 页（该页已删）。**

## 已落地实现（原方案 F1/F2/F3 全部完成）

### F1. 写入门禁：待审条目对 Agent 不可见

- `SearchEntry`/`ListEntry`/assembler 注入（GetCustomerProfile、
  AllKnowledge、产品目录）排除 `pending_review`（与 archived 同待遇）。
- 例外：F4 审批卡/审核页本身要能列出 pending（走独立查询）。
- AI 写入后工具 result 提示语改为「已暂存待审——你暂不能检索到它，
  用户批准后生效」。
- 语义影响：同一轮里 Agent 写完再 search 验证会 miss——prompt 要加
  一句指引（「自己刚写的条目审批前检索不到是预期」）。
- **审批生效后立即对 Agent 可见**（approve 只是状态翻转，无需重载）。

### F2. 审批交互：对话内权限式弹条（去跳页）

- 待审条目产生时，**对话输入框上方浮出审批条**（sticky，不遮内容）：
  「📝 Agent 请求写入 [type] 「title」 [预览] ｜ 批准 / 拒绝」。
- 审批条不处理就一直在（刷新后回放也重现——按条目 pending 状态驱动，
  非事件驱动）；多条待审堆叠展示，处理一条少一条。
- 复用现有 ReviewCard 的 approve/reject API（零后端改动——F4 已铺好）。
- /review 独立审核页**已删除**（用户追加裁决「不要有单独的页面」）——对话内
  审批弹条是唯一入口；后端 /api/review/* API 保留供弹条调用。

### F3. ADR-005 语义修订

- 原文「AI 可写（可用但降权）」→ 修订为「AI 写入需人审生效：
  pending 期间对 Agent 不可见，批准即生效，拒绝即弃」。
- decision 文档补充 2026-08-15 裁决记录。

## 非目标

- 不做批量审批工作流（堆叠卡逐个点，量级不需要）。
- 不做审批超时自动通过/拒绝（保持显式人为准）。

## 已通过的验证（均落地）

- F1：集成测试——ensure 写 threat → 立即 SearchEntry 同关键词 **miss**
  → approve → SearchEntry **hit**；assembler 注入断言 pending 不在。
- F2：前端 tsc/lint/build 绿；手动冒烟：对话中让 Agent 记一条行业趋势
  → 审批条浮现 → 批准 → 条目 verified 且后续轮 Agent 能检索到。
- F3：ADR 文档修订 diff。
