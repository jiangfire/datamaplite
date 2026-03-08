package store

import (
	"context"
	"fmt"

	"git.neolidy.top/neo/fuckcmdb/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PostgresStore PostgreSQL存储实现
type PostgresStore struct {
	pool *pgxpool.Pool
	log  *zap.Logger
}

// NewPostgresStore 创建PostgreSQL存储实例
func NewPostgresStore(ctx context.Context, cfg *config.DatabaseConfig, log *zap.Logger) (*PostgresStore, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("PostgreSQL connected",
		zap.String("host", cfg.Host),
		zap.Int32("max_conns", cfg.MaxConns),
		zap.Int32("min_conns", cfg.MinConns),
	)

	return &PostgresStore{
		pool: pool,
		log:  log,
	}, nil
}

// Close 关闭存储连接
func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

// WithTx 在事务中执行
func (s *PostgresStore) WithTx(ctx context.Context, fn func(Store) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	txStore := &PostgresTxStore{tx: tx, log: s.log}
	if err := fn(txStore); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// PostgresTxStore PostgreSQL事务存储
type PostgresTxStore struct {
	tx  pgx.Tx
	log *zap.Logger
}

// CreateDataSource 创建数据源
func (s *PostgresStore) CreateDataSource(ctx context.Context, source *DataSourceCreate) error {
	query := `
		INSERT INTO data_sources (id, name, description, type, host, port, database, connection_config, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active')
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query,
		id, source.Name, source.Description, source.Type, source.Host,
		source.Port, source.Database, source.ConnectionConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create data source: %w", err)
	}
	return nil
}

// GetDataSource 获取数据源
func (s *PostgresStore) GetDataSource(ctx context.Context, id string) (*DataSourceRow, error) {
	query := `
		SELECT id, name, description, type, host, port, database, connection_config, status,
		       last_sync_at::text, last_sync_error, created_at::text, updated_at::text
		FROM data_sources
		WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, query, id)

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
func (s *PostgresStore) ListDataSources(ctx context.Context) ([]*DataSourceRow, error) {
	query := `
		SELECT id, name, description, type, host, port, database, connection_config, status,
		       last_sync_at::text, last_sync_error, created_at::text, updated_at::text
		FROM data_sources
		ORDER BY created_at DESC
	`
	rows, err := s.pool.Query(ctx, query)
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
func (s *PostgresStore) UpdateDataSource(ctx context.Context, id string, updates *DataSourceUpdate) error {
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
	result, err := s.pool.Exec(ctx, query,
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
func (s *PostgresStore) DeleteDataSource(ctx context.Context, id string) error {
	// 首先删除关联的变更记录和字段
	if err := s.deleteSourceRelatedData(ctx, id); err != nil {
		return err
	}

	query := `DELETE FROM data_sources WHERE id = $1`
	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}

// deleteSourceRelatedData 删除数据源相关的数据
func (s *PostgresStore) deleteSourceRelatedData(ctx context.Context, sourceID string) error {
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
	query := `DELETE FROM schema_changes WHERE source_id = $1`
	_, err = s.pool.Exec(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete schema changes: %w", err)
	}

	return nil
}

// getObjectIDsBySource 获取数据源的所有对象ID
func (s *PostgresStore) getObjectIDsBySource(ctx context.Context, sourceID string) ([]string, error) {
	query := `SELECT id FROM schema_objects WHERE source_id = $1`
	rows, err := s.pool.Query(ctx, query, sourceID)
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
func (s *PostgresStore) UpdateDataSourceSyncStatus(ctx context.Context, id string, status string, errorMsg *string) error {
	var query string
	var args []interface{}

	if errorMsg != nil {
		query = `UPDATE data_sources SET status = $1, last_sync_at = NOW(), last_sync_error = $2, updated_at = NOW() WHERE id = $3`
		args = []interface{}{status, errorMsg, id}
	} else {
		query = `UPDATE data_sources SET status = $1, last_sync_at = NOW(), last_sync_error = NULL, updated_at = NOW() WHERE id = $2`
		args = []interface{}{status, id}
	}

	result, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}
	return nil
}
