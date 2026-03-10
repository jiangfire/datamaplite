package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CreateTag 创建标签
func (s *PostgresStore) CreateTag(ctx context.Context, tag *TagCreate) (string, error) {
	query := `
		INSERT INTO tags (id, name, color, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`
	id := uuid.New().String()
	var returnedID string
	err := s.pool.QueryRow(ctx, query, id, tag.Name, tag.Color, tag.Description).Scan(&returnedID)
	if err != nil {
		return "", fmt.Errorf("failed to create tag: %w", err)
	}
	return returnedID, nil
}

// GetTag 根据ID获取标签
func (s *PostgresStore) GetTag(ctx context.Context, id string) (*TagRow, error) {
	query := `
		SELECT id, name, color, description, created_at, updated_at
		FROM tags WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, query, id)
	var tag TagRow
	err := row.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("tag not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}
	return &tag, nil
}

// GetTagByName 根据名称获取标签
func (s *PostgresStore) GetTagByName(ctx context.Context, name string) (*TagRow, error) {
	query := `
		SELECT id, name, color, description, created_at, updated_at
		FROM tags WHERE name = $1
	`
	row := s.pool.QueryRow(ctx, query, name)
	var tag TagRow
	err := row.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("tag not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get tag by name: %w", err)
	}
	return &tag, nil
}

// ListTags 列出所有标签
func (s *PostgresStore) ListTags(ctx context.Context) ([]*TagRow, error) {
	query := `
		SELECT id, name, color, description, created_at, updated_at
		FROM tags ORDER BY name ASC
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []*TagRow
	for rows.Next() {
		var tag TagRow
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}

	return tags, rows.Err()
}

// UpdateTag 更新标签
func (s *PostgresStore) UpdateTag(ctx context.Context, id string, updates *TagUpdate) error {
	query := `
		UPDATE tags SET
			name = COALESCE($2, name),
			color = COALESCE($3, color),
			description = COALESCE($4, description),
			updated_at = NOW()
		WHERE id = $1
	`
	result, err := s.pool.Exec(ctx, query, id, updates.Name, updates.Color, updates.Description)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("tag not found: %s", id)
	}
	return nil
}

// DeleteTag 删除标签
func (s *PostgresStore) DeleteTag(ctx context.Context, id string) error {
	query := `DELETE FROM tags WHERE id = $1`
	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("tag not found: %s", id)
	}
	return nil
}

// AddTagToColumn 给字段添加标签
func (s *PostgresStore) AddTagToColumn(ctx context.Context, columnID string, tagID string) error {
	query := `
		INSERT INTO column_tags (id, column_id, tag_id, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (column_id, tag_id) DO NOTHING
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query, id, columnID, tagID)
	if err != nil {
		return fmt.Errorf("failed to add tag to column: %w", err)
	}
	return nil
}

// RemoveTagFromColumn 从字段移除标签
func (s *PostgresStore) RemoveTagFromColumn(ctx context.Context, columnID string, tagID string) error {
	query := `DELETE FROM column_tags WHERE column_id = $1 AND tag_id = $2`
	result, err := s.pool.Exec(ctx, query, columnID, tagID)
	if err != nil {
		return fmt.Errorf("failed to remove tag from column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("tag not found on column")
	}
	return nil
}

// GetColumnTags 获取字段的所有标签
func (s *PostgresStore) GetColumnTags(ctx context.Context, columnID string) ([]*TagRow, error) {
	query := `
		SELECT t.id, t.name, t.color, t.description, t.created_at, t.updated_at
		FROM tags t
		JOIN column_tags ct ON t.id = ct.tag_id
		WHERE ct.column_id = $1
		ORDER BY t.name ASC
	`
	rows, err := s.pool.Query(ctx, query, columnID)
	if err != nil {
		return nil, fmt.Errorf("failed to get column tags: %w", err)
	}
	defer rows.Close()

	var tags []*TagRow
	for rows.Next() {
		var tag TagRow
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}

	return tags, rows.Err()
}

// SearchColumnsByTag 根据标签搜索字段
func (s *PostgresStore) SearchColumnsByTag(ctx context.Context, tagID string, limit int) ([]*ColumnSearchRow, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT c.id, c.name, c.data_type, so.name as object_name,
		       so.source_id, ds.name as source_name, ds.type as source_type,
		       c.confidence, c.parent_column_path
		FROM columns c
		JOIN schema_objects so ON c.object_id = so.id
		JOIN data_sources ds ON so.source_id = ds.id
		JOIN column_tags ct ON c.id = ct.column_id
		WHERE ct.tag_id = $1
		ORDER BY c.name ASC
		LIMIT $2
	`
	rows, err := s.pool.Query(ctx, query, tagID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search columns by tag: %w", err)
	}
	defer rows.Close()

	var columns []*ColumnSearchRow
	for rows.Next() {
		var col ColumnSearchRow
		err := rows.Scan(
			&col.ID, &col.Name, &col.DataType, &col.ObjectName,
			&col.SourceID, &col.SourceName, &col.SourceType,
			&col.Confidence, &col.ParentColumnPath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		columns = append(columns, &col)
	}

	return columns, rows.Err()
}
