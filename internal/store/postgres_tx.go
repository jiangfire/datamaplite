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

// CreateUser 创建用户
func (t *PostgresTxStore) CreateUser(ctx context.Context, user *UserCreate) (string, error) {
	query := `
		INSERT INTO users (id, username, email, password_hash, role)
		VALUES ($1, $2, $3, $4, $5)
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query, id, user.Username, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	return id, nil
}

// GetUserByID 根据ID获取用户
func (t *PostgresTxStore) GetUserByID(ctx context.Context, id string) (*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at::text, updated_at::text
		FROM users WHERE id = $1
	`
	row := t.tx.QueryRow(ctx, query, id)
	var user UserRow
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (t *PostgresTxStore) GetUserByUsername(ctx context.Context, username string) (*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at::text, updated_at::text
		FROM users WHERE username = $1
	`
	row := t.tx.QueryRow(ctx, query, username)
	var user UserRow
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// ListUsers 列出所有用户
func (t *PostgresTxStore) ListUsers(ctx context.Context) ([]*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at::text, updated_at::text
		FROM users ORDER BY created_at DESC
	`
	rows, err := t.tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*UserRow
	for rows.Next() {
		var user UserRow
		if err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
			&user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

// UpdateUser 更新用户
func (t *PostgresTxStore) UpdateUser(ctx context.Context, id string, updates *UserUpdate) error {
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

	result, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// DeleteUser 删除用户
func (t *PostgresTxStore) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := t.tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// CreateDQRule 创建数据质量规则
func (t *PostgresTxStore) CreateDQRule(ctx context.Context, rule *DQRuleCreate) (string, error) {
	query := `
		INSERT INTO dq_rules (id, source_id, object_id, column_id, name, description, rule_type, rule_config, severity, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, rule.SourceID, rule.ObjectID, rule.ColumnID, rule.Name,
		rule.Description, rule.RuleType, rule.RuleConfig, rule.Severity, rule.IsActive)
	if err != nil {
		return "", fmt.Errorf("failed to create dq rule: %w", err)
	}
	return id, nil
}

// GetDQRule 获取数据质量规则
func (t *PostgresTxStore) GetDQRule(ctx context.Context, id string) (*DQRuleRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// ListDQRules 列出数据质量规则
func (t *PostgresTxStore) ListDQRules(ctx context.Context, filter *DQRuleFilter) ([]*DQRuleRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// UpdateDQRule 更新数据质量规则
func (t *PostgresTxStore) UpdateDQRule(ctx context.Context, id string, updates *DQRuleUpdate) error {
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

	result, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update dq rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("dq rule not found: %s", id)
	}
	return nil
}

// DeleteDQRule 删除数据质量规则
func (t *PostgresTxStore) DeleteDQRule(ctx context.Context, id string) error {
	query := `DELETE FROM dq_rules WHERE id = $1`
	result, err := t.tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dq rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("dq rule not found: %s", id)
	}
	return nil
}

// CreateDQResult 创建数据质量检测结果
func (t *PostgresTxStore) CreateDQResult(ctx context.Context, result *DQResultCreate) error {
	query := `
		INSERT INTO dq_results (id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
	`
	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, result.RuleID, result.CheckBatchID, result.ColumnID, result.Status,
		result.TotalRows, result.FailedRows, result.PassRate, result.SampleErrors, result.ErrorMessage)
	if err != nil {
		return fmt.Errorf("failed to create dq result: %w", err)
	}
	return nil
}

// ListDQResults 列出数据质量检测结果
func (t *PostgresTxStore) ListDQResults(ctx context.Context, filter *DQResultFilter) ([]*DQResultRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// GetLatestDQResult 获取最新的数据质量检测结果
func (t *PostgresTxStore) GetLatestDQResult(ctx context.Context, ruleID string) (*DQResultRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// GetDQStats 获取数据质量统计
func (t *PostgresTxStore) GetDQStats(ctx context.Context) (*DQStatsRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}
