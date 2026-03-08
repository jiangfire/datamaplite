package scanner

import (
	"context"
	"encoding/json"
	"fmt"
)

// SchemaInfo Schema扫描结果
type SchemaInfo struct {
	Objects []ObjectInfo `json:"objects"`
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Name        string       `json:"name"`
	Type        string       `json:"type"` // table, view, collection
	Schema      *string      `json:"schema,omitempty"`
	Description *string      `json:"description,omitempty"`
	RowCount    *int64       `json:"row_count,omitempty"`
	SizeBytes   *int64       `json:"size_bytes,omitempty"`
	Columns     []ColumnInfo `json:"columns"`
}

// ColumnInfo 字段信息
type ColumnInfo struct {
	Name             string  `json:"name"`
	DataType         string  `json:"data_type"`
	FullDataType     string  `json:"full_data_type,omitempty"`
	IsNullable       bool    `json:"is_nullable"`
	DefaultValue     *string `json:"default_value,omitempty"`
	IsPrimaryKey     bool    `json:"is_primary_key,omitempty"`
	IsUnique         bool    `json:"is_unique,omitempty"`
	OrdinalPosition  int     `json:"ordinal_position"`
	Description      *string `json:"description,omitempty"`
	ParentColumnPath *string `json:"parent_column_path,omitempty"`
	Confidence       float64 `json:"confidence,omitempty"` // MongoDB抽样置信度
}

// ConnectionConfig 连接配置
type ConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode,omitempty"`
}

// ToJSON 转换为JSON字符串
func (c ConnectionConfig) ToJSON() (string, error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal connection config: %w", err)
	}
	return string(bytes), nil
}

// ConnectionConfigFromJSON 从JSON字符串解析
func ConnectionConfigFromJSON(s string) (*ConnectionConfig, error) {
	var c ConnectionConfig
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connection config: %w", err)
	}
	return &c, nil
}

// SchemaScanner Schema扫描器接口
type SchemaScanner interface {
	// TestConnection 测试连接
	TestConnection(ctx context.Context, config ConnectionConfig) error
	// ScanSchema 扫描Schema
	ScanSchema(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error)
}

// Registry 扫描器注册表
type Registry struct {
	scanners map[string]SchemaScanner
}

// NewRegistry 创建新的注册表
func NewRegistry() *Registry {
	return &Registry{
		scanners: make(map[string]SchemaScanner),
	}
}

// Register 注册扫描器
func (r *Registry) Register(dbType string, scanner SchemaScanner) {
	r.scanners[dbType] = scanner
}

// Get 获取扫描器
func (r *Registry) Get(dbType string) (SchemaScanner, error) {
	scanner, ok := r.scanners[dbType]
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
	return scanner, nil
}
