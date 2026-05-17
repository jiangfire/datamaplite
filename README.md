# DataMap-Lite

轻量级数据目录系统，面向半导体显示研发部门的元数据治理解决方案。解决"同义不同名"（如 PanelID/plt_no/玻璃编号）的数据一致性问题。

## 功能特性

- **多数据源支持**: MySQL, PostgreSQL, MongoDB
- **Schema 浏览器**: 可视化展示表结构和字段信息
- **全局搜索**: 跨数据源字段搜索，支持模糊匹配
- **字段映射**: 建立同义不同名字段之间的关系
- **数据血缘**: 追踪数据流转路径，分析字段依赖
- **影响分析**: 评估字段变更的影响范围
- **业务术语**: 数据标准化管理
- **DDL 生成**: 支持 MySQL 和 PostgreSQL 的 DDL 导出

## 快速开始

### Docker Compose (推荐)

```bash
# 克隆仓库
git clone https://github.com/jiangfire/datamaplite.git
cd datamaplite

# 配置环境
cp .env.example .env
# 编辑 .env 设置数据库密码和加密密钥

# 启动服务
make docker-up
```

说明：
- 默认 `docker-compose.yml` 已切到适合中国网络环境的镜像源与构建参数，可直接使用 `goproxy.cn`、`sum.golang.google.cn`、`registry.npmmirror.com`
- SQLite 运行时已切换到纯 Go 驱动 `modernc.org/sqlite`，生产构建不再依赖 CGO

访问:
- Web UI: http://localhost:8080
- API: http://localhost:8080

## 统一 HTTP 响应结构

除 MCP 协议报文外，前端与后端 HTTP 接口统一使用 `pkg/response.HttpResult`：

```json
{
  "code": 0,
  "message": "",
  "error_code": "",
  "data": {}
}
```

约定如下：

- `code=0` 表示成功
- 非 `0` 的 `code` 使用对应 HTTP 语义错误码
- `error_code` 为稳定业务错误码，例如 `BAD_REQUEST`、`UNAUTHORIZED`、`NOT_FOUND`
- 业务数据统一放在 `data`

登录、刷新 token、业务增删改查都遵循这一结构；不要再使用旧的 `success/error` 包装格式

## 开发

### 后端开发

```bash
# 启动后端服务
go run ./cmd/datamap

# 运行测试
go test ./...

# 代码检查
golangci-lint run
```

### 前端开发

```bash
cd web

# 安装依赖
pnpm install

# 启动开发服务器
pnpm dev

# 构建
pnpm build
```

## 与 Cornerstone 联调

当前已支持向 `cornerstone` 推送两类标准治理事件：

- `metadata.schema.changed`
- `dq.rule.failed`

事件目标接口：

- `POST /api/integrations/events`

### 启用方式

在 `datamaplite` 环境变量中增加：

```env
DATAMAP_GOVERNANCE_ENABLED=true
DATAMAP_GOVERNANCE_ENDPOINT=http://localhost:8081/api/integrations/events
DATAMAP_GOVERNANCE_INTEGRATION_TOKEN=your-integration-token
DATAMAP_GOVERNANCE_SOURCE_SYSTEM=cornerstone
DATAMAP_GOVERNANCE_TIMEOUT=5s
```

说明：

- `DATAMAP_GOVERNANCE_ENABLED=false` 时不会发送治理事件
- `DATAMAP_GOVERNANCE_INTEGRATION_TOKEN` 需要与 `cornerstone` 的 `INTEGRATION_SHARED_TOKEN` 或 `INTEGRATION_TOKENS` 对应
- `DATAMAP_GOVERNANCE_SOURCE_SYSTEM` 默认值为 `cornerstone`

### 明日联调建议步骤

1. 在 `cornerstone` 配置 `INTEGRATION_SHARED_TOKEN` 或 `INTEGRATION_TOKENS`
2. 启动 `cornerstone`
3. 启动 `datamaplite` 并开启治理事件发送
4. 在 `datamaplite` 触发一次数据源同步，确认 `cornerstone` 自动生成结构变更任务
5. 在 `datamaplite` 触发一次 DQ 检查且保证至少一条失败，确认 `cornerstone` 自动生成 DQ 任务

### 当前边界

当前已完成：

- `datamaplite -> cornerstone` HTTP 治理事件推送
- 结构变更事件发送
- DQ 失败事件发送
- 治理事件 outbox 持久化与后台补偿投递

当前未完成：

- `dq.alert.triggered`
- `ai.recommendation.generated`
- `cornerstone -> datamaplite` 审核通过后回写

## MCP 服务

系统当前已提供一个可用的 MCP Server（Model Context Protocol Server，模型上下文协议服务），用于让 AI Agent 或支持 MCP 的客户端直接做元数据查询和治理动作。

### 适用场景

- 让 AI Agent 查询数据源、Schema、字段详情
- 通过 MCP 给字段绑定业务术语和标签
- 建立字段映射关系
- 触发数据源同步并读取最近结构变更

### 快速启动

MCP 现在只通过主服务的 HTTP 入口提供。启动后端即可：

```powershell
go run ./cmd/datamap
```

MCP Endpoint（MCP HTTP 端点）：

```text
http://localhost:8080/mcp
```

使用前提：

- 必须先通过 `/api/v1/auth/login` 获取 `access_token`
- 调用 MCP 时必须携带 `Authorization: Bearer <access_token>`
- 如果系统认证被关闭，`/mcp` 会直接拒绝访问，不再匿名开放

### 当前提供的治理工具

- `list_sources`
- `search_columns`
- `get_source_schema`
- `get_column_detail`
- `list_terms`
- `list_tags`
- `list_schema_changes`
- `assign_term_to_column`
- `assign_tags_to_column`
- `create_column_mapping`
- `trigger_source_sync`
- `create_term`
- `create_tag`
- `replay_governance_outbox_event`
- `get_governance_outbox_stats`
- `force_release_source_sync_lease`

### 当前提供的资源

- `datamap://catalog/sources`
- `datamap://catalog/terms`
- `datamap://catalog/tags`
- `datamap://governance/outbox`
- `datamap://governance/outbox/stats`
- `datamap://sources/{source_id}/schema`
- `datamap://columns/{column_id}`

### MCP 审计链

当启用治理事件推送时，MCP 写操作会额外发送 `mcp.governance.action` 事件。

- 每次 MCP 写操作都会生成独立 `trace_id`
- `trigger_source_sync` 触发后，后续 `metadata.schema.changed` 事件会复用同一条 trace
- 这样可以把“谁通过 MCP 发起了什么治理动作”与“后续触发了哪些结构治理事件”串起来

详细接入方式、客户端配置和治理流程见 [docs/MCP_USAGE.md](docs/MCP_USAGE.md)。

## 近期稳定性加固

截至 2026-03-28，围绕“同步 -> 结构变更 -> 告警 -> 治理事件”这条主链路，已额外完成一轮真实缺陷挖掘、修复和边界测试补强。

### 已修复的高价值问题

- `TriggerSync` 从进程内互斥升级为“进程内保护 + 数据库租约”，多实例共享同一数据库时也能拒绝同源并发同步
- 结构变更记录不再通过“查最新一条”回填，避免同时间戳下把错误变更串进审计链
- HTTP / MCP 触发的同步会把 `actor_id`、`trace_id`、`audit_origin`、`audit_operation` 继续透传到 `metadata.schema.changed`
- 修复 `drop_object` 事件丢失 `object_id`，保证对象级告警规则和治理审计链可追踪
- 修复 SQLite 告警链路中 `alert_rules` / `notifications` / `user_notifications` 非事务路径主键生成不一致问题
- 修复 SQLite 下通知创建时“边遍历用户边写关联”导致的单连接卡死问题
- 修复 webhook 失败后在请求取消场景下 `webhook_error` 无法持久化的问题
- 修复允许创建“`SourceID` 与 `ObjectID` 不属于同一数据源”的脏告警规则问题
- webhook 发送不再只依赖单个 `Idempotency-Key` 头；现在会同时在 Header 和 JSON Body 里带稳定幂等标识、通知 ID、规则 ID、变更 ID，并把接收方返回的 `409 Conflict` / `208 Already Reported` 视为重复确认成功
- 治理事件发送改为“先入本地 outbox，再尽力投递，再后台补偿”的最终一致性链路，并补齐死信、重放和统计能力；死信不会被自动再次 claim，手工重放也只允许失败/死信事件，避免误重发已成功事件
- 修复 SQLite 下 outbox `retryable_count` 对 RFC3339 时间串的统计失真，避免监控把“已到重试时间”的事件漏报为 0
- 同步租约新增“陈旧租约强制释放”能力，并修复 SQLite 下租约 `updated_at` 时间格式与 RFC3339 不兼容导致的治理失败
- 修复 SQLite 告警规则 `change_types` 用子串匹配导致的误触发问题，统一为逗号分隔 token 精确匹配
- 修复事务版通知创建在 `notify_in_app=false` 时仍然给所有用户生成未读消息的语义漂移问题
- 修复 `AssignTermToColumn` 在目标字段不存在时静默成功的问题，避免 MCP / 服务层出现“写成功但未生效”的假象
- 收紧 MCP 写工具空 ID 校验，避免“依赖未配置”掩盖真实坏输入
- 收紧字段搜索空白查询，避免纯空格请求退化成全表模糊搜索

### 已补的真实边界测试

- `saveSchema` 对象/字段增删改、对象和字段 ID 稳定性、血缘清理、默认值/指针等价比较
- `TriggerSync` 成功、扫描失败、落库失败、单实例并发拒绝、多实例并发互斥、早期解密失败后的恢复
- `AlertService` webhook 成功、4xx 不重试、5xx 重试成功、退避期间取消、重复变更去重、稳定幂等键复用、失败后重试复用同一通知
- 治理 outbox 入队、重复 `event_id` 去重、失败后补偿投递、多 dispatcher 抢占同一事件时的租约互斥、死信、重放和统计
- MCP / HTTP 写操作到治理事件、再到结构变更事件的 trace 透传
- MCP 治理入口最小测试：工具/资源注册、治理 outbox 重放入口、缺省禁用保护
- 同步租约读取、陈旧租约拒绝/释放、SQLite 时间格式兼容解析

### 当前仍是已知边界

- 同步租约底层仍是 TTL 模型；当前已支持通过 MCP / 服务端接口手工释放陈旧租约，但若完全无人干预，实例崩溃后仍以 TTL 作为兜底接管窗口
- webhook 的最终业务去重仍取决于接收方是否正确消费 Header / Body 中的幂等字段；如果对方完全忽略这些字段，仍可能产生业务级重复消费
- 治理 outbox 已有死信、重放和基础统计，但还没有独立的可视化运维面板与告警阈值

## Makefile 命令

```bash
make help          # 显示所有可用命令
make dev           # 同时启动前后端开发服务器
make docker-up     # 使用 Docker Compose 启动服务
make docker-down   # 停止 Docker 服务
make test          # 运行所有测试
make lint          # 运行代码检查
make build         # 构建所有二进制文件
make clean         # 清理构建产物
```

## 部署

详见 [部署文档](docs/DEPLOYMENT.md)

### 生产环境部署

```bash
# 使用 Docker Compose
docker-compose up -d

# 或使用预构建镜像
wget https://github.com/jiangfire/datamaplite/raw/branch/main/docker-compose.yml
docker-compose up -d
```

## 系统架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   React Frontend │────▶│   Go Backend    │────▶│  Metadata DB   │
│   (Rsbuild)     │     │   (Gin)         │     │  (SQLite/PG)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │  Target DBs     │
                        │ MySQL/PG/MongoDB│
                        └─────────────────┘
```

## 项目结构

```
datamaplite/
├── cmd/datamap/          # 应用入口
├── internal/
│   ├── api/              # HTTP handlers
│   ├── service/          # 业务逻辑
│   ├── store/            # 数据访问层
│   ├── scanner/          # 元数据采集器
│   ├── model/            # 数据模型
│   ├── config/           # 配置管理
│   └── crypto/           # 加密工具
├── web/                  # React 前端
├── migrations/           # 数据库迁移
├── docs/                 # 文档
└── docker-compose.yml    # Docker 部署配置
```

## CI/CD

- **CI**: GitHub Actions 分离执行后端测试、前端构建、嵌入式构建校验和 Docker 构建校验
- **CD**: 自动构建 Docker 镜像并推送到可配置的 OCI 仓库，默认按中国区仓库场景设计（推荐阿里云 ACR / 腾讯云 TCR / 华为云 SWR）
- **发布**: 打 tag 自动生成 Linux / Windows / macOS 多平台二进制并创建 GitHub Release

## 技术栈

**后端:**
- Go 1.25
- Gin Web Framework
- PostgreSQL (pgx)
- SQLite（`modernc.org/sqlite` 纯 Go 驱动，无 CGO）
- 内置启动迁移
- Zap (日志)
- Viper (配置)

**前端:**
- React 19
- TypeScript
- Rsbuild
- Tailwind CSS 4
- React Router 7

**运维:**
- Docker & Docker Compose
- GitHub Actions
- Makefile

## 贡献

欢迎提交 Issue 和 Pull Request!

## 许可证

MIT License

## 联系方式

- 仓库: https://github.com/jiangfire/datamaplite
