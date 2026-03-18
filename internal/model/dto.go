package model

import (
	"time"
)

// BaseResponse 基础响应
type BaseResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ListResponse 列表响应
type ListResponse struct {
	Total int64       `json:"total"`
	Items interface{} `json:"items"`
}

// CreateSourceRequest 创建数据源请求
type CreateSourceRequest struct {
	Name        string         `json:"name" validate:"required,max=100"`
	Description string         `json:"description"`
	Type        DataSourceType `json:"type" validate:"required,oneof=mysql postgres mongodb"`
	Host        string         `json:"host" validate:"required,max=255"`
	Port        int            `json:"port" validate:"required,min=1,max=65535"`
	Database    string         `json:"database" validate:"required,max=255"`
	Username    string         `json:"username" validate:"required,max=100"`
	Password    string         `json:"password" validate:"required,max=255"`
	SSLMode     string         `json:"ssl_mode"`
}

// UpdateSourceRequest 更新数据源请求
type UpdateSourceRequest struct {
	Name        string `json:"name,omitempty" validate:"omitempty,max=100"`
	Description string `json:"description,omitempty"`
	Host        string `json:"host,omitempty" validate:"omitempty,max=255"`
	Port        int    `json:"port,omitempty" validate:"omitempty,min=1,max=65535"`
	Database    string `json:"database,omitempty" validate:"omitempty,max=255"`
	Username    string `json:"username,omitempty" validate:"omitempty,max=100"`
	Password    string `json:"password,omitempty" validate:"omitempty,max=255"`
}

// SourceResponse 数据源响应
type SourceResponse struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   *string          `json:"description"`
	Type          DataSourceType   `json:"type"`
	Host          string           `json:"host"`
	Port          int              `json:"port"`
	Database      string           `json:"database"`
	Status        DataSourceStatus `json:"status"`
	LastSyncAt    *string          `json:"last_sync_at"`
	LastSyncError *string          `json:"last_sync_error"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at"`
}

// SourceListItem 数据源列表项（精简）
type SourceListItem struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   *string          `json:"description"`
	Type          DataSourceType   `json:"type"`
	Host          string           `json:"host"`
	Port          int              `json:"port"`
	Database      string           `json:"database"`
	Status        DataSourceStatus `json:"status"`
	LastSyncAt    *string          `json:"last_sync_at"`
	LastSyncError *string          `json:"last_sync_error"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at"`
}

// ConnectionTestRequest 连接测试请求
type ConnectionTestRequest struct {
	Type     DataSourceType `json:"type" validate:"required,oneof=mysql postgres mongodb"`
	Host     string         `json:"host" validate:"required,max=255"`
	Port     int            `json:"port" validate:"required,min=1,max=65535"`
	Database string         `json:"database" validate:"required,max=255"`
	Username string         `json:"username" validate:"required,max=100"`
	Password string         `json:"password" validate:"required,max=255"`
}

// ConnectionTestResponse 连接测试结果
type ConnectionTestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SyncResponse 同步响应
type SyncResponse struct {
	SourceID     string    `json:"source_id"`
	StartedAt    time.Time `json:"started_at"`
	ObjectsCount int       `json:"objects_count"`
}

// SchemaObjectResponse Schema对象响应
type SchemaObjectResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        ObjectType `json:"type"`
	Schema      *string    `json:"schema"`
	Description *string    `json:"description"`
	RowCount    *int64     `json:"row_count"`
	SizeBytes   *int64     `json:"size_bytes"`
	ColumnCount int        `json:"column_count"`
}

// SchemaTreeResponse Schema树响应
type SchemaTreeResponse struct {
	SourceID string                    `json:"source_id"`
	Objects  []SchemaObjectWithColumns `json:"objects"`
}

// SchemaObjectWithColumns 带字段的对象
type SchemaObjectWithColumns struct {
	SchemaObjectResponse
	Columns []ColumnResponse `json:"columns"`
}

// ColumnResponse 字段响应
type ColumnResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	DataType        string  `json:"data_type"`
	FullDataType    string  `json:"full_data_type"`
	IsNullable      bool    `json:"is_nullable"`
	DefaultValue    *string `json:"default_value"`
	IsPrimaryKey    bool    `json:"is_primary_key"`
	OrdinalPosition int     `json:"ordinal_position"`
	Description     *string `json:"description"`
}

// ColumnDetailResponse 字段详情响应
type ColumnDetailResponse struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	DataType         string         `json:"data_type"`
	FullDataType     string         `json:"full_data_type"`
	IsNullable       bool           `json:"is_nullable"`
	DefaultValue     *string        `json:"default_value"`
	IsPrimaryKey     bool           `json:"is_primary_key"`
	OrdinalPosition  int            `json:"ordinal_position"`
	Description      *string        `json:"description"`
	Confidence       float64        `json:"confidence"`
	ParentColumnPath *string        `json:"parent_column_path,omitempty"`
	Object           ObjectSummary  `json:"object"`
	Source           SourceSummary  `json:"source"`
	Term             *TermSummary   `json:"term,omitempty"`
	MappedColumns    []MappedColumn `json:"mapped_columns,omitempty"`
}

// ObjectSummary 对象摘要
type ObjectSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// SourceSummary 数据源摘要
type SourceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// TermSummary 术语摘要
type TermSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MappedColumn 映射的字段
type MappedColumn struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ObjectName  string  `json:"object_name"`
	SourceName  string  `json:"source_name"`
	MappingType string  `json:"mapping_type"`
	Confidence  float64 `json:"confidence"`
}

// ColumnSearchResult 字段搜索结果
type ColumnSearchResult struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	DataType         string  `json:"data_type"`
	ObjectName       string  `json:"object_name"`
	SourceID         string  `json:"source_id"`
	SourceName       string  `json:"source_name"`
	SourceType       string  `json:"source_type"`
	Confidence       float64 `json:"confidence"`
	ParentColumnPath *string `json:"parent_column_path,omitempty"`
}

// BusinessTermRequest 业务术语请求
type BusinessTermRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}

// BusinessTermResponse 业务术语响应
type BusinessTermResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AssignTermRequest 分配术语请求
type AssignTermRequest struct {
	TermID *string `json:"term_id"`
}

// DDLGenerateRequest DDL生成请求
type DDLGenerateRequest struct {
	ObjectID   string `json:"object_id" validate:"required"`
	TargetType string `json:"target_type" validate:"required,oneof=mysql postgres"`
}

// DDLGenerateResponse DDL生成响应
type DDLGenerateResponse struct {
	ObjectID string `json:"object_id"`
	SQL      string `json:"sql"`
}

// SchemaChangeResponse 变更记录响应
type SchemaChangeResponse struct {
	ID           string  `json:"id"`
	ObjectID     *string `json:"object_id,omitempty"`
	ChangeType   string  `json:"change_type"`
	ObjectType   string  `json:"object_type"`
	ObjectName   string  `json:"object_name"`
	OldValue     *string `json:"old_value,omitempty"`
	NewValue     *string `json:"new_value,omitempty"`
	DetectedAt   string  `json:"detected_at"`
	Acknowledged bool    `json:"acknowledged"`
}

// SearchColumnsRequest 搜索字段请求
type SearchColumnsRequest struct {
	Query string `form:"q" validate:"required,min=1,max=100"`
	Limit int    `form:"limit" validate:"omitempty,min=1,max=100"`
}

// ColumnMappingRequest 创建字段映射请求
type ColumnMappingRequest struct {
	SourceColumnID string  `json:"source_column_id" validate:"required"`
	TargetColumnID string  `json:"target_column_id" validate:"required"`
	MappingType    string  `json:"mapping_type" validate:"required,oneof=alias transform derived synonym"`
	Confidence     float64 `json:"confidence" validate:"omitempty,min=0,max=1"`
}

// ColumnMappingResponse 字段映射响应
type ColumnMappingResponse struct {
	ID             string  `json:"id"`
	SourceColumnID string  `json:"source_column_id"`
	TargetColumnID string  `json:"target_column_id"`
	MappingType    string  `json:"mapping_type"`
	Confidence     float64 `json:"confidence"`
	CreatedAt      string  `json:"created_at"`
	// 目标字段详情
	TargetColumn ColumnSummary `json:"target_column"`
}

// ColumnSummary 字段摘要
type ColumnSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	DataType   string `json:"data_type"`
	ObjectName string `json:"object_name"`
	SourceName string `json:"source_name"`
}

// LineageNode 血缘节点
type LineageNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // column | object
	DataType string `json:"data_type,omitempty"`
	Source   string `json:"source"`
}

// LineageEdgeResponse 血缘边响应
type LineageEdgeResponse struct {
	Source       LineageNode `json:"source"`
	Target       LineageNode `json:"target"`
	TransformSQL *string     `json:"transform_sql,omitempty"`
	JobName      *string     `json:"job_name,omitempty"`
}

// LineageResponse 血缘响应
type LineageResponse struct {
	ColumnID string                `json:"column_id"`
	Upward   []LineageEdgeResponse `json:"upward"`   // 上游血缘
	Downward []LineageEdgeResponse `json:"downward"` // 下游血缘
}

// ImpactAnalysisResponse 影响分析响应
type ImpactAnalysisResponse struct {
	ColumnID      string         `json:"column_id"`
	ImpactObjects []ImpactObject `json:"impact_objects"`
	TotalObjects  int            `json:"total_objects"`
}

// ImpactObject 影响对象
type ImpactObject struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"` // object | column
	ObjectName string `json:"object_name,omitempty"`
	SourceName string `json:"source_name"`
	ImpactPath string `json:"impact_path"`
	Distance   int    `json:"distance"`
}

// PaginationQuery 分页查询
type PaginationQuery struct {
	Page     int `query:"page"`
	PageSize int `query:"page_size"`
}

// GetOffset 获取偏移量
func (p PaginationQuery) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	return (p.Page - 1) * p.GetLimit()
}

// GetLimit 获取限制数
func (p PaginationQuery) GetLimit() int {
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p.PageSize
}

// ============ Tag Types ============

// TagRequest 创建/更新标签请求
type TagRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Color       string `json:"color" validate:"required,hexcolor"`
	Description string `json:"description,omitempty"`
}

// TagResponse 标签响应
type TagResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ColumnTagRequest 添加标签到字段请求
type ColumnTagRequest struct {
	TagID string `json:"tag_id" validate:"required"`
}

// ColumnWithTagsResponse 带标签的字段响应
type ColumnWithTagsResponse struct {
	ColumnDetailResponse
	Tags []TagResponse `json:"tags"`
}
