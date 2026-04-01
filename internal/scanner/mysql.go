package scanner

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLScanner MySQL扫描器
type MySQLScanner struct{}

// NewMySQLScanner 创建MySQL扫描器
func NewMySQLScanner() *MySQLScanner {
	return &MySQLScanner{}
}

// TestConnection 测试MySQL连接
func (s *MySQLScanner) TestConnection(ctx context.Context, config ConnectionConfig) error {
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

// ScanSchema 扫描MySQL Schema
func (s *MySQLScanner) ScanSchema(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error) {
	db, err := s.connect(ctx, config)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(db.Close())
	}()

	objects, err := s.getTables(ctx, db, config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	for i := range objects {
		columns, err := s.getColumns(ctx, db, config.Database, objects[i].Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", objects[i].Name, err)
		}
		objects[i].Columns = columns
	}

	return &SchemaInfo{Objects: objects}, nil
}

// connect 建立连接
func (s *MySQLScanner) connect(ctx context.Context, config ConnectionConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	if config.SSLMode == "require" {
		dsn += "&tls=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Second)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to ping mysql: %w (close failed: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	return db, nil
}

// getTables 获取所有表
func (s *MySQLScanner) getTables(ctx context.Context, db *sql.DB, database string) ([]ObjectInfo, error) {
	query := `
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			TABLE_COMMENT,
			TABLE_ROWS,
			(DATA_LENGTH + INDEX_LENGTH) as size_bytes
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME
	`

	rows, err := db.QueryContext(ctx, query, database)
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
		var rowCount, sizeBytes sql.NullInt64
		var comment sql.NullString

		err := rows.Scan(&obj.Name, &tableType, &comment, &rowCount, &sizeBytes)
		if err != nil {
			return nil, err
		}

		// Convert table type
		switch tableType {
		case "BASE TABLE":
			obj.Type = "table"
		case "VIEW":
			obj.Type = "view"
		default:
			obj.Type = "table"
		}

		if comment.Valid && comment.String != "" {
			obj.Description = &comment.String
		}
		if rowCount.Valid {
			obj.RowCount = &rowCount.Int64
		}
		if sizeBytes.Valid {
			obj.SizeBytes = &sizeBytes.Int64
		}

		objects = append(objects, obj)
	}

	return objects, rows.Err()
}

// getColumns 获取表的字段
func (s *MySQLScanner) getColumns(ctx context.Context, db *sql.DB, database, table string) ([]ColumnInfo, error) {
	query := `
		SELECT
			COLUMN_NAME,
			DATA_TYPE,
			COLUMN_TYPE,
			IS_NULLABLE,
			COLUMN_DEFAULT,
			COLUMN_COMMENT,
			ORDINAL_POSITION,
			COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(rows.Close())
	}()

	// 获取主键列
	pkColumns, err := s.getPrimaryKeyColumns(ctx, db, database, table)
	if err != nil {
		return nil, err
	}

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable, columnKey string
		var defaultValue, comment sql.NullString

		err := rows.Scan(
			&col.Name, &col.DataType, &col.FullDataType, &isNullable,
			&defaultValue, &comment, &col.OrdinalPosition, &columnKey,
		)
		if err != nil {
			return nil, err
		}

		col.IsNullable = (isNullable == "YES")
		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}
		if comment.Valid && comment.String != "" {
			col.Description = &comment.String
		}

		// 检查是否主键
		if _, ok := pkColumns[col.Name]; ok {
			col.IsPrimaryKey = true
		}

		// 检查是否唯一
		if columnKey == "UNI" {
			col.IsUnique = true
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// getPrimaryKeyColumns 获取主键列
func (s *MySQLScanner) getPrimaryKeyColumns(ctx context.Context, db *sql.DB, database, table string) (map[string]bool, error) {
	query := `
		SELECT COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ?
		  AND TABLE_NAME = ?
		  AND CONSTRAINT_NAME = 'PRIMARY'
	`

	rows, err := db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(rows.Close())
	}()

	pkColumns := make(map[string]bool)
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			return nil, err
		}
		pkColumns[colName] = true
	}

	return pkColumns, rows.Err()
}
