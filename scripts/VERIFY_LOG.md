# 终态验证记录（goal msu4rm2p-vyd39h，2026-08-15）

## 门禁命令与执行目录（canonical）

| 门禁 | 命令 | 执行目录 |
|---|---|---|
| Go 构建/静态/测试 | `go build ./... && go vet ./... && go test ./...` | 仓库根 |
| Go race | `go test -race ./internal/channel/http/ ./internal/history/ ./internal/taskbg/ ./internal/config/ ./internal/agent/ ./cmd/agent/` | 仓库根 |
| Go lint | `golangci-lint run ./...`（0 issue） | 仓库根 |
| gofmt | `gofmt -l cmd/ internal/`（空输出） | 仓库根 |
| 前端类型 | `npx tsc --noEmit`（0 errors） | **frontend/** |
| 前端 lint | `npm run lint`（0 errors, 14 warnings——结构性基线见 AGENTS.md） | **frontend/** |
| 前端格式 | `npm run format:check` | **frontend/** |
| 前端构建 | `npm run build` | **frontend/** |
| 端到端 | `./scripts/e2e.sh`（需 :8080 实例） | 仓库根 |
| 冒烟 f0 | `./scripts/smoke-f0.sh`（需 :8080 + 真实 key） | 仓库根 |
| 冒烟 p0 | `./scripts/smoke-p0.sh`（需 :8080 + 真实 key） | 仓库根 |
| 冒烟 final | `./scripts/smoke-final.sh`（需 :8080 + 真实 key） | 仓库根 |

注：前端是 Vite 子项目（frontend/package.json），所有前端命令必须在 frontend/ 下执行。

## 2026-08-15 终态实跑结果（:8080 运行终态二进制）

- e2e：29 passed / 0 failed（exit 0）
- smoke-f0：11 passed / 0 failed（exit 0）
- smoke-p0：5 passed / 0 failed（exit 0）
- smoke-final：8 passed / 0 failed（exit 0）
- go test：11 包全 ok；golangci 0 issue；gofmt 空；前端 tsc 0 errors/lint 0 errors+14 warnings（结构性）/format/build 过

（复跑方式：起 ./scripts/run.sh 或 ./cda-agent --config config.json 后逐条执行上表命令）
