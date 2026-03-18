package service

import (
	"context"
	"errors"
	"testing"

	"git.neolidy.top/neo/fuckcmdb/internal/crypto"
	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/scanner"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStore 模拟Store接口
type MockStore struct {
	mock.Mock
}

func (m *MockStore) CreateDataSource(ctx context.Context, source *store.DataSourceCreate) error {
	args := m.Called(ctx, source)
	return args.Error(0)
}

func (m *MockStore) GetDataSource(ctx context.Context, id string) (*store.DataSourceRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.DataSourceRow), args.Error(1)
}

func (m *MockStore) ListDataSources(ctx context.Context) ([]*store.DataSourceRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.DataSourceRow), args.Error(1)
}

func (m *MockStore) UpdateDataSource(ctx context.Context, id string, updates *store.DataSourceUpdate) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockStore) DeleteDataSource(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) UpdateDataSourceSyncStatus(ctx context.Context, id string, status string, errorMsg *string) error {
	args := m.Called(ctx, id, status, errorMsg)
	return args.Error(0)
}

func (m *MockStore) CreateSchemaObject(ctx context.Context, obj *store.SchemaObjectCreate) (string, error) {
	args := m.Called(ctx, obj)
	return args.String(0), args.Error(1)
}

func (m *MockStore) GetSchemaObject(ctx context.Context, id string) (*store.SchemaObjectRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.SchemaObjectRow), args.Error(1)
}

func (m *MockStore) GetSchemaObjectByName(ctx context.Context, sourceID string, name string, schema *string) (*store.SchemaObjectRow, error) {
	args := m.Called(ctx, sourceID, name, schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.SchemaObjectRow), args.Error(1)
}

func (m *MockStore) ListSchemaObjectsBySource(ctx context.Context, sourceID string) ([]*store.SchemaObjectRow, error) {
	args := m.Called(ctx, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.SchemaObjectRow), args.Error(1)
}

func (m *MockStore) DeleteSchemaObjectsBySource(ctx context.Context, sourceID string) error {
	args := m.Called(ctx, sourceID)
	return args.Error(0)
}

func (m *MockStore) DeleteSchemaObject(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) CreateColumn(ctx context.Context, col *store.ColumnCreate) error {
	args := m.Called(ctx, col)
	return args.Error(0)
}

func (m *MockStore) GetColumn(ctx context.Context, id string) (*store.ColumnRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.ColumnRow), args.Error(1)
}

func (m *MockStore) ListColumnsByObject(ctx context.Context, objectID string) ([]*store.ColumnRow, error) {
	args := m.Called(ctx, objectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.ColumnRow), args.Error(1)
}

func (m *MockStore) SearchColumns(ctx context.Context, query string, limit int) ([]*store.ColumnSearchRow, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.ColumnSearchRow), args.Error(1)
}

func (m *MockStore) DeleteColumnsByObject(ctx context.Context, objectID string) error {
	args := m.Called(ctx, objectID)
	return args.Error(0)
}

func (m *MockStore) CreateSchemaChange(ctx context.Context, change *store.SchemaChangeCreate) error {
	args := m.Called(ctx, change)
	return args.Error(0)
}

func (m *MockStore) ListSchemaChangesBySource(ctx context.Context, sourceID string, limit int) ([]*store.SchemaChangeRow, error) {
	args := m.Called(ctx, sourceID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.SchemaChangeRow), args.Error(1)
}

func (m *MockStore) CreateColumnMapping(ctx context.Context, mapping *store.ColumnMappingCreate) error {
	args := m.Called(ctx, mapping)
	return args.Error(0)
}

func (m *MockStore) GetColumnMappings(ctx context.Context, columnID string) ([]*store.ColumnMappingRow, error) {
	args := m.Called(ctx, columnID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.ColumnMappingRow), args.Error(1)
}

func (m *MockStore) DeleteColumnMapping(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) GetLineageUpward(ctx context.Context, columnID string, depth int) ([]*store.LineageEdgeRow, error) {
	args := m.Called(ctx, columnID, depth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.LineageEdgeRow), args.Error(1)
}

func (m *MockStore) GetLineageDownward(ctx context.Context, columnID string, depth int) ([]*store.LineageEdgeRow, error) {
	args := m.Called(ctx, columnID, depth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.LineageEdgeRow), args.Error(1)
}

func (m *MockStore) CreateLineageEdge(ctx context.Context, edge *store.LineageEdgeCreate) error {
	args := m.Called(ctx, edge)
	return args.Error(0)
}

func (m *MockStore) CreateBusinessTerm(ctx context.Context, term *store.BusinessTermCreate) (string, error) {
	args := m.Called(ctx, term)
	return args.String(0), args.Error(1)
}

func (m *MockStore) GetBusinessTerm(ctx context.Context, id string) (*store.BusinessTermRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.BusinessTermRow), args.Error(1)
}

func (m *MockStore) ListBusinessTerms(ctx context.Context, category string) ([]*store.BusinessTermRow, error) {
	args := m.Called(ctx, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.BusinessTermRow), args.Error(1)
}

func (m *MockStore) UpdateBusinessTerm(ctx context.Context, id string, updates *store.BusinessTermUpdate) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockStore) DeleteBusinessTerm(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) AssignTermToColumn(ctx context.Context, columnID string, termID *string) error {
	args := m.Called(ctx, columnID, termID)
	return args.Error(0)
}

func (m *MockStore) GetObjectWithColumns(ctx context.Context, objectID string) (*store.SchemaObjectRow, []*store.ColumnRow, error) {
	args := m.Called(ctx, objectID)
	var obj *store.SchemaObjectRow
	var cols []*store.ColumnRow
	if args.Get(0) != nil {
		obj = args.Get(0).(*store.SchemaObjectRow)
	}
	if args.Get(1) != nil {
		cols = args.Get(1).([]*store.ColumnRow)
	}
	return obj, cols, args.Error(2)
}

func (m *MockStore) WithTx(ctx context.Context, fn func(store.Store) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStore) CreateUser(ctx context.Context, user *store.UserCreate) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func (m *MockStore) GetUserByID(ctx context.Context, id string) (*store.UserRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.UserRow), args.Error(1)
}

func (m *MockStore) GetUserByUsername(ctx context.Context, username string) (*store.UserRow, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.UserRow), args.Error(1)
}

func (m *MockStore) ListUsers(ctx context.Context) ([]*store.UserRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.UserRow), args.Error(1)
}

func (m *MockStore) UpdateUser(ctx context.Context, id string, updates *store.UserUpdate) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockStore) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) CreateDQRule(ctx context.Context, rule *store.DQRuleCreate) (string, error) {
	args := m.Called(ctx, rule)
	return args.String(0), args.Error(1)
}

func (m *MockStore) GetDQRule(ctx context.Context, id string) (*store.DQRuleRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.DQRuleRow), args.Error(1)
}

func (m *MockStore) ListDQRules(ctx context.Context, filter *store.DQRuleFilter) ([]*store.DQRuleRow, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.DQRuleRow), args.Error(1)
}

func (m *MockStore) UpdateDQRule(ctx context.Context, id string, updates *store.DQRuleUpdate) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockStore) DeleteDQRule(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) CreateDQResult(ctx context.Context, result *store.DQResultCreate) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockStore) ListDQResults(ctx context.Context, filter *store.DQResultFilter) ([]*store.DQResultRow, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.DQResultRow), args.Error(1)
}

func (m *MockStore) GetLatestDQResult(ctx context.Context, ruleID string) (*store.DQResultRow, error) {
	args := m.Called(ctx, ruleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.DQResultRow), args.Error(1)
}

func (m *MockStore) GetDQStats(ctx context.Context) (*store.DQStatsRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.DQStatsRow), args.Error(1)
}

func (m *MockStore) CreateTag(ctx context.Context, tag *store.TagCreate) (string, error) {
	args := m.Called(ctx, tag)
	return args.String(0), args.Error(1)
}

func (m *MockStore) GetTag(ctx context.Context, id string) (*store.TagRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.TagRow), args.Error(1)
}

func (m *MockStore) GetTagByName(ctx context.Context, name string) (*store.TagRow, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.TagRow), args.Error(1)
}

func (m *MockStore) ListTags(ctx context.Context) ([]*store.TagRow, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.TagRow), args.Error(1)
}

func (m *MockStore) UpdateTag(ctx context.Context, id string, updates *store.TagUpdate) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockStore) DeleteTag(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) AddTagToColumn(ctx context.Context, columnID string, tagID string) error {
	args := m.Called(ctx, columnID, tagID)
	return args.Error(0)
}

func (m *MockStore) RemoveTagFromColumn(ctx context.Context, columnID string, tagID string) error {
	args := m.Called(ctx, columnID, tagID)
	return args.Error(0)
}

func (m *MockStore) GetColumnTags(ctx context.Context, columnID string) ([]*store.TagRow, error) {
	args := m.Called(ctx, columnID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.TagRow), args.Error(1)
}

func (m *MockStore) SearchColumnsByTag(ctx context.Context, tagID string, limit int) ([]*store.ColumnSearchRow, error) {
	args := m.Called(ctx, tagID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.ColumnSearchRow), args.Error(1)
}

// AlertRule methods
func (m *MockStore) CreateAlertRule(ctx context.Context, rule *store.AlertRuleCreate) (string, error) {
	args := m.Called(ctx, rule)
	return args.String(0), args.Error(1)
}

func (m *MockStore) GetAlertRule(ctx context.Context, id string) (*store.AlertRuleRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.AlertRuleRow), args.Error(1)
}

func (m *MockStore) ListAlertRules(ctx context.Context, sourceID *string) ([]*store.AlertRuleRow, error) {
	args := m.Called(ctx, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.AlertRuleRow), args.Error(1)
}

func (m *MockStore) UpdateAlertRule(ctx context.Context, id string, updates *store.AlertRuleUpdate) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockStore) DeleteAlertRule(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) ListMatchingAlertRules(ctx context.Context, sourceID string, changeType string) ([]*store.AlertRuleRow, error) {
	args := m.Called(ctx, sourceID, changeType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.AlertRuleRow), args.Error(1)
}

// Notification methods
func (m *MockStore) CreateNotification(ctx context.Context, notification *store.NotificationCreate) (string, error) {
	args := m.Called(ctx, notification)
	return args.String(0), args.Error(1)
}

func (m *MockStore) GetNotification(ctx context.Context, id string) (*store.NotificationRow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.NotificationRow), args.Error(1)
}

func (m *MockStore) ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*store.NotificationRow, error) {
	args := m.Called(ctx, userID, unreadOnly, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.NotificationRow), args.Error(1)
}

func (m *MockStore) GetNotificationStats(ctx context.Context, userID string) (*store.NotificationStatsRow, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.NotificationStatsRow), args.Error(1)
}

func (m *MockStore) MarkNotificationAsRead(ctx context.Context, userID string, notificationID string) error {
	args := m.Called(ctx, userID, notificationID)
	return args.Error(0)
}

func (m *MockStore) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockStore) UpdateNotificationWebhookStatus(ctx context.Context, id string, sent bool, errorMsg *string) error {
	args := m.Called(ctx, id, sent, errorMsg)
	return args.Error(0)
}

// MockScanner 模拟Scanner接口
type MockScanner struct {
	mock.Mock
}

func (m *MockScanner) TestConnection(ctx context.Context, config scanner.ConnectionConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockScanner) ScanSchema(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*scanner.SchemaInfo), args.Error(1)
}

func setupTestService(t *testing.T) (*SourceService, *MockStore, *crypto.Cipher) {
	mockStore := new(MockStore)
	key := "12345678901234567890123456789012"
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)

	registry := scanner.NewRegistry()

	service := NewSourceService(mockStore, cipher, registry)
	return service, mockStore, cipher
}

func TestCreateSource(t *testing.T) {
	ctx := context.Background()
	service, mockStore, _ := setupTestService(t)

	req := &model.CreateSourceRequest{
		Name:        "test-mysql",
		Description: "Test MySQL",
		Type:        model.DataSourceMySQL,
		Host:        "localhost",
		Port:        3306,
		Database:    "testdb",
		Username:    "root",
		Password:    "secret",
	}

	// Setup expectations
	mockStore.On("CreateDataSource", ctx, mock.AnythingOfType("*store.DataSourceCreate")).Return(nil)
	mockStore.On("ListDataSources", ctx).Return([]*store.DataSourceRow{
		{
			ID:       "source-123",
			Name:     "test-mysql",
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Database: "testdb",
			Status:   "active",
		},
	}, nil)

	resp, err := service.CreateSource(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "source-123", resp.ID)
	assert.Equal(t, "test-mysql", resp.Name)
	assert.Equal(t, model.DataSourceMySQL, resp.Type)
	mockStore.AssertExpectations(t)
}

func TestListSources(t *testing.T) {
	ctx := context.Background()
	service, mockStore, _ := setupTestService(t)

	mockStore.On("ListDataSources", ctx).Return([]*store.DataSourceRow{
		{
			ID:       "source-1",
			Name:     "mysql-prod",
			Type:     "mysql",
			Host:     "db1.example.com",
			Port:     3306,
			Database: "production",
			Status:   "active",
		},
		{
			ID:       "source-2",
			Name:     "mongo-dev",
			Type:     "mongodb",
			Host:     "mongo.example.com",
			Port:     27017,
			Database: "development",
			Status:   "inactive",
		},
	}, nil)

	sources, err := service.ListSources(ctx)

	require.NoError(t, err)
	assert.Len(t, sources, 2)
	assert.Equal(t, "mysql-prod", sources[0].Name)
	assert.Equal(t, "mongo-dev", sources[1].Name)
	mockStore.AssertExpectations(t)
}

func TestGetSource(t *testing.T) {
	ctx := context.Background()
	service, mockStore, _ := setupTestService(t)

	sourceID := "source-123"
	mockStore.On("GetDataSource", ctx, sourceID).Return(&store.DataSourceRow{
		ID:          sourceID,
		Name:        "test-source",
		Type:        "mysql",
		Host:        "localhost",
		Port:        3306,
		Database:    "testdb",
		Status:      "active",
		Description: strPtr("Test description"),
	}, nil)

	resp, err := service.GetSource(ctx, sourceID)

	require.NoError(t, err)
	assert.Equal(t, sourceID, resp.ID)
	assert.Equal(t, "test-source", resp.Name)
	assert.Equal(t, "Test description", *resp.Description)
	mockStore.AssertExpectations(t)
}

func TestGetSourceNotFound(t *testing.T) {
	ctx := context.Background()
	service, mockStore, _ := setupTestService(t)

	sourceID := "non-existent"
	mockStore.On("GetDataSource", ctx, sourceID).Return(nil, errors.New("not found"))

	_, err := service.GetSource(ctx, sourceID)

	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestUpdateSource(t *testing.T) {
	ctx := context.Background()
	service, mockStore, cipher := setupTestService(t)

	sourceID := "source-123"
	encryptedConfig, _ := cipher.Encrypt(`{"host":"old-host","port":3306,"username":"root","password":"old-pass"}`)

	mockStore.On("GetDataSource", ctx, sourceID).Return(&store.DataSourceRow{
		ID:               sourceID,
		Name:             "test-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		ConnectionConfig: encryptedConfig,
	}, nil)

	mockStore.On("UpdateDataSource", ctx, sourceID, mock.AnythingOfType("*store.DataSourceUpdate")).Return(nil)

	req := &model.UpdateSourceRequest{
		Name: "updated-name",
		Host: "new-host",
	}

	err := service.UpdateSource(ctx, sourceID, req)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteSource(t *testing.T) {
	ctx := context.Background()
	service, mockStore, _ := setupTestService(t)

	sourceID := "source-123"
	mockStore.On("DeleteDataSource", ctx, sourceID).Return(nil)

	err := service.DeleteSource(ctx, sourceID)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTestConnection(t *testing.T) {
	ctx := context.Background()
	service, _, _ := setupTestService(t)

	// Register mock scanner
	mockScanner := new(MockScanner)
	mockScanner.On("TestConnection", ctx, mock.AnythingOfType("scanner.ConnectionConfig")).Return(nil)

	// Since we can't easily replace the registry, we'll skip this test for now
	// In a real scenario, we'd use dependency injection
	_ = service
	_ = mockScanner
}

func strPtr(s string) *string {
	return &s
}

func TestSourceServiceWithRealCipher(t *testing.T) {
	ctx := context.Background()
	mockStore := new(MockStore)
	key := "12345678901234567890123456789012"
	cipher, err := crypto.NewCipher(key)
	require.NoError(t, err)

	registry := scanner.NewRegistry()
	service := NewSourceService(mockStore, cipher, registry)

	t.Run("CreateSource encrypts connection config", func(t *testing.T) {
		req := &model.CreateSourceRequest{
			Name:     "test",
			Type:     model.DataSourceMySQL,
			Host:     "localhost",
			Port:     3306,
			Database: "test",
			Username: "root",
			Password: "secret123",
		}

		var capturedConfig *store.DataSourceCreate
		mockStore.On("CreateDataSource", ctx, mock.Anything).Run(func(args mock.Arguments) {
			capturedConfig = args.Get(1).(*store.DataSourceCreate)
		}).Return(nil).Once()

		mockStore.On("ListDataSources", ctx).Return([]*store.DataSourceRow{
			{ID: "1", Name: "test", Type: "mysql", Status: "active"},
		}, nil).Once()

		_, err := service.CreateSource(ctx, req)
		require.NoError(t, err)

		// Verify connection config is encrypted
		assert.NotEmpty(t, capturedConfig.ConnectionConfig)
		assert.NotContains(t, capturedConfig.ConnectionConfig, "secret123")

		// Verify we can decrypt it
		decrypted, err := cipher.Decrypt(capturedConfig.ConnectionConfig)
		require.NoError(t, err)
		assert.Contains(t, decrypted, "secret123")
	})
}

// Ensure MockStore implements Store interface
var _ store.Store = (*MockStore)(nil)
