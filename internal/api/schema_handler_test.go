package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupSchemaHandlerTest() (*gin.Engine, *MockMetadataService, *SchemaHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockMetadataSvc := new(MockMetadataService)

	handler := &SchemaHandler{
		Handler:         NewHandler(),
		metadataService: MetadataService(mockMetadataSvc),
	}

	return router, mockMetadataSvc, handler
}

func TestSchemaHandler_GetColumnDetail(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	detail := &model.ColumnDetailResponse{
		ID:           columnID,
		Name:         "user_id",
		DataType:     "int",
		FullDataType: "int(11)",
		IsNullable:   false,
		Object: model.ObjectSummary{
			ID:   "550e8400-e29b-41d4-a716-446655440002",
			Name: "users",
			Type: "table",
		},
		Source: model.SourceSummary{
			ID:   "550e8400-e29b-41d4-a716-446655440000",
			Name: "Test MySQL",
			Type: "mysql",
		},
	}

	mockSvc.On("GetColumnDetail", mock.Anything, columnID).Return(detail, nil)

	router.GET("/columns/:id", handler.GetColumnDetail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/"+columnID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_GetColumnDetail_NotFound(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"

	mockSvc.On("GetColumnDetail", mock.Anything, columnID).Return(nil, errors.New("column not found"))

	router.GET("/columns/:id", handler.GetColumnDetail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/"+columnID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "NOT_FOUND", resp.ErrorCode)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_SearchColumns(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	results := []model.ColumnSearchResult{
		{
			ID:         "550e8400-e29b-41d4-a716-446655440001",
			Name:       "user_id",
			DataType:   "int",
			ObjectName: "users",
			SourceID:   "550e8400-e29b-41d4-a716-446655440000",
			SourceName: "Test MySQL",
			SourceType: "mysql",
		},
	}

	mockSvc.On("SearchColumns", mock.Anything, "user", 20).Return(results, nil)

	router.GET("/columns/search", handler.SearchColumns)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/search?q=user", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_SearchColumns_EmptyQuery(t *testing.T) {
	router, _, handler := setupSchemaHandlerTest()

	router.GET("/columns/search", handler.SearchColumns)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/search?q=", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestSchemaHandler_SearchColumns_MissingQuery(t *testing.T) {
	router, _, handler := setupSchemaHandlerTest()

	router.GET("/columns/search", handler.SearchColumns)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/search", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestSchemaHandler_SearchColumns_WhitespaceQuery(t *testing.T) {
	router, _, handler := setupSchemaHandlerTest()

	router.GET("/columns/search", handler.SearchColumns)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/search?q=%20%20%20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestSchemaHandler_GetColumnMappings(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	mappings := []*model.ColumnMappingResponse{
		{
			ID:             "550e8400-e29b-41d4-a716-446655440003",
			SourceColumnID: columnID,
			TargetColumnID: "550e8400-e29b-41d4-a716-446655440004",
			MappingType:    "alias",
			Confidence:     0.95,
			TargetColumn: model.ColumnSummary{
				ID:         "550e8400-e29b-41d4-a716-446655440004",
				Name:       "id",
				DataType:   "int",
				ObjectName: "accounts",
				SourceName: "Test MySQL",
			},
		},
	}

	mockSvc.On("GetColumnMappings", mock.Anything, columnID).Return(mappings, nil)

	router.GET("/columns/:id/mappings", handler.GetColumnMappings)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/"+columnID+"/mappings", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_CreateColumnMapping(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	mappingReq := &model.ColumnMappingRequest{
		SourceColumnID: columnID,
		TargetColumnID: "550e8400-e29b-41d4-a716-446655440004",
		MappingType:    "alias",
		Confidence:     0.95,
	}

	mockSvc.On("CreateColumnMapping", mock.Anything, mappingReq).Return(nil)

	router.POST("/columns/:id/mappings", handler.CreateColumnMapping)

	body, _ := json.Marshal(mappingReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/mappings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_CreateColumnMapping_PathIDOverridesBody(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"

	mockSvc.
		On("CreateColumnMapping", mock.Anything, mock.MatchedBy(func(req *model.ColumnMappingRequest) bool {
			return req.SourceColumnID == columnID &&
				req.TargetColumnID == "550e8400-e29b-41d4-a716-446655440004" &&
				req.MappingType == "alias" &&
				req.Confidence == 0.95
		})).
		Return(nil)

	router.POST("/columns/:id/mappings", handler.CreateColumnMapping)

	body := `{
		"source_column_id": "different-column-id",
		"target_column_id": "550e8400-e29b-41d4-a716-446655440004",
		"mapping_type": "alias",
		"confidence": 0.95
	}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/mappings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_CreateColumnMapping_InvalidRequest(t *testing.T) {
	router, _, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"

	router.POST("/columns/:id/mappings", handler.CreateColumnMapping)

	// 无效的JSON
	body := `{"mapping_type": "invalid_type",}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/mappings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSchemaHandler_CreateColumnMapping_ValidationFailure(t *testing.T) {
	router, _, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"

	router.POST("/columns/:id/mappings", handler.CreateColumnMapping)

	body := `{
		"target_column_id": "550e8400-e29b-41d4-a716-446655440004",
		"mapping_type": "invalid",
		"confidence": 1.5
	}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/mappings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "BAD_REQUEST", resp.ErrorCode)
}

func TestSchemaHandler_CreateColumnMapping_ServiceError(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	mappingReq := &model.ColumnMappingRequest{
		SourceColumnID: columnID,
		TargetColumnID: "550e8400-e29b-41d4-a716-446655440004",
		MappingType:    "alias",
		Confidence:     0.95,
	}

	mockSvc.On("CreateColumnMapping", mock.Anything, mappingReq).Return(errors.New("create failed"))

	router.POST("/columns/:id/mappings", handler.CreateColumnMapping)

	body, _ := json.Marshal(mappingReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/columns/"+columnID+"/mappings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_DeleteColumnMapping(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	mappingID := "550e8400-e29b-41d4-a716-446655440003"

	mockSvc.On("DeleteColumnMapping", mock.Anything, mappingID).Return(nil)

	router.DELETE("/columns/:id/mappings/:mappingId", handler.DeleteColumnMapping)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/columns/col-1/mappings/"+mappingID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_DeleteColumnMapping_ServiceError(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	mappingID := "550e8400-e29b-41d4-a716-446655440003"

	mockSvc.On("DeleteColumnMapping", mock.Anything, mappingID).Return(errors.New("delete failed"))

	router.DELETE("/columns/:id/mappings/:mappingId", handler.DeleteColumnMapping)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/columns/col-1/mappings/"+mappingID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_GetLineage(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	lineage := &model.LineageResponse{
		ColumnID: columnID,
		Upward: []model.LineageEdgeResponse{
			{
				Source: model.LineageNode{
					ID:   "550e8400-e29b-41d4-a716-446655440005",
					Name: "raw_user_id",
					Type: "column",
				},
				Target: model.LineageNode{
					ID:   columnID,
					Name: "user_id",
					Type: "column",
				},
			},
		},
		Downward: []model.LineageEdgeResponse{},
	}

	mockSvc.On("GetLineage", mock.Anything, columnID).Return(lineage, nil)

	router.GET("/columns/:id/lineage", handler.GetLineage)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/"+columnID+"/lineage", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_GetImpactAnalysis_Error(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"

	mockSvc.On("GetImpactAnalysis", mock.Anything, columnID).Return(nil, errors.New("impact failed"))

	router.GET("/columns/:id/impact", handler.GetImpactAnalysis)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/"+columnID+"/impact", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)

	mockSvc.AssertExpectations(t)
}

func TestSchemaHandler_GetImpactAnalysis(t *testing.T) {
	router, mockSvc, handler := setupSchemaHandlerTest()

	columnID := "550e8400-e29b-41d4-a716-446655440001"
	impact := &model.ImpactAnalysisResponse{
		ColumnID: columnID,
		ImpactObjects: []model.ImpactObject{
			{
				ID:         "550e8400-e29b-41d4-a716-446655440006",
				Name:       "user_orders_view",
				Type:       "object",
				SourceName: "Test MySQL",
				ImpactPath: "users.user_id -> user_orders_view",
				Distance:   1,
			},
		},
		TotalObjects: 1,
	}

	mockSvc.On("GetImpactAnalysis", mock.Anything, columnID).Return(impact, nil)

	router.GET("/columns/:id/impact", handler.GetImpactAnalysis)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/columns/"+columnID+"/impact", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	mockSvc.AssertExpectations(t)
}
