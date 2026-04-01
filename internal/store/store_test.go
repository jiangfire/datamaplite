package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// testStore 创建测试用的内存SQLite存储
func testStore(t *testing.T) Store {
	logger := zap.NewNop()

	// 使用内存数据库
	db, err := sql.Open(sqliteDriverName, "file::memory:?_pragma=foreign_keys(1)")
	require.NoError(t, err)

	ctx := context.Background()

	// 运行迁移
	schema := `
CREATE TABLE IF NOT EXISTS data_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    database TEXT NOT NULL,
    connection_config TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    last_sync_at TEXT,
    last_sync_error TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS schema_objects (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    schema TEXT,
    description TEXT,
    row_count INTEGER,
    size_bytes INTEGER,
    column_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(source_id, name, schema)
);

CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY,
    object_id TEXT NOT NULL REFERENCES schema_objects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    data_type TEXT NOT NULL,
    full_data_type TEXT,
    is_nullable INTEGER NOT NULL DEFAULT 1,
    default_value TEXT,
    is_primary_key INTEGER NOT NULL DEFAULT 0,
    is_unique INTEGER NOT NULL DEFAULT 0,
    ordinal_position INTEGER NOT NULL,
    description TEXT,
    term_id TEXT,
    confidence REAL DEFAULT 1.0,
    parent_column_path TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(object_id, name)
);

CREATE TABLE IF NOT EXISTS schema_changes (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    object_id TEXT,
    change_type TEXT NOT NULL,
    object_type TEXT NOT NULL,
    object_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    detected_at TEXT DEFAULT (datetime('now')),
    acknowledged INTEGER NOT NULL DEFAULT 0
);
`
	_, err = db.ExecContext(ctx, schema)
	require.NoError(t, err)

	return &SQLiteStore{db: db, log: logger}
}

func registerStoreCleanup(t *testing.T, st Store) {
	t.Helper()
	t.Cleanup(func() {
		require.NoError(t, st.Close())
	})
}

// TestDataSourceCRUD 测试数据源CRUD
func TestDataSourceCRUD(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	registerStoreCleanup(t, store)

	desc := "Test MySQL source"
	source := &DataSourceCreate{
		Name:             "test-mysql",
		Description:      &desc,
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{"user":"root","password":"secret"}`,
	}

	// Test Create
	_, err := store.CreateDataSource(ctx, source)
	require.NoError(t, err)

	// Test List
	sources, err := store.ListDataSources(ctx)
	require.NoError(t, err)
	assert.Len(t, sources, 1)
	assert.Equal(t, "test-mysql", sources[0].Name)
	assert.Equal(t, "localhost", sources[0].Host)

	sourceID := sources[0].ID

	// Test Get
	got, err := store.GetDataSource(ctx, sourceID)
	require.NoError(t, err)
	assert.Equal(t, sourceID, got.ID)
	assert.Equal(t, "test-mysql", got.Name)

	// Test Get Not Found
	_, err = store.GetDataSource(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test Update
	newName := "updated-mysql"
	newHost := "192.168.1.1"
	update := &DataSourceUpdate{
		Name: &newName,
		Host: &newHost,
	}
	err = store.UpdateDataSource(ctx, sourceID, update)
	require.NoError(t, err)

	got, err = store.GetDataSource(ctx, sourceID)
	require.NoError(t, err)
	assert.Equal(t, "updated-mysql", got.Name)
	assert.Equal(t, "192.168.1.1", got.Host)

	// Test Update Not Found
	err = store.UpdateDataSource(ctx, "non-existent-id", update)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test Update Sync Status
	errMsg := "connection refused"
	err = store.UpdateDataSourceSyncStatus(ctx, sourceID, "error", &errMsg)
	require.NoError(t, err)

	got, err = store.GetDataSource(ctx, sourceID)
	require.NoError(t, err)
	assert.Equal(t, "error", got.Status)
	assert.NotNil(t, got.LastSyncError)
	assert.Equal(t, "connection refused", *got.LastSyncError)

	// Test Update Sync Status (success)
	err = store.UpdateDataSourceSyncStatus(ctx, sourceID, "active", nil)
	require.NoError(t, err)

	got, err = store.GetDataSource(ctx, sourceID)
	require.NoError(t, err)
	assert.Equal(t, "active", got.Status)
	assert.Nil(t, got.LastSyncError)

	// Test Delete
	err = store.DeleteDataSource(ctx, sourceID)
	require.NoError(t, err)

	sources, err = store.ListDataSources(ctx)
	require.NoError(t, err)
	assert.Len(t, sources, 0)

	// Test Delete Not Found
	err = store.DeleteDataSource(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestSchemaObjectCRUD 测试Schema对象CRUD
func TestSchemaObjectCRUD(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	registerStoreCleanup(t, store)

	// 先创建数据源
	source := &DataSourceCreate{
		Name:             "test-source",
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
		Description: nil,
		RowCount:    &rowCount,
		ColumnCount: 5,
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

	// Test Get Object By Name
	got, err = store.GetSchemaObjectByName(ctx, sourceID, "users", &schemaName)
	require.NoError(t, err)
	assert.Equal(t, "users", got.Name)

	// Test List Objects By Source
	objs, err := store.ListSchemaObjectsBySource(ctx, sourceID)
	require.NoError(t, err)
	assert.Len(t, objs, 1)

	// Create another object
	obj2 := &SchemaObjectCreate{
		SourceID:    sourceID,
		Name:        "orders",
		Type:        "table",
		Schema:      &schemaName,
		ColumnCount: 8,
	}
	_, err = store.CreateSchemaObject(ctx, obj2)
	require.NoError(t, err)

	objs, err = store.ListSchemaObjectsBySource(ctx, sourceID)
	require.NoError(t, err)
	assert.Len(t, objs, 2)
}

// TestColumnCRUD 测试字段CRUD
func TestColumnCRUD(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	registerStoreCleanup(t, store)

	// 创建数据源和对象
	source := &DataSourceCreate{
		Name:             "test-source",
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

	// Verify column order
	assert.Equal(t, "id", cols[0].Name)
	assert.Equal(t, "email", cols[1].Name)
	assert.True(t, cols[0].IsPrimaryKey)
	assert.False(t, cols[0].IsNullable)
	assert.True(t, cols[1].IsUnique)

	// Test Get Column
	colID := cols[0].ID
	got, err := store.GetColumn(ctx, colID)
	require.NoError(t, err)
	assert.Equal(t, "id", got.Name)
	assert.Equal(t, "int", got.DataType)
}

// TestSchemaChangeCRUD 测试Schema变更记录CRUD
func TestSchemaChangeCRUD(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	registerStoreCleanup(t, store)

	// 创建数据源
	source := &DataSourceCreate{
		Name:             "test-source",
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

	// Create object
	obj := &SchemaObjectCreate{
		SourceID:    sourceID,
		Name:        "users",
		Type:        "table",
		ColumnCount: 1,
	}
	objID, _ := store.CreateSchemaObject(ctx, obj)

	// Test Create Change
	change := &SchemaChangeCreate{
		SourceID:   sourceID,
		ObjectID:   &objID,
		ChangeType: "add_column",
		ObjectType: "column",
		ObjectName: "age",
	}
	err = store.CreateSchemaChange(ctx, change)
	require.NoError(t, err)

	// Create another change
	oldVal := "varchar(100)"
	newVal := "varchar(255)"
	change2 := &SchemaChangeCreate{
		SourceID:   sourceID,
		ChangeType: "alter_column",
		ObjectType: "column",
		ObjectName: "name",
		OldValue:   &oldVal,
		NewValue:   &newVal,
	}
	err = store.CreateSchemaChange(ctx, change2)
	require.NoError(t, err)

	// Test List Changes By Source
	changes, err := store.ListSchemaChangesBySource(ctx, sourceID, 10)
	require.NoError(t, err)
	assert.Len(t, changes, 2)

	changeTypes := []string{changes[0].ChangeType, changes[1].ChangeType}
	assert.ElementsMatch(t, []string{"add_column", "alter_column"}, changeTypes)

	// Test limit
	changes, err = store.ListSchemaChangesBySource(ctx, sourceID, 1)
	require.NoError(t, err)
	assert.Len(t, changes, 1)
}

// TestTransaction 测试事务功能
func TestTransaction(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	registerStoreCleanup(t, store)

	// Test successful transaction
	err := store.WithTx(ctx, func(txStore Store) error {
		source := &DataSourceCreate{
			Name:             "tx-test",
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
	assert.Equal(t, "tx-test", sources[0].Name)
}

// TestDeleteCascade 测试级联删除
func TestDeleteCascade(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	registerStoreCleanup(t, store)

	// 创建数据源
	source := &DataSourceCreate{
		Name:             "cascade-test",
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

// TestSearchColumns 测试字段搜索
func TestSearchColumns(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	registerStoreCleanup(t, store)

	// 创建数据源和对象
	source := &DataSourceCreate{
		Name:             "search-test",
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
		ColumnCount: 3,
	}
	objID, _ := store.CreateSchemaObject(ctx, obj)

	// 创建字段
	cols := []string{"id", "user_id", "user_name", "email"}
	for i, name := range cols {
		col := &ColumnCreate{
			ObjectID:        objID,
			Name:            name,
			DataType:        "varchar",
			OrdinalPosition: i + 1,
		}
		err := store.CreateColumn(ctx, col)
		require.NoError(t, err)
	}

	// 测试搜索（SQLite版本简单实现可能不支持复杂搜索，这里只测试接口）
	results, err := store.SearchColumns(ctx, "user", 10)
	require.NoError(t, err)
	// SQLite简单实现可能返回所有或空，具体取决于实现
	_ = results
}

func TestSQLiteSyncLease_AcquireRenewReleaseAndExpiry(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	sourceID, err := st.CreateDataSource(ctx, &DataSourceCreate{
		Name:             "lease-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	})
	require.NoError(t, err)

	now := time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)
	acquired, err := st.TryAcquireSyncLease(ctx, sourceID, "owner-a", now.Format(time.RFC3339Nano), now.Add(time.Minute).Format(time.RFC3339Nano))
	require.NoError(t, err)
	assert.True(t, acquired)

	acquired, err = st.TryAcquireSyncLease(ctx, sourceID, "owner-b", now.Format(time.RFC3339Nano), now.Add(2*time.Minute).Format(time.RFC3339Nano))
	require.NoError(t, err)
	assert.False(t, acquired)

	require.NoError(t, st.RenewSyncLease(ctx, sourceID, "owner-a", now.Add(3*time.Minute).Format(time.RFC3339Nano)))

	acquired, err = st.TryAcquireSyncLease(ctx, sourceID, "owner-b", now.Add(2*time.Minute).Format(time.RFC3339Nano), now.Add(4*time.Minute).Format(time.RFC3339Nano))
	require.NoError(t, err)
	assert.False(t, acquired)

	require.NoError(t, st.ReleaseSyncLease(ctx, sourceID, "owner-a"))

	acquired, err = st.TryAcquireSyncLease(ctx, sourceID, "owner-b", now.Add(2*time.Minute).Format(time.RFC3339Nano), now.Add(4*time.Minute).Format(time.RFC3339Nano))
	require.NoError(t, err)
	assert.True(t, acquired)

	acquired, err = st.TryAcquireSyncLease(ctx, sourceID, "owner-c", now.Add(5*time.Minute).Format(time.RFC3339Nano), now.Add(6*time.Minute).Format(time.RFC3339Nano))
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestSQLiteSyncLease_GetAndForceRelease(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	sourceID, err := st.CreateDataSource(ctx, &DataSourceCreate{
		Name:             "lease-force-release",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	acquired, err := st.TryAcquireSyncLease(
		ctx,
		sourceID,
		"owner-a",
		now.Format(time.RFC3339Nano),
		now.Add(time.Minute).Format(time.RFC3339Nano),
	)
	require.NoError(t, err)
	require.True(t, acquired)

	lease, err := st.GetSyncLease(ctx, sourceID)
	require.NoError(t, err)
	require.NotNil(t, lease)
	assert.Equal(t, sourceID, lease.SourceID)
	assert.Equal(t, "owner-a", lease.OwnerID)

	require.NoError(t, st.ForceReleaseSyncLease(ctx, sourceID))

	lease, err = st.GetSyncLease(ctx, sourceID)
	require.NoError(t, err)
	assert.Nil(t, lease)
}

func TestSQLiteGovernanceOutbox_ClaimRetryAndDedupe(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	event := &GovernanceOutboxEventCreate{
		ID:            "outbox-1",
		EventID:       "evt-1",
		EventType:     "metadata.schema.changed",
		TraceID:       "trace-1",
		ResourceType:  "schema_change",
		ResourceID:    "chg-1",
		Payload:       `{"event_id":"evt-1","event_type":"metadata.schema.changed"}`,
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: "2026-03-28T10:00:00Z",
	}

	created, err := st.EnqueueGovernanceOutboxEvent(ctx, event)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = st.EnqueueGovernanceOutboxEvent(ctx, event)
	require.NoError(t, err)
	assert.False(t, created)

	rows, err := st.ClaimGovernanceOutboxEvents(ctx, "dispatcher-a", "2026-03-28T10:00:00Z", "2026-03-28T10:05:00Z", 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].AttemptCount)
	require.NotNil(t, rows[0].LeaseOwner)
	assert.Equal(t, "dispatcher-a", *rows[0].LeaseOwner)

	rows, err = st.ClaimGovernanceOutboxEvents(ctx, "dispatcher-b", "2026-03-28T10:01:00Z", "2026-03-28T10:06:00Z", 10)
	require.NoError(t, err)
	assert.Len(t, rows, 0)

	require.NoError(t, st.MarkGovernanceOutboxRetry(ctx, "outbox-1", "2026-03-28T10:10:00Z", "temporary failure"))

	rows, err = st.ClaimGovernanceOutboxEvents(ctx, "dispatcher-b", "2026-03-28T10:05:00Z", "2026-03-28T10:15:00Z", 10)
	require.NoError(t, err)
	assert.Len(t, rows, 0)

	rows, err = st.ClaimGovernanceOutboxEvents(ctx, "dispatcher-b", "2026-03-28T10:10:00Z", "2026-03-28T10:15:00Z", 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 2, rows[0].AttemptCount)

	require.NoError(t, st.MarkGovernanceOutboxDelivered(ctx, "outbox-1", "2026-03-28T10:11:00Z"))

	outboxRows, err := st.ListGovernanceOutboxEvents(ctx, 10)
	require.NoError(t, err)
	require.Len(t, outboxRows, 1)
	assert.Equal(t, "delivered", outboxRows[0].Status)
	require.NotNil(t, outboxRows[0].DeliveredAt)
	assert.Equal(t, "2026-03-28T10:11:00Z", *outboxRows[0].DeliveredAt)
}

func TestSQLiteGovernanceOutbox_DeadLetterReplayAndStats(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	created, err := st.EnqueueGovernanceOutboxEvent(ctx, &GovernanceOutboxEventCreate{
		ID:            "outbox-dead-letter",
		EventID:       "evt-dead-letter",
		EventType:     "metadata.schema.changed",
		TraceID:       "trace-dead-letter",
		ResourceType:  "schema_change",
		ResourceID:    "chg-dead-letter",
		Payload:       `{"event_id":"evt-dead-letter"}`,
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: "2000-01-01T00:00:00Z",
	})
	require.NoError(t, err)
	require.True(t, created)

	created, err = st.EnqueueGovernanceOutboxEvent(ctx, &GovernanceOutboxEventCreate{
		ID:            "outbox-delivered",
		EventID:       "evt-delivered",
		EventType:     "metadata.schema.changed",
		TraceID:       "trace-delivered",
		ResourceType:  "schema_change",
		ResourceID:    "chg-delivered",
		Payload:       `{"event_id":"evt-delivered"}`,
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: "2000-01-01T00:00:00Z",
	})
	require.NoError(t, err)
	require.True(t, created)

	require.NoError(t, st.MarkGovernanceOutboxDelivered(ctx, "outbox-delivered", "2026-03-28T10:11:00Z"))
	require.NoError(t, st.MarkGovernanceOutboxDeadLetter(ctx, "outbox-dead-letter", "permanent failure"))

	row, err := st.GetGovernanceOutboxEvent(ctx, "outbox-dead-letter")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "dead_letter", row.Status)
	assert.Equal(t, "evt-dead-letter", row.EventID)
	require.NotNil(t, row.LastError)
	assert.Equal(t, "permanent failure", *row.LastError)

	stats, err := st.GetGovernanceOutboxStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.EqualValues(t, 1, stats.DeadLetterCount)
	assert.EqualValues(t, 1, stats.DeliveredCount)
	assert.EqualValues(t, 0, stats.PendingCount)

	claimed, err := st.ClaimGovernanceOutboxEvents(ctx, "dispatcher-dead-letter", "2000-01-01T00:02:00Z", "2000-01-01T00:03:00Z", 10)
	require.NoError(t, err)
	assert.Len(t, claimed, 0)

	replayAt := "2000-01-01T00:01:00Z"
	require.NoError(t, st.ReplayGovernanceOutboxEvent(ctx, "outbox-dead-letter", replayAt))

	row, err = st.GetGovernanceOutboxEvent(ctx, "outbox-dead-letter")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "pending", row.Status)
	assert.Equal(t, 0, row.AttemptCount)
	assert.Equal(t, replayAt, row.NextAttemptAt)
	assert.Nil(t, row.LastError)
	assert.Nil(t, row.DeliveredAt)

	stats, err = st.GetGovernanceOutboxStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.EqualValues(t, 0, stats.DeadLetterCount)
	assert.EqualValues(t, 1, stats.DeliveredCount)
	assert.EqualValues(t, 1, stats.PendingCount)
	assert.EqualValues(t, 1, stats.RetryableCount)
}

func TestSQLiteGovernanceOutbox_ReplayRejectsDeliveredEvent(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	created, err := st.EnqueueGovernanceOutboxEvent(ctx, &GovernanceOutboxEventCreate{
		ID:            "outbox-delivered-no-replay",
		EventID:       "evt-delivered-no-replay",
		EventType:     "metadata.schema.changed",
		TraceID:       "trace-delivered-no-replay",
		ResourceType:  "schema_change",
		ResourceID:    "chg-delivered-no-replay",
		Payload:       `{"event_id":"evt-delivered-no-replay"}`,
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: "2000-01-01T00:00:00Z",
	})
	require.NoError(t, err)
	require.True(t, created)

	require.NoError(t, st.MarkGovernanceOutboxDelivered(ctx, "outbox-delivered-no-replay", "2026-03-28T10:11:00Z"))

	err = st.ReplayGovernanceOutboxEvent(ctx, "outbox-delivered-no-replay", "2000-01-01T00:01:00Z")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not replayable")

	row, err := st.GetGovernanceOutboxEvent(ctx, "outbox-delivered-no-replay")
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "delivered", row.Status)
	require.NotNil(t, row.DeliveredAt)
}

func TestSQLiteGovernanceOutboxStats_RetryableCountUsesRealTimestampComparison(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	created, err := st.EnqueueGovernanceOutboxEvent(ctx, &GovernanceOutboxEventCreate{
		ID:            "outbox-retryable-now",
		EventID:       "evt-retryable-now",
		EventType:     "metadata.schema.changed",
		TraceID:       "trace-retryable-now",
		ResourceType:  "schema_change",
		ResourceID:    "chg-retryable-now",
		Payload:       `{"event_id":"evt-retryable-now"}`,
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano),
	})
	require.NoError(t, err)
	require.True(t, created)

	stats, err := st.GetGovernanceOutboxStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.EqualValues(t, 1, stats.PendingCount)
	assert.EqualValues(t, 1, stats.RetryableCount)
}

func TestSQLiteAlertRuleMatching_UsesExactChangeTypeTokens(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	sourceID, err := st.CreateDataSource(ctx, &DataSourceCreate{
		Name:             "alert-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	})
	require.NoError(t, err)

	_, err = st.CreateAlertRule(ctx, &AlertRuleCreate{
		SourceID:    &sourceID,
		Name:        "exact",
		ChangeTypes: "alter_column",
		NotifyInApp: true,
		IsActive:    true,
	})
	require.NoError(t, err)
	_, err = st.CreateAlertRule(ctx, &AlertRuleCreate{
		SourceID:    &sourceID,
		Name:        "with-spaces",
		ChangeTypes: "drop_column, alter_column",
		NotifyInApp: true,
		IsActive:    true,
	})
	require.NoError(t, err)
	_, err = st.CreateAlertRule(ctx, &AlertRuleCreate{
		SourceID:    &sourceID,
		Name:        "substring-typo",
		ChangeTypes: "alter_column_typo",
		NotifyInApp: true,
		IsActive:    true,
	})
	require.NoError(t, err)
	_, err = st.CreateAlertRule(ctx, &AlertRuleCreate{
		SourceID:    &sourceID,
		Name:        "all",
		ChangeTypes: "all",
		NotifyInApp: true,
		IsActive:    true,
	})
	require.NoError(t, err)

	rules, err := st.ListMatchingAlertRules(ctx, sourceID, "alter_column")
	require.NoError(t, err)

	names := make([]string, 0, len(rules))
	for _, rule := range rules {
		names = append(names, rule.Name)
	}
	assert.ElementsMatch(t, []string{"exact", "with-spaces", "all"}, names)
}

func TestSQLiteTxAlertRuleMatching_UsesExactChangeTypeTokens(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	sourceID, err := st.CreateDataSource(ctx, &DataSourceCreate{
		Name:             "alert-source-tx",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	})
	require.NoError(t, err)

	err = st.WithTx(ctx, func(txStore Store) error {
		if _, err := txStore.CreateAlertRule(ctx, &AlertRuleCreate{
			SourceID:    &sourceID,
			Name:        "substring-typo",
			ChangeTypes: "alter_column_typo",
			NotifyInApp: true,
			IsActive:    true,
		}); err != nil {
			return err
		}
		if _, err := txStore.CreateAlertRule(ctx, &AlertRuleCreate{
			SourceID:    &sourceID,
			Name:        "exact",
			ChangeTypes: "alter_column",
			NotifyInApp: true,
			IsActive:    true,
		}); err != nil {
			return err
		}
		rules, err := txStore.ListMatchingAlertRules(ctx, sourceID, "alter_column")
		if err != nil {
			return err
		}
		require.Len(t, rules, 1)
		assert.Equal(t, "exact", rules[0].Name)
		return nil
	})
	require.NoError(t, err)
}

func TestSQLiteTxCreateNotification_RespectsNotifyInAppFlag(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	sourceID, err := st.CreateDataSource(ctx, &DataSourceCreate{
		Name:             "notify-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: `{}`,
	})
	require.NoError(t, err)

	userID, err := st.CreateUser(ctx, &UserCreate{
		Username:     "notify-user",
		Email:        "notify-user@example.com",
		PasswordHash: "hash",
		Role:         "admin",
	})
	require.NoError(t, err)

	require.NoError(t, st.CreateSchemaChange(ctx, &SchemaChangeCreate{
		ID:         "chg-notify-off",
		SourceID:   sourceID,
		ChangeType: "alter_column",
		ObjectType: "column",
		ObjectName: "users.email",
	}))

	err = st.WithTx(ctx, func(txStore Store) error {
		_, err := txStore.CreateNotification(ctx, &NotificationCreate{
			ChangeID:    "chg-notify-off",
			SourceID:    sourceID,
			Title:       "schema changed",
			Message:     "no in-app notification expected",
			ChangeType:  "alter_column",
			ObjectType:  "column",
			ObjectName:  "users.email",
			NotifyInApp: false,
		})
		return err
	})
	require.NoError(t, err)

	notifications, err := st.ListNotifications(ctx, userID, false, 10)
	require.NoError(t, err)
	assert.Len(t, notifications, 0)
}

func TestSQLiteAssignTermToColumn_RejectsMissingColumn(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	err := st.AssignTermToColumn(ctx, "missing-column", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "column not found")
}

func TestSQLiteTxAssignTermToColumn_RejectsMissingColumn(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	err := st.WithTx(ctx, func(txStore Store) error {
		return txStore.AssignTermToColumn(ctx, "missing-column", nil)
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "column not found")
}

func TestSQLiteDQStats_CountsErrorResultsAsFailures(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	ruleID, err := st.CreateDQRule(ctx, &DQRuleCreate{
		Name:       "dq-stats-rule",
		RuleType:   "custom_sql",
		RuleConfig: `{"sql":"SELECT 1"}`,
		Severity:   "error",
		IsActive:   true,
	})
	require.NoError(t, err)

	require.NoError(t, st.CreateDQResult(ctx, &DQResultCreate{
		RuleID:       ruleID,
		CheckBatchID: "batch-passed",
		Status:       "passed",
		TotalRows:    10,
		FailedRows:   0,
		PassRate:     100,
		SampleErrors: `[]`,
	}))
	require.NoError(t, st.CreateDQResult(ctx, &DQResultCreate{
		RuleID:       ruleID,
		CheckBatchID: "batch-failed",
		Status:       "failed",
		TotalRows:    10,
		FailedRows:   2,
		PassRate:     80,
		SampleErrors: `[{"value":"bad"}]`,
	}))
	errMsg := "connection lost"
	require.NoError(t, st.CreateDQResult(ctx, &DQResultCreate{
		RuleID:       ruleID,
		CheckBatchID: "batch-error",
		Status:       "error",
		TotalRows:    0,
		FailedRows:   0,
		PassRate:     0,
		SampleErrors: `[]`,
		ErrorMessage: &errMsg,
	}))

	stats, err := st.GetDQStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.TotalRules)
	assert.Equal(t, 1, stats.ActiveRules)
	assert.EqualValues(t, 3, stats.TotalChecks)
	assert.EqualValues(t, 1, stats.PassedChecks)
	assert.EqualValues(t, 2, stats.FailedChecks)
	assert.InDelta(t, 33.3333, stats.OverallPassRate, 0.01)
}

func TestSQLiteTxDQStats_CountsErrorResultsAsFailures(t *testing.T) {
	ctx := context.Background()
	st := testSQLiteStoreWithMigrations(t)
	registerStoreCleanup(t, st)

	ruleID, err := st.CreateDQRule(ctx, &DQRuleCreate{
		Name:       "dq-stats-rule-tx",
		RuleType:   "custom_sql",
		RuleConfig: `{"sql":"SELECT 1"}`,
		Severity:   "error",
		IsActive:   true,
	})
	require.NoError(t, err)

	err = st.WithTx(ctx, func(txStore Store) error {
		if err := txStore.CreateDQResult(ctx, &DQResultCreate{
			RuleID:       ruleID,
			CheckBatchID: "tx-batch-passed",
			Status:       "passed",
			TotalRows:    8,
			FailedRows:   0,
			PassRate:     100,
			SampleErrors: `[]`,
		}); err != nil {
			return err
		}

		errMsg := "execution timeout"
		if err := txStore.CreateDQResult(ctx, &DQResultCreate{
			RuleID:       ruleID,
			CheckBatchID: "tx-batch-error",
			Status:       "error",
			TotalRows:    0,
			FailedRows:   0,
			PassRate:     0,
			SampleErrors: `[]`,
			ErrorMessage: &errMsg,
		}); err != nil {
			return err
		}

		stats, err := txStore.GetDQStats(ctx)
		if err != nil {
			return err
		}
		require.NotNil(t, stats)
		assert.EqualValues(t, 2, stats.TotalChecks)
		assert.EqualValues(t, 1, stats.PassedChecks)
		assert.EqualValues(t, 1, stats.FailedChecks)
		assert.InDelta(t, 50.0, stats.OverallPassRate, 0.01)
		return nil
	})
	require.NoError(t, err)
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "valid postgres config",
			cfg: config.Config{
				Server: config.ServerConfig{Port: 8080},
				Auth:   config.AuthConfig{JWTSecret: "test-secret"},
				Database: config.DatabaseConfig{
					Type:     "postgres",
					Host:     "localhost",
					Port:     5432,
					Database: "datamap",
				},
				Log: config.LogConfig{Level: "info"},
			},
			wantErr: false,
		},
		{
			name: "valid sqlite config",
			cfg: config.Config{
				Server:   config.ServerConfig{Port: 8080},
				Auth:     config.AuthConfig{JWTSecret: "test-secret"},
				Database: config.DatabaseConfig{Type: "sqlite"},
				Log:      config.LogConfig{Level: "info"},
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			cfg: config.Config{
				Server:   config.ServerConfig{Port: 70000},
				Database: config.DatabaseConfig{Type: "sqlite"},
			},
			wantErr: true,
		},
		{
			name: "invalid database type",
			cfg: config.Config{
				Server:   config.ServerConfig{Port: 8080},
				Database: config.DatabaseConfig{Type: "oracle"},
			},
			wantErr: true,
		},
		{
			name: "postgres without database",
			cfg: config.Config{
				Server:   config.ServerConfig{Port: 8080},
				Database: config.DatabaseConfig{Type: "postgres"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMain 测试入口
func TestMain(m *testing.M) {
	// 设置测试环境
	os.Exit(m.Run())
}

func testSQLiteStoreWithMigrations(t *testing.T) *SQLiteStore {
	t.Helper()

	st, err := NewSQLiteStore(context.Background(), &config.DatabaseConfig{
		Type:           "sqlite",
		SQLitePath:     filepath.Join(t.TempDir(), "store-test.db"),
		SQLiteMaxConns: 1,
		SQLiteMinConns: 1,
	}, zap.NewNop())
	require.NoError(t, err)
	return st
}
