package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockAlertService is a mock for AlertService
type MockAlertService struct {
	mock.Mock
}

func (m *MockAlertService) CreateAlertRule(ctx context.Context, req *model.AlertRuleRequest) (*model.AlertRuleResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertRuleResponse), args.Error(1)
}

func (m *MockAlertService) GetAlertRule(ctx context.Context, id string) (*model.AlertRuleResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertRuleResponse), args.Error(1)
}

func (m *MockAlertService) ListAlertRules(ctx context.Context, sourceID *string) ([]*model.AlertRuleResponse, error) {
	args := m.Called(ctx, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AlertRuleResponse), args.Error(1)
}

func (m *MockAlertService) UpdateAlertRule(ctx context.Context, id string, req *model.AlertRuleRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *MockAlertService) DeleteAlertRule(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockNotificationService is a mock for NotificationService
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*model.NotificationResponse, error) {
	args := m.Called(ctx, userID, unreadOnly, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.NotificationResponse), args.Error(1)
}

func (m *MockNotificationService) GetNotificationStats(ctx context.Context, userID string) (*model.NotificationStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.NotificationStats), args.Error(1)
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, userID string, id string) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockNotificationService) MarkManyAsRead(ctx context.Context, userID string, ids []string) error {
	args := m.Called(ctx, userID, ids)
	return args.Error(0)
}

func (m *MockNotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func setupAlertHandlerTest() (*gin.Engine, *MockAlertService, *MockNotificationService, *AlertHandler, *NotificationHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mockAlertSvc := new(MockAlertService)
	mockNotifSvc := new(MockNotificationService)
	logger := zap.NewNop()
	alertHandler := NewAlertHandler(nil, nil, logger)
	alertHandler.alertService = mockAlertSvc
	notifHandler := NewNotificationHandler(nil, logger)
	notifHandler.notifService = mockNotifSvc
	return router, mockAlertSvc, mockNotifSvc, alertHandler, notifHandler
}

func TestAlertHandler_CreateAlertRule_Success(t *testing.T) {
	router, mockSvc, _, handler, _ := setupAlertHandlerTest()

	req := &model.AlertRuleRequest{
		Name:        "test-rule",
		ChangeTypes: "add_object",
	}
	resp := &model.AlertRuleResponse{
		ID:   "rule-1",
		Name: "test-rule",
	}

	mockSvc.On("CreateAlertRule", mock.Anything, req).Return(resp, nil)

	router.POST("/alerts/rules", handler.CreateAlertRule)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/alerts/rules", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	result := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, result.Code)
	mockSvc.AssertExpectations(t)
}

func TestAlertHandler_ListAlertRules_Success(t *testing.T) {
	router, mockSvc, _, handler, _ := setupAlertHandlerTest()

	rules := []*model.AlertRuleResponse{
		{ID: "rule-1", Name: "rule-1"},
	}

	mockSvc.On("ListAlertRules", mock.Anything, (*string)(nil)).Return(rules, nil)

	router.GET("/alerts/rules", handler.ListAlertRules)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/alerts/rules", nil)
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	result := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, result.Code)
	mockSvc.AssertExpectations(t)
}

func TestAlertHandler_GetAlertRule_NotFound(t *testing.T) {
	router, mockSvc, _, handler, _ := setupAlertHandlerTest()

	mockSvc.On("GetAlertRule", mock.Anything, "rule-999").Return(nil, errors.New("not found"))

	router.GET("/alerts/rules/:id", handler.GetAlertRule)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/alerts/rules/rule-999", nil)
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
	result := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusNotFound, result.Code)
	mockSvc.AssertExpectations(t)
}

func TestNotificationHandler_ListNotifications_NoAuth(t *testing.T) {
	router, _, _, _, handler := setupAlertHandlerTest()

	router.GET("/notifications", handler.ListNotifications)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/notifications", nil)
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	result := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusUnauthorized, result.Code)
	assert.Equal(t, "UNAUTHORIZED", result.ErrorCode)
}
