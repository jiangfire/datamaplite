package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/scanner"
	"github.com/jiangfire/datamaplite/internal/store"
)

type dqDialect string

const (
	dqDialectMySQL    dqDialect = "mysql"
	dqDialectPostgres dqDialect = "postgres"
)

type dqExecutionContext struct {
	source    *store.DataSourceRow
	object    *store.SchemaObjectRow
	column    *store.ColumnRow
	refObject *store.SchemaObjectRow
	refColumn *store.ColumnRow
	config    *scanner.ConnectionConfig
	dialect   dqDialect
}

// executeRule 执行数据质量规则检查
func (s *DQService) executeRule(ctx context.Context, rule *store.DQRuleRow, batchID string, sampleLimit int) (*store.DQResultCreate, error) {
	var ruleConfig map[string]interface{}
	if err := json.Unmarshal([]byte(rule.RuleConfig), &ruleConfig); err != nil {
		return nil, fmt.Errorf("invalid rule config: %w", err)
	}
	if model.DQRuleType(rule.RuleType) == model.DQRuleTypeCustomSQL {
		sqlText, _ := getStringValue(ruleConfig["sql"])
		if err := s.validateCustomSQL(sqlText); err != nil {
			return nil, err
		}
	}

	execCtx, err := s.prepareExecutionContext(ctx, rule, ruleConfig)
	if err != nil {
		return nil, err
	}

	db, err := s.openSourceDB(execCtx)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(db.Close())
	}()

	totalQuery, totalArgs, failedQuery, failedArgs, sampleQuery, sampleArgs, err := s.buildRuleQueries(execCtx, rule, ruleConfig)
	if err != nil {
		return nil, err
	}

	totalRows, err := s.queryCount(ctx, db, totalQuery, totalArgs)
	if err != nil {
		return nil, err
	}

	failedRows, err := s.queryCount(ctx, db, failedQuery, failedArgs)
	if err != nil {
		return nil, err
	}

	sampleQuery, sampleArgs = s.appendLimit(sampleQuery, sampleArgs, sampleLimit, execCtx.dialect)
	sampleErrors, err := s.querySampleRows(ctx, db, sampleQuery, sampleArgs)
	if err != nil {
		return nil, err
	}

	passRate := 100.0
	status := string(model.DQResultStatusPassed)
	if totalRows > 0 {
		passRate = float64(totalRows-failedRows) / float64(totalRows) * 100
		passRate = math.Round(passRate*100) / 100
	}
	if failedRows > 0 {
		status = string(model.DQResultStatusFailed)
	}

	sampleErrorsJSON, err := json.Marshal(sampleErrors)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sample errors: %w", err)
	}

	return &store.DQResultCreate{
		RuleID:       rule.ID,
		CheckBatchID: batchID,
		ColumnID:     rule.ColumnID,
		Status:       status,
		TotalRows:    totalRows,
		FailedRows:   failedRows,
		PassRate:     passRate,
		SampleErrors: string(sampleErrorsJSON),
	}, nil
}

func (s *DQService) buildErrorResult(rule *store.DQRuleRow, batchID string, err error) *store.DQResultCreate {
	sampleErrors, _ := json.Marshal([]map[string]interface{}{})
	errMsg := err.Error()
	return &store.DQResultCreate{
		RuleID:       rule.ID,
		CheckBatchID: batchID,
		ColumnID:     rule.ColumnID,
		Status:       string(model.DQResultStatusError),
		TotalRows:    0,
		FailedRows:   0,
		PassRate:     0,
		SampleErrors: string(sampleErrors),
		ErrorMessage: &errMsg,
	}
}

func (s *DQService) prepareExecutionContext(ctx context.Context, rule *store.DQRuleRow, ruleConfig map[string]interface{}) (*dqExecutionContext, error) {
	execCtx := &dqExecutionContext{}

	if rule.ColumnID != nil {
		column, err := s.store.GetColumn(ctx, *rule.ColumnID)
		if err != nil {
			return nil, fmt.Errorf("failed to load rule column: %w", err)
		}
		execCtx.column = column

		object, err := s.store.GetSchemaObject(ctx, column.ObjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to load rule object: %w", err)
		}
		execCtx.object = object
	}

	if execCtx.object == nil && rule.ObjectID != nil {
		object, err := s.store.GetSchemaObject(ctx, *rule.ObjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to load rule object: %w", err)
		}
		execCtx.object = object
	}

	if execCtx.object == nil && model.DQRuleType(rule.RuleType) != model.DQRuleTypeCustomSQL {
		return nil, fmt.Errorf("rule %s requires object context", rule.ID)
	}

	var sourceID *string
	if execCtx.object != nil {
		sourceID = &execCtx.object.SourceID
	} else {
		sourceID = rule.SourceID
	}
	if sourceID == nil {
		return nil, fmt.Errorf("rule %s requires source_id, object_id or column_id", rule.ID)
	}

	source, err := s.store.GetDataSource(ctx, *sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load rule data source: %w", err)
	}
	execCtx.source = source

	config, err := s.decryptConnectionConfig(source.ConnectionConfig)
	if err != nil {
		return nil, err
	}
	execCtx.config = config

	switch source.Type {
	case string(model.DataSourceMySQL):
		execCtx.dialect = dqDialectMySQL
	case string(model.DataSourcePostgreSQL):
		execCtx.dialect = dqDialectPostgres
	default:
		return nil, fmt.Errorf("dq execution is only supported for mysql and postgres sources")
	}

	if model.DQRuleType(rule.RuleType) == model.DQRuleTypeReferential {
		refObjectID, _ := getStringValue(ruleConfig["ref_object_id"])
		refColumnID, _ := getStringValue(ruleConfig["ref_column_id"])
		if refObjectID == "" || refColumnID == "" {
			return nil, fmt.Errorf("referential rule requires ref_object_id and ref_column_id")
		}

		refObject, err := s.store.GetSchemaObject(ctx, refObjectID)
		if err != nil {
			return nil, fmt.Errorf("failed to load referenced object: %w", err)
		}
		refColumn, err := s.store.GetColumn(ctx, refColumnID)
		if err != nil {
			return nil, fmt.Errorf("failed to load referenced column: %w", err)
		}
		if refObject.SourceID != source.ID {
			return nil, fmt.Errorf("referential rule must reference a column in the same data source")
		}
		execCtx.refObject = refObject
		execCtx.refColumn = refColumn
	}

	return execCtx, nil
}

func (s *DQService) decryptConnectionConfig(ciphertext string) (*scanner.ConnectionConfig, error) {
	if s.cipher != nil {
		if decrypted, err := s.cipher.Decrypt(ciphertext); err == nil && decrypted != "" {
			return scanner.ConnectionConfigFromJSON(decrypted)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(ciphertext), "{") {
		return scanner.ConnectionConfigFromJSON(ciphertext)
	}
	return nil, fmt.Errorf("failed to decrypt data source connection config")
}

func (s *DQService) openSourceDB(execCtx *dqExecutionContext) (*sql.DB, error) {
	switch execCtx.dialect {
	case dqDialectMySQL:
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true",
			execCtx.config.Username, execCtx.config.Password, execCtx.config.Host, execCtx.config.Port, execCtx.config.Database)
		if execCtx.config.SSLMode == "require" {
			dsn += "&tls=true"
		}
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(2)
		db.SetMaxIdleConns(1)
		return db, nil
	case dqDialectPostgres:
		sslMode := execCtx.config.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			execCtx.config.Host, execCtx.config.Port, execCtx.config.Username, execCtx.config.Password, execCtx.config.Database, sslMode)
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(2)
		db.SetMaxIdleConns(1)
		return db, nil
	default:
		return nil, fmt.Errorf("unsupported dq dialect: %s", execCtx.dialect)
	}
}

func (s *DQService) buildRuleQueries(execCtx *dqExecutionContext, rule *store.DQRuleRow, ruleConfig map[string]interface{}) (string, []interface{}, string, []interface{}, string, []interface{}, error) {
	tableName := ""
	if execCtx.object != nil {
		tableName = s.qualifyObject(execCtx.object, execCtx.dialect)
	}

	totalQuery := failedQueryBase(tableName)
	var totalArgs []interface{}
	var failedQuery string
	var failedArgs []interface{}
	var sampleQuery string
	var sampleArgs []interface{}

	switch model.DQRuleType(rule.RuleType) {
	case model.DQRuleTypeNotNull:
		if execCtx.column == nil {
			return "", nil, "", nil, "", nil, fmt.Errorf("not_null rule requires column context")
		}
		col := s.quoteIdent(execCtx.column.Name, execCtx.dialect)
		failedQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s IS NULL", tableName, col)
		sampleQuery = fmt.Sprintf("SELECT %s AS value FROM %s WHERE %s IS NULL", col, tableName, col)
	case model.DQRuleTypeUnique:
		if execCtx.column == nil {
			return "", nil, "", nil, "", nil, fmt.Errorf("unique rule requires column context")
		}
		col := s.quoteIdent(execCtx.column.Name, execCtx.dialect)
		failedQuery = fmt.Sprintf(
			"SELECT COALESCE(SUM(duplicate_count - 1), 0) FROM (SELECT %s, COUNT(*) AS duplicate_count FROM %s WHERE %s IS NOT NULL GROUP BY %s HAVING COUNT(*) > 1) dq_duplicates",
			col, tableName, col, col,
		)
		sampleQuery = fmt.Sprintf(
			"SELECT %s AS value, COUNT(*) AS duplicate_count FROM %s WHERE %s IS NOT NULL GROUP BY %s HAVING COUNT(*) > 1 ORDER BY duplicate_count DESC",
			col, tableName, col, col,
		)
	case model.DQRuleTypeRegex:
		if execCtx.column == nil {
			return "", nil, "", nil, "", nil, fmt.Errorf("regex rule requires column context")
		}
		col := s.quoteIdent(execCtx.column.Name, execCtx.dialect)
		pattern, _ := getStringValue(ruleConfig["pattern"])
		flags, _ := getStringValue(ruleConfig["flags"])
		condition, args := s.buildRegexCondition(col, pattern, flags, execCtx.dialect)
		failedArgs = append(failedArgs, args...)
		sampleArgs = append(sampleArgs, args...)
		failedQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s IS NOT NULL AND %s", tableName, col, condition)
		sampleQuery = fmt.Sprintf("SELECT %s AS value FROM %s WHERE %s IS NOT NULL AND %s", col, tableName, col, condition)
	case model.DQRuleTypeRange:
		if execCtx.column == nil {
			return "", nil, "", nil, "", nil, fmt.Errorf("range rule requires column context")
		}
		col := s.quoteIdent(execCtx.column.Name, execCtx.dialect)
		conditions := make([]string, 0, 2)
		if min, ok := getFloatValue(ruleConfig["min"]); ok {
			conditions = append(conditions, fmt.Sprintf("%s < %s", col, s.placeholder(len(failedArgs)+1, execCtx.dialect)))
			failedArgs = append(failedArgs, min)
			sampleArgs = append(sampleArgs, min)
		}
		if max, ok := getFloatValue(ruleConfig["max"]); ok {
			conditions = append(conditions, fmt.Sprintf("%s > %s", col, s.placeholder(len(failedArgs)+1, execCtx.dialect)))
			failedArgs = append(failedArgs, max)
			sampleArgs = append(sampleArgs, max)
		}
		if len(conditions) == 0 {
			return "", nil, "", nil, "", nil, fmt.Errorf("range rule requires min or max")
		}
		whereClause := strings.Join(conditions, " OR ")
		failedQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", tableName, whereClause)
		sampleQuery = fmt.Sprintf("SELECT %s AS value FROM %s WHERE %s", col, tableName, whereClause)
	case model.DQRuleTypeEnum:
		if execCtx.column == nil {
			return "", nil, "", nil, "", nil, fmt.Errorf("enum rule requires column context")
		}
		values, err := getStringSliceValue(ruleConfig["values"])
		if err != nil || len(values) == 0 {
			return "", nil, "", nil, "", nil, fmt.Errorf("enum rule requires non-empty values")
		}
		col := s.quoteIdent(execCtx.column.Name, execCtx.dialect)
		placeholders := make([]string, 0, len(values))
		for _, value := range values {
			placeholders = append(placeholders, s.placeholder(len(failedArgs)+1, execCtx.dialect))
			failedArgs = append(failedArgs, value)
			sampleArgs = append(sampleArgs, value)
		}
		notInClause := fmt.Sprintf("%s IS NOT NULL AND %s NOT IN (%s)", col, col, strings.Join(placeholders, ", "))
		failedQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", tableName, notInClause)
		sampleQuery = fmt.Sprintf("SELECT %s AS value FROM %s WHERE %s", col, tableName, notInClause)
	case model.DQRuleTypeCustomSQL:
		sqlText, _ := getStringValue(ruleConfig["sql"])
		if strings.TrimSpace(sqlText) == "" {
			return "", nil, "", nil, "", nil, fmt.Errorf("custom_sql rule requires sql")
		}
		failedQuery = fmt.Sprintf("SELECT COUNT(*) FROM (%s) dq_custom_failures", sqlText)
		sampleQuery = fmt.Sprintf("SELECT * FROM (%s) dq_custom_failures", sqlText)
		if tableName == "" {
			totalQuery = failedQuery
		}
	case model.DQRuleTypeReferential:
		if execCtx.column == nil || execCtx.refObject == nil || execCtx.refColumn == nil {
			return "", nil, "", nil, "", nil, fmt.Errorf("referential rule requires source and reference column context")
		}
		col := s.quoteIdent(execCtx.column.Name, execCtx.dialect)
		refTable := s.qualifyObject(execCtx.refObject, execCtx.dialect)
		refCol := s.quoteIdent(execCtx.refColumn.Name, execCtx.dialect)
		condition := fmt.Sprintf("%s IS NOT NULL AND NOT EXISTS (SELECT 1 FROM %s ref WHERE ref.%s = src.%s)", col, refTable, refCol, col)
		failedQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s src WHERE %s", tableName, condition)
		sampleQuery = fmt.Sprintf("SELECT src.%s AS value FROM %s src WHERE %s", col, tableName, condition)
	default:
		return "", nil, "", nil, "", nil, fmt.Errorf("unsupported dq rule type: %s", rule.RuleType)
	}

	return totalQuery, totalArgs, failedQuery, failedArgs, sampleQuery, sampleArgs, nil
}

func failedQueryBase(tableName string) string {
	if tableName == "" {
		return "SELECT 0"
	}
	return fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
}

func (s *DQService) queryCount(ctx context.Context, db *sql.DB, query string, args []interface{}) (int64, error) {
	var count int64
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("dq count query failed: %w", err)
	}
	return count, nil
}

func (s *DQService) querySampleRows(ctx context.Context, db *sql.DB, query string, args []interface{}) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("dq sample query failed: %w", err)
	}
	defer closeSQLRows(rows)

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		item := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			item[col] = normalizeSQLValue(values[i])
		}
		results = append(results, item)
	}

	return results, rows.Err()
}

func normalizeSQLValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func (s *DQService) appendLimit(query string, args []interface{}, limit int, dialect dqDialect) (string, []interface{}) {
	if limit <= 0 {
		limit = 5
	}
	query += " LIMIT " + s.placeholder(len(args)+1, dialect)
	args = append(args, limit)
	return query, args
}

func (s *DQService) placeholder(pos int, dialect dqDialect) string {
	if dialect == dqDialectPostgres {
		return fmt.Sprintf("$%d", pos)
	}
	return "?"
}

func (s *DQService) quoteIdent(ident string, dialect dqDialect) string {
	if dialect == dqDialectPostgres {
		return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
	}
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

func (s *DQService) qualifyObject(obj *store.SchemaObjectRow, dialect dqDialect) string {
	name := s.quoteIdent(obj.Name, dialect)
	if dialect == dqDialectPostgres {
		schemaName := "public"
		if obj.Schema != nil && *obj.Schema != "" {
			schemaName = *obj.Schema
		}
		return s.quoteIdent(schemaName, dialect) + "." + name
	}
	if obj.Schema != nil && *obj.Schema != "" {
		return s.quoteIdent(*obj.Schema, dialect) + "." + name
	}
	return name
}

func (s *DQService) buildRegexCondition(columnExpr, pattern, flags string, dialect dqDialect) (string, []interface{}) {
	if dialect == dqDialectPostgres {
		operator := "!~"
		if strings.Contains(strings.ToLower(flags), "i") {
			operator = "!~*"
		}
		return fmt.Sprintf("%s %s %s", columnExpr, operator, s.placeholder(1, dialect)), []interface{}{pattern}
	}

	if flags != "" {
		return fmt.Sprintf("NOT REGEXP_LIKE(%s, %s, %s)", columnExpr, s.placeholder(1, dialect), s.placeholder(2, dialect)), []interface{}{pattern, flags}
	}
	return fmt.Sprintf("NOT REGEXP_LIKE(%s, %s)", columnExpr, s.placeholder(1, dialect)), []interface{}{pattern}
}

func getStringValue(value interface{}) (string, bool) {
	if value == nil {
		return "", false
	}
	switch v := value.(type) {
	case string:
		return v, true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func getFloatValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func getStringSliceValue(value interface{}) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case []interface{}:
		values := make([]string, 0, len(v))
		for _, item := range v {
			str, ok := getStringValue(item)
			if !ok {
				return nil, fmt.Errorf("enum values must be strings")
			}
			values = append(values, str)
		}
		return values, nil
	default:
		return nil, fmt.Errorf("enum values must be an array")
	}
}
