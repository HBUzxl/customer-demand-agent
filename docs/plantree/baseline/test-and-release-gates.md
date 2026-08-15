# 测试与发布门禁

> 2026-08-13 经 `/init` 校准：原「一期无测试」已过时，项目已过原型阶段。

## 当前（已落地）

- **后端测试**：`go test ./...`
  - 有测试的包：`internal/agent`（含 mock LLM 全链路集成测试）、`internal/channel/http`、`internal/memory/{longterm,shortterm,tools,assembler}`
  - 无测试的包（补测优先）：`config`、`domain`、`history`、`llm`、`model`、`review`、`api`
  - 集成测试用 `httptest` + mock LLM 网关，**无需真实 API key**
- **端到端冒烟**：`./scripts/e2e.sh`（健康/配置/记忆 CRUD/审核流/会话，需后端先起，无需 LLM）
- **前端**：`npm run build`（= `tsc -b && vite build`）作为类型 + 构建门禁；暂无前端单元测试
- **CI**：`.github/workflows/ci.yml`
  - 阻塞门禁：后端 `go build` / `go vet` / `go test -race` / `golangci-lint`（0 issue）/ `govulncheck`（0 漏洞，需 go1.26.6+ 工具链）；前端 `tsc --noEmit` / `npm run lint`（0 errors） / `npm run format:check` / `npm run build`
  - 非阻塞 informational：`npm audit`（4 漏洞基线，vite 8 breaking 升级待用户决策）
- **发布**：本地 `./scripts/run.sh` 起服务；Docker 镜像已备（Dockerfile + VOLUME /var/lib/cda）

## lint / format（基线已清，2026-08-13 起全阻塞）

- Go：`.golangci.yml`（v2），`golangci-lint run ./...` → **0 issue**
- 前端：`npm run lint`（eslint flat config）→ **0 errors, 14 warnings**（结构性基线：router lazy 导出 react-refresh + 标准 load 模式 exhaustive-deps——不阻塞）；`npm run format:check`（prettier）→ 全合规

## 后续

- ~~补无测试包的单测~~（11 包全覆盖：含 history 持久化/model 路由/llm 协议/config/taskbg/cmd）
- ~~清 lint 基线升级 CI 阻塞~~（已完成，见上）
- ~~容器化~~（Dockerfile 已备）→ 内网部署 → 钉钉 webhook 对接
