package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStore_GetDashboardCounts_EmptyDB(t *testing.T) {
	st := testSQLiteStoreWithMigrations(t)
	defer func() { _ = st.Close() }()

	ctx := context.Background()
	counts, err := st.GetDashboardCounts(ctx, "nonexistent-user")
	require.NoError(t, err)
	require.NotNil(t, counts)

	assert.Equal(t, int64(0), counts.TotalSources)
	assert.Equal(t, int64(0), counts.TotalObjects)
	assert.Equal(t, int64(0), counts.TotalColumns)
	assert.Equal(t, int64(0), counts.TotalTerms)
	assert.Equal(t, int64(0), counts.TotalMappings)
	assert.Equal(t, int64(0), counts.TotalDQRules)
	assert.Equal(t, int64(0), counts.ActiveDQRules)
	assert.Equal(t, float64(0), counts.OverallPassRate)
	assert.Equal(t, int64(0), counts.TotalTags)
	assert.Equal(t, int64(0), counts.RecentChanges)
	assert.Equal(t, int64(0), counts.TotalAlertRules)
	assert.Equal(t, int64(0), counts.TotalUsers)
	assert.Equal(t, int64(0), counts.UnreadNotifications)
}

func TestSQLiteStore_GetDashboardCounts_WithData(t *testing.T) {
	st := testSQLiteStoreWithMigrations(t)
	defer func() { _ = st.Close() }()

	ctx := context.Background()

	userID, err := st.CreateUser(ctx, &UserCreate{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         "admin",
	})
	require.NoError(t, err)

	sourceID, err := st.CreateDataSource(ctx, &DataSourceCreate{
		Name: "test-source",
		Type: "mysql",
		Host: "localhost",
		Port: 3306,
		Database: "testdb",
		ConnectionConfig: "encrypted-config",
	})
	require.NoError(t, err)

	objID, err := st.CreateSchemaObject(ctx, &SchemaObjectCreate{
		SourceID: sourceID,
		Name:     "test_table",
		Type:     "table",
	})
	require.NoError(t, err)

	err = st.CreateColumn(ctx, &ColumnCreate{
		ObjectID:        objID,
		Name:            "id",
		DataType:        "int",
		FullDataType:    "int(11)",
		IsNullable:      false,
		IsPrimaryKey:    true,
		OrdinalPosition: 1,
	})
	require.NoError(t, err)

	termID, err := st.CreateBusinessTerm(ctx, &BusinessTermCreate{
		Name:     "Test Term",
		Category: "general",
	})
	require.NoError(t, err)
	_ = termID

	tagID, err := st.CreateTag(ctx, &TagCreate{
		Name:  "test-tag",
		Color: "#ff0000",
	})
	require.NoError(t, err)
	_ = tagID

	counts, err := st.GetDashboardCounts(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, counts)

	assert.Equal(t, int64(1), counts.TotalSources)
	assert.Equal(t, int64(1), counts.TotalObjects)
	assert.Equal(t, int64(1), counts.TotalColumns)
	assert.Equal(t, int64(1), counts.TotalTerms)
	assert.Equal(t, int64(1), counts.TotalTags)
	assert.Equal(t, int64(0), counts.TotalMappings)
	assert.Equal(t, int64(0), counts.TotalDQRules)
	assert.Equal(t, int64(0), counts.UnreadNotifications)
}

func TestSQLiteStore_GetDashboardCounts_UnreadNotifications(t *testing.T) {
	st := testSQLiteStoreWithMigrations(t)
	defer func() { _ = st.Close() }()

	ctx := context.Background()

	user1ID, err := st.CreateUser(ctx, &UserCreate{
		Username:     "user1",
		Email:        "user1@example.com",
		PasswordHash: "hash1",
		Role:         "user",
	})
	require.NoError(t, err)

	user2ID, err := st.CreateUser(ctx, &UserCreate{
		Username:     "user2",
		Email:        "user2@example.com",
		PasswordHash: "hash2",
		Role:         "user",
	})
	require.NoError(t, err)

	sourceID, err := st.CreateDataSource(ctx, &DataSourceCreate{
		Name:             "test-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: "encrypted-config",
	})
	require.NoError(t, err)

	objID, err := st.CreateSchemaObject(ctx, &SchemaObjectCreate{
		SourceID: sourceID,
		Name:     "test_table",
		Type:     "table",
	})
	require.NoError(t, err)

	changeID1 := "change-1"
	err = st.CreateSchemaChange(ctx, &SchemaChangeCreate{
		ID:         changeID1,
		SourceID:   sourceID,
		ObjectID:   &objID,
		ChangeType: "add_column",
		ObjectType: "column",
		ObjectName: "test_col",
		DetectedAt: "2026-05-28T00:00:00Z",
	})
	require.NoError(t, err)

	changeID2 := "change-2"
	err = st.CreateSchemaChange(ctx, &SchemaChangeCreate{
		ID:         changeID2,
		SourceID:   sourceID,
		ObjectID:   &objID,
		ChangeType: "add_column",
		ObjectType: "column",
		ObjectName: "test_col2",
		DetectedAt: "2026-05-28T00:00:00Z",
	})
	require.NoError(t, err)

	changeID3 := "change-3"
	err = st.CreateSchemaChange(ctx, &SchemaChangeCreate{
		ID:         changeID3,
		SourceID:   sourceID,
		ObjectID:   &objID,
		ChangeType: "add_column",
		ObjectType: "column",
		ObjectName: "test_col3",
		DetectedAt: "2026-05-28T00:00:00Z",
	})
	require.NoError(t, err)

	notif1ID, err := st.CreateNotification(ctx, &NotificationCreate{
		ChangeID:    changeID1,
		SourceID:    sourceID,
		Title:       "Test Notification 1",
		Message:     "Test message 1",
		ChangeType:  "add_column",
		ObjectType:  "column",
		ObjectName:  "test_col",
		NotifyInApp: true,
	})
	require.NoError(t, err)
	_ = notif1ID

	_, err = st.CreateNotification(ctx, &NotificationCreate{
		ChangeID:    changeID2,
		SourceID:    sourceID,
		Title:       "Test Notification 2",
		Message:     "Test message 2",
		ChangeType:  "add_column",
		ObjectType:  "column",
		ObjectName:  "test_col2",
		NotifyInApp: true,
	})
	require.NoError(t, err)

	notif3ID, err := st.CreateNotification(ctx, &NotificationCreate{
		ChangeID:    changeID3,
		SourceID:    sourceID,
		Title:       "Test Notification 3",
		Message:     "Test message 3",
		ChangeType:  "add_column",
		ObjectType:  "column",
		ObjectName:  "test_col3",
		NotifyInApp: true,
	})
	require.NoError(t, err)

	err = st.MarkNotificationAsRead(ctx, user1ID, notif3ID)
	require.NoError(t, err)

	counts1, err := st.GetDashboardCounts(ctx, user1ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts1.UnreadNotifications, "user1 should have 2 unread (3 total - 1 read)")

	counts2, err := st.GetDashboardCounts(ctx, user2ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), counts2.UnreadNotifications, "user2 should have 3 unread (none read)")
}
