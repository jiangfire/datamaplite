package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// CreateTag 创建标签
func (t *SQLiteTxStore) CreateTag(ctx context.Context, tag *TagCreate) (string, error) {
	query := `
		INSERT INTO tags (id, name, color, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query, id, tag.Name, tag.Color, tag.Description)
	if err != nil {
		return "", fmt.Errorf("failed to create tag: %w", err)
	}
	return id, nil
}

// GetTag 根据ID获取标签
func (t *SQLiteTxStore) GetTag(ctx context.Context, id string) (*TagRow, error) {
	query := `
		SELECT id, name, color, description, created_at, updated_at
		FROM tags WHERE id = ?
	`
	row := t.tx.QueryRowContext(ctx, query, id)
	var tag TagRow
	err := row.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tag not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}
	return &tag, nil
}

// GetTagByName 根据名称获取标签
func (t *SQLiteTxStore) GetTagByName(ctx context.Context, name string) (*TagRow, error) {
	query := `
		SELECT id, name, color, description, created_at, updated_at
		FROM tags WHERE name = ?
	`
	row := t.tx.QueryRowContext(ctx, query, name)
	var tag TagRow
	err := row.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description, &tag.CreatedAt, &tag.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tag not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get tag by name: %w", err)
	}
	return &tag, nil
}

// ListTags 列出所有标签
func (t *SQLiteTxStore) ListTags(ctx context.Context) ([]*TagRow, error) {
	query := `
		SELECT id, name, color, description, created_at, updated_at
		FROM tags ORDER BY name ASC
	`
	rows, err := t.tx.QueryContext(ctx, query)
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
func (t *SQLiteTxStore) UpdateTag(ctx context.Context, id string, updates *TagUpdate) error {
	query := `
		UPDATE tags SET
			name = COALESCE(?, name),
			color = COALESCE(?, color),
			description = COALESCE(?, description),
			updated_at = datetime('now')
		WHERE id = ?
	`
	result, err := t.tx.ExecContext(ctx, query, updates.Name, updates.Color, updates.Description, id)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("tag not found: %s", id)
	}
	return nil
}

// DeleteTag 删除标签
func (t *SQLiteTxStore) DeleteTag(ctx context.Context, id string) error {
	query := `DELETE FROM tags WHERE id = ?`
	result, err := t.tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("tag not found: %s", id)
	}
	return nil
}

// AddTagToColumn 给字段添加标签
func (t *SQLiteTxStore) AddTagToColumn(ctx context.Context, columnID string, tagID string) error {
	query := `
		INSERT OR IGNORE INTO column_tags (id, column_id, tag_id, created_at)
		VALUES (?, ?, ?, datetime('now'))
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query, id, columnID, tagID)
	if err != nil {
		return fmt.Errorf("failed to add tag to column: %w", err)
	}
	return nil
}

// RemoveTagFromColumn 从字段移除标签
func (t *SQLiteTxStore) RemoveTagFromColumn(ctx context.Context, columnID string, tagID string) error {
	query := `DELETE FROM column_tags WHERE column_id = ? AND tag_id = ?`
	result, err := t.tx.ExecContext(ctx, query, columnID, tagID)
	if err != nil {
		return fmt.Errorf("failed to remove tag from column: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("tag not found on column")
	}
	return nil
}

// GetColumnTags 获取字段的所有标签
func (t *SQLiteTxStore) GetColumnTags(ctx context.Context, columnID string) ([]*TagRow, error) {
	query := `
		SELECT t.id, t.name, t.color, t.description, t.created_at, t.updated_at
		FROM tags t
		JOIN column_tags ct ON t.id = ct.tag_id
		WHERE ct.column_id = ?
		ORDER BY t.name ASC
	`
	rows, err := t.tx.QueryContext(ctx, query, columnID)
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
func (t *SQLiteTxStore) SearchColumnsByTag(ctx context.Context, tagID string, limit int) ([]*ColumnSearchRow, error) {
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
		WHERE ct.tag_id = ?
		ORDER BY c.name ASC
		LIMIT ?
	`
	rows, err := t.tx.QueryContext(ctx, query, tagID, limit)
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
