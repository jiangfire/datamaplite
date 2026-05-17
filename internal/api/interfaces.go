package api

import (
	"context"

	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/scanner"
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

// AuthService 定义认证服务接口
type AuthService interface {
	IsEnabled() bool
	Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*model.LoginResponse, error)
	Register(ctx context.Context, req *model.RegisterRequest) (*model.UserInfo, error)
	GetUserByID(ctx context.Context, id string) (*model.UserInfo, error)
	ListUsers(ctx context.Context) ([]*model.UserInfo, error)
	UpdateUser(ctx context.Context, id string, req *model.UpdateUserRequest) (*model.UserInfo, error)
	DeleteUser(ctx context.Context, id string) error
}

// AlertService 定义告警服务接口
type AlertService interface {
	CreateAlertRule(ctx context.Context, req *model.AlertRuleRequest) (*model.AlertRuleResponse, error)
	GetAlertRule(ctx context.Context, id string) (*model.AlertRuleResponse, error)
	ListAlertRules(ctx context.Context, sourceID *string) ([]*model.AlertRuleResponse, error)
	UpdateAlertRule(ctx context.Context, id string, req *model.AlertRuleRequest) error
	DeleteAlertRule(ctx context.Context, id string) error
}

// NotificationService 定义通知服务接口
type NotificationService interface {
	ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*model.NotificationResponse, error)
	GetNotificationStats(ctx context.Context, userID string) (*model.NotificationStats, error)
	MarkAsRead(ctx context.Context, userID string, id string) error
	MarkAllAsRead(ctx context.Context, userID string) error
}

// DQService 定义数据质量服务接口
type DQService interface {
	CreateRule(ctx context.Context, req *model.DQRuleRequest) (*model.DQRule, error)
	ListRules(ctx context.Context, filter *model.DQRuleFilter) ([]*model.DQRule, error)
	GetRule(ctx context.Context, id string) (*model.DQRule, error)
	UpdateRule(ctx context.Context, id string, req *model.DQRuleRequest) error
	DeleteRule(ctx context.Context, id string) error
	CheckRules(ctx context.Context, req *model.DQCheckRequest) (*model.DQCheckResponse, error)
	GetResults(ctx context.Context, ruleID *string, batchID *string, limit int) ([]*model.DQResult, error)
	GetStats(ctx context.Context) (*model.DQStats, error)
}

// TagService 定义标签服务接口
type TagService interface {
	ListTags(ctx context.Context) ([]*model.TagResponse, error)
	CreateTag(ctx context.Context, req *model.TagRequest) (*model.TagResponse, error)
	GetTag(ctx context.Context, id string) (*model.TagResponse, error)
	UpdateTag(ctx context.Context, id string, req *model.TagRequest) error
	DeleteTag(ctx context.Context, id string) error
	GetColumnsByTag(ctx context.Context, tagID string) ([]*model.ColumnSearchResult, error)
	AddTagToColumn(ctx context.Context, columnID string, tagID string) error
	RemoveTagFromColumn(ctx context.Context, columnID string, tagID string) error
	GetColumnTags(ctx context.Context, columnID string) ([]*model.TagResponse, error)
}
