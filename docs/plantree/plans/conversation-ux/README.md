# 对话交互增强（conversation-ux）

> 对话流的操作控制：服务端任务模型、停止/编辑重发、回溯重刷、
> 用户选项交互、前端系统性补课。状态：**Done**（F0/F1/F3/F4 + G2/G4/G6/G8 + H2
> 全部落地，2026-08-14；F2 完整版经裁决不做——轻量截断重发已够用）。2026-08-14 用户实测提出，同日扩充
> 架构级发现。

## 问题（已代码核实，全部缺失）

0. **运行绑定连接（架构级，最重）**：`agent.Message(r.Context(),...)`
   跑在 HTTP 请求 context 里——刷新/切会话/关标签页 = 断连 = **后端
   执行直接死**。前端 messageStream 活在组件 send() 里，切换会话时
   loading 守卫挡住新会话加载（显示旧会话消息）或清空（流结果丢失）。
   后端无任何 Job/Run 实体。**用户判断：跑的状态属于服务端——只要
   不主动停止，客户端怎么折腾都不应影响运行。**
1. **无法停止**：前端无 AbortController，SSE 开流后只能等完或刷新
   （刷新=断连，后端不写 [失败] 但该轮 user 消息已成孤儿）。
2. **无法编辑重发**：没有消息级删除/截断接口，发错的用户消息改不了。
3. **无法回溯重刷**：消息线性 seq 链，无 parent/分支/截断概念；对某个
   回答不满意，想回到之前的消息重新生成——做不到。
4. **无法给用户选项**：Agent 没有 ask_user 类工具；missing_info 只能
   文本罗列，用户不能点选（比如"部署环境是公有云还是私有云？"）。

## 方案草案

### F0：服务端任务模型（地基，先做）✅ 已落地 2026-08-14

**落地证据**：RunManager（channel/http/runs.go——Start/Cancel/Subscribe/
DropSession，per-session 事件缓冲 500 条带 seq，每会话单 Run 并发 409）；
POST /api/message→202{session_id,run_id}（executeTurn goroutine 执行+落库，
与连接解耦，显式取消不写[失败]）；GET /api/sessions/{id}/stream?since=N
订阅式 SSE（replay→live→结束补尾）；POST .../runs/{run_id}/cancel；
GET /api/sessions/{id}/running（列表附 running）。前端：submitMessage/
subscribeStream（AbortController 只退订）/cancelRun/truncateMessages；
切回会话恢复（sessionGet 重建+running 查询+resume 续订）；侧栏运行绿点
（自适应轮询 3s/15s）。测试：TestMessageEndpoint202/Busy409/
RunSurvivesDisconnect/RunCancelExplicit/MessagesTruncate + e2e 26/0 +
smoke-f0.sh 11/11（断连运行继续/replay/游标增量/cancel 404 语义/截断/重发全链路）。

把「执行流」变「订阅流」——运行是服务端一等实体，连接只是视图：

- `POST /api/message` → 创建 Run（服务端 goroutine），**立即返回**
  `{session_id, run_id}`（202 Accepted）；执行独立于任何连接。
- **RunManager**（channel 层新组件）：每会话串行执行（排队或拒绝
  并发，简单优先）；Run 内部 context 取消 = 唯一停止途径；事件写
  per-session **事件缓冲**（内存 ring buffer，带递增 seq；重启即失，
  已完成结果在 history 库不丢）。
- `GET /api/sessions/{id}/stream?since=N`（SSE 订阅）：先按 seq replay
  缓冲中的历史事件，再续传 live——刷新/切换/重连任意次都无缝。
- 会话列表与详情带 `running` 状态；侧栏会话上显示运行指示。
- 停止 = `POST /api/sessions/{id}/runs/{run_id}/cancel`（取消 Run
  context），断连不再隐式中断。
- messageCore 的落库逻辑（system prompt/assistant/tool_calls）移入
  Run 完成回调，与连接彻底解耦。

注意：ADR-015 的「前台全记忆」不变——Run 仍是完整 Agent 循环；变的
只是执行宿主从请求变任务。

### F1：停止 + 编辑重发（依赖 F0）✅ 已落地 2026-08-14（随 F0 同批）

**落地证据**：发送按钮 loading 态变红色方块「停止」（cancelRun+订阅结束
+assistant 中断态"（已停止）"）；user 消息 hover「✎ 编辑重发」（DELETE
/api/sessions/{id}/messages/after?seq=N 三表联删——messages+tool_calls
(message_id 关联)+checkpoints（链式整体作废）；运行中 409 先停止）+
原文填回输入框重发。

- **前端**：messageStream 内部用 AbortController；发送按钮旁/流式中
  显示「停止」；停止后该轮 assistant 显示"已中断"，user 消息气泡上给
  「编辑重发」（把原文填回输入框）。
- **后端**：`DELETE /api/sessions/{id}/messages/after?seq=N`——删除该
  seq 之后的消息 + tool_calls + checkpoints（孤儿清理 + 截断复用同一
  接口）。编辑重发 = truncate 到该 user 消息前 → 重新 POST。
- 客户端断开已有正确处理（不写 [失败]），停止即断连，后端自然收尾。

### F2：回溯重刷（两个档位，需用户裁决）

- **轻量：截断重发（推荐先做）**——任意历史 user 消息挂「从这里重刷」
  按钮 = F1 的 truncate 接口 + 重发。线性体验，实现简单，覆盖 90% 场景。
- **完整：消息树（ChatGPT 式分支）**——messages 加 parent_id，重刷/
  编辑产生兄弟节点，UI 左右箭头切换版本。数据模型改动大：checkpoint
  链也要跟着分支（checkpoint 需记 message 锚点），回放/重建全要适配。
  **或走 MiMo 路线**：「从这里分叉新会话」——复制前缀到新 session
  （fork），原会话不动。分支=新会话，模型零改动。

### F3：ask_user 选项交互 ✅ 已落地（工具+SSE+选项卡+missing_answer 联动）

- 新业务工具 `ask_user`（与 analysis_submit 同层，agent 拦截）：
  params `{question, options: [{label, value, description?}]}`。
- **轮次终止式**（推荐，无服务端等待状态）：Agent 调用 ask_user →
  executeTool 拦截记录 pending 问题 → 本轮正常结束（content 可带引导
  语）→ SSE 发 `ask_user` 事件（question+options）→ 前端渲染成按钮组
  → 用户点选 = 把 value 作为普通消息发回 → checkpoint followup 的
  Question/Answer 天然记录问答，BuildContext 注入最近 pending 问题。
- 备选：服务端 hold SSE 等用户答复再续循环——状态复杂（连接断开怎么办），
  不推荐。
- 与 missing_info 联动：prompt 指引"需要客户确认的关键信息用 ask_user
  给选项"，答案自动走 missing_answer 闭环。

### F4：内联审核 ✅ 已落地（needs_review+审批卡+回放兼容+/review 兜底）

用户明确：**审核不要单独页面**——攒一堆待审项等用户主动去翻，
使用负担大且容易积压（"一看审查，怎么那么多东西"）。改为**对话内
实时弹窗**，像权限控制那样：

- Agent 调 memory_ensure/observe 写入需要审核的类型（threat/compliance/
  industry）→ tool_result 事件流里带 `needs_review:true` + 条目标识 →
  前端在对话流内弹出审批卡片：「Agent 想记住：[威胁] 官网挂马常伴随
  Webshell…【通过】【拒绝】【忽略】」
- 通过/拒绝调**现有** reviewApprove/reject API（零后端新增）；处理后
  卡片就地变为已通过/已拒状态（知识即刻生效或归档，不用再等批量审）
- 忽略 = 保持 pending，不强迫当场决定
- 回放兼容：tool_calls 持久化了 params/result → 回放同样渲染审批卡片；
  已处理的显示终态（需查条目当前 status——GET /api/memory/{type}/{title}
  已有）
- /review 页面降级为**积压兜底**（错过的忽略项），侧栏入口可以收小
  甚至隐藏；不做日常主路径
- 技术上 = F3 ask_user 的同款交互模式（对话内卡片+按钮组），两处
  共用组件

后端小改：memory_ensure/observe 的 result JSON 加 `needs_review` 与
条目标识字段（现只有 message 文本）。

### 依赖与顺序

F0（地基）→ F1（停止/编辑重发，依赖 F0 的 Run cancel）→ F3（ask_user）
→ F4（内联审核，与 F3 共用卡片组件，可与 F3 同期）→ F2 轻量
（截断重发）→ F2 完整（先验证轻量够不够）。
F4 不依赖 F0（现状连接模型下也能先做，刷新丢卡片属可接受过渡）。

## 前端系统性问题清单（2026-08-14 审查）

除上述交互缺口外，顺带审出的前端欠账：

- **H1 Prompt 透明化**：→ 已升级独立立项 [[plans/console/README.md|console]]
  （配置中心 + 可观测性）。conversation-ux 只保留 H2（意图判定属对话流）。
- **H2 意图判定 ✅**（ResultCard 头部触发依据）：Agent 走分析还是聊天是模型自主判断（灰区），
  界面无任何信号。修法：Agent 意图作为事件流的一部分——reasoning
  已流式可见，但「判定：真实需求 → 完整分析」的显式意图标记没有。
  方案：prompt 要求模型在 analysis_submit 前的总结里显式说明判定
  理由，或 ResultCard 头部显示触发依据（低成本起步）。
- **H3 Prompt 模板配置化**：→ 已并入 console plan 的 C3（外置/热加载/
  版本化），不在本 plan 重复。
- **G1 切换会话状态错乱**（F0 的前端侧）：messages 状态单一、不分
  session；切换会话时流式结果写错地方或丢失。修法：状态按 session
  分离（Map<sessionId, ChatMsg[]>）+ 订阅对应会话的 stream。
- **G2 输入草稿不保留 ✅**（localStorage 按会话）：切走再回来，输入框清空。修法：draft 按
  session 存 localStorage。
- **G3 侧栏无运行/完成提示**：哪个会话在跑、切走的会话何时跑完，
  无感知。修法：F0 的 running 状态 + 完成时列表项轻提示（未读点）。
- **G4 会话标题 ✅**（Run 完成后 LLM 生成）：长文本很丑。修法：后台 LLM 生成标题
  （Run 完成后一次性调用，正好是 TaskBackground 域第一个用户）。
- **G5 回放/继续对话切换生硬**：两个页面割裂（/analyze/:id 与
  /history/:id），「继续对话」跳页丢上下文。远期合并为同一视图。
- **G6 无会话搜索 ✅**（侧栏标题过滤）：会话多了找不到。标题+消息关键词过滤即可起步。
- **G7 error 状态不持久**：错误显示在消息流里，刷新即失。修法：错误
  也走事件缓冲（F0 后自然解决）。
- **G8 滚动位置 ✅**（记忆位恢复/流式跟底）：重载历史后强制滚底，长会话找回位置麻烦。

## 明确不做（倾向）

- 服务端 hold 连接等用户输入（F3 备选已否）
- 消息树完整版不急做，先验证截断重发 + 分叉会话是否覆盖需求

## 下一步

**已完成（2026-08-14）**：F0 服务端 Run 任务模型 + F1 停止/编辑重发
（goal mssz523g，commit 78264ea 主体 + 九轮审计修复链
92e73ca/d80e115/8aa8f65/686e208/8275da4/2def45a/ee56367/9ce0977/
87050fa/cdd558f，最终 cdd558f；并发策略裁决=单 Run 拒绝并发 409；
e2e 26/0、smoke-f0 11/11）。

**候选批次（已全部落地，2026-08-15）**：F3 ask_user ✅ + F4 内联审核 ✅（共用对话内卡片组件，同批落地）
→ F2 轻量截断重发已在 F1 落地（验证够用否再决定消息树/分叉）。
