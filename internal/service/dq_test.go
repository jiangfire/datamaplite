package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiangfire/datamaplite/internal/config"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDQService_ValidateRuleConfig_CustomSQLRejectsDangerousQuery(t *testing.T) {
	service := NewDQService(new(MockStore), nil)

	err := service.validateRuleConfig(model.DQRuleTypeCustomSQL, map[string]interface{}{
		"sql": "SELECT * FROM users WHERE name = xp_cmdshell('dir')",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid custom_sql rule")
}

func TestDQService_ValidateRuleConfig_CustomSQLAllowsSelect(t *testing.T) {
	service := NewDQService(new(MockStore), nil)

	err := service.validateRuleConfig(model.DQRuleTypeCustomSQL, map[string]interface{}{
		"sql": "SELECT id, email FROM users WHERE email IS NOT NULL",
	})

	require.NoError(t, err)
}

func TestDQService_ExecuteRule_CustomSQLRejectsStoredDangerousQuery(t *testing.T) {
	service := NewDQService(new(MockStore), nil)

	_, err := service.executeRule(context.Background(), &store.DQRuleRow{
		ID:         "rule-1",
		RuleType:   string(model.DQRuleTypeCustomSQL),
		RuleConfig: `{"sql":"SELECT * FROM users WHERE name = xp_cmdshell('dir')"}`,
	}, "batch-1", 5)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid custom_sql rule")
}

func TestDQService_CheckRules_ReturnsErrorWhenResultPersistenceFails(t *testing.T) {
	ctx := context.Background()
	mockStore := new(MockStore)
	service := NewDQService(mockStore, nil)

	mockStore.
		On("ListDQRules", ctx, mock.MatchedBy(func(filter *store.DQRuleFilter) bool {
			return filter != nil && filter.IsActive != nil && *filter.IsActive
		})).
		Return([]*store.DQRuleRow{
			{
				ID:         "rule-persist-fail",
				Name:       "dangerous custom sql",
				RuleType:   string(model.DQRuleTypeCustomSQL),
				RuleConfig: `{"sql":"SELECT * FROM users WHERE name = xp_cmdshell('dir')"}`,
				Severity:   string(model.DQSeverityError),
				IsActive:   true,
			},
		}, nil).
		Once()

	mockStore.
		On("CreateDQResult", ctx, mock.MatchedBy(func(result *store.DQResultCreate) bool {
			return result.RuleID == "rule-persist-fail" &&
				result.Status == string(model.DQResultStatusError) &&
				result.ErrorMessage != nil &&
				strings.Contains(*result.ErrorMessage, "invalid custom_sql rule")
		})).
		Return(errors.New("write failed")).
		Once()

	resp, err := service.CheckRules(ctx, &model.DQCheckRequest{CheckAll: true})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to persist dq result for rule rule-persist-fail")
	assert.Contains(t, err.Error(), "write failed")
	mockStore.AssertNotCalled(t, "ListDQResults", mock.Anything, mock.Anything)
}

func TestDQService_CheckRules_ReturnsErrorWhenPersistedResultCannotBeLoaded(t *testing.T) {
	ctx := context.Background()
	mockStore := new(MockStore)
	service := NewDQService(mockStore, nil)

	mockStore.
		On("ListDQRules", ctx, mock.MatchedBy(func(filter *store.DQRuleFilter) bool {
			return filter != nil && filter.IsActive != nil && *filter.IsActive
		})).
		Return([]*store.DQRuleRow{
			{
				ID:         "rule-readback-fail",
				Name:       "dangerous custom sql",
				RuleType:   string(model.DQRuleTypeCustomSQL),
				RuleConfig: `{"sql":"SELECT * FROM users WHERE name = xp_cmdshell('dir')"}`,
				Severity:   string(model.DQSeverityError),
				IsActive:   true,
			},
		}, nil).
		Once()

	mockStore.
		On("CreateDQResult", ctx, mock.AnythingOfType("*store.DQResultCreate")).
		Return(nil).
		Once()
	mockStore.
		On("ListDQResults", ctx, mock.MatchedBy(func(filter *store.DQResultFilter) bool {
			return filter != nil &&
				filter.RuleID != nil && *filter.RuleID == "rule-readback-fail" &&
				filter.BatchID != nil && *filter.BatchID != "" &&
				filter.Limit == 1
		})).
		Return(nil, errors.New("read failed")).
		Once()

	resp, err := service.CheckRules(ctx, &model.DQCheckRequest{CheckAll: true})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to load persisted dq result for rule rule-readback-fail")
	assert.Contains(t, err.Error(), "list dq results failed: read failed")
}

func TestDQService_CheckRules_WithRuleIDsReturnsErrorForMissingRule(t *testing.T) {
	ctx := context.Background()
	mockStore := new(MockStore)
	service := NewDQService(mockStore, nil)

	mockStore.
		On("GetDQRule", ctx, "missing-rule").
		Return(nil, errors.New("dq rule not found: missing-rule")).
		Once()

	resp, err := service.CheckRules(ctx, &model.DQCheckRequest{
		RuleIDs: []string{"missing-rule"},
	})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to load dq rule missing-rule")
	assert.Contains(t, err.Error(), "dq rule not found: missing-rule")
	mockStore.AssertNotCalled(t, "CreateDQResult", mock.Anything, mock.Anything)
}

func TestDQService_CheckRules_WithRuleIDsExecutesInactiveRule(t *testing.T) {
	ctx := context.Background()
	mockStore := new(MockStore)
	service := NewDQService(mockStore, nil)

	mockStore.
		On("GetDQRule", ctx, "inactive-rule").
		Return(&store.DQRuleRow{
			ID:         "inactive-rule",
			Name:       "inactive custom sql",
			RuleType:   string(model.DQRuleTypeCustomSQL),
			RuleConfig: `{"sql":"SELECT * FROM users WHERE name = xp_cmdshell('dir')"}`,
			Severity:   string(model.DQSeverityError),
			IsActive:   false,
		}, nil).
		Once()
	mockStore.
		On("CreateDQResult", ctx, mock.MatchedBy(func(result *store.DQResultCreate) bool {
			return result.RuleID == "inactive-rule" &&
				result.Status == string(model.DQResultStatusError)
		})).
		Return(nil).
		Once()
	mockStore.
		On("ListDQResults", ctx, mock.MatchedBy(func(filter *store.DQResultFilter) bool {
			return filter != nil &&
				filter.RuleID != nil && *filter.RuleID == "inactive-rule" &&
				filter.BatchID != nil && *filter.BatchID != "" &&
				filter.Limit == 1
		})).
		Return([]*store.DQResultRow{
			{
				ID:           "result-inactive-rule",
				RuleID:       "inactive-rule",
				CheckBatchID: "batch-inactive-rule",
				Status:       string(model.DQResultStatusError),
				TotalRows:    0,
				FailedRows:   0,
				PassRate:     0,
				SampleErrors: `[]`,
				CheckedAt:    "2026-03-28T12:00:00Z",
			},
		}, nil).
		Once()

	resp, err := service.CheckRules(ctx, &model.DQCheckRequest{
		RuleIDs: []string{"inactive-rule"},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, resp.TotalRules)
	assert.Equal(t, 0, resp.PassedRules)
	assert.Equal(t, 1, resp.FailedRules)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "inactive-rule", resp.Results[0].RuleID)
	assert.Equal(t, model.DQResultStatusError, resp.Results[0].Status)
}

func TestDQService_PublishDQFailureEvent_ErrorResultPublishesGovernanceEvent(t *testing.T) {
	var captured GovernanceEvent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	mockStore := new(MockStore)
	service := NewDQService(mockStore, nil)
	service.SetGovernanceEventService(NewGovernanceEventService(config.GovernanceConfig{
		Enabled:          true,
		Endpoint:         server.URL,
		IntegrationToken: "integration-token",
		SourceSystem:     "fuckcmdb",
		Timeout:          time.Second,
	}, zap.NewNop()))

	sourceID := "src-1"
	objectID := "obj-1"
	columnID := "col-1"
	mockStore.On("GetDataSource", mock.Anything, sourceID).Return(&store.DataSourceRow{
		ID:   sourceID,
		Name: "mysql-prod",
	}, nil).Once()
	mockStore.On("GetSchemaObject", mock.Anything, objectID).Return(&store.SchemaObjectRow{
		ID:       objectID,
		SourceID: sourceID,
		Name:     "users",
	}, nil).Once()
	mockStore.On("GetColumn", mock.Anything, columnID).Return(&store.ColumnRow{
		ID:       columnID,
		ObjectID: objectID,
		Name:     "email",
	}, nil).Once()

	errMsg := "dial tcp timeout"
	err := service.publishDQFailureEvent(context.Background(), &store.DQRuleRow{
		ID:       "rule-event-error",
		SourceID: &sourceID,
		ObjectID: &objectID,
		ColumnID: &columnID,
		Name:     "email not null",
		RuleType: string(model.DQRuleTypeNotNull),
		Severity: string(model.DQSeverityError),
	}, &model.DQResult{
		ID:           "result-event-error",
		CheckBatchID: "batch-event-error",
		Status:       model.DQResultStatusError,
		CheckedAt:    time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
		ErrorMessage: &errMsg,
	})

	require.NoError(t, err)
	assert.Equal(t, "dq.rule.failed", captured.EventType)
	assert.Equal(t, "result-event-error", captured.ResourceID)
	assert.Equal(t, "batch-event-error", captured.TraceID)
	assert.Equal(t, "error", captured.Payload["status"])
	assert.Equal(t, errMsg, captured.Payload["error_message"])
	assert.Equal(t, "mysql-prod", captured.Payload["source_name"])
	assert.Equal(t, "users", captured.Payload["object_name"])
	assert.Equal(t, "email", captured.Payload["column_name"])
	assert.Equal(t, "email", captured.Payload["display_name"])
	assert.Equal(t, "high", captured.Payload["priority"])
}
