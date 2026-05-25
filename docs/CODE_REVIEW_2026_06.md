# DataMap-Lite 代码复审报告（2026-06）

**版本**: v1.0  
**日期**: 2026-06-24  
**范围**: 后端（Go / Gin / pgx / modernc-sqlite）、前端（React 19 / Rsbuild / Tailwind 4）、CI、文档、配置  
**复审方法**: 基于前两轮 review（2026-03、2026-05）的增量检查 + 全量功能缺失分析  
**与上一版关系**: 与 [`CODE_REVIEW_2026_05.md`](./CODE_REVIEW_2026_05.md) 并列保留。本版聚焦两轮 review 之后的**剩余缺陷**与**功能覆盖缺口**。

---

## 目录

1. [整体结论](#整体结论)
2. [🔴 必须修（Critical，3 项）](#critical3-项)
3. [🟡 应该修（Important，多项）](#important)
4. [🟢 Polish（择期）](#polish)
5. [功能缺失分析](#功能缺失分析)
6. [测试覆盖分析](#测试覆盖分析)
7. [本轮执行计划](#本轮执行计划)
8. [验证清单](#验证清单)

---

## 整体结论

项目经过 2026-03 和 2026-05 两轮大规模修复后，**稳定性核心链路（同步→变更→告警→治理事件）已 solid**。但当前状态从"功能可用"到"企业级完备"仍有显著差距。

**关键数据**:
- 后端测试覆盖率: **29.5%**（目标 60-80%）
- Store 层覆盖率: **13.1%**（PostgreSQL 路径 **0%**）
- 前端测试: 仅 8 个文件
- REMEDIATION_PLAN 中的 Phase B/C/D 功能**大面积未实现**

主要风险集中在三处:
1. **PostgreSQL 路径零测试**：docker-compose 生产默认走 PG，但所有测试跑 SQLite
2. **企业级功能缺失**：定时同步、监控指标、审计日志、RBAC 权限、数据仪表盘等均未实现
3. **配置安全隐患**：`.env.example` 的密钥占位符仍可能绕过检测

---

## 🔴 Critical（3 项）

### C1 — `.env.example` 加密密钥占位符仍有隐患
- `.env.example:19`：`ENCRYPTION_KEY=REPLACE_ME_WITH_32_BYTE_RANDOM_K`
- 正好 **32 字节**，且 `REPLACE_ME_WITH_...` 能绕过 `config.go:248-251` 的占位符检测（只检查 `change-me`/`replace_me`/`replace-me`，但 example 用的是下划线+大写混合）
- 如果用户直接 `cp .env.example .env` 后 `go run`（不走 docker-compose 的 fail-closed），会用弱密钥启动
- **修法**：把 example 值改成非 32 字节，并在 `config.go` 增强检测逻辑

### C2 — Oracle / MSSQL 扫描器"挂羊头卖狗肉"
- `internal/model/models.go:16-17` 保留 `DataSourceOracle`、`DataSourceMSSQL` 常量
- `migrations/001_init_schema.sql:14` 的 CHECK 约束包含 `'oracle', 'mssql'`
- 但 `cmd/datamap/main.go:135-137` 的 scanner registry **只注册了 mysql、postgres、mongodb**
- 用户若通过 API 直接创建 oracle/mssql 数据源，运行时会报错或行为未定义
- **修法**：从模型、迁移、前端类型中彻底移除（或至少禁用）未实现的类型

### C3 — PostgreSQL Store 零测试，但 docker-compose 默认就是 Postgres
- docker-compose.yml 默认注入 `DATAMAP_DATABASE_TYPE=postgres`
- `internal/store/postgres*.go`（12+ 文件，~2000 行）在测试中**完全未执行**
- 任何 PG 特有语法、事务行为、时间格式问题只在生产首次接收流量时暴露
- **修法**：补充 `internal/store/postgres_test.go` 和 `postgres_tx_test.go`，使用 testcontainers-go 或环境变量切换测试数据库

---

## 🟡 Important

### 测试覆盖

| 包 | 覆盖率 | 目标 | 状态 |
|---|---|---|---|
| `internal/store` | **13.1%** | 80% | PostgreSQL 路径 0% |
| `internal/api` | **41.0%** | 60% | handler 测试已补但不深 |
| `internal/service` | **55.6%** | 70% | notification、metadata 未覆盖 |
| `internal/scanner` | **20.9%** | - | PG scanner 有测试但覆盖不足 |
| `internal/model` | **0.0%** | - | 纯结构体也应补序列化测试 |
| `pkg/response` | **0.0%** | - | HTTP 统一响应格式无测试 |
| `pkg/utils` | **0.0%** | - | 工具函数无测试 |

### 前端测试缺口

前端测试仅 8 个文件，大量 hooks 和组件无测试:
- `DQStatsCard.tsx` / `DQResultCard.tsx`：无测试（pass_rate 显示逻辑）
- `api.test.ts`：未覆盖 401→refresh→失败→redirect 完整链路
- `useSchema.ts` / `useTerms.ts` / `useSources.ts` / `useColumnDetail`：无测试（race condition 保护）

### 文档一致性

- `REMEDIATION_TASKS.md` 底部的"整体进度跟踪"表显示所有阶段 **0% 完成**，与顶部"2026-03-28 当日收口"中 16+ 项已完成的事实严重矛盾
- `ENV_VARS.md` 已丢失（可能误删），`DATAMAP_AUTH_ENABLED` 默认值说明无处可查

---

## 🟢 Polish

- `internal/store/postgres_tx.go:333-334` 把 `interface{}` 当时间格式化的写法散布多处
- `docker-compose.yml:8` `version: '3.8'` 已废弃
- `web/Dockerfile` + `web/nginx.conf` 疑似死代码（当前走 embedded 模式）
- `pnpm-workspace.yaml` 存在但仅一个子包

---

## 功能缺失分析

以下功能在 REMEDIATION_PLAN 中有规划，但当前代码中**完全缺失**:

| 模块 | 缺失内容 | 影响 |
|---|---|---|
| **B1 规则模板库** | 无 `dq_rule_templates` 表、无 API、无 UI | DQ 规则每次从零手写 |
| **B2 Oracle/MSSQL 扫描器** | 类型声明存在但实现缺失 | 无法接入这两类数据库 |
| **B3 定时同步** | 无 cron 调度引擎、无 `sync_schedules` 表 | 只能手动同步 |
| **C1 数据资产仪表盘** | 无 `/dashboard` 页、无聚合统计 API | 无法一眼看到系统整体情况 |
| **C2 细粒度权限** | 只有 `users.role` (`admin`/`user`)，无资源级权限 | admin 能看/删一切 |
| **C2 数据分类/脱敏** | 无敏感等级、无分类标签、无脱敏规则 | 无法满足数据安全分级要求 |
| **C2 审计日志** | 只有 `governance_outbox`，无内部操作审计表 | 无法查询"谁改了什么" |
| **C3 可观测性** | 无 `/metrics` (Prometheus)、无 OpenTelemetry | 生产无法监控 |
| **C1 评论/评分** | 无 `column_comments` / `column_ratings` | 无法协作维护元数据 |
| **C1 数据预览** | Scanner 接口无 `PreviewData` 方法 | 无法从 UI 看字段实际内容 |

---

## 测试覆盖分析

当前整体覆盖率 **29.5%**。低于 60% 的包:

- `cmd/datamap` — 3.6%
- `cmd/embedassets` — 0.0%
- `internal/api` — 41.0%
- `internal/mcpserver` — 57.0%
- `internal/model` — 0.0%
- `internal/scanner` — 20.9%
- `internal/service` — 55.6%
- `internal/store` — 13.1% ⭐ 最关键
- `pkg/response` — 0.0%
- `pkg/utils` — 0.0%

**PostgreSQL 特有文件覆盖率为 0%**，包括:
- `internal/store/postgres.go`
- `internal/store/postgres_*.go`（共 12+ 文件）

---

## 本轮执行计划

### Step 1 — 安全与配置（P0，~2h）
| ID | 改动文件 | 内容 |
|---|---|---|
| C1 | `.env.example` | 密钥占位符改非 32 字节 |
| C1 | `internal/config/config.go` | 增强 `REPLACE_ME` 检测 |
| C2 | `internal/model/models.go` | 移除 `oracle`/`mssql` 常量 |
| C2 | `migrations/001_init_schema.sql` | 移除 CHECK 约束中的 `oracle`/`mssql` |

### Step 2 — PostgreSQL 测试覆盖（P0，~6-8h）
| ID | 改动文件 | 内容 |
|---|---|---|
| C3 | `internal/store/postgres_test.go` | 核心 CRUD 测试 |
| C3 | `internal/store/postgres_tx_test.go` | 事务路径测试 |
| C3 | `internal/store/store_test.go` | 根据 `DATAMAP_TEST_POSTGRES_DSN` 切换数据库 |

### Step 3 — 定时同步（P1，~8-12h）
| ID | 改动文件 | 内容 |
|---|---|---|
| B3 | `migrations/004_sync_schedules.sql` | 新增 `sync_schedules` 表 |
| B3 | `internal/model/models.go` | 新增 `SyncSchedule` 模型 |
| B3 | `internal/store/*.go` | 新增 sync schedule CRUD |
| B3 | `internal/service/scheduler.go` | cron 调度引擎 |
| B3 | `internal/api/router.go` | 新增 `/api/v1/sync/schedules` 路由 |
| B3 | `web/src/pages/SyncSchedulesPage.tsx` | 前端配置页面 |

### Step 4 — 可观测性（P1，~4-6h）
| ID | 改动文件 | 内容 |
|---|---|---|
| C3 | `internal/api/metrics.go` | Prometheus `/metrics` 端点 |
| C3 | `internal/api/health.go` | 增强健康检查（DB 连接） |
| C3 | `go.mod` | 新增 `prometheus/client_golang` |

### Step 5 — 企业级功能（P2，~2-3天）
| ID | 改动文件 | 内容 |
|---|---|---|
| C1 | `internal/api/dashboard.go` | 资产统计聚合 API |
| C1 | `web/src/pages/DashboardPage.tsx` | 数据资产仪表盘 |
| C2 | `migrations/005_audit_logs.sql` | 审计日志表 |
| C2 | `internal/api/audit.go` | 审计日志中间件 + API |

---

## 验证清单

修完后端:

```bash
make ci                          # backend lint + test + frontend lint + test + build
go test -race -cover ./...
# Postgres 矩阵
DATAMAP_TEST_POSTGRES_DSN=postgres://... go test -race ./internal/store/...
```

修完前端手测:

1. **C1**：`cp .env.example .env` → `go run ./cmd/datamap` → 应 fail，提示 encryption key 需替换
2. **C2**：尝试创建 oracle/mssql 数据源 → 应被校验拒绝
3. **定时同步**：配置 cron → 到时间自动触发同步
4. **/metrics**：`curl localhost:8080/metrics` → 返回 Prometheus 格式指标

---

## 一句话总结

**"稳定性债已还清，功能债刚开始。"** 系统核心链路已 solid，但企业级功能（定时同步、监控、权限、仪表盘）大面积缺失，测试覆盖率未达标。

---

## 修复记录

| 步骤 | 项 | 状态 | 关键文件 |
|---|---|---|---|
| Step 1 | C1 `.env.example` 密钥占位符 | 完成 | `.env.example`、`internal/config/config.go` |
| Step 1 | C2 移除 oracle/mssql | 完成 | `internal/model/models.go`、`migrations/001_init_schema.sql` |

**END OF DOCUMENT**
