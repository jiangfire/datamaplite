package scanner

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresScanner PostgreSQL 扫描器
type PostgresScanner struct{}

// NewPostgresScanner 创建 PostgreSQL 扫描器
func NewPostgresScanner() *PostgresScanner {
	return &PostgresScanner{}
}

// TestConnection 测试 PostgreSQL 连接
func (s *PostgresScanner) TestConnection(ctx context.Context, config ConnectionConfig) error {
	db, err := s.connect(ctx, config)
	if err != nil {
		return err
	}
	defer func() {
		ignoreError(db.Close())
	}()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

// ScanSchema 扫描 PostgreSQL Schema
func (s *PostgresScanner) ScanSchema(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error) {
	db, err := s.connect(ctx, config)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(db.Close())
	}()

	objects, err := s.getTables(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres tables: %w", err)
	}

	for i := range objects {
		schemaName := "public"
		if objects[i].Schema != nil && *objects[i].Schema != "" {
			schemaName = *objects[i].Schema
		}

		columns, err := s.getColumns(ctx, db, schemaName, objects[i].Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for %s.%s: %w", schemaName, objects[i].Name, err)
		}
		objects[i].Columns = columns
	}

	return &SchemaInfo{Objects: objects}, nil
}

func (s *PostgresScanner) connect(ctx context.Context, config ConnectionConfig) (*sql.DB, error) {
	sslMode := config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, sslMode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Second)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to ping postgres: %w (close failed: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}

func (s *PostgresScanner) getTables(ctx context.Context, db *sql.DB) ([]ObjectInfo, error) {
	query := `
		SELECT
			t.table_schema,
			t.table_name,
			t.table_type,
			COALESCE(obj_description(c.oid), '') AS table_comment,
			COALESCE(ps.n_live_tup::bigint, 0) AS row_count,
			COALESCE(pg_total_relation_size(c.oid), 0) AS size_bytes
		FROM information_schema.tables t
		JOIN pg_class c ON c.relname = t.table_name
		JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		LEFT JOIN pg_stat_user_tables ps ON ps.relid = c.oid
		WHERE t.table_schema NOT IN ('pg_catalog', 'information_schema')
		  AND t.table_type IN ('BASE TABLE', 'VIEW')
		ORDER BY t.table_schema, t.table_name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(rows.Close())
	}()

	var objects []ObjectInfo
	for rows.Next() {
		var obj ObjectInfo
		var tableType string
		var schemaName string
		var rowCount, sizeBytes int64
		var comment string

		if err := rows.Scan(&schemaName, &obj.Name, &tableType, &comment, &rowCount, &sizeBytes); err != nil {
			return nil, err
		}

		obj.Schema = &schemaName
		if tableType == "VIEW" {
			obj.Type = "view"
		} else {
			obj.Type = "table"
		}
		if comment != "" {
			obj.Description = &comment
		}
		obj.RowCount = &rowCount
		obj.SizeBytes = &sizeBytes
		objects = append(objects, obj)
	}

	return objects, rows.Err()
}

func (s *PostgresScanner) getColumns(ctx context.Context, db *sql.DB, schemaName, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.udt_name,
			c.is_nullable,
			c.column_default,
			COALESCE(col_description(format('%I.%I', c.table_schema, c.table_name)::regclass::oid, c.ordinal_position), '') AS column_comment,
			c.ordinal_position,
			EXISTS (
				SELECT 1
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu
				  ON tc.constraint_name = kcu.constraint_name
				 AND tc.table_schema = kcu.table_schema
				 AND tc.table_name = kcu.table_name
				WHERE tc.constraint_type = 'PRIMARY KEY'
				  AND tc.table_schema = c.table_schema
				  AND tc.table_name = c.table_name
				  AND kcu.column_name = c.column_name
			) AS is_primary_key,
			EXISTS (
				SELECT 1
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu
				  ON tc.constraint_name = kcu.constraint_name
				 AND tc.table_schema = kcu.table_schema
				 AND tc.table_name = kcu.table_name
				WHERE tc.constraint_type = 'UNIQUE'
				  AND tc.table_schema = c.table_schema
				  AND tc.table_name = c.table_name
				  AND kcu.column_name = c.column_name
			) AS is_unique
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(rows.Close())
	}()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable string
		var fullDataType, defaultValue, comment sql.NullString

		if err := rows.Scan(
			&col.Name,
			&col.DataType,
			&fullDataType,
			&isNullable,
			&defaultValue,
			&comment,
			&col.OrdinalPosition,
			&col.IsPrimaryKey,
			&col.IsUnique,
		); err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"
		if fullDataType.Valid {
			col.FullDataType = fullDataType.String
		}
		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}
		if comment.Valid && comment.String != "" {
			col.Description = &comment.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}
