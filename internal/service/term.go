package service

import (
	"context"
	"fmt"

	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/store"
)

// TermService 业务术语服务
type TermService struct {
	store store.Store
}

// NewTermService 创建业务术语服务
func NewTermService(store store.Store) *TermService {
	return &TermService{store: store}
}

// CreateTerm 创建业务术语
func (s *TermService) CreateTerm(ctx context.Context, req *model.BusinessTermRequest) (*model.BusinessTermResponse, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}

	term := &store.BusinessTermCreate{
		Name:             req.Name,
		Description:      strPtrOrNil(req.Description),
		Category:         req.Category,
		StandardCode:     strPtrOrNil(req.StandardCode),
		Domain:           strPtrOrNil(req.Domain),
		DataTypeStandard: strPtrOrNil(req.DataTypeStandard),
		ValidationRule:   strPtrOrNil(req.ValidationRule),
		Owner:            strPtrOrNil(req.Owner),
		Status:           strPtrOrNil(status),
	}

	id, err := s.store.CreateBusinessTerm(ctx, term)
	if err != nil {
		return nil, fmt.Errorf("failed to create term: %w", err)
	}

	return s.GetTerm(ctx, id)
}

// GetTerm 获取业务术语
func (s *TermService) GetTerm(ctx context.Context, id string) (*model.BusinessTermResponse, error) {
	term, err := s.store.GetBusinessTerm(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toTermResponse(term), nil
}

// ListTerms 列出业务术语
func (s *TermService) ListTerms(ctx context.Context, category string) ([]*model.BusinessTermResponse, error) {
	terms, err := s.store.ListBusinessTerms(ctx, category)
	if err != nil {
		return nil, err
	}

	var resp []*model.BusinessTermResponse
	for _, t := range terms {
		resp = append(resp, s.toTermResponse(t))
	}
	return resp, nil
}

// UpdateTerm 更新业务术语
func (s *TermService) UpdateTerm(ctx context.Context, id string, req *model.BusinessTermRequest) error {
	updates := &store.BusinessTermUpdate{
		Name:             strPtrOrNil(req.Name),
		Description:      strPtrOrNil(req.Description),
		Category:         strPtrOrNil(req.Category),
		StandardCode:     strPtrOrNil(req.StandardCode),
		Domain:           strPtrOrNil(req.Domain),
		DataTypeStandard: strPtrOrNil(req.DataTypeStandard),
		ValidationRule:   strPtrOrNil(req.ValidationRule),
		Owner:            strPtrOrNil(req.Owner),
		Status:           strPtrOrNil(req.Status),
	}
	return s.store.UpdateBusinessTerm(ctx, id, updates)
}

// DeleteTerm 删除业务术语
func (s *TermService) DeleteTerm(ctx context.Context, id string) error {
	return s.store.DeleteBusinessTerm(ctx, id)
}

// AssignTermToColumn 分配术语到字段
func (s *TermService) AssignTermToColumn(ctx context.Context, columnID string, req *model.AssignTermRequest) error {
	return s.store.AssignTermToColumn(ctx, columnID, req.TermID)
}

func (s *TermService) toTermResponse(t *store.BusinessTermRow) *model.BusinessTermResponse {
	return &model.BusinessTermResponse{
		ID:               t.ID,
		Name:             t.Name,
		Description:      strOrEmpty(t.Description),
		Category:         t.Category,
		StandardCode:     strOrEmpty(t.StandardCode),
		Domain:           strOrEmpty(t.Domain),
		DataTypeStandard: strOrEmpty(t.DataTypeStandard),
		ValidationRule:   strOrEmpty(t.ValidationRule),
		Owner:            strOrEmpty(t.Owner),
		Status:           t.Status,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

// DDLService DDL生成服务
type DDLService struct {
	store     store.Store
	generator *DDLGenerator
}

// NewDDLService 创建DDL服务
func NewDDLService(store store.Store) *DDLService {
	return &DDLService{
		store:     store,
		generator: NewDDLGenerator(),
	}
}

// GenerateDDL 生成DDL
func (s *DDLService) GenerateDDL(ctx context.Context, objectID string, targetType string) (*model.DDLGenerateResponse, error) {
	obj, cols, err := s.store.GetObjectWithColumns(ctx, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns found for object")
	}

	var sql string
	switch targetType {
	case "mysql":
		sql = s.generator.GenerateMySQLDDL(obj, cols)
	case "postgres":
		sql = s.generator.GeneratePostgresDDL(obj, cols)
	default:
		return nil, fmt.Errorf("unsupported target type: %s", targetType)
	}

	return &model.DDLGenerateResponse{
		ObjectID: objectID,
		SQL:      sql,
	}, nil
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
