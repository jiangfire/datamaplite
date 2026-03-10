package scanner

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockScanner 模拟Scanner接口
type MockScanner struct {
	TestConnectionFunc func(ctx context.Context, config ConnectionConfig) error
	ScanSchemaFunc     func(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error)
}

func (m *MockScanner) TestConnection(ctx context.Context, config ConnectionConfig) error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx, config)
	}
	return nil
}

func (m *MockScanner) ScanSchema(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error) {
	if m.ScanSchemaFunc != nil {
		return m.ScanSchemaFunc(ctx, config)
	}
	return &SchemaInfo{}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	mockScanner := &MockScanner{}

	// 注册扫描器
	registry.Register("mysql", mockScanner)

	// 获取扫描器
	scanner, err := registry.Get("mysql")

	require.NoError(t, err)
	assert.Equal(t, mockScanner, scanner)
}

func TestRegistry_Get_UnsupportedType(t *testing.T) {
	registry := NewRegistry()

	// 尝试获取未注册的扫描器
	scanner, err := registry.Get("unsupported")

	assert.Error(t, err)
	assert.Nil(t, scanner)
	assert.Contains(t, err.Error(), "unsupported database type")
}

func TestRegistry_MultipleScanners(t *testing.T) {
	registry := NewRegistry()
	mysqlScanner := &MockScanner{}
	mongoScanner := &MockScanner{}
	postgresScanner := &MockScanner{}

	// 注册多个扫描器
	registry.Register("mysql", mysqlScanner)
	registry.Register("mongodb", mongoScanner)
	registry.Register("postgres", postgresScanner)

	// 验证每个扫描器都能正确获取
	gotMySQL, err := registry.Get("mysql")
	require.NoError(t, err)
	assert.Equal(t, mysqlScanner, gotMySQL)

	gotMongo, err := registry.Get("mongodb")
	require.NoError(t, err)
	assert.Equal(t, mongoScanner, gotMongo)

	gotPostgres, err := registry.Get("postgres")
	require.NoError(t, err)
	assert.Equal(t, postgresScanner, gotPostgres)
}

func TestRegistry_Overwrite(t *testing.T) {
	registry := NewRegistry()
	oldScanner := &MockScanner{}
	newScanner := &MockScanner{}

	// 注册旧扫描器
	registry.Register("mysql", oldScanner)

	// 覆盖注册新扫描器
	registry.Register("mysql", newScanner)

	// 验证新扫描器被返回
	got, err := registry.Get("mysql")
	require.NoError(t, err)
	assert.Equal(t, newScanner, got)
}

func TestConnectionConfig_ToJSON(t *testing.T) {
	config := ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Username: "root",
		Password: "secret",
		SSLMode:  "require",
	}

	jsonStr, err := config.ToJSON()

	require.NoError(t, err)
	assert.Contains(t, jsonStr, "localhost")
	assert.Contains(t, jsonStr, "3306")
	assert.Contains(t, jsonStr, "testdb")
	assert.Contains(t, jsonStr, "root")
	assert.Contains(t, jsonStr, "secret")
	assert.Contains(t, jsonStr, "require")
}

func TestConnectionConfig_ToJSON_Empty(t *testing.T) {
	config := ConnectionConfig{}

	jsonStr, err := config.ToJSON()

	require.NoError(t, err)
	// 空结构体会序列化为包含零值字段的JSON
	assert.Contains(t, jsonStr, "host")
	assert.Contains(t, jsonStr, "port")
}

func TestConnectionConfigFromJSON(t *testing.T) {
	jsonStr := `{"host":"localhost","port":3306,"database":"testdb","username":"root","password":"secret","ssl_mode":"require"}`

	config, err := ConnectionConfigFromJSON(jsonStr)

	require.NoError(t, err)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 3306, config.Port)
	assert.Equal(t, "testdb", config.Database)
	assert.Equal(t, "root", config.Username)
	assert.Equal(t, "secret", config.Password)
	assert.Equal(t, "require", config.SSLMode)
}

func TestConnectionConfigFromJSON_InvalidJSON(t *testing.T) {
	jsonStr := `{"host":"localhost",invalid json}`

	config, err := ConnectionConfigFromJSON(jsonStr)

	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestConnectionConfig_RoundTrip(t *testing.T) {
	original := ConnectionConfig{
		Host:     "db.example.com",
		Port:     5432,
		Database: "production",
		Username: "admin",
		Password: "complex-password-123",
		SSLMode:  "verify-full",
	}

	jsonStr, err := original.ToJSON()
	require.NoError(t, err)

	parsed, err := ConnectionConfigFromJSON(jsonStr)
	require.NoError(t, err)

	assert.Equal(t, original.Host, parsed.Host)
	assert.Equal(t, original.Port, parsed.Port)
	assert.Equal(t, original.Database, parsed.Database)
	assert.Equal(t, original.Username, parsed.Username)
	assert.Equal(t, original.Password, parsed.Password)
	assert.Equal(t, original.SSLMode, parsed.SSLMode)
}

func TestMockScanner_TestConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &MockScanner{
			TestConnectionFunc: func(ctx context.Context, config ConnectionConfig) error {
				return nil
			},
		}

		err := mock.TestConnection(context.Background(), ConnectionConfig{})
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("connection failed")
		mock := &MockScanner{
			TestConnectionFunc: func(ctx context.Context, config ConnectionConfig) error {
				return expectedErr
			},
		}

		err := mock.TestConnection(context.Background(), ConnectionConfig{})
		assert.Equal(t, expectedErr, err)
	})
}

func TestMockScanner_ScanSchema(t *testing.T) {
	t.Run("returns schema info", func(t *testing.T) {
		expectedInfo := &SchemaInfo{
			Objects: []ObjectInfo{
				{Name: "table1", Type: "table"},
			},
		}
		mock := &MockScanner{
			ScanSchemaFunc: func(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error) {
				return expectedInfo, nil
			},
		}

		info, err := mock.ScanSchema(context.Background(), ConnectionConfig{})
		require.NoError(t, err)
		assert.Equal(t, expectedInfo, info)
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("scan failed")
		mock := &MockScanner{
			ScanSchemaFunc: func(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error) {
				return nil, expectedErr
			},
		}

		info, err := mock.ScanSchema(context.Background(), ConnectionConfig{})
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Equal(t, expectedErr, err)
	})
}

func TestSchemaInfo(t *testing.T) {
	info := SchemaInfo{
		Objects: []ObjectInfo{
			{
				Name: "users",
				Type: "table",
				Columns: []ColumnInfo{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "varchar"},
				},
			},
		},
	}

	assert.Len(t, info.Objects, 1)
	assert.Equal(t, "users", info.Objects[0].Name)
	assert.Len(t, info.Objects[0].Columns, 2)
}

func TestObjectInfo_WithOptionalFields(t *testing.T) {
	schema := "public"
	desc := "A test table"
	rowCount := int64(100)
	sizeBytes := int64(1024)

	obj := ObjectInfo{
		Name:        "test_table",
		Type:        "table",
		Schema:      &schema,
		Description: &desc,
		RowCount:    &rowCount,
		SizeBytes:   &sizeBytes,
		Columns:     []ColumnInfo{},
	}

	assert.Equal(t, "test_table", obj.Name)
	assert.Equal(t, "table", obj.Type)
	assert.Equal(t, &schema, obj.Schema)
	assert.Equal(t, &desc, obj.Description)
	assert.Equal(t, &rowCount, obj.RowCount)
	assert.Equal(t, &sizeBytes, obj.SizeBytes)
}

func TestColumnInfo_WithOptionalFields(t *testing.T) {
	defaultVal := "0"
	desc := "Primary key"
	parentPath := "user.address"

	col := ColumnInfo{
		Name:             "id",
		DataType:         "int",
		FullDataType:     "int(11)",
		IsNullable:       false,
		DefaultValue:     &defaultVal,
		IsPrimaryKey:     true,
		IsUnique:         true,
		OrdinalPosition:  1,
		Description:      &desc,
		ParentColumnPath: &parentPath,
		Confidence:       0.95,
	}

	assert.Equal(t, "id", col.Name)
	assert.Equal(t, "int", col.DataType)
	assert.Equal(t, "int(11)", col.FullDataType)
	assert.False(t, col.IsNullable)
	assert.Equal(t, &defaultVal, col.DefaultValue)
	assert.True(t, col.IsPrimaryKey)
	assert.True(t, col.IsUnique)
	assert.Equal(t, 1, col.OrdinalPosition)
	assert.Equal(t, &desc, col.Description)
	assert.Equal(t, &parentPath, col.ParentColumnPath)
	assert.InDelta(t, 0.95, col.Confidence, 0.001)
}
