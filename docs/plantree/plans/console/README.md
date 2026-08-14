# 控制台化（console）：后端配置与运行的前端可见

> 用户总纲：**尽可能把后端的配置、每个环节的 system prompt、正在跑的
> 任务都在前端展示**——前端是后端的控制台，不是黑盒。
> 状态：**Shaping**。2026-08-14 用户提出（源起 H1/H3，升级独立立项）。

## 现状盘点（2026-08-14 核实）

**已有**：Settings（模型管理+测试+获取模型列表、路由表 default/routes、
按任务类型指定模型）；审核队列页；回放页可看每轮实际 system prompt 实例。

**缺**：

- system prompt 模板硬编码在 assembler.go 常量（systemRole/
  systemAutonomy/systemGoals/systemConstraints + 分级输出模板
  styleForLevel），前端看不到，改不了
- fallback 链/重试参数（router.fallback：max_retries/backoff_base_ms/
  chain）在 config 但前端未展示
- 行为参数（MaxIterations=15、检索字段权重、Answered 聚合上限 20、
  prompt 各注入段的 token 上限）散落代码常量
- 数据位置（wiki_dir/history_db/default_tenant/default_user/
  llm_timeout_sec）前端不可见（联动 P9 数据根）
- **零后台任务视图**：当前后端无任何常驻后台任务（唯一 goroutine 是
  优雅关闭）——但 F0 Run、G4 标题生成、P5 Lint、P7 Dream 都在路上，
> 届时「正在跑什么」必须有地方看
- LLM 调用审计：每轮实际用了哪个模型（路由命中/回退）、耗时——
  trace 里没记，回放页看不到

## 方案

### C1 设置页扩为配置中心

新 tab 或分区，全部先「只读展示」，编辑按项裁决：

- **系统提示词**：分块展示模板（角色/自主性指引/目标/约束/分级输出
  风格），每块标注在拼装中的作用 + 运行时注入的部分（知识库/客户画像/
  分析上下文是动态的，模板是静态的，分开说清楚）
- **模型与路由**：现有 + fallback 链可视化（A 失败→B→C，重试次数/
  退避参数）
- **运行参数**：MaxIterations、检索权重、聚合上限、超时等常量归拢成
  一张表（只读起步）
- **数据与租户**：wiki_dir/history_db/tenant/default_user + 记忆统计
  （各类型条目数：产品 8/威胁 7/...）

### C2 运行时可观测

- **后台任务视图**：任务列表（类型/状态/触发时间/结果摘要）——空列表
  也要先立住，F0/P5/P7/G4 逐个挂上；含手动触发入口（Lint 现在就能
  手动跑）
- **Run 实时状态**（联动 F0）：正在跑的会话、当前轮次、事件流回放
- **LLM 调用审计**：model manager 在 trace/Event 里补记每轮模型命中
  （路由/回退）与耗时；回放表格与 C2 都消费
- **系统健康**：wiki 加载状态、checkpoint 链深度、history 规模、
  启动配置快照（打码 api_key）

### C3 Prompt 模板配置化（后端改造，支撑 C1 编辑态）

模板外置（config.json 新字段或 prompts/ 文件，二选一需裁决）+ 热加载

- 版本记录（改动留痕可回滚）+ 校验（模板必含占位符检查）。编辑权限
  归人工（ADR-005 精神：行为资产人工维护），前端编辑器是自然延伸。

## 顺序与依赖

C1 只读展示（纯前端 + 少量只读 API）→ C2 任务视图骨架（等 F0）→
C3 配置化（独立后端改造）→ C1 编辑态（等 C3）。
H2（意图判定可见）仍留 conversation-ux（属对话流，非配置）。

## 下一步

用户裁决：C3 外置形态（config 字段 vs prompts/ 目录）；C1 编辑态是否
一期就要。裁决后 C1 只读部分可直接开工。
