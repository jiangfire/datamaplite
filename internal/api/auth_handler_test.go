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
	"github.com/stretchr/testify/require"
)

// MockAuthService is a mock for AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LoginResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, token string) (*model.LoginResponse, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LoginResponse), args.Error(1)
}

func (m *MockAuthService) Register(ctx context.Context, req *model.RegisterRequest) (*model.UserInfo, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserInfo), args.Error(1)
}

func (m *MockAuthService) GetUserByID(ctx context.Context, id string) (*model.UserInfo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserInfo), args.Error(1)
}

func (m *MockAuthService) ListUsers(ctx context.Context) ([]*model.UserInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.UserInfo), args.Error(1)
}

func (m *MockAuthService) UpdateUser(ctx context.Context, id string, req *model.UpdateUserRequest) (*model.UserInfo, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserInfo), args.Error(1)
}

func (m *MockAuthService) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupAuthHandlerTest() (*gin.Engine, *MockAuthService, *AuthHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(nil)
	handler.authService = mockSvc
	return router, mockSvc, handler
}

func TestAuthHandler_Login_Success(t *testing.T) {
	router, mockSvc, handler := setupAuthHandlerTest()

	loginReq := &model.LoginRequest{
		Username: "admin",
		Password: "admin123",
	}
	loginResp := &model.LoginResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    900,
		User: &model.UserInfo{
			ID:       "user-1",
			Username: "admin",
			Role:     "admin",
		},
	}

	mockSvc.On("IsEnabled").Return(true)
	mockSvc.On("Login", mock.Anything, loginReq).Return(loginResp, nil)

	router.POST("/login", handler.Login)

	body, _ := json.Marshal(loginReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Login_AuthDisabled(t *testing.T) {
	router, mockSvc, handler := setupAuthHandlerTest()

	mockSvc.On("IsEnabled").Return(false)

	router.POST("/login", handler.Login)

	body := `{"username":"admin","password":"admin123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Equal(t, "AUTH_DISABLED", resp.ErrorCode)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	router, mockSvc, handler := setupAuthHandlerTest()

	loginReq := &model.LoginRequest{
		Username: "admin",
		Password: "wrong",
	}

	mockSvc.On("IsEnabled").Return(true)
	mockSvc.On("Login", mock.Anything, loginReq).Return(nil, errors.New("invalid credentials"))

	router.POST("/login", handler.Login)

	body, _ := json.Marshal(loginReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	router, mockSvc, handler := setupAuthHandlerTest()

	registerReq := &model.RegisterRequest{
		Username: "newuser",
		Password: "password123",
		Email:    "newuser@example.com",
		Role:     "user",
	}
	userResp := &model.UserInfo{
		ID:       "user-2",
		Username: "newuser",
		Role:     "user",
	}

	mockSvc.On("Register", mock.Anything, registerReq).Return(userResp, nil)

	router.POST("/register", handler.Register)

	body, _ := json.Marshal(registerReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestAuthHandler_ListUsers_Success(t *testing.T) {
	router, mockSvc, handler := setupAuthHandlerTest()

	users := []*model.UserInfo{
		{ID: "user-1", Username: "admin", Role: "admin"},
		{ID: "user-2", Username: "john", Role: "user"},
	}

	mockSvc.On("ListUsers", mock.Anything).Return(users, nil)

	router.GET("/users", handler.ListUsers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)

	data, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var result []*model.UserInfo
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Len(t, result, 2)
	mockSvc.AssertExpectations(t)
}
