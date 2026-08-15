# 记忆系统二期（memory-v2）

> 短期记忆 checkpoint 链的演进 + Agent 记忆能力。状态：**Done**
> （全部候选项已落地或消解：P0-P11，2026-08-14 一期收尾 goal）。

## 为什么有这个 plan

一期 checkpoint 链已落地（ADR-003/015）并经三轮审计加固。2026-08-14 用户
要求调研小米 MiMo Code 的实现作对照，发现我们的方向不同但存在真实缺口；
同日按 Agent 自评修复了检索精度/客户画像/追问闭环（已落地，见 roadmap）。
本 plan 承接**尚未落地的部分**。

## 调研结论：MiMo Code vs 本项目（2026-08-14）

来源：MiMo Code 官方博客《将编程 Agent 扩展到长程任务》+ sessions 文档 +
PR #1208 / #1636。

| 维度 | MiMo Code | 本项目现状 |
|---|---|---|
| checkpoint 触发 | 上下文用量阈值（20/45/70%）被动触发 | 每轮结束业务事件驱动（initial/followup/reanalysis） |
| 提取者 | 独立 writer subagent（LLM 总结，有改写偏差风险） | 无提取者——直接存结构化事实（analysis_submit 契约输出） |
| 窗口耗尽 | 核心问题，cycle + rebuild | 不存在（每轮 LLM 上下文有界，15 迭代内轮结束即弃） |
| 四层记忆 | checkpoint.md → MEMORY.md → global → history(SQLite) | checkpoint 链 → Wiki 长期记忆 → history(SQLite，无 Agent 查询工具) |
| notes.md | 主 Agent 唯一写通道，writer 消费路由 | 通道存在但无调用方（历史遗留） |
| 用户原话保真 | 专门 PR（#1208）保证 rebuild 注入逐字切片 | messages 库存原文，但 Agent 无法回捞 |

**判断**：cycle/rebuild 机制不适用于对话式场景（无窗口耗尽问题），不抄。
值得吸收的是三个思想：

1. **结构化契约优于 LLM 总结**——我们的 analysis_submit 天然满足（无改写
   偏差），保持。
2. **history 兜底工具是真缺口**——Agent 无法回溯原始对话轨迹。
3. **逐字保真原则**——checkpoint 注入用摘要没问题，但原文必须可回捞。

## 调研二：长期记忆 Wiki 生态（2026-08-14）

来源：Karpathy LLM Wiki gist（2026-04）+ 评论区一线实现踩坑（MindBase /
stigmergy / LWC / Provenance-First-Wiki / SIGN）+ 记忆系统三巨头对比
（Mem0 62.7k★ / Letta 24.1k★ / Zep-Graphiti 29.6k★，benchmark 全为自测，
唯一独立复现差 45 分——勿信分数，只看架构）。

**两大流派**：我们是 Karpathy 文件流派（markdown + 人审 + 实体盖章）。
另一派是服务化记忆系统：

- **Mem0**：被动抽取管线（LLM 抽事实 → 向量比对 → ADD/UPDATE/DELETE/NOOP）。
  应用零侵入，但每次写都是 LLM 判断（坏 UPDATE 静默污染）。
- **Letta**（MemGPT）：Agent 自管三层（Core 常驻 / Recall 可搜索 / Archival
  向量库）+ sleep-time compute（空闲时异步 agent 整理记忆）。
- **Zep/Graphiti**：时序知识图谱——事实带 valid_at/invalid_at，矛盾时
  **失效而非删除**，"过去相信什么、何时改变"是一等查询。

**社区踩坑实录 → 我们的对照**：

| 踩坑 | 社区解法 | 我们现状 |
|---|---|---|
| wiki 自引用污染（派生页与源页同权索引 → 编造滚雪球） | quote 指向不可变源的指针；禁止 wiki 页作为另一页的 grounding | ⚠️ pending 与 verified 同索引检索（有标记+降权，无结构性隔离） |
| 团队规模化（"坏页面变成公司信念"） | 人审落地 + diff 确定性门禁 + 实体出生须人批 | ✅ 已有（ADR-005 审核队列 + 权限矩阵）——超前 |
| 实体归属（读时猜"关于谁"会错） | 写时代码盖章 about，先查实体注册表 | ✅ 已有（typed entries 代码构建，不走模型推断） |
| 规模上限（~1000 文件崩） | 混合检索（BM25/向量+重排） | ✅ bigram+权重倒排已落地（2026-08-14） |
| 过时知识（lint 查不出"世界变了笔记没变"） | 声明携带 provenance，源变更级联标记 stale | ❌ 无 lint 无时效性 |
| 维护信任（"AI 背后改我笔记"） | ingest 先讨论要点→勾选计划→只写批准项 | 部分（审核队列覆盖 threat/industry，customer 直写） |

## 候选项（按优先级）

### P0：System Prompt 分层落地（ADR-016 v2，用户裁决，最优先）✅ 已落地 2026-08-14

见 [[decisions/README.md|ADR-016 v2]]。分层注入：L0 静态模板 +
L1 使用者画像（风格）+ L2 会话状态（checkpoint 链 + 客户身份一行）+
L3a 产品目录索引；其余知识全走工具。废除 AllKnowledge 全量注入与
客户画像内容注入；补会话头客户名 UI。P8 随之消解。

**落地证据**：assembler 目录索引（名字+Description+别名）替代
renderKnowledge 全量段（已删）；客户身份行注入；systemConstraints
强化「推荐前必须检索」；TestAssemblerProductCatalogOnly +
TestAssemblerCustomerInjection（身份行注入/画像内容禁入双向断言）；
TestMessageCustomerFieldPersisted（HTTP 端到端）；真实冒烟 prompt
3387→约 1794 字符（需求轮，smoke-p0.sh 可复现），LLM 行为正常（知道
客户身份并主动检索）。
Trace.SystemPrompt 改为聚合全部 system 消息（模板+动态注入均落库
回放可见）。

### P1：history_search 工具（Agent 记忆真空缺）✅ 已落地 2026-08-14

让 Agent 能查 SQLite 里的原始对话轨迹（messages + tool_calls），当
checkpoint 注入的摘要不够用时回捞细节。

**落地证据**：history.Store.SearchMessages（tenant 隔离 JOIN + 会话
范围可选 + LIKE + limit 封顶）+ TestSearchMessages（4 断言组）；
agent 层 history_search 业务工具（SetHistorySearcher 回调 + turnState
带 tenant/session）；TestAgentHistorySearch 集成（跨会话命中后基于
历史回答）；prompt 指引「回溯细节 → history_search」；main.go 接线。
语义对应 MiMo 的 history 工具；也是"跨会话客户上下文"缺口的另一半：
画像给"是什么"，history 给"聊过什么"。

### P2：✅ 已落地：Notes 全链删除（observe 已覆盖职责）

#### 原方案

Notes 通道启用或删除

`Manager.AppendNote/DrainNotes` 无调用方。两个方向二选一：

- 启用：作为 Agent 轻量草稿通道（类似 MiMo notes.md），暂存跨 checkpoint
  的零散观察，checkpoint 创建时消费
- 删除：dead code，清理（倾向此方向——memory_observe 已覆盖该职责）

### P3：✅ 已落地：链软上限 50 双端滚动归档（内存 compactChainLocked + SQLite 同规则裁行；LIMIT 负数 bug 修复）

#### 原方案

checkpoint 链截断策略

会话几百轮时链无限增长（内存 + SQLite 全量）。BuildContext 只取局部所以
token 无压力，但存储无界。方案：链长超过 N（如 50）时滚动归档老 followup
（保留 initial/reanalysis 骨架）。

### P4：✅ 已落地：SessionDetail.Checkpoints + Replay 页 checkpoint 行穿插（initial 前置 user/followup 尾随 assistant）

#### 原方案

checkpoint 可视化（回放页）

回放时间线表格目前只展示消息+工具调用。可加 checkpoint 事件行
（initial/followup/reanalysis + Analysis 摘要），让"记忆系统在做什么"
对销售可见。低优先，体验向。

### P5：✅ 已落地：RunLint 确定性检测（孤儿/残缺/别名冲突）挂 taskbg + POST /api/tasks/lint + 观测台展示

#### 原方案

Wiki Lint 健康检查（后台任务候选）

对应 Karpathy 三操作里的 lint，我们完全缺失。检查项：

- 同类型条目间的矛盾（如两个威胁条目描述重叠冲突）
- 孤儿条目（无 tags/keywords/别名，检索永远到不了）
- 该建未建（Agent 检索 miss 率高的关键词 → 建议建条目）
- 空摘要/空正文的残缺条目

实现形态：**后台任务**（ADR-015 的 TaskBackground 域，无记忆依赖的
一次性 LLM 调用）或手动触发的审核队列入口。产出是**审核建议**不是
直接写入（人审落地，沿用 ADR-005 模式）。

### P6：✅ 已落地：Entry 三时效字段 + 覆盖归档 .superseded-<ts>.md（archived+invalid_at+superseded_by 留痕）+ Load 排除归档

#### 原方案

事实时效性（invalid_at，Zep 思想）

现状：UpsertEntry 直接覆盖，客户需求变了老事实无迹可寻（memory_observe
的时间戳注记是唯一残迹）。售前场景真需求："这客户上个月要的是什么、
什么时候变的"。

方案（轻量版，不引入图数据库）：Entry frontmatter 加 `superseded_by`

- `valid_at/invalid_at`，UpsertEntry 检测到同 title 实质变更时旧版本
归档而非覆盖（history 表已有审核轨迹可关联）。查询默认排除已失效，
`memory_recall` 可显式带 `include_invalid=true` 看演变史。

### P7：记忆整理 Dream（Letta sleep-time / MiMo Dream 思想）→ 并入 P11

本项升级并入 P11 记忆管线（固化阶段即专职记忆 Agent），不再单列。

wiki 只增不练，长期使用积累重复与过时。定期（如每周）后台任务：
合并重复条目、验证 tags/keywords 有效性、压缩冗长 Notes、把多次出现
的 session 观察（memory_observe 注记）提升为正式条目（走审核）。

依赖：后台任务基础设施（与 P5 共用）；调度器（cron 或启动时检查）。

### P8：派生污染的结构性隔离 → 已由 ADR-016 消解

原方案（pending/verified 分区注入）作废：ADR-016 废除一切知识注入后，
system prompt 不再有知识污染面。剩余小尾巴：工具检索结果里 pending
条目带 status 字段传递（现状已满足，marshalHits 保留即可）。

### P9：✅ 已落地：CDA_DATA_DIR 三级优先 + XDG 默认 + 旧 ./data 就地兼容 + 服务端播种 + Dockerfile

#### 原方案

数据全局存放位置（部署可迁移性）

2026-08-14 用户发现：**wiki 与历史库存放在项目目录下**（`./data/wiki`、
`./data/history.db`），路径接线有三处绑死：

1. `config.json` 的 `wiki_dir`/`history_db` 字段（当前值即相对路径）
2. `internal/config/config.go` 默认值 `./wiki` / `./data/history.db`（空字段回填同值）
3. `scripts/run.sh` 播种逻辑（`cp -r ./wiki/*` 进运行目录）

**问题**：Docker 化后镜像重建即丢数据（wiki 增量 + 全部会话历史）；
项目目录迁移/重 clone 同样丢；备份策略无从谈起。种子 `wiki/` 与
运行数据混在 `data/` 下，且种子会被 git 追踪而运行数据被 ignore——
边界靠约定不靠结构。

**方案**（三级优先）：

1. 环境变量 `CDA_DATA_DIR`（部署层指定，Docker 用它指向 volume）
2. `config.json` 显式路径（绝对或相对数据根）
3. XDG 默认：`$XDG_DATA_HOME/customer-demand-agent/`（未设 XDG 则
   `~/.local/share/customer-demand-agent/`，Windows 沿用户目录约定）

数据根内布局：`wiki/`（运行副本）、`history.db`、`config.json` 也应
默认落这里（当前 config 在项目根，同样有迁移问题）。种子 `wiki/`
保留在项目内（git 追踪），**首次启动**播种到数据根、此后只认数据根。

**迁移兼容**：启动时检测项目内旧 `./data/` 存在且数据根为空 → 自动
搬迁 + 日志告知（一次性）；run.sh 的播种逻辑同步改造。

**Docker 约定**：`ENV CDA_DATA_DIR=/var/lib/cda` + `VOLUME /var/lib/cda`，
镜像无状态、状态全在卷上。

改动面：config.go（默认值+解析）、main.go（启动日志）、run.sh、
新增迁移逻辑 + 测试。

### P10：AI 记忆更新粒度补齐（用户命题 2026-08-14：应不应该≠能不能）

**✅ 核心项已落地 2026-08-14（本 goal）**：核实 execEnsure 对
threat/compliance/industry 一律置 pending（与旧条目状态无关）——AI
覆盖 verified 必然降级，行为正确，补 TestEnsureOverwriteVerifiedDemotes
锁定；人工通道 memoryUpsertReq 移除 status 传参（一律 verified，
pending 是 AI 专属语义），TestMemoryUpsertIgnoresStatusParam +
e2e 第 4 节改测 P10 新语义。**字段级 patch / 留痕 / 软删回收站**
（下述 2-4 项）仍为候选项。

用户提出：记忆更新能力上"理论上都应该能更新"。现状盘点：

- AI 通道：threat/compliance/industry/customer 可 upsert（同 title 整条
  覆盖）；product/user 只读（ADR-005 防编造——"应不应该"层面的收窄，
  不是能力缺失）
- 人工通道：六类全部可增删改查（HTTP API + 记忆库页）

**保持**：ADR-005 权限矩阵不动（product/user 的 AI 只读是审慎设计，
放开需用户明确裁决）。ensure=全量更新 / observe=追加注记的语义分离
保持。

**补齐项**：

1. **AI 更新的记忆可见性**：memory_ensure 覆盖已有 pending 条目时，
   修改内容对 Agent 立即可见（同库）；覆盖 verified 条目会降级回
   pending（现状如此？需核实 UpsertEntry 对已 verified 条目的状态
   处理——若直接保持 verified 则是漏洞：AI 改动未审核即生效）。
2. **字段级更新**：ensure 只能整条覆盖；观察 AI 实际使用，若频繁
   为改一个字段重写全篇，考虑支持 patch 语义（低优先，先观察）。
3. **人工通道的版本痕迹**：前端/HTTP 修改无任何留痕（谁改的、改了
   什么），与 AI 写入的 pending 留痕不对称。对策：Entry 加
   updated_by（ai/human）+ 变更摘要，或写操作记入 history 库。
4. **删除语义对称**：AI 只能删 customer；人工全删。产品误删无恢复
   （无版本）。对策：删除一律软删（archive）+ 回收站视图（与 P6
   时效性呼应——supersede 链就是版本史）。

### P11：✅ 已落地：taskbg 包（Runner+固化管线 observe→LLM 抽结构→pending 人审）+ GET /api/tasks + 手动触发

#### 原方案

记忆管线三阶段（捕获→固化→人审，方案待用户裁决）

用户命题（2026-08-14）：记忆写入是直接放还是专职 Agent 处理？AI 总结
保结构但丢细节；结构化才能快速检索——两难。

**现状**：主 Agent 对话中直接 ensure（兼职知识管理，MiMo 点名反对的
模式）；ensure 总结无 provenance，细节丢了不可回捞。

**方案**（三阶段管线，原料系统已有：history=逐字原文层、wiki=结构层、
observe=轻量注记）：

1. **捕获**（对话中）：主 Agent 只用 observe 近逐字记录（customer 画像
   例外可 ensure——操作数据字段简单）。零总结零结构化，不分心。
2. **固化**（后台，TaskBackground 域第一个正式居民）：专职记忆 Agent
   读观察 + history 原文 → 产出结构化卡片（summary/keywords/tags）→
   **每条 claim 带 provenance**（session_id/message_id，总结丢细节可
   回捞，Provenance-First-Wiki 教训）→ 走 pending 人审（F4 内联弹窗
   承接）。
3. **取用**：结构化字段就是检索高权重来源——观察原文检索性差不要紧，
   固化后自然可检索。

对应"需要 AI 总结 vs 直接放"：捕获层=无损直接放；固化层=AI 总结但
带 provenance + 人审。取舍从写入瞬间移到后台批处理。

= MiMo writer subagent + Letta sleep-time + P7 Dream 的合流升级；
P7 并入本项。主 Agent 对 threat/compliance/industry 的 ensure 是否
收紧为仅 observe，随本项裁决（行为变更，需用户确认）。

**人工写入通道对齐（2026-08-14 用户追问：AI 有处理，用户呢？）**：
现状 handleMemoryUpsert 裸奔——直接写库、status 可随意传（默认
verified）、零校验零留痕零结构化帮助；结构化高权重字段（capabilities/
keywords/typical_signs）前端编辑器不提供，新建条目检索质量全靠手写
summary/tags。对齐方案（并入 P11 管线，人工侧）：

1. **结构化编辑器**：记忆编辑页按类型渲染 typed 字段表单（产品的
   capabilities 列表、威胁的 typical_signs、客户的 pain_points…），
   让人工也能产出高检索权重结构（前端为主）。
2. **可选 AI 辅助**：粘贴长文档时提供「AI 帮我抽结构」按钮（一次
   TaskBackground 调用：抽 summary/keywords/tags 供人工确认后提交，
   不是静默改写）——人工输入+AI 建议的混合模式。
3. **留痕对齐**：与 P10-3 同一件事（updated_by=human/ai + 变更记录）。
4. **直接 verified 合理性**：人工=可信源，默认 verified 保留；但
   status 参数应从请求体移除（前端不该能传 pending，那是 AI 专属）。

## 明确不做（已有裁决）

- **不做 RAG/向量检索**（ADR-002，一期 bigram+权重已显著改善关键词检索）
- **不做 cycle/rebuild**（对话式场景无窗口耗尽问题）
- **反馈闭环 analysis_feedback**（延期清单，等一期真实使用数据）
- **不引入 Mem0/Letta/Zep 等外部记忆系统**（ADR-008 自研轻量底座；
  三巨头的被动抽取/Agent 自管/图数据库均与我们的 wiki+人审+实体盖章
  模型冲突，且 benchmark 不可信、写路径全是 LLM 调用成本）
- **不上图数据库**（P6 用 frontmatter 时效字段达成 Zep 的核心收益，
  避免图库运维负担）

## 已在别处落地（勿重复）

2026-08-14 按 Agent 自评修复（属 agent-autonomy-refactor 尾巴，非本 plan）：
CJK bigram tokenize + 字段加权索引、missing_answer 追问闭环工具、
UpsertEntry typed=nil 索引漏建修复、analysis_submit 覆盖语义澄清。
~~会话客户画像注入（SetCustomerResolver）~~ → **同日被 ADR-016 废除**
（知识不走注入，改 memory_search type=customer）。

## 下一步

**已完成（2026-08-14 先锋批 goal）**：P0 分层落地（ADR-016 v2 生效）、
P1 history_search、P10 核心门禁、P8 悬案核实（随 ADR-016 消解）。
可复现冒烟：`./scripts/smoke-p0.sh`（需真实 LLM 的运行实例）。

**候选批次**：P9 数据全局位置（Docker 迁移刚需）→ P11 记忆管线三阶段
（方案待用户裁决：主 Agent ensure 是否收紧为仅 observe）→ P5/P6 后台
任务基础设施（TaskBackground 域，P11/P5 共用）。
