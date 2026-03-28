# DataMap-Lite 整体整改方案计划

**版本**: v1.0  
**日期**: 2026-03-24  
**编制**: Code Review Team  

---

## 1. 现状分析

### 1.1 项目概况
DataMap-Lite 是一个轻量级数据目录系统，当前已实现核心元数据管理功能，包括数据源管理、Schema采集、字段搜索、血缘分析、业务术语管理等。

### 1.2 主要问题清单

#### 🔴 严重问题 (P0)
| 序号 | 问题描述 | 影响范围 | 风险等级 |
|------|----------|----------|----------|
| 1 | SQL注入风险 (`dq_executor.go` 自定义SQL规则) | 数据安全 | 高 |
| 2 | Store层测试覆盖率仅 6.6% | 代码质量 | 高 |
| 3 | JSON解析错误被静默忽略 | 数据完整性 | 中 |

#### 🟠 中等问题 (P1)
| 序号 | 问题描述 | 影响范围 | 风险等级 |
|------|----------|----------|----------|
| 4 | Oracle/MSSQL扫描器未实现 | 功能完整性 | 中 |
| 5 | 缺少定时自动同步机制 | 运维效率 | 中 |
| 6 | 事件发布goroutine错误处理缺失 | 可观测性 | 中 |
| 7 | 事务嵌套潜在问题 | 数据一致性 | 中 |

#### 🟡 一般问题 (P2)
| 序号 | 问题描述 | 影响范围 | 风险等级 |
|------|----------|----------|----------|
| 8 | 前端缺少全局错误边界 | 用户体验 | 低 |
| 9 | 硬编码状态值 | 代码规范 | 低 |
| 10 | 缺少数据资产总览 | 功能缺失 | 低 |

---

## 2. 整改目标

### 2.1 总体目标
通过系统性整改，将 DataMap-Lite 从**功能可用**提升到**生产就绪**水平，满足企业级数据治理平台的基本要求。

### 2.2 具体目标

| 维度 | 现状 | 目标 | 衡量标准 |
|------|------|------|----------|
| **测试覆盖** | 平均 ~20% | 核心模块 >80% | `go test -cover` |
| **代码质量** | 存在技术债 | 无明显技术债 | SonarQube 扫描 |
| **功能完整** | 核心功能有 | 企业级功能完备 | 功能清单验收 |
| **安全合规** | 存在SQL注入风险 | 无高危安全漏洞 | 安全扫描报告 |
| **可观测性** | 基础日志 | 全链路可观测 | Trace/Metrics |
| **用户体验** | 基础功能可用 | 体验流畅 | 用户反馈评分 |

---

## 3. 整改内容规划

### Phase A: 安全与质量加固 (4周)

#### A1. 安全漏洞修复 (1周)

**任务清单**:
- [ ] **A1.1** SQL注入防护 (`service/dq_executor.go`)
  - 实现SQL语法解析验证
  - 添加白名单机制（只允许 SELECT）
  - 参数化查询重构
  - 添加SQL审计日志
  
- [ ] **A1.2** 敏感数据脱敏
  - 连接配置加密存储 ✅ 已存在，需审计
  - 密码字段返回脱敏
  - API 响应敏感信息过滤

**验收标准**:
```bash
# SQLMap 扫描无高危漏洞
sqlmap -u "http://localhost:8080/api/v1/dq/check" --batch

# 代码安全扫描通过
gosec -fmt sarif -out security.sarif ./...
```

#### A2. 测试覆盖率提升 (2周)

**任务清单**:
- [ ] **A2.1** Store层测试 (覆盖率达到 80%)
  - `store/postgres_test.go` - PostgreSQL实现测试
  - `store/sqlite_test.go` - SQLite实现测试
  - `store/transaction_test.go` - 事务测试
  
- [ ] **A2.2** Service层补充测试 (覆盖率达到 70%)
  - `service/dq_test.go` - DQ服务测试
  - `service/alert_test.go` - 告警服务测试
  - `service/notification_test.go` - 通知服务测试

- [ ] **A2.3** API层集成测试 (覆盖率达到 60%)
  - 端到端API测试
  - 认证流程测试
  - 错误处理测试

**测试框架**:
```go
// 使用 testcontainers-go 进行数据库集成测试
// 使用 httptest 进行 HTTP 测试
// 使用 mockgen 生成 Store 接口 mock
```

#### A3. 代码质量优化 (1周)

**任务清单**:
- [ ] **A3.1** 错误处理规范化
  - 统一错误类型定义
  - 错误包装标准化 (`fmt.Errorf("...: %w", err)`)
  - JSON解析错误处理修复

- [ ] **A3.2** 常量提取
  - 状态值常量化
  - 魔法数字消除
  - 配置参数结构化

- [ ] **A3.3** 代码重构
  - PostgreSQL/SQLite 重复代码抽象
  - 事件发布机制重构
  - goroutine 错误处理改进

### Phase B: 功能完善 (6周)

#### B1. 数据质量模块增强 (2周)

**任务清单**:
- [ ] **B1.1** 规则模板库
  ```go
  // 预定义常用规则模板
  var RuleTemplates = map[string]DQRuleTemplate{
      "email": {
          Name: "邮箱格式校验",
          RuleType: "regex",
          Config: map[string]interface{}{
              "pattern": `^[\w.-]+@[\w.-]+\.\w+$`,
          },
      },
      "phone": {...},
      "id_card": {...},
  }
  ```

- [ ] **B1.2** 历史趋势分析
  - 质量分数趋势图表
  - 规则执行历史查询
  - 数据质量报告生成

- [ ] **B1.3** 规则调度器
  - 基于 cron 的定时任务
  - 支持一次性/周期性执行
  - 执行历史记录

#### B2. 扫描器扩展 (2周)

**任务清单**:
- [ ] **B2.1** Oracle扫描器实现
  ```go
  type OracleScanner struct {
      // 使用 github.com/godror/godror
  }
  ```
  
- [ ] **B2.2** MSSQL扫描器实现
  ```go
  type MSSQLScanner struct {
      // 使用 github.com/microsoft/go-mssqldb
  }
  ```

- [ ] **B2.3** 扫描器性能优化
  - 并发采集支持
  - 大数据量分页处理
  - 采样策略优化

#### B3. 自动同步机制 (2周)

**任务清单**:
- [ ] **B3.1** 调度引擎
  ```go
  type SyncScheduler struct {
      cron     *cron.Cron
      store    store.Store
      registry *scanner.Registry
  }
  ```

- [ ] **B3.2** 同步策略配置
  - 按数据源配置同步周期
  - 支持增量/全量同步
  - 同步窗口设置

- [ ] **B3.3** 同步监控
  - 同步任务状态跟踪
  - 失败重试机制
  - 同步结果通知

### Phase C: 企业级特性 (6周)

#### C1. 数据资产目录 (2周)

**任务清单**:
- [ ] **C1.1** 资产总览仪表盘 (前端)
  - 数据源统计卡片
  -  Schema变更趋势图
  - 数据质量评分看板
  - 热门搜索关键词

- [ ] **C1.2** 资产目录树
  - 按业务域分类展示
  - 支持标签筛选
  - 快速搜索定位

- [ ] **C1.3** 元数据详情增强
  - 数据样本预览
  - 数据分布统计
  - 评论与评分

#### C2. 权限与安全 (2周)

**任务清单**:
- [ ] **C2.1** 细粒度权限控制
  ```go
  // 基于RBAC的权限模型
  type Permission struct {
      Resource string // source, schema, term, etc.
      Action   string // read, write, admin
      Scope    string // global, source-specific
  }
  ```

- [ ] **C2.2** 数据分级分类
  - 敏感等级标签（公开/内部/机密/绝密）
  - 数据分类标准（个人/财务/业务等）
  - 脱敏规则配置

- [ ] **C2.3** 审计日志
  - 操作日志记录
  - 数据访问审计
  - 日志查询分析

#### C3. 可观测性建设 (2周)

**任务清单**:
- [ ] **C3.1** 指标监控 (Metrics)
  ```go
  // Prometheus 指标
  var (
      httpRequestsTotal = prometheus.NewCounterVec(...)
      dbQueryDuration = prometheus.NewHistogram(...)
      syncDuration = prometheus.NewGauge(...)
  )
  ```

- [ ] **C3.2** 链路追踪 (Tracing)
  - OpenTelemetry 集成
  - 跨服务调用追踪
  - 性能瓶颈分析

- [ ] **C3.3** 健康检查增强
  - 数据库连接健康
  - 外部依赖检查
  - 磁盘/内存监控

### Phase D: 前端优化 (4周)

#### D1. 架构升级 (1周)

**任务清单**:
- [ ] **D1.1** 状态管理引入
  ```typescript
  // 使用 Zustand 替代 useState
  interface AppState {
      user: UserInfo | null;
      permissions: string[];
      theme: 'light' | 'dark';
  }
  ```

- [ ] **D1.2** React Query 集成
  - 服务端状态管理
  - 缓存策略优化
  - 乐观更新支持

- [ ] **D1.3** 错误边界
  ```typescript
  class ErrorBoundary extends React.Component {
      // 全局错误捕获
      // 错误上报
      // 降级UI展示
  }
  ```

#### D2. 功能页面 (2周)

**任务清单**:
- [ ] **D2.1** 数据资产总览页
  - 统计卡片组件
  - ECharts 图表集成
  - 实时数据刷新

- [ ] **D2.2** 数据预览弹窗
  - 表格数据预览
  - JSON数据格式化
  - 分页加载

- [ ] **D2.3** 血缘可视化增强
  - 交互式图谱（使用 AntV G6 或 D3.js）
  - 缩放/拖拽/高亮
  - 路径分析

#### D3. 体验优化 (1周)

**任务清单**:
- [ ] **D3.1** 加载状态统一
  - Skeleton 骨架屏
  - 加载动画规范
  
- [ ] **D3.2** 消息通知
  - 操作成功/失败提示
  - 通知中心
  
- [ ] **D3.3** 响应式适配
  - 移动端基础适配
  - 表格横向滚动优化

---

## 4. 实施计划

### 4.1 时间规划

```
Week 1-4  : Phase A - 安全与质量加固
Week 5-10 : Phase B - 功能完善
Week 11-16: Phase C - 企业级特性
Week 17-20: Phase D - 前端优化

总计: 20周 (约5个月)
```

### 4.2 里程碑

| 里程碑 | 时间 | 交付物 | 验收标准 |
|--------|------|--------|----------|
| M1 | Week 4 | 安全加固完成 | 安全扫描通过，测试覆盖>60% |
| M2 | Week 10 | 功能完善完成 | Oracle/MSSQL支持，定时同步 |
| M3 | Week 16 | 企业特性完成 | 权限/审计/监控上线 |
| M4 | Week 20 | 全面优化完成 | 用户体验评分>4.0 |

### 4.3 资源需求

**人力资源**:
- Go后端开发: 2人
- 前端开发: 1-2人
- 测试工程师: 1人 (Week 2-4集中投入)
- DevOps工程师: 0.5人 (Week 15-16支持)

**技术资源**:
- Oracle测试环境
- MSSQL测试环境
- Prometheus + Grafana 监控平台
- SonarQube 代码质量平台

---

## 5. 风险与应对

### 5.1 技术风险

| 风险 | 概率 | 影响 | 应对措施 |
|------|------|------|----------|
| Oracle/MSSQL驱动兼容性问题 | 中 | 高 | 提前进行技术预研，准备备选方案 |
| 事务重构引入数据一致性问题 | 中 | 高 | 增加集成测试覆盖，灰度发布 |
| 前端架构升级引入回归问题 | 高 | 中 | 建立自动化E2E测试 |

### 5.2 进度风险

| 风险 | 概率 | 影响 | 应对措施 |
|------|------|------|----------|
| 测试覆盖提升进度慢 | 中 | 中 | 引入测试自动生成工具，调整优先级 |
| 企业级特性需求变更 | 高 | 中 | 敏捷迭代，分期交付 |

### 5.3 运维风险

| 风险 | 概率 | 影响 | 应对措施 |
|------|------|------|----------|
| 自动同步任务影响源库性能 | 低 | 高 | 限流、限并发、只读账号 |
| 监控数据存储成本 | 中 | 低 | 设置数据保留策略 |

---

## 6. 验收标准

### 6.1 代码质量验收

```bash
# 测试覆盖检查
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
# 要求: 核心包 > 80%, 平均 > 60%

# 代码扫描检查
golangci-lint run --timeout 5m
# 要求: 0个Error，Warning < 10

# 安全扫描检查
gosec -quiet ./...
# 要求: 0个High/Medium风险
```

### 6.2 功能验收

| 功能模块 | 验收标准 |
|----------|----------|
| SQL注入防护 | SQLMap扫描无注入漏洞，自定义SQL只允许SELECT |
| Oracle/MSSQL | 支持11g+/2012+版本，采集准确率>99% |
| 定时同步 | 支持cron表达式，失败重试3次，有告警通知 |
| 权限控制 | 支持资源级权限，权限变更实时生效 |
| 资产总览 | 数据刷新延迟<5s，图表渲染<2s |

### 6.3 性能验收

| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| API响应时间(P99) | < 500ms | k6压力测试 |
| 并发用户数 | > 100 | JMeter并发测试 |
| 数据库连接池 | < 50 | 监控观察 |
| 前端首屏加载 | < 3s | Lighthouse |

---

## 7. 附录

### 7.1 依赖库建议

**后端新增依赖**:
```go
// SQL解析
github.com/xwb1989/sqlparser // SQL语法解析

// 定时任务
github.com/robfig/cron/v3

// 监控
github.com/prometheus/client_golang

go.opentelemetry.io/otel

// 测试
github.com/testcontainers/testcontainers-go
github.com/golang/mock/mockgen
```

**前端新增依赖**:
```json
{
  "zustand": "^4.4.0",
  "@tanstack/react-query": "^5.0.0",
  "echarts": "^5.4.0",
  "@antv/g6": "^4.8.0",
  "react-error-boundary": "^4.0.0"
}
```

### 7.2 参考文档

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [OWASP SQL Injection Prevention](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/instrumentation/go/)

### 7.3 变更记录

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|----------|------|
| v1.0 | 2026-03-24 | 初始版本 | Code Review Team |

---

## 8. 执行检查清单

### 启动前准备
- [ ] 团队评审确认方案
- [ ] 测试环境准备完成
- [ ] 监控平台搭建完成
- [ ] 代码分支策略确定

### 每阶段检查
- [ ] 代码Review通过
- [ ] 测试用例补充完成
- [ ] 文档更新完成
- [ ] 回归测试通过

### 上线前检查
- [ ] 安全扫描通过
- [ ] 性能测试通过
- [ ] 部署文档更新
- [ ] 回滚方案准备

---

**END OF DOCUMENT**
