package scanner

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBScanner MongoDB扫描器
type MongoDBScanner struct {
	sampleSize int
}

// NewMongoDBScanner 创建MongoDB扫描器
func NewMongoDBScanner(sampleSize int) *MongoDBScanner {
	if sampleSize <= 0 {
		sampleSize = 1000
	}
	return &MongoDBScanner{sampleSize: sampleSize}
}

// TestConnection 测试MongoDB连接
func (s *MongoDBScanner) TestConnection(ctx context.Context, config ConnectionConfig) error {
	client, err := s.connect(ctx, config)
	if err != nil {
		return err
	}
	defer func() {
		ignoreError(client.Disconnect(ctx))
	}()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return client.Ping(ctx, nil)
}

// ScanSchema 扫描MongoDB Schema（抽样推断）
func (s *MongoDBScanner) ScanSchema(ctx context.Context, config ConnectionConfig) (*SchemaInfo, error) {
	client, err := s.connect(ctx, config)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(client.Disconnect(ctx))
	}()

	db := client.Database(config.Database)

	// 获取所有集合
	collectionNames, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var objects []ObjectInfo
	for _, collName := range collectionNames {
		obj, err := s.inferCollectionSchema(ctx, db, collName)
		if err != nil {
			return nil, fmt.Errorf("failed to infer schema for collection %s: %w", collName, err)
		}
		objects = append(objects, *obj)
	}

	return &SchemaInfo{Objects: objects}, nil
}

// connect 建立连接
func (s *MongoDBScanner) connect(ctx context.Context, config ConnectionConfig) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	clientOpts := options.Client().ApplyURI(uri)
	if config.SSLMode == "require" {
		clientOpts.SetTLSConfig(nil) // 使用系统默认TLS配置
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			return nil, fmt.Errorf("failed to ping mongodb: %w (disconnect failed: %v)", err, disconnectErr)
		}
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return client, nil
}

// inferCollectionSchema 推断集合的Schema
func (s *MongoDBScanner) inferCollectionSchema(ctx context.Context, db *mongo.Database, collName string) (*ObjectInfo, error) {
	coll := db.Collection(collName)

	obj := ObjectInfo{
		Name:    collName,
		Type:    "collection",
		Columns: []ColumnInfo{},
	}

	// 获取文档数量（估计值）
	stats, err := db.RunCommand(ctx, bson.M{"collStats": collName}).Raw()
	if err == nil {
		if count, ok := stats.Lookup("count").Int64OK(); ok {
			obj.RowCount = &count
		}
		if size, ok := stats.Lookup("size").Int64OK(); ok {
			obj.SizeBytes = &size
		}
	}

	// 抽样推断Schema
	fieldTypes := make(map[string]map[string]int) // field -> type -> count

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$sample", Value: bson.M{"size": s.sampleSize}}},
	}

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() {
		ignoreError(cursor.Close(ctx))
	}()

	sampleCount := 0
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		sampleCount++
		s.extractFields("", doc, fieldTypes)
	}

	// 转换字段推断结果为ColumnInfo
	position := 1
	for fieldName, types := range fieldTypes {
		totalCount := 0
		mostCommonType := ""
		maxCount := 0

		for t, count := range types {
			totalCount += count
			if count > maxCount {
				maxCount = count
				mostCommonType = t
			}
		}

		confidence := float64(maxCount) / float64(totalCount)
		if sampleCount == 0 {
			confidence = 0
		}

		col := ColumnInfo{
			Name:            fieldName,
			DataType:        s.mapMongoTypeToSQL(mostCommonType),
			FullDataType:    mostCommonType,
			IsNullable:      confidence < 1.0, // 不是所有文档都有此字段则为nullable
			OrdinalPosition: position,
			Confidence:      confidence,
		}

		// 检查是否有嵌套路径
		if parentPath := s.getParentPath(fieldName); parentPath != "" {
			col.ParentColumnPath = &parentPath
		}

		obj.Columns = append(obj.Columns, col)
		position++
	}

	return &obj, nil
}

// extractFields 递归提取字段
func (s *MongoDBScanner) extractFields(prefix string, doc bson.M, fieldTypes map[string]map[string]int) {
	for key, value := range doc {
		fieldName := key
		if prefix != "" {
			fieldName = prefix + "." + key
		}

		fieldType := s.getBSONType(value)
		if fieldTypes[fieldName] == nil {
			fieldTypes[fieldName] = make(map[string]int)
		}
		fieldTypes[fieldName][fieldType]++

		// 递归处理嵌套文档
		if nestedDoc, ok := value.(bson.M); ok {
			s.extractFields(fieldName, nestedDoc, fieldTypes)
		} else if nestedDoc, ok := value.(map[string]interface{}); ok {
			s.extractFieldsGeneric(fieldName, nestedDoc, fieldTypes)
		}
	}
}

// extractFieldsGeneric 递归提取字段（map[string]interface{}版本）
func (s *MongoDBScanner) extractFieldsGeneric(prefix string, doc map[string]interface{}, fieldTypes map[string]map[string]int) {
	for key, value := range doc {
		fieldName := key
		if prefix != "" {
			fieldName = prefix + "." + key
		}

		fieldType := s.getBSONType(value)
		if fieldTypes[fieldName] == nil {
			fieldTypes[fieldName] = make(map[string]int)
		}
		fieldTypes[fieldName][fieldType]++

		// 递归处理嵌套文档
		if nestedDoc, ok := value.(map[string]interface{}); ok {
			s.extractFieldsGeneric(fieldName, nestedDoc, fieldTypes)
		} else if nestedDoc, ok := value.(bson.M); ok {
			s.extractFields(fieldName, nestedDoc, fieldTypes)
		}
	}
}

// getBSONType 获取BSON值的类型
func (s *MongoDBScanner) getBSONType(value interface{}) string {
	switch v := value.(type) {
	case string:
		return "string"
	case int32, int64, int:
		return "int"
	case float64:
		return "double"
	case bool:
		return "bool"
	case time.Time:
		return "date"
	case bson.M, map[string]interface{}:
		return "object"
	case bson.A, []interface{}:
		return "array"
	case nil:
		return "null"
	case primitive.ObjectID:
		return "objectId"
	case primitive.Binary:
		return "binData"
	case primitive.Decimal128:
		return "decimal"
	case primitive.DateTime:
		return "date"
	case primitive.Timestamp:
		return "timestamp"
	case primitive.Regex:
		return "regex"
	case primitive.JavaScript:
		return "javascript"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// mapMongoTypeToSQL 将MongoDB类型映射为SQL类型
func (s *MongoDBScanner) mapMongoTypeToSQL(mongoType string) string {
	switch mongoType {
	case "string":
		return "VARCHAR"
	case "int":
		return "BIGINT"
	case "double":
		return "DOUBLE"
	case "bool":
		return "BOOLEAN"
	case "date":
		return "TIMESTAMP"
	case "object":
		return "JSON"
	case "array":
		return "ARRAY"
	case "objectId":
		return "VARCHAR(24)"
	case "binData":
		return "BLOB"
	case "decimal":
		return "DECIMAL"
	default:
		return "TEXT"
	}
}

// getParentPath 获取父路径
func (s *MongoDBScanner) getParentPath(fieldName string) string {
	for i := len(fieldName) - 1; i >= 0; i-- {
		if fieldName[i] == '.' {
			return fieldName[:i]
		}
	}
	return ""
}
