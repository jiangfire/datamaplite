package store

import (
	"context"
	"fmt"

	"git.neolidy.top/neo/fuckcmdb/internal/config"
	"go.uber.org/zap"
)

// Store 数据存储接口
type Store interface {
	// DataSource
	CreateDataSource(ctx context.Context, source *DataSourceCreate) error
	GetDataSource(ctx context.Context, id string) (*DataSourceRow, error)
	ListDataSources(ctx context.Context) ([]*DataSourceRow, error)
	UpdateDataSource(ctx context.Context, id string, updates *DataSourceUpdate) error
	DeleteDataSource(ctx context.Context, id string) error
	UpdateDataSourceSyncStatus(ctx context.Context, id string, status string, errorMsg *string) error

	// SchemaObject
	CreateSchemaObject(ctx context.Context, obj *SchemaObjectCreate) (string, error)
	GetSchemaObject(ctx context.Context, id string) (*SchemaObjectRow, error)
	GetSchemaObjectByName(ctx context.Context, sourceID string, name string, schema *string) (*SchemaObjectRow, error)
	ListSchemaObjectsBySource(ctx context.Context, sourceID string) ([]*SchemaObjectRow, error)
	DeleteSchemaObjectsBySource(ctx context.Context, sourceID string) error

	// Column
	CreateColumn(ctx context.Context, col *ColumnCreate) error
	GetColumn(ctx context.Context, id string) (*ColumnRow, error)
	ListColumnsByObject(ctx context.Context, objectID string) ([]*ColumnRow, error)
	SearchColumns(ctx context.Context, query string, limit int) ([]*ColumnSearchRow, error)
	DeleteColumnsByObject(ctx context.Context, objectID string) error

	// SchemaChange
	CreateSchemaChange(ctx context.Context, change *SchemaChangeCreate) error
	ListSchemaChangesBySource(ctx context.Context, sourceID string, limit int) ([]*SchemaChangeRow, error)

	// ColumnMapping 字段映射
	CreateColumnMapping(ctx context.Context, mapping *ColumnMappingCreate) error
	GetColumnMappings(ctx context.Context, columnID string) ([]*ColumnMappingRow, error)
	DeleteColumnMapping(ctx context.Context, id string) error

	// Lineage 血缘
	GetLineageUpward(ctx context.Context, columnID string, depth int) ([]*LineageEdgeRow, error)
	GetLineageDownward(ctx context.Context, columnID string, depth int) ([]*LineageEdgeRow, error)
	CreateLineageEdge(ctx context.Context, edge *LineageEdgeCreate) error

	// BusinessTerm 业务术语
	CreateBusinessTerm(ctx context.Context, term *BusinessTermCreate) (string, error)
	GetBusinessTerm(ctx context.Context, id string) (*BusinessTermRow, error)
	ListBusinessTerms(ctx context.Context, category string) ([]*BusinessTermRow, error)
	UpdateBusinessTerm(ctx context.Context, id string, updates *BusinessTermUpdate) error
	DeleteBusinessTerm(ctx context.Context, id string) error
	AssignTermToColumn(ctx context.Context, columnID string, termID *string) error

	// DDL 生成
	GetObjectWithColumns(ctx context.Context, objectID string) (*SchemaObjectRow, []*ColumnRow, error)

	// User 用户管理
	CreateUser(ctx context.Context, user *UserCreate) (string, error)
	GetUserByID(ctx context.Context, id string) (*UserRow, error)
	GetUserByUsername(ctx context.Context, username string) (*UserRow, error)
	ListUsers(ctx context.Context) ([]*UserRow, error)
	UpdateUser(ctx context.Context, id string, updates *UserUpdate) error
	DeleteUser(ctx context.Context, id string) error

	// Data Quality 数据质量
	CreateDQRule(ctx context.Context, rule *DQRuleCreate) (string, error)
	GetDQRule(ctx context.Context, id string) (*DQRuleRow, error)
	ListDQRules(ctx context.Context, filter *DQRuleFilter) ([]*DQRuleRow, error)
	UpdateDQRule(ctx context.Context, id string, updates *DQRuleUpdate) error
	DeleteDQRule(ctx context.Context, id string) error
	CreateDQResult(ctx context.Context, result *DQResultCreate) error
	ListDQResults(ctx context.Context, filter *DQResultFilter) ([]*DQResultRow, error)
	GetLatestDQResult(ctx context.Context, ruleID string) (*DQResultRow, error)
	GetDQStats(ctx context.Context) (*DQStatsRow, error)

	// Transaction
	WithTx(ctx context.Context, fn func(Store) error) error

	// Close 关闭连接
	Close() error
}

// DataSourceCreate 创建数据源参数
type DataSourceCreate struct {
	Name             string
	Description      *string
	Type             string
	Host             string
	Port             int
	Database         string
	ConnectionConfig string
}

// DataSourceRow 数据源行
type DataSourceRow struct {
	ID               string
	Name             string
	Description      *string
	Type             string
	Host             string
	Port             int
	Database         string
	ConnectionConfig string
	Status           string
	LastSyncAt       *string
	LastSyncError    *string
	CreatedAt        string
	UpdatedAt        string
}

// DataSourceUpdate 更新数据源参数
type DataSourceUpdate struct {
	Name             *string
	Description      *string
	Host             *string
	Port             *int
	Database         *string
	ConnectionConfig *string
	Status           *string
}

// SchemaObjectCreate 创建Schema对象参数
type SchemaObjectCreate struct {
	SourceID    string
	Name        string
	Type        string
	Schema      *string
	Description *string
	RowCount    *int64
	SizeBytes   *int64
	ColumnCount int
}

// SchemaObjectRow Schema对象行
type SchemaObjectRow struct {
	ID          string
	SourceID    string
	Name        string
	Type        string
	Schema      *string
	Description *string
	RowCount    *int64
	SizeBytes   *int64
	ColumnCount int
	CreatedAt   string
	UpdatedAt   string
}

// ColumnCreate 创建字段参数
type ColumnCreate struct {
	ObjectID         string
	Name             string
	DataType         string
	FullDataType     string
	IsNullable       bool
	DefaultValue     *string
	IsPrimaryKey     bool
	IsUnique         bool
	OrdinalPosition  int
	Description      *string
	ParentColumnPath *string
}

// ColumnRow 字段行
type ColumnRow struct {
	ID               string
	ObjectID         string
	Name             string
	DataType         string
	FullDataType     string
	IsNullable       bool
	DefaultValue     *string
	IsPrimaryKey     bool
	IsUnique         bool
	OrdinalPosition  int
	Description      *string
	TermID           *string
	Confidence       float64
	ParentColumnPath *string
	CreatedAt        string
	UpdatedAt        string
}

// ColumnSearchRow 字段搜索行
type ColumnSearchRow struct {
	ColumnRow
	ObjectName string
	SourceID   string
	SourceName string
	SourceType string
}

// SchemaChangeCreate 创建变更记录参数
type SchemaChangeCreate struct {
	SourceID   string
	ObjectID   *string
	ChangeType string
	ObjectType string
	ObjectName string
	OldValue   *string
	NewValue   *string
}

// SchemaChangeRow 变更记录行
type SchemaChangeRow struct {
	ID           string
	SourceID     string
	ObjectID     *string
	ChangeType   string
	ObjectType   string
	ObjectName   string
	OldValue     *string
	NewValue     *string
	DetectedAt   string
	Acknowledged bool
}

// ColumnMappingCreate 创建字段映射参数
type ColumnMappingCreate struct {
	SourceColumnID string
	TargetColumnID string
	MappingType    string
	Confidence     float64
}

// ColumnMappingRow 字段映射行
type ColumnMappingRow struct {
	ID               string
	SourceColumnID   string
	TargetColumnID   string
	MappingType      string
	Confidence       float64
	CreatedAt        string
	TargetColumnName string
	TargetObjectName string
	TargetSourceName string
}

// LineageEdgeCreate 创建血缘边参数
type LineageEdgeCreate struct {
	SourceID     string
	TargetID     string
	SourceType   string
	TargetType   string
	TransformSQL *string
	JobName      *string
}

// LineageEdgeRow 血缘边行
type LineageEdgeRow struct {
	ID             string
	SourceID       string
	TargetID       string
	SourceType     string
	TargetType     string
	TransformSQL   *string
	JobName        *string
	CreatedAt      string
	SourceName     string
	TargetName     string
	SourceDataType string
	TargetDataType string
}

// BusinessTermCreate 创建业务术语参数
type BusinessTermCreate struct {
	Name        string
	Description *string
	Category    string
}

// BusinessTermRow 业务术语行
type BusinessTermRow struct {
	ID          string
	Name        string
	Description *string
	Category    string
	CreatedAt   string
	UpdatedAt   string
}

// BusinessTermUpdate 更新业务术语参数
type BusinessTermUpdate struct {
	Name        *string
	Description *string
	Category    *string
}

// UserCreate 创建用户参数
type UserCreate struct {
	Username     string
	Email        string
	PasswordHash string
	Role         string
}

// UserRow 用户行
type UserRow struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    string
	UpdatedAt    string
}

// UserUpdate 更新用户参数
type UserUpdate struct {
	Username     *string
	Email        *string
	PasswordHash *string
	Role         *string
}

// DQRuleCreate 创建数据质量规则参数
type DQRuleCreate struct {
	SourceID    *string
	ObjectID    *string
	ColumnID    *string
	Name        string
	Description *string
	RuleType    string
	RuleConfig  string // JSON
	Severity    string
	IsActive    bool
}

// DQRuleRow 数据质量规则行
type DQRuleRow struct {
	ID          string
	SourceID    *string
	ObjectID    *string
	ColumnID    *string
	Name        string
	Description *string
	RuleType    string
	RuleConfig  string
	Severity    string
	IsActive    bool
	CreatedAt   string
	UpdatedAt   string
}

// DQRuleUpdate 更新数据质量规则参数
type DQRuleUpdate struct {
	Name        *string
	Description *string
	RuleConfig  *string
	Severity    *string
	IsActive    *bool
}

// DQRuleFilter 数据质量规则过滤条件
type DQRuleFilter struct {
	SourceID *string
	ObjectID *string
	ColumnID *string
	RuleType *string
	IsActive *bool
}

// DQResultCreate 创建数据质量结果参数
type DQResultCreate struct {
	RuleID       string
	CheckBatchID string
	ColumnID     *string
	Status       string
	TotalRows    int64
	FailedRows   int64
	PassRate     float64
	SampleErrors string // JSON
	ErrorMessage *string
}

// DQResultRow 数据质量结果行
type DQResultRow struct {
	ID            string
	RuleID        string
	CheckBatchID  string
	ColumnID      *string
	Status        string
	TotalRows     int64
	FailedRows    int64
	PassRate      float64
	SampleErrors  string
	ErrorMessage  *string
	CheckedAt     string
}

// DQResultFilter 数据质量结果过滤条件
type DQResultFilter struct {
	RuleID       *string
	BatchID      *string
	ColumnID     *string
	Status       *string
	Limit        int
}

// DQStatsRow 数据质量统计行
type DQStatsRow struct {
	TotalRules      int
	ActiveRules     int
	TotalChecks     int64
	PassedChecks    int64
	FailedChecks    int64
	OverallPassRate float64
}

// New 创建新的存储实例
func New(ctx context.Context, cfg *config.Config, log *zap.Logger) (Store, error) {
	// 根据配置选择数据库类型
	dbType := cfg.Database.Type
	if dbType == "" {
		// 默认使用 SQLite（方便本地开发测试）
		dbType = "sqlite"
	}

	switch dbType {
	case "postgres", "postgresql":
		return NewPostgresStore(ctx, &cfg.Database, log)
	case "sqlite":
		return NewSQLiteStore(ctx, &cfg.Database, log)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
