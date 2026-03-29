package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTermService 是TermService的mock
type MockTermService struct {
	mock.Mock
}

func (m *MockTermService) CreateTerm(ctx context.Context, req *model.BusinessTermRequest) (*model.BusinessTermResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.BusinessTermResponse), args.Error(1)
}

func (m *MockTermService) ListTerms(ctx context.Context, category string) ([]*model.BusinessTermResponse, error) {
	args := m.Called(ctx, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.BusinessTermResponse), args.Error(1)
}

func (m *MockTermService) GetTerm(ctx context.Context, id string) (*model.BusinessTermResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.BusinessTermResponse), args.Error(1)
}

func (m *MockTermService) UpdateTerm(ctx context.Context, id string, req *model.BusinessTermRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *MockTermService) DeleteTerm(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTermService) AssignTermToColumn(ctx context.Context, columnID string, req *model.AssignTermRequest) error {
	args := m.Called(ctx, columnID, req)
	return args.Error(0)
}

// MockDDLService 是DDLService的mock
type MockDDLService struct {
	mock.Mock
}

func (m *MockDDLService) GenerateDDL(ctx context.Context, objectID string, targetType string) (*model.DDLGenerateResponse, error) {
	args := m.Called(ctx, objectID, targetType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DDLGenerateResponse), args.Error(1)
}

func setupTermHandlerTest() (*gin.Engine, *MockTermService, *MockDDLService, *TermHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockTermSvc := new(MockTermService)
	mockDDLSvc := new(MockDDLService)

	handler := &TermHandler{
		Handler:     NewHandler(),
		termService: TermService(mockTermSvc),
		ddlService:  DDLService(mockDDLSvc),
	}

	return router, mockTermSvc, mockDDLSvc, handler
}

func TestTermHandler_CreateTerm(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	termReq := &model.BusinessTermRequest{
		Name:        "CustomerID",
		Description: "Unique identifier for a customer",
		Category:    "identifier",
	}

	termResp := &model.BusinessTermResponse{
		ID:          "550e8400-e29b-41d4-a716-446655440010",
		Name:        "CustomerID",
		Description: "Unique identifier for a customer",
		Category:    "identifier",
	}

	mockSvc.On("CreateTerm", mock.Anything, termReq).Return(termResp, nil)

	router.POST("/terms", handler.CreateTerm)

	body, _ := json.Marshal(termReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/terms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_CreateTerm_InvalidRequest(t *testing.T) {
	router, _, _, handler := setupTermHandlerTest()

	router.POST("/terms", handler.CreateTerm)

	// 缺少必填字段name
	body := `{"description": "test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/terms", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestTermHandler_ListTerms(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	terms := []*model.BusinessTermResponse{
		{
			ID:       "550e8400-e29b-41d4-a716-446655440010",
			Name:     "CustomerID",
			Category: "identifier",
		},
		{
			ID:       "550e8400-e29b-41d4-a716-446655440011",
			Name:     "OrderDate",
			Category: "date",
		},
	}

	mockSvc.On("ListTerms", mock.Anything, "").Return(terms, nil)

	router.GET("/terms", handler.ListTerms)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/terms", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_ListTerms_WithCategory(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	terms := []*model.BusinessTermResponse{
		{
			ID:       "550e8400-e29b-41d4-a716-446655440010",
			Name:     "CustomerID",
			Category: "identifier",
		},
	}

	mockSvc.On("ListTerms", mock.Anything, "identifier").Return(terms, nil)

	router.GET("/terms", handler.ListTerms)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/terms?category=identifier", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_GetTerm(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	termID := "550e8400-e29b-41d4-a716-446655440010"
	term := &model.BusinessTermResponse{
		ID:          termID,
		Name:        "CustomerID",
		Description: "Unique identifier for a customer",
		Category:    "identifier",
	}

	mockSvc.On("GetTerm", mock.Anything, termID).Return(term, nil)

	router.GET("/terms/:id", handler.GetTerm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/terms/"+termID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_GetTerm_NotFound(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	termID := "550e8400-e29b-41d4-a716-446655440010"

	mockSvc.On("GetTerm", mock.Anything, termID).Return(nil, errors.New("term not found"))

	router.GET("/terms/:id", handler.GetTerm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/terms/"+termID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "NOT_FOUND", resp.ErrorCode)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_GetTerm_MissingID(t *testing.T) {
	router, _, _, handler := setupTermHandlerTest()

	router.GET("/terms/:id", handler.GetTerm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/terms/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTermHandler_UpdateTerm(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	termID := "550e8400-e29b-41d4-a716-446655440010"
	termReq := &model.BusinessTermRequest{
		Name:        "CustomerID",
		Description: "Updated description",
		Category:    "identifier",
	}

	mockSvc.On("UpdateTerm", mock.Anything, termID, termReq).Return(nil)

	router.PUT("/terms/:id", handler.UpdateTerm)

	body, _ := json.Marshal(termReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/terms/"+termID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_DeleteTerm(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	termID := "550e8400-e29b-41d4-a716-446655440010"

	mockSvc.On("DeleteTerm", mock.Anything, termID).Return(nil)

	router.DELETE("/terms/:id", handler.DeleteTerm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/terms/"+termID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_AssignTermToColumn(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	termID := "550e8400-e29b-41d4-a716-446655440010"
	assignReq := &model.AssignTermRequest{
		TermID: &termID,
	}

	mockSvc.On("AssignTermToColumn", mock.Anything, columnID, assignReq).Return(nil)

	router.POST("/columns/:id/term", handler.AssignTermToColumn)

	body, _ := json.Marshal(assignReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/term", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_AssignTermToColumn_Unassign(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	assignReq := &model.AssignTermRequest{
		TermID: nil,
	}

	mockSvc.On("AssignTermToColumn", mock.Anything, columnID, assignReq).Return(nil)

	router.POST("/columns/:id/term", handler.AssignTermToColumn)

	body, _ := json.Marshal(assignReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/term", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_AssignTermToColumn_InvalidRequest(t *testing.T) {
	router, _, _, handler := setupTermHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"

	router.POST("/columns/:id/term", handler.AssignTermToColumn)

	body := `{"term_id": }`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/term", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestTermHandler_AssignTermToColumn_ServiceError(t *testing.T) {
	router, mockSvc, _, handler := setupTermHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	termID := "550e8400-e29b-41d4-a716-446655440010"
	assignReq := &model.AssignTermRequest{
		TermID: &termID,
	}

	mockSvc.On("AssignTermToColumn", mock.Anything, columnID, assignReq).Return(errors.New("assign failed"))

	router.POST("/columns/:id/term", handler.AssignTermToColumn)

	body, _ := json.Marshal(assignReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/term", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)

	mockSvc.AssertExpectations(t)
}

func TestTermHandler_AssignTermToColumn_MissingColumnID(t *testing.T) {
	router, _, _, handler := setupTermHandlerTest()

	router.POST("/columns/:id/term", handler.AssignTermToColumn)

	termID := "550e8400-e29b-41d4-a716-446655440010"
	assignReq := &model.AssignTermRequest{
		TermID: &termID,
	}

	body, _ := json.Marshal(assignReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns//term", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// gin将空字符串作为id参数传递，handler返回400而不是404
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTermHandler_GenerateDDL_ServiceError(t *testing.T) {
	router, _, mockDDLSvc, handler := setupTermHandlerTest()

	ddlReq := &model.DDLGenerateRequest{
		ObjectID:   "550e8400-e29b-41d4-a716-446655440002",
		TargetType: "postgres",
	}

	mockDDLSvc.On("GenerateDDL", mock.Anything, ddlReq.ObjectID, ddlReq.TargetType).Return(nil, errors.New("ddl failed"))

	router.POST("/ddl/generate", handler.GenerateDDL)

	body, _ := json.Marshal(ddlReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ddl/generate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)

	mockDDLSvc.AssertExpectations(t)
}

func TestTermHandler_GenerateDDL(t *testing.T) {
	router, _, mockDDLSvc, handler := setupTermHandlerTest()

	ddlReq := &model.DDLGenerateRequest{
		ObjectID:   "550e8400-e29b-41d4-a716-446655440002",
		TargetType: "postgres",
	}

	ddlResp := &model.DDLGenerateResponse{
		ObjectID: "550e8400-e29b-41d4-a716-446655440002",
		SQL:      "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));",
	}

	mockDDLSvc.On("GenerateDDL", mock.Anything, ddlReq.ObjectID, ddlReq.TargetType).Return(ddlResp, nil)

	router.POST("/ddl/generate", handler.GenerateDDL)

	body, _ := json.Marshal(ddlReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ddl/generate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockDDLSvc.AssertExpectations(t)
}

func TestTermHandler_GenerateDDL_InvalidRequest(t *testing.T) {
	router, _, _, handler := setupTermHandlerTest()

	router.POST("/ddl/generate", handler.GenerateDDL)

	// 缺少必填字段
	body := `{"object_id": "test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ddl/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}
