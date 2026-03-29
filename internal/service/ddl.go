package service

import (
	"fmt"
	"regexp"
	"strings"

	"git.neolidy.top/neo/fuckcmdb/internal/store"
)

// DDLGenerator DDL生成器
type DDLGenerator struct{}

var (
	numericDefaultPattern = regexp.MustCompile(`^[+-]?\d+(\.\d+)?$`)
	varcharTypePattern    = regexp.MustCompile(`^(?:character varying|varchar)\s*\((\d+)\)$`)
	charTypePattern       = regexp.MustCompile(`^(?:character|char)\s*\((\d+)\)$`)
	decimalTypePattern    = regexp.MustCompile(`^(?:decimal|numeric)\s*\(([^)]+)\)$`)
)

// NewDDLGenerator 创建DDL生成器
func NewDDLGenerator() *DDLGenerator {
	return &DDLGenerator{}
}

// GenerateMySQLDDL 生成MySQL DDL
func (g *DDLGenerator) GenerateMySQLDDL(obj *store.SchemaObjectRow, cols []*store.ColumnRow) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", g.quoteIdentifier(obj.Name, "mysql")))

	var colDefs []string
	var pkCols []string

	for _, col := range cols {
		colDef := g.generateMySQLColumnDef(col)
		colDefs = append(colDefs, "  "+colDef)

		if col.IsPrimaryKey {
			pkCols = append(pkCols, g.quoteIdentifier(col.Name, "mysql"))
		}
	}

	sb.WriteString(strings.Join(colDefs, ",\n"))

	if len(pkCols) > 0 {
		sb.WriteString(fmt.Sprintf(",\n  PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}

	sb.WriteString("\n);")

	return sb.String()
}

// GeneratePostgresDDL 生成PostgreSQL DDL
func (g *DDLGenerator) GeneratePostgresDDL(obj *store.SchemaObjectRow, cols []*store.ColumnRow) string {
	var sb strings.Builder

	schema := "public"
	if obj.Schema != nil && *obj.Schema != "" {
		schema = *obj.Schema
	}

	sb.WriteString(fmt.Sprintf(
		"CREATE TABLE %s.%s (\n",
		g.quoteIdentifier(schema, "postgres"),
		g.quoteIdentifier(obj.Name, "postgres"),
	))

	var colDefs []string
	var pkCols []string

	for _, col := range cols {
		colDef := g.generatePostgresColumnDef(col)
		colDefs = append(colDefs, "  "+colDef)

		if col.IsPrimaryKey {
			pkCols = append(pkCols, g.quoteIdentifier(col.Name, "postgres"))
		}
	}

	sb.WriteString(strings.Join(colDefs, ",\n"))

	if len(pkCols) > 0 {
		sb.WriteString(fmt.Sprintf(",\n  PRIMARY KEY (%s)", strings.Join(pkCols, ", ")))
	}

	sb.WriteString("\n);")

	return sb.String()
}

func (g *DDLGenerator) generateMySQLColumnDef(col *store.ColumnRow) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s ", g.quoteIdentifier(col.Name, "mysql")))

	// 数据类型映射
	dataType := g.mapToMySQLType(col.DataType, col.FullDataType)
	sb.WriteString(dataType)

	if !col.IsNullable {
		sb.WriteString(" NOT NULL")
	}

	if col.DefaultValue != nil && *col.DefaultValue != "" {
		sb.WriteString(fmt.Sprintf(" DEFAULT %s", g.formatDefaultValue(col, "mysql")))
	}

	if col.IsUnique && !col.IsPrimaryKey {
		sb.WriteString(" UNIQUE")
	}

	return sb.String()
}

func (g *DDLGenerator) generatePostgresColumnDef(col *store.ColumnRow) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s ", g.quoteIdentifier(col.Name, "postgres")))

	// 数据类型映射
	dataType := g.mapToPostgresType(col.DataType, col.FullDataType)
	sb.WriteString(dataType)

	if !col.IsNullable {
		sb.WriteString(" NOT NULL")
	}

	if col.DefaultValue != nil && *col.DefaultValue != "" {
		sb.WriteString(fmt.Sprintf(" DEFAULT %s", g.formatDefaultValue(col, "postgres")))
	}

	return sb.String()
}

func (g *DDLGenerator) mapToMySQLType(dataType, fullDataType string) string {
	lowerFull := strings.ToLower(strings.TrimSpace(fullDataType))
	if lowerFull != "" {
		switch {
		case varcharTypePattern.MatchString(lowerFull):
			matches := varcharTypePattern.FindStringSubmatch(lowerFull)
			return fmt.Sprintf("VARCHAR(%s)", matches[1])
		case charTypePattern.MatchString(lowerFull):
			matches := charTypePattern.FindStringSubmatch(lowerFull)
			return fmt.Sprintf("CHAR(%s)", matches[1])
		case decimalTypePattern.MatchString(lowerFull):
			matches := decimalTypePattern.FindStringSubmatch(lowerFull)
			return fmt.Sprintf("DECIMAL(%s)", matches[1])
		case lowerFull == "timestamp without time zone", lowerFull == "timestamp with time zone":
			return "TIMESTAMP"
		case lowerFull == "double precision":
			return "DOUBLE"
		case lowerFull == "real":
			return "FLOAT"
		case lowerFull == "jsonb":
			return "JSON"
		case strings.HasPrefix(lowerFull, "varchar"):
			return strings.ToUpper(fullDataType)
		case strings.HasPrefix(lowerFull, "bool"), strings.HasPrefix(lowerFull, "boolean"):
			return "TINYINT(1)"
		case strings.HasPrefix(lowerFull, "bigint"):
			return "BIGINT"
		case strings.HasPrefix(lowerFull, "smallint"):
			return "SMALLINT"
		case strings.HasPrefix(lowerFull, "int"), strings.HasPrefix(lowerFull, "integer"):
			return "INT"
		case strings.HasPrefix(lowerFull, "decimal"), strings.HasPrefix(lowerFull, "numeric"):
			return strings.ToUpper(strings.Replace(lowerFull, "numeric", "decimal", 1))
		case strings.HasPrefix(lowerFull, "text"):
			return "TEXT"
		case strings.HasPrefix(lowerFull, "date"):
			return "DATE"
		case strings.HasPrefix(lowerFull, "time"):
			return "TIME"
		case strings.HasPrefix(lowerFull, "json"):
			return "JSON"
		}
	}

	switch strings.ToLower(dataType) {
	case "int", "integer":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "varchar", "string", "character varying":
		return "VARCHAR(255)"
	case "character", "char":
		return "CHAR(1)"
	case "text":
		return "TEXT"
	case "bool", "boolean":
		return "TINYINT(1)"
	case "float":
		return "FLOAT"
	case "double", "double precision":
		return "DOUBLE"
	case "decimal", "numeric":
		return "DECIMAL(18,2)"
	case "date":
		return "DATE"
	case "datetime":
		return "DATETIME"
	case "timestamp", "timestamp without time zone", "timestamp with time zone":
		return "TIMESTAMP"
	case "json", "jsonb":
		return "JSON"
	default:
		return "VARCHAR(255)"
	}
}

func (g *DDLGenerator) mapToPostgresType(dataType, fullDataType string) string {
	if fullDataType != "" {
		// 转换MySQL类型到PostgreSQL
		return g.convertMySQLToPostgresType(fullDataType)
	}

	switch strings.ToLower(dataType) {
	case "int", "integer":
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "varchar", "string":
		return "VARCHAR(255)"
	case "text":
		return "TEXT"
	case "bool", "boolean":
		return "BOOLEAN"
	case "float":
		return "REAL"
	case "double":
		return "DOUBLE PRECISION"
	case "decimal", "numeric":
		return "NUMERIC(18,2)"
	case "date":
		return "DATE"
	case "datetime", "timestamp":
		return "TIMESTAMP"
	case "json":
		return "JSONB"
	default:
		return "VARCHAR(255)"
	}
}

func (g *DDLGenerator) convertMySQLToPostgresType(mysqlType string) string {
	// 简单转换规则
	lower := strings.ToLower(mysqlType)

	switch {
	case strings.Contains(lower, "tinyint(1)"):
		return "BOOLEAN"
	case strings.HasPrefix(lower, "bigint"):
		return "BIGINT"
	case strings.HasPrefix(lower, "smallint"):
		return "SMALLINT"
	case strings.HasPrefix(lower, "mediumint"):
		return "INTEGER"
	case strings.HasPrefix(lower, "integer"), strings.HasPrefix(lower, "int"):
		return "INTEGER"
	case strings.Contains(lower, "varchar"):
		return strings.ToUpper(mysqlType)
	case strings.Contains(lower, "char"):
		return strings.ToUpper(mysqlType)
	case strings.Contains(lower, "decimal"), strings.Contains(lower, "numeric"):
		return strings.ToUpper(mysqlType)
	case strings.Contains(lower, "double"):
		return "DOUBLE PRECISION"
	case strings.Contains(lower, "float"):
		return "REAL"
	case strings.Contains(lower, "datetime"):
		return "TIMESTAMP"
	case strings.Contains(lower, "timestamp"):
		return "TIMESTAMP"
	case strings.Contains(lower, "date"):
		return "DATE"
	case strings.Contains(lower, "time"):
		return "TIME"
	case strings.Contains(lower, "json"):
		return "JSONB"
	case strings.Contains(lower, "text"):
		return "TEXT"
	case strings.Contains(lower, "bool"), strings.Contains(lower, "boolean"):
		return "BOOLEAN"
	default:
		return mysqlType
	}
}

func (g *DDLGenerator) quoteIdentifier(name string, dialect string) string {
	switch dialect {
	case "postgres":
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	default:
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
}

func (g *DDLGenerator) formatDefaultValue(col *store.ColumnRow, dialect string) string {
	if col.DefaultValue == nil {
		return ""
	}

	value := strings.TrimSpace(*col.DefaultValue)
	if value == "" {
		return ""
	}

	if g.isRawDefaultValue(value) {
		return value
	}

	if g.isNumericLikeType(col) || g.isBooleanLikeType(col) {
		return value
	}

	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func (g *DDLGenerator) isNumericLikeType(col *store.ColumnRow) bool {
	typeValue := strings.ToLower(strings.TrimSpace(col.FullDataType))
	if typeValue == "" {
		typeValue = strings.ToLower(strings.TrimSpace(col.DataType))
	}

	numericPrefixes := []string{
		"int",
		"integer",
		"bigint",
		"smallint",
		"mediumint",
		"decimal",
		"numeric",
		"float",
		"double",
		"real",
		"serial",
		"bigserial",
	}

	for _, prefix := range numericPrefixes {
		if strings.HasPrefix(typeValue, prefix) {
			return true
		}
	}

	return false
}

func (g *DDLGenerator) isBooleanLikeType(col *store.ColumnRow) bool {
	typeValue := strings.ToLower(strings.TrimSpace(col.FullDataType))
	if typeValue == "" {
		typeValue = strings.ToLower(strings.TrimSpace(col.DataType))
	}

	return strings.HasPrefix(typeValue, "bool") ||
		strings.HasPrefix(typeValue, "boolean") ||
		strings.HasPrefix(typeValue, "tinyint(1)")
}

func (g *DDLGenerator) isRawDefaultValue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))

	rawKeywords := map[string]struct{}{
		"null":              {},
		"true":              {},
		"false":             {},
		"current_timestamp": {},
		"current_date":      {},
		"current_time":      {},
		"localtimestamp":    {},
		"localtime":         {},
	}
	if _, ok := rawKeywords[lower]; ok {
		return true
	}

	rawPrefixes := []string{
		"current_timestamp(",
		"now(",
		"timezone(",
		"nextval(",
		"uuid_generate_v4(",
		"gen_random_uuid(",
	}
	for _, prefix := range rawPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	if strings.Contains(lower, " on update current_timestamp") {
		return true
	}

	if numericDefaultPattern.MatchString(lower) {
		return true
	}

	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return true
	}

	return false
}
