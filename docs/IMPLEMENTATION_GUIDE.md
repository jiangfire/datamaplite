# DataMap-Lite 整改实施技术指南

**配套文档**: [REMEDIATION_PLAN.md](./REMEDIATION_PLAN.md) | [REMEDIATION_TASKS.md](./REMEDIATION_TASKS.md)

---

## 1. SQL注入防护实现指南

### 1.1 当前风险点

```go
// service/dq_executor.go - 存在注入风险
func (s *DQService) buildRuleQueries(...) {
    // 直接拼接用户输入的SQL
    customSQL := ruleConfig["sql"].(string)  // 用户输入
    countSQL = fmt.Sprintf("SELECT COUNT(*) FROM (%s) t", customSQL)
}
```

### 1.2 防护方案

```go
// internal/sqlparser/validator.go
package sqlparser

import (
    "fmt"
    "strings"
    
    "github.com/xwb1989/sqlparser"
)

type SQLValidator struct {
    allowedTables map[string]bool // 允许的表白名单
    maxRows       int             // 最大返回行数限制
}

func NewSQLValidator() *SQLValidator {
    return &SQLValidator{
        maxRows: 10000,
    }
}

// ValidateSelectSQL 验证只允许SELECT语句
func (v *SQLValidator) ValidateSelectSQL(sql string) error {
    // 解析SQL
    stmt, err := sqlparser.Parse(sql)
    if err != nil {
        return fmt.Errorf("sql parse error: %w", err)
    }
    
    // 只允许SELECT
    _, ok := stmt.(*sqlparser.Select)
    if !ok {
        return fmt.Errorf("only SELECT statements are allowed")
    }
    
    // 检查危险函数
    if err := v.checkDangerousFunctions(stmt); err != nil {
        return err
    }
    
    // 检查表权限
    if err := v.checkTablePermissions(stmt); err != nil {
        return err
    }
    
    return nil
}

// AddLimit 自动添加LIMIT限制
func (v *SQLValidator) AddLimit(sql string, limit int) string {
    // 使用sqlparser改写SQL添加LIMIT
    stmt, _ := sqlparser.Parse(sql)
    if sel, ok := stmt.(*sqlparser.Select); ok {
        sel.Limit = &sqlparser.Limit{Rowcount: sqlparser.NewIntVal([]byte(fmt.Sprintf("%d", limit)))}
        return sqlparser.String(sel)
    }
    return sql
}

func (v *SQLValidator) checkDangerousFunctions(stmt sqlparser.Statement) error {
    // 检查是否包含危险函数：xp_cmdshell, system等
    dangerous := []string{"xp_cmdshell", "system", "exec", "eval"}
    sqlStr := strings.ToLower(sqlparser.String(stmt))
    for _, fn := range dangerous {
        if strings.Contains(sqlStr, fn) {
            return fmt.Errorf("dangerous function detected: %s", fn)
        }
    }
    return nil
}
```

### 1.3 集成到DQ服务

```go
// service/dq.go

type DQService struct {
    // ...
    sqlValidator *sqlparser.SQLValidator
}

func (s *DQService) executeRule(ctx context.Context, rule *store.DQRuleRow, batchID string, sampleLimit int) (*model.DQResult, error) {
    // ...
    
    if rule.RuleType == string(model.DQRuleTypeCustomSQL) {
        rawSQL := ruleConfig["sql"].(string)
        
        // 验证SQL
        if err := s.sqlValidator.ValidateSelectSQL(rawSQL); err != nil {
            return nil, fmt.Errorf("invalid custom sql: %w", err)
        }
        
        // 添加LIMIT
        safeSQL := s.sqlValidator.AddLimit(rawSQL, 10000)
        
        // 审计日志
        s.logAudit(ctx, rule.ID, safeSQL)
        
        // 执行...
    }
}
```

---

## 2. 测试覆盖率提升指南

### 2.0 2026-03-28 已落地的高价值测试补强

本轮重点不是“把覆盖率数字做高”，而是补能直接打出真实缺陷的测试。当前已经落地并验证的方向：

| 方向 | 已覆盖边界 | 结果 |
|------|------------|------|
| `saveSchema` | 对象/字段增删改、对象和字段 ID 稳定性、血缘清理、默认值与指针等价比较 | 已发现并修复伪造 `alter_column` 与审计串错问题 |
| `TriggerSync` | 成功、扫描失败、落库失败、同源并发拒绝、多实例共享同库互斥、早期解密失败后的恢复 | 已发现并修复同步互斥与恢复路径问题 |
| `AlertService` | webhook 成功、4xx 不重试、5xx 重试、退避期间取消、重复处理去重、稳定幂等键、对象级删除告警 | 已发现并修复状态持久化、主键生成、SQLite 卡死与重复告警问题 |
| 治理 outbox | 持久化入队、最终一致性补投递、重复 `event_id` 去重、多 dispatcher 租约互斥、死信、重放、统计 | 已补齐治理事件可靠投递主链路和运维最小闭环 |
| MCP / HTTP 审计链 | HTTP / MCP 写操作到治理事件，再到结构变更事件 trace 透传 | 已验证链路可追踪 |
| 同步租约治理 | 多实例互斥、陈旧租约释放、SQLite 时间格式兼容 | 已发现并修复 SQLite `updated_at` 解析缺陷 |

建议后续继续按这种方式补测试：

- 优先选“失败会造成错误数据、错误审计、错误治理动作”的路径
- 优先用真实 SQLite / `httptest.Server` 跑集成测试，而不是纯 mock
- 每补一组测试，都要求能回答“它能打出什么真实问题”

### 2.1 Store层测试框架

```go
// store/postgres_test.go
package store

import (
    "context"
    "testing"
    
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

func setupTestPostgres(t *testing.T) (*PostgresStore, func()) {
    ctx := context.Background()
    
    // 启动PostgreSQL容器
    container, err := postgres.RunContainer(ctx,
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections"),
        ),
    )
    if err != nil {
        t.Fatalf("failed to start container: %v", err)
    }
    
    // 获取连接信息
    connStr, _ := container.ConnectionString(ctx)
    
    // 创建Store实例
    cfg := &config.DatabaseConfig{
        Type: "postgres",
        DSN:  connStr,
    }
    
    store, err := NewPostgresStore(ctx, cfg, zap.NewNop())
    if err != nil {
        t.Fatalf("failed to create store: %v", err)
    }
    
    cleanup := func() {
        store.Close()
        container.Terminate(ctx)
    }
    
    return store, cleanup
}

func TestPostgresStore_DataSource(t *testing.T) {
    store, cleanup := setupTestPostgres(t)
    defer cleanup()
    
    ctx := context.Background()
    
    t.Run("Create and Get", func(t *testing.T) {
        create := &DataSourceCreate{
            Name: "test-source",
            Type: "mysql",
            Host: "localhost",
            Port: 3306,
            Database: "test",
        }
        
        id, err := store.CreateDataSource(ctx, create)
        if err != nil {
            t.Fatalf("CreateDataSource failed: %v", err)
        }
        
        source, err := store.GetDataSource(ctx, id)
        if err != nil {
            t.Fatalf("GetDataSource failed: %v", err)
        }
        
        if source.Name != create.Name {
            t.Errorf("expected name %s, got %s", create.Name, source.Name)
        }
    })
    
    // 更多测试用例...
}
```

### 2.2 Service层Mock测试

```go
// generate mock: go generate ./...
//go:generate mockgen -source=store.go -destination=mock_store.go -package=store

// service/dq_test.go
package service

import (
    "testing"
    
    "github.com/jiangfire/datamaplite/internal/model"
    "github.com/jiangfire/datamaplite/internal/store"
    "github.com/golang/mock/gomock"
)

func TestDQService_CreateRule(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockStore := store.NewMockStore(ctrl)
    cipher, _ := crypto.NewCipher([]byte("test-key-12345678"))
    
    service := NewDQService(mockStore, cipher)
    
    tests := []struct {
        name    string
        req     *model.DQRuleRequest
        mock    func()
        wantErr bool
    }{
        {
            name: "success - not_null rule",
            req: &model.DQRuleRequest{
                Name:     "test-rule",
                RuleType: model.DQRuleTypeNotNull,
                Severity: model.DQSeverityError,
            },
            mock: func() {
                mockStore.EXPECT().
                    CreateDQRule(gomock.Any(), gomock.Any()).
                    Return("rule-123", nil)
                mockStore.EXPECT().
                    GetDQRule(gomock.Any(), "rule-123").
                    Return(&store.DQRuleRow{
                        ID:       "rule-123",
                        Name:     "test-rule",
                        RuleType: "not_null",
                    }, nil)
            },
            wantErr: false,
        },
        {
            name: "invalid regex rule - missing pattern",
            req: &model.DQRuleRequest{
                Name:     "test-regex",
                RuleType: model.DQRuleTypeRegex,
                RuleConfig: map[string]interface{}{},
            },
            mock:    func() {},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.mock()
            
            _, err := service.CreateRule(context.Background(), tt.req)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateRule() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## 3. Oracle/MSSQL扫描器实现

### 3.1 Oracle扫描器

```go
// scanner/oracle.go
package scanner

import (
    "context"
    "database/sql"
    "fmt"
    
    _ "github.com/godror/godror" // Oracle驱动
)

type OracleScanner struct{}

func NewOracleScanner() *OracleScanner {
    return &OracleScanner{}
}

func (s *OracleScanner) TestConnection(ctx context.Context, config ConnectionConfig) error {
    dsn := s.buildDSN(config)
    db, err := sql.Open("godror", dsn)
    if err != nil {
        return err
    }
    defer db.Close()
    
    return db.PingContext(ctx)
}

func (s *OracleScanner) ScanSchema(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error) {
    dsn := s.buildDSN(config)
    db, err := sql.Open("godror", dsn)
    if err != nil {
        return nil, err
    }
    defer db.Close()
    
    // 查询表信息
    tablesQuery := `
        SELECT owner, table_name, num_rows
        FROM all_tables
        WHERE owner = :1
        ORDER BY table_name
    `
    
    rows, err := db.QueryContext(ctx, tablesQuery, config.Database)
    if err != nil {
        return nil, fmt.Errorf("failed to query tables: %w", err)
    }
    defer rows.Close()
    
    var objects []ObjectInfo
    for rows.Next() {
        var owner, tableName string
        var numRows sql.NullInt64
        if err := rows.Scan(&owner, &tableName, &numRows); err != nil {
            return nil, err
        }
        
        // 查询字段
        columns, err := s.scanColumns(ctx, db, owner, tableName)
        if err != nil {
            return nil, err
        }
        
        var rowCount *int64
        if numRows.Valid {
            rc := numRows.Int64
            rowCount = &rc
        }
        
        objects = append(objects, ObjectInfo{
            Name:     tableName,
            Type:     "table",
            Schema:   &owner,
            RowCount: rowCount,
            Columns:  columns,
        })
    }
    
    return &SchemaInfo{Objects: objects}, nil
}

func (s *OracleScanner) scanColumns(ctx context.Context, db *sql.DB, owner, tableName string) ([]ColumnInfo, error) {
    query := `
        SELECT column_name, data_type, data_length, nullable,
               data_default, column_id
        FROM all_tab_columns
        WHERE owner = :1 AND table_name = :2
        ORDER BY column_id
    `
    
    rows, err := db.QueryContext(ctx, query, owner, tableName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var columns []ColumnInfo
    for rows.Next() {
        var col ColumnInfo
        var nullable string
        var defaultValue sql.NullString
        
        err := rows.Scan(
            &col.Name,
            &col.DataType,
            &col.FullDataType,
            &nullable,
            &defaultValue,
            &col.OrdinalPosition,
        )
        if err != nil {
            return nil, err
        }
        
        col.IsNullable = nullable == "Y"
        if defaultValue.Valid {
            col.DefaultValue = &defaultValue.String
        }
        
        columns = append(columns, col)
    }
    
    return columns, nil
}

func (s *OracleScanner) buildDSN(config ConnectionConfig) string {
    // Oracle EZ Connect格式
    return fmt.Sprintf("%s/%s@%s:%d/%s",
        config.Username,
        config.Password,
        config.Host,
        config.Port,
        config.Database,
    )
}
```

### 3.2 MSSQL扫描器

```go
// scanner/mssql.go
package scanner

import (
    "context"
    "database/sql"
    "fmt"
    
    _ "github.com/microsoft/go-mssqldb"
)

type MSSQLScanner struct{}

func NewMSSQLScanner() *MSSQLScanner {
    return &MSSQLScanner{}
}

func (s *MSSQLScanner) TestConnection(ctx context.Context, config ConnectionConfig) error {
    dsn := s.buildDSN(config)
    db, err := sql.Open("sqlserver", dsn)
    if err != nil {
        return err
    }
    defer db.Close()
    
    return db.PingContext(ctx)
}

func (s *MSSQLScanner) ScanSchema(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error) {
    dsn := s.buildDSN(config)
    db, err := sql.Open("sqlserver", dsn)
    if err != nil {
        return nil, err
    }
    defer db.Close()
    
    // 使用INFORMATION_SCHEMA
    query := `
        SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE
        FROM INFORMATION_SCHEMA.TABLES
        WHERE TABLE_TYPE IN ('BASE TABLE', 'VIEW')
        ORDER BY TABLE_SCHEMA, TABLE_NAME
    `
    
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var objects []ObjectInfo
    for rows.Next() {
        var schema, name, tableType string
        if err := rows.Scan(&schema, &name, &tableType); err != nil {
            return nil, err
        }
        
        objType := "table"
        if tableType == "VIEW" {
            objType = "view"
        }
        
        columns, err := s.scanColumns(ctx, db, schema, name)
        if err != nil {
            return nil, err
        }
        
        objects = append(objects, ObjectInfo{
            Name:    name,
            Type:    objType,
            Schema:  &schema,
            Columns: columns,
        })
    }
    
    return &SchemaInfo{Objects: objects}, nil
}

func (s *MSSQLScanner) buildDSN(config ConnectionConfig) string {
    return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
        config.Username,
        config.Password,
        config.Host,
        config.Port,
        config.Database,
    )
}
```

---

## 4. 定时同步机制实现

### 4.1 调度引擎

```go
// service/sync_scheduler.go
package service

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/robfig/cron/v3"
    "go.uber.org/zap"
)

type SyncJob struct {
    ID         string
    SourceID   string
    CronExpr   string
    LastSyncAt *time.Time
    NextSyncAt *time.Time
    Status     string // active, paused, error
}

type SyncScheduler struct {
    cron     *cron.Cron
    store    store.Store
    registry *scanner.Registry
    cipher   *crypto.Cipher
    logger   *zap.Logger
    
    jobs     map[string]cron.EntryID
    jobsMu   sync.RWMutex
}

func NewSyncScheduler(store store.Store, registry *scanner.Registry, cipher *crypto.Cipher, logger *zap.Logger) *SyncScheduler {
    return &SyncScheduler{
        cron:     cron.New(cron.WithSeconds()),
        store:    store,
        registry: registry,
        cipher:   cipher,
        logger:   logger,
        jobs:     make(map[string]cron.EntryID),
    }
}

func (s *SyncScheduler) Start() {
    s.cron.Start()
    
    // 加载已有任务
    ctx := context.Background()
    jobs, _ := s.store.ListSyncJobs(ctx)
    for _, job := range jobs {
        if job.Status == "active" {
            s.AddJob(ctx, job)
        }
    }
}

func (s *SyncScheduler) Stop() {
    ctx := s.cron.Stop()
    <-ctx.Done()
}

func (s *SyncScheduler) AddJob(ctx context.Context, job *SyncJob) error {
    s.jobsMu.Lock()
    defer s.jobsMu.Unlock()
    
    // 如果已存在，先移除
    if entryID, exists := s.jobs[job.ID]; exists {
        s.cron.Remove(entryID)
    }
    
    entryID, err := s.cron.AddFunc(job.CronExpr, func() {
        s.executeSync(job.SourceID)
    })
    if err != nil {
        return fmt.Errorf("invalid cron expression: %w", err)
    }
    
    s.jobs[job.ID] = entryID
    
    // 更新下次执行时间
    entry := s.cron.Entry(entryID)
    nextRun := entry.Next
    s.store.UpdateSyncJobNextRun(ctx, job.ID, &nextRun)
    
    return nil
}

func (s *SyncScheduler) executeSync(sourceID string) {
    ctx := context.Background()
    
    s.logger.Info("starting scheduled sync", zap.String("source_id", sourceID))
    
    // 获取数据源
    src, err := s.store.GetDataSource(ctx, sourceID)
    if err != nil {
        s.logger.Error("failed to get source", zap.Error(err))
        s.recordSyncError(sourceID, err)
        return
    }
    
    // 解密连接配置
    configJSON, err := s.cipher.Decrypt(src.ConnectionConfig)
    if err != nil {
        s.logger.Error("failed to decrypt config", zap.Error(err))
        s.recordSyncError(sourceID, err)
        return
    }
    
    connConfig, err := scanner.ConnectionConfigFromJSON(configJSON)
    if err != nil {
        s.recordSyncError(sourceID, err)
        return
    }
    
    // 获取扫描器
    sc, err := s.registry.Get(src.Type)
    if err != nil {
        s.recordSyncError(sourceID, err)
        return
    }
    
    // 执行同步（带重试）
    var syncErr error
    for i := 0; i < 3; i++ {
        schemaInfo, syncErr := sc.ScanSchema(ctx, *connConfig)
        if syncErr == nil {
            // 保存结果
            if err := s.saveSchema(ctx, sourceID, schemaInfo); err != nil {
                syncErr = err
            }
            break
        }
        s.logger.Warn("sync attempt failed, retrying", 
            zap.Int("attempt", i+1), 
            zap.Error(syncErr))
        time.Sleep(time.Second * time.Duration(i+1))
    }
    
    if syncErr != nil {
        s.recordSyncError(sourceID, syncErr)
    } else {
        s.recordSyncSuccess(sourceID)
    }
}

func (s *SyncScheduler) recordSyncError(sourceID string, err error) {
    errMsg := err.Error()
    s.store.UpdateDataSourceSyncStatus(context.Background(), sourceID, "error", &errMsg)
    // 发送告警通知...
}

func (s *SyncScheduler) recordSyncSuccess(sourceID string) {
    s.store.UpdateDataSourceSyncStatus(context.Background(), sourceID, "active", nil)
}
```

---

## 5. Prometheus 监控集成

```go
// metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // HTTP指标
    HTTPRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "datamap_http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "path", "status"},
    )
    
    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "datamap_http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
    
    // 数据库指标
    DBQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "datamap_db_query_duration_seconds",
            Help:    "Database query duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation", "table"},
    )
    
    // 业务指标
    SyncDuration = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "datamap_sync_duration_seconds",
            Help: "Schema sync duration",
        },
        []string{"source_id", "status"},
    )
    
    DQCheckTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "datamap_dq_checks_total",
            Help: "Total DQ checks",
        },
        []string{"status"},
    )
)

// Gin中间件
func PrometheusMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())
        
        HTTPRequestsTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            status,
        ).Inc()
        
        HTTPRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration)
    }
}
```

---

## 6. 前端 React Query 集成

```typescript
// web/src/hooks/useSourcesQuery.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sourceService } from '../services';
import type { DataSource, DataSourceCreate } from '../types';

const SOURCES_KEY = 'sources';

export const useSourcesQuery = () => {
  return useQuery({
    queryKey: [SOURCES_KEY],
    queryFn: () => sourceService.listSources(),
    staleTime: 5 * 60 * 1000, // 5分钟
  });
};

export const useCreateSourceMutation = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: DataSourceCreate) => sourceService.createSource(data),
    onSuccess: () => {
      // 成功后刷新列表
      queryClient.invalidateQueries({ queryKey: [SOURCES_KEY] });
    },
  });
};

export const useSourceQuery = (id: string | undefined) => {
  return useQuery({
    queryKey: [SOURCES_KEY, id],
    queryFn: () => sourceService.getSource(id!),
    enabled: !!id,
  });
};
```

```typescript
// web/src/main.tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 3,
      refetchOnWindowFocus: false,
    },
  },
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={queryClient}>
    <App />
  </QueryClientProvider>,
);
```

---

## 7. 快速开始检查清单

### 开发环境准备

```bash
# 1. 安装工具
go install github.com/golang/mock/mockgen@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 2. 前端依赖
cd web
npm install @tanstack/react-query zustand echarts @antv/g6 react-error-boundary

# 3. 启动测试依赖
docker run -d --name postgres-test -e POSTGRES_PASSWORD=test -p 5432:5432 postgres:15
docker run -d --name oracle-test -p 1521:1521 gvenzl/oracle-xe:21-slim
docker run -d --name mssql-test -e 'ACCEPT_EULA=Y' -e 'SA_PASSWORD=Test@123' -p 1433:1433 mcr.microsoft.com/mssql/server:2022-latest
```

### 代码生成

```bash
# 生成Mock
go generate ./internal/store/...

# 运行测试
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 安全扫描
gosec -fmt sarif -out security.sarif ./...

# 代码检查
golangci-lint run
```

---

**文档更新历史**

| 版本 | 日期 | 更新内容 | 更新人 |
|------|------|----------|--------|
| v1.0 | 2026-03-24 | 初始版本 | Code Review Team |
| v1.1 | 2026-03-27 | 补充真实缺陷导向测试策略与本轮落地项 | Codex |

