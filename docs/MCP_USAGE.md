# DataMap MCP 使用说明

本文说明如何通过 HTTP 方式接入 DataMap MCP 服务，并通过该服务做基础数据治理操作。

## 1. 能力边界

当前 MCP 服务提供两类能力：

- **查询类**
  - 列出数据源、术语、标签
  - 搜索字段
  - 获取数据源 Schema（数据库结构元数据）
  - 获取字段详情
  - 查看最近结构变更
- **治理写操作**
  - 给字段绑定/解绑业务术语
  - 给字段打标签
  - 创建字段映射
  - 触发数据源同步
  - 创建业务术语
  - 创建标签

当前实现是 **可用版本（MVP，最小可用版本）**，重点覆盖“查目录 + 做治理动作”的主链路，不包含审批流、回写联动和批量事务补偿。

## 2. 启动前准备

MCP 服务复用主系统配置，因此需要和 `datamap` 一样的数据库与密钥环境变量。

至少确认以下配置可用：

```env
DATAMAP_ENCRYPTION_KEY=12345678901234567890123456789012
DATAMAP_AUTH_JWT_SECRET=change-this-jwt-secret
```

如果你使用 `.env` 或 `configs/config.yaml`，保持与主服务一致即可。

## 3. 启动方式

MCP 现在不再提供独立的 CLI / stdio Server。

它只作为主服务的一条 HTTP 路由暴露，因此启动方式就是启动后端：

```powershell
go run ./cmd/datamap
```

或：

```powershell
make backend
```

默认 MCP Endpoint：

```text
http://localhost:8080/mcp
```

说明：

- 传输方式为 **Streamable HTTP（基于 HTTP 的 MCP 传输）**
- MCP 与主 API 共用同一个进程、同一个监听端口
- MCP 入口现在要求 **Bearer Token（HTTP Bearer 访问令牌）**；必须先通过主系统登录拿到 `access_token`
- 如果系统认证被关闭，MCP 入口会返回不可用，不再匿名开放
- 当前没有单独的 MCP 鉴权层，若对外暴露，建议放在反向代理、内网网关或 API Gateway 后面

## 4. MCP Client 配置示例

当前应使用支持 **HTTP MCP / Streamable HTTP** 的客户端，并把服务地址指向 `/mcp`。

通用配置要点：

- MCP Server URL：`http://localhost:8080/mcp`
- Transport：`streamable_http` 或客户端等价配置
- Header：`Authorization: Bearer <access_token>`
- 如果部署在反向代理后，填写代理后的完整 URL

示例：

```json
{
  "mcpServers": {
    "datamap": {
      "transport": "streamable_http",
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer <access_token>"
      }
    }
  }
}
```

不同客户端的字段名可能略有差异，但核心都是“通过 HTTP URL 连接”，不再是拉起本地命令。

获取 token 的方式与主系统 API 一致，例如：

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "your-password"
}
```

HTTP 登录接口返回也遵循主系统统一响应结构：

```json
{
  "code": 0,
  "data": {
    "access_token": "<access_token>",
    "refresh_token": "<refresh_token>",
    "expires_in": 3600
  }
}
```

## 5. 当前暴露的 Tools

| Tool | 用途 |
|---|---|
| `list_sources` | 列出所有已登记数据源 |
| `search_columns` | 按关键词搜索字段 |
| `get_source_schema` | 获取指定数据源的 Schema 树 |
| `get_column_detail` | 获取指定字段详情 |
| `list_terms` | 列出业务术语 |
| `list_tags` | 列出治理标签 |
| `list_schema_changes` | 查看指定数据源最近结构变更 |
| `assign_term_to_column` | 给字段绑定或解绑术语 |
| `assign_tags_to_column` | 给字段绑定一个或多个标签 |
| `create_column_mapping` | 创建字段语义映射 |
| `trigger_source_sync` | 触发数据源元数据同步 |
| `create_term` | 新建业务术语 |
| `create_tag` | 新建标签 |
| `replay_governance_outbox_event` | 手工重放指定治理 outbox 死信/失败事件 |
| `get_governance_outbox_stats` | 查看治理 outbox 聚合统计 |
| `force_release_source_sync_lease` | 强制释放陈旧同步租约 |

## 6. 当前暴露的 Resources

| Resource URI | 用途 |
|---|---|
| `datamap://catalog/sources` | 全量数据源目录 |
| `datamap://catalog/terms` | 全量业务术语目录 |
| `datamap://catalog/tags` | 全量标签目录 |
| `datamap://governance/outbox` | 最近治理 outbox 事件与死信列表 |
| `datamap://governance/outbox/stats` | 治理 outbox 统计数据 |
| `datamap://sources/{source_id}/schema` | 指定数据源 Schema |
| `datamap://columns/{column_id}` | 指定字段详情 |

## 7. 典型治理流程

### 场景 1：识别“同义不同名”字段并建立治理关系

1. 调用 `search_columns`，搜索例如 `panel`、`glass`、`plt_no`
2. 调用 `get_column_detail`，确认字段来源、类型、置信度和已绑定术语
3. 对标准字段调用 `assign_term_to_column`
4. 对同类字段调用 `assign_tags_to_column`
5. 如存在等价关系，调用 `create_column_mapping`

### 场景 2：治理新接入数据源

1. 调用 `list_sources` 找到目标数据源
2. 调用 `trigger_source_sync` 触发同步
3. 调用 `get_source_schema` 检查新采集的对象和字段
4. 调用 `list_schema_changes` 查看最近新增/变更项
5. 对关键字段补充术语、标签和映射关系

## 8. 审计链路

当 `DATAMAP_GOVERNANCE_ENABLED=true` 时，MCP 写操作会额外发送治理审计事件：

- 事件类型：`mcp.governance.action`
- 事件内容：写操作名称、资源类型、资源 ID、关键输入参数
- 审计来源：`audit_origin=mcp`

其中 `trigger_source_sync` 比其他写操作多一层链路：

1. MCP 先发送 `mcp.governance.action`
2. 同一次操作生成的 `trace_id` 会继续透传到后续结构变更事件
3. 后续 `metadata.schema.changed` 会带上相同 trace，用于串联“触发动作”和“结果事件”

截至 2026-03-28，这条链路已经额外做过一轮稳定性加固：

- `trigger_source_sync` 已升级为“进程内保护 + 数据库租约”双层互斥，多实例共享同一数据库时也会拒绝同源并发同步
- 同步触发的 `metadata.schema.changed` 会继续携带同一条 `trace_id`
- `drop_object` 场景现在会带上稳定 `object_id`，对象级规则和外部治理平台可以准确关联到被删对象
- webhook 会同时在 Header 与 JSON Body 中带稳定幂等标识、通知 ID、规则 ID、变更 ID；重复处理同一 `(rule_id, change_id)` 时会复用同一条通知记录，接收方返回 `409` / `208` 也会被视为重复确认成功
- 治理事件现在会先写入本地 outbox（外发表），再尽力投递；如果外部治理平台暂时不可用，会由后台 dispatcher（补偿投递器）继续重试，达到上限后进入死信，并可通过 MCP 手工重放；死信不会被自动再次 claim，且手工重放只允许失败/死信事件
- 同步租约除自动 TTL 接管外，还新增了“陈旧租约强制释放”入口，便于实例异常退出后的人工治理

这条链路现在具备**本地持久化 + 最终一致性（允许短时失败，最终补偿成功）**能力，但仍有边界：

- 同步租约底层仍是 TTL 模型；当前只是额外提供了陈旧租约的人工释放入口
- webhook 的最终去重仍依赖接收方正确消费 Header / Body 中的幂等字段
- outbox 已具备死信、手工重放和基础统计，但还没有独立运维面板和告警阈值

## 9. 已知限制

- 当前没有单独的 MCP 鉴权层，权限边界仍依赖服务部署环境、反向代理和网络边界
- 没有批量回滚和多步骤事务编排
- `assign_tags_to_column` 当前是追加写入，不包含“覆盖式重置标签”语义
- 资源以只读查询为主，写操作全部通过 tool 完成
- outbox 当前是系统内透明运行，没有面向运维的可视化管理界面

## 10. 推荐的后续增强

- 增加 `remove_term_from_column`、`remove_tags_from_column` 等显式治理动作
- 增加批量治理工具，减少客户端多次往返调用
- 增加审批、审计、事件回写链路
- 增加面向 AI Agent 的治理建议工具，例如“推荐术语”“推荐映射”
- 为 webhook / outbox 增加可观测性面板和告警阈值
- 为 `datamap://governance/outbox` 增加过滤、分页和死信专用视图
