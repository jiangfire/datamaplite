package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiangfire/datamaplite/internal/config"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGovernanceEventService_Publish_UsesAuditMeta(t *testing.T) {
	var capturedBody GovernanceEvent
	var capturedTraceID string
	var capturedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID = r.Header.Get("X-Trace-ID")
		capturedAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())

	ctx := WithGovernanceAuditMeta(context.Background(), GovernanceAuditMeta{
		ActorID:   "mcp:datamap",
		TraceID:   "trace-123",
		Origin:    "mcp",
		Operation: "create_tag",
	})

	err := eventService.Publish(ctx, GovernanceEvent{
		EventType:    "mcp.governance.action",
		ResourceType: "tag",
		ResourceID:   "tag-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "Bearer integration-token", capturedAuth)
	assert.Equal(t, "trace-123", capturedTraceID)
	assert.Equal(t, "mcp:datamap", capturedBody.ActorID)
	assert.Equal(t, "trace-123", capturedBody.TraceID)
	assert.Equal(t, "mcp", capturedBody.Payload["audit_origin"])
	assert.Equal(t, "create_tag", capturedBody.Payload["audit_operation"])
}

func TestSourceService_PublishSchemaChangeEvent_ReusesAuditTrace(t *testing.T) {
	var capturedBody GovernanceEvent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())

	sourceService := &SourceService{
		governanceEventService: eventService,
	}

	ctx := WithGovernanceAuditMeta(context.Background(), GovernanceAuditMeta{
		ActorID:   "mcp:datamap",
		TraceID:   "trace-sync-001",
		Origin:    "mcp",
		Operation: "trigger_source_sync",
	})

	err := sourceService.publishSchemaChangeEvent(ctx, "mysql-prod", &SchemaChangeInfo{
		ID:         "chg-1",
		SourceID:   "src-1",
		ChangeType: "add_column",
		ObjectType: "column",
		ObjectName: "users.email",
		DetectedAt: "2026-03-27T09:00:00Z",
	})

	require.NoError(t, err)
	assert.Equal(t, "metadata.schema.changed", capturedBody.EventType)
	assert.Equal(t, "mcp:datamap", capturedBody.ActorID)
	assert.Equal(t, "trace-sync-001", capturedBody.TraceID)
	assert.Equal(t, "mcp", capturedBody.Payload["audit_origin"])
	assert.Equal(t, "trigger_source_sync", capturedBody.Payload["audit_operation"])
}

func TestGovernanceEventService_Publish_RejectsMissingEventType(t *testing.T) {
	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         "http://example.invalid",
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())

	err := eventService.Publish(context.Background(), GovernanceEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "event_type is required")
}

func TestGovernanceEventService_OutboxOperations_RequireStore(t *testing.T) {
	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         "http://example.invalid",
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())

	_, err := eventService.ListOutbox(context.Background(), 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "governance outbox store is not configured")

	_, err = eventService.GetOutboxStats(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "governance outbox store is not configured")

	err = eventService.ReplayOutboxEvent(context.Background(), "outbox-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "governance outbox store is not configured")
}

func TestGovernanceEventService_Publish_RejectsNon2xxResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())

	err := eventService.Publish(context.Background(), GovernanceEvent{
		EventType:    "metadata.schema.changed",
		ResourceType: "schema_change",
		ResourceID:   "chg-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 502")
}

func TestGovernanceEventService_Publish_OutboxPersistsAndEventuallyDelivers(t *testing.T) {
	ctx := context.Background()
	st := newGovernanceSQLiteStore(t)

	var fail atomic.Bool
	fail.Store(true)
	var attempts int32
	deliveredEventIDs := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event GovernanceEvent
		require.NoError(t, json.NewDecoder(r.Body).Decode(&event))
		atomic.AddInt32(&attempts, 1)
		if fail.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		deliveredEventIDs <- event.EventID
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())
	eventService.SetStore(st)
	eventService.outboxRetryBaseDelay = 20 * time.Millisecond

	err := eventService.Publish(ctx, GovernanceEvent{
		EventID:      "evt-outbox-1",
		EventType:    "metadata.schema.changed",
		ResourceType: "schema_change",
		ResourceID:   "chg-1",
	})
	require.NoError(t, err)

	row := waitForOutboxRow(t, st, "evt-outbox-1", func(row *store.GovernanceOutboxEventRow) bool {
		return row.AttemptCount == 1 && row.Status == "pending" && row.DeliveredAt == nil
	})
	require.NotNil(t, row.LastError)
	assert.Contains(t, *row.LastError, "status 502")

	fail.Store(false)
	time.Sleep(30 * time.Millisecond)
	require.NoError(t, eventService.ProcessOutbox(ctx, 10))

	row = waitForOutboxRow(t, st, "evt-outbox-1", func(row *store.GovernanceOutboxEventRow) bool {
		return row.Status == "delivered" && row.DeliveredAt != nil
	})
	assert.Equal(t, 2, row.AttemptCount)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))

	select {
	case eventID := <-deliveredEventIDs:
		assert.Equal(t, "evt-outbox-1", eventID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for delivered governance event")
	}
}

func TestGovernanceEventService_Publish_DuplicateEventIDDoesNotDuplicateOutboxOrDelivery(t *testing.T) {
	ctx := context.Background()
	st := newGovernanceSQLiteStore(t)

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())
	eventService.SetStore(st)

	event := GovernanceEvent{
		EventID:      "evt-duplicate-1",
		EventType:    "mcp.governance.action",
		ResourceType: "tag",
		ResourceID:   "tag-1",
	}
	require.NoError(t, eventService.Publish(ctx, event))
	require.NoError(t, eventService.Publish(ctx, event))

	row := waitForOutboxRow(t, st, "evt-duplicate-1", func(row *store.GovernanceOutboxEventRow) bool {
		return row.Status == "delivered"
	})
	assert.Equal(t, 1, row.AttemptCount)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))

	rows, err := st.ListGovernanceOutboxEvents(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestGovernanceEventService_Publish_DuplicateAckIsTreatedAsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAlreadyReported)
	}))
	defer server.Close()

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())

	err := eventService.Publish(context.Background(), GovernanceEvent{
		EventID:      "evt-duplicate-ack",
		EventType:    "mcp.governance.action",
		ResourceType: "tag",
		ResourceID:   "tag-1",
	})

	require.NoError(t, err)
}

func TestGovernanceEventService_ProcessOutbox_DeadLettersAfterMaxAttemptsAndCanReplay(t *testing.T) {
	ctx := context.Background()
	st := newGovernanceSQLiteStore(t)

	var fail atomic.Bool
	fail.Store(true)
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		if fail.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())
	eventService.SetStore(st)
	eventService.outboxMaxAttempts = 1

	require.NoError(t, eventService.Publish(ctx, GovernanceEvent{
		EventID:      "evt-dead-letter-1",
		EventType:    "metadata.schema.changed",
		ResourceType: "schema_change",
		ResourceID:   "chg-dead-letter-1",
	}))

	row := waitForOutboxRow(t, st, "evt-dead-letter-1", func(row *store.GovernanceOutboxEventRow) bool {
		return row.Status == "dead_letter"
	})
	assert.Equal(t, 1, row.AttemptCount)
	require.NotNil(t, row.LastError)
	assert.Contains(t, *row.LastError, "status 502")

	stats, err := eventService.GetOutboxStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.EqualValues(t, 1, stats.DeadLetterCount)
	assert.EqualValues(t, 0, stats.DeliveredCount)

	require.NoError(t, eventService.ProcessOutbox(ctx, 10))
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))

	require.NoError(t, eventService.ReplayOutboxEvent(ctx, row.ID))
	replayed, err := st.GetGovernanceOutboxEvent(ctx, row.ID)
	require.NoError(t, err)
	require.NotNil(t, replayed)
	assert.Equal(t, "pending", replayed.Status)
	assert.Equal(t, 0, replayed.AttemptCount)
	assert.Nil(t, replayed.DeliveredAt)
	assert.Nil(t, replayed.LastError)

	fail.Store(false)
	require.NoError(t, eventService.ProcessOutbox(ctx, 10))

	delivered := waitForOutboxRow(t, st, "evt-dead-letter-1", func(row *store.GovernanceOutboxEventRow) bool {
		return row.Status == "delivered" && row.DeliveredAt != nil
	})
	assert.Equal(t, 1, delivered.AttemptCount)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))

	stats, err = eventService.GetOutboxStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.EqualValues(t, 0, stats.DeadLetterCount)
	assert.EqualValues(t, 1, stats.DeliveredCount)
}

func TestGovernanceEventService_ProcessOutbox_InvalidPayloadDeadLettersWithoutDispatch(t *testing.T) {
	ctx := context.Background()
	st := newGovernanceSQLiteStore(t)

	created, err := st.EnqueueGovernanceOutboxEvent(ctx, &store.GovernanceOutboxEventCreate{
		ID:            "outbox-invalid-payload",
		EventID:       "evt-invalid-payload",
		EventType:     "metadata.schema.changed",
		TraceID:       "trace-invalid-payload",
		ResourceType:  "schema_change",
		ResourceID:    "chg-invalid-payload",
		Payload:       `{"event_id":`,
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano),
	})
	require.NoError(t, err)
	require.True(t, created)

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())
	eventService.SetStore(st)
	eventService.outboxMaxAttempts = 1

	err = eventService.ProcessOutbox(ctx, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
	assert.Equal(t, int32(0), atomic.LoadInt32(&attempts))

	row := waitForOutboxRow(t, st, "evt-invalid-payload", func(row *store.GovernanceOutboxEventRow) bool {
		return row.Status == "dead_letter"
	})
	assert.Equal(t, 1, row.AttemptCount)
	require.NotNil(t, row.LastError)
	assert.Contains(t, *row.LastError, "unexpected end of JSON input")

	require.NoError(t, eventService.ProcessOutbox(ctx, 10))
	assert.Equal(t, int32(0), atomic.LoadInt32(&attempts))
}

func TestGovernanceEventService_ReplayOutboxEvent_RejectsDeliveredEvent(t *testing.T) {
	ctx := context.Background()
	st := newGovernanceSQLiteStore(t)

	created, err := st.EnqueueGovernanceOutboxEvent(ctx, &store.GovernanceOutboxEventCreate{
		ID:            "outbox-replay-delivered",
		EventID:       "evt-replay-delivered",
		EventType:     "metadata.schema.changed",
		TraceID:       "trace-replay-delivered",
		ResourceType:  "schema_change",
		ResourceID:    "chg-replay-delivered",
		Payload:       `{"event_id":"evt-replay-delivered","event_type":"metadata.schema.changed"}`,
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, st.MarkGovernanceOutboxDelivered(ctx, "outbox-replay-delivered", time.Now().UTC().Format(time.RFC3339Nano)))

	eventService := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         "http://example.invalid",
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())
	eventService.SetStore(st)

	err = eventService.ReplayOutboxEvent(ctx, "outbox-replay-delivered")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not replayable")
}

func TestGovernanceEventService_ProcessOutbox_LeasePreventsDoubleDispatchAcrossInstances(t *testing.T) {
	ctx := context.Background()
	st := newGovernanceSQLiteStore(t)

	payload, err := json.Marshal(GovernanceEvent{
		EventID:      "evt-lease-1",
		EventType:    "metadata.schema.changed",
		ResourceType: "schema_change",
		ResourceID:   "chg-lease-1",
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
		ActorID:      "system",
		TraceID:      "trace-lease-1",
	})
	require.NoError(t, err)
	created, err := st.EnqueueGovernanceOutboxEvent(ctx, &store.GovernanceOutboxEventCreate{
		ID:            "outbox-lease-1",
		EventID:       "evt-lease-1",
		EventType:     "metadata.schema.changed",
		TraceID:       "trace-lease-1",
		ResourceType:  "schema_change",
		ResourceID:    "chg-lease-1",
		Payload:       string(payload),
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	require.NoError(t, err)
	require.True(t, created)

	requestStarted := make(chan struct{}, 1)
	releaseRequest := make(chan struct{})
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		requestStarted <- struct{}{}
		<-releaseRequest
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	first := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())
	first.SetStore(st)

	second := NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())
	second.SetStore(st)

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- first.ProcessOutbox(ctx, 1)
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first dispatcher request")
	}

	require.NoError(t, second.ProcessOutbox(ctx, 1))
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))

	close(releaseRequest)
	require.NoError(t, <-firstDone)

	row := waitForOutboxRow(t, st, "evt-lease-1", func(row *store.GovernanceOutboxEventRow) bool {
		return row.Status == "delivered"
	})
	assert.Equal(t, 1, row.AttemptCount)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func newGovernanceSQLiteStore(t *testing.T) store.Store {
	t.Helper()

	st, err := store.NewSQLiteStore(context.Background(), &config.DatabaseConfig{
		Type:           "sqlite",
		SQLitePath:     filepath.Join(t.TempDir(), "governance-test.db"),
		SQLiteMaxConns: 1,
		SQLiteMinConns: 1,
	}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, st.Close())
	})
	return st
}

func waitForOutboxRow(t *testing.T, st store.Store, eventID string, predicate func(*store.GovernanceOutboxEventRow) bool) *store.GovernanceOutboxEventRow {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rows, err := st.ListGovernanceOutboxEvents(context.Background(), 20)
		require.NoError(t, err)
		for _, row := range rows {
			if row.EventID == eventID && predicate(row) {
				return row
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	rows, err := st.ListGovernanceOutboxEvents(context.Background(), 20)
	require.NoError(t, err)
	t.Fatalf("timed out waiting for governance outbox row %s; rows=%v", eventID, rows)
	return nil
}
