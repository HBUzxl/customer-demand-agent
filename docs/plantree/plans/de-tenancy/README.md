# 移除多租户（de-tenancy）

> 用户裁决 2026-08-14（深夜）：**砍掉多租户**——单租户部署，代码里的
> tenant 概念整体移除。状态：**Done**（同日落地，收尾 goal task 1）。

## 裁决与动机

- 用户原话：「这个地方一定要做出来……把多租户砍掉」——产品定位为
  单租户内部工具，多租户是过度设计。
- ADR-011（tenant_id 数据隔离、一期无鉴权）随之**废止**。

## 影响面盘点（2026-08-14 已核实）

tenant 渗透 9 个文件 124 行（非测试）+ 44 行测试：

| 位置 | 内容 |
|---|---|
| `internal/api/message.go` | OutboundMessage/InboundMessage 的 TenantID 字段（channel 协议层） |
| `internal/agent/agent.go` + `submit.go` | Message(tenantID,...)、SetCheckpointSink/Source、SetCustomerResolver、SetHistorySearcher、turnState.tenantID、history_search 工具 |
| `internal/history/store.go` | sessions.tenant_id 列、EnsureSession 归属校验、ListSessions/GetSession/ListCheckpoints/SearchMessages/TruncateAfter 的 tenant 参数与 JOIN |
| `internal/channel/http/server.go` | tenantFrom/X-Tenant-ID 解析、tenantKey context、withTenant 中间件 |
| `internal/channel/http/analyze.go` | messageCore/executeTurn/running/cancel/truncate/stream 的租户校验 |
| `internal/channel/http/config.go` | 会话列表租户过滤 |
| `internal/config/config.go` | default_tenant 配置 |
| `cmd/agent/main.go` | resolver/searcher/sink 回调接线 |
| 测试 | cross_tenant_test.go（整文件）、TestTenantIsolation、TestRunEndpointsTenantIsolation、TestSearchMessages 隔离断言等 44 行 |
| e2e/smoke | postt/posttcode/deleteAs 租户头辅助、跨租户 6 断言 |
| 前端 | 无（前端从不传 X-Tenant-ID，全是后端默认租户兜底）——零改动 |

## 方案

1. **协议层**（api/message.go）：删 TenantID 字段（channel 契约变更，
   ADR-012 钉钉未来也不需要）。
2. **Agent 层**：全部回调与 Message 签名去 tenantID 参数。
3. **存储层**：SQLite **保留 tenant_id 列**（老数据兼容 + 未来若复活
   不用迁移），但**代码不再读写**；EnsureSession 删归属校验（同库即同
   租户）；查询去 JOIN/参数。或者直接 DROP COLUMN（SQLite 支持
   ALTER TABLE DROP COLUMN）——**推荐保留列**，砍代码不砍数据。
4. **HTTP 层**：删 tenantFrom/withTenant/tenantKey；X-Tenant-ID 请求头
   **接受但忽略**（老客户端不炸）；default_tenant 配置项保留解析但无
   行为（或删——config.example 同步）。
5. **测试**：删跨租户测试文件与断言；其余测试去租户参数。
6. **e2e/smoke**：postt 辅助简化（头仍可发，服务端忽略）；删跨租户断言。
7. **文档**：README/AGENTS.md/plan-tree（ADR-011 废止记录）、本 plan
   状态转 Done。

## 明确不做

- 不搬数据不删列：tenant_id 列保留；老库（NOT NULL 无默认）Open 时表重建补 DEFAULT 'default'，历史值原样保留；代码 INSERT/SELECT 均不涉及该列。
- 不删 wiki/记忆的组织级共享概念（ADR-011 的另一半——产品/行业/客户
  记忆组织级共享——不受影响，本来就无租户维度）。

## 下一步

排期后开工（预计半天：签名链 9 文件 + 测试 + 文档）。

## 落地记录（2026-08-14）

- 协议层：InboundMessage 删 TenantID 字段
- Agent 层：Message/四大回调（Sink/Source/CustomerResolver/HistorySearcher）签名去 tenantID；turnState 去 tenant
- 存储层：EnsureSession/AppendCheckpoint/ListCheckpoints/ListSessions/GetSession/DeleteSession/SearchMessages/TruncateAfter 八方法去参；归属校验与 JOIN 删除；tenant_id 列保留（新库 DDL 带 DEFAULT，老库表重建补 DEFAULT，INSERT 不提及该列，SELECT 不查）
- HTTP 层：tenantKey/withTenant/tenantFrom 全删；路由直挂；请求日志去租户列；X-Tenant-ID 头自然忽略（无任何代码读它）
- 配置：default_tenant 字段删除（example 与运行 config 同步）
- 测试：cross_tenant_test.go 删除；TestTenantIsolation/TestRunEndpointsTenantIsolation 删除或改写为「无租户头可访问+任意头忽略」正向测试；TestSearchMessages/TestCheckpoint* 适配单租户
- e2e：第 6 节改 de-tenancy 断言（任意头建会话 202 + 跨头访问不再 403 + 清理），26/0
- grep 复核：非测试代码无功能性 tenant 残留（仅列定义/兼容注释/JSON 字段恒 default）
- AGENTS.md 单租户段 + README 租户行 + 本记录同步；ADR-011 废止入档
