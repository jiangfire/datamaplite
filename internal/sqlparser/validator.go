package sqlparser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xwb1989/sqlparser"
)

// Validator SQL验证器
type Validator struct {
	maxQueryLength int
	enforceLimit   bool
	defaultLimit   int
}

// NewValidator 创建新的SQL验证器
func NewValidator() *Validator {
	return &Validator{
		maxQueryLength: 10000,
		enforceLimit:   true,
		defaultLimit:   10000,
	}
}

// ValidationError 验证错误
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ValidateSelectSQL 验证SQL语句是否安全
// 只允许SELECT语句，禁止危险函数和操作
func (v *Validator) ValidateSelectSQL(sql string) error {
	if len(sql) > v.maxQueryLength {
		return &ValidationError{
			Code:    "QUERY_TOO_LONG",
			Message: fmt.Sprintf("SQL query exceeds maximum length of %d characters", v.maxQueryLength),
		}
	}

	// 对多语句和注释类关键字先做文本级拦截，避免解析器直接返回语法错误。
	if err := v.checkDangerousKeywords(sql); err != nil {
		return err
	}

	// 解析SQL
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return &ValidationError{
			Code:    "PARSE_ERROR",
			Message: fmt.Sprintf("failed to parse SQL: %v", err),
		}
	}

	// 只允许SELECT语句，显式拒绝UNION
	switch stmt.(type) {
	case *sqlparser.Union:
		return &ValidationError{
			Code:    "UNION_NOT_ALLOWED",
			Message: "UNION is not allowed in custom SQL rules",
		}
	case *sqlparser.Select:
		// valid
	default:
		return &ValidationError{
			Code:    "NOT_SELECT",
			Message: "only SELECT statements are allowed",
		}
	}

	// 检查危险函数
	if err := v.checkDangerousFunctions(sql); err != nil {
		return err
	}
	return nil
}

// dangerousFunctions 危险函数列表
var dangerousFunctions = []string{
	"xp_cmdshell", "sp_oamethod", "sp_oacreate", "sp_oagetproperty",
	"system", "exec", "eval", "benchmark", "sleep", "pg_sleep",
	"waitfor", "delay", "load_file", "into outfile", "into dumpfile",
	"bcp", "bulk insert", "openrowset", "opendatasource",
}

// checkDangerousFunctions 检查SQL中是否包含危险函数
func (v *Validator) checkDangerousFunctions(sql string) error {
	sqlLower := strings.ToLower(sql)
	for _, fn := range dangerousFunctions {
		// 使用正则表达式匹配函数调用
		pattern := fmt.Sprintf(`\b%s\s*\(`, regexp.QuoteMeta(fn))
		matched, _ := regexp.MatchString(pattern, sqlLower)
		if matched {
			return &ValidationError{
				Code:    "DANGEROUS_FUNCTION",
				Message: fmt.Sprintf("dangerous function detected: %s", fn),
			}
		}
	}
	return nil
}

// dangerousKeywords 危险关键字列表
var dangerousKeywords = []string{
	";", "--", "/*", "*/", "@@", "exec(", "execute(",
}

// checkDangerousKeywords 检查SQL中是否包含危险关键字
func (v *Validator) checkDangerousKeywords(sql string) error {
	sqlLower := strings.ToLower(sql)
	for _, kw := range dangerousKeywords {
		if strings.Contains(sqlLower, kw) {
			return &ValidationError{
				Code:    "DANGEROUS_KEYWORD",
				Message: fmt.Sprintf("dangerous keyword detected: %s", kw),
			}
		}
	}
	return nil
}

// SanitizeAndAddLimit 清理SQL并添加LIMIT限制
func (v *Validator) SanitizeAndAddLimit(sql string, limit int) (string, error) {
	if limit <= 0 {
		limit = v.defaultLimit
	}

	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return "", fmt.Errorf("failed to parse SQL: %w", err)
	}

	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		return "", fmt.Errorf("only SELECT statements are supported")
	}

	// 如果已经有LIMIT，检查是否超过限制
	if selectStmt.Limit != nil {
		// 保留现有的LIMIT，但在执行时会检查
		return sqlparser.String(selectStmt), nil
	}

	// 添加LIMIT
	selectStmt.Limit = &sqlparser.Limit{
		Rowcount: sqlparser.NewIntVal([]byte(fmt.Sprintf("%d", limit))),
	}

	return sqlparser.String(selectStmt), nil
}

// ExtractTableNames 从SQL中提取表名
func (v *Validator) ExtractTableNames(sql string) ([]string, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, err
	}

	tables := make([]string, 0)
	seen := make(map[string]struct{})
	sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		if t, ok := node.(sqlparser.TableName); ok {
			tableName := t.Name.String()
			if t.Qualifier.String() != "" {
				tableName = t.Qualifier.String() + "." + tableName
			}
			tableName = strings.TrimSpace(tableName)
			if tableName == "" {
				return true, nil
			}
			if _, exists := seen[tableName]; !exists {
				seen[tableName] = struct{}{}
				tables = append(tables, tableName)
			}
		}
		return true, nil
	}, stmt)

	return tables, nil
}

// IsSimpleSelect 检查是否为简单SELECT（无子查询、无JOIN等）
func (v *Validator) IsSimpleSelect(sql string) (bool, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return false, err
	}

	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		return false, fmt.Errorf("not a SELECT statement")
	}

	// 检查是否有JOIN
	if len(selectStmt.From) > 1 {
		return false, nil
	}

	// 检查是否有子查询
	hasSubquery := false
	sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		if _, ok := node.(*sqlparser.Subquery); ok {
			hasSubquery = true
			return false, nil
		}
		return true, nil
	}, selectStmt)

	return !hasSubquery, nil
}
