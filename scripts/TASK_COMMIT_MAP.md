# task ↔ commit 映射（goal msu4rm2p-vyd39h 二期，13 task）

| task | 功能 commit | 追修 commit |
|---|---|---|
| ui-shared | 3bd2c2b（ConfirmDialog 五处替换+CopyBtn 四处） | cbac944（原生弹窗二次扫描）+ 5addb0b（Settings 残留 prompt/alert 清零） |
| session-batch | 583c233（History 复选框+批量删） | cbac944（hover 浮现+全选并入批量条） |
| session-search | 34608f1（SearchSessions+API+侧栏接入） | 5addb0b（History 页搜索入口补） |
| customer-bind | b4b9c44（bind 工具+chip 删） | cbac944（徽章挪时间旁）+ 5addb0b（过时文案删） |
| memory-batch | 4ef7de0（计数+正文+六类型批量） | — |
| memory-search-ia | 61b9475（搜索语义） | cbac944（入口/文档徽章删——用户裁决）+ 5addb0b（「全部」tab 过滤器） |
| wiki-hygiene | 79d2dc2（清理+existing_hint+指引） | 5addb0b（审批条重叠提示 overlap_titles） |
| sidebar-models-ux | b0c2293（收缩态+五字段+下拉） | 5addb0b（死代码清） |
| review-gating | dcc0641（门禁+审批条+ADR-005） | cbac944（/review 页面彻底删——用户裁决） |
| console-config | 1433aae（三态+热生效） | 5addb0b（回退链可编辑） |
| checkpoint-tree | 34b030c（链→树+软分叉+分支过滤） | fccf75d（smoke-f0 断言）+ 5addb0b（Conversation 分支切换+PrevID 父指针语义） |
| ui-copy-cleanup | 59c834c（三类清扫+终扫断言） | 5addb0b（lint unused 清理连带） |
| verify-close（第 10 项：终态验证，goal task 形态无独立 plan 文件） | 8cdc157（九功能 plan 转 Done）+ 852caa4（终态收口） | 本轮 commit |

审计修复轮：fccf75d / 5addb0b（七项）/ 本轮（lint warnings unused 清零+baseline 门禁文档同步+四脚本留证）

> 说明：goal 的「10 个 plan」= 9 个功能 plan + 终态验证（verify-close task）。验证证据 = 本对照表 + scripts/VERIFY_LOG.md + 四脚本可随时复跑。
