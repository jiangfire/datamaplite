package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WithTx 嵌套事务支持
func (t *PostgresTxStore) WithTx(ctx context.Context, fn func(Store) error) error {
	// 嵌套事务不开启新事务，直接使用当前事务
	return fn(t)
}

// Close 关闭事务存储（空实现，因为事务由上层管理）
func (t *PostgresTxStore) Close() error {
	return nil
}

// CreateDataSource 创建数据源
func (t *PostgresTxStore) CreateDataSource(ctx context.Context, source *DataSourceCreate) error {
	query := `
		INSERT INTO data_sources (id, name, description, type, host, port, database, connection_config, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active')
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, source.Name, source.Description, source.Type, source.Host,
		source.Port, source.Database, source.ConnectionConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create data source: %w", err)
	}
	return nil
}

// GetDataSource 获取数据源
func (t *PostgresTxStore) GetDataSource(ctx context.Context, id string) (*DataSourceRow, error) {
	query := `
		SELECT id, name, description, type, host, port, database, connection_config, status,
		       last_sync_at::text, last_sync_error, created_at::text, updated_at::text
		FROM data_sources
		WHERE id = $1
	`
	row := t.tx.QueryRow(ctx, query, id)

	var ds DataSourceRow
	err := row.Scan(
		&ds.ID, &ds.Name, &ds.Description, &ds.Type, &ds.Host,
		&ds.Port, &ds.Database, &ds.ConnectionConfig, &ds.Status,
		&ds.LastSyncAt, &ds.LastSyncError, &ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("data source not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}
	return &ds, nil
}

// ListDataSources 列出所有数据源
func (t *PostgresTxStore) ListDataSources(ctx context.Context) ([]*DataSourceRow, error) {
	query := `
		SELECT id, name, description, type, host, port, database, connection_config, status,
		       last_sync_at::text, last_sync_error, created_at::text, updated_at::text
		FROM data_sources
		ORDER BY created_at DESC
	`
	rows, err := t.tx.Query(ctx, query)
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
func (t *PostgresTxStore) UpdateDataSource(ctx context.Context, id string, updates *DataSourceUpdate) error {
	query := `
		UPDATE data_sources
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    host = COALESCE($3, host),
		    port = COALESCE($4, port),
		    database = COALESCE($5, database),
		    connection_config = COALESCE($6, connection_config),
		    status = COALESCE($7, status),
		    updated_at = NOW()
		WHERE id = $8
	`
	result, err := t.tx.Exec(ctx, query,
		updates.Name, updates.Description, updates.Host, updates.Port,
		updates.Database, updates.ConnectionConfig, updates.Status, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update data source: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}

// DeleteDataSource 删除数据源
func (t *PostgresTxStore) DeleteDataSource(ctx context.Context, id string) error {
	// 删除变更记录
	_, err := t.tx.Exec(ctx, `DELETE FROM schema_changes WHERE source_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete schema changes: %w", err)
	}

	// 删除字段和对象（通过级联删除）
	_, err = t.tx.Exec(ctx, `DELETE FROM schema_objects WHERE source_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete schema objects: %w", err)
	}

	result, err := t.tx.Exec(ctx, `DELETE FROM data_sources WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}

// UpdateDataSourceSyncStatus 更新同步状态
func (t *PostgresTxStore) UpdateDataSourceSyncStatus(ctx context.Context, id string, status string, errorMsg *string) error {
	var query string
	var args []interface{}

	if errorMsg != nil {
		query = `UPDATE data_sources SET status = $1, last_sync_at = NOW(), last_sync_error = $2, updated_at = NOW() WHERE id = $3`
		args = []interface{}{status, errorMsg, id}
	} else {
		query = `UPDATE data_sources SET status = $1, last_sync_at = NOW(), last_sync_error = NULL, updated_at = NOW() WHERE id = $2`
		args = []interface{}{status, id}
	}

	result, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}
