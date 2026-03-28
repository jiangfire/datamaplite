package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateColumn 创建字段
func (s *PostgresStore) CreateColumn(ctx context.Context, col *ColumnCreate) error {
	query := `
		INSERT INTO columns (id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		                     is_primary_key, is_unique, ordinal_position, description, parent_column_path, confidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query,
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
func (s *PostgresStore) GetColumn(ctx context.Context, id string) (*ColumnRow, error) {
	query := `
		SELECT id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		       is_primary_key, is_unique, ordinal_position, description, term_id::text,
		       confidence, parent_column_path, created_at::text, updated_at::text
		FROM columns
		WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, query, id)

	var col ColumnRow
	err := row.Scan(
		&col.ID, &col.ObjectID, &col.Name, &col.DataType, &col.FullDataType,
		&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsUnique,
		&col.OrdinalPosition, &col.Description, &col.TermID, &col.Confidence,
		&col.ParentColumnPath, &col.CreatedAt, &col.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("column not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get column: %w", err)
	}
	return &col, nil
}

// ListColumnsByObject 获取对象的所有字段
func (s *PostgresStore) ListColumnsByObject(ctx context.Context, objectID string) ([]*ColumnRow, error) {
	query := `
		SELECT id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		       is_primary_key, is_unique, ordinal_position, description, term_id::text,
		       confidence, parent_column_path, created_at::text, updated_at::text
		FROM columns
		WHERE object_id = $1
		ORDER BY ordinal_position, name
	`
	rows, err := s.pool.Query(ctx, query, objectID)
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
func (s *PostgresStore) UpdateColumn(ctx context.Context, id string, updates *ColumnUpdate) error {
	query := `
		UPDATE columns
		SET data_type = $1, full_data_type = $2, is_nullable = $3, default_value = $4, is_primary_key = $5,
		    is_unique = $6, ordinal_position = $7, description = $8, parent_column_path = $9, confidence = $10,
		    updated_at = NOW()
		WHERE id = $11
	`
	result, err := s.pool.Exec(ctx, query,
		updates.DataType, updates.FullDataType, updates.IsNullable, updates.DefaultValue,
		updates.IsPrimaryKey, updates.IsUnique, updates.OrdinalPosition, updates.Description,
		updates.ParentColumnPath, updates.Confidence, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("column not found: %s", id)
	}
	return nil
}

// DeleteColumn 删除单个字段
func (s *PostgresStore) DeleteColumn(ctx context.Context, id string) error {
	result, err := s.pool.Exec(ctx, `DELETE FROM columns WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("column not found: %s", id)
	}
	return nil
}

// SearchColumns 搜索字段
func (s *PostgresStore) SearchColumns(ctx context.Context, query string, limit int) ([]*ColumnSearchRow, error) {
	if limit <= 0 {
		limit = 20
	}

	sql := `
		SELECT c.id, c.object_id, c.name, c.data_type, c.full_data_type, c.is_nullable,
		       c.default_value, c.is_primary_key, c.is_unique, c.ordinal_position,
		       c.description, c.term_id::text, c.confidence, c.parent_column_path,
		       c.created_at::text, c.updated_at::text,
		       o.name as object_name, o.source_id, ds.name as source_name, ds.type as source_type
		FROM columns c
		JOIN schema_objects o ON c.object_id = o.id
		JOIN data_sources ds ON o.source_id = ds.id
		WHERE c.name ILIKE $1 OR c.description ILIKE $1
		ORDER BY c.name
		LIMIT $2
	`
	likeQuery := "%" + query + "%"
	rows, err := s.pool.Query(ctx, sql, likeQuery, limit)
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
func (s *PostgresStore) DeleteColumnsByObject(ctx context.Context, objectID string) error {
	query := `DELETE FROM columns WHERE object_id = $1`
	_, err := s.pool.Exec(ctx, query, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete columns: %w", err)
	}
	return nil
}

// Transaction implementations for PostgresTxStore

func (t *PostgresTxStore) CreateColumn(ctx context.Context, col *ColumnCreate) error {
	query := `
		INSERT INTO columns (id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		                     is_primary_key, is_unique, ordinal_position, description, parent_column_path, confidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, col.ObjectID, col.Name, col.DataType, col.FullDataType,
		col.IsNullable, col.DefaultValue, col.IsPrimaryKey, col.IsUnique,
		col.OrdinalPosition, col.Description, col.ParentColumnPath, col.Confidence,
	)
	if err != nil {
		return fmt.Errorf("failed to create column: %w", err)
	}
	return nil
}

func (t *PostgresTxStore) GetColumn(ctx context.Context, id string) (*ColumnRow, error) {
	query := `
		SELECT id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		       is_primary_key, is_unique, ordinal_position, description, term_id::text,
		       confidence, parent_column_path, created_at::text, updated_at::text
		FROM columns
		WHERE id = $1
	`
	row := t.tx.QueryRow(ctx, query, id)

	var col ColumnRow
	err := row.Scan(
		&col.ID, &col.ObjectID, &col.Name, &col.DataType, &col.FullDataType,
		&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey, &col.IsUnique,
		&col.OrdinalPosition, &col.Description, &col.TermID, &col.Confidence,
		&col.ParentColumnPath, &col.CreatedAt, &col.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("column not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get column: %w", err)
	}
	return &col, nil
}

func (t *PostgresTxStore) ListColumnsByObject(ctx context.Context, objectID string) ([]*ColumnRow, error) {
	query := `
		SELECT id, object_id, name, data_type, full_data_type, is_nullable, default_value,
		       is_primary_key, is_unique, ordinal_position, description, term_id::text,
		       confidence, parent_column_path, created_at::text, updated_at::text
		FROM columns
		WHERE object_id = $1
		ORDER BY ordinal_position, name
	`
	rows, err := t.tx.Query(ctx, query, objectID)
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

func (t *PostgresTxStore) SearchColumns(ctx context.Context, query string, limit int) ([]*ColumnSearchRow, error) {
	if limit <= 0 {
		limit = 20
	}

	sql := `
		SELECT c.id, c.object_id, c.name, c.data_type, c.full_data_type, c.is_nullable,
		       c.default_value, c.is_primary_key, c.is_unique, c.ordinal_position,
		       c.description, c.term_id::text, c.confidence, c.parent_column_path,
		       c.created_at::text, c.updated_at::text,
		       o.name as object_name, o.source_id, ds.name as source_name, ds.type as source_type
		FROM columns c
		JOIN schema_objects o ON c.object_id = o.id
		JOIN data_sources ds ON o.source_id = ds.id
		WHERE c.name ILIKE $1 OR c.description ILIKE $1
		ORDER BY c.name
		LIMIT $2
	`
	likeQuery := "%" + query + "%"
	rows, err := t.tx.Query(ctx, sql, likeQuery, limit)
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

func (t *PostgresTxStore) DeleteColumnsByObject(ctx context.Context, objectID string) error {
	query := `DELETE FROM columns WHERE object_id = $1`
	_, err := t.tx.Exec(ctx, query, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete columns: %w", err)
	}
	return nil
}

func (t *PostgresTxStore) UpdateColumn(ctx context.Context, id string, updates *ColumnUpdate) error {
	query := `
		UPDATE columns
		SET data_type = $1, full_data_type = $2, is_nullable = $3, default_value = $4, is_primary_key = $5,
		    is_unique = $6, ordinal_position = $7, description = $8, parent_column_path = $9, confidence = $10,
		    updated_at = NOW()
		WHERE id = $11
	`
	result, err := t.tx.Exec(ctx, query,
		updates.DataType, updates.FullDataType, updates.IsNullable, updates.DefaultValue,
		updates.IsPrimaryKey, updates.IsUnique, updates.OrdinalPosition, updates.Description,
		updates.ParentColumnPath, updates.Confidence, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("column not found: %s", id)
	}
	return nil
}

func (t *PostgresTxStore) DeleteColumn(ctx context.Context, id string) error {
	result, err := t.tx.Exec(ctx, `DELETE FROM columns WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("column not found: %s", id)
	}
	return nil
}
