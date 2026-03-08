package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// CreateDataSource 创建数据源
func (s *SQLiteStore) CreateDataSource(ctx context.Context, source *DataSourceCreate) error {
	query := `
		INSERT INTO data_sources (id, name, description, type, host, port, database, connection_config, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'active')
	`
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, query,
		id, source.Name, source.Description, source.Type, source.Host,
		source.Port, source.Database, source.ConnectionConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create data source: %w", err)
	}
	return nil
}

// GetDataSource 获取数据源
func (s *SQLiteStore) GetDataSource(ctx context.Context, id string) (*DataSourceRow, error) {
	query := `
		SELECT id, name, description, type, host, port, database, connection_config, status,
		       last_sync_at, last_sync_error, created_at, updated_at
		FROM data_sources
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var ds DataSourceRow
	err := row.Scan(
		&ds.ID, &ds.Name, &ds.Description, &ds.Type, &ds.Host,
		&ds.Port, &ds.Database, &ds.ConnectionConfig, &ds.Status,
		&ds.LastSyncAt, &ds.LastSyncError, &ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("data source not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}
	return &ds, nil
}

// ListDataSources 列出所有数据源
func (s *SQLiteStore) ListDataSources(ctx context.Context) ([]*DataSourceRow, error) {
	query := `
		SELECT id, name, description, type, host, port, database, connection_config, status,
		       last_sync_at, last_sync_error, created_at, updated_at
		FROM data_sources
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list data sources: %w", err)
	}
	defer rows.Close()

	var sources []*DataSourceRow
	for rows.Next() {
		var ds DataSourceRow
		err := rows.Scan(
			&ds.ID, &ds.Name, &ds.Description, &ds.Type, &ds.Host,
			&ds.Port, &ds.Database, &ds.ConnectionConfig, &ds.Status,
			&ds.LastSyncAt, &ds.LastSyncError, &ds.CreatedAt, &ds.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data source: %w", err)
		}
		sources = append(sources, &ds)
	}
	return sources, rows.Err()
}

// UpdateDataSource 更新数据源
func (s *SQLiteStore) UpdateDataSource(ctx context.Context, id string, updates *DataSourceUpdate) error {
	query := `
		UPDATE data_sources
		SET name = COALESCE(?, name),
		    description = COALESCE(?, description),
		    host = COALESCE(?, host),
		    port = COALESCE(?, port),
		    database = COALESCE(?, database),
		    connection_config = COALESCE(?, connection_config),
		    status = COALESCE(?, status),
		    updated_at = datetime('now')
		WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query,
		updates.Name, updates.Description, updates.Host, updates.Port,
		updates.Database, updates.ConnectionConfig, updates.Status, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update data source: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}

// DeleteDataSource 删除数据源
func (s *SQLiteStore) DeleteDataSource(ctx context.Context, id string) error {
	// 首先删除关联的变更记录和字段
	if err := s.deleteSourceRelatedData(ctx, id); err != nil {
		return err
	}

	query := `DELETE FROM data_sources WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}

// deleteSourceRelatedData 删除数据源相关的数据
func (s *SQLiteStore) deleteSourceRelatedData(ctx context.Context, sourceID string) error {
	// 获取所有对象ID
	objectIDs, err := s.getObjectIDsBySource(ctx, sourceID)
	if err != nil {
		return err
	}

	// 删除所有字段
	for _, objID := range objectIDs {
		if err := s.DeleteColumnsByObject(ctx, objID); err != nil {
			return err
		}
	}

	// 删除所有对象
	if err := s.DeleteSchemaObjectsBySource(ctx, sourceID); err != nil {
		return err
	}

	// 删除变更记录
	query := `DELETE FROM schema_changes WHERE source_id = ?`
	_, err = s.db.ExecContext(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete schema changes: %w", err)
	}

	return nil
}

// getObjectIDsBySource 获取数据源的所有对象ID
func (s *SQLiteStore) getObjectIDsBySource(ctx context.Context, sourceID string) ([]string, error) {
	query := `SELECT id FROM schema_objects WHERE source_id = ?`
	rows, err := s.db.QueryContext(ctx, query, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list object ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan object id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// UpdateDataSourceSyncStatus 更新同步状态
func (s *SQLiteStore) UpdateDataSourceSyncStatus(ctx context.Context, id string, status string, errorMsg *string) error {
	var query string
	var args []interface{}

	if errorMsg != nil {
		query = `UPDATE data_sources SET status = ?, last_sync_at = datetime('now'), last_sync_error = ?, updated_at = datetime('now') WHERE id = ?`
		args = []interface{}{status, errorMsg, id}
	} else {
		query = `UPDATE data_sources SET status = ?, last_sync_at = datetime('now'), last_sync_error = NULL, updated_at = datetime('now') WHERE id = ?`
		args = []interface{}{status, id}
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}

// Transaction implementations for SQLiteTxStore

func (t *SQLiteTxStore) CreateDataSource(ctx context.Context, source *DataSourceCreate) error {
	query := `
		INSERT INTO data_sources (id, name, description, type, host, port, database, connection_config, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'active')
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, source.Name, source.Description, source.Type, source.Host,
		source.Port, source.Database, source.ConnectionConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create data source: %w", err)
	}
	return nil
}

func (t *SQLiteTxStore) GetDataSource(ctx context.Context, id string) (*DataSourceRow, error) {
	query := `
		SELECT id, name, description, type, host, port, database, connection_config, status,
		       last_sync_at, last_sync_error, created_at, updated_at
		FROM data_sources
		WHERE id = ?
	`
	row := t.tx.QueryRowContext(ctx, query, id)

	var ds DataSourceRow
	err := row.Scan(
		&ds.ID, &ds.Name, &ds.Description, &ds.Type, &ds.Host,
		&ds.Port, &ds.Database, &ds.ConnectionConfig, &ds.Status,
		&ds.LastSyncAt, &ds.LastSyncError, &ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("data source not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}
	return &ds, nil
}

func (t *SQLiteTxStore) ListDataSources(ctx context.Context) ([]*DataSourceRow, error) {
	query := `
		SELECT id, name, description, type, host, port, database, connection_config, status,
		       last_sync_at, last_sync_error, created_at, updated_at
		FROM data_sources
		ORDER BY created_at DESC
	`
	rows, err := t.tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list data sources: %w", err)
	}
	defer rows.Close()

	var sources []*DataSourceRow
	for rows.Next() {
		var ds DataSourceRow
		err := rows.Scan(
			&ds.ID, &ds.Name, &ds.Description, &ds.Type, &ds.Host,
			&ds.Port, &ds.Database, &ds.ConnectionConfig, &ds.Status,
			&ds.LastSyncAt, &ds.LastSyncError, &ds.CreatedAt, &ds.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data source: %w", err)
		}
		sources = append(sources, &ds)
	}
	return sources, rows.Err()
}

func (t *SQLiteTxStore) UpdateDataSource(ctx context.Context, id string, updates *DataSourceUpdate) error {
	query := `
		UPDATE data_sources
		SET name = COALESCE(?, name),
		    description = COALESCE(?, description),
		    host = COALESCE(?, host),
		    port = COALESCE(?, port),
		    database = COALESCE(?, database),
		    connection_config = COALESCE(?, connection_config),
		    status = COALESCE(?, status),
		    updated_at = datetime('now')
		WHERE id = ?
	`
	result, err := t.tx.ExecContext(ctx, query,
		updates.Name, updates.Description, updates.Host, updates.Port,
		updates.Database, updates.ConnectionConfig, updates.Status, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update data source: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}

func (t *SQLiteTxStore) DeleteDataSource(ctx context.Context, id string) error {
	// 删除变更记录
	_, err := t.tx.ExecContext(ctx, `DELETE FROM schema_changes WHERE source_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete schema changes: %w", err)
	}

	// 删除字段和对象（通过级联删除）
	_, err = t.tx.ExecContext(ctx, `DELETE FROM schema_objects WHERE source_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete schema objects: %w", err)
	}

	result, err := t.tx.ExecContext(ctx, `DELETE FROM data_sources WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}

func (t *SQLiteTxStore) UpdateDataSourceSyncStatus(ctx context.Context, id string, status string, errorMsg *string) error {
	var query string
	var args []interface{}

	if errorMsg != nil {
		query = `UPDATE data_sources SET status = ?, last_sync_at = datetime('now'), last_sync_error = ?, updated_at = datetime('now') WHERE id = ?`
		args = []interface{}{status, errorMsg, id}
	} else {
		query = `UPDATE data_sources SET status = ?, last_sync_at = datetime('now'), last_sync_error = NULL, updated_at = datetime('now') WHERE id = ?`
		args = []interface{}{status, id}
	}

	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}
