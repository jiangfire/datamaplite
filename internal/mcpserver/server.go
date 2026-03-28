package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	catalogSourcesURI        = "datamap://catalog/sources"
	catalogTermsURI          = "datamap://catalog/terms"
	catalogTagsURI           = "datamap://catalog/tags"
	governanceOutboxURI      = "datamap://governance/outbox"
	governanceOutboxStatsURI = "datamap://governance/outbox/stats"
)

type Dependencies struct {
	SourceService       *service.SourceService
	MetadataService     *service.MetadataService
	TermService         *service.TermService
	TagService          *service.TagService
	GovernancePublisher governancePublisher
}

type Server struct {
	deps        *Dependencies
	mcp         *mcp.Server
	httpHandler http.Handler
}

type governancePublisher interface {
	Enabled() bool
	Publish(ctx context.Context, event service.GovernanceEvent) error
	ListOutbox(ctx context.Context, limit int) ([]*store.GovernanceOutboxEventRow, error)
	GetOutboxStats(ctx context.Context) (*store.GovernanceOutboxStatsRow, error)
	ReplayOutboxEvent(ctx context.Context, id string) error
}

func New(deps *Dependencies) *Server {
	server := &Server{
		deps: deps,
		mcp: mcp.NewServer(&mcp.Implementation{
			Name:    "datamap-mcp",
			Version: "0.1.0",
		}, nil),
	}

	server.registerTools()
	server.registerResources()
	server.httpHandler = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server.mcp
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: 10 * time.Minute,
	})
	return server
}

func (s *Server) HTTPHandler() http.Handler {
	if s == nil {
		return nil
	}
	return s.httpHandler
}

func (s *Server) requireSourceService() (*service.SourceService, error) {
	if s == nil || s.deps == nil || s.deps.SourceService == nil {
		return nil, fmt.Errorf("source service is not configured")
	}
	return s.deps.SourceService, nil
}

func (s *Server) requireMetadataService() (*service.MetadataService, error) {
	if s == nil || s.deps == nil || s.deps.MetadataService == nil {
		return nil, fmt.Errorf("metadata service is not configured")
	}
	return s.deps.MetadataService, nil
}

func (s *Server) requireTermService() (*service.TermService, error) {
	if s == nil || s.deps == nil || s.deps.TermService == nil {
		return nil, fmt.Errorf("term service is not configured")
	}
	return s.deps.TermService, nil
}

func (s *Server) requireTagService() (*service.TagService, error) {
	if s == nil || s.deps == nil || s.deps.TagService == nil {
		return nil, fmt.Errorf("tag service is not configured")
	}
	return s.deps.TagService, nil
}

func (s *Server) requireGovernancePublisher() (governancePublisher, error) {
	if s == nil || s.deps == nil || s.deps.GovernancePublisher == nil || !s.deps.GovernancePublisher.Enabled() {
		return nil, fmt.Errorf("governance outbox is not enabled")
	}
	return s.deps.GovernancePublisher, nil
}

func (s *Server) registerTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "list_sources",
		Title:       "List Sources",
		Description: "List all registered data sources in DataMap.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listSourcesOutput, error) {
		sourceService, err := s.requireSourceService()
		if err != nil {
			return nil, listSourcesOutput{}, err
		}
		sources, err := sourceService.ListSources(ctx)
		if err != nil {
			return nil, listSourcesOutput{}, err
		}
		return nil, listSourcesOutput{Sources: sources}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "search_columns",
		Title:       "Search Columns",
		Description: "Search columns by keyword across all indexed metadata.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in searchColumnsInput) (*mcp.CallToolResult, searchColumnsOutput, error) {
		metadataService, err := s.requireMetadataService()
		if err != nil {
			return nil, searchColumnsOutput{}, err
		}
		results, err := metadataService.SearchColumns(ctx, strings.TrimSpace(in.Query), in.normalizedLimit())
		if err != nil {
			return nil, searchColumnsOutput{}, err
		}
		return nil, searchColumnsOutput{Results: results}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "get_source_schema",
		Title:       "Get Source Schema",
		Description: "Return the schema tree for a specific source.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sourceIDInput) (*mcp.CallToolResult, schemaTreeOutput, error) {
		metadataService, err := s.requireMetadataService()
		if err != nil {
			return nil, schemaTreeOutput{}, err
		}
		tree, err := metadataService.GetSchemaTree(ctx, strings.TrimSpace(in.SourceID))
		if err != nil {
			return nil, schemaTreeOutput{}, err
		}
		return nil, schemaTreeOutput{Schema: tree}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "get_column_detail",
		Title:       "Get Column Detail",
		Description: "Return detailed metadata for a specific column.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in columnIDInput) (*mcp.CallToolResult, columnDetailOutput, error) {
		metadataService, err := s.requireMetadataService()
		if err != nil {
			return nil, columnDetailOutput{}, err
		}
		detail, err := metadataService.GetColumnDetail(ctx, strings.TrimSpace(in.ColumnID))
		if err != nil {
			return nil, columnDetailOutput{}, err
		}
		return nil, columnDetailOutput{Column: detail}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "list_terms",
		Title:       "List Terms",
		Description: "List business terms, optionally filtered by category.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listTermsInput) (*mcp.CallToolResult, listTermsOutput, error) {
		termService, err := s.requireTermService()
		if err != nil {
			return nil, listTermsOutput{}, err
		}
		terms, err := termService.ListTerms(ctx, strings.TrimSpace(in.Category))
		if err != nil {
			return nil, listTermsOutput{}, err
		}
		return nil, listTermsOutput{Terms: terms}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "list_tags",
		Title:       "List Tags",
		Description: "List all governance tags.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listTagsOutput, error) {
		tagService, err := s.requireTagService()
		if err != nil {
			return nil, listTagsOutput{}, err
		}
		tags, err := tagService.ListTags(ctx)
		if err != nil {
			return nil, listTagsOutput{}, err
		}
		return nil, listTagsOutput{Tags: tags}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "list_schema_changes",
		Title:       "List Schema Changes",
		Description: "List recent schema changes for a source.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listSchemaChangesInput) (*mcp.CallToolResult, listSchemaChangesOutput, error) {
		metadataService, err := s.requireMetadataService()
		if err != nil {
			return nil, listSchemaChangesOutput{}, err
		}
		changes, err := metadataService.ListSchemaChanges(ctx, strings.TrimSpace(in.SourceID), in.normalizedLimit())
		if err != nil {
			return nil, listSchemaChangesOutput{}, err
		}
		return nil, listSchemaChangesOutput{Changes: changes}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "assign_term_to_column",
		Title:       "Assign Term To Column",
		Description: "Bind or unbind a business term to a column for metadata governance.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in assignTermInput) (*mcp.CallToolResult, mutationOutput, error) {
		columnID := strings.TrimSpace(in.ColumnID)
		if columnID == "" {
			return nil, mutationOutput{}, fmt.Errorf("column_id is required")
		}
		termService, err := s.requireTermService()
		if err != nil {
			return nil, mutationOutput{}, err
		}
		ctx = withMutationAudit(ctx, "assign_term_to_column")
		req := &model.AssignTermRequest{TermID: normalizedOptionalString(in.TermID)}
		if err := termService.AssignTermToColumn(ctx, columnID, req); err != nil {
			return nil, mutationOutput{}, err
		}
		_ = s.publishMutationAuditEvent(ctx, "assign_term_to_column", "column", columnID, map[string]interface{}{
			"column_id": columnID,
			"term_id":   normalizedOptionalStringValue(in.TermID),
		})
		return nil, mutationOutput{
			Status:  "ok",
			Message: "term assignment updated",
		}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "assign_tags_to_column",
		Title:       "Assign Tags To Column",
		Description: "Attach one or more existing governance tags to a column.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in assignTagsInput) (*mcp.CallToolResult, mutationOutput, error) {
		ctx = withMutationAudit(ctx, "assign_tags_to_column")
		columnID := strings.TrimSpace(in.ColumnID)
		if columnID == "" {
			return nil, mutationOutput{}, fmt.Errorf("column_id is required")
		}
		tagIDs := normalizeStringSlice(in.TagIDs)
		if len(tagIDs) == 0 {
			return nil, mutationOutput{}, fmt.Errorf("tag_ids is required")
		}
		tagService, err := s.requireTagService()
		if err != nil {
			return nil, mutationOutput{}, err
		}
		for _, tagID := range tagIDs {
			if err := tagService.AddTagToColumn(ctx, columnID, tagID); err != nil {
				return nil, mutationOutput{}, err
			}
		}
		_ = s.publishMutationAuditEvent(ctx, "assign_tags_to_column", "column", columnID, map[string]interface{}{
			"column_id": columnID,
			"tag_ids":   tagIDs,
		})
		return nil, mutationOutput{
			Status:  "ok",
			Message: fmt.Sprintf("assigned %d tag(s)", len(tagIDs)),
		}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "create_column_mapping",
		Title:       "Create Column Mapping",
		Description: "Create a semantic mapping relationship between two columns.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createMappingInput) (*mcp.CallToolResult, mutationOutput, error) {
		ctx = withMutationAudit(ctx, "create_column_mapping")
		sourceColumnID := strings.TrimSpace(in.SourceColumnID)
		targetColumnID := strings.TrimSpace(in.TargetColumnID)
		mappingType := strings.TrimSpace(in.MappingType)
		if sourceColumnID == "" {
			return nil, mutationOutput{}, fmt.Errorf("source_column_id is required")
		}
		if targetColumnID == "" {
			return nil, mutationOutput{}, fmt.Errorf("target_column_id is required")
		}
		if mappingType == "" {
			return nil, mutationOutput{}, fmt.Errorf("mapping_type is required")
		}
		metadataService, err := s.requireMetadataService()
		if err != nil {
			return nil, mutationOutput{}, err
		}
		req := &model.ColumnMappingRequest{
			SourceColumnID: sourceColumnID,
			TargetColumnID: targetColumnID,
			MappingType:    mappingType,
			Confidence:     in.Confidence,
		}
		if err := metadataService.CreateColumnMapping(ctx, req); err != nil {
			return nil, mutationOutput{}, err
		}
		_ = s.publishMutationAuditEvent(ctx, "create_column_mapping", "column_mapping", "", map[string]interface{}{
			"source_column_id": req.SourceColumnID,
			"target_column_id": req.TargetColumnID,
			"mapping_type":     req.MappingType,
			"confidence":       req.Confidence,
		})
		return nil, mutationOutput{
			Status:  "ok",
			Message: "column mapping created",
		}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "trigger_source_sync",
		Title:       "Trigger Source Sync",
		Description: "Trigger an asynchronous metadata sync for a source.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sourceIDInput) (*mcp.CallToolResult, mutationOutput, error) {
		sourceID := strings.TrimSpace(in.SourceID)
		if sourceID == "" {
			return nil, mutationOutput{}, fmt.Errorf("source_id is required")
		}
		sourceService, err := s.requireSourceService()
		if err != nil {
			return nil, mutationOutput{}, err
		}
		ctx = withMutationAudit(ctx, "trigger_source_sync")
		if err := sourceService.TriggerSync(ctx, sourceID); err != nil {
			return nil, mutationOutput{}, err
		}
		_ = s.publishMutationAuditEvent(ctx, "trigger_source_sync", "source", sourceID, map[string]interface{}{
			"source_id": sourceID,
		})
		return nil, mutationOutput{
			Status:  "accepted",
			Message: "source sync triggered",
		}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "create_term",
		Title:       "Create Term",
		Description: "Create a new business term for governance taxonomy.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createTermInput) (*mcp.CallToolResult, createTermOutput, error) {
		termService, err := s.requireTermService()
		if err != nil {
			return nil, createTermOutput{}, err
		}
		ctx = withMutationAudit(ctx, "create_term")
		term, err := termService.CreateTerm(ctx, &model.BusinessTermRequest{
			Name:             strings.TrimSpace(in.Name),
			Description:      strings.TrimSpace(in.Description),
			Category:         strings.TrimSpace(in.Category),
			StandardCode:     strings.TrimSpace(in.StandardCode),
			Domain:           strings.TrimSpace(in.Domain),
			DataTypeStandard: strings.TrimSpace(in.DataTypeStandard),
			ValidationRule:   strings.TrimSpace(in.ValidationRule),
			Owner:            strings.TrimSpace(in.Owner),
			Status:           strings.TrimSpace(in.Status),
		})
		if err != nil {
			return nil, createTermOutput{}, err
		}
		_ = s.publishMutationAuditEvent(ctx, "create_term", "business_term", term.ID, map[string]interface{}{
			"term_id":       term.ID,
			"name":          term.Name,
			"category":      term.Category,
			"standard_code": term.StandardCode,
			"domain":        term.Domain,
			"status":        term.Status,
		})
		return nil, createTermOutput{Term: term}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "create_tag",
		Title:       "Create Tag",
		Description: "Create a new governance tag.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createTagInput) (*mcp.CallToolResult, createTagOutput, error) {
		tagService, err := s.requireTagService()
		if err != nil {
			return nil, createTagOutput{}, err
		}
		ctx = withMutationAudit(ctx, "create_tag")
		tag, err := tagService.CreateTag(ctx, &model.TagRequest{
			Name:        strings.TrimSpace(in.Name),
			Color:       strings.TrimSpace(in.Color),
			Description: strings.TrimSpace(in.Description),
		})
		if err != nil {
			return nil, createTagOutput{}, err
		}
		_ = s.publishMutationAuditEvent(ctx, "create_tag", "tag", tag.ID, map[string]interface{}{
			"tag_id": tag.ID,
			"name":   tag.Name,
			"color":  tag.Color,
		})
		return nil, createTagOutput{Tag: tag}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "replay_governance_outbox_event",
		Title:       "Replay Governance Outbox Event",
		Description: "Replay a failed or dead-letter governance outbox event by outbox ID.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in governanceOutboxReplayInput) (*mcp.CallToolResult, mutationOutput, error) {
		outboxID := strings.TrimSpace(in.OutboxID)
		if outboxID == "" {
			return nil, mutationOutput{}, fmt.Errorf("outbox_id is required")
		}
		publisher, err := s.requireGovernancePublisher()
		if err != nil {
			return nil, mutationOutput{}, err
		}
		if err := publisher.ReplayOutboxEvent(ctx, outboxID); err != nil {
			return nil, mutationOutput{}, err
		}
		return nil, mutationOutput{
			Status:  "accepted",
			Message: "governance outbox replay scheduled",
		}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "get_governance_outbox_stats",
		Title:       "Get Governance Outbox Stats",
		Description: "Return governance outbox counts for monitoring and operations.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, governanceOutboxStatsOutput, error) {
		publisher, err := s.requireGovernancePublisher()
		if err != nil {
			return nil, governanceOutboxStatsOutput{}, err
		}
		stats, err := publisher.GetOutboxStats(ctx)
		if err != nil {
			return nil, governanceOutboxStatsOutput{}, err
		}
		return nil, governanceOutboxStatsOutput{Stats: stats}, nil
	})

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "force_release_source_sync_lease",
		Title:       "Force Release Source Sync Lease",
		Description: "Force release a stale source sync lease left behind by a crashed or stuck instance.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in releaseSyncLeaseInput) (*mcp.CallToolResult, mutationOutput, error) {
		sourceService, err := s.requireSourceService()
		if err != nil {
			return nil, mutationOutput{}, err
		}
		staleAfter := time.Duration(in.StaleSeconds) * time.Second
		if err := sourceService.ForceReleaseStaleSyncLease(ctx, strings.TrimSpace(in.SourceID), staleAfter); err != nil {
			return nil, mutationOutput{}, err
		}
		return nil, mutationOutput{
			Status:  "ok",
			Message: "stale sync lease released",
		}, nil
	})
}

func (s *Server) registerResources() {
	s.mcp.AddResource(&mcp.Resource{
		Name:        "sources",
		Title:       "Data Sources",
		Description: "Catalog of all registered data sources.",
		URI:         catalogSourcesURI,
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		sourceService, err := s.requireSourceService()
		if err != nil {
			return nil, err
		}
		sources, err := sourceService.ListSources(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResourceResult(req.Params.URI, sources)
	})

	s.mcp.AddResource(&mcp.Resource{
		Name:        "terms",
		Title:       "Business Terms",
		Description: "Catalog of all business terms.",
		URI:         catalogTermsURI,
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		termService, err := s.requireTermService()
		if err != nil {
			return nil, err
		}
		terms, err := termService.ListTerms(ctx, "")
		if err != nil {
			return nil, err
		}
		return jsonResourceResult(req.Params.URI, terms)
	})

	s.mcp.AddResource(&mcp.Resource{
		Name:        "tags",
		Title:       "Governance Tags",
		Description: "Catalog of all tags.",
		URI:         catalogTagsURI,
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		tagService, err := s.requireTagService()
		if err != nil {
			return nil, err
		}
		tags, err := tagService.ListTags(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResourceResult(req.Params.URI, tags)
	})

	s.mcp.AddResource(&mcp.Resource{
		Name:        "governance-outbox",
		Title:       "Governance Outbox",
		Description: "Recent governance outbox events, including dead-letter entries.",
		URI:         governanceOutboxURI,
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		publisher, err := s.requireGovernancePublisher()
		if err != nil {
			return nil, err
		}
		outbox, err := publisher.ListOutbox(ctx, 50)
		if err != nil {
			return nil, err
		}
		return jsonResourceResult(req.Params.URI, outbox)
	})

	s.mcp.AddResource(&mcp.Resource{
		Name:        "governance-outbox-stats",
		Title:       "Governance Outbox Stats",
		Description: "Governance outbox aggregate counters for operations and monitoring.",
		URI:         governanceOutboxStatsURI,
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		publisher, err := s.requireGovernancePublisher()
		if err != nil {
			return nil, err
		}
		stats, err := publisher.GetOutboxStats(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResourceResult(req.Params.URI, stats)
	})

	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "source-schema",
		Title:       "Source Schema",
		Description: "Schema tree for a specific source.",
		URITemplate: "datamap://sources/{source_id}/schema",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		metadataService, err := s.requireMetadataService()
		if err != nil {
			return nil, err
		}
		sourceID, err := parseTemplateURI(req.Params.URI, "sources", "schema")
		if err != nil {
			return nil, err
		}
		tree, err := metadataService.GetSchemaTree(ctx, sourceID)
		if err != nil {
			return nil, err
		}
		return jsonResourceResult(req.Params.URI, tree)
	})

	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "column-detail",
		Title:       "Column Detail",
		Description: "Detailed metadata for a specific column.",
		URITemplate: "datamap://columns/{column_id}",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		metadataService, err := s.requireMetadataService()
		if err != nil {
			return nil, err
		}
		columnID, err := parseTemplateURI(req.Params.URI, "columns")
		if err != nil {
			return nil, err
		}
		detail, err := metadataService.GetColumnDetail(ctx, columnID)
		if err != nil {
			return nil, err
		}
		return jsonResourceResult(req.Params.URI, detail)
	})
}

func jsonResourceResult(uri string, value any) (*mcp.ReadResourceResult, error) {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal resource %s failed: %w", uri, err)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(body),
			},
		},
	}, nil
}

func parseTemplateURI(rawURI string, host string, suffix ...string) (string, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return "", mcp.ResourceNotFoundError(rawURI)
	}
	if parsed.Scheme != "datamap" || parsed.Host != host {
		return "", mcp.ResourceNotFoundError(rawURI)
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", mcp.ResourceNotFoundError(rawURI)
	}
	if len(parts) != len(suffix)+1 {
		return "", mcp.ResourceNotFoundError(rawURI)
	}
	for idx, expected := range suffix {
		if parts[idx+1] != expected {
			return "", mcp.ResourceNotFoundError(rawURI)
		}
	}
	return parts[0], nil
}

func normalizeStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizedOptionalStringValue(value *string) string {
	trimmed := normalizedOptionalString(value)
	if trimmed == nil {
		return ""
	}
	return *trimmed
}

func withMutationAudit(ctx context.Context, operation string) context.Context {
	return service.WithGovernanceAuditMeta(ctx, service.GovernanceAuditMeta{
		ActorID:   "mcp:datamap",
		TraceID:   "mcp_" + uuid.NewString(),
		Origin:    "mcp",
		Operation: operation,
	})
}

func (s *Server) publishMutationAuditEvent(ctx context.Context, action string, resourceType string, resourceID string, payload map[string]interface{}) error {
	if s == nil || s.deps == nil || s.deps.GovernancePublisher == nil || !s.deps.GovernancePublisher.Enabled() {
		return nil
	}

	if payload == nil {
		payload = map[string]interface{}{}
	}
	payload["action"] = action
	payload["tool_name"] = action
	payload["title"] = fmt.Sprintf("记录 MCP 治理操作：%s", action)
	payload["summary"] = fmt.Sprintf("MCP 执行治理写操作 [%s]", action)
	if _, exists := payload["display_name"]; !exists {
		if resourceID != "" {
			payload["display_name"] = resourceID
		} else {
			payload["display_name"] = action
		}
	}
	if _, exists := payload["priority"]; !exists {
		payload["priority"] = "low"
	}

	return s.deps.GovernancePublisher.Publish(ctx, service.GovernanceEvent{
		EventType:    "mcp.governance.action",
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Payload:      payload,
	})
}

type listSourcesOutput struct {
	Sources []*model.SourceListItem `json:"sources"`
}

type searchColumnsInput struct {
	Query string `json:"query" jsonschema:"Search keyword for column names or metadata"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of results to return, default 20, max 100"`
}

func (in searchColumnsInput) normalizedLimit() int {
	if in.Limit <= 0 {
		return 20
	}
	if in.Limit > 100 {
		return 100
	}
	return in.Limit
}

type searchColumnsOutput struct {
	Results []model.ColumnSearchResult `json:"results"`
}

type sourceIDInput struct {
	SourceID string `json:"source_id" jsonschema:"Data source ID"`
}

type schemaTreeOutput struct {
	Schema *model.SchemaTreeResponse `json:"schema"`
}

type columnIDInput struct {
	ColumnID string `json:"column_id" jsonschema:"Column ID"`
}

type columnDetailOutput struct {
	Column *model.ColumnDetailResponse `json:"column"`
}

type listTermsInput struct {
	Category string `json:"category,omitempty" jsonschema:"Optional term category filter"`
}

type listTermsOutput struct {
	Terms []*model.BusinessTermResponse `json:"terms"`
}

type listTagsOutput struct {
	Tags []*model.TagResponse `json:"tags"`
}

type listSchemaChangesInput struct {
	SourceID string `json:"source_id" jsonschema:"Data source ID"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Maximum number of changes to return, default 20, max 100"`
}

func (in listSchemaChangesInput) normalizedLimit() int {
	if in.Limit <= 0 {
		return 20
	}
	if in.Limit > 100 {
		return 100
	}
	return in.Limit
}

type listSchemaChangesOutput struct {
	Changes []*model.SchemaChangeResponse `json:"changes"`
}

type assignTermInput struct {
	ColumnID string  `json:"column_id" jsonschema:"Column ID"`
	TermID   *string `json:"term_id,omitempty" jsonschema:"Business term ID. Leave empty to unbind the term"`
}

type assignTagsInput struct {
	ColumnID string   `json:"column_id" jsonschema:"Column ID"`
	TagIDs   []string `json:"tag_ids" jsonschema:"List of existing tag IDs to attach to the column"`
}

type createMappingInput struct {
	SourceColumnID string  `json:"source_column_id" jsonschema:"Source column ID"`
	TargetColumnID string  `json:"target_column_id" jsonschema:"Target column ID"`
	MappingType    string  `json:"mapping_type" jsonschema:"Mapping type: alias, transform, derived, or synonym"`
	Confidence     float64 `json:"confidence,omitempty" jsonschema:"Confidence score between 0 and 1"`
}

type mutationOutput struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type createTermInput struct {
	Name             string `json:"name" jsonschema:"Business term name"`
	Description      string `json:"description,omitempty" jsonschema:"Business meaning of the term"`
	Category         string `json:"category,omitempty" jsonschema:"Optional term category"`
	StandardCode     string `json:"standard_code,omitempty" jsonschema:"Optional external or internal standard code"`
	Domain           string `json:"domain,omitempty" jsonschema:"Optional domain or data subject area"`
	DataTypeStandard string `json:"data_type_standard,omitempty" jsonschema:"Optional standard data type description"`
	ValidationRule   string `json:"validation_rule,omitempty" jsonschema:"Optional validation rule summary"`
	Owner            string `json:"owner,omitempty" jsonschema:"Optional owning team or person"`
	Status           string `json:"status,omitempty" jsonschema:"Optional term status, defaults to active"`
}

type createTermOutput struct {
	Term *model.BusinessTermResponse `json:"term"`
}

type createTagInput struct {
	Name        string `json:"name" jsonschema:"Tag name"`
	Color       string `json:"color" jsonschema:"Hex color like #2563eb"`
	Description string `json:"description,omitempty" jsonschema:"Optional tag description"`
}

type createTagOutput struct {
	Tag *model.TagResponse `json:"tag"`
}

type governanceOutboxReplayInput struct {
	OutboxID string `json:"outbox_id" jsonschema:"Governance outbox row ID"`
}

type governanceOutboxStatsOutput struct {
	Stats *store.GovernanceOutboxStatsRow `json:"stats"`
}

type releaseSyncLeaseInput struct {
	SourceID     string `json:"source_id" jsonschema:"Data source ID"`
	StaleSeconds int    `json:"stale_seconds,omitempty" jsonschema:"Optional stale threshold in seconds; defaults to service threshold when omitted"`
}
