package model

import (
	"time"

	"github.com/google/uuid"
)

// DataSourceType 数据源类型
type DataSourceType string

const (
	DataSourceMySQL      DataSourceType = "mysql"
	DataSourcePostgreSQL DataSourceType = "postgres"
	DataSourceMongoDB    DataSourceType = "mongodb"
	DataSourceOracle     DataSourceType = "oracle"
	DataSourceMSSQL      DataSourceType = "mssql"
)

// ObjectType 对象类型
type ObjectType string

const (
	ObjectTypeTable      ObjectType = "table"
	ObjectTypeView       ObjectType = "view"
	ObjectTypeCollection ObjectType = "collection"
)

// DataSourceStatus 数据源状态
type DataSourceStatus string

const (
	DataSourceStatusActive   DataSourceStatus = "active"
	DataSourceStatusInactive DataSourceStatus = "inactive"
	DataSourceStatusError    DataSourceStatus = "error"
	DataSourceStatusSyncing  DataSourceStatus = "syncing"
)

// DataSource 数据源实体
type DataSource struct {
	ID               uuid.UUID        `db:"id"`
	Name             string           `db:"name"`
	Description      *string          `db:"description"`
	Type             DataSourceType   `db:"type"`
	Host             string           `db:"host"`
	Port             int              `db:"port"`
	Database         string           `db:"database"`
	ConnectionConfig string           `db:"connection_config"` // 加密的连接配置JSON
	Status           DataSourceStatus `db:"status"`
	LastSyncAt       *time.Time       `db:"last_sync_at"`
	LastSyncError    *string          `db:"last_sync_error"`
	CreatedAt        time.Time        `db:"created_at"`
	UpdatedAt        time.Time        `db:"updated_at"`
}

// BusinessTerm 业务术语
type BusinessTerm struct {
	ID               uuid.UUID `db:"id"`
	Name             string    `db:"name"`
	Description      *string   `db:"description"`
	Category         string    `db:"category"`
	StandardCode     *string   `db:"standard_code"`
	Domain           *string   `db:"domain"`
	DataTypeStandard *string   `db:"data_type_standard"`
	ValidationRule   *string   `db:"validation_rule"`
	Owner            *string   `db:"owner"`
	Status           string    `db:"status"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// SchemaObject Schema对象（表/视图/集合）
type SchemaObject struct {
	ID          uuid.UUID  `db:"id"`
	SourceID    uuid.UUID  `db:"source_id"`
	Name        string     `db:"name"`
	Type        ObjectType `db:"type"`
	Schema      *string    `db:"schema"` // MySQL/PostgreSQL schema
	Description *string    `db:"description"`
	RowCount    *int64     `db:"row_count"`
	SizeBytes   *int64     `db:"size_bytes"`
	ColumnCount int        `db:"column_count"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// Column 字段元数据
type Column struct {
	ID               uuid.UUID  `db:"id"`
	ObjectID         uuid.UUID  `db:"object_id"`
	Name             string     `db:"name"`
	DataType         string     `db:"data_type"`
	FullDataType     string     `db:"full_data_type"` // 如 varchar(255)
	IsNullable       bool       `db:"is_nullable"`
	DefaultValue     *string    `db:"default_value"`
	IsPrimaryKey     bool       `db:"is_primary_key"`
	IsUnique         bool       `db:"is_unique"`
	OrdinalPosition  int        `db:"ordinal_position"`
	Description      *string    `db:"description"`
	TermID           *uuid.UUID `db:"term_id"`
	Confidence       float64    `db:"confidence"`         // MongoDB推断置信度
	ParentColumnPath *string    `db:"parent_column_path"` // MongoDB嵌套字段路径
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
}

// ColumnMapping 字段映射
type ColumnMapping struct {
	ID          uuid.UUID `db:"id"`
	SourceColID uuid.UUID `db:"source_column_id"`
	TargetColID uuid.UUID `db:"target_column_id"`
	MappingType string    `db:"mapping_type"` // 如 alias, transform, derived
	Confidence  float64   `db:"confidence"`
	CreatedAt   time.Time `db:"created_at"`
}

// LineageEdge 血缘边
type LineageEdge struct {
	ID           uuid.UUID `db:"id"`
	SourceID     uuid.UUID `db:"source_id"` // 可以是column或object
	TargetID     uuid.UUID `db:"target_id"`
	SourceType   string    `db:"source_type"` // column或object
	TargetType   string    `db:"target_type"`
	TransformSQL *string   `db:"transform_sql"`
	JobName      *string   `db:"job_name"`
	CreatedAt    time.Time `db:"created_at"`
}

// SchemaChange Schema变更记录
type SchemaChange struct {
	ID           uuid.UUID  `db:"id"`
	SourceID     uuid.UUID  `db:"source_id"`
	ObjectID     *uuid.UUID `db:"object_id"`
	ChangeType   string     `db:"change_type"` // add_column, drop_column, alter_column
	ObjectType   string     `db:"object_type"` // column或object
	ObjectName   string     `db:"object_name"`
	OldValue     *string    `db:"old_value"`
	NewValue     *string    `db:"new_value"`
	DetectedAt   time.Time  `db:"detected_at"`
	Acknowledged bool       `db:"acknowledged"`
}

// ColumnWithObject 字段及其所属对象信息
type ColumnWithObject struct {
	Column
	SourceID   uuid.UUID `db:"source_id"`
	ObjectName string    `db:"object_name"`
	SourceName string    `db:"source_name"`
}

// ObjectWithColumns 对象及其字段列表
type ObjectWithColumns struct {
	SchemaObject
	Columns []Column
}
