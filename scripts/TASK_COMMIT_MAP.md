# task ↔ commit 映射（goal mst4eup8-iu47um，16 task）

| task | commit | 说明 |
|---|---|---|
| de-tenancy | 901def9 | 四层去 tenant（协议/Agent/存储/HTTP/config/test/e2e） |
| p2-p3 | 4e4278b + fbd3426 | P2 Notes 全链删除 / P3 链滚动归档（+e55b276 LIMIT 修复） |
| p9 | 4a7046f | CDA_DATA_DIR 三级+XDG 默认+播种+Docker（+c360ec6 真迁移+seed_test） |
| p6 | 4ce2c23 | 时效三字段+覆盖归档（+6bd3b3b 历史链+cb28ba9 typedSubdir/parsePage 修复+API 层测试） |
| f3-ask-user | 4578530 | 工具+SSE+选项卡+测试 |
| f4-inline-review | 92b259d | needs_review+审批卡（+6bd3b3b 忽略+回放态；+794b307/1ecd5db smoke 硬化） |
| p11-pipeline | c877c35 | taskbg 包+固化管线（+6bd3b3b 串行/race；+3d8325e 真链路 e2e；+c360ec6 extractObserves 修复） |
| p5-lint | a64c50a | RunLint 三检查（+c360ec6 overlap；+a631019 missing-entry） |
| g4-title | aca7e08 | Run 完成回调+BuildTitlePrompt+真实冒烟 |
| c1-config-center | 9e8180c | 四只读 API+Settings 配置中心 tab |
| c2-observe | aa40ffe + 58690e6 | LogRing/SSE tail/LLM 审计/observe 页（+修复：具名返回值） |
| c3-prompt-external | cc7627e + a9defd1 | 外置模板+快照（+修复：外置正文真实生效+template_raw 渲染） |
| p4-checkpoint-viz | c55f86a | SessionDetail.Checkpoints+Replay 行 |
| g-frontend-misc | e18baeb | G2/G6/G8/H2 |
| smoke-final | a3f26ff | 脚本入库（+多轮硬化 794b307/1ecd5db/cf1a55a/ac870aa） |
| verify-close | 5816cd7 + 2aa2963 + 17f4b1a | plan-tree 终态+AGENTS/README |

审计修复 commit（goal 要求的「全套门禁绿」维持轮）：6bd3b3b / c360ec6 / cb28ba9+692c2cd / a9defd1+1ecd5db / ac870aa / cf1a55a / b0ccd17 / 17f4b1a
