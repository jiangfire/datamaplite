package store

import (
	"context"
	"fmt"
	"os"

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

	if err := runPostgresMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run postgres migrations: %w", err)
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

func runPostgresMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return err
	}

	migrations := []struct {
		version string
		path    string
	}{
		{version: "001_init_schema", path: "migrations/001_init_schema.sql"},
		{version: "002_business_terms_fields", path: "migrations/002_business_terms_fields.sql"},
		{version: "003_reliability", path: "migrations/003_reliability.sql"},
	}

	for _, migration := range migrations {
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, migration.version).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}

		migrationSQL, err := os.ReadFile(migration.path)
		if err != nil {
			return err
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(migrationSQL)); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, migration.version); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
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
func (s *PostgresStore) CreateDataSource(ctx context.Context, source *DataSourceCreate) (string, error) {
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
		return "", fmt.Errorf("failed to create data source: %w", err)
	}
	return id, nil
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

// CreateUser 创建用户
func (s *PostgresStore) CreateUser(ctx context.Context, user *UserCreate) (string, error) {
	query := `
		INSERT INTO users (id, username, email, password_hash, role)
		VALUES ($1, $2, $3, $4, $5)
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query, id, user.Username, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	return id, nil
}

// GetUserByID 根据ID获取用户
func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, query, id)
	var user UserRow
	var createdAt, updatedAt interface{}
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	user.CreatedAt = fmt.Sprintf("%v", createdAt)
	user.UpdatedAt = fmt.Sprintf("%v", updatedAt)
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE username = $1
	`
	row := s.pool.QueryRow(ctx, query, username)
	var user UserRow
	var createdAt, updatedAt interface{}
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	user.CreatedAt = fmt.Sprintf("%v", createdAt)
	user.UpdatedAt = fmt.Sprintf("%v", updatedAt)
	return &user, nil
}

// ListUsers 列出所有用户
func (s *PostgresStore) ListUsers(ctx context.Context) ([]*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*UserRow
	for rows.Next() {
		var user UserRow
		var createdAt, updatedAt interface{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		user.CreatedAt = fmt.Sprintf("%v", createdAt)
		user.UpdatedAt = fmt.Sprintf("%v", updatedAt)
		users = append(users, &user)
	}
	return users, rows.Err()
}

// UpdateUser 更新用户
func (s *PostgresStore) UpdateUser(ctx context.Context, id string, updates *UserUpdate) error {
	query := `UPDATE users SET updated_at = NOW()`
	var args []interface{}
	argCount := 1

	if updates.Username != nil {
		query += fmt.Sprintf(", username = $%d", argCount)
		args = append(args, *updates.Username)
		argCount++
	}
	if updates.Email != nil {
		query += fmt.Sprintf(", email = $%d", argCount)
		args = append(args, *updates.Email)
		argCount++
	}
	if updates.PasswordHash != nil {
		query += fmt.Sprintf(", password_hash = $%d", argCount)
		args = append(args, *updates.PasswordHash)
		argCount++
	}
	if updates.Role != nil {
		query += fmt.Sprintf(", role = $%d", argCount)
		args = append(args, *updates.Role)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	result, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// DeleteUser 删除用户
func (s *PostgresStore) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// CreateDQRule 创建数据质量规则
func (s *PostgresStore) CreateDQRule(ctx context.Context, rule *DQRuleCreate) (string, error) {
	query := `
		INSERT INTO dq_rules (id, source_id, object_id, column_id, name, description, rule_type, rule_config, severity, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query,
		id, rule.SourceID, rule.ObjectID, rule.ColumnID, rule.Name,
		rule.Description, rule.RuleType, rule.RuleConfig, rule.Severity, rule.IsActive)
	if err != nil {
		return "", fmt.Errorf("failed to create dq rule: %w", err)
	}
	return id, nil
}

// GetDQRule 获取数据质量规则
func (s *PostgresStore) GetDQRule(ctx context.Context, id string) (*DQRuleRow, error) {
	query := `
		SELECT id, source_id, object_id, column_id, name, description, rule_type, rule_config::text, severity, is_active, created_at::text, updated_at::text
		FROM dq_rules WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, query, id)
	var rule DQRuleRow
	var createdAt, updatedAt interface{}
	err := row.Scan(&rule.ID, &rule.SourceID, &rule.ObjectID, &rule.ColumnID, &rule.Name,
		&rule.Description, &rule.RuleType, &rule.RuleConfig, &rule.Severity, &rule.IsActive, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("dq rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get dq rule: %w", err)
	}
	rule.CreatedAt = fmt.Sprintf("%v", createdAt)
	rule.UpdatedAt = fmt.Sprintf("%v", updatedAt)
	return &rule, nil
}

// ListDQRules 列出数据质量规则
func (s *PostgresStore) ListDQRules(ctx context.Context, filter *DQRuleFilter) ([]*DQRuleRow, error) {
	query := `
		SELECT id, source_id, object_id, column_id, name, description, rule_type, rule_config::text, severity, is_active, created_at::text, updated_at::text
		FROM dq_rules WHERE 1=1
	`
	var args []interface{}
	argCount := 0

	if filter != nil {
		if filter.SourceID != nil {
			argCount++
			query += fmt.Sprintf(" AND source_id = $%d", argCount)
			args = append(args, *filter.SourceID)
		}
		if filter.ObjectID != nil {
			argCount++
			query += fmt.Sprintf(" AND object_id = $%d", argCount)
			args = append(args, *filter.ObjectID)
		}
		if filter.ColumnID != nil {
			argCount++
			query += fmt.Sprintf(" AND column_id = $%d", argCount)
			args = append(args, *filter.ColumnID)
		}
		if filter.RuleType != nil {
			argCount++
			query += fmt.Sprintf(" AND rule_type = $%d", argCount)
			args = append(args, *filter.RuleType)
		}
		if filter.IsActive != nil {
			argCount++
			query += fmt.Sprintf(" AND is_active = $%d", argCount)
			args = append(args, *filter.IsActive)
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list dq rules: %w", err)
	}
	defer rows.Close()

	var rules []*DQRuleRow
	for rows.Next() {
		var rule DQRuleRow
		var createdAt, updatedAt interface{}
		if err := rows.Scan(&rule.ID, &rule.SourceID, &rule.ObjectID, &rule.ColumnID, &rule.Name,
			&rule.Description, &rule.RuleType, &rule.RuleConfig, &rule.Severity, &rule.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan dq rule: %w", err)
		}
		rule.CreatedAt = fmt.Sprintf("%v", createdAt)
		rule.UpdatedAt = fmt.Sprintf("%v", updatedAt)
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

// UpdateDQRule 更新数据质量规则
func (s *PostgresStore) UpdateDQRule(ctx context.Context, id string, updates *DQRuleUpdate) error {
	query := `UPDATE dq_rules SET updated_at = NOW()`
	var args []interface{}
	argCount := 1

	if updates.Name != nil {
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *updates.Name)
		argCount++
	}
	if updates.Description != nil {
		query += fmt.Sprintf(", description = $%d", argCount)
		args = append(args, *updates.Description)
		argCount++
	}
	if updates.RuleConfig != nil {
		query += fmt.Sprintf(", rule_config = $%d::jsonb", argCount)
		args = append(args, *updates.RuleConfig)
		argCount++
	}
	if updates.Severity != nil {
		query += fmt.Sprintf(", severity = $%d", argCount)
		args = append(args, *updates.Severity)
		argCount++
	}
	if updates.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argCount)
		args = append(args, *updates.IsActive)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	result, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update dq rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("dq rule not found: %s", id)
	}
	return nil
}

// DeleteDQRule 删除数据质量规则
func (s *PostgresStore) DeleteDQRule(ctx context.Context, id string) error {
	query := `DELETE FROM dq_rules WHERE id = $1`
	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dq rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("dq rule not found: %s", id)
	}
	return nil
}

// CreateDQResult 创建数据质量检测结果
func (s *PostgresStore) CreateDQResult(ctx context.Context, result *DQResultCreate) error {
	query := `
		INSERT INTO dq_results (id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
	`
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx, query,
		id, result.RuleID, result.CheckBatchID, result.ColumnID, result.Status,
		result.TotalRows, result.FailedRows, result.PassRate, result.SampleErrors, result.ErrorMessage)
	if err != nil {
		return fmt.Errorf("failed to create dq result: %w", err)
	}
	return nil
}

// ListDQResults 列出数据质量检测结果
func (s *PostgresStore) ListDQResults(ctx context.Context, filter *DQResultFilter) ([]*DQResultRow, error) {
	query := `
		SELECT id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors::text, error_message, checked_at::text
		FROM dq_results WHERE 1=1
	`
	var args []interface{}
	argCount := 0

	if filter != nil {
		if filter.RuleID != nil {
			argCount++
			query += fmt.Sprintf(" AND rule_id = $%d", argCount)
			args = append(args, *filter.RuleID)
		}
		if filter.BatchID != nil {
			argCount++
			query += fmt.Sprintf(" AND check_batch_id = $%d", argCount)
			args = append(args, *filter.BatchID)
		}
		if filter.ColumnID != nil {
			argCount++
			query += fmt.Sprintf(" AND column_id = $%d", argCount)
			args = append(args, *filter.ColumnID)
		}
		if filter.Status != nil {
			argCount++
			query += fmt.Sprintf(" AND status = $%d", argCount)
			args = append(args, *filter.Status)
		}
	}

	query += " ORDER BY checked_at DESC"

	if filter != nil && filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list dq results: %w", err)
	}
	defer rows.Close()

	var results []*DQResultRow
	for rows.Next() {
		var result DQResultRow
		var checkedAt interface{}
		if err := rows.Scan(&result.ID, &result.RuleID, &result.CheckBatchID, &result.ColumnID,
			&result.Status, &result.TotalRows, &result.FailedRows, &result.PassRate,
			&result.SampleErrors, &result.ErrorMessage, &checkedAt); err != nil {
			return nil, fmt.Errorf("failed to scan dq result: %w", err)
		}
		result.CheckedAt = fmt.Sprintf("%v", checkedAt)
		results = append(results, &result)
	}
	return results, rows.Err()
}

// GetLatestDQResult 获取最新的数据质量检测结果
func (s *PostgresStore) GetLatestDQResult(ctx context.Context, ruleID string) (*DQResultRow, error) {
	query := `
		SELECT id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors::text, error_message, checked_at::text
		FROM dq_results WHERE rule_id = $1 ORDER BY checked_at DESC LIMIT 1
	`
	row := s.pool.QueryRow(ctx, query, ruleID)
	var result DQResultRow
	var checkedAt interface{}
	err := row.Scan(&result.ID, &result.RuleID, &result.CheckBatchID, &result.ColumnID,
		&result.Status, &result.TotalRows, &result.FailedRows, &result.PassRate,
		&result.SampleErrors, &result.ErrorMessage, &checkedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest dq result: %w", err)
	}
	result.CheckedAt = fmt.Sprintf("%v", checkedAt)
	return &result, nil
}

// GetDQStats 获取数据质量统计
func (s *PostgresStore) GetDQStats(ctx context.Context) (*DQStatsRow, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM dq_rules) as total_rules,
			(SELECT COUNT(*) FROM dq_rules WHERE is_active = true) as active_rules,
			(SELECT COUNT(*) FROM dq_results) as total_checks,
			(SELECT COUNT(*) FROM dq_results WHERE status = 'passed') as passed_checks,
			(SELECT COUNT(*) FROM dq_results WHERE status IN ('failed', 'error')) as failed_checks
	`
	row := s.pool.QueryRow(ctx, query)
	var stats DQStatsRow
	err := row.Scan(&stats.TotalRules, &stats.ActiveRules, &stats.TotalChecks, &stats.PassedChecks, &stats.FailedChecks)
	if err != nil {
		return nil, fmt.Errorf("failed to get dq stats: %w", err)
	}

	if stats.TotalChecks > 0 {
		stats.OverallPassRate = float64(stats.PassedChecks) / float64(stats.TotalChecks) * 100
	}

	return &stats, nil
}
