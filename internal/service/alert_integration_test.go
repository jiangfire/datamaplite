package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/scanner"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type sqliteAlertTestEnv struct {
	sourceEnv           *sqliteSourceTestEnv
	alertService        *AlertService
	notificationService *NotificationService
	testUserID          string
}

func newSQLiteAlertTestEnv(t *testing.T) *sqliteAlertTestEnv {
	t.Helper()

	sourceEnv := newSQLiteSourceTestEnv(t)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("testpass"), 10)
	require.NoError(t, err)

	userID, err := sourceEnv.store.CreateUser(context.Background(), &store.UserCreate{
		Username:     "testadmin",
		Email:        "test@example.com",
		PasswordHash: string(passwordHash),
		Role:         "admin",
	})
	require.NoError(t, err)

	alertSvc := NewAlertService(sourceEnv.store, zap.NewNop())
	alertSvc.webhookValidator = func(string) error { return nil }

	return &sqliteAlertTestEnv{
		sourceEnv:           sourceEnv,
		alertService:        alertSvc,
		notificationService: NewNotificationService(sourceEnv.store, zap.NewNop()),
		testUserID:          userID,
	}
}

func TestAlertService_CreateAlertRule_SQLiteRoundTrip(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	resp, err := env.alertService.CreateAlertRule(ctx, &model.AlertRuleRequest{
		SourceID:      &env.sourceEnv.sourceID,
		Name:          "schema-change-rule",
		Description:   strPtr("watch schema changes"),
		ChangeTypes:   "alter_column",
		NotifyWebhook: false,
		NotifyInApp:   true,
		IsActive:      true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.ID)

	got, err := env.alertService.GetAlertRule(ctx, resp.ID)
	require.NoError(t, err)
	assert.Equal(t, resp.ID, got.ID)
	assert.Equal(t, "schema-change-rule", got.Name)
	assert.Equal(t, "alter_column", got.ChangeTypes)
	require.NotNil(t, got.SourceID)
	assert.Equal(t, env.sourceEnv.sourceID, *got.SourceID)
}

func TestAlertService_CreateAlertRule_RejectsObjectFromDifferentSource(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	otherSourceID := createAdditionalSourceForTest(t, env)
	require.NoError(t, env.sourceEnv.service.saveSchema(ctx, otherSourceID, &scanner.SchemaInfo{
		Objects: []scanner.ObjectInfo{
			{
				Name: "foreign_orders",
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
	}))

	foreignObj := mustGetSchemaObjectByName(t, env.sourceEnv.store, otherSourceID, "foreign_orders", nil)

	_, err := env.alertService.CreateAlertRule(ctx, &model.AlertRuleRequest{
		SourceID:      &env.sourceEnv.sourceID,
		ObjectID:      &foreignObj.ID,
		Name:          "cross-source-rule",
		Description:   strPtr("invalid cross source rule"),
		ChangeTypes:   "alter_column",
		NotifyWebhook: false,
		NotifyInApp:   true,
		IsActive:      true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "object does not belong to source")
}

func TestAlertService_CreateAlertRule_NormalizesChangeTypes(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	resp, err := env.alertService.CreateAlertRule(ctx, &model.AlertRuleRequest{
		SourceID:      &env.sourceEnv.sourceID,
		Name:          "normalized-change-types",
		Description:   strPtr("normalize spaces and duplicates"),
		ChangeTypes:   " drop_column, alter_column , drop_column ",
		NotifyWebhook: false,
		NotifyInApp:   true,
		IsActive:      true,
	})
	require.NoError(t, err)

	got, err := env.alertService.GetAlertRule(ctx, resp.ID)
	require.NoError(t, err)
	assert.Equal(t, "drop_column,alter_column", got.ChangeTypes)
}

func TestAlertService_CreateAlertRule_RejectsEmptyNormalizedChangeTypes(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	_, err := env.alertService.CreateAlertRule(ctx, &model.AlertRuleRequest{
		SourceID:      &env.sourceEnv.sourceID,
		Name:          "invalid-change-types",
		Description:   strPtr("invalid"),
		ChangeTypes:   " , , ",
		NotifyWebhook: false,
		NotifyInApp:   true,
		IsActive:      true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "change types are required")
}

func TestAlertService_ProcessSchemaChange_WebhookSuccessPersistsNotificationState(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	payloadCh := make(chan model.WebhookPayload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload model.WebhookPayload
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		payloadCh <- payload
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	rule := createAlertRuleForTest(t, env, "success-rule", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))

	payload := waitForWebhookPayload(t, payloadCh)
	assert.Equal(t, "schema_change", payload.Event)
	assert.Equal(t, env.sourceEnv.sourceID, payload.SourceID)
	assert.Equal(t, "sqlite-source", payload.SourceName)
	assert.Equal(t, "users.email", payload.ObjectName)
	assert.Equal(t, payload.ID, payload.NotificationID)
	require.NotNil(t, payload.RuleID)
	assert.Equal(t, rule.ID, *payload.RuleID)

	notifications, err := env.notificationService.ListNotifications(ctx, env.testUserID, false, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.Equal(t, "users.email", notifications[0].ObjectName)
	assert.True(t, notifications[0].WebhookSent)
	assert.Empty(t, notifications[0].WebhookError)

	stored, err := env.sourceEnv.store.GetNotification(ctx, notifications[0].ID)
	require.NoError(t, err)
	assert.True(t, stored.WebhookSent)
	assert.Nil(t, stored.WebhookError)
	require.NotNil(t, stored.RuleID)
	assert.Equal(t, rule.ID, *stored.RuleID)
	assert.Equal(t, change.ID, stored.ChangeID)
}

func TestAlertService_ProcessSchemaChange_DuplicateDeliverySkipsSecondWebhookAndReusesNotification(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	rule := createAlertRuleForTest(t, env, "dedupe-rule", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))
	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))

	notifications, err := env.notificationService.ListNotifications(ctx, env.testUserID, false, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.True(t, notifications[0].WebhookSent)
	require.NotNil(t, notifications[0].RuleID)
	assert.Equal(t, rule.ID, *notifications[0].RuleID)

	stored, err := env.sourceEnv.store.GetNotificationByRuleAndChange(ctx, rule.ID, change.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, notifications[0].ID, stored.ID)
}

func TestAlertService_ProcessSchemaChange_WebhookClientErrorPersistsFailureWithoutRetry(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	createAlertRuleForTest(t, env, "client-error-rule", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))

	notifications, err := env.notificationService.ListNotifications(ctx, env.testUserID, false, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.False(t, notifications[0].WebhookSent)
	assert.Contains(t, notifications[0].WebhookError, "webhook returned non-2xx status: 400")
}

func TestAlertService_ProcessSchemaChange_DuplicateAfterFailureReusesNotificationIDAndEventuallySucceeds(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	var succeed atomic.Bool
	receivedIDs := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload model.WebhookPayload
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		receivedIDs <- payload.ID

		if !succeed.Load() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	rule := createAlertRuleForTest(t, env, "retry-after-failure", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))
	first, err := env.sourceEnv.store.GetNotificationByRuleAndChange(ctx, rule.ID, change.ID)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.False(t, first.WebhookSent)

	succeed.Store(true)
	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))

	second, err := env.sourceEnv.store.GetNotificationByRuleAndChange(ctx, rule.ID, change.ID)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.ID, second.ID)
	assert.True(t, second.WebhookSent)

	notifications, err := env.notificationService.ListNotifications(ctx, env.testUserID, false, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.True(t, notifications[0].WebhookSent)

	for i := 0; i < 2; i++ {
		select {
		case got := <-receivedIDs:
			assert.Equal(t, first.ID, got)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for webhook payload id on attempt %d", i+1)
		}
	}
}

func TestAlertService_ProcessSchemaChange_WebhookServerErrorRetriesAndClearsFailure(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	createAlertRuleForTest(t, env, "retry-rule", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))

	notifications, err := env.notificationService.ListNotifications(ctx, env.testUserID, false, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.True(t, notifications[0].WebhookSent)
	assert.Empty(t, notifications[0].WebhookError)
}

func TestAlertService_ProcessSchemaChange_RetryAttemptsReuseStableIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	var attempts int32
	headers := make(chan string, 3)
	payloadKeys := make(chan string, 3)
	payloadIDs := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload model.WebhookPayload
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		headers <- r.Header.Get("Idempotency-Key")
		payloadKeys <- payload.IdempotencyKey
		payloadIDs <- payload.NotificationID
		assert.Equal(t, r.Header.Get("Idempotency-Key"), r.Header.Get("X-DataMap-Idempotency-Key"))
		assert.Equal(t, r.Header.Get("X-DataMap-Notification-ID"), payload.NotificationID)
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	rule := createAlertRuleForTest(t, env, "stable-idempotency", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))

	expectedKey := env.alertService.webhookIdempotencyKey(rule.ID, change.ID)
	for i := 0; i < 3; i++ {
		select {
		case got := <-headers:
			assert.Equal(t, expectedKey, got)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for idempotency header on attempt %d", i+1)
		}
		select {
		case got := <-payloadKeys:
			assert.Equal(t, expectedKey, got)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for idempotency payload on attempt %d", i+1)
		}
		select {
		case got := <-payloadIDs:
			assert.NotEmpty(t, got)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for notification payload id on attempt %d", i+1)
		}
	}
}

func TestAlertService_ProcessSchemaChange_WebhookConflictAcknowledgedAsSuccess(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	createAlertRuleForTest(t, env, "conflict-ack", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))

	notifications, err := env.notificationService.ListNotifications(ctx, env.testUserID, false, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.True(t, notifications[0].WebhookSent)
	assert.Empty(t, notifications[0].WebhookError)
}

func TestAlertService_ProcessSchemaChange_WebhookAlreadyReportedAcknowledgedAsSuccess(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAlreadyReported)
	}))
	defer server.Close()

	createAlertRuleForTest(t, env, "already-reported-ack", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))

	notifications, err := env.notificationService.ListNotifications(ctx, env.testUserID, false, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.True(t, notifications[0].WebhookSent)
	assert.Empty(t, notifications[0].WebhookError)
}

func TestAlertService_ProcessSchemaChange_ContextCancelDuringBackoffPersistsFailure(t *testing.T) {
	env := newSQLiteAlertTestEnv(t)

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	createAlertRuleForTest(t, env, "cancel-rule", "alter_column", server.URL, true, true)
	change := createStoredSchemaChangeForTest(t, env.sourceEnv.store, env.sourceEnv.sourceID, "alter_column", "users.email")

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	require.NoError(t, env.alertService.ProcessSchemaChange(ctx, change))
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))

	notifications, err := env.notificationService.ListNotifications(context.Background(), env.testUserID, false, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.False(t, notifications[0].WebhookSent)
	assert.Contains(t, notifications[0].WebhookError, "webhook cancelled during backoff")
}

func TestSourceService_SaveSchema_DropObjectTriggersScopedAlertRule(t *testing.T) {
	ctx := context.Background()
	env := newSQLiteAlertTestEnv(t)
	env.sourceEnv.service.SetAlertService(env.alertService)

	require.NoError(t, env.sourceEnv.service.saveSchema(ctx, env.sourceEnv.sourceID, &scanner.SchemaInfo{
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
	}))

	obj := mustGetSchemaObjectByName(t, env.sourceEnv.store, env.sourceEnv.sourceID, "legacy_users", nil)
	createAlertRuleForTestWithObject(t, env, "drop-object-rule", "drop_object", &obj.ID, "", false, true)

	require.NoError(t, env.sourceEnv.service.saveSchema(ctx, env.sourceEnv.sourceID, &scanner.SchemaInfo{Objects: []scanner.ObjectInfo{}}))

	notifications := waitForNotifications(t, env.notificationService, env.testUserID, 1)
	require.Len(t, notifications, 1)
	assert.Equal(t, "drop_object", notifications[0].ChangeType)
	assert.Equal(t, "legacy_users", notifications[0].ObjectName)
	assert.Contains(t, notifications[0].Title, "删除对象")
}

func createAlertRuleForTest(
	t *testing.T,
	env *sqliteAlertTestEnv,
	name string,
	changeTypes string,
	webhookURL string,
	notifyWebhook bool,
	notifyInApp bool,
) *model.AlertRuleResponse {
	t.Helper()

	desc := name
	var whURL *string
	if webhookURL != "" {
		whURL = &webhookURL
	}
	resp, err := env.alertService.CreateAlertRule(context.Background(), &model.AlertRuleRequest{
		SourceID:      &env.sourceEnv.sourceID,
		Name:          name,
		Description:   &desc,
		ChangeTypes:   changeTypes,
		NotifyWebhook: notifyWebhook,
		WebhookURL:    whURL,
		NotifyInApp:   notifyInApp,
		IsActive:      true,
	})
	require.NoError(t, err)
	return resp
}

func createAlertRuleForTestWithObject(
	t *testing.T,
	env *sqliteAlertTestEnv,
	name string,
	changeTypes string,
	objectID *string,
	webhookURL string,
	notifyWebhook bool,
	notifyInApp bool,
) *model.AlertRuleResponse {
	t.Helper()

	desc := name
	var whURL *string
	if webhookURL != "" {
		whURL = &webhookURL
	}
	resp, err := env.alertService.CreateAlertRule(context.Background(), &model.AlertRuleRequest{
		SourceID:      &env.sourceEnv.sourceID,
		ObjectID:      objectID,
		Name:          name,
		Description:   &desc,
		ChangeTypes:   changeTypes,
		NotifyWebhook: notifyWebhook,
		WebhookURL:    whURL,
		NotifyInApp:   notifyInApp,
		IsActive:      true,
	})
	require.NoError(t, err)
	return resp
}

func createStoredSchemaChangeForTest(
	t *testing.T,
	st store.Store,
	sourceID string,
	changeType string,
	objectName string,
) *SchemaChangeInfo {
	t.Helper()

	oldValue := "type=varchar(255)"
	newValue := "type=text"
	change := &store.SchemaChangeCreate{
		SourceID:   sourceID,
		ChangeType: changeType,
		ObjectType: "column",
		ObjectName: objectName,
		OldValue:   &oldValue,
		NewValue:   &newValue,
	}
	require.NoError(t, st.CreateSchemaChange(context.Background(), change))

	return &SchemaChangeInfo{
		ID:         change.ID,
		SourceID:   sourceID,
		ChangeType: changeType,
		ObjectType: "column",
		ObjectName: objectName,
		OldValue:   change.OldValue,
		NewValue:   change.NewValue,
		DetectedAt: change.DetectedAt,
	}
}

func createAdditionalSourceForTest(t *testing.T, env *sqliteAlertTestEnv) string {
	t.Helper()

	configJSON, err := scanner.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "other",
		Username: "root",
		Password: "secret",
	}.ToJSON()
	require.NoError(t, err)

	encryptedConfig, err := env.sourceEnv.service.cipher.Encrypt(configJSON)
	require.NoError(t, err)

	sourceID, err := env.sourceEnv.store.CreateDataSource(context.Background(), &store.DataSourceCreate{
		Name:             "other-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "other",
		ConnectionConfig: encryptedConfig,
	})
	require.NoError(t, err)
	return sourceID
}

func waitForWebhookPayload(t *testing.T, ch <-chan model.WebhookPayload) model.WebhookPayload {
	t.Helper()

	select {
	case payload := <-ch:
		return payload
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook payload")
		return model.WebhookPayload{}
	}
}

func waitForNotifications(t *testing.T, svc *NotificationService, userID string, count int) []*model.NotificationResponse {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		notifications, err := svc.ListNotifications(context.Background(), userID, false, 10)
		require.NoError(t, err)
		if len(notifications) >= count {
			return notifications
		}
		time.Sleep(20 * time.Millisecond)
	}

	notifications, err := svc.ListNotifications(context.Background(), userID, false, 10)
	require.NoError(t, err)
	t.Fatalf("timed out waiting for %d notifications, got %d", count, len(notifications))
	return nil
}

func TestValidateWebhookURL_BlocksRestrictedAddresses(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"loopback IPv4", "http://127.0.0.1/webhook", true},
		{"loopback IPv6", "http://[::1]/webhook", true},
		{"private IPv4", "http://192.168.1.1/webhook", true},
		{"private IPv4 10.x", "http://10.0.0.1/webhook", true},
		{"private IPv4 172.16", "http://172.16.0.1/webhook", true},
		{"link-local", "http://169.254.1.1/webhook", true},
		{"zero IPv4", "http://0.0.0.0/webhook", true},
		{"unspecified IPv6", "http://[::]/webhook", true},
		{"non-http scheme", "ftp://example.com/webhook", true},
		{"public http", "http://example.com/webhook", false},
		{"public https", "https://example.com/webhook", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWebhookURL(tc.url)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
