# Plantree

> 客户需求分析智能体 — 项目规划根

## 注册根

| Plan | 状态 | 当前阶段 | 最后落地 | 下一目标 |
|------|------|----------|----------|----------|
| [[plans/customer-demand-agent/README.md\|客户需求分析智能体]] | Landed（一期） | — | 一期全栈实现 + 安全加固 + 基线清零（2026-08-13） | — |
| [[plans/agent-autonomy-refactor/README.md\|Agent 自主化重构]] | Done | 四 Phase 全部落地 | 2026-08-14 全套验证通过（含审计修复：回放 message_id 归属、LastAnalysis 跨聊天、前端竞态、空响应契约） | — |
| [[plans/memory-v2/README.md\|记忆系统二期]] | **Done** | **P0/P1/P10 核心已落地（2026-08-14，ADR-016 v2 生效：需求轮 prompt 全量时代 3387 → 目录索引时代 ~1794 字符，知识全走工具）**——P0-P11 全部落地/消解 | 2026-08-14 一期收尾 goal：P2/P3/P4/P5/P6/P9/P11 补完（e2e 29/0、四冒烟脚本全绿） | — |
| [[plans/conversation-ux/README.md\|对话交互增强]] | **Done** | **F0+F1 已落地 2026-08-14**（Run 任务模型：202+订阅流+cancel+编辑重发+断连运行继续+多轮隔离+结构化失败语义，smoke-f0 11/11）——F0/F1/F3/F4+G2/G4/G6/G8+H2 全落地 | 2026-08-14 一期收尾 goal：F3/F4/G 系列补完（F2 完整版裁决不做） | — |
| [[plans/console/README.md\|控制台化]] | **Done** | C1 配置中心（prompt/路由/参数/数据位置）+ C2 可观测（后台任务/LLM 审计）+ C3 prompt 配置化 | 2026-08-14 一期收尾 goal：C1/C2/C3 落地（四个 console API+观测台+prompt 外置快照） | — |
| [[plans/frontend/README.md\|前端工作台]] | **Done** | **IA 总蓝图**：三层可见性（对话内/会话飞行记录仪/系统观测台），六页收敛五区，G5 否决（对话与回放不合并） | 2026-08-14 立项；同日收尾 goal 落地主体（订阅模式/观测台/配置中心/回放 checkpoint/审核融入对话/草稿/搜索/滚动记忆） | 残余归档二期候选 |
| [[plans/review-gating/README.md\|审核机制重构：写入即审批]] | **Done** | visibleToAgent 门禁（pending 对 Agent 不可见，批准即生效）+ 对话内 sticky 审批弹条（/review 页已删）+ ADR-005 修订；TestReviewGatingVisibleToAgent + TestReviewGatingCustomerProfilePending 集成测试覆盖 | 2026-08-15 用户裁决（「按理说我通过之后才进记忆库」「直接在输入框上面弹个东西」） | — |
| [[plans/checkpoint-tree/README.md\|Checkpoint 语义重构：对话版本树]] | **Done** | 用户裁决三层：①概念统一「与 Agent 的对话」（每轮必有 checkpoint，废弃纯聊天轻量说法）②checkpoint=对话版本的 git：链→树（ParentID+BranchID，编辑重发=开分支不再截断删）③对话内版本树 UI+任意节点一键回到；含 F1 编辑重发语义变化/断点续传/LastAnalysis 适配 | 2026-08-15 用户裁决（「只有一种概念叫与 Agent 的对话」「像 git 的树状结构，从第二轮重新问=新分支」） | 方案已定—（数据模型/概念/UI 三层） |
| [[plans/wiki-hygiene/README.md\|Wiki 卫生与客户记忆治理]] | **Done** | 真实使用暴露三类库污染（自测残留 4+冒烟污染 11+类型错 1）+ 产品缺陷：同客户重复建条目无复用机制（7 变体）；F1 清理/F2 ensure 前置检索+prompt 续写指引/F3 审核类型徽章+重叠提示/F4 自测命名约束 | 2026-08-15 从会话 sess_cfdc2e91beb0 挖出 | — |
| [[plans/ui-copy-cleanup/README.md\|UI 文案去废话]] | **Done** |  |  |  |
| [[plans/verify-close/README.md\|终态验证]] | **Done** | 全套门禁+四脚本一键复现（verify-all.sh 13 项/verify-plan-commits.sh 每 plan 独立 commit 机械复核 13/13） | 2026-08-15 goal 收口 | 复跑即验 |
| [[plans/memory-search-ia/README.md\|记忆库搜索与信息架构]] | **Done** | 用户问「搜的是 6 张卡片还是底下文章」——数据层核实为全量搜索（FAQ 子文档/正文词实测命中），真问题在 UI：①搜索范围不透明（结果平铺主/子混排无类型标注+形态突变无解释）②卡片无「入口 vs 文章」分类（产品主卡是入口点进去是目录，文章条目点进去是正文，长得一模一样）；方案：搜索语义显性化（结果头「全库搜索 N 条含子文档」+类型徽章+tab 过滤器）+入口/文章视觉分类（图标语义全库统一）+无结果 CTA 预填新建 | 2026-08-15 用户反馈（「应该搜索的是文章才对；有些卡片是文章有些是入口」） | — |
| [[plans/console-config/README.md\|配置中心可配置化]] | **Done** | 用户裁决「全是写死只读的没啥用」→ 三态改造：①开放可配（llm_timeout/agent_max_iterations 提升为 config 字段+回退链+记忆行为参数，全部可选带默认值走 PUT /api/config 热更新）②隐藏（Prompt 分层视图/工具提示词/记忆统计——内部调试信息撤出配置中心）③只读（数据位置带迁移提示）；Prompt 编辑不做（一期排除项） | 2026-08-15 用户裁决（「该开放的开放，不该开放的全省略掉」） | — |
| [[plans/sidebar-models-ux/README.md\|侧栏与模型管理体验修复]] | **Done** | ①侧栏收缩态完整可用（现在 collapsed 直接不渲染会话列表——无法切换会话；改收缩态圆点头像列+tooltip+运行中绿点+搜索词清理）②模型配置面补全（context_window/timeout_sec/max_retries/remark/enabled 五新字段零迁移默认值兼容+必选红星/可选折叠高级区）③获取模型改下拉选择（现在 window.prompt 手动输入——后端已返回全量列表纯 UI 问题） | 2026-08-15 用户反馈（「收缩后只能新建」「缺少最大上下文等，可配置的应该是全的」「应该出现下拉列表而不是弹窗手输」） | — |
| [[plans/memory-management/README.md\|记忆管理增强]] | **Done** | ①产品计数修正（docCount 子文档+1 把主文档算进去——万象「1 篇」实为主文档自身，无子文档不该显示篇数）②主产品页显示正文（现只显示子文档卡片）③六类型记忆条目多选+批量删除（复选框+全选+ConfirmDialog+allSettled 汇总，与 session-management 同套交互） | 2026-08-15 用户反馈（「说有一篇文章但里头啥也没有」「每一个记忆都需要多选」） | — |
| [[plans/session-management/README.md\|会话管理增强]] | **Done** | 应用内确认弹窗（替换两处 window.confirm + alert）+ History 批量选择/批量删除（复选框+全选+Promise.allSettled 循环单删，不建后端批量端点）+ F2b 会话搜索升级标题→内容（新 /api/sessions/search 复用 SearchMessages LIKE+snippet 高亮，标题命中优先）+ F2c 会话列表显示绑定客户徽章+长名截断（侧栏 conv-item 加客户徽章截 6 字/History 列 ellipsis；无绑定不显示不占位）+ F3 客户绑定 Agent 自主化（删手动 chip，session_bind_customer 工具+ask_user 问客户，Agent 行为；绑定=L2 画像注入无其它副作用）+ F4 消息复制按钮 | 2026-08-15 用户提出（一期收尾后首项：「想从头来需要批量删除，做不到」；二批：客户绑定自主化+复制按钮） | — |
| [[plans/de-tenancy/README.md\|移除多租户]] | Done | 四层去 tenant 落地（X-Tenant-ID 忽略、列保留、e2e 26/0、grep 无功能残留） | 2026-08-14 收尾 goal task 1 | — |

## 基线

项目全局上下文，计划根通过链接引用：

- [[baseline/README.md|基线索引]]
- [[baseline/module-map.md|模块地图]]
- [[baseline/runtime-flows.md|运行时流程]]
- [[baseline/storage-and-state.md|存储与状态]]
- [[baseline/test-and-release-gates.md|测试与发布门禁]]
- [[baseline/risk-hotspots.md|风险热点]]

## 想法收件箱

- [[ideas/inbox.md|Inbox]]

## 如何阅读

1. 从上方注册表找到目标计划
2. 进入计划根 `README.md` 了解范围、文件地图和阅读路径
3. 阅读 `roadmap.md` 了解当前进度
4. 按需进入 `topics/`、`decisions/`、`open-questions.md`

## 权威顺序

1. 本文件（根注册表）
2. `baseline/` — 项目全局事实
3. `plans/<plan>/README.md` — 计划范围与文件地图
4. `plans/<plan>/roadmap.md` — 当前阶段、进度、TODO
5. `plans/<plan>/topics/` — 领域专题
6. `plans/<plan>/decisions/` — 已定决策
7. `plans/<plan>/open-questions.md` — 未决问题
