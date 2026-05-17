package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/store"
)

// MetadataService 元数据服务
type MetadataService struct {
	store store.Store
}

// NewMetadataService 创建元数据服务
func NewMetadataService(store store.Store) *MetadataService {
	return &MetadataService{store: store}
}

// GetSchemaTree 获取Schema树
func (s *MetadataService) GetSchemaTree(ctx context.Context, sourceID string) (*model.SchemaTreeResponse, error) {
	// 验证数据源存在
	if _, err := s.store.GetDataSource(ctx, sourceID); err != nil {
		return nil, err
	}

	// 获取所有对象
	objects, err := s.store.ListSchemaObjectsBySource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schema objects: %w", err)
	}

	var objectResp []model.SchemaObjectWithColumns
	for _, obj := range objects {
		columns, err := s.store.ListColumnsByObject(ctx, obj.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list columns: %w", err)
		}

		var colResp []model.ColumnResponse
		for _, col := range columns {
			colResp = append(colResp, model.ColumnResponse{
				ID:              col.ID,
				Name:            col.Name,
				DataType:        col.DataType,
				FullDataType:    col.FullDataType,
				IsNullable:      col.IsNullable,
				DefaultValue:    col.DefaultValue,
				IsPrimaryKey:    col.IsPrimaryKey,
				OrdinalPosition: col.OrdinalPosition,
				Description:     col.Description,
			})
		}

		objectResp = append(objectResp, model.SchemaObjectWithColumns{
			SchemaObjectResponse: model.SchemaObjectResponse{
				ID:          obj.ID,
				Name:        obj.Name,
				Type:        model.ObjectType(obj.Type),
				Schema:      obj.Schema,
				Description: obj.Description,
				RowCount:    obj.RowCount,
				SizeBytes:   obj.SizeBytes,
				ColumnCount: len(colResp),
			},
			Columns: colResp,
		})
	}

	return &model.SchemaTreeResponse{
		SourceID: sourceID,
		Objects:  objectResp,
	}, nil
}

// GetColumnDetail 获取字段详情
func (s *MetadataService) GetColumnDetail(ctx context.Context, columnID string) (*model.ColumnDetailResponse, error) {
	col, err := s.store.GetColumn(ctx, columnID)
	if err != nil {
		return nil, err
	}

	// 获取对象信息
	obj, err := s.store.GetSchemaObject(ctx, col.ObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	// 获取数据源信息
	src, err := s.store.GetDataSource(ctx, obj.SourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}

	// 构建响应
	resp := &model.ColumnDetailResponse{
		ID:               col.ID,
		Name:             col.Name,
		DataType:         col.DataType,
		FullDataType:     col.FullDataType,
		IsNullable:       col.IsNullable,
		DefaultValue:     col.DefaultValue,
		IsPrimaryKey:     col.IsPrimaryKey,
		OrdinalPosition:  col.OrdinalPosition,
		Description:      col.Description,
		Confidence:       col.Confidence,
		ParentColumnPath: col.ParentColumnPath,
		Object: model.ObjectSummary{
			ID:   obj.ID,
			Name: obj.Name,
			Type: obj.Type,
		},
		Source: model.SourceSummary{
			ID:   src.ID,
			Name: src.Name,
			Type: src.Type,
		},
		MappedColumns: make([]model.MappedColumn, 0),
	}

	// 如果有术语ID，添加术语信息
	if col.TermID != nil {
		term, err := s.store.GetBusinessTerm(ctx, *col.TermID)
		if err != nil {
			return nil, fmt.Errorf("failed to get business term: %w", err)
		}
		resp.Term = &model.TermSummary{
			ID:   term.ID,
			Name: term.Name,
		}
	}

	mappings, err := s.store.GetColumnMappings(ctx, columnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get column mappings: %w", err)
	}
	for _, mapping := range mappings {
		resp.MappedColumns = append(resp.MappedColumns, model.MappedColumn{
			ID:          mapping.ID,
			Name:        mapping.TargetColumnName,
			ObjectName:  mapping.TargetObjectName,
			SourceName:  mapping.TargetSourceName,
			MappingType: mapping.MappingType,
			Confidence:  mapping.Confidence,
		})
	}

	return resp, nil
}

// SearchColumns 搜索字段
func (s *MetadataService) SearchColumns(ctx context.Context, query string, limit int) ([]model.ColumnSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	if limit <= 0 {
		limit = 20
	}

	results, err := s.store.SearchColumns(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	var searchResults []model.ColumnSearchResult
	for _, r := range results {
		searchResults = append(searchResults, model.ColumnSearchResult{
			ID:               r.ID,
			Name:             r.Name,
			DataType:         r.DataType,
			ObjectName:       r.ObjectName,
			SourceID:         r.SourceID,
			SourceName:       r.SourceName,
			SourceType:       r.SourceType,
			Confidence:       r.Confidence,
			ParentColumnPath: r.ParentColumnPath,
		})
	}

	return searchResults, nil
}

// ListSchemaChanges 获取Schema变更记录
func (s *MetadataService) ListSchemaChanges(ctx context.Context, sourceID string, limit int) ([]*model.SchemaChangeResponse, error) {
	if limit <= 0 {
		limit = 50
	}

	changes, err := s.store.ListSchemaChangesBySource(ctx, sourceID, limit)
	if err != nil {
		return nil, err
	}

	var resp []*model.SchemaChangeResponse
	for _, c := range changes {
		resp = append(resp, &model.SchemaChangeResponse{
			ID:           c.ID,
			ObjectID:     c.ObjectID,
			ChangeType:   c.ChangeType,
			ObjectType:   c.ObjectType,
			ObjectName:   c.ObjectName,
			OldValue:     c.OldValue,
			NewValue:     c.NewValue,
			DetectedAt:   c.DetectedAt,
			Acknowledged: c.Acknowledged,
		})
	}

	return resp, nil
}

// GetColumnMappings 获取字段映射
func (s *MetadataService) GetColumnMappings(ctx context.Context, columnID string) ([]*model.ColumnMappingResponse, error) {
	mappings, err := s.store.GetColumnMappings(ctx, columnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get column mappings: %w", err)
	}

	var resp []*model.ColumnMappingResponse
	for _, m := range mappings {
		resp = append(resp, &model.ColumnMappingResponse{
			ID:             m.ID,
			SourceColumnID: m.SourceColumnID,
			TargetColumnID: m.TargetColumnID,
			MappingType:    m.MappingType,
			Confidence:     m.Confidence,
			CreatedAt:      m.CreatedAt,
			TargetColumn: model.ColumnSummary{
				ID:         m.TargetColumnID,
				Name:       m.TargetColumnName,
				ObjectName: m.TargetObjectName,
				SourceName: m.TargetSourceName,
			},
		})
	}

	return resp, nil
}

// CreateColumnMapping 创建字段映射
func (s *MetadataService) CreateColumnMapping(ctx context.Context, req *model.ColumnMappingRequest) error {
	mapping := &store.ColumnMappingCreate{
		SourceColumnID: req.SourceColumnID,
		TargetColumnID: req.TargetColumnID,
		MappingType:    req.MappingType,
		Confidence:     req.Confidence,
	}
	if mapping.Confidence <= 0 {
		mapping.Confidence = 1.0
	}
	return s.store.CreateColumnMapping(ctx, mapping)
}

// DeleteColumnMapping 删除字段映射
func (s *MetadataService) DeleteColumnMapping(ctx context.Context, id string) error {
	return s.store.DeleteColumnMapping(ctx, id)
}

// GetLineage 获取血缘关系
func (s *MetadataService) GetLineage(ctx context.Context, columnID string) (*model.LineageResponse, error) {
	upward, err := s.store.GetLineageUpward(ctx, columnID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get lineage upward: %w", err)
	}

	downward, err := s.store.GetLineageDownward(ctx, columnID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get lineage downward: %w", err)
	}

	resp := &model.LineageResponse{
		ColumnID: columnID,
		Upward:   make([]model.LineageEdgeResponse, 0, len(upward)),
		Downward: make([]model.LineageEdgeResponse, 0, len(downward)),
	}

	for _, e := range upward {
		resp.Upward = append(resp.Upward, model.LineageEdgeResponse{
			Source: model.LineageNode{
				ID:       e.SourceID,
				Name:     e.SourceName,
				Type:     e.SourceType,
				DataType: e.SourceDataType,
				Source:   e.SourceSourceName,
			},
			Target: model.LineageNode{
				ID:       e.TargetID,
				Name:     e.TargetName,
				Type:     e.TargetType,
				DataType: e.TargetDataType,
				Source:   e.TargetSourceName,
			},
			TransformSQL: e.TransformSQL,
			JobName:      e.JobName,
		})
	}

	for _, e := range downward {
		resp.Downward = append(resp.Downward, model.LineageEdgeResponse{
			Source: model.LineageNode{
				ID:       e.SourceID,
				Name:     e.SourceName,
				Type:     e.SourceType,
				DataType: e.SourceDataType,
				Source:   e.SourceSourceName,
			},
			Target: model.LineageNode{
				ID:       e.TargetID,
				Name:     e.TargetName,
				Type:     e.TargetType,
				DataType: e.TargetDataType,
				Source:   e.TargetSourceName,
			},
			TransformSQL: e.TransformSQL,
			JobName:      e.JobName,
		})
	}

	return resp, nil
}

// GetImpactAnalysis 获取影响分析
func (s *MetadataService) GetImpactAnalysis(ctx context.Context, columnID string) (*model.ImpactAnalysisResponse, error) {
	// 获取下游血缘（影响范围）
	downward, err := s.store.GetLineageDownward(ctx, columnID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get impact analysis: %w", err)
	}

	rootName := columnID
	adjacency := make(map[string][]*store.LineageEdgeRow)
	for _, edge := range downward {
		if edge.SourceID == columnID && edge.SourceName != "" {
			rootName = edge.SourceName
		}
		adjacency[edge.SourceID] = append(adjacency[edge.SourceID], edge)
	}

	type queueItem struct {
		NodeID string
		Path   []string
	}

	bestDistance := map[string]int{
		columnID: 0,
	}
	queue := []queueItem{{
		NodeID: columnID,
		Path:   []string{rootName},
	}}
	impactMap := make(map[string]model.ImpactObject)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range adjacency[current.NodeID] {
			distance := bestDistance[current.NodeID] + 1
			if previousDistance, exists := bestDistance[edge.TargetID]; exists && previousDistance <= distance {
				continue
			}

			nextPath := append(append([]string{}, current.Path...), edge.TargetName)
			bestDistance[edge.TargetID] = distance
			impactMap[edge.TargetID] = model.ImpactObject{
				ID:         edge.TargetID,
				Name:       edge.TargetName,
				Type:       edge.TargetType,
				ObjectName: edge.TargetObjectName,
				SourceName: edge.TargetSourceName,
				ImpactPath: strings.Join(nextPath, " -> "),
				Distance:   distance,
			}
			queue = append(queue, queueItem{
				NodeID: edge.TargetID,
				Path:   nextPath,
			})
		}
	}

	impacts := make([]model.ImpactObject, 0, len(impactMap))
	for _, impact := range impactMap {
		impacts = append(impacts, impact)
	}
	sort.Slice(impacts, func(i, j int) bool {
		if impacts[i].Distance != impacts[j].Distance {
			return impacts[i].Distance < impacts[j].Distance
		}
		if impacts[i].Name != impacts[j].Name {
			return impacts[i].Name < impacts[j].Name
		}
		return impacts[i].ID < impacts[j].ID
	})

	return &model.ImpactAnalysisResponse{
		ColumnID:      columnID,
		ImpactObjects: impacts,
		TotalObjects:  len(impacts),
	}, nil
}
