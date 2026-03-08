package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateSchemaObject 创建Schema对象
func (s *PostgresStore) CreateSchemaObject(ctx context.Context, obj *SchemaObjectCreate) (string, error) {
	query := `
		INSERT INTO schema_objects (id, source_id, name, type, schema, description, row_count, size_bytes, column_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query,
		id, obj.SourceID, obj.Name, obj.Type, obj.Schema,
		obj.Description, obj.RowCount, obj.SizeBytes, obj.ColumnCount,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create schema object: %w", err)
	}
	return id, nil
}

// GetSchemaObject 获取Schema对象
func (s *PostgresStore) GetSchemaObject(ctx context.Context, id string) (*SchemaObjectRow, error) {
	query := `
		SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at::text, updated_at::text
		FROM schema_objects
		WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, query, id)

	var obj SchemaObjectRow
	err := row.Scan(
		&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
		&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
		&obj.CreatedAt, &obj.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("schema object not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get schema object: %w", err)
	}
	return &obj, nil
}

// GetSchemaObjectByName 根据名称获取Schema对象
func (s *PostgresStore) GetSchemaObjectByName(ctx context.Context, sourceID string, name string, schema *string) (*SchemaObjectRow, error) {
	var query string
	var row pgx.Row

	if schema != nil {
		query = `
			SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at::text, updated_at::text
			FROM schema_objects
			WHERE source_id = $1 AND name = $2 AND schema = $3
		`
		row = s.pool.QueryRow(ctx, query, sourceID, name, *schema)
	} else {
		query = `
			SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at::text, updated_at::text
			FROM schema_objects
			WHERE source_id = $1 AND name = $2 AND schema IS NULL
		`
		row = s.pool.QueryRow(ctx, query, sourceID, name)
	}

	var obj SchemaObjectRow
	err := row.Scan(
		&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
		&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
		&obj.CreatedAt, &obj.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get schema object by name: %w", err)
	}
	return &obj, nil
}

// ListSchemaObjectsBySource 获取数据源的所有Schema对象
func (s *PostgresStore) ListSchemaObjectsBySource(ctx context.Context, sourceID string) ([]*SchemaObjectRow, error) {
	query := `
		SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at::text, updated_at::text
		FROM schema_objects
		WHERE source_id = $1
		ORDER BY name
	`
	rows, err := s.pool.Query(ctx, query, sourceID)
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

// DeleteSchemaObjectsBySource 删除数据源的所有Schema对象
func (s *PostgresStore) DeleteSchemaObjectsBySource(ctx context.Context, sourceID string) error {
	query := `DELETE FROM schema_objects WHERE source_id = $1`
	_, err := s.pool.Exec(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete schema objects: %w", err)
	}
	return nil
}

// Transaction implementations for PostgresTxStore

func (t *PostgresTxStore) CreateSchemaObject(ctx context.Context, obj *SchemaObjectCreate) (string, error) {
	query := `
		INSERT INTO schema_objects (id, source_id, name, type, schema, description, row_count, size_bytes, column_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, obj.SourceID, obj.Name, obj.Type, obj.Schema,
		obj.Description, obj.RowCount, obj.SizeBytes, obj.ColumnCount,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create schema object: %w", err)
	}
	return id, nil
}

func (t *PostgresTxStore) GetSchemaObject(ctx context.Context, id string) (*SchemaObjectRow, error) {
	query := `
		SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at::text, updated_at::text
		FROM schema_objects
		WHERE id = $1
	`
	row := t.tx.QueryRow(ctx, query, id)

	var obj SchemaObjectRow
	err := row.Scan(
		&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
		&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
		&obj.CreatedAt, &obj.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("schema object not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get schema object: %w", err)
	}
	return &obj, nil
}

func (t *PostgresTxStore) GetSchemaObjectByName(ctx context.Context, sourceID string, name string, schema *string) (*SchemaObjectRow, error) {
	var query string
	var row pgx.Row

	if schema != nil {
		query = `
			SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at::text, updated_at::text
			FROM schema_objects
			WHERE source_id = $1 AND name = $2 AND schema = $3
		`
		row = t.tx.QueryRow(ctx, query, sourceID, name, *schema)
	} else {
		query = `
			SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at::text, updated_at::text
			FROM schema_objects
			WHERE source_id = $1 AND name = $2 AND schema IS NULL
		`
		row = t.tx.QueryRow(ctx, query, sourceID, name)
	}

	var obj SchemaObjectRow
	err := row.Scan(
		&obj.ID, &obj.SourceID, &obj.Name, &obj.Type, &obj.Schema,
		&obj.Description, &obj.RowCount, &obj.SizeBytes, &obj.ColumnCount,
		&obj.CreatedAt, &obj.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get schema object by name: %w", err)
	}
	return &obj, nil
}

func (t *PostgresTxStore) ListSchemaObjectsBySource(ctx context.Context, sourceID string) ([]*SchemaObjectRow, error) {
	query := `
		SELECT id, source_id, name, type, schema, description, row_count, size_bytes, column_count, created_at::text, updated_at::text
		FROM schema_objects
		WHERE source_id = $1
		ORDER BY name
	`
	rows, err := t.tx.Query(ctx, query, sourceID)
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

func (t *PostgresTxStore) DeleteSchemaObjectsBySource(ctx context.Context, sourceID string) error {
	query := `DELETE FROM schema_objects WHERE source_id = $1`
	_, err := t.tx.Exec(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete schema objects: %w", err)
	}
	return nil
}
