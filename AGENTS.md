# Repository Guidelines

## Project Structure & Module Organization
仓库分为 Go 后端和 React 前端两部分。后端入口在 `cmd/datamap`，核心代码位于 `internal/`，其中 `api` 负责 Gin HTTP 处理器，`service` 负责业务逻辑，`store` 负责持久化，`scanner` 负责元数据采集，`model`/`config`/`crypto` 提供共享模型与基础能力。前端位于 `web/`，页面在 `web/src/pages`，复用组件在 `web/src/components`，数据访问在 `web/src/services`，自定义 hooks 在 `web/src/hooks`。数据库迁移在 `migrations/`，部署和设计文档在 `docs/` 与根目录说明文件中。

## Build, Test, and Development Commands
优先使用 `Makefile` 统一入口：

- `make dev`：并行启动后端和前端开发环境。
- `make build`：构建 `bin/datamap` 并打包前端。
- `make test`：运行 Go 与前端测试。
- `make lint`：运行 `golangci-lint` 和 `web` 下的 ESLint。
- `make ci`：本地执行与 CI 接近的完整检查。

单独开发时常用：

- `go run ./cmd/datamap`
- `go test -v ./...`
- `cd web && pnpm dev`
- `cd web && pnpm test`

## Coding Style & Naming Conventions
Go 代码保持 `gofmt`/`goimports` 默认风格，使用制表符缩进，导出标识符使用 PascalCase，包名保持简短小写。前端使用 TypeScript + React 函数组件，默认两空格缩进，组件文件采用 PascalCase，如 `SourceDetailPage.tsx`，hooks 使用 `useXxx.ts`，服务模块使用 `xxxService.ts`。提交前运行 `make fmt`、`make lint`。

## Testing Guidelines
后端测试文件使用 `*_test.go`，推荐与被测代码同目录放置，当前使用 Go `testing`，部分接口测试结合 `testify` 与 `httptest`。前端测试位于 `web/src/**/__tests__` 或与模块同层，命名使用 `*.test.ts` / `*.test.tsx`，通过 Vitest 运行。新增 API、服务逻辑或 UI 行为时应补充对应测试；涉及回归修复时，优先先写失败测试。

## Commit & Pull Request Guidelines
现有提交历史采用 Conventional Commits，常见格式如 `feat(api): ...`、`test(frontend): ...`、`chore(deps): ...`。请保持 `type(scope): summary` 风格，scope 建议使用 `api`、`backend`、`frontend`、`deps` 等明确范围。PR 需要说明变更目的、影响模块、测试结果；涉及 UI 的改动请附截图，涉及配置或迁移的改动请注明 `.env`、`configs/config.yaml` 或 `migrations/` 的同步要求。

## Security & Configuration Tips
不要提交真实凭据；本地配置从 `.env.example` 复制生成 `.env`。加密、认证和数据库相关修改需同时检查 `ENV_VARS.md`、`docker-compose.yml` 与 `configs/config.yaml`，确保开发环境和容器环境配置一致。
