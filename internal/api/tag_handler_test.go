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
)

// MockTagService is a mock for TagService
type MockTagService struct {
	mock.Mock
}

func (m *MockTagService) ListTags(ctx context.Context) ([]*model.TagResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TagResponse), args.Error(1)
}

func (m *MockTagService) CreateTag(ctx context.Context, req *model.TagRequest) (*model.TagResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TagResponse), args.Error(1)
}

func (m *MockTagService) GetTag(ctx context.Context, id string) (*model.TagResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TagResponse), args.Error(1)
}

func (m *MockTagService) UpdateTag(ctx context.Context, id string, req *model.TagRequest) error {
	args := m.Called(ctx, id, req)
	return args.Error(0)
}

func (m *MockTagService) DeleteTag(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTagService) GetColumnsByTag(ctx context.Context, id string) ([]*model.ColumnSearchResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ColumnSearchResult), args.Error(1)
}

func (m *MockTagService) AddTagToColumn(ctx context.Context, columnID string, tagID string) error {
	args := m.Called(ctx, columnID, tagID)
	return args.Error(0)
}

func (m *MockTagService) RemoveTagFromColumn(ctx context.Context, columnID string, tagID string) error {
	args := m.Called(ctx, columnID, tagID)
	return args.Error(0)
}

func (m *MockTagService) GetColumnTags(ctx context.Context, columnID string) ([]*model.TagResponse, error) {
	args := m.Called(ctx, columnID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TagResponse), args.Error(1)
}

func setupTagHandlerTest() (*gin.Engine, *MockTagService, *TagHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mockSvc := new(MockTagService)
	handler := NewTagHandler(nil)
	handler.tagService = mockSvc
	return router, mockSvc, handler
}

func TestTagHandler_ListTags_Success(t *testing.T) {
	router, mockSvc, handler := setupTagHandlerTest()

	tags := []*model.TagResponse{
		{ID: "tag-1", Name: "PII", Color: "#ff0000"},
		{ID: "tag-2", Name: "Sensitive", Color: "#00ff00"},
	}

	mockSvc.On("ListTags", mock.Anything).Return(tags, nil)

	router.GET("/tags", handler.ListTags)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/tags", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestTagHandler_CreateTag_Success(t *testing.T) {
	router, mockSvc, handler := setupTagHandlerTest()

	req := &model.TagRequest{
		Name:  "NewTag",
		Color: "#0000ff",
	}
	resp := &model.TagResponse{
		ID:    "tag-3",
		Name:  "NewTag",
		Color: "#0000ff",
	}

	mockSvc.On("CreateTag", mock.Anything, req).Return(resp, nil)

	router.POST("/tags", handler.CreateTag)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodPost, "/tags", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	result := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, result.Code)
	mockSvc.AssertExpectations(t)
}

func TestTagHandler_GetTag_NotFound(t *testing.T) {
	router, mockSvc, handler := setupTagHandlerTest()

	mockSvc.On("GetTag", mock.Anything, "tag-999").Return(nil, errors.New("not found"))

	router.GET("/tags/:id", handler.GetTag)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/tags/tag-999", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusNotFound, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestTagHandler_DeleteTag_Success(t *testing.T) {
	router, mockSvc, handler := setupTagHandlerTest()

	mockSvc.On("DeleteTag", mock.Anything, "tag-1").Return(nil)

	router.DELETE("/tags/:id", handler.DeleteTag)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/tags/tag-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	mockSvc.AssertExpectations(t)
}
