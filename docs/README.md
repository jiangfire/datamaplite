# DataMap-Lite 整改文档目录

本目录包含 DataMap-Lite 代码审查后的完整整改方案和实施文档。

---

## 📚 文档清单

| 文档 | 用途 | 目标读者 |
|------|------|----------|
| [REMEDIATION_PLAN.md](./REMEDIATION_PLAN.md) | 整体整改方案计划 | 项目经理、技术负责人、架构师 |
| [REMEDIATION_TASKS.md](./REMEDIATION_TASKS.md) | 详细任务跟踪清单 | 开发团队、测试团队 |
| [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) | 技术实施指南 | 后端/前端开发工程师 |
| [MCP_USAGE.md](./MCP_USAGE.md) | MCP 服务启动、接入与治理使用说明 | 后端工程师、平台工程师、AI Agent 集成人员 |
| [GITEA_ACTIONS_COMPATIBILITY.md](./GITEA_ACTIONS_COMPATIBILITY.md) | Gitea Actions 版本兼容与选型说明 | 平台工程师、后端工程师、CI 维护者 |

---

## 🚀 快速导航

### 如果你是项目经理
👉 先看 [REMEDIATION_PLAN.md](./REMEDIATION_PLAN.md) 了解：
- 整改目标和范围
- 时间规划（20周分4个Phase）
- 资源需求和里程碑
- 风险评估

### 如果你是开发工程师
👉 先看 [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md) 了解：
- SQL注入防护的具体实现
- 测试覆盖率提升的方法
- Oracle/MSSQL扫描器开发
- 定时同步机制实现
- 监控指标集成

### 如果你要接 AI Agent / MCP Client
👉 先看 [MCP_USAGE.md](./MCP_USAGE.md) 了解：
- 如何通过主服务的 `/mcp` HTTP 入口接入
- MCP Client 如何配置
- 当前有哪些可用 tools / resources
- 如何通过 MCP 做字段术语、标签和映射治理

### 如果你是Team Leader
👉 使用 [REMEDIATION_TASKS.md](./REMEDIATION_TASKS.md) 进行：
- 任务分配和跟踪
- 进度监控
- 里程碑验收

---

## 📋 整改概览

### HTTP 响应契约

除 MCP 协议本身外，系统 HTTP 接口统一使用 `pkg/response.HttpResult`：

```json
{
  "code": 0,
  "message": "",
  "error_code": "",
  "data": {}
}
```

含义约定：

- `code=0` 为成功
- 失败时 `code` 为对应 HTTP 语义码
- `error_code` 为稳定业务错误码
- `data` 为实际载荷

`success/error` 兼容包装层已经移除，前后端联调与测试都应直接按该结构解析

### 2026-03-28 进展快照

本轮不是单纯补测试覆盖率，而是针对主链路做“真实问题导向”的深挖，当前已落地：

- `internal/service/source.go`
  - 修复同一数据源重复 `TriggerSync` 的并发覆盖风险
  - 引入数据库租约，补齐多实例共享同库时的同步互斥
  - 修复 `drop_object` 审计事件丢失 `object_id`
- `internal/service/alert.go`
  - 修复 webhook 失败状态在取消场景下无法持久化
  - 增加告警规则 `source_id` / `object_id` 一致性校验
  - 增加 webhook 稳定幂等键和通知去重复用
- `internal/store/sqlite_alert.go`
  - 修复 SQLite 非事务路径 `alert_rules` / `notifications` / `user_notifications` 主键生成问题
  - 修复单连接 SQLite 下通知落库时的死锁式卡住问题
- `internal/service/governance_event.go`
  - 引入治理事件 outbox 持久化、后台补偿投递和重复 `event_id` 去重
  - 补齐死信、手工重放和聚合统计
- `internal/mcpserver/server.go`
  - 增加治理 outbox 重放、统计和陈旧同步租约释放入口
- `internal/api/middleware.go`
  - HTTP 写请求的审计信息可继续透传到治理事件和结构变更事件
- `internal/service/source_sqlite_test.go`
  - 补充 `saveSchema`、`TriggerSync`、多实例互斥、审计链、陈旧租约释放和恢复路径的 SQLite 真实测试
- `internal/service/alert_integration_test.go`
  - 补充 webhook 成功 / 4xx / 5xx 重试 / 取消 / 幂等 / 通知复用 / 409 重复确认 / 对象级删除告警等真实集成测试
- `internal/service/governance_event_test.go`
  - 补充 outbox 持久化、最终一致性补投递、重复事件去重、多 dispatcher 租约互斥、死信、重放、重复确认测试
- `internal/mcpserver/server_test.go`
  - 补充治理 outbox 工具/资源与禁用保护的最小测试

### 当前已验证通过

```powershell
$env:GOCACHE='C:\Users\yimo\Codes\datamaplite\.gocache'; go test ./internal/service -count=1
$env:GOCACHE='C:\Users\yimo\Codes\datamaplite\.gocache'; go test ./... -count=1
```

### 当前剩余主要风险

- webhook 的最终业务去重仍依赖外部接收方正确消费 Header / Body 中的幂等字段
- outbox 已具备死信、重放和统计，但仍缺少可视化运维面板与告警阈值
- 同步租约依赖 TTL 作为兜底；虽然已经支持人工释放陈旧租约，但完全无人干预时仍存在短暂接管等待窗口

### 整改阶段

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Phase A (Week 1-4)    │   Phase B (Week 5-10)   │  Phase C (Week 11-16)   │
│  安全与质量加固          │   功能完善               │   企业级特性            │
│                        │                         │                         │
│  🔒 SQL注入修复         │   🔌 Oracle/MSSQL支持   │   🏢 资产总览           │
│  🧪 测试覆盖提升         │   ⏰ 定时同步机制        │   🔐 权限控制           │
│  📝 代码质量优化         │   📊 DQ模块增强         │   📈 可观测性           │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  Phase D (Week 17-20)                                                       │
│  前端优化                                                                    │
│                                                                            │
│  ⚛️ 架构升级 (Zustand + React Query)                                        │
│  🎨 功能页面 (资产总览 + 血缘可视化)                                         │
│  ✨ 体验优化 (加载状态 + 响应式)                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 关键问题修复

| 优先级 | 问题 | 状态 | 影响 |
|--------|------|------|------|
| 🔴 P0 | SQL注入风险 | 待修复 | 安全风险 |
| 🔴 P0 | Store层测试覆盖率6.6% | 待提升 | 代码质量 |
| 🟠 P1 | Oracle/MSSQL扫描器缺失 | 待实现 | 功能完整 |
| 🟠 P1 | 定时自动同步缺失 | 待实现 | 运维效率 |
| 🟡 P2 | 前端架构待升级 | 待优化 | 用户体验 |

---

## 🎯 关键里程碑

| 里程碑 | 时间 | 关键交付 |
|--------|------|----------|
| **M1** | Week 4 | 安全扫描通过，测试覆盖>60% |
| **M2** | Week 10 | Oracle/MSSQL支持，定时同步上线 |
| **M3** | Week 16 | 权限/审计/监控体系建成 |
| **M4** | Week 20 | 前端重构完成，体验评分>4.0 |

---

## 🛠️ 快速开始

### 1. 环境准备

```bash
# 安装开发工具
go install github.com/golang/mock/mockgen@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 前端依赖
cd web
npm install @tanstack/react-query zustand echarts @antv/g6
```

### 2. 查看当前问题

```bash
# 运行测试查看覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 安全扫描
gosec ./...

# 代码检查
golangci-lint run
```

### 3. 按Phase实施

建议按照Phase A → B → C → D 的顺序实施，每个Phase完成后进行验收。

---

## 📊 验收标准

### 代码质量
- [ ] 测试覆盖率：核心包 >80%，平均 >60%
- [ ] 安全扫描：0个High/Medium风险
- [ ] 代码检查：0个Error，Warning < 10

### 功能验收
- [ ] SQL注入防护：SQLMap扫描无漏洞
- [ ] Oracle/MSSQL：支持11g+/2012+，准确率>99%
- [ ] 定时同步：支持cron，失败重试，有告警

### 性能指标
- [ ] API响应时间(P99) < 500ms
- [ ] 并发用户数 > 100
- [ ] 前端首屏加载 < 3s

---

## 📞 问题反馈

如在整改实施过程中遇到问题：
1. 技术实现问题 → 参考 [IMPLEMENTATION_GUIDE.md](./IMPLEMENTATION_GUIDE.md)
2. 进度跟踪问题 → 更新 [REMEDIATION_TASKS.md](./REMEDIATION_TASKS.md)
3. 方案调整需求 → 在 REMEDIATION_PLAN.md 的"备注区"记录

---

## 📅 更新记录

| 日期 | 版本 | 变更内容 |
|------|------|----------|
| 2026-03-24 | v1.0 | 初始版本，包含完整整改方案 |
| 2026-03-27 | v1.1 | 补充本轮真实缺陷修复、边界测试和当前风险快照 |
| 2026-03-28 | v1.2 | 收口 webhook 幂等、outbox 运维能力、同步租约治理入口，并补齐 MCP / outbox / 租约边界测试 |

---

**开始整改前，建议团队集体评审 [REMEDIATION_PLAN.md](./REMEDIATION_PLAN.md) 确保理解一致。**
