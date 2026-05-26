package store

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jiangfire/datamaplite/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// testPostgresStore 根据环境变量创建 PostgreSQL 测试存储
func testPostgresStore(t *testing.T) Store {
	t.Helper()

	dsn := os.Getenv("DATAMAP_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("Skipping PostgreSQL test: DATAMAP_TEST_POSTGRES_DSN not set")
	}

	logger := zap.NewNop()

	// 解析 DSN 提取连接信息
	cfg := &config.DatabaseConfig{
		Type:            "postgres",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 0,
		MaxConnIdleTime: 0,
	}

	// 简单解析 host, port, user, password, dbname
	parts := strings.FieldsFunc(dsn, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t'
	})
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "host":
			cfg.Host = kv[1]
		case "port":
			if _, err := fmt.Sscanf(kv[1], "%d", &cfg.Port); err != nil {
				cfg.Port = 5432
			}
		case "user":
			cfg.Username = kv[1]
		case "password":
			cfg.Password = kv[1]
		case "dbname":
			cfg.Database = kv[1]
		case "sslmode":
			cfg.SSLMode = kv[1]
		}
	}

	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	ctx := context.Background()
	st, err := NewPostgresStore(ctx, cfg, logger)
	require.NoError(t, err)

	// 清理测试数据
	t.Cleanup(func() {
		cleanupPostgresTestData(ctx, st)
		require.NoError(t, st.Close())
	})

	return st
}

func cleanupPostgresTestData(ctx context.Context, st Store) {
	// 使用类型断言来执行清理
	if pg, ok := st.(*PostgresStore); ok {
		_, _ = pg.pool.Exec(ctx, `DELETE FROM user_notifications`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM notifications`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM alert_rules`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM column_tags`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM tags`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM dq_results`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM dq_rules`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM lineage_edges`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM column_mappings`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM columns`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM schema_changes`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM schema_objects`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM governance_outbox`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM sync_leases`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM business_terms`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM users`)
		_, _ = pg.pool.Exec(ctx, `DELETE FROM data_sources`)
	}
}

func TestPostgresDataSourceCRUD(t *testing.T) {
	ctx := context.Background()
	store := testPostgresStore(t)
	if store == nil {
		return
	}

	desc := "Test PostgreSQL source"
	source := &DataSourceCreate{
		Name:             "test-pg-source",
		Description:      &desc,
		Type:             "postgres",
		Host:             "localhost",
		Port:             5432,
		Database:         "testdb",
		ConnectionConfig: `{"user":"postgres","password":"secret"}`,
	}

	// Test Create
	id, err := store.CreateDataSource(ctx, source)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	// Test List
	sources, err := store.ListDataSources(ctx)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, "test-pg-source", sources[0].Name)
	assert.Equal(t, "localhost", sources[0].Host)

	sourceID := sources[0].ID

	// Test Get
	got, err := store.GetDataSource(ctx, sourceID)
	require.NoError(t, err)
	assert.Equal(t, sourceID, got.ID)
	assert.Equal(t, "test-pg-source", got.Name)

	// Test Get Not Found
	_, err = store.GetDataSource(ctx, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test Update
	newName := "updated-pg-source"
	newHost := "192.168.1.1"
	update := &DataSourceUpdate{
		Name: &newName,
		Host: &newHost,
	}
	err = store.UpdateDataSource(ctx, sourceID, update)
	require.NoError(t, err)

	got, err = store.GetDataSource(ctx, sourceID)
	require.NoError(t, err)
	assert.Equal(t, "updated-pg-source", got.Name)
	assert.Equal(t, "192.168.1.1", got.Host)

	// Test Update Sync Status
	errMsg := "connection refused"
	err = store.UpdateDataSourceSyncStatus(ctx, sourceID, "error", &errMsg)
	require.NoError(t, err)

	got, err = store.GetDataSource(ctx, sourceID)
	require.NoError(t, err)
	assert.Equal(t, "error", got.Status)
	assert.NotNil(t, got.LastSyncError)
	assert.Equal(t, "connection refused", *got.LastSyncError)

	// Test Delete
	err = store.DeleteDataSource(ctx, sourceID)
	require.NoError(t, err)

	sources, err = store.ListDataSources(ctx)
	require.NoError(t, err)
	assert.Len(t, sources, 0)
}

func TestPostgresSchemaObjectCRUD(t *testing.T) {
	ctx := context.Background()
	store := testPostgresStore(t)
	if store == nil {
		return
	}

	// 先创建数据源
	source := &DataSourceCreate{
		Name:             "test-pg-object-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	}
	_, err := store.CreateDataSource(ctx, source)
	require.NoError(t, err)

	sources, _ := store.ListDataSources(ctx)
	sourceID := sources[0].ID

	// Test Create Object
	schemaName := "public"
	rowCount := int64(1000)
	obj := &SchemaObjectCreate{
		SourceID:    sourceID,
		Name:        "users",
		Type:        "table",
		Schema:      &schemaName,
		ColumnCount: 5,
		RowCount:    &rowCount,
	}

	objID, err := store.CreateSchemaObject(ctx, obj)
	require.NoError(t, err)
	assert.NotEmpty(t, objID)

	// Test Get Object
	got, err := store.GetSchemaObject(ctx, objID)
	require.NoError(t, err)
	assert.Equal(t, "users", got.Name)
	assert.Equal(t, "table", got.Type)
	assert.Equal(t, int64(1000), *got.RowCount)

	// Test List Objects By Source
	objs, err := store.ListSchemaObjectsBySource(ctx, sourceID)
	require.NoError(t, err)
	assert.Len(t, objs, 1)
}

func TestPostgresColumnCRUD(t *testing.T) {
	ctx := context.Background()
	store := testPostgresStore(t)
	if store == nil {
		return
	}

	// 创建数据源和对象
	source := &DataSourceCreate{
		Name:             "test-pg-column-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	}
	_, err := store.CreateDataSource(ctx, source)
	require.NoError(t, err)

	sources, _ := store.ListDataSources(ctx)
	sourceID := sources[0].ID

	obj := &SchemaObjectCreate{
		SourceID:    sourceID,
		Name:        "users",
		Type:        "table",
		ColumnCount: 2,
	}
	objID, err := store.CreateSchemaObject(ctx, obj)
	require.NoError(t, err)

	// Test Create Column
	col := &ColumnCreate{
		ObjectID:        objID,
		Name:            "id",
		DataType:        "int",
		FullDataType:    "int(11)",
		IsNullable:      false,
		IsPrimaryKey:    true,
		OrdinalPosition: 1,
	}
	err = store.CreateColumn(ctx, col)
	require.NoError(t, err)

	// Create another column
	desc := "User email"
	col2 := &ColumnCreate{
		ObjectID:        objID,
		Name:            "email",
		DataType:        "varchar",
		FullDataType:    "varchar(255)",
		IsNullable:      false,
		IsUnique:        true,
		OrdinalPosition: 2,
		Description:     &desc,
	}
	err = store.CreateColumn(ctx, col2)
	require.NoError(t, err)

	// Test List Columns By Object
	cols, err := store.ListColumnsByObject(ctx, objID)
	require.NoError(t, err)
	assert.Len(t, cols, 2)
	assert.Equal(t, "id", cols[0].Name)
	assert.Equal(t, "email", cols[1].Name)
}

func TestPostgresTransaction(t *testing.T) {
	ctx := context.Background()
	store := testPostgresStore(t)
	if store == nil {
		return
	}

	// Test successful transaction
	err := store.WithTx(ctx, func(txStore Store) error {
		source := &DataSourceCreate{
			Name:             "tx-test-pg",
			Type:             "mysql",
			Host:             "localhost",
			Port:             3306,
			Database:         "testdb",
			ConnectionConfig: `{}`,
		}
		_, err := txStore.CreateDataSource(ctx, source)
		return err
	})
	require.NoError(t, err)

	// Verify data was committed
	sources, err := store.ListDataSources(ctx)
	require.NoError(t, err)
	assert.Len(t, sources, 1)
	assert.Equal(t, "tx-test-pg", sources[0].Name)

	// Test transaction rollback
	err = store.WithTx(ctx, func(txStore Store) error {
		_, err := txStore.CreateDataSource(ctx, &DataSourceCreate{
			Name:             "tx-rollback-test",
			Type:             "mysql",
			Host:             "localhost",
			Port:             3306,
			Database:         "testdb",
			ConnectionConfig: `{}`,
		})
		if err != nil {
			return err
		}
		return fmt.Errorf("intentional rollback")
	})
	require.Error(t, err)

	// Verify rollback - should still only have 1 source
	sources, err = store.ListDataSources(ctx)
	require.NoError(t, err)
	assert.Len(t, sources, 1)
}

func TestPostgresSyncLease(t *testing.T) {
	ctx := context.Background()
	store := testPostgresStore(t)
	if store == nil {
		return
	}

	sourceID, err := store.CreateDataSource(ctx, &DataSourceCreate{
		Name:             "lease-pg-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	})
	require.NoError(t, err)

	// Test acquire lease
	acquired, err := store.TryAcquireSyncLease(ctx, sourceID, "owner-a", "2026-06-24T10:00:00Z", "2026-06-24T10:01:00Z")
	require.NoError(t, err)
	assert.True(t, acquired)

	// Test second acquire should fail
	acquired, err = store.TryAcquireSyncLease(ctx, sourceID, "owner-b", "2026-06-24T10:00:00Z", "2026-06-24T10:02:00Z")
	require.NoError(t, err)
	assert.False(t, acquired)

	// Test get lease
	lease, err := store.GetSyncLease(ctx, sourceID)
	require.NoError(t, err)
	require.NotNil(t, lease)
	assert.Equal(t, sourceID, lease.SourceID)
	assert.Equal(t, "owner-a", lease.OwnerID)

	// Test release lease
	err = store.ReleaseSyncLease(ctx, sourceID, "owner-a")
	require.NoError(t, err)

	// Verify released
	lease, err = store.GetSyncLease(ctx, sourceID)
	require.NoError(t, err)
	assert.Nil(t, lease)
}

func TestPostgresGovernanceOutbox(t *testing.T) {
	ctx := context.Background()
	store := testPostgresStore(t)
	if store == nil {
		return
	}

	event := &GovernanceOutboxEventCreate{
		ID:            "pg-outbox-1",
		EventID:       "pg-evt-1",
		EventType:     "metadata.schema.changed",
		TraceID:       "pg-trace-1",
		ResourceType:  "schema_change",
		ResourceID:    "pg-chg-1",
		Payload:       `{"event_id":"pg-evt-1"}`,
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: "2026-06-24T10:00:00Z",
	}

	// Test enqueue
	created, err := store.EnqueueGovernanceOutboxEvent(ctx, event)
	require.NoError(t, err)
	assert.True(t, created)

	// Test duplicate enqueue
	created, err = store.EnqueueGovernanceOutboxEvent(ctx, event)
	require.NoError(t, err)
	assert.False(t, created)

	// Test claim
	rows, err := store.ClaimGovernanceOutboxEvents(ctx, "dispatcher-a", "2026-06-24T10:00:00Z", "2026-06-24T10:05:00Z", 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].AttemptCount)

	// Test mark delivered
	err = store.MarkGovernanceOutboxDelivered(ctx, "pg-outbox-1", "2026-06-24T10:01:00Z")
	require.NoError(t, err)

	// Test stats
	stats, err := store.GetGovernanceOutboxStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.EqualValues(t, 1, stats.DeliveredCount)
	assert.EqualValues(t, 0, stats.PendingCount)
}

func TestPostgresDeleteCascade(t *testing.T) {
	ctx := context.Background()
	store := testPostgresStore(t)
	if store == nil {
		return
	}

	// 创建数据源
	source := &DataSourceCreate{
		Name:             "cascade-pg-test",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	}
	_, err := store.CreateDataSource(ctx, source)
	require.NoError(t, err)

	sources, _ := store.ListDataSources(ctx)
	sourceID := sources[0].ID

	// 创建对象
	obj := &SchemaObjectCreate{
		SourceID:    sourceID,
		Name:        "users",
		Type:        "table",
		ColumnCount: 1,
	}
	objID, _ := store.CreateSchemaObject(ctx, obj)

	// 创建字段
	col := &ColumnCreate{
		ObjectID:        objID,
		Name:            "id",
		DataType:        "int",
		OrdinalPosition: 1,
	}
	err = store.CreateColumn(ctx, col)
	require.NoError(t, err)

	// 验证对象和字段存在
	objs, _ := store.ListSchemaObjectsBySource(ctx, sourceID)
	assert.Len(t, objs, 1)

	cols, _ := store.ListColumnsByObject(ctx, objID)
	assert.Len(t, cols, 1)

	// 删除数据源（应该级联删除对象和字段）
	err = store.DeleteDataSource(ctx, sourceID)
	require.NoError(t, err)

	// 验证级联删除
	objs, _ = store.ListSchemaObjectsBySource(ctx, sourceID)
	assert.Len(t, objs, 0)
}
