# Plantree

> 客户需求分析智能体 — 项目规划根

## 注册根

| Plan | 状态 | 当前阶段 | 最后落地 | 下一目标 |
|------|------|----------|----------|----------|
| [[plans/customer-demand-agent/README.md\|客户需求分析智能体]] | Landed（一期） | — | 一期全栈实现 + 安全加固 + 基线清零（2026-08-13） | — |
| [[plans/agent-autonomy-refactor/README.md\|Agent 自主化重构]] | Done | 四 Phase 全部落地 | 2026-08-14 全套验证通过（含审计修复：回放 message_id 归属、LastAnalysis 跨聊天、前端竞态、空响应契约） | — |
| [[plans/memory-v2/README.md\|记忆系统二期]] | Active（先锋批落地） | **P0/P1/P10 核心已落地（2026-08-14，ADR-016 v2 生效：需求轮 prompt 全量时代 3387 → 目录索引时代 ~1794 字符，知识全走工具）**；P2-P9/P11 Researching | 2026-08-14 先锋批 goal 完成（目录索引+身份行+history_search+P10 语义收紧，e2e 23/0，冒烟 smoke-p0.sh 3/3） | 下一批：P9 数据位置 / P11 记忆管线（待裁决） |
| [[plans/conversation-ux/README.md\|对话交互增强]] | Active | **F0+F1 已落地 2026-08-14**（Run 任务模型：202+订阅流+cancel+编辑重发+断连运行继续，smoke-f0 8/8）；F2-F4/G/H2 待做 | 2026-08-14 F0+F1 goal（commit 78264ea，e2e 25/0） | F3 ask_user + F4 内联审核（共用卡片组件）；F2 截断重发验证够用否 |
| [[plans/console/README.md\|控制台化]] | Shaping | C1 配置中心（prompt/路由/参数/数据位置）+ C2 可观测（后台任务/LLM 审计）+ C3 prompt 配置化 | 2026-08-14 独立立项（源起 H1/H3 升级：后端配置与运行全面前端可见） | C1 只读可直接开工；裁决 C3 外置形态 |
| [[plans/frontend/README.md\|前端工作台]] | Shaping | **IA 总蓝图**：三层可见性（对话内/会话飞行记录仪/系统观测台），六页收敛五区，G5 否决（对话与回放不合并） | 2026-08-14 立项，汇总三 plan 前端形态 + 后端依赖 API 缺口清单 | 跟随 F0/G1 先行；观测台新页 |

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
