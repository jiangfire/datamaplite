package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// CreateColumn 创建字段
func (s *SQLiteStore) CreateColumn(ctx context.Context, col *ColumnCreate) error {
	query := `
		INSERT INTO columns (id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		                     is_primary_key, is_unique, ordinal_position, description, parent_column_path, confidence)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, query,
		id, col.ObjectID, col.Name, col.DataType, col.FullDataType,
		col.IsNullable, col.DefaultValue, col.IsPrimaryKey, col.IsUnique,
		col.OrdinalPosition, col.Description, col.ParentColumnPath, col.Confidence,
	)
	if err != nil {
		return fmt.Errorf("failed to create column: %w", err)
	}
	return nil
}

// GetColumn 获取字段
func (s *SQLiteStore) GetColumn(ctx context.Context, id string) (*ColumnRow, error) {
	query := `
		SELECT id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		       is_primary_key, is_unique, ordinal_position, description, term_id,
		       confidence, parent_column_path, created_at, updated_at
		FROM columns
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var col ColumnRow
	err := row.Scan(
		&col.ID, &col.ObjectID, &col.Name, &col.DataType, &col.FullDataType,
		&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsUnique,
		&col.OrdinalPosition, &col.Description, &col.TermID, &col.Confidence,
		&col.ParentColumnPath, &col.CreatedAt, &col.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("column not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get column: %w", err)
	}
	return &col, nil
}

// ListColumnsByObject 获取对象的所有字段
func (s *SQLiteStore) ListColumnsByObject(ctx context.Context, objectID string) ([]*ColumnRow, error) {
	query := `
		SELECT id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		       is_primary_key, is_unique, ordinal_position, description, term_id,
		       confidence, parent_column_path, created_at, updated_at
		FROM columns
		WHERE object_id = ?
		ORDER BY ordinal_position, name
	`
	rows, err := s.db.QueryContext(ctx, query, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list columns: %w", err)
	}
	defer rows.Close()

	var columns []*ColumnRow
	for rows.Next() {
		var col ColumnRow
		err := rows.Scan(
			&col.ID, &col.ObjectID, &col.Name, &col.DataType, &col.FullDataType,
			&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsUnique,
			&col.OrdinalPosition, &col.Description, &col.TermID, &col.Confidence,
			&col.ParentColumnPath, &col.CreatedAt, &col.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		columns = append(columns, &col)
	}
	return columns, rows.Err()
}

// UpdateColumn 更新字段
func (s *SQLiteStore) UpdateColumn(ctx context.Context, id string, updates *ColumnUpdate) error {
	query := `
		UPDATE columns
		SET data_type = ?, full_data_type = ?, is_nullable = ?, default_value = ?, is_primary_key = ?,
		    is_unique = ?, ordinal_position = ?, description = ?, parent_column_path = ?, confidence = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query,
		updates.DataType, updates.FullDataType, updates.IsNullable, updates.DefaultValue,
		updates.IsPrimaryKey, updates.IsUnique, updates.OrdinalPosition, updates.Description,
		updates.ParentColumnPath, updates.Confidence, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update column: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("column not found: %s", id)
	}
	return nil
}

// DeleteColumn 删除单个字段
func (s *SQLiteStore) DeleteColumn(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM columns WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete column: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("column not found: %s", id)
	}
	return nil
}

// SearchColumns 搜索字段
func (s *SQLiteStore) SearchColumns(ctx context.Context, query string, limit int) ([]*ColumnSearchRow, error) {
	if limit <= 0 {
		limit = 20
	}

	sql := `
		SELECT c.id, c.object_id, c.name, c.data_type, c.full_data_type, c.is_nullable,
		       c.default_value, c.is_primary_key, c.is_unique, c.ordinal_position,
		       c.description, c.term_id, c.confidence, c.parent_column_path,
		       c.created_at, c.updated_at,
		       o.name as object_name, o.source_id, ds.name as source_name, ds.type as source_type
		FROM columns c
		JOIN schema_objects o ON c.object_id = o.id
		JOIN data_sources ds ON o.source_id = ds.id
		WHERE c.name LIKE ? OR c.description LIKE ?
		ORDER BY c.name
		LIMIT ?
	`
	likeQuery := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, sql, likeQuery, likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search columns: %w", err)
	}
	defer rows.Close()

	var columns []*ColumnSearchRow
	for rows.Next() {
		var col ColumnSearchRow
		err := rows.Scan(
			&col.ID, &col.ObjectID, &col.Name, &col.DataType, &col.FullDataType,
			&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsUnique,
			&col.OrdinalPosition, &col.Description, &col.TermID, &col.Confidence,
			&col.ParentColumnPath, &col.CreatedAt, &col.UpdatedAt,
			&col.ObjectName, &col.SourceID, &col.SourceName, &col.SourceType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column search result: %w", err)
		}
		columns = append(columns, &col)
	}
	return columns, rows.Err()
}

// DeleteColumnsByObject 删除对象的所有字段
func (s *SQLiteStore) DeleteColumnsByObject(ctx context.Context, objectID string) error {
	query := `DELETE FROM columns WHERE object_id = ?`
	_, err := s.db.ExecContext(ctx, query, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete columns: %w", err)
	}
	return nil
}

// Transaction implementations for SQLiteTxStore

func (t *SQLiteTxStore) CreateColumn(ctx context.Context, col *ColumnCreate) error {
	query := `
		INSERT INTO columns (id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		                     is_primary_key, is_unique, ordinal_position, description, parent_column_path, confidence)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, col.ObjectID, col.Name, col.DataType, col.FullDataType,
		col.IsNullable, col.DefaultValue, col.IsPrimaryKey, col.IsUnique,
		col.OrdinalPosition, col.Description, col.ParentColumnPath, col.Confidence,
	)
	if err != nil {
		return fmt.Errorf("failed to create column: %w", err)
	}
	return nil
}

func (t *SQLiteTxStore) GetColumn(ctx context.Context, id string) (*ColumnRow, error) {
	query := `
		SELECT id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		       is_primary_key, is_unique, ordinal_position, description, term_id,
		       confidence, parent_column_path, created_at, updated_at
		FROM columns
		WHERE id = ?
	`
	row := t.tx.QueryRowContext(ctx, query, id)

	var col ColumnRow
	err := row.Scan(
		&col.ID, &col.ObjectID, &col.Name, &col.DataType, &col.FullDataType,
		&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsUnique,
		&col.OrdinalPosition, &col.Description, &col.TermID, &col.Confidence,
		&col.ParentColumnPath, &col.CreatedAt, &col.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("column not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get column: %w", err)
	}
	return &col, nil
}

func (t *SQLiteTxStore) ListColumnsByObject(ctx context.Context, objectID string) ([]*ColumnRow, error) {
	query := `
		SELECT id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		       is_primary_key, is_unique, ordinal_position, description, term_id,
		       confidence, parent_column_path, created_at, updated_at
		FROM columns
		WHERE object_id = ?
		ORDER BY ordinal_position, name
	`
	rows, err := t.tx.QueryContext(ctx, query, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list columns: %w", err)
	}
	defer rows.Close()

	var columns []*ColumnRow
	for rows.Next() {
		var col ColumnRow
		err := rows.Scan(
			&col.ID, &col.ObjectID, &col.Name, &col.DataType, &col.FullDataType,
			&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsUnique,
			&col.OrdinalPosition, &col.Description, &col.TermID, &col.Confidence,
			&col.ParentColumnPath, &col.CreatedAt, &col.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		columns = append(columns, &col)
	}
	return columns, rows.Err()
}

func (t *SQLiteTxStore) SearchColumns(ctx context.Context, query string, limit int) ([]*ColumnSearchRow, error) {
	if limit <= 0 {
		limit = 20
	}

	sql := `
		SELECT c.id, c.object_id, c.name, c.data_type, c.full_data_type, c.is_nullable,
		       c.default_value, c.is_primary_key, c.is_unique, c.ordinal_position,
		       c.description, c.term_id, c.confidence, c.parent_column_path,
		       c.created_at, c.updated_at,
		       o.name as object_name, o.source_id, ds.name as source_name, ds.type as source_type
		FROM columns c
		JOIN schema_objects o ON c.object_id = o.id
		JOIN data_sources ds ON o.source_id = ds.id
		WHERE c.name LIKE ? OR c.description LIKE ?
		ORDER BY c.name
		LIMIT ?
	`
	likeQuery := "%" + query + "%"
	rows, err := t.tx.QueryContext(ctx, sql, likeQuery, likeQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search columns: %w", err)
	}
	defer rows.Close()

	var columns []*ColumnSearchRow
	for rows.Next() {
		var col ColumnSearchRow
		err := rows.Scan(
			&col.ID, &col.ObjectID, &col.Name, &col.DataType, &col.FullDataType,
			&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsUnique,
			&col.OrdinalPosition, &col.Description, &col.TermID, &col.Confidence,
			&col.ParentColumnPath, &col.CreatedAt, &col.UpdatedAt,
			&col.ObjectName, &col.SourceID, &col.SourceName, &col.SourceType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column search result: %w", err)
		}
		columns = append(columns, &col)
	}
	return columns, rows.Err()
}

func (t *SQLiteTxStore) DeleteColumnsByObject(ctx context.Context, objectID string) error {
	query := `DELETE FROM columns WHERE object_id = ?`
	_, err := t.tx.ExecContext(ctx, query, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete columns: %w", err)
	}
	return nil
}

func (t *SQLiteTxStore) UpdateColumn(ctx context.Context, id string, updates *ColumnUpdate) error {
	query := `
		UPDATE columns
		SET data_type = ?, full_data_type = ?, is_nullable = ?, default_value = ?, is_primary_key = ?,
		    is_unique = ?, ordinal_position = ?, description = ?, parent_column_path = ?, confidence = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`
	result, err := t.tx.ExecContext(ctx, query,
		updates.DataType, updates.FullDataType, updates.IsNullable, updates.DefaultValue,
		updates.IsPrimaryKey, updates.IsUnique, updates.OrdinalPosition, updates.Description,
		updates.ParentColumnPath, updates.Confidence, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update column: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("column not found: %s", id)
	}
	return nil
}

func (t *SQLiteTxStore) DeleteColumn(ctx context.Context, id string) error {
	result, err := t.tx.ExecContext(ctx, `DELETE FROM columns WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete column: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("column not found: %s", id)
	}
	return nil
}
