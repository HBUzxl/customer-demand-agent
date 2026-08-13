# 客户需求分析智能体

> 基于[[第一组—客户需求分析智能体一页纸规划]]的工程实现计划。

## 范围

**一句话定位**：面向销售与售前团队的 AI 解决方案助手，通过理解客户业务场景和安全痛点，辅助完成安全需求识别、风险分析、产品匹配和初版方案生成。

**一期目标**：本地 HTTP 服务跑通核心链路——输入客户沟通文本 → 输出结构化分析（需求理解 + 产品匹配 + 可行性判断）。

## 文件地图

| 文件 | 角色 |
|------|------|
| [[roadmap.md]] | 当前阶段、进度、TODO |
| [[topics/memory-system.md|记忆系统设计]] | 长期记忆 + 短期记忆 + 拼装层 |
| [[topics/memory-tools.md|Memory Function Calling]] | AI 常驻的 6 个记忆管理工具 |
| [[topics/model-management.md|模型管理系统]] | 模型注册/路由/回退/调用 |
| [[topics/agent-autonomy.md|自主 Agent 设计]] | 自主循环，非 workflow |
| [[topics/history-tenancy.md|历史记录与多租户]] | 完整轨迹持久化 + 使用者隔离 |
| [[topics/wiki-knowledge.md|Wiki 知识库]] | 产品页面结构 + 检索策略 |
| [[topics/agent-core.md|Agent 核心]] | 自主循环 + 输出解析 |
| [[open-questions.md]] | 未决问题 |
| [[decisions/]] | 架构决策记录 |

## 阅读路径

1. 先看 [[roadmap.md]] 了解当前进度
2. [[topics/memory-system.md]] 理解记忆系统——整个 Agent 的脊梁
3. [[topics/memory-tools.md]] 理解 AI 能用什么工具操作记忆
4. [[topics/model-management.md]] 理解模型怎么调度
5. [[topics/agent-autonomy.md]] 理解为什么是自主 Agent 而非 workflow
6. [[topics/history-tenancy.md]] 理解历史记录和多租户
5. [[topics/agent-core.md]] 理解编排逻辑
5. [[open-questions.md]] 看看还没定的事
