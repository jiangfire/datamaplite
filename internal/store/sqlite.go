package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"git.neolidy.top/neo/fuckcmdb/internal/config"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// SQLiteStore SQLite存储实现
type SQLiteStore struct {
	db  *sql.DB
	log *zap.Logger
}

// NewSQLiteStore 创建SQLite存储实例
func NewSQLiteStore(ctx context.Context, cfg *config.DatabaseConfig, log *zap.Logger) (*SQLiteStore, error) {
	// 确保数据目录存在
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "datamap.db")
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 运行迁移
	if err := runSQLiteMigrations(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info("SQLite connected", zap.String("path", dbPath))

	return &SQLiteStore{
		db:  db,
		log: log,
	}, nil
}

// Close 关闭存储连接
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateUser 创建用户
func (s *SQLiteStore) CreateUser(ctx context.Context, user *UserCreate) (string, error) {
	query := `
		INSERT INTO users (id, username, email, password_hash, role)
		VALUES (?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, query, id, user.Username, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	return id, nil
}

// GetUserByID 根据ID获取用户
func (s *SQLiteStore) GetUserByID(ctx context.Context, id string) (*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	var user UserRow
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *SQLiteStore) GetUserByUsername(ctx context.Context, username string) (*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE username = ?
	`
	row := s.db.QueryRowContext(ctx, query, username)
	var user UserRow
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// ListUsers 列出所有用户
func (s *SQLiteStore) ListUsers(ctx context.Context) ([]*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
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
func (s *SQLiteStore) UpdateUser(ctx context.Context, id string, updates *UserUpdate) error {
	query := `UPDATE users SET updated_at = datetime('now')`
	var args []interface{}

	if updates.Username != nil {
		query += ", username = ?"
		args = append(args, *updates.Username)
	}
	if updates.Email != nil {
		query += ", email = ?"
		args = append(args, *updates.Email)
	}
	if updates.PasswordHash != nil {
		query += ", password_hash = ?"
		args = append(args, *updates.PasswordHash)
	}
	if updates.Role != nil {
		query += ", role = ?"
		args = append(args, *updates.Role)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// DeleteUser 删除用户
func (s *SQLiteStore) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// WithTx 在事务中执行
func (s *SQLiteStore) WithTx(ctx context.Context, fn func(Store) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback()

	txStore := &SQLiteTxStore{tx: tx, log: s.log}
	if err := fn(txStore); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// runSQLiteMigrations 运行SQLite数据库迁移
func runSQLiteMigrations(ctx context.Context, db *sql.DB) error {
	// 读取迁移文件
	migrationSQL, err := os.ReadFile("migrations/001_init_schema.sql")
	if err != nil {
		// 如果文件不存在，使用内置迁移
		return runSQLiteBuiltinMigrations(ctx, db)
	}

	_, err = db.ExecContext(ctx, string(migrationSQL))
	return err
}

// runSQLiteBuiltinMigrations 运行内置迁移（SQLite适配版）
func runSQLiteBuiltinMigrations(ctx context.Context, db *sql.DB) error {
	schema := `
-- SQLite 适配的数据库Schema

-- 1. 数据源表
CREATE TABLE IF NOT EXISTS data_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL CHECK (type IN ('mysql', 'postgres', 'mongodb', 'oracle', 'mssql')),
    host TEXT NOT NULL,
    port INTEGER NOT NULL CHECK (port > 0 AND port <= 65535),
    database TEXT NOT NULL,
    connection_config TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'error', 'syncing')),
    last_sync_at TEXT,
    last_sync_error TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_data_sources_type ON data_sources(type);
CREATE INDEX IF NOT EXISTS idx_data_sources_status ON data_sources(status);

-- 2. 业务术语表
CREATE TABLE IF NOT EXISTS business_terms (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    category TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_business_terms_category ON business_terms(category);

-- 3. Schema对象表
CREATE TABLE IF NOT EXISTS schema_objects (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('table', 'view', 'collection')),
    schema TEXT,
    description TEXT,
    row_count INTEGER,
    size_bytes INTEGER,
    column_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(source_id, name, schema)
);

CREATE INDEX IF NOT EXISTS idx_schema_objects_source ON schema_objects(source_id);
CREATE INDEX IF NOT EXISTS idx_schema_objects_name ON schema_objects(name);
CREATE INDEX IF NOT EXISTS idx_schema_objects_type ON schema_objects(type);

-- 4. 字段表
CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY,
    object_id TEXT NOT NULL REFERENCES schema_objects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    data_type TEXT NOT NULL,
    full_data_type TEXT,
    is_nullable INTEGER NOT NULL DEFAULT 1,
    default_value TEXT,
    is_primary_key INTEGER NOT NULL DEFAULT 0,
    is_unique INTEGER NOT NULL DEFAULT 0,
    ordinal_position INTEGER NOT NULL,
    description TEXT,
    term_id TEXT REFERENCES business_terms(id),
    confidence REAL DEFAULT 1.0,
    parent_column_path TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(object_id, name)
);

CREATE INDEX IF NOT EXISTS idx_columns_object ON columns(object_id);
CREATE INDEX IF NOT EXISTS idx_columns_term ON columns(term_id);
CREATE INDEX IF NOT EXISTS idx_columns_name ON columns(name);

-- 5. 字段映射表
CREATE TABLE IF NOT EXISTS column_mappings (
    id TEXT PRIMARY KEY,
    source_column_id TEXT NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    target_column_id TEXT NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    mapping_type TEXT NOT NULL DEFAULT 'alias' CHECK (mapping_type IN ('alias', 'transform', 'derived', 'synonym')),
    confidence REAL DEFAULT 1.0,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(source_column_id, target_column_id)
);

CREATE INDEX IF NOT EXISTS idx_mappings_source ON column_mappings(source_column_id);
CREATE INDEX IF NOT EXISTS idx_mappings_target ON column_mappings(target_column_id);

-- 6. 数据血缘表
CREATE TABLE IF NOT EXISTS lineage_edges (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('column', 'object')),
    target_type TEXT NOT NULL CHECK (target_type IN ('column', 'object')),
    transform_sql TEXT,
    job_name TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_lineage_source ON lineage_edges(source_id, source_type);
CREATE INDEX IF NOT EXISTS idx_lineage_target ON lineage_edges(target_id, target_type);

-- 7. Schema变更审计表
CREATE TABLE IF NOT EXISTS schema_changes (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    object_id TEXT REFERENCES schema_objects(id) ON DELETE CASCADE,
    change_type TEXT NOT NULL CHECK (change_type IN ('add_object', 'drop_object', 'add_column', 'drop_column', 'alter_column', 'change_type')),
    object_type TEXT NOT NULL CHECK (object_type IN ('column', 'object')),
    object_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    detected_at TEXT DEFAULT (datetime('now')),
    acknowledged INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_schema_changes_source ON schema_changes(source_id);
CREATE INDEX IF NOT EXISTS idx_schema_changes_detected ON schema_changes(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_schema_changes_ack ON schema_changes(acknowledged);

-- 8. 用户表
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'user')) DEFAULT 'user',
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 插入默认管理员用户 (密码: admin123)
INSERT OR IGNORE INTO users (id, username, email, password_hash, role)
VALUES ('admin-001', 'admin', 'admin@datamap.local', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin');

-- 9. 数据质量规则表
CREATE TABLE IF NOT EXISTS dq_rules (
    id TEXT PRIMARY KEY,
    source_id TEXT REFERENCES data_sources(id) ON DELETE CASCADE,
    object_id TEXT REFERENCES schema_objects(id) ON DELETE CASCADE,
    column_id TEXT REFERENCES columns(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    rule_type TEXT NOT NULL CHECK (rule_type IN ('not_null', 'unique', 'regex', 'range', 'enum', 'custom_sql', 'referential')),
    rule_config TEXT NOT NULL DEFAULT '{}',
    severity TEXT NOT NULL DEFAULT 'error' CHECK (severity IN ('error', 'warning', 'info')),
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_dq_rules_source ON dq_rules(source_id);
CREATE INDEX IF NOT EXISTS idx_dq_rules_object ON dq_rules(object_id);
CREATE INDEX IF NOT EXISTS idx_dq_rules_column ON dq_rules(column_id);
CREATE INDEX IF NOT EXISTS idx_dq_rules_type ON dq_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_dq_rules_active ON dq_rules(is_active);

-- 10. 数据质量检测结果表
CREATE TABLE IF NOT EXISTS dq_results (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL REFERENCES dq_rules(id) ON DELETE CASCADE,
    check_batch_id TEXT NOT NULL,
    column_id TEXT REFERENCES columns(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('passed', 'failed', 'error')),
    total_rows INTEGER NOT NULL DEFAULT 0,
    failed_rows INTEGER NOT NULL DEFAULT 0,
    pass_rate REAL NOT NULL DEFAULT 0.00,
    sample_errors TEXT DEFAULT '[]',
    error_message TEXT,
    checked_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_dq_results_rule ON dq_results(rule_id);
CREATE INDEX IF NOT EXISTS idx_dq_results_batch ON dq_results(check_batch_id);
CREATE INDEX IF NOT EXISTS idx_dq_results_status ON dq_results(status);
CREATE INDEX IF NOT EXISTS idx_dq_results_checked_at ON dq_results(checked_at);

-- 11. 标签表
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#6366f1',
    description TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);

-- 12. 字段标签关联表
CREATE TABLE IF NOT EXISTS column_tags (
    id TEXT PRIMARY KEY,
    column_id TEXT NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(column_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_column_tags_column ON column_tags(column_id);
CREATE INDEX IF NOT EXISTS idx_column_tags_tag ON column_tags(tag_id);
`
	_, err := db.ExecContext(ctx, schema)
	return err
}

// SQLiteTxStore SQLite事务存储
type SQLiteTxStore struct {
	tx  *sql.Tx
	log *zap.Logger
}

// WithTx 嵌套事务支持
func (t *SQLiteTxStore) WithTx(ctx context.Context, fn func(Store) error) error {
	return fn(t)
}

// Close 关闭事务存储
func (t *SQLiteTxStore) Close() error {
	return nil
}

// CreateUser 创建用户
func (t *SQLiteTxStore) CreateUser(ctx context.Context, user *UserCreate) (string, error) {
	query := `
		INSERT INTO users (id, username, email, password_hash, role)
		VALUES (?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query, id, user.Username, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	return id, nil
}

// GetUserByID 根据ID获取用户
func (t *SQLiteTxStore) GetUserByID(ctx context.Context, id string) (*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE id = ?
	`
	row := t.tx.QueryRowContext(ctx, query, id)
	var user UserRow
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (t *SQLiteTxStore) GetUserByUsername(ctx context.Context, username string) (*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE username = ?
	`
	row := t.tx.QueryRowContext(ctx, query, username)
	var user UserRow
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// ListUsers 列出所有用户
func (t *SQLiteTxStore) ListUsers(ctx context.Context) ([]*UserRow, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`
	rows, err := t.tx.QueryContext(ctx, query)
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
func (t *SQLiteTxStore) UpdateUser(ctx context.Context, id string, updates *UserUpdate) error {
	query := `UPDATE users SET updated_at = datetime('now')`
	var args []interface{}

	if updates.Username != nil {
		query += ", username = ?"
		args = append(args, *updates.Username)
	}
	if updates.Email != nil {
		query += ", email = ?"
		args = append(args, *updates.Email)
	}
	if updates.PasswordHash != nil {
		query += ", password_hash = ?"
		args = append(args, *updates.PasswordHash)
	}
	if updates.Role != nil {
		query += ", role = ?"
		args = append(args, *updates.Role)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// DeleteUser 删除用户
func (t *SQLiteTxStore) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = ?`
	result, err := t.tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", id)
	}
	return nil
}

// CreateDQRule 创建数据质量规则
func (s *SQLiteStore) CreateDQRule(ctx context.Context, rule *DQRuleCreate) (string, error) {
	query := `
		INSERT INTO dq_rules (id, source_id, object_id, column_id, name, description, rule_type, rule_config, severity, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, query,
		id, rule.SourceID, rule.ObjectID, rule.ColumnID, rule.Name,
		rule.Description, rule.RuleType, rule.RuleConfig, rule.Severity, rule.IsActive)
	if err != nil {
		return "", fmt.Errorf("failed to create dq rule: %w", err)
	}
	return id, nil
}

// GetDQRule 获取数据质量规则
func (s *SQLiteStore) GetDQRule(ctx context.Context, id string) (*DQRuleRow, error) {
	query := `
		SELECT id, source_id, object_id, column_id, name, description, rule_type, rule_config, severity, is_active, created_at, updated_at
		FROM dq_rules WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	var rule DQRuleRow
	err := row.Scan(
		&rule.ID, &rule.SourceID, &rule.ObjectID, &rule.ColumnID, &rule.Name,
		&rule.Description, &rule.RuleType, &rule.RuleConfig, &rule.Severity,
		&rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dq rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get dq rule: %w", err)
	}
	return &rule, nil
}

// ListDQRules 列出数据质量规则
func (s *SQLiteStore) ListDQRules(ctx context.Context, filter *DQRuleFilter) ([]*DQRuleRow, error) {
	query := `
		SELECT id, source_id, object_id, column_id, name, description, rule_type, rule_config, severity, is_active, created_at, updated_at
		FROM dq_rules WHERE 1=1
	`
	var args []interface{}

	if filter != nil {
		if filter.SourceID != nil {
			query += " AND source_id = ?"
			args = append(args, *filter.SourceID)
		}
		if filter.ObjectID != nil {
			query += " AND object_id = ?"
			args = append(args, *filter.ObjectID)
		}
		if filter.ColumnID != nil {
			query += " AND column_id = ?"
			args = append(args, *filter.ColumnID)
		}
		if filter.RuleType != nil {
			query += " AND rule_type = ?"
			args = append(args, *filter.RuleType)
		}
		if filter.IsActive != nil {
			query += " AND is_active = ?"
			args = append(args, *filter.IsActive)
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list dq rules: %w", err)
	}
	defer rows.Close()

	var rules []*DQRuleRow
	for rows.Next() {
		var rule DQRuleRow
		if err := rows.Scan(
			&rule.ID, &rule.SourceID, &rule.ObjectID, &rule.ColumnID, &rule.Name,
			&rule.Description, &rule.RuleType, &rule.RuleConfig, &rule.Severity,
			&rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan dq rule: %w", err)
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

// UpdateDQRule 更新数据质量规则
func (s *SQLiteStore) UpdateDQRule(ctx context.Context, id string, updates *DQRuleUpdate) error {
	query := `UPDATE dq_rules SET updated_at = datetime('now')`
	var args []interface{}

	if updates.Name != nil {
		query += ", name = ?"
		args = append(args, *updates.Name)
	}
	if updates.Description != nil {
		query += ", description = ?"
		args = append(args, *updates.Description)
	}
	if updates.RuleConfig != nil {
		query += ", rule_config = ?"
		args = append(args, *updates.RuleConfig)
	}
	if updates.Severity != nil {
		query += ", severity = ?"
		args = append(args, *updates.Severity)
	}
	if updates.IsActive != nil {
		query += ", is_active = ?"
		args = append(args, *updates.IsActive)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update dq rule: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("dq rule not found: %s", id)
	}
	return nil
}

// DeleteDQRule 删除数据质量规则
func (s *SQLiteStore) DeleteDQRule(ctx context.Context, id string) error {
	query := `DELETE FROM dq_rules WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dq rule: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("dq rule not found: %s", id)
	}
	return nil
}

// CreateDQResult 创建数据质量检测结果
func (s *SQLiteStore) CreateDQResult(ctx context.Context, result *DQResultCreate) error {
	query := `
		INSERT INTO dq_results (id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx, query,
		id, result.RuleID, result.CheckBatchID, result.ColumnID, result.Status,
		result.TotalRows, result.FailedRows, result.PassRate, result.SampleErrors, result.ErrorMessage)
	if err != nil {
		return fmt.Errorf("failed to create dq result: %w", err)
	}
	return nil
}

// ListDQResults 列出数据质量检测结果
func (s *SQLiteStore) ListDQResults(ctx context.Context, filter *DQResultFilter) ([]*DQResultRow, error) {
	query := `
		SELECT id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors, error_message, checked_at
		FROM dq_results WHERE 1=1
	`
	var args []interface{}

	if filter != nil {
		if filter.RuleID != nil {
			query += " AND rule_id = ?"
			args = append(args, *filter.RuleID)
		}
		if filter.BatchID != nil {
			query += " AND check_batch_id = ?"
			args = append(args, *filter.BatchID)
		}
		if filter.ColumnID != nil {
			query += " AND column_id = ?"
			args = append(args, *filter.ColumnID)
		}
		if filter.Status != nil {
			query += " AND status = ?"
			args = append(args, *filter.Status)
		}
	}

	query += " ORDER BY checked_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list dq results: %w", err)
	}
	defer rows.Close()

	var results []*DQResultRow
	for rows.Next() {
		var result DQResultRow
		if err := rows.Scan(
			&result.ID, &result.RuleID, &result.CheckBatchID, &result.ColumnID,
			&result.Status, &result.TotalRows, &result.FailedRows, &result.PassRate,
			&result.SampleErrors, &result.ErrorMessage, &result.CheckedAt); err != nil {
			return nil, fmt.Errorf("failed to scan dq result: %w", err)
		}
		results = append(results, &result)
	}
	return results, rows.Err()
}

// GetLatestDQResult 获取最新的数据质量检测结果
func (s *SQLiteStore) GetLatestDQResult(ctx context.Context, ruleID string) (*DQResultRow, error) {
	query := `
		SELECT id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors, error_message, checked_at
		FROM dq_results WHERE rule_id = ? ORDER BY checked_at DESC LIMIT 1
	`
	row := s.db.QueryRowContext(ctx, query, ruleID)
	var result DQResultRow
	err := row.Scan(
		&result.ID, &result.RuleID, &result.CheckBatchID, &result.ColumnID,
		&result.Status, &result.TotalRows, &result.FailedRows, &result.PassRate,
		&result.SampleErrors, &result.ErrorMessage, &result.CheckedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no dq result found for rule: %s", ruleID)
		}
		return nil, fmt.Errorf("failed to get latest dq result: %w", err)
	}
	return &result, nil
}

// GetDQStats 获取数据质量统计
func (s *SQLiteStore) GetDQStats(ctx context.Context) (*DQStatsRow, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM dq_rules) as total_rules,
			(SELECT COUNT(*) FROM dq_rules WHERE is_active = 1) as active_rules,
			(SELECT COUNT(*) FROM dq_results) as total_checks,
			(SELECT COUNT(*) FROM dq_results WHERE status = 'passed') as passed_checks,
			(SELECT COUNT(*) FROM dq_results WHERE status = 'failed') as failed_checks
	`
	row := s.db.QueryRowContext(ctx, query)
	var stats DQStatsRow
	err := row.Scan(
		&stats.TotalRules, &stats.ActiveRules, &stats.TotalChecks, &stats.PassedChecks, &stats.FailedChecks)
	if err != nil {
		return nil, fmt.Errorf("failed to get dq stats: %w", err)
	}

	if stats.TotalChecks > 0 {
		stats.OverallPassRate = float64(stats.PassedChecks) / float64(stats.TotalChecks) * 100
	}

	return &stats, nil
}

// CreateDQRule 创建数据质量规则
func (t *SQLiteTxStore) CreateDQRule(ctx context.Context, rule *DQRuleCreate) (string, error) {
	query := `
		INSERT INTO dq_rules (id, source_id, object_id, column_id, name, description, rule_type, rule_config, severity, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, rule.SourceID, rule.ObjectID, rule.ColumnID, rule.Name,
		rule.Description, rule.RuleType, rule.RuleConfig, rule.Severity, rule.IsActive)
	if err != nil {
		return "", fmt.Errorf("failed to create dq rule: %w", err)
	}
	return id, nil
}

// GetDQRule 获取数据质量规则
func (t *SQLiteTxStore) GetDQRule(ctx context.Context, id string) (*DQRuleRow, error) {
	query := `
		SELECT id, source_id, object_id, column_id, name, description, rule_type, rule_config, severity, is_active, created_at, updated_at
		FROM dq_rules WHERE id = ?
	`
	row := t.tx.QueryRowContext(ctx, query, id)
	var rule DQRuleRow
	err := row.Scan(
		&rule.ID, &rule.SourceID, &rule.ObjectID, &rule.ColumnID, &rule.Name,
		&rule.Description, &rule.RuleType, &rule.RuleConfig, &rule.Severity,
		&rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dq rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get dq rule: %w", err)
	}
	return &rule, nil
}

// ListDQRules 列出数据质量规则
func (t *SQLiteTxStore) ListDQRules(ctx context.Context, filter *DQRuleFilter) ([]*DQRuleRow, error) {
	query := `
		SELECT id, source_id, object_id, column_id, name, description, rule_type, rule_config, severity, is_active, created_at, updated_at
		FROM dq_rules WHERE 1=1
	`
	var args []interface{}

	if filter != nil {
		if filter.SourceID != nil {
			query += " AND source_id = ?"
			args = append(args, *filter.SourceID)
		}
		if filter.ObjectID != nil {
			query += " AND object_id = ?"
			args = append(args, *filter.ObjectID)
		}
		if filter.ColumnID != nil {
			query += " AND column_id = ?"
			args = append(args, *filter.ColumnID)
		}
		if filter.RuleType != nil {
			query += " AND rule_type = ?"
			args = append(args, *filter.RuleType)
		}
		if filter.IsActive != nil {
			query += " AND is_active = ?"
			args = append(args, *filter.IsActive)
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list dq rules: %w", err)
	}
	defer rows.Close()

	var rules []*DQRuleRow
	for rows.Next() {
		var rule DQRuleRow
		if err := rows.Scan(
			&rule.ID, &rule.SourceID, &rule.ObjectID, &rule.ColumnID, &rule.Name,
			&rule.Description, &rule.RuleType, &rule.RuleConfig, &rule.Severity,
			&rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan dq rule: %w", err)
		}
		rules = append(rules, &rule)
	}
	return rules, rows.Err()
}

// UpdateDQRule 更新数据质量规则
func (t *SQLiteTxStore) UpdateDQRule(ctx context.Context, id string, updates *DQRuleUpdate) error {
	query := `UPDATE dq_rules SET updated_at = datetime('now')`
	var args []interface{}

	if updates.Name != nil {
		query += ", name = ?"
		args = append(args, *updates.Name)
	}
	if updates.Description != nil {
		query += ", description = ?"
		args = append(args, *updates.Description)
	}
	if updates.RuleConfig != nil {
		query += ", rule_config = ?"
		args = append(args, *updates.RuleConfig)
	}
	if updates.Severity != nil {
		query += ", severity = ?"
		args = append(args, *updates.Severity)
	}
	if updates.IsActive != nil {
		query += ", is_active = ?"
		args = append(args, *updates.IsActive)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update dq rule: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("dq rule not found: %s", id)
	}
	return nil
}

// DeleteDQRule 删除数据质量规则
func (t *SQLiteTxStore) DeleteDQRule(ctx context.Context, id string) error {
	query := `DELETE FROM dq_rules WHERE id = ?`
	result, err := t.tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dq rule: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("dq rule not found: %s", id)
	}
	return nil
}

// CreateDQResult 创建数据质量检测结果
func (t *SQLiteTxStore) CreateDQResult(ctx context.Context, result *DQResultCreate) error {
	query := `
		INSERT INTO dq_results (id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, result.RuleID, result.CheckBatchID, result.ColumnID, result.Status,
		result.TotalRows, result.FailedRows, result.PassRate, result.SampleErrors, result.ErrorMessage)
	if err != nil {
		return fmt.Errorf("failed to create dq result: %w", err)
	}
	return nil
}

// ListDQResults 列出数据质量检测结果
func (t *SQLiteTxStore) ListDQResults(ctx context.Context, filter *DQResultFilter) ([]*DQResultRow, error) {
	query := `
		SELECT id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors, error_message, checked_at
		FROM dq_results WHERE 1=1
	`
	var args []interface{}

	if filter != nil {
		if filter.RuleID != nil {
			query += " AND rule_id = ?"
			args = append(args, *filter.RuleID)
		}
		if filter.BatchID != nil {
			query += " AND check_batch_id = ?"
			args = append(args, *filter.BatchID)
		}
		if filter.ColumnID != nil {
			query += " AND column_id = ?"
			args = append(args, *filter.ColumnID)
		}
		if filter.Status != nil {
			query += " AND status = ?"
			args = append(args, *filter.Status)
		}
	}

	query += " ORDER BY checked_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list dq results: %w", err)
	}
	defer rows.Close()

	var results []*DQResultRow
	for rows.Next() {
		var result DQResultRow
		if err := rows.Scan(
			&result.ID, &result.RuleID, &result.CheckBatchID, &result.ColumnID,
			&result.Status, &result.TotalRows, &result.FailedRows, &result.PassRate,
			&result.SampleErrors, &result.ErrorMessage, &result.CheckedAt); err != nil {
			return nil, fmt.Errorf("failed to scan dq result: %w", err)
		}
		results = append(results, &result)
	}
	return results, rows.Err()
}

// GetLatestDQResult 获取最新的数据质量检测结果
func (t *SQLiteTxStore) GetLatestDQResult(ctx context.Context, ruleID string) (*DQResultRow, error) {
	query := `
		SELECT id, rule_id, check_batch_id, column_id, status, total_rows, failed_rows, pass_rate, sample_errors, error_message, checked_at
		FROM dq_results WHERE rule_id = ? ORDER BY checked_at DESC LIMIT 1
	`
	row := t.tx.QueryRowContext(ctx, query, ruleID)
	var result DQResultRow
	err := row.Scan(
		&result.ID, &result.RuleID, &result.CheckBatchID, &result.ColumnID,
		&result.Status, &result.TotalRows, &result.FailedRows, &result.PassRate,
		&result.SampleErrors, &result.ErrorMessage, &result.CheckedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no dq result found for rule: %s", ruleID)
		}
		return nil, fmt.Errorf("failed to get latest dq result: %w", err)
	}
	return &result, nil
}

// GetDQStats 获取数据质量统计
func (t *SQLiteTxStore) GetDQStats(ctx context.Context) (*DQStatsRow, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM dq_rules) as total_rules,
			(SELECT COUNT(*) FROM dq_rules WHERE is_active = 1) as active_rules,
			(SELECT COUNT(*) FROM dq_results) as total_checks,
			(SELECT COUNT(*) FROM dq_results WHERE status = 'passed') as passed_checks,
			(SELECT COUNT(*) FROM dq_results WHERE status = 'failed') as failed_checks
	`
	row := t.tx.QueryRowContext(ctx, query)
	var stats DQStatsRow
	err := row.Scan(
		&stats.TotalRules, &stats.ActiveRules, &stats.TotalChecks, &stats.PassedChecks, &stats.FailedChecks)
	if err != nil {
		return nil, fmt.Errorf("failed to get dq stats: %w", err)
	}

	if stats.TotalChecks > 0 {
		stats.OverallPassRate = float64(stats.PassedChecks) / float64(stats.TotalChecks) * 100
	}

	return &stats, nil
}
