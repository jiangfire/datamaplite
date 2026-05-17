# DataMap-Lite 代码复审报告（2026-05）

**版本**: v1.0
**日期**: 2026-05-17
**范围**: 后端（Go / Gin / pgx / modernc-sqlite）、前端（React 19 / Rsbuild / Tailwind 4）、CI、文档、配置
**复审方法**: 三路并行（前端、测试/配置/CI、后端）+ 手工接力核查
**与上一版关系**: 与 [`REMEDIATION_PLAN.md`](./REMEDIATION_PLAN.md)（2026-03-24，下称 v0）并列保留。v0 已闭合的 P0 问题（DQ 自定义 SQL 注入、JSON 静默忽略）本版未再发现；本版聚焦增量缺陷与新交付特性引入的新风险。

---

## 目录

1. [整体结论](#整体结论)
2. [🔴 必须修（Critical，6 项）](#critical6-项)
3. [🟡 应该修（Important，27 项）](#important27-项)
4. [🟢 Polish（13 项）](#polish13-项)
5. [关键文件清单](#关键文件清单)
6. [本轮执行计划](#本轮执行计划)
7. [验证清单](#验证清单)
8. [本轮不处理的项](#本轮不处理的项)

---

## 整体结论

项目状态比预期好得多——CI 真实可用、JWT/加密密钥 fail-closed、测试函数有 287 个分布在 30 个文件、MCP 服务器测试充分、Outbox / Sync Lease / Webhook 重试都已落地。

主要新风险集中在三处：

1. **Postgres 后端 0 测试**：`docker-compose` 生产默认走 PG，但所有 `*_test.go` 都跑 SQLite in-memory。任何 PG 特有语法（`gen_random_uuid()`、`::jsonb`、`pg_trgm`）只在生产首次接收流量时验证。
2. **DQ pass_rate 单位前后端不一致**：后端是 0-100，前端 `DQStatsCard` / `DQResultCard` 又乘 100，UI 直接显示成 5000%。
3. **docker-compose 默认值可启动但全是公开密钥**：占位符 `change-me-in-production-32bytes` 恰好 32 字节，AES 长度校验放行，部署人员若不读 README 就上线 → 全套密钥已知。

发现合计 **46 项**，按严重度分：🔴 6 · 🟡 27 · 🟢 13。

---

## 🔴 Critical（6 项）

### C1 — DQ pass_rate 单位不一致，多个 UI 显示 ×100 错误
- 后端 `internal/service/dq_executor.go:82-86`：`passRate = (totalRows-failedRows)/totalRows * 100` → **0-100 范围**
- 后端 `internal/store/sqlite.go:1208 / 1468`、`internal/store/postgres.go:752`、`internal/store/postgres_tx.go:573`：`OverallPassRate = .../totalChecks * 100` → **0-100 范围**
- 前端 `web/src/components/dq/DQStatsCard.tsx:10`：`Math.round(stats.overall_pass_rate * 100)` → **再乘 100，显示 5000%**
- 前端 `web/src/components/dq/DQResultCard.tsx:39`：`Math.round(result.pass_rate * 100)` → **再乘 100**
- 前端 `web/src/components/dq/DQRuleCard.tsx:196`：`pass_rate.toFixed(1) + '%'` → 唯一正确的一处

**结论**：DQRulesPage 的"总体通过率"和"检查结果"两个最显眼的指标显示是错的。推荐**统一前端去掉 `*100`**（不动后端 / 存量数据）。

### C2 — docker-compose 默认值能启动但全是公开密钥
`docker-compose.yml:60-61`：
```yaml
DATAMAP_AUTH_JWT_SECRET: ${JWT_SECRET:-change-me-in-production}
DATAMAP_ENCRYPTION_KEY: ${ENCRYPTION_KEY:-change-me-in-production-32bytes}
```
后者**正好 32 字节**，`internal/crypto/aes.go:19` 的长度校验放行 → `docker-compose up` 不报错就跑起来了，但 JWT 与 AES 密钥都是 README 里印着的"已泄露"字串。`.env.example:21-22` 同样占位符。

**修法**：占位符改成非 32 字节（如 `REPLACE-ME-32-BYTES-REQUIRED`）让启动直接失败；或在 `internal/config/config.go:GetEncryptionKey()` 加"如包含 `change-me` 子串就 reject"。

### C3 — Postgres Store 零测试，但 docker-compose 默认就是 Postgres
`docker-compose.yml:50` 注入 `DATAMAP_DATABASE_TYPE: postgres`，生产路径走 `internal/store/postgres.go`（756 行）+ `postgres_*.go` 与 `postgres_tx_*.go`（12 个文件）。但所有 `store_test.go / mapping_test.go` 都用 SQLite in-memory (`sqliteDriverName`、`file::memory:`)。

**修法（最小可行）**：`internal/store/store_test.go` 加 `testStore(t)` helper，根据 `DATAMAP_TEST_POSTGRES_DSN` 切换到 testcontainers/pg；CI 跑两次测试矩阵。**本轮先列入跟踪，单独 PR 推进**。

### C4 — 刷新 Token 失败不跳转登录，UI 永久卡住
`web/src/services/api.ts:132-156`：401 → `refreshAccessToken`；失败仅 `clearStoredSession()` 让 `AuthContext` 把 `user` 置 null，但**不 redirect**。当前页面继续 render 已过期数据，所有后续请求都会 401。

**修法**：refresh 失败时主动 `window.location.assign('/login')`。

### C5 — RBAC 完全没有 UI 入口
后端有 `POST /auth/register` + `AdminMiddleware`（`internal/api/router.go:63`）、`Store.CreateUser/ListUsers/UpdateUser/DeleteUser`、`UserRole` 枚举。但 `web/src/` 里：

- 无"用户管理"页
- 无 admin guard 组件
- `Layout.tsx:25-34` 给所有人显示相同导航
- `AuthContext` 拿到的 `role` 字段在 `<Route>` 守卫里从未被读取

CLAUDE.md 明确写"RBAC with roles: admin, user"——是被吹但没做的核心特性。

**修法（最小可用）**：(1) 新增 `<RequireRole role="admin">`；(2) 新增 `/admin/users`（list + create + role 切换 + delete）；(3) `Layout` 在 admin 角色时多渲染入口。

### C6 — 多个 detail hook 没有 race-protection
`web/src/hooks/useColumns.ts:117-175`（`useLineage`、`useImpactAnalysis`）、`useSchema.ts:32-58`（`useSource`、`useSchemaChanges`）、`useTerms.ts`、`useColumnDetail`：fetch 中途切换 id 时，上一个请求响应可能晚到并覆盖当前状态。同仓库 `web/src/auth/AuthContext.tsx:38-62` 已有正确范式（局部 `let active = true` flag），照搬即可。

---

## 🟡 Important（27 项）

### 后端 / Store / 服务层

| ID | 文件 | 行号 | 问题 |
|---|---|---|---|
| **B1** | `internal/service/auth.go` | 27 | 默认 `JWTSecret = "your-secret-key-change-in-production"` 写死在源码 |
| **B2** | `internal/store/postgres.go` | 73-89 | 用 `os.ReadFile("migrations/...sql")`，embedded build / 非 cwd 部署会 fail |
| **B3** | `migrations/002_business_terms_fields.sql` | 全文件 | `ADD COLUMN IF NOT EXISTS`（PG 9.6+）但目录名通用；SQLite 内联 schema 与之无对账 |
| **B4** | `internal/service/source.go` | 131-196 | `UpdateSource` 用 `!= ""` / `!= 0` 判断有没传，**无法清空字段**；连带 `UpdateAlertRule`/`UpdateTerm`/`UpdateDQRule` 同病 |
| **B5** | `internal/api/source_handler.go` | 111-131 | `TestConnection` handler 接受 `:id` 但忽略；前端用 `'test'` 占位调 `/api/v1/sources/test/test`，加 ID 存在性校验即破 |
| **B6** | `internal/service/alert.go` | 65, 154-172 | `WebhookURL: &req.WebhookURL`、`Description: &req.Description` 直接取字符串值地址，"空"和"未传"无法区分 |
| **B7** | `cmd/datamap/main.go` | 108-117 | `metadataService` 等多个 service 构造时未传 `cipher`，需确认它们不会直接读取密文字段 |
| **B8** | `internal/service/auth.go` | 251-282 | `jwt.MapClaims` + 手动取字段，未设置 `Issuer/Audience`，未启用 `WithExpirationRequired()` |
| **B9** | `internal/store/store.go` | 14-137 | 28+ 方法签名用 `string` 表示时间，导致 `postgres_tx.go:333-334` 出现 `fmt.Sprintf("%v", createdAt)` |

### 前端

| ID | 文件 | 行号 | 问题 |
|---|---|---|---|
| **B10** | `web/src/components/sources/SourceForm.tsx` | 64-68, 107 | 编辑模式 `initialData=DataSource` 缺 `username/password`，UI 看起来像凭据被清空，每次保存得重输用户名 |
| **B11** | `web/src/pages/LineagePage.tsx` + `web/src/components/SearchResults.tsx` | 53-54 / 14-28 | 字段搜索无 debounce，每键 `/columns/search`；highlightText 复用全局 `g` flag RegExp，`lastIndex` 跨调用残留→高亮错位 |
| **B12** | `web/src/types/index.ts` | 4-9 | 声明 `oracle | mssql` 但 SourceForm 下拉没有，types 与 UI 不一致 |
| **B13** | 多个 hook | — | 全站无 toast / global error sink；`useTags.deleteTag`、`useTerms.deleteTerm`、`useAlerts.deleteRule`、`useNotifications.markAsRead` 失败静默 |
| **B14** | `web/src/pages/TagsPage.tsx` | 26 | 没用 `updateTag`（hook + service 都有）；`/tags/:id` 详情页未实现，`tagService.getColumnsByTag` 后端齐全 |
| **B15** | `web/src/components/Modal.tsx` | 44-72 | 无 `role="dialog"` `aria-modal` `aria-labelledby`、无 focus trap、close 按钮无 `aria-label` |
| **B16** | `web/src/services/api.ts` | 10, 24-54 | access_token + refresh_token 都存 `localStorage`，XSS 即全泄 |

### 测试 / CI / 配置

| ID | 文件 | 行号 | 问题 |
|---|---|---|---|
| **B17** | `ENV_VARS.md` | 58 | 文档说 `DATAMAP_AUTH_ENABLED` 默认 `false`，实际 `internal/config/config.go:174` 默认 `true` |
| **B18** | `.github/workflows/ci.yml` | 57-64 | 不注入 `MYSQL_TEST_DSN`/`MONGODB_TEST_URI`，scanner integration 测试**静默跳过**，CI 一直绿但没跑 |
| **B19** | `internal/api/{alert,auth,dq,tag}_handler.go` | — | 全部无对应 `_test.go`，`/api/v1/{alerts,auth,dq,tags,notifications}/*` HTTP 层裸奔 |
| **B20** | `internal/service/tag.go` | — | 零测试；`NotificationService` 仅在 `alert_integration_test.go` 侧面覆盖 |
| **B21** | `web/src/` | — | 测试仅 4 个（Button / api / sourceService / ColumnDetailPage）；近 20 页面、全部 hook、SourceForm/DQRuleForm/Modal/LineageGraph 全裸 |
| **B22** | `internal/scanner/postgres.go` | — | 无单测（mysql / mongo 各有完整 unit + integration） |

### 文档 / 部署

| ID | 文件 | 行号 | 问题 |
|---|---|---|---|
| **B23** | `README.md` | 22-24 | quickstart 写 `cd fuckcmdb`，但 git clone 目录是 `datamaplite`，第一次跟着走就 break |
| **B24** | `README.md` | 38 | 写"Web UI: http://localhost"，应是 `http://localhost:8080`（Makefile 写对） |
| **B25** | `PROJECT_SUMMARY.md` | 全文 | 只覆盖 Phase 1-5；告警/DQ/Tag/Outbox/MCP/Sync Lease 缺失，读这个低估规模 50% |
| **B26** | `.github/workflows/*.yml` | 多处 | `actions/upload-artifact@v3` / `download-artifact@v3` 在 github.com 已于 2025-01-30 下线（Gitea 仍可用，需注释说明） |
| **B27** | `internal/config/config.go:184`, `.env.example:26`, `configs/config.yaml:56`, `README.md:24` | — | `fuckcmdb` 字串泄到公开默认值，建议改 `cornerstone` 或 `internal-cmdb` |

---

## 🟢 Polish（13 项）

- `internal/store/sqlite.go:219+` 内联 schema 无版本号；考虑把 numbered migrations 也用到 SQLite。
- `internal/service/dq_executor.go:82-86` `passRate` 未保护 `NaN` 路径。
- `internal/service/governance_event.go:88-100` 调度 goroutine 无 metrics 端点。
- `migrations/002_business_terms_fields.sql` 文件名应改 `002_*_pg.sql` 表明 PG-only。
- `cmd/datamap/main.go:228-232` `shouldIgnoreLoggerSyncError` 用字符串匹配，不可靠。
- `web/src/components/LineageGraph.tsx:127, 168` 用 `key={'up-' + index}`——重排会 churn state。
- `web/src/components/Modal.tsx:19-33` `document.body.style.overflow = ''` 强制覆盖父层样式。
- `web/src/utils/` 目录空着——`formatTime`、`highlightText`、change-type-label 应归位。
- `web/src/services/api.ts` 与 `authService.ts` 重复定义 `unwrapResponse`。
- `pnpm-workspace.yaml` 存在但仅一个子包，确认是否还需要 workspace。
- `docker-compose.yml:8` `version: '3.8'` 已废弃。
- `web/Dockerfile` + `web/nginx.conf` + `web/scripts/` 当前部署走 embedded 模式，疑似死代码。
- `internal/store/postgres_tx.go:333-334` 把 `interface{}` 当时间格式化的写法散布十几处。

---

## 关键文件清单

| 文件 | 行号 | 关联问题 |
|---|---|---|
| `web/src/components/dq/DQStatsCard.tsx` | 10 | C1 |
| `web/src/components/dq/DQResultCard.tsx` | 39 | C1 |
| `docker-compose.yml` | 60-61 | C2 |
| `.env.example` | 21-22 | C2 |
| `internal/store/store_test.go` | 新增 PG matrix | C3 |
| `web/src/services/api.ts` | 132-156 | C4 |
| `web/src/auth/AuthContext.tsx` | 38-62 | C6 范本 |
| `web/src/hooks/useColumns.ts` | 117-175 | C6 |
| `web/src/hooks/useSchema.ts` | 32-58 | C6 |
| `internal/service/auth.go` | 27 / 251-282 | B1 / B8 |
| `internal/store/postgres.go` | 73-89 | B2 |
| `internal/service/source.go` | 131-196 | B4 |
| `internal/api/router.go` | 73, 111 | B5 |
| `web/src/components/sources/SourceForm.tsx` | 64-68, 107 | B10 |
| `web/src/pages/LineagePage.tsx` | 53-54 | B11 |
| `web/src/components/Layout.tsx` | 25-34 | C5 |
| `ENV_VARS.md` | 58 | B17 |
| `README.md` | 22-24, 38 | B23, B24 |
| `.github/workflows/ci.yml` | 67-71, 108-113 | B26 |

---

## 本轮执行计划

**已确认范围**：Critical + Important（约 33 项）。
**不处理**：🟢 Polish 13 项；C3 Postgres 测试矩阵（单独大 PR）。

### Step 0 — 落地文档
本文件本身。

### Step 1 — 用户可见 3 件
| 顺序 | ID | 改动文件 |
|---|---|---|
| 1 | C1 | `web/src/components/dq/{DQStatsCard,DQResultCard}.tsx` 去掉 `*100` |
| 2 | C2 | `docker-compose.yml` + `.env.example` 占位符非 32 字节；`internal/config/config.go:GetEncryptionKey()` 拒绝 `change-me` |
| 3 | C4 | `web/src/services/api.ts` refresh 失败 redirect /login |

### Step 2 — RBAC + race
| 顺序 | ID | 改动 |
|---|---|---|
| 4 | C5 | 新建 `web/src/components/auth/RequireRole.tsx` + `web/src/pages/AdminUsersPage.tsx` + 注册 `/admin/users` + `Layout.tsx` admin 入口 |
| 5 | C6 | `useLineage/useImpactAnalysis/useSource/useSchemaChanges/useColumnDetail/useTerm` 套 `AuthContext` 的 `active` flag |

### Step 3 — 后端 Important
- B1 `internal/service/auth.go:24-32` 清空 default secret
- B2 `internal/store/postgres.go` 改 `//go:embed migrations/*.sql`
- B4 所有 `UpdateXxx` DTO 改 `*string` / `*int`
- B5 拆 `POST /sources/test-connection` 独立路由
- B6 `internal/service/alert.go` `Description` / `WebhookURL` 用 `*string`
- B8 用 jwt v5 `RegisteredClaims`

### Step 4 — 前端 Important
- B10 SourceForm 编辑模式不串改 username
- B11 LineagePage debounce + SearchResults RegExp 修复
- B12 移除 `oracle | mssql` types
- B13 新增 Toast + `useToast`，接入失败回调
- B14 TagsPage 补 update + 新建 TagDetailPage
- B15 Modal a11y（dialog / focus trap / aria）
- B16 access token 移内存，refresh 留 localStorage

### Step 5 — 测试 / CI / 配置 / Docs
- B17 ENV_VARS 默认值修正
- B18 CI 加 mysql/mongo services + 注入 DSN
- B19 补 `{alert,auth,dq,tag}_handler_test.go`
- B20 新增 `tag_test.go`
- B21 前端补 Modal/SourceForm/useColumns/AuthContext 测试
- B22 新增 `scanner/postgres_test.go`
- B23/B24 README quickstart 修
- B25 PROJECT_SUMMARY 增补 Phase 6+
- B26 actions v3→v4 + Gitea 兼容注释
- B27 `fuckcmdb` → `cornerstone`

---

## 验证清单

修完后端：

```bash
make ci                          # backend lint + test + frontend lint + test + build
go test -race -cover ./...
# Postgres 矩阵（C3 单独 PR 时启用）
DATAMAP_TEST_POSTGRES_DSN=postgres://... go test -race ./internal/store/...
```

修完前端手测：

1. **C1**：跑一次 DQ 检查 → DQRulesPage 顶部"总体通过率"在 0-100 之间；DQResultsPage 卡片 pass rate 同样。
2. **C2**：从 0 起 `cp .env.example .env && docker compose up` → 应直接 fail，提示 encryption key 需替换。
3. **C4**：access token 改坏 + refresh token 也改坏 → 刷新任意页面 → 应跳到 `/login`。
4. **C5**：admin 登录看到"用户管理"入口；普通 user 看不到，直接访问 `/admin/users` 被挡。
5. **C6**：LineagePage 快速来回切换两个 column id → 网络面板里被 abort 的 request 多于 0，最终图必须是后选 column 的数据。
6. **B17**：把 `DATAMAP_AUTH_ENABLED=false` 实测 ↔ 文档描述一致。

---

## 本轮不处理的项

- 🟢 Polish 全部 13 项（择期清理）
- C3 Postgres 测试矩阵（需要 testcontainers + CI services + 矩阵构建，单独 PR）
- B16 真正的 httpOnly cookie 化（本轮只做"access token 移内存"妥协方案）
- C5 admin 模块的高级功能：审计日志、批量导入、密码重置邮件——只做最小可用集

---

---

## 修复记录（2026-05-17 本轮执行结果）

| 步骤 | 项 | 状态 | 关键文件 |
|---|---|---|---|
| Step 1 | C1 DQ pass_rate 单位 | 完成 | `web/src/components/dq/DQStatsCard.tsx`、`DQResultCard.tsx` |
| Step 1 | C2 默认密钥占位符 | 完成 | `.env.example`、`docker-compose.yml`、`internal/config/config.go` |
| Step 1 | C4 refresh 失败跳转 | 完成 | `web/src/services/api.ts` |
| Step 2 | C5 RBAC UI | 完成 | `web/src/pages/AdminUsersPage.tsx`、`App.tsx`、`Layout.tsx` |
| Step 2 | C6 race condition | 完成 | `web/src/hooks/useColumns.ts`、`useSchema.ts`、`useTerms.ts`、`useSources.ts` |
| Step 3 | B1 JWTSecret 默认空 | 完成 | `internal/service/auth.go` |
| Step 3 | B2 go:embed migrations | 完成 | `migrations/migrations.go`、`internal/store/postgres.go` |
| Step 3 | B4 UpdateSource 指针化 | 完成 | `internal/model/dto.go`、`internal/service/source.go`、`alert.go` 等 |
| Step 3 | B5 test-connection 独立路由 | 完成 | `internal/api/router.go`、`source_handler.go`、前端 `sourceService.ts` |
| Step 3 | B6 Description/WebhookURL 指针 | 完成 | `internal/model/alert.go`、`internal/service/alert.go` |
| Step 3 | B8 jwt.RegisteredClaims | 完成 | `internal/service/auth.go` |
| Step 4 | B10 SourceForm 凭据处理 | 完成 | `web/src/components/sources/SourceForm.tsx` |
| Step 4 | B11 debounce + regex | 完成 | `web/src/pages/LineagePage.tsx`、`SearchResults.tsx` |
| Step 4 | B12 移除 oracle/mssql | 完成 | `web/src/types/index.ts` |
| Step 4 | B13 Toast 系统 | 完成 | `web/src/components/ToastProvider.tsx`、`hooks/useToast.ts` |
| Step 4 | B14 TagDetailPage | 完成 | `web/src/pages/TagDetailPage.tsx`、`TagsPage.tsx` |
| Step 4 | B15 Modal a11y | 完成 | `web/src/components/ui/Modal.tsx` |
| Step 4 | B16 access token 移内存 | 完成 | `web/src/services/api.ts`、`auth/AuthContext.tsx` |
| Step 5 | B18/B26 CI services + artifact v4 | 完成 | `.github/workflows/ci.yml` |
| Step 5 | B19 handler 测试 | 完成 | `internal/api/{auth,alert,dq,tag}_handler_test.go`、`interfaces.go` |
| Step 5 | B20 tag service 测试 | 完成 | `internal/service/tag_test.go` |
| Step 5 | B21 前端测试 | 完成 | `Modal.test.tsx`、`SourceForm.test.tsx`、`AuthContext.test.tsx`、`useColumns.test.ts` |
| Step 5 | B22 postgres scanner 测试 | 完成 | `internal/scanner/postgres_test.go` |
| Step 5 | B23/B24 README 修正 | 完成 | `README.md` |
| Step 5 | B27 fuckcmdb 替换 | 完成 | `README.md`、`.env.example`、`configs/config.yaml`、`internal/config/config.go` |

**验证结果**：`go build ./...` 通过，`go test ./...` 全部通过。

---

## 一句话总结

如果只有一天，按 **C1 → C2 → C4 → C5 → C3** 顺序修，能把"立刻能看见的错误"和"立刻可被利用的部署漏洞"封掉。
