package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDQService is a mock for DQService
type MockDQService struct {
	mock.Mock
}

func (m *MockDQService) CreateRule(ctx context.Context, req *model.DQRuleRequest) (*model.DQRule, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DQRule), args.Error(1)
}

func (m *MockDQService) ListRules(ctx context.Context, filter *model.DQRuleFilter) ([]*model.DQRule, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.DQRule), args.Error(1)
}

func (m *MockDQService) GetRule(ctx context.Context, id string) (*model.DQRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DQRule), args.Error(1)
}

func (m *MockDQService) UpdateRule(ctx context.Context, id string, req *model.DQRuleRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *MockDQService) DeleteRule(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDQService) CheckRules(ctx context.Context, req *model.DQCheckRequest) (*model.DQCheckResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DQCheckResponse), args.Error(1)
}

func (m *MockDQService) GetResults(ctx context.Context, ruleID *string, batchID *string, limit int) ([]*model.DQResult, error) {
	args := m.Called(ctx, ruleID, batchID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.DQResult), args.Error(1)
}

func (m *MockDQService) GetStats(ctx context.Context) (*model.DQStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DQStats), args.Error(1)
}

func setupDQHandlerTest() (*gin.Engine, *MockDQService, *DQHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mockSvc := new(MockDQService)
	handler := NewDQHandler(nil)
	handler.dqService = mockSvc
	return router, mockSvc, handler
}

func TestDQHandler_ListRules_Success(t *testing.T) {
	router, mockSvc, handler := setupDQHandlerTest()

	rules := []*model.DQRule{
		{ID: "rule-1", Name: "not-null-check"},
	}

	mockSvc.On("ListRules", mock.Anything, mock.Anything).Return(rules, nil)

	router.GET("/dq/rules", handler.ListRules)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/dq/rules", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestDQHandler_GetRule_NotFound(t *testing.T) {
	router, mockSvc, handler := setupDQHandlerTest()

	mockSvc.On("GetRule", mock.Anything, "rule-999").Return(nil, errors.New("not found"))

	router.GET("/dq/rules/:id", handler.GetRule)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/dq/rules/rule-999", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusNotFound, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestDQHandler_GetStats_Success(t *testing.T) {
	router, mockSvc, handler := setupDQHandlerTest()

	stats := &model.DQStats{
		TotalRules:      10,
		ActiveRules:     8,
		TotalChecks:     100,
		FailedChecks:    5,
		OverallPassRate: 95.0,
	}

	mockSvc.On("GetStats", mock.Anything).Return(stats, nil)

	router.GET("/dq/stats", handler.GetStats)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/dq/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	mockSvc.AssertExpectations(t)
}
