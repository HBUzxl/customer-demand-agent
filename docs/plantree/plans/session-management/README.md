# 会话管理增强

> 会话删除的交互样式与批量管理能力。状态：**Ready**（用户提出 2026-08-15，
> 一期收尾后首项；范围明确，可直接开工）。

## 背景

一期收尾后用户在实际使用中提出：想「从头来」清理全部会话时，只能一个一个
点删除；且删除确认用的是浏览器原生 `window.confirm`——与应用整体 UI 割裂。

## 需求（用户原话 2026-08-15）

1. **弹窗样式**：点击删除后弹出的是浏览器原生弹窗，不是应用自己的弹窗——
   需要改成应用内模态弹窗。
2. **批量删除**：点击「全部」（History 页）时应支持多选，支持批量管理删除
   （「比如我想从头来，需要批量删除，但现在做不到」）。

## 现状盘点（2026-08-15）

| 位置 | 现状 | 差距 |
|---|---|---|
| 侧栏 AppLayout.tsx:77 | `window.confirm("删除这个对话？")` | 原生弹窗 |
| History.tsx:27 | `confirm("删除该会话及其全部记录？")` | 原生弹窗 |
| History.tsx 列表 | 单行删除按钮，无复选框/多选状态 | 无批量选择 |
| 后端 DELETE /api/sessions/{id} | 单会话删除（会话+消息+工具调用+checkpoint 级联） | 批量=前端循环调用即可，无需后端改造（一期结论：不建批量端点，逐个 DELETE 复用现有级联+鉴权语义） |
| History.tsx 错误处理 | `alert(e.message)` | 原生 alert 同样要换（同弹窗改造一并处理） |

## 方案

### F1. 应用内确认弹窗（ConfirmDialog 组件）

- 新建共享组件 `components/ConfirmDialog.tsx`：模态遮罩 + 标题/正文/确认
  （危险色）/取消；`onConfirm`/`onCancel` 回调；点遮罩/Esc = 取消。
- 替换两处 `window.confirm`（AppLayout 侧栏删除、History 删除）。
- 替换 History 的 `alert` 错误提示（改为页面内 toast/错误条——最小实现：
  页面顶部错误横幅，避免再引入 alert）。
- 复用现有 CSS 变量（--danger/--red、--surface 等），风格与 /review 审批
  卡片一致（项目无 UI 库，纯手写模态）。

### F2. History 批量选择 + 批量删除

- 列表行加复选框（表头全选/全不选）；选中集合 `Set<string>` 状态。
- 选中时顶栏出现批量操作条：「已选 N 项 | 批量删除 | 取消选择」。
- 批量删除走 ConfirmDialog 确认（「删除 N 个会话？不可恢复」）→ 前端
  `Promise.allSettled` 逐个 DELETE → 汇总成败（成功 X / 失败 Y+首个错误
  信息）→ 刷新列表。
- 侧栏「最近对话」不做多选（窄面板放不下，批量场景在 History 全量页）。
- 「全部」入口（conv-all → /history）保持现状，批量能力落在 History 页内。

### F2b. 会话搜索升级：标题 → 标题+内容（用户提出 2026-08-15 第三批）

现状：侧栏搜索纯前端过滤 `title.includes || session_id.includes`
（AppLayout.tsx:139）——只搜标题。后端 `SearchMessages`（内容 LIKE 检索）
已存在但只服务于 Agent 的 history_search 工具，UI 未用。

用户裁决：「我觉得可以加一下搜索内容，你觉得呢？你觉得没必要就不用加；
觉得可以加那就加。」

**判断：值得加。** 会话标题是首句话截断/LLM 概括，经常不含用户要找的
关键词（聊「刷号防护」的会话标题可能是「医院客户需求分析」）；用户找
历史会话的真实心智是「我上次聊 XX 的那个对话」——内容搜索命中率高得多。

方案：

- 新增后端 `GET /api/sessions/search?q=&limit=`：复用 SearchMessages 的
  LIKE 匹配，返回 session 维度聚合（session_id/title/customer/命中片段
  snippet/更新时间），去重按最近命中排序。
- 侧栏/History 搜索防抖 300ms 后走该接口（标题命中优先排前，内容命中
  次之）；结果行显示命中片段（snippet 高亮，截 80 字）而非只标题。
- 无匹配回退纯标题过滤（现有行为兜底）。

### F2c. 会话列表显示绑定客户 + 长名截断（用户提出 2026-08-15 第三批）

用户确认的语义：

- 绑定不是必须的——「有些情况下不需要绑定客户」「没有绑定客户的会话
  应该直接显示，不用管」（无绑定的正常列出不占特殊标记）。
- 绑定机制复用 F3（Agent 行为：ask_user 问 + session_bind_customer 工具）
  ——「绑定之后对 AI 来说上下文其实没有什么变化，或者说只是插入一些
  相关信息」（即现状 L2 客户画像注入语义，无其它副作用）。
- **列表显示**：会话列表（侧栏 conv-item + History 表）要有一个小位置
  显示该会话绑定的客户是谁。
  - 侧栏：conv-title 下或旁边加客户小徽章（如「某跨境电商」截 6 字，
    title 属性给全名）；无绑定不显示（不留空位）。
  - History 表：现有 customer 列保留，长名 CSS 截断（max-width+ellipsis，
    hover title 全名）——现状长名把表格撑爆。
- conv-item 现有 CSS 已有 ellipsis 基础（.conv-title），徽章沿用同规则。

### F3. 客户绑定 Agent 自主化（用户裁决 2026-08-15 第二批）

现状：对话页「+ 关联客户」chip 由用户手动填写客户名 → POST customer 字段
→ sessions.customer 列 → Agent 每轮读注入 L2 身份行。**Agent 从不主动问。**

用户裁决（原话）：「这个关联客户应该是 Agent 自己的行为。就是 Agent，它会
看目前都有什么客户，然后 ask user question……当前你问的是谁家客户对吧？
这应该是 Agent 的行为，而不是用户主动，也不是用户绑定的。」

- **手动入口彻底删掉**（用户明确：「手动填的入口彻底删掉」）——前端
  customerChip/editingCustomer/customer state 全链移除；POST customer
  字段保留但只有 Agent 工具会写。
- 新增 agent 业务工具 `session_bind_customer`（类似 ask_user 的轮次内
  工具）：Agent 发现会话未关联客户且对话涉及真实客户需求时 → 先
  memory_list(customer) 拿已有客户 → ask_user「这是谁家客户？」选项=
  已有客户 + 「新客户」→ 用户答后调 bind 工具写 sessions.customer。
- prompt（systemAutonomy）加指引：对话开始涉及具体客户而未绑定 →
  主动问一次（每会话只问一次，不重复）。
- 前端绑定后自动展示「客户：XX」（只读 chip，不可点编辑）。
- 验证：集成测试——mock LLM 流 ask_user(选项来自 memory_list) →
  bind 工具调用 → sessions.customer 落库 → 下一轮 prompt L2 身份行出现；
  前端无手动输入入口。

### F4. 消息复制按钮（用户提出 2026-08-15）

- 用户消息气泡 + Agent 回复气泡 hover 显示复制按钮（点击复制完整文本到
  剪贴板，短暂反馈「已复制」）。
- Agent 回复制的是最终 markdown 文本（m.text），不含 reasoning/工具轨迹。
- Replay 页同步补（消息行展开详情处）。
- 实现：navigator.clipboard.writeText + 现有消息气泡组件加 hover action。

### 非目标

- 不做后端批量删除端点（循环单删够用；若将来会话量大再议）。
- 不做按日期范围/客户维度的批量筛选删除（用户提的是全选场景）。
- 侧栏不做批量选择。

## 验证

- 前端 tsc/lint/format/build 绿。
- 手动冒烟：单删（侧栏+History 两处）弹应用内弹窗；全选→批量删除→列表清空
  →侧栏同步清空；部分失败场景（构造：删除时断网）显示汇总错误。
- e2e 不动（会话删除已有断言；弹窗是纯前端行为，e2e 无头环境测不了 modal）。
