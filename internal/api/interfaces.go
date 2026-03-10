package api

import (
	"context"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/scanner"
)

// SourceService 定义数据源服务接口
type SourceService interface {
	ListSources(ctx context.Context) ([]*model.SourceListItem, error)
	CreateSource(ctx context.Context, req *model.CreateSourceRequest) (*model.SourceResponse, error)
	GetSource(ctx context.Context, id string) (*model.SourceResponse, error)
	UpdateSource(ctx context.Context, id string, req *model.UpdateSourceRequest) error
	DeleteSource(ctx context.Context, id string) error
	TestConnection(ctx context.Context, dbType string, config scanner.ConnectionConfig) error
	TriggerSync(ctx context.Context, id string) error
}

// MetadataService 定义元数据服务接口
type MetadataService interface {
	GetSchemaTree(ctx context.Context, sourceID string) (*model.SchemaTreeResponse, error)
	ListSchemaChanges(ctx context.Context, sourceID string, limit int) ([]*model.SchemaChangeResponse, error)
	GetColumnDetail(ctx context.Context, columnID string) (*model.ColumnDetailResponse, error)
	SearchColumns(ctx context.Context, query string, limit int) ([]model.ColumnSearchResult, error)
	GetColumnMappings(ctx context.Context, columnID string) ([]*model.ColumnMappingResponse, error)
	CreateColumnMapping(ctx context.Context, req *model.ColumnMappingRequest) error
	DeleteColumnMapping(ctx context.Context, mappingID string) error
	GetLineage(ctx context.Context, columnID string) (*model.LineageResponse, error)
	GetImpactAnalysis(ctx context.Context, columnID string) (*model.ImpactAnalysisResponse, error)
}

// TermService 定义业务术语服务接口
type TermService interface {
	CreateTerm(ctx context.Context, req *model.BusinessTermRequest) (*model.BusinessTermResponse, error)
	ListTerms(ctx context.Context, category string) ([]*model.BusinessTermResponse, error)
	GetTerm(ctx context.Context, id string) (*model.BusinessTermResponse, error)
	UpdateTerm(ctx context.Context, id string, req *model.BusinessTermRequest) error
	DeleteTerm(ctx context.Context, id string) error
	AssignTermToColumn(ctx context.Context, columnID string, req *model.AssignTermRequest) error
}

// DDLService 定义DDL服务接口
type DDLService interface {
	GenerateDDL(ctx context.Context, objectID string, targetType string) (*model.DDLGenerateResponse, error)
}
