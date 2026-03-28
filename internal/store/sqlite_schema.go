package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// CreateSchemaObject 创建Schema对象
func (s *SQLiteStore) CreateSchemaObject(ctx context.Context, obj *SchemaObjectCreate) (string, error) {
	query := `
		INSERT INTO schema_objects (id, source_id, name, type, schema, description, row_count, size_bytes, column_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, query,
		id, obj.SourceID, obj.Name, obj.Type, obj.Schema,
		obj.Description, obj.RowCount, obj.SizeBytes, obj.ColumnCount,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create schema object: %w", err)
	}
	return id, nil
}

// GetSchemaObject 获取Schema对象
func (s *SQLiteStore) GetSchemaObject(ctx context.Context, id string) (*SchemaObjectRow, error) {
	query := `
		SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at, updated_at
		FROM schema_objects
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var obj SchemaObjectRow
	err := row.Scan(
		&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
		&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
		&obj.CreatedAt, &obj.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("schema object not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get schema object: %w", err)
	}
	return &obj, nil
}

// GetSchemaObjectByName 根据名称获取Schema对象
func (s *SQLiteStore) GetSchemaObjectByName(ctx context.Context, sourceID string, name string, schema *string) (*SchemaObjectRow, error) {
	var query string
	var row *sql.Row

	if schema != nil {
		query = `
			SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at, updated_at
			FROM schema_objects
			WHERE source_id = ? AND name = ? AND schema = ?
		`
		row = s.db.QueryRowContext(ctx, query, sourceID, name, *schema)
	} else {
		query = `
			SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at, updated_at
			FROM schema_objects
			WHERE source_id = ? AND name = ? AND schema IS NULL
		`
		row = s.db.QueryRowContext(ctx, query, sourceID, name)
	}

	var obj SchemaObjectRow
	err := row.Scan(
		&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
		&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
		&obj.CreatedAt, &obj.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get schema object by name: %w", err)
	}
	return &obj, nil
}

// ListSchemaObjectsBySource 获取数据源的所有Schema对象
func (s *SQLiteStore) ListSchemaObjectsBySource(ctx context.Context, sourceID string) ([]*SchemaObjectRow, error) {
	query := `
		SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at, updated_at
		FROM schema_objects
		WHERE source_id = ?
		ORDER BY name
	`
	rows, err := s.db.QueryContext(ctx, query, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schema objects: %w", err)
	}
	defer rows.Close()

	var objects []*SchemaObjectRow
	for rows.Next() {
		var obj SchemaObjectRow
		err := rows.Scan(
			&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
			&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
			&obj.CreatedAt, &obj.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schema object: %w", err)
		}
		objects = append(objects, &obj)
	}
	return objects, rows.Err()
}

// UpdateSchemaObject 更新Schema对象
func (s *SQLiteStore) UpdateSchemaObject(ctx context.Context, id string, updates *SchemaObjectUpdate) error {
	query := `
		UPDATE schema_objects
		SET type = ?, schema = ?, description = ?, row_count = ?, size_bytes = ?, column_count = ?, updated_at = datetime('now')
		WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query,
		updates.Type, updates.Schema, updates.Description, updates.RowCount, updates.SizeBytes, updates.ColumnCount, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update schema object: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("schema object not found: %s", id)
	}
	return nil
}

// DeleteSchemaObject 删除单个 Schema 对象
func (s *SQLiteStore) DeleteSchemaObject(ctx context.Context, id string) error {
	query := `DELETE FROM schema_objects WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schema object: %w", err)
	}
	return nil
}

// DeleteSchemaObjectsBySource 删除数据源的所有Schema对象
func (s *SQLiteStore) DeleteSchemaObjectsBySource(ctx context.Context, sourceID string) error {
	query := `DELETE FROM schema_objects WHERE source_id = ?`
	_, err := s.db.ExecContext(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete schema objects: %w", err)
	}
	return nil
}

// Transaction implementations for SQLiteTxStore

func (t *SQLiteTxStore) CreateSchemaObject(ctx context.Context, obj *SchemaObjectCreate) (string, error) {
	query := `
		INSERT INTO schema_objects (id, source_id, name, type, schema, description, row_count, size_bytes, column_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, obj.SourceID, obj.Name, obj.Type, obj.Schema,
		obj.Description, obj.RowCount, obj.SizeBytes, obj.ColumnCount,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create schema object: %w", err)
	}
	return id, nil
}

func (t *SQLiteTxStore) GetSchemaObject(ctx context.Context, id string) (*SchemaObjectRow, error) {
	query := `
		SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at, updated_at
		FROM schema_objects
		WHERE id = ?
	`
	row := t.tx.QueryRowContext(ctx, query, id)

	var obj SchemaObjectRow
	err := row.Scan(
		&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
		&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
		&obj.CreatedAt, &obj.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("schema object not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get schema object: %w", err)
	}
	return &obj, nil
}

func (t *SQLiteTxStore) GetSchemaObjectByName(ctx context.Context, sourceID string, name string, schema *string) (*SchemaObjectRow, error) {
	var query string
	var row *sql.Row

	if schema != nil {
		query = `
			SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at, updated_at
			FROM schema_objects
			WHERE source_id = ? AND name = ? AND schema = ?
		`
		row = t.tx.QueryRowContext(ctx, query, sourceID, name, *schema)
	} else {
		query = `
			SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at, updated_at
			FROM schema_objects
			WHERE source_id = ? AND name = ? AND schema IS NULL
		`
		row = t.tx.QueryRowContext(ctx, query, sourceID, name)
	}

	var obj SchemaObjectRow
	err := row.Scan(
		&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
		&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
		&obj.CreatedAt, &obj.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get schema object by name: %w", err)
	}
	return &obj, nil
}

func (t *SQLiteTxStore) ListSchemaObjectsBySource(ctx context.Context, sourceID string) ([]*SchemaObjectRow, error) {
	query := `
		SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at, updated_at
		FROM schema_objects
		WHERE source_id = ?
		ORDER BY name
	`
	rows, err := t.tx.QueryContext(ctx, query, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schema objects: %w", err)
	}
	defer rows.Close()

	var objects []*SchemaObjectRow
	for rows.Next() {
		var obj SchemaObjectRow
		err := rows.Scan(
			&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
			&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
			&obj.CreatedAt, &obj.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schema object: %w", err)
		}
		objects = append(objects, &obj)
	}
	return objects, rows.Err()
}

func (t *SQLiteTxStore) UpdateSchemaObject(ctx context.Context, id string, updates *SchemaObjectUpdate) error {
	query := `
		UPDATE schema_objects
		SET type = ?, schema = ?, description = ?, row_count = ?, size_bytes = ?, column_count = ?, updated_at = datetime('now')
		WHERE id = ?
	`
	result, err := t.tx.ExecContext(ctx, query,
		updates.Type, updates.Schema, updates.Description, updates.RowCount, updates.SizeBytes, updates.ColumnCount, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update schema object: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("schema object not found: %s", id)
	}
	return nil
}

func (t *SQLiteTxStore) DeleteSchemaObject(ctx context.Context, id string) error {
	query := `DELETE FROM schema_objects WHERE id = ?`
	_, err := t.tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schema object: %w", err)
	}
	return nil
}

func (t *SQLiteTxStore) DeleteSchemaObjectsBySource(ctx context.Context, sourceID string) error {
	query := `DELETE FROM schema_objects WHERE source_id = ?`
	_, err := t.tx.ExecContext(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete schema objects: %w", err)
	}
	return nil
}
