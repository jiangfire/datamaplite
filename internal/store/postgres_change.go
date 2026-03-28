package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateSchemaChange 创建变更记录
func (s *PostgresStore) CreateSchemaChange(ctx context.Context, change *SchemaChangeCreate) error {
	if change.ID == "" {
		change.ID = uuid.New().String()
	}
	if change.DetectedAt == "" {
		change.DetectedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	query := `
		INSERT INTO schema_changes (id, source_id, object_id, change_type, object_type, object_name, old_value, new_value, detected_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.pool.Exec(ctx, query,
		change.ID, change.SourceID, change.ObjectID, change.ChangeType,
		change.ObjectType, change.ObjectName, change.OldValue, change.NewValue, change.DetectedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create schema change: %w", err)
	}
	return nil
}

// ListSchemaChangesBySource 获取数据源的变更记录
func (s *PostgresStore) ListSchemaChangesBySource(ctx context.Context, sourceID string, limit int) ([]*SchemaChangeRow, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, source_id, object_id, change_type, object_type, object_name,
		       old_value, new_value, detected_at::text, acknowledged
		FROM schema_changes
		WHERE source_id = $1
		ORDER BY detected_at DESC
		LIMIT $2
	`
	rows, err := s.pool.Query(ctx, query, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list schema changes: %w", err)
	}
	defer rows.Close()

	var changes []*SchemaChangeRow
	for rows.Next() {
		var change SchemaChangeRow
		err := rows.Scan(
			&change.ID, &change.SourceID, &change.ObjectID, &change.ChangeType,
			&change.ObjectType, &change.ObjectName, &change.OldValue, &change.NewValue,
			&change.DetectedAt, &change.Acknowledged,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schema change: %w", err)
		}
		changes = append(changes, &change)
	}
	return changes, rows.Err()
}

// Transaction implementations for PostgresTxStore

func (t *PostgresTxStore) CreateSchemaChange(ctx context.Context, change *SchemaChangeCreate) error {
	if change.ID == "" {
		change.ID = uuid.New().String()
	}
	if change.DetectedAt == "" {
		change.DetectedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	query := `
		INSERT INTO schema_changes (id, source_id, object_id, change_type, object_type, object_name, old_value, new_value, detected_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := t.tx.Exec(ctx, query,
		change.ID, change.SourceID, change.ObjectID, change.ChangeType,
		change.ObjectType, change.ObjectName, change.OldValue, change.NewValue, change.DetectedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create schema change: %w", err)
	}
	return nil
}

func (t *PostgresTxStore) ListSchemaChangesBySource(ctx context.Context, sourceID string, limit int) ([]*SchemaChangeRow, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, source_id, object_id, change_type, object_type, object_name,
		       old_value, new_value, detected_at::text, acknowledged
		FROM schema_changes
		WHERE source_id = $1
		ORDER BY detected_at DESC
		LIMIT $2
	`
	rows, err := t.tx.Query(ctx, query, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list schema changes: %w", err)
	}
	defer rows.Close()

	var changes []*SchemaChangeRow
	for rows.Next() {
		var change SchemaChangeRow
		err := rows.Scan(
			&change.ID, &change.SourceID, &change.ObjectID, &change.ChangeType,
			&change.ObjectType, &change.ObjectName, &change.OldValue, &change.NewValue,
			&change.DetectedAt, &change.Acknowledged,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schema change: %w", err)
		}
		changes = append(changes, &change)
	}
	return changes, rows.Err()
}
