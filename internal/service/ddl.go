package service

import (
	"fmt"
	"strings"

	"git.neolidy.top/neo/fuckcmdb/internal/store"
)

// DDLGenerator DDL生成器
type DDLGenerator struct{}

// NewDDLGenerator 创建DDL生成器
func NewDDLGenerator() *DDLGenerator {
	return &DDLGenerator{}
}

// GenerateMySQLDDL 生成MySQL DDL
func (g *DDLGenerator) GenerateMySQLDDL(obj *store.SchemaObjectRow, cols []*store.ColumnRow) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", obj.Name))

	var colDefs []string
	var pkCols []string

	for _, col := range cols {
		colDef := g.generateMySQLColumnDef(col)
		colDefs = append(colDefs, "  "+colDef)

		if col.IsPrimaryKey {
			pkCols = append(pkCols, fmt.Sprintf("`%s`", col.Name))
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

	sb.WriteString(fmt.Sprintf("CREATE TABLE %s.%s (\n", schema, obj.Name))

	var colDefs []string
	var pkCols []string

	for _, col := range cols {
		colDef := g.generatePostgresColumnDef(col)
		colDefs = append(colDefs, "  "+colDef)

		if col.IsPrimaryKey {
			pkCols = append(pkCols, fmt.Sprintf("\"%s\"", col.Name))
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

	sb.WriteString(fmt.Sprintf("`%s` ", col.Name))

	// 数据类型映射
	dataType := g.mapToMySQLType(col.DataType, col.FullDataType)
	sb.WriteString(dataType)

	if !col.IsNullable {
		sb.WriteString(" NOT NULL")
	}

	if col.DefaultValue != nil && *col.DefaultValue != "" {
		sb.WriteString(fmt.Sprintf(" DEFAULT '%s'", *col.DefaultValue))
	}

	if col.IsUnique && !col.IsPrimaryKey {
		sb.WriteString(" UNIQUE")
	}

	return sb.String()
}

func (g *DDLGenerator) generatePostgresColumnDef(col *store.ColumnRow) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\"%s\" ", col.Name))

	// 数据类型映射
	dataType := g.mapToPostgresType(col.DataType, col.FullDataType)
	sb.WriteString(dataType)

	if !col.IsNullable {
		sb.WriteString(" NOT NULL")
	}

	if col.DefaultValue != nil && *col.DefaultValue != "" {
		sb.WriteString(fmt.Sprintf(" DEFAULT '%s'", *col.DefaultValue))
	}

	return sb.String()
}

func (g *DDLGenerator) mapToMySQLType(dataType, fullDataType string) string {
	if fullDataType != "" {
		return fullDataType
	}

	switch strings.ToLower(dataType) {
	case "int", "integer":
		return "INT"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "varchar", "string":
		return "VARCHAR(255)"
	case "text":
		return "TEXT"
	case "bool", "boolean":
		return "TINYINT(1)"
	case "float":
		return "FLOAT"
	case "double":
		return "DOUBLE"
	case "decimal":
		return "DECIMAL(18,2)"
	case "date":
		return "DATE"
	case "datetime":
		return "DATETIME"
	case "timestamp":
		return "TIMESTAMP"
	case "json":
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
	case strings.Contains(lower, "int"):
		return "INTEGER"
	case strings.Contains(lower, "varchar"):
		return mysqlType // 保留原样
	case strings.Contains(lower, "datetime"):
		return "TIMESTAMP"
	case strings.Contains(lower, "json"):
		return "JSONB"
	default:
		return mysqlType
	}
}
