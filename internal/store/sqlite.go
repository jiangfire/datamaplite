package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"git.neolidy.top/neo/fuckcmdb/internal/config"
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
