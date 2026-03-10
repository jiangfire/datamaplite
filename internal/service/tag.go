package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
)

// TagService 标签服务
type TagService struct {
	store store.Store
}

// NewTagService 创建标签服务
func NewTagService(store store.Store) *TagService {
	return &TagService{store: store}
}

// CreateTag 创建标签
func (s *TagService) CreateTag(ctx context.Context, req *model.TagRequest) (*model.TagResponse, error) {
	// 验证颜色格式
	if !isValidHexColor(req.Color) {
		return nil, fmt.Errorf("invalid color format, expected hex color like #6366f1")
	}

	// 检查名称是否已存在
	if _, err := s.store.GetTagByName(ctx, req.Name); err == nil {
		return nil, fmt.Errorf("tag with name '%s' already exists", req.Name)
	}

	tagCreate := &store.TagCreate{
		Name:        req.Name,
		Color:       req.Color,
		Description: stringPtr(req.Description),
	}

	id, err := s.store.CreateTag(ctx, tagCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return &model.TagResponse{
		ID:          id,
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// GetTag 获取标签
func (s *TagService) GetTag(ctx context.Context, id string) (*model.TagResponse, error) {
	tag, err := s.store.GetTag(ctx, id)
	if err != nil {
		return nil, err
	}

	return tagRowToResponse(tag), nil
}

// ListTags 列出所有标签
func (s *TagService) ListTags(ctx context.Context) ([]*model.TagResponse, error) {
	tags, err := s.store.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	result := make([]*model.TagResponse, len(tags))
	for i, tag := range tags {
		result[i] = tagRowToResponse(tag)
	}

	return result, nil
}

// UpdateTag 更新标签
func (s *TagService) UpdateTag(ctx context.Context, id string, req *model.TagRequest) error {
	// 验证颜色格式
	if !isValidHexColor(req.Color) {
		return fmt.Errorf("invalid color format, expected hex color like #6366f1")
	}

	// 检查标签是否存在
	existingTag, err := s.store.GetTag(ctx, id)
	if err != nil {
		return err
	}

	// 如果名称变更，检查新名称是否已存在
	if req.Name != existingTag.Name {
		if _, err := s.store.GetTagByName(ctx, req.Name); err == nil {
			return fmt.Errorf("tag with name '%s' already exists", req.Name)
		}
	}

	updates := &store.TagUpdate{
		Name:        &req.Name,
		Color:       &req.Color,
		Description: stringPtr(req.Description),
	}

	if err := s.store.UpdateTag(ctx, id, updates); err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	return nil
}

// DeleteTag 删除标签
func (s *TagService) DeleteTag(ctx context.Context, id string) error {
	if err := s.store.DeleteTag(ctx, id); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	return nil
}

// AddTagToColumn 给字段添加标签
func (s *TagService) AddTagToColumn(ctx context.Context, columnID string, tagID string) error {
	if err := s.store.AddTagToColumn(ctx, columnID, tagID); err != nil {
		return fmt.Errorf("failed to add tag to column: %w", err)
	}
	return nil
}

// RemoveTagFromColumn 从字段移除标签
func (s *TagService) RemoveTagFromColumn(ctx context.Context, columnID string, tagID string) error {
	if err := s.store.RemoveTagFromColumn(ctx, columnID, tagID); err != nil {
		return fmt.Errorf("failed to remove tag from column: %w", err)
	}
	return nil
}

// GetColumnTags 获取字段的所有标签
func (s *TagService) GetColumnTags(ctx context.Context, columnID string) ([]*model.TagResponse, error) {
	tags, err := s.store.GetColumnTags(ctx, columnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get column tags: %w", err)
	}

	result := make([]*model.TagResponse, len(tags))
	for i, tag := range tags {
		result[i] = tagRowToResponse(tag)
	}

	return result, nil
}

// GetColumnsByTag 获取带有指定标签的所有字段
func (s *TagService) GetColumnsByTag(ctx context.Context, tagID string) ([]*model.ColumnSearchResult, error) {
	columns, err := s.store.SearchColumnsByTag(ctx, tagID, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns by tag: %w", err)
	}

	result := make([]*model.ColumnSearchResult, len(columns))
	for i, col := range columns {
		result[i] = &model.ColumnSearchResult{
			ID:               col.ID,
			Name:             col.Name,
			DataType:         col.DataType,
			ObjectName:       col.ObjectName,
			SourceID:         col.SourceID,
			SourceName:       col.SourceName,
			SourceType:       col.SourceType,
			Confidence:       col.Confidence,
			ParentColumnPath: col.ParentColumnPath,
		}
	}

	return result, nil
}

// 辅助函数

func tagRowToResponse(tag *store.TagRow) *model.TagResponse {
	return &model.TagResponse{
		ID:          tag.ID,
		Name:        tag.Name,
		Color:       tag.Color,
		Description: derefString(tag.Description),
		CreatedAt:   tag.CreatedAt,
		UpdatedAt:   tag.UpdatedAt,
	}
}

func isValidHexColor(color string) bool {
	// 支持 #RGB 或 #RRGGBB 格式
	color = strings.ToLower(color)
	match, _ := regexp.MatchString(`^#([0-9a-f]{3}|[0-9a-f]{6})$`, color)
	return match
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
