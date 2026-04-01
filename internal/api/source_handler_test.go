package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/scanner"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSourceService 是SourceService的mock
type MockSourceService struct {
	mock.Mock
}

func (m *MockSourceService) ListSources(ctx context.Context) ([]*model.SourceListItem, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.SourceListItem), args.Error(1)
}

func (m *MockSourceService) CreateSource(ctx context.Context, req *model.CreateSourceRequest) (*model.SourceResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SourceResponse), args.Error(1)
}

func (m *MockSourceService) GetSource(ctx context.Context, id string) (*model.SourceResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SourceResponse), args.Error(1)
}

func (m *MockSourceService) UpdateSource(ctx context.Context, id string, req *model.UpdateSourceRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *MockSourceService) DeleteSource(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSourceService) TestConnection(ctx context.Context, dbType string, config scanner.ConnectionConfig) error {
	args := m.Called(ctx, dbType, config)
	return args.Error(0)
}

func (m *MockSourceService) TriggerSync(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockMetadataService 是MetadataService的mock
type MockMetadataService struct {
	mock.Mock
}

func (m *MockMetadataService) GetSchemaTree(ctx context.Context, sourceID string) (*model.SchemaTreeResponse, error) {
	args := m.Called(ctx, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SchemaTreeResponse), args.Error(1)
}

func (m *MockMetadataService) ListSchemaChanges(ctx context.Context, sourceID string, limit int) ([]*model.SchemaChangeResponse, error) {
	args := m.Called(ctx, sourceID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.SchemaChangeResponse), args.Error(1)
}

func (m *MockMetadataService) GetColumnDetail(ctx context.Context, columnID string) (*model.ColumnDetailResponse, error) {
	args := m.Called(ctx, columnID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ColumnDetailResponse), args.Error(1)
}

func (m *MockMetadataService) SearchColumns(ctx context.Context, query string, limit int) ([]model.ColumnSearchResult, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ColumnSearchResult), args.Error(1)
}

func (m *MockMetadataService) GetColumnMappings(ctx context.Context, columnID string) ([]*model.ColumnMappingResponse, error) {
	args := m.Called(ctx, columnID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ColumnMappingResponse), args.Error(1)
}

func (m *MockMetadataService) CreateColumnMapping(ctx context.Context, req *model.ColumnMappingRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockMetadataService) DeleteColumnMapping(ctx context.Context, mappingID string) error {
	args := m.Called(ctx, mappingID)
	return args.Error(0)
}

func (m *MockMetadataService) GetLineage(ctx context.Context, columnID string) (*model.LineageResponse, error) {
	args := m.Called(ctx, columnID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LineageResponse), args.Error(1)
}

func (m *MockMetadataService) GetImpactAnalysis(ctx context.Context, columnID string) (*model.ImpactAnalysisResponse, error) {
	args := m.Called(ctx, columnID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ImpactAnalysisResponse), args.Error(1)
}

func setupSourceHandlerTest() (*gin.Engine, *MockSourceService, *MockMetadataService, *SourceHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockSourceSvc := new(MockSourceService)
	mockMetadataSvc := new(MockMetadataService)

	handler := &SourceHandler{
		Handler:         NewHandler(),
		sourceService:   mockSourceSvc,
		metadataService: mockMetadataSvc,
	}

	return router, mockSourceSvc, mockMetadataSvc, handler
}

func TestSourceHandler_ListSources(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	sources := []*model.SourceListItem{
		{
			ID:     "550e8400-e29b-41d4-a716-446655440000",
			Name:   "Test MySQL",
			Type:   model.DataSourceMySQL,
			Host:   "localhost",
			Port:   3306,
			Status: model.DataSourceStatusActive,
		},
	}

	mockSvc.On("ListSources", mock.Anything).Return(sources, nil)

	router.GET("/sources", handler.ListSources)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/sources", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	data, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var result []*model.SourceListItem
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "Test MySQL", result[0].Name)

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_ListSources_Error(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	mockSvc.On("ListSources", mock.Anything).Return(nil, errors.New("database error"))

	router.GET("/sources", handler.ListSources)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/sources", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_CreateSource(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	createReq := &model.CreateSourceRequest{
		Name:     "Test MySQL",
		Type:     model.DataSourceMySQL,
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Username: "root",
		Password: "password",
	}

	createdSource := &model.SourceResponse{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		Name:     "Test MySQL",
		Type:     model.DataSourceMySQL,
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Status:   model.DataSourceStatusActive,
	}

	mockSvc.On("CreateSource", mock.Anything, createReq).Return(createdSource, nil)

	router.POST("/sources", handler.CreateSource)

	body, _ := json.Marshal(createReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/sources", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_CreateSource_InvalidRequest(t *testing.T) {
	router, _, _, handler := setupSourceHandlerTest()

	router.POST("/sources", handler.CreateSource)

	// 缺少必填字段
	body := `{"name": "test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/sources", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestSourceHandler_GetSource(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	sourceID := "550e8400-e29b-41d4-a716-446655440000"
	source := &model.SourceResponse{
		ID:       sourceID,
		Name:     "Test MySQL",
		Type:     model.DataSourceMySQL,
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Status:   model.DataSourceStatusActive,
	}

	mockSvc.On("GetSource", mock.Anything, sourceID).Return(source, nil)

	router.GET("/sources/:id", handler.GetSource)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/sources/"+sourceID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_GetSource_NotFound(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	sourceID := "550e8400-e29b-41d4-a716-446655440000"

	mockSvc.On("GetSource", mock.Anything, sourceID).Return(nil, errors.New("source not found"))

	router.GET("/sources/:id", handler.GetSource)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/sources/"+sourceID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "NOT_FOUND", resp.ErrorCode)

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_GetSource_MissingID(t *testing.T) {
	router, _, _, handler := setupSourceHandlerTest()

	router.GET("/sources/:id", handler.GetSource)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/sources/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSourceHandler_UpdateSource(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	sourceID := "550e8400-e29b-41d4-a716-446655440000"
	updateReq := &model.UpdateSourceRequest{
		Name: "Updated MySQL",
	}

	mockSvc.On("UpdateSource", mock.Anything, sourceID, updateReq).Return(nil)

	router.PUT("/sources/:id", handler.UpdateSource)

	body, _ := json.Marshal(updateReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/sources/"+sourceID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_DeleteSource(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	sourceID := "550e8400-e29b-41d4-a716-446655440000"

	mockSvc.On("DeleteSource", mock.Anything, sourceID).Return(nil)

	router.DELETE("/sources/:id", handler.DeleteSource)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/sources/"+sourceID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_TestConnection_Success(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	testReq := &model.ConnectionTestRequest{
		Type:     model.DataSourceMySQL,
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Username: "root",
		Password: "password",
	}

	mockSvc.On("TestConnection", mock.Anything, string(model.DataSourceMySQL), scanner.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Username: "root",
		Password: "password",
	}).Return(nil)

	router.POST("/test", handler.TestConnection)

	body, _ := json.Marshal(testReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	assert.Equal(t, "Connection successful", resp.Message)
	assert.Nil(t, resp.Data)

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_TestConnection_Failure(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	testReq := &model.ConnectionTestRequest{
		Type:     model.DataSourceMySQL,
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Username: "root",
		Password: "wrongpassword",
	}

	mockSvc.On("TestConnection", mock.Anything, string(model.DataSourceMySQL), scanner.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		Username: "root",
		Password: "wrongpassword",
	}).Return(errors.New("connection refused"))

	router.POST("/test", handler.TestConnection)

	body, _ := json.Marshal(testReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadGateway, resp.Code)
	assert.Equal(t, "CONNECTION_TEST_FAILED", resp.ErrorCode)
	assert.Contains(t, resp.Message, "connection refused")

	mockSvc.AssertExpectations(t)
}

func TestSourceHandler_TriggerSync(t *testing.T) {
	router, mockSvc, _, handler := setupSourceHandlerTest()

	sourceID := "550e8400-e29b-41d4-a716-446655440000"

	mockSvc.On("TriggerSync", mock.Anything, sourceID).Return(nil)

	router.POST("/sources/:id/sync", handler.TriggerSync)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/sources/"+sourceID+"/sync", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	var syncResp model.SyncResponse
	data, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &syncResp))
	assert.Equal(t, sourceID, syncResp.SourceID)
	assert.WithinDuration(t, time.Now(), syncResp.StartedAt, time.Second)

	mockSvc.AssertExpectations(t)
}
