package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/config"
	"github.com/jiangfire/datamaplite/internal/crypto"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/scanner"
	"github.com/jiangfire/datamaplite/internal/service"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type apiSyncTestScanner struct {
	schema *scanner.SchemaInfo
	err    error
}

func (s *apiSyncTestScanner) TestConnection(ctx context.Context, config scanner.ConnectionConfig) error {
	return nil
}

func (s *apiSyncTestScanner) ScanSchema(ctx context.Context, config scanner.ConnectionConfig) (*scanner.SchemaInfo, error) {
	return s.schema, s.err
}

func TestTriggerSyncAPI_PropagatesAuditToGovernanceEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	st, err := store.NewSQLiteStore(ctx, &config.DatabaseConfig{
		Type:           "sqlite",
		SQLitePath:     filepath.Join(t.TempDir(), "api-sync.db"),
		SQLiteMaxConns: 1,
		SQLiteMinConns: 1,
	}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, st.Close())
	})

	cipher, err := crypto.NewCipher("12345678901234567890123456789012")
	require.NoError(t, err)

	registry := scanner.NewRegistry()
	registry.Register("mysql", &apiSyncTestScanner{
		schema: &scanner.SchemaInfo{
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
					},
				},
			},
		},
	})

	sourceService := service.NewSourceService(st, cipher, registry)

	eventCh := make(chan service.GovernanceEvent, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event service.GovernanceEvent
		require.NoError(t, json.NewDecoder(r.Body).Decode(&event))
		eventCh <- event
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sourceService.SetGovernanceEventService(service.NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop()))

	sourceHandler := &SourceHandler{
		Handler:       NewHandler(),
		sourceService: sourceService,
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("pass123456"), bcrypt.DefaultCost)
	require.NoError(t, err)

	_, err = st.CreateUser(ctx, &store.UserCreate{
		Username:     "tester",
		Email:        "tester@example.com",
		PasswordHash: string(passwordHash),
		Role:         string(model.UserRoleAdmin),
	})
	require.NoError(t, err)

	authService := service.NewAuthService(st, &service.AuthConfig{
		Enabled:         true,
		JWTSecret:       "test-secret",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		BcryptCost:      bcrypt.DefaultCost,
	})

	loginResp, err := authService.Login(ctx, &model.LoginRequest{
		Username: "tester",
		Password: "pass123456",
	})
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
		Name:             "http-sync-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "datamap",
		ConnectionConfig: encryptedConfig,
	})
	require.NoError(t, err)

	engine := gin.New()
	protected := engine.Group("/api/v1")
	protected.Use(AuthMiddleware(authService), GovernanceAuditMiddleware())
	protected.POST("/sources/:id/sync", sourceHandler.TriggerSync)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/sources/"+sourceID+"/sync", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
	req.Header.Set("X-Trace-ID", "trace-http-sync-001")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	row := waitForAPISourceState(t, st, sourceID, func(row *store.DataSourceRow) bool {
		return row.Status == "active"
	})
	assert.Equal(t, "active", row.Status)

	events := collectAPIEvents(t, eventCh, 2, 2*time.Second)
	require.Len(t, events, 2)

	seen := make(map[string]string, len(events))
	for _, event := range events {
		assert.Equal(t, "metadata.schema.changed", event.EventType)
		assert.Equal(t, loginResp.User.ID, event.ActorID)
		assert.Equal(t, "trace-http-sync-001", event.TraceID)
		assert.Equal(t, "http", event.Payload["audit_origin"])
		assert.Equal(t, "POST /api/v1/sources/:id/sync", event.Payload["audit_operation"])
		seen[event.Payload["object_name"].(string)] = event.Payload["change_type"].(string)
	}

	assert.Equal(t, "add_object", seen["users"])
	assert.Equal(t, "add_column", seen["users.id"])
}

func collectAPIEvents(t *testing.T, eventCh <-chan service.GovernanceEvent, count int, timeout time.Duration) []service.GovernanceEvent {
	t.Helper()

	deadline := time.After(timeout)
	events := make([]service.GovernanceEvent, 0, count)
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

func waitForAPISourceState(t *testing.T, st store.Store, sourceID string, predicate func(*store.DataSourceRow) bool) *store.DataSourceRow {
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
