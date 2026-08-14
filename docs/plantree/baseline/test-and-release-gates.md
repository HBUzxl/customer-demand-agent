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
  - 阻塞门禁：后端 `go build` / `go vet` / `go test -race` / `govulncheck`；前端 `tsc --noEmit` / `npm run build`
  - 非阻塞 informational：`golangci-lint`（24 遗留 issue）、`npm audit`（4 漏洞，见 AGENTS.md「已知基线」）
- **发布**：本地 `./scripts/run.sh` 起服务；无容器化/CD

## lint / format（已配，基线待清）

- Go：`.golangci.yml`（v2），`golangci-lint run ./...`
- 前端：`npm run lint`（eslint flat config）/ `npm run format:check`（prettier）
- 当前基线未清，故 lint 未作为 CI 阻塞门禁；清完升级

## 后续

- 补无测试包的单测（尤其 `history` 持久化、`model` 路由/回退、`llm` 协议解析）
- 清 golangci-lint / eslint / prettier 基线后，将 lint 升级为 CI 阻塞门禁
- 容器化（Docker 镜像）→ 内网部署 → 钉钉 webhook 对接
