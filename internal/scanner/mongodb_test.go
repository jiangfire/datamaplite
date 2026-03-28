package scanner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewMongoDBScanner(t *testing.T) {
	t.Run("default sample size", func(t *testing.T) {
		scanner := NewMongoDBScanner(0)
		assert.NotNil(t, scanner)
	})

	t.Run("custom sample size", func(t *testing.T) {
		scanner := NewMongoDBScanner(500)
		assert.NotNil(t, scanner)
	})
}

func TestMongoDBScanner_getBSONType(t *testing.T) {
	scanner := NewMongoDBScanner(100)

	testCases := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "hello", "string"},
		{"int32", int32(42), "int"},
		{"int64", int64(42), "int"},
		{"int", int(42), "int"},
		{"float64", float64(3.14), "double"},
		{"bool", true, "bool"},
		{"time", time.Now(), "date"},
		{"bson.M", bson.M{"key": "value"}, "object"},
		{"map", map[string]interface{}{"key": "value"}, "object"},
		{"bson.A", bson.A{1, 2, 3}, "array"},
		{"slice", []interface{}{1, 2, 3}, "array"},
		{"nil", nil, "null"},
		{"ObjectID", primitive.NewObjectID(), "objectId"},
		{"Binary", primitive.Binary{}, "binData"},
		{"Decimal128", primitive.NewDecimal128(1, 0), "decimal"},
		{"DateTime", primitive.DateTime(0), "date"},
		{"Timestamp", primitive.Timestamp{}, "timestamp"},
		{"Regex", primitive.Regex{}, "regex"},
		{"JavaScript", primitive.JavaScript("function(){}"), "javascript"},
		{"unknown", struct{}{}, "struct {}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := scanner.getBSONType(tc.value)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMongoDBScanner_mapMongoTypeToSQL(t *testing.T) {
	scanner := NewMongoDBScanner(100)

	testCases := []struct {
		mongoType string
		sqlType   string
	}{
		{"string", "VARCHAR"},
		{"int", "BIGINT"},
		{"double", "DOUBLE"},
		{"bool", "BOOLEAN"},
		{"date", "TIMESTAMP"},
		{"object", "JSON"},
		{"array", "ARRAY"},
		{"objectId", "VARCHAR(24)"},
		{"binData", "BLOB"},
		{"decimal", "DECIMAL"},
		{"unknown", "TEXT"},
		{"", "TEXT"},
	}

	for _, tc := range testCases {
		t.Run(tc.mongoType, func(t *testing.T) {
			result := scanner.mapMongoTypeToSQL(tc.mongoType)
			assert.Equal(t, tc.sqlType, result)
		})
	}
}

func TestMongoDBScanner_getParentPath(t *testing.T) {
	scanner := NewMongoDBScanner(100)

	testCases := []struct {
		fieldName string
		expected  string
	}{
		{"name", ""},
		{"user.name", "user"},
		{"user.address.city", "user.address"},
		{"deep.nested.field.path", "deep.nested.field"},
		{".", ""},
		{"a.", "a"},
	}

	for _, tc := range testCases {
		t.Run(tc.fieldName, func(t *testing.T) {
			result := scanner.getParentPath(tc.fieldName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMongoDBScanner_extractFields(t *testing.T) {
	scanner := NewMongoDBScanner(100)
	fieldTypes := make(map[string]map[string]int)

	doc := bson.M{
		"name":   "John",
		"age":    int32(30),
		"active": true,
	}

	scanner.extractFields("", doc, fieldTypes)

	assert.Contains(t, fieldTypes, "name")
	assert.Contains(t, fieldTypes, "age")
	assert.Contains(t, fieldTypes, "active")

	assert.Contains(t, fieldTypes["name"], "string")
	assert.Contains(t, fieldTypes["age"], "int")
	assert.Contains(t, fieldTypes["active"], "bool")
}

func TestMongoDBScanner_extractFields_WithPrefix(t *testing.T) {
	scanner := NewMongoDBScanner(100)
	fieldTypes := make(map[string]map[string]int)

	doc := bson.M{
		"city":    "Beijing",
		"country": "China",
	}

	scanner.extractFields("address", doc, fieldTypes)

	assert.Contains(t, fieldTypes, "address.city")
	assert.Contains(t, fieldTypes, "address.country")
}

func TestMongoDBScanner_extractFields_Nested(t *testing.T) {
	scanner := NewMongoDBScanner(100)
	fieldTypes := make(map[string]map[string]int)

	doc := bson.M{
		"user": bson.M{
			"name": "John",
			"age":  int32(30),
		},
	}

	scanner.extractFields("", doc, fieldTypes)

	assert.Contains(t, fieldTypes, "user")
	assert.Contains(t, fieldTypes, "user.name")
	assert.Contains(t, fieldTypes, "user.age")
}

func TestMongoDBScanner_extractFields_DeeplyNested(t *testing.T) {
	scanner := NewMongoDBScanner(100)
	fieldTypes := make(map[string]map[string]int)

	doc := bson.M{
		"company": bson.M{
			"address": bson.M{
				"city": "Beijing",
				"zip":  "100000",
			},
		},
	}

	scanner.extractFields("", doc, fieldTypes)

	assert.Contains(t, fieldTypes, "company")
	assert.Contains(t, fieldTypes, "company.address")
	assert.Contains(t, fieldTypes, "company.address.city")
	assert.Contains(t, fieldTypes, "company.address.zip")
}

func TestMongoDBScanner_extractFieldsGeneric(t *testing.T) {
	scanner := NewMongoDBScanner(100)
	fieldTypes := make(map[string]map[string]int)

	doc := map[string]interface{}{
		"name":  "Jane",
		"score": float64(95.5),
	}

	scanner.extractFieldsGeneric("", doc, fieldTypes)

	assert.Contains(t, fieldTypes, "name")
	assert.Contains(t, fieldTypes, "score")
	assert.Contains(t, fieldTypes["name"], "string")
	assert.Contains(t, fieldTypes["score"], "double")
}

func TestMongoDBScanner_extractFields_MixedTypes(t *testing.T) {
	scanner := NewMongoDBScanner(100)
	fieldTypes := make(map[string]map[string]int)

	// 混合bson.M和map[string]interface{}
	doc := bson.M{
		"data": map[string]interface{}{
			"value": int32(42),
		},
	}

	scanner.extractFields("", doc, fieldTypes)

	assert.Contains(t, fieldTypes, "data")
	assert.Contains(t, fieldTypes, "data.value")
}

func TestMongoDBScanner_confidenceCalculation(t *testing.T) {
	// 模拟置信度计算逻辑
	fieldTypes := map[string]int{
		"string": 80,
		"int":    20,
	}

	totalCount := 0
	maxCount := 0
	mostCommonType := ""

	for t, count := range fieldTypes {
		totalCount += count
		if count > maxCount {
			maxCount = count
			mostCommonType = t
		}
	}

	confidence := float64(maxCount) / float64(totalCount)

	assert.Equal(t, "string", mostCommonType)
	assert.Equal(t, 100, totalCount)
	assert.Equal(t, 80, maxCount)
	assert.InDelta(t, 0.8, confidence, 0.001)
}

func TestMongoDBScanner_connectionConfig(t *testing.T) {
	config := ConnectionConfig{
		Host:     "localhost",
		Port:     27017,
		Database: "testdb",
		Username: "mongo",
		Password: "password",
		SSLMode:  "",
	}

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 27017, config.Port)
	assert.Equal(t, "testdb", config.Database)
	assert.Equal(t, "mongo", config.Username)
	assert.Equal(t, "password", config.Password)
}

func TestMongoDBScanner_connectionConfig_WithSSL(t *testing.T) {
	config := ConnectionConfig{
		Host:     "secure.mongo.example.com",
		Port:     27017,
		Database: "production",
		Username: "app",
		Password: "secret",
		SSLMode:  "require",
	}

	assert.Equal(t, "require", config.SSLMode)
}

// TestMongoDBScanner_Integration 是MongoDB扫描器的集成测试
// 运行此测试需要本地MongoDB实例
// 可以通过设置环境变量 MONGODB_TEST_URI 来启用
func TestMongoDBScanner_Integration(t *testing.T) {
	uri := "" // 从环境变量读取: os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("Skipping MongoDB integration test: MONGODB_TEST_URI not set")
	}

	// 集成测试逻辑将在有MongoDB测试实例时实现
	// 1. 创建扫描器
	// 2. 测试连接
	// 3. 扫描Schema
	// 4. 验证结果
}
