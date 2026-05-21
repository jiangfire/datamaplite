package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiangfire/datamaplite/internal/config"
	"github.com/jiangfire/datamaplite/internal/crypto"
	"github.com/jiangfire/datamaplite/internal/scanner"
	"github.com/jiangfire/datamaplite/internal/service"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeGovernancePublisher struct {
	enabled     bool
	ctx         context.Context
	event       service.GovernanceEvent
	err         error
	outbox      []*store.GovernanceOutboxEventRow
	stats       *store.GovernanceOutboxStatsRow
	replayedID  string
	replayError error
}

func (f *fakeGovernancePublisher) Enabled() bool {
	return f.enabled
}

func (f *fakeGovernancePublisher) Publish(ctx context.Context, event service.GovernanceEvent) error {
	f.ctx = ctx
	f.event = event
	return f.err
}

func (f *fakeGovernancePublisher) ListOutbox(ctx context.Context, limit int) ([]*store.GovernanceOutboxEventRow, error) {
	return f.outbox, nil
}

func (f *fakeGovernancePublisher) GetOutboxStats(ctx context.Context) (*store.GovernanceOutboxStatsRow, error) {
	return f.stats, nil
}

func (f *fakeGovernancePublisher) ReplayOutboxEvent(ctx context.Context, id string) error {
	f.replayedID = id
	return f.replayError
}

func connectTestClient(t *testing.T, server *Server) *mcp.ClientSession {
	t.Helper()

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	_, err := server.mcp.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = session.Close()
	})
	return session
}

func connectHTTPTestClient(t *testing.T, server *Server) *mcp.ClientSession {
	t.Helper()

	httpServer := httptest.NewServer(server.HTTPHandler())
	t.Cleanup(httpServer.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "http-test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL,
		DisableStandaloneSSE: true,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = session.Close()
	})
	return session
}

func newSQLiteSourceServiceForMCPTest(t *testing.T) (*service.SourceService, store.Store, string) {
	t.Helper()

	ctx := context.Background()
	st, err := store.NewSQLiteStore(ctx, &config.DatabaseConfig{
		Type:           "sqlite",
		SQLitePath:     filepath.Join(t.TempDir(), "mcp-server-test.db"),
		SQLiteMaxConns: 1,
		SQLiteMinConns: 1,
	}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, st.Close())
	})

	cipher, err := crypto.NewCipher("12345678901234567890123456789012")
	require.NoError(t, err)

	cfgJSON, err := scanner.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "datamap",
		Username: "root",
		Password: "secret",
	}.ToJSON()
	require.NoError(t, err)

	encryptedConfig, err := cipher.Encrypt(cfgJSON)
	require.NoError(t, err)

	sourceID, err := st.CreateDataSource(ctx, &store.DataSourceCreate{
		Name:             "mcp-source",
		Type:             "mysql",
		Host:             "localhost",
		Port:             3306,
		Database:         "datamap",
		ConnectionConfig: encryptedConfig,
	})
	require.NoError(t, err)

	return service.NewSourceService(st, cipher, scanner.NewRegistry()), st, sourceID
}

func TestServer_ListsToolsAndResources(t *testing.T) {
	server := New(&Dependencies{})
	session := connectTestClient(t, server)

	ctx := context.Background()
	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)

	toolNames := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	assert.Contains(t, toolNames, "list_sources")
	assert.Contains(t, toolNames, "assign_term_to_column")
	assert.Contains(t, toolNames, "create_column_mapping")
	assert.Contains(t, toolNames, "trigger_source_sync")
	assert.Contains(t, toolNames, "replay_governance_outbox_event")
	assert.Contains(t, toolNames, "get_governance_outbox_stats")
	assert.Contains(t, toolNames, "force_release_source_sync_lease")

	resources, err := session.ListResources(ctx, nil)
	require.NoError(t, err)

	resourceURIs := make([]string, 0, len(resources.Resources))
	for _, resource := range resources.Resources {
		resourceURIs = append(resourceURIs, resource.URI)
	}

	assert.Contains(t, resourceURIs, catalogSourcesURI)
	assert.Contains(t, resourceURIs, catalogTermsURI)
	assert.Contains(t, resourceURIs, catalogTagsURI)
	assert.Contains(t, resourceURIs, governanceOutboxURI)
	assert.Contains(t, resourceURIs, governanceOutboxStatsURI)

	templates, err := session.ListResourceTemplates(ctx, nil)
	require.NoError(t, err)

	templateURIs := make([]string, 0, len(templates.ResourceTemplates))
	for _, template := range templates.ResourceTemplates {
		templateURIs = append(templateURIs, template.URITemplate)
	}

	assert.Contains(t, templateURIs, "datamap://sources/{source_id}/schema")
	assert.Contains(t, templateURIs, "datamap://columns/{column_id}")
}

func TestServer_HTTPHandler_ListsTools(t *testing.T) {
	server := New(&Dependencies{})
	session := connectHTTPTestClient(t, server)

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	toolNames := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	assert.Contains(t, toolNames, "list_sources")
	assert.Contains(t, toolNames, "trigger_source_sync")
	assert.Contains(t, toolNames, "force_release_source_sync_lease")
}

func TestServer_PublishMutationAuditEvent(t *testing.T) {
	publisher := &fakeGovernancePublisher{enabled: true}
	server := New(&Dependencies{GovernancePublisher: publisher})

	ctx := withMutationAudit(context.Background(), "create_tag")
	err := server.publishMutationAuditEvent(ctx, "create_tag", "tag", "tag-123", map[string]interface{}{
		"name": "PII",
	})

	require.NoError(t, err)
	assert.Equal(t, "mcp.governance.action", publisher.event.EventType)
	assert.Equal(t, "tag", publisher.event.ResourceType)
	assert.Equal(t, "tag-123", publisher.event.ResourceID)
	assert.Equal(t, "create_tag", publisher.event.Payload["action"])
	assert.Equal(t, "create_tag", publisher.event.Payload["tool_name"])

	auditMeta := service.GovernanceAuditMetaFromContext(publisher.ctx)
	assert.Equal(t, "mcp", auditMeta.Origin)
	assert.Equal(t, "create_tag", auditMeta.Operation)
	assert.Equal(t, "mcp:datamap", auditMeta.ActorID)
	assert.NotEmpty(t, auditMeta.TraceID)
}

func TestServer_PublishMutationAuditEvent_PropagatesError(t *testing.T) {
	publisher := &fakeGovernancePublisher{
		enabled: true,
		err:     errors.New("publish failed"),
	}
	server := New(&Dependencies{GovernancePublisher: publisher})

	err := server.publishMutationAuditEvent(withMutationAudit(context.Background(), "create_term"), "create_term", "business_term", "term-1", nil)

	require.Error(t, err)
	assert.Equal(t, "publish failed", err.Error())
}

func TestServer_PublishMutationAuditEvent_NoOpWhenPublisherDisabled(t *testing.T) {
	publisher := &fakeGovernancePublisher{enabled: false}
	server := New(&Dependencies{GovernancePublisher: publisher})

	err := server.publishMutationAuditEvent(context.Background(), "create_tag", "tag", "tag-1", nil)

	require.NoError(t, err)
	assert.Empty(t, publisher.event.EventType)
	assert.Nil(t, publisher.ctx)
}

func TestServer_AssignTagsToColumn_RejectsEmptyTagIDs(t *testing.T) {
	server := New(&Dependencies{})
	session := connectTestClient(t, server)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "assign_tags_to_column",
		Arguments: map[string]any{
			"column_id": "col-1",
			"tag_ids":   []string{" ", ""},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "tag_ids is required")
}

func TestServer_GovernanceOutboxToolsAndResources_RequirePublisher(t *testing.T) {
	server := New(&Dependencies{})
	session := connectTestClient(t, server)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_governance_outbox_stats",
		Arguments: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "governance outbox is not enabled")
}

func TestServer_ReplayGovernanceOutboxEvent_ForwardsToPublisher(t *testing.T) {
	publisher := &fakeGovernancePublisher{
		enabled: true,
		stats: &store.GovernanceOutboxStatsRow{
			PendingCount:    1,
			DeliveredCount:  2,
			DeadLetterCount: 3,
			RetryableCount:  1,
		},
		outbox: []*store.GovernanceOutboxEventRow{
			{ID: "outbox-1", EventID: "evt-1", Status: "dead_letter"},
		},
	}
	server := New(&Dependencies{GovernancePublisher: publisher})
	session := connectTestClient(t, server)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "replay_governance_outbox_event",
		Arguments: map[string]any{
			"outbox_id": " outbox-1 ",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Equal(t, "outbox-1", publisher.replayedID)

	resources, err := session.ListResources(context.Background(), nil)
	require.NoError(t, err)
	resourceURIs := make([]string, 0, len(resources.Resources))
	for _, resource := range resources.Resources {
		resourceURIs = append(resourceURIs, resource.URI)
	}
	assert.Contains(t, resourceURIs, governanceOutboxURI)
	assert.Contains(t, resourceURIs, governanceOutboxStatsURI)
}

func TestServer_GovernanceOutboxResources_ReturnJSONContent(t *testing.T) {
	publisher := &fakeGovernancePublisher{
		enabled: true,
		outbox: []*store.GovernanceOutboxEventRow{
			{ID: "outbox-1", EventID: "evt-1", Status: "dead_letter", AttemptCount: 3},
		},
		stats: &store.GovernanceOutboxStatsRow{
			PendingCount:    1,
			DeliveredCount:  2,
			DeadLetterCount: 3,
			RetryableCount:  1,
		},
	}
	server := New(&Dependencies{GovernancePublisher: publisher})
	session := connectTestClient(t, server)

	outboxRes, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: governanceOutboxURI})
	require.NoError(t, err)
	require.Len(t, outboxRes.Contents, 1)
	var outbox []*store.GovernanceOutboxEventRow
	require.NoError(t, json.Unmarshal([]byte(outboxRes.Contents[0].Text), &outbox))
	require.Len(t, outbox, 1)
	assert.Equal(t, "outbox-1", outbox[0].ID)
	assert.Equal(t, "dead_letter", outbox[0].Status)

	statsRes, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: governanceOutboxStatsURI})
	require.NoError(t, err)
	require.Len(t, statsRes.Contents, 1)
	var stats store.GovernanceOutboxStatsRow
	require.NoError(t, json.Unmarshal([]byte(statsRes.Contents[0].Text), &stats))
	assert.EqualValues(t, 3, stats.DeadLetterCount)
	assert.EqualValues(t, 2, stats.DeliveredCount)
}

func TestServer_GovernanceOutboxReadResource_RequiresPublisher(t *testing.T) {
	server := New(&Dependencies{})
	session := connectTestClient(t, server)

	_, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: governanceOutboxURI})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "governance outbox is not enabled")
}

func TestServer_CoreTools_RequireConfiguredServices(t *testing.T) {
	server := New(&Dependencies{})
	session := connectTestClient(t, server)

	tests := []struct {
		name       string
		tool       string
		args       map[string]any
		wantErrMsg string
	}{
		{
			name:       "list sources",
			tool:       "list_sources",
			args:       map[string]any{},
			wantErrMsg: "source service is not configured",
		},
		{
			name:       "search columns",
			tool:       "search_columns",
			args:       map[string]any{"query": "user"},
			wantErrMsg: "metadata service is not configured",
		},
		{
			name:       "list terms",
			tool:       "list_terms",
			args:       map[string]any{},
			wantErrMsg: "term service is not configured",
		},
		{
			name:       "list tags",
			tool:       "list_tags",
			args:       map[string]any{},
			wantErrMsg: "tag service is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      tt.tool,
				Arguments: tt.args,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, result.IsError)
			require.Len(t, result.Content, 1)
			text, ok := result.Content[0].(*mcp.TextContent)
			require.True(t, ok)
			assert.Contains(t, text.Text, tt.wantErrMsg)
		})
	}
}

func TestServer_CoreResources_RequireConfiguredServices(t *testing.T) {
	server := New(&Dependencies{})
	session := connectTestClient(t, server)

	tests := []struct {
		name       string
		uri        string
		wantErrMsg string
	}{
		{
			name:       "sources",
			uri:        catalogSourcesURI,
			wantErrMsg: "source service is not configured",
		},
		{
			name:       "terms",
			uri:        catalogTermsURI,
			wantErrMsg: "term service is not configured",
		},
		{
			name:       "tags",
			uri:        catalogTagsURI,
			wantErrMsg: "tag service is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: tt.uri})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestServer_TemplateResources_RequireMetadataService(t *testing.T) {
	server := New(&Dependencies{})
	session := connectTestClient(t, server)

	for _, uri := range []string{
		"datamap://sources/source-1/schema",
		"datamap://columns/column-1",
	} {
		_, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata service is not configured")
	}
}

func TestServer_ReplayGovernanceOutboxEvent_PropagatesPublisherError(t *testing.T) {
	publisher := &fakeGovernancePublisher{
		enabled:     true,
		replayError: errors.New("outbox replay failed"),
	}
	server := New(&Dependencies{GovernancePublisher: publisher})
	session := connectTestClient(t, server)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "replay_governance_outbox_event",
		Arguments: map[string]any{
			"outbox_id": "outbox-1",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "outbox replay failed")
}

func TestServer_GovernanceOutboxTools_RejectPublisherWithoutStore(t *testing.T) {
	publisher := service.NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         "http://example.invalid",
		IntegrationToken: "integration-token",
		SourceSystem:     "cornerstone",
		Timeout:          time.Second,
	}, zap.NewNop())

	server := New(&Dependencies{GovernancePublisher: publisher})
	session := connectTestClient(t, server)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_governance_outbox_stats",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "governance outbox store is not configured")
}

func TestServer_ForceReleaseSourceSyncLease_ReleasesStaleLease(t *testing.T) {
	sourceService, st, sourceID := newSQLiteSourceServiceForMCPTest(t)
	server := New(&Dependencies{SourceService: sourceService})
	session := connectTestClient(t, server)

	now := time.Now().UTC()
	acquired, err := st.TryAcquireSyncLease(
		context.Background(),
		sourceID,
		"owner-a",
		now.Format(time.RFC3339Nano),
		now.Add(time.Minute).Format(time.RFC3339Nano),
	)
	require.NoError(t, err)
	require.True(t, acquired)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "force_release_source_sync_lease",
		Arguments: map[string]any{
			"source_id":     sourceID,
			"stale_seconds": 3600,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	time.Sleep(1100 * time.Millisecond)

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "force_release_source_sync_lease",
		Arguments: map[string]any{
			"source_id":     sourceID,
			"stale_seconds": 1,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	lease, err := st.GetSyncLease(context.Background(), sourceID)
	require.NoError(t, err)
	assert.Nil(t, lease)
}

func TestServer_ForceReleaseSourceSyncLease_RejectsBlankSourceID(t *testing.T) {
	sourceService, _, _ := newSQLiteSourceServiceForMCPTest(t)
	server := New(&Dependencies{SourceService: sourceService})
	session := connectTestClient(t, server)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "force_release_source_sync_lease",
		Arguments: map[string]any{
			"source_id": "   ",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "source id is required")
}

func TestParseTemplateURI(t *testing.T) {
	id, err := parseTemplateURI("datamap://sources/source-1/schema", "sources", "schema")
	require.NoError(t, err)
	assert.Equal(t, "source-1", id)

	_, err = parseTemplateURI("datamap://sources/source-1/unknown", "sources", "schema")
	require.Error(t, err)
}

func TestNormalizeHelpers(t *testing.T) {
	searchLimit := searchColumnsInput{Limit: -1}.normalizedLimit()
	assert.Equal(t, 20, searchLimit)
	assert.Equal(t, 100, searchColumnsInput{Limit: 1000}.normalizedLimit())
	assert.Equal(t, 20, listSchemaChangesInput{Limit: 0}.normalizedLimit())
	assert.Equal(t, 100, listSchemaChangesInput{Limit: 200}.normalizedLimit())

	normalizedTags := normalizeStringSlice([]string{" tag-a ", "", "   ", "tag-b"})
	assert.Equal(t, []string{"tag-a", "tag-b"}, normalizedTags)

	blank := "   "
	assert.Nil(t, normalizedOptionalString(&blank))
	assert.Equal(t, "", normalizedOptionalStringValue(&blank))

	value := "  term-1  "
	normalized := normalizedOptionalString(&value)
	require.NotNil(t, normalized)
	assert.Equal(t, "term-1", *normalized)
	assert.Equal(t, "term-1", normalizedOptionalStringValue(&value))
}
