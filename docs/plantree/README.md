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
