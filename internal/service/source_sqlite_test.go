package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiangfire/datamaplite/internal/config"
	"github.com/jiangfire/datamaplite/internal/crypto"
	"github.com/jiangfire/datamaplite/internal/scanner"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type sqliteSourceTestEnv struct {
	service  *SourceService
	store    store.Store
	sourceID string
}

type renewFailingStore struct {
	store.Store
	renewErr error
}

type stubSchemaScanner struct {
	testConnectionFn func(ctx context.Context, config scanner.ConnectionConfig) error
	scanSchemaFn     func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error)
	lastScanCtx      context.Context
}

func newSQLiteSourceTestEnv(t *testing.T) *sqliteSourceTestEnv {
	t.Helper()
	return newSQLiteSourceTestEnvAtPath(t, filepath.Join(t.TempDir(), "datamap-test.db"))
}

func newSQLiteSourceTestEnvAtPath(t *testing.T, dbPath string) *sqliteSourceTestEnv {
	t.Helper()

	ctx := context.Background()
	st, err := store.NewSQLiteStore(ctx, &config.DatabaseConfig{
		Type:           "sqlite",
		SQLitePath:     dbPath,
		SQLiteMaxConns: 1,
		SQLiteMinConns: 1,
	}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, st.Close())
	})

	cipher, err := crypto.NewCipher("12345678901234567890123456789012")
	require.NoError(t, err)

	configJSON, err := scanner.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "datamap",
		Username: "root",
		Password: "secret",
	}.ToJSON()
	require.NoError(t, err)

	encryptedConfig, err := cipher.Encrypt(configJSON)
	require.NoError(t, err)

	sourceID, err := st.CreateDataSource(ctx, &store.DataSourceCreate{
		Name:             "sqlite-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "datamap",
		ConnectionConfig: encryptedConfig,
	})
	require.NoError(t, err)

	return &sqliteSourceTestEnv{
		service:  NewSourceService(st, cipher, scanner.NewRegistry()),
		store:    st,
		sourceID: sourceID,
	}
}

func (s *stubSchemaScanner) TestConnection(ctx context.Context, config scanner.ConnectionConfig) error {
	if s.testConnectionFn != nil {
		return s.testConnectionFn(ctx, config)
	}
	return nil
}

func (s *stubSchemaScanner) ScanSchema(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
	s.lastScanCtx = ctx
	if s.scanSchemaFn != nil {
		return s.scanSchemaFn(ctx, config)
	}
	return &scanner.SchemaInfo{}, nil
}

func (s *renewFailingStore) RenewSyncLease(ctx context.Context, sourceID string, ownerID string, leaseUntil string) error {
	if s.renewErr != nil {
		return s.renewErr
	}
	return s.Store.RenewSyncLease(ctx, sourceID, ownerID, leaseUntil)
}

func TestSourceService_SaveSchema_SQLiteLifecycle(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	publicSchema := "public"
	userDesc := "user table"
	legacyDesc := "legacy table"
	emailDesc := "user email"
	rowCount := int64(100)
	sizeBytes := int64(2048)

	err := env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name:        "users",
				Type:        "table",
				Schema:      &publicSchema,
				Description: &userDesc,
				RowCount:    &rowCount,
				SizeBytes:   &sizeBytes,
				Columns: []scanner.ColumnInfo{
					{
						Name:            "id",
						DataType:        "bigint",
						FullDataType:    "bigint",
						IsNullable:      false,
						IsPrimaryKey:    true,
						OrdinalPosition: 1,
						Confidence:      0,
					},
					{
						Name:            "email",
						DataType:        "varchar",
						FullDataType:    "varchar(255)",
						IsNullable:      false,
						IsUnique:        true,
						OrdinalPosition: 2,
						Description:     &emailDesc,
						Confidence:      0.75,
					},
				},
			},
			{
				Name:        "legacy_users",
				Type:        "table",
				Description: &legacyDesc,
				Columns: []scanner.ColumnInfo{
					{
						Name:            "legacy_id",
						DataType:        "int",
						FullDataType:    "int",
						IsNullable:      false,
						OrdinalPosition: 1,
						Confidence:      1,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	usersObj := mustGetSchemaObjectByName(t, env.store, env.sourceID, "users", &publicSchema)
	assert.Equal(t, 2, usersObj.ColumnCount)
	assert.Equal(t, rowCount, *usersObj.RowCount)
	assert.Equal(t, sizeBytes, *usersObj.SizeBytes)

	userColumns := mustListColumnsByObject(t, env.store, usersObj.ID)
	userColumnsByName := columnsByName(userColumns)
	require.Len(t, userColumnsByName, 2)
	assert.Equal(t, 1.0, userColumnsByName["id"].Confidence)

	legacyObj := mustGetSchemaObjectByName(t, env.store, env.sourceID, "legacy_users", nil)
	legacyColumns := mustListColumnsByObject(t, env.store, legacyObj.ID)
	require.Len(t, legacyColumns, 1)

	require.NoError(t, env.store.CreateLineageEdge(ctx, &store.LineageEdgeCreate{
		SourceID:   userColumnsByName["email"].ID,
		TargetID:   userColumnsByName["id"].ID,
		SourceType: "column",
		TargetType: "column",
	}))
	require.NoError(t, env.store.CreateLineageEdge(ctx, &store.LineageEdgeCreate{
		SourceID:   legacyObj.ID,
		TargetID:   usersObj.ID,
		SourceType: "object",
		TargetType: "object",
	}))

	originalObjectID := usersObj.ID
	originalIDColumnID := userColumnsByName["id"].ID

	updatedUserDesc := "user table v2"
	updatedRowCount := int64(250)
	updatedSizeBytes := int64(4096)
	defaultNow := "CURRENT_TIMESTAMP"

	err = env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name:        "users",
				Type:        "table",
				Schema:      &publicSchema,
				Description: &updatedUserDesc,
				RowCount:    &updatedRowCount,
				SizeBytes:   &updatedSizeBytes,
				Columns: []scanner.ColumnInfo{
					{
						Name:            "id",
						DataType:        "uuid",
						FullDataType:    "uuid",
						IsNullable:      false,
						IsPrimaryKey:    true,
						OrdinalPosition: 1,
						Confidence:      0.9,
					},
					{
						Name:            "created_at",
						DataType:        "timestamp",
						FullDataType:    "timestamp",
						IsNullable:      false,
						DefaultValue:    &defaultNow,
						OrdinalPosition: 3,
						Confidence:      0,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	updatedUsersObj := mustGetSchemaObjectByName(t, env.store, env.sourceID, "users", &publicSchema)
	assert.Equal(t, originalObjectID, updatedUsersObj.ID)
	assert.Equal(t, 2, updatedUsersObj.ColumnCount)
	assert.Equal(t, updatedUserDesc, *updatedUsersObj.Description)
	assert.Equal(t, updatedRowCount, *updatedUsersObj.RowCount)
	assert.Equal(t, updatedSizeBytes, *updatedUsersObj.SizeBytes)

	updatedColumns := columnsByName(mustListColumnsByObject(t, env.store, updatedUsersObj.ID))
	require.Len(t, updatedColumns, 2)
	assert.NotContains(t, updatedColumns, "email")
	assert.Equal(t, originalIDColumnID, updatedColumns["id"].ID)
	assert.Equal(t, "uuid", updatedColumns["id"].DataType)
	assert.Equal(t, "uuid", updatedColumns["id"].FullDataType)
	assert.Equal(t, 1.0, updatedColumns["created_at"].Confidence)
	require.NotNil(t, updatedColumns["created_at"].DefaultValue)
	assert.Equal(t, defaultNow, *updatedColumns["created_at"].DefaultValue)

	droppedLegacyObj, err := env.store.GetSchemaObjectByName(ctx, env.sourceID, "legacy_users", nil)
	require.NoError(t, err)
	assert.Nil(t, droppedLegacyObj)

	changes, err := env.store.ListSchemaChangesBySource(ctx, env.sourceID, 20)
	require.NoError(t, err)
	assert.Len(t, changes, 9)
	assertSchemaChangeExists(t, changes, "add_object", "users", nil, func(change *store.SchemaChangeRow) bool {
		return change.NewValue != nil && *change.NewValue == "table"
	})
	assertSchemaChangeExists(t, changes, "add_column", "users.id", nil, func(change *store.SchemaChangeRow) bool {
		return change.NewValue != nil && assertContainsAll(*change.NewValue, "type=bigint", "confidence=1.000000")
	})
	assertSchemaChangeExists(t, changes, "add_column", "users.email", nil, func(change *store.SchemaChangeRow) bool {
		return change.NewValue != nil && assertContainsAll(*change.NewValue, "type=varchar", "confidence=0.750000")
	})
	assertSchemaChangeExists(t, changes, "add_object", "legacy_users", nil, nil)
	assertSchemaChangeExists(t, changes, "drop_object", "legacy_users", nil, func(change *store.SchemaChangeRow) bool {
		return change.OldValue != nil && *change.OldValue == "table"
	})
	assertSchemaChangeExists(t, changes, "alter_column", "users.id", nil, func(change *store.SchemaChangeRow) bool {
		return change.OldValue != nil &&
			change.NewValue != nil &&
			assertContainsAll(*change.OldValue, "type=bigint") &&
			assertContainsAll(*change.NewValue, "type=uuid")
	})
	assertSchemaChangeExists(t, changes, "drop_column", "users.email", nil, func(change *store.SchemaChangeRow) bool {
		return change.OldValue != nil && assertContainsAll(*change.OldValue, "varchar(255)")
	})
	assertSchemaChangeExists(t, changes, "add_column", "users.created_at", nil, func(change *store.SchemaChangeRow) bool {
		return change.NewValue != nil && assertContainsAll(*change.NewValue, "CURRENT_TIMESTAMP", "confidence=1.000000")
	})
}

func TestSourceService_SaveSchema_DoesNotCreateFalsePositiveChangeWhenConfidenceDefaultsMatch(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	err := env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name: "events",
				Type: "table",
				Columns: []scanner.ColumnInfo{
					{
						Name:            "payload",
						DataType:        "json",
						FullDataType:    "json",
						IsNullable:      true,
						OrdinalPosition: 1,
						Confidence:      0,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	initialChanges, err := env.store.ListSchemaChangesBySource(ctx, env.sourceID, 10)
	require.NoError(t, err)
	require.Len(t, initialChanges, 2)

	err = env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name: "events",
				Type: "table",
				Columns: []scanner.ColumnInfo{
					{
						Name:            "payload",
						DataType:        "json",
						FullDataType:    "json",
						IsNullable:      true,
						OrdinalPosition: 1,
						Confidence:      1,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	finalChanges, err := env.store.ListSchemaChangesBySource(ctx, env.sourceID, 10)
	require.NoError(t, err)
	assert.Len(t, finalChanges, 2)
}

func TestSourceService_SaveSchema_DoesNotCreateFalsePositiveChangeForEquivalentPointerValues(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	initialDescription := "event payload"
	initialDefault := "{}"
	initialParent := "payload.root"

	err := env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name: "events",
				Type: "table",
				Columns: []scanner.ColumnInfo{
					{
						Name:             "payload",
						DataType:         "json",
						FullDataType:     "json",
						IsNullable:       true,
						DefaultValue:     &initialDefault,
						OrdinalPosition:  1,
						Description:      &initialDescription,
						ParentColumnPath: &initialParent,
						Confidence:       1,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	initialChanges, err := env.store.ListSchemaChangesBySource(ctx, env.sourceID, 10)
	require.NoError(t, err)
	require.Len(t, initialChanges, 2)

	sameDescription := "event payload"
	sameDefault := "{}"
	sameParent := "payload.root"

	err = env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name: "events",
				Type: "table",
				Columns: []scanner.ColumnInfo{
					{
						Name:             "payload",
						DataType:         "json",
						FullDataType:     "json",
						IsNullable:       true,
						DefaultValue:     &sameDefault,
						OrdinalPosition:  1,
						Description:      &sameDescription,
						ParentColumnPath: &sameParent,
						Confidence:       1,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	finalChanges, err := env.store.ListSchemaChangesBySource(ctx, env.sourceID, 10)
	require.NoError(t, err)
	assert.Len(t, finalChanges, 2)
}

func TestSourceService_RecordSchemaChange_DoesNotMixRowsWhenDetectedAtMatches(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	fixedTime := "2026-03-27T12:00:00.000000000Z"
	detectedChanges := make([]*SchemaChangeInfo, 0, 2)

	err := env.store.WithTx(ctx, func(txStore store.Store) error {
		first := &store.SchemaChangeCreate{
			SourceID:   env.sourceID,
			ChangeType: "add_object",
			ObjectType: "object",
			ObjectName: "users",
			DetectedAt: fixedTime,
		}
		if err := env.service.recordSchemaChange(ctx, txStore, &detectedChanges, first); err != nil {
			return err
		}

		second := &store.SchemaChangeCreate{
			SourceID:   env.sourceID,
			ChangeType: "add_column",
			ObjectType: "column",
			ObjectName: "users.email",
			DetectedAt: fixedTime,
		}
		return env.service.recordSchemaChange(ctx, txStore, &detectedChanges, second)
	})
	require.NoError(t, err)
	require.Len(t, detectedChanges, 2)

	assert.Equal(t, "add_object", detectedChanges[0].ChangeType)
	assert.Equal(t, "users", detectedChanges[0].ObjectName)
	assert.Equal(t, fixedTime, detectedChanges[0].DetectedAt)
	assert.Equal(t, "add_column", detectedChanges[1].ChangeType)
	assert.Equal(t, "users.email", detectedChanges[1].ObjectName)
	assert.Equal(t, fixedTime, detectedChanges[1].DetectedAt)
	assert.NotEqual(t, detectedChanges[0].ID, detectedChanges[1].ID)
}

func TestSourceService_SaveSchema_PublishesDistinctGovernanceEvents(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	err := env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name: "users",
				Type: "table",
				Columns: []scanner.ColumnInfo{
					{
						Name:            "id",
						DataType:        "bigint",
						FullDataType:    "bigint",
						IsNullable:      false,
						IsPrimaryKey:    true,
						OrdinalPosition: 1,
						Confidence:      1,
					},
					{
						Name:            "email",
						DataType:        "varchar",
						FullDataType:    "varchar(255)",
						IsNullable:      false,
						OrdinalPosition: 2,
						Confidence:      1,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	eventCh := make(chan GovernanceEvent, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event GovernanceEvent
		require.NoError(t, json.NewDecoder(r.Body).Decode(&event))
		eventCh <- event
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	env.service.SetGovernanceEventService(NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "fuckcmdb",
		Timeout:          time.Second,
	}, zap.NewNop()))

	auditCtx := WithGovernanceAuditMeta(ctx, GovernanceAuditMeta{
		ActorID:   "mcp:datamap",
		TraceID:   "trace-save-schema",
		Origin:    "mcp",
		Operation: "trigger_source_sync",
	})

	err = env.service.saveSchema(auditCtx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name: "users",
				Type: "table",
				Columns: []scanner.ColumnInfo{
					{
						Name:            "id",
						DataType:        "uuid",
						FullDataType:    "uuid",
						IsNullable:      false,
						IsPrimaryKey:    true,
						OrdinalPosition: 1,
						Confidence:      1,
					},
					{
						Name:            "created_at",
						DataType:        "timestamp",
						FullDataType:    "timestamp",
						IsNullable:      false,
						OrdinalPosition: 3,
						Confidence:      1,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	events := collectGovernanceEvents(t, eventCh, 3, 2*time.Second)
	require.Len(t, events, 3)

	seen := make(map[string]GovernanceEvent, len(events))
	for _, event := range events {
		require.Equal(t, "metadata.schema.changed", event.EventType)
		require.Equal(t, "mcp:datamap", event.ActorID)
		require.Equal(t, "trace-save-schema", event.TraceID)
		require.Equal(t, "mcp", event.Payload["audit_origin"])
		require.Equal(t, "trigger_source_sync", event.Payload["audit_operation"])
		require.Contains(t, []string{"users.id", "users.email", "users.created_at"}, event.Payload["object_name"])
		seen[event.Payload["object_name"].(string)+"|"+event.Payload["change_type"].(string)] = event
	}

	assert.Contains(t, seen, "users.id|alter_column")
	assert.Contains(t, seen, "users.email|drop_column")
	assert.Contains(t, seen, "users.created_at|add_column")
}

func TestSourceService_SaveSchema_DropObjectPreservesObjectIDForScopedAudit(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	err := env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name: "legacy_users",
				Type: "table",
				Columns: []scanner.ColumnInfo{
					{
						Name:            "id",
						DataType:        "bigint",
						FullDataType:    "bigint",
						IsNullable:      false,
						IsPrimaryKey:    true,
						OrdinalPosition: 1,
						Confidence:      1,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	legacyObj := mustGetSchemaObjectByName(t, env.store, env.sourceID, "legacy_users", nil)
	legacyObjectID := legacyObj.ID

	err = env.service.saveSchema(ctx, env.sourceID, &scanner.SchemaInfo{Objects: []scanner.ObjectInfo{}})
	require.NoError(t, err)

	changes, err := env.store.ListSchemaChangesBySource(ctx, env.sourceID, 10)
	require.NoError(t, err)
	assertSchemaChangeExists(t, changes, "drop_object", "legacy_users", &legacyObjectID, func(change *store.SchemaChangeRow) bool {
		return change.OldValue != nil && *change.OldValue == "table"
	})
}

func TestSourceService_TriggerSync_SuccessPersistsSchemaAndStatus(t *testing.T) {
	ctx := WithGovernanceAuditMeta(context.Background(), GovernanceAuditMeta{
		ActorID:   "user-1",
		TraceID:   "trace-trigger-sync-success",
		Origin:    "http",
		Operation: "POST /api/v1/sources/:id/sync",
	})
	env := newSQLiteSourceTestEnv(t)

	schemaName := "public"
	sc := &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name:   "orders",
						Type:   "table",
						Schema: &schemaName,
						Columns: []scanner.ColumnInfo{
							{
								Name:            "id",
								DataType:        "bigint",
								FullDataType:    "bigint",
								IsNullable:      false,
								IsPrimaryKey:    true,
								OrdinalPosition: 1,
							},
						},
					},
				},
			}, nil
		},
	}
	env.service.registry.Register("mysql", sc)

	err := env.service.TriggerSync(ctx, env.sourceID)
	require.NoError(t, err)

	row := waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "active"
	})
	assert.Equal(t, "active", row.Status)
	assert.Nil(t, row.LastSyncError)
	assert.NotNil(t, row.LastSyncAt)

	meta := GovernanceAuditMetaFromContext(sc.lastScanCtx)
	assert.Equal(t, "user-1", meta.ActorID)
	assert.Equal(t, "trace-trigger-sync-success", meta.TraceID)
	assert.Equal(t, "http", meta.Origin)

	obj := mustGetSchemaObjectByName(t, env.store, env.sourceID, "orders", &schemaName)
	cols := mustListColumnsByObject(t, env.store, obj.ID)
	require.Len(t, cols, 1)
	assert.Equal(t, "id", cols[0].Name)
}

func TestSourceService_TriggerSync_ScannerFailureSetsErrorStatus(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	env.service.registry.Register("mysql", &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			return nil, assert.AnError
		},
	})

	err := env.service.TriggerSync(ctx, env.sourceID)
	require.NoError(t, err)

	row := waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "error"
	})
	assert.Equal(t, "error", row.Status)
	require.NotNil(t, row.LastSyncError)
	assert.Contains(t, *row.LastSyncError, assert.AnError.Error())

	objects, err := env.store.ListSchemaObjectsBySource(ctx, env.sourceID)
	require.NoError(t, err)
	assert.Len(t, objects, 0)
}

func TestSourceService_TriggerSync_SaveSchemaFailureRollsBackAndSetsErrorStatus(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	env.service.registry.Register("mysql", &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name: "broken_table",
						Type: "table",
						Columns: []scanner.ColumnInfo{
							{
								Name:            "dup_col",
								DataType:        "varchar",
								FullDataType:    "varchar(10)",
								OrdinalPosition: 1,
							},
							{
								Name:            "dup_col",
								DataType:        "varchar",
								FullDataType:    "varchar(10)",
								OrdinalPosition: 2,
							},
						},
					},
				},
			}, nil
		},
	})

	err := env.service.TriggerSync(ctx, env.sourceID)
	require.NoError(t, err)

	row := waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "error"
	})
	assert.Equal(t, "error", row.Status)
	require.NotNil(t, row.LastSyncError)
	assert.Contains(t, *row.LastSyncError, "failed to create column")

	objects, err := env.store.ListSchemaObjectsBySource(ctx, env.sourceID)
	require.NoError(t, err)
	assert.Len(t, objects, 0)

	changes, err := env.store.ListSchemaChangesBySource(ctx, env.sourceID, 10)
	require.NoError(t, err)
	assert.Len(t, changes, 0)
}

func TestSourceService_TriggerSync_RejectsConcurrentSyncAndAllowsRetryAfterCompletion(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	schemaName := "public"
	releaseFirst := make(chan struct{})
	started := make(chan int, 2)
	finished := make(chan int, 2)
	var mu sync.Mutex
	callCount := 0

	env.service.registry.Register("mysql", &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			mu.Lock()
			callCount++
			call := callCount
			mu.Unlock()

			started <- call
			if call == 1 {
				<-releaseFirst
			}

			finished <- call
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name:   "orders",
						Type:   "table",
						Schema: &schemaName,
						Columns: []scanner.ColumnInfo{
							{
								Name:            "id",
								DataType:        "bigint",
								FullDataType:    "bigint",
								IsNullable:      false,
								IsPrimaryKey:    true,
								OrdinalPosition: 1,
							},
						},
					},
				},
			}, nil
		},
	})

	require.NoError(t, env.service.TriggerSync(ctx, env.sourceID))
	assert.Equal(t, 1, waitForSyncCall(t, started, "first scan started"))

	row := waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "syncing"
	})
	assert.Equal(t, "syncing", row.Status)

	err := env.service.TriggerSync(ctx, env.sourceID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source sync already in progress")

	close(releaseFirst)
	assert.Equal(t, 1, waitForSyncCall(t, finished, "first scan finished"))

	row = waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "active"
	})
	assert.Equal(t, "active", row.Status)

	require.NoError(t, env.service.TriggerSync(ctx, env.sourceID))
	assert.Equal(t, 2, waitForSyncCall(t, started, "second scan started"))
	assert.Equal(t, 2, waitForSyncCall(t, finished, "second scan finished"))

	row = waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "active"
	})
	assert.Equal(t, "active", row.Status)
}

func TestSourceService_TriggerSync_RejectsConcurrentSyncAcrossInstancesAndAllowsRetryAfterCompletion(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "shared-sync.db")
	env := newSQLiteSourceTestEnvAtPath(t, dbPath)

	secondStore, err := store.NewSQLiteStore(ctx, &config.DatabaseConfig{
		Type:           "sqlite",
		SQLitePath:     dbPath,
		SQLiteMaxConns: 1,
		SQLiteMinConns: 1,
	}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, secondStore.Close())
	})

	secondCipher, err := crypto.NewCipher("12345678901234567890123456789012")
	require.NoError(t, err)
	secondService := NewSourceService(secondStore, secondCipher, scanner.NewRegistry())

	schemaName := "public"
	releaseFirst := make(chan struct{})
	firstStarted := make(chan struct{}, 1)
	firstFinished := make(chan struct{}, 1)
	secondStarted := make(chan struct{}, 1)
	secondFinished := make(chan struct{}, 1)

	env.service.registry.Register("mysql", &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			firstStarted <- struct{}{}
			<-releaseFirst
			firstFinished <- struct{}{}
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name:   "orders",
						Type:   "table",
						Schema: &schemaName,
						Columns: []scanner.ColumnInfo{
							{
								Name:            "id",
								DataType:        "bigint",
								FullDataType:    "bigint",
								IsNullable:      false,
								IsPrimaryKey:    true,
								OrdinalPosition: 1,
							},
						},
					},
				},
			}, nil
		},
	})
	secondService.registry.Register("mysql", &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			secondStarted <- struct{}{}
			secondFinished <- struct{}{}
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name:   "orders",
						Type:   "table",
						Schema: &schemaName,
						Columns: []scanner.ColumnInfo{
							{
								Name:            "id",
								DataType:        "bigint",
								FullDataType:    "bigint",
								IsNullable:      false,
								IsPrimaryKey:    true,
								OrdinalPosition: 1,
							},
						},
					},
				},
			}, nil
		},
	})

	require.NoError(t, env.service.TriggerSync(ctx, env.sourceID))
	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first instance scan to start")
	}

	row := waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "syncing"
	})
	assert.Equal(t, "syncing", row.Status)

	err = secondService.TriggerSync(ctx, env.sourceID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source sync already in progress")

	select {
	case <-secondStarted:
		t.Fatal("second instance unexpectedly started scanning")
	default:
	}

	close(releaseFirst)
	select {
	case <-firstFinished:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first instance scan to finish")
	}

	row = waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "active"
	})
	assert.Equal(t, "active", row.Status)

	require.NoError(t, secondService.TriggerSync(ctx, env.sourceID))
	select {
	case <-secondStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second instance scan to start")
	}
	select {
	case <-secondFinished:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second instance scan to finish")
	}

	row = waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "active"
	})
	assert.Equal(t, "active", row.Status)
}

func TestSourceService_TriggerSync_DoesNotPersistAfterLeaseLoss(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "shared-lease-loss.db")
	baseEnv := newSQLiteSourceTestEnvAtPath(t, dbPath)

	secondStore, err := store.NewSQLiteStore(ctx, &config.DatabaseConfig{
		Type:           "sqlite",
		SQLitePath:     dbPath,
		SQLiteMaxConns: 1,
		SQLiteMinConns: 1,
	}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, secondStore.Close())
	})

	firstService := NewSourceService(&renewFailingStore{
		Store:    baseEnv.store,
		renewErr: assert.AnError,
	}, baseEnv.service.cipher, scanner.NewRegistry())
	firstService.syncLeaseTTL = 40 * time.Millisecond
	firstService.syncLeaseStaleAfter = 40 * time.Millisecond

	secondCipher, err := crypto.NewCipher("12345678901234567890123456789012")
	require.NoError(t, err)
	secondService := NewSourceService(secondStore, secondCipher, scanner.NewRegistry())
	secondService.syncLeaseTTL = 40 * time.Millisecond
	secondService.syncLeaseStaleAfter = 40 * time.Millisecond

	firstStarted := make(chan struct{}, 1)
	releaseFirst := make(chan struct{})
	firstService.registry.Register("mysql", &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			firstStarted <- struct{}{}
			<-releaseFirst
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name: "stale_writer_table",
						Type: "table",
						Columns: []scanner.ColumnInfo{
							{
								Name:            "id",
								DataType:        "bigint",
								FullDataType:    "bigint",
								IsNullable:      false,
								IsPrimaryKey:    true,
								OrdinalPosition: 1,
							},
						},
					},
				},
			}, nil
		},
	})
	secondService.registry.Register("mysql", &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name: "fresh_writer_table",
						Type: "table",
						Columns: []scanner.ColumnInfo{
							{
								Name:            "id",
								DataType:        "bigint",
								FullDataType:    "bigint",
								IsNullable:      false,
								IsPrimaryKey:    true,
								OrdinalPosition: 1,
							},
						},
					},
				},
			}, nil
		},
	})

	require.NoError(t, firstService.TriggerSync(ctx, baseEnv.sourceID))
	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first stale sync to start")
	}

	time.Sleep(120 * time.Millisecond)

	require.NoError(t, secondService.TriggerSync(ctx, baseEnv.sourceID))
	row := waitForSourceState(t, secondStore, baseEnv.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "active"
	})
	assert.Equal(t, "active", row.Status)

	freshObj := waitForSchemaObjectByName(t, secondStore, baseEnv.sourceID, "fresh_writer_table", nil, 2*time.Second)
	require.NotNil(t, freshObj)

	close(releaseFirst)

	time.Sleep(250 * time.Millisecond)

	staleObj, err := secondStore.GetSchemaObjectByName(ctx, baseEnv.sourceID, "stale_writer_table", nil)
	require.NoError(t, err)
	assert.Nil(t, staleObj)

	freshObj, err = secondStore.GetSchemaObjectByName(ctx, baseEnv.sourceID, "fresh_writer_table", nil)
	require.NoError(t, err)
	require.NotNil(t, freshObj)

	row, err = secondStore.GetDataSource(ctx, baseEnv.sourceID)
	require.NoError(t, err)
	assert.Equal(t, "active", row.Status)
}

func TestSourceService_TriggerSync_LeaseLossWithoutTakeoverMarksError(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	serviceWithFailingRenew := NewSourceService(&renewFailingStore{
		Store:    env.store,
		renewErr: assert.AnError,
	}, env.service.cipher, scanner.NewRegistry())
	serviceWithFailingRenew.syncLeaseTTL = 40 * time.Millisecond
	serviceWithFailingRenew.syncLeaseStaleAfter = 40 * time.Millisecond

	started := make(chan struct{}, 1)
	releaseScan := make(chan struct{})
	serviceWithFailingRenew.registry.Register("mysql", &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			started <- struct{}{}
			<-releaseScan
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name: "should_not_persist",
						Type: "table",
						Columns: []scanner.ColumnInfo{
							{
								Name:            "id",
								DataType:        "bigint",
								FullDataType:    "bigint",
								IsNullable:      false,
								IsPrimaryKey:    true,
								OrdinalPosition: 1,
							},
						},
					},
				},
			}, nil
		},
	})

	require.NoError(t, serviceWithFailingRenew.TriggerSync(ctx, env.sourceID))
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for lease-loss sync to start")
	}

	time.Sleep(120 * time.Millisecond)
	close(releaseScan)

	row := waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "error"
	})
	assert.Equal(t, "error", row.Status)
	require.NotNil(t, row.LastSyncError)
	assert.Contains(t, *row.LastSyncError, "sync lease lost")

	obj, err := env.store.GetSchemaObjectByName(ctx, env.sourceID, "should_not_persist", nil)
	require.NoError(t, err)
	assert.Nil(t, obj)
}

func TestSourceService_TriggerSync_ReleasesInFlightAfterEarlyDecryptFailure(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	badConfig := "not-encrypted"
	require.NoError(t, env.store.UpdateDataSource(ctx, env.sourceID, &store.DataSourceUpdate{
		ConnectionConfig: &badConfig,
	}))

	err := env.service.TriggerSync(ctx, env.sourceID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt connection config")

	sc := &stubSchemaScanner{
		scanSchemaFn: func(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
			return &scanner.SchemaInfo{
				Objects: []scanner.ObjectInfo{
					{
						Name: "orders",
						Type: "table",
						Columns: []scanner.ColumnInfo{
							{
								Name:            "id",
								DataType:        "bigint",
								FullDataType:    "bigint",
								IsNullable:      false,
								IsPrimaryKey:    true,
								OrdinalPosition: 1,
							},
						},
					},
				},
			}, nil
		},
	}
	env.service.registry.Register("mysql", sc)

	configJSON, err := scanner.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "datamap",
		Username: "root",
		Password: "secret",
	}.ToJSON()
	require.NoError(t, err)
	encryptedConfig, err := env.service.cipher.Encrypt(configJSON)
	require.NoError(t, err)
	require.NoError(t, env.store.UpdateDataSource(ctx, env.sourceID, &store.DataSourceUpdate{
		ConnectionConfig: &encryptedConfig,
	}))

	require.NoError(t, env.service.TriggerSync(ctx, env.sourceID))

	row := waitForSourceState(t, env.store, env.sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "active"
	})
	assert.Equal(t, "active", row.Status)
}

func TestSourceService_ForceReleaseStaleSyncLease_RejectsFreshLeaseAndAllowsStaleRelease(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteSourceTestEnv(t)

	now := time.Now().UTC()
	acquired, err := env.store.TryAcquireSyncLease(
		ctx,
		env.sourceID,
		"owner-a",
		now.Format(time.RFC3339Nano),
		now.Add(time.Minute).Format(time.RFC3339Nano),
	)
	require.NoError(t, err)
	require.True(t, acquired)

	lease, err := env.service.GetSyncLease(ctx, env.sourceID)
	require.NoError(t, err)
	require.NotNil(t, lease)
	assert.Equal(t, "owner-a", lease.OwnerID)

	err = env.service.ForceReleaseStaleSyncLease(ctx, env.sourceID, time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sync lease still fresh")

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, env.service.ForceReleaseStaleSyncLease(ctx, env.sourceID, time.Nanosecond))

	lease, err = env.service.GetSyncLease(ctx, env.sourceID)
	require.NoError(t, err)
	assert.Nil(t, lease)
}

func mustGetSchemaObjectByName(t *testing.T, st store.Store, sourceID string, name string, schema *string) *store.SchemaObjectRow {
	t.Helper()

	obj, err := st.GetSchemaObjectByName(context.Background(), sourceID, name, schema)
	require.NoError(t, err)
	return obj
}

func mustListColumnsByObject(t *testing.T, st store.Store, objectID string) []*store.ColumnRow {
	t.Helper()

	cols, err := st.ListColumnsByObject(context.Background(), objectID)
	require.NoError(t, err)
	return cols
}

func columnsByName(cols []*store.ColumnRow) map[string]*store.ColumnRow {
	result := make(map[string]*store.ColumnRow, len(cols))
	for _, col := range cols {
		result[col.Name] = col
	}
	return result
}

func assertSchemaChangeExists(
	t *testing.T,
	changes []*store.SchemaChangeRow,
	changeType string,
	objectName string,
	objectID *string,
	matcher func(*store.SchemaChangeRow) bool,
) {
	t.Helper()

	for _, change := range changes {
		if change.ChangeType != changeType || change.ObjectName != objectName {
			continue
		}
		if objectID != nil {
			if change.ObjectID == nil || *change.ObjectID != *objectID {
				continue
			}
		}
		if matcher == nil || matcher(change) {
			return
		}
	}

	t.Fatalf("schema change not found: type=%s object=%s", changeType, objectName)
}

func assertContainsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}

func collectGovernanceEvents(t *testing.T, eventCh <-chan GovernanceEvent, count int, timeout time.Duration) []GovernanceEvent {
	t.Helper()

	deadline := time.After(timeout)
	events := make([]GovernanceEvent, 0, count)
	for len(events) < count {
		select {
		case event := <-eventCh:
			events = append(events, event)
		case <-deadline:
			t.Fatalf("timed out waiting for %d governance events, got %d", count, len(events))
		}
	}
	return events
}

func waitForSourceState(t *testing.T, st store.Store, sourceID string, predicate func(*store.DataSourceRow) bool) *store.DataSourceRow {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		row, err := st.GetDataSource(context.Background(), sourceID)
		require.NoError(t, err)
		if predicate(row) {
			return row
		}
		time.Sleep(20 * time.Millisecond)
	}

	row, err := st.GetDataSource(context.Background(), sourceID)
	require.NoError(t, err)
	t.Fatalf("timed out waiting for source state, last status=%s", row.Status)
	return nil
}

func waitForSyncCall(t *testing.T, ch <-chan int, name string) int {
	t.Helper()

	select {
	case call := <-ch:
		return call
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
		return 0
	}
}

func waitForSchemaObjectByName(t *testing.T, st store.Store, sourceID string, name string, schema *string, timeout time.Duration) *store.SchemaObjectRow {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		obj, err := st.GetSchemaObjectByName(context.Background(), sourceID, name, schema)
		require.NoError(t, err)
		if obj != nil {
			return obj
		}
		time.Sleep(20 * time.Millisecond)
	}

	_, err := st.GetSchemaObjectByName(context.Background(), sourceID, name, schema)
	require.NoError(t, err)
	t.Fatalf("timed out waiting for schema object %s", name)
	return nil
}
