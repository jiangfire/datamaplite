package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CreateSchemaChange 创建变更记录
func (s *SQLiteStore) CreateSchemaChange(ctx context.Context, change *SchemaChangeCreate) error {
	query := `
		INSERT INTO schema_changes (id, source_id, object_id, change_type, object_type, object_name, old_value, new_value)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, query,
		id, change.SourceID, change.ObjectID, change.ChangeType,
		change.ObjectType, change.ObjectName, change.OldValue, change.NewValue,
	)
	if err != nil {
		return fmt.Errorf("failed to create schema change: %w", err)
	}
	return nil
}

// ListSchemaChangesBySource 获取数据源的变更记录
func (s *SQLiteStore) ListSchemaChangesBySource(ctx context.Context, sourceID string, limit int) ([]*SchemaChangeRow, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, source_id, object_id, change_type, object_type, object_name,
		       old_value, new_value, detected_at, acknowledged
		FROM schema_changes
		WHERE source_id = ?
		ORDER BY detected_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, sourceID, limit)
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

// Transaction implementations for SQLiteTxStore

func (t *SQLiteTxStore) CreateSchemaChange(ctx context.Context, change *SchemaChangeCreate) error {
	query := `
		INSERT INTO schema_changes (id, source_id, object_id, change_type, object_type, object_name, old_value, new_value)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, change.SourceID, change.ObjectID, change.ChangeType,
		change.ObjectType, change.ObjectName, change.OldValue, change.NewValue,
	)
	if err != nil {
		return fmt.Errorf("failed to create schema change: %w", err)
	}
	return nil
}

func (t *SQLiteTxStore) ListSchemaChangesBySource(ctx context.Context, sourceID string, limit int) ([]*SchemaChangeRow, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, source_id, object_id, change_type, object_type, object_name,
		       old_value, new_value, detected_at, acknowledged
		FROM schema_changes
		WHERE source_id = ?
		ORDER BY detected_at DESC
		LIMIT ?
	`
	rows, err := t.tx.QueryContext(ctx, query, sourceID, limit)
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
