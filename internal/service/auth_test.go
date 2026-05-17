package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupAuthService(_ *testing.T) (*AuthService, *MockStore) {
	mockStore := new(MockStore)
	config := &AuthConfig{
		Enabled:         true,
		JWTSecret:       "test-secret-key-for-jwt-signing-32",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      10,
	}
	service := NewAuthService(mockStore, config)
	return service, mockStore
}

func TestAuthService_Login_Success(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupAuthService(t)

	// bcrypt hash for "admin123"
	passwordHash := "$2a$10$cu/yC6//Yf3oCCW8fkh5/ue6nQfR3cM19a2Onc8T1g8TD5G7vJUVC"

	mockStore.On("GetUserByUsername", ctx, "testuser").Return(&store.UserRow{
		ID:           "user-123",
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: passwordHash,
		Role:         "user",
		CreatedAt:    "2024-01-01T00:00:00Z",
	}, nil)

	req := &model.LoginRequest{
		Username: "testuser",
		Password: "admin123", // 默认密码
	}

	resp, err := service.Login(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "user-123", resp.User.ID)
	assert.Equal(t, "testuser", resp.User.Username)
	mockStore.AssertExpectations(t)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupAuthService(t)

	passwordHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

	mockStore.On("GetUserByUsername", ctx, "testuser").Return(&store.UserRow{
		ID:           "user-123",
		Username:     "testuser",
		PasswordHash: passwordHash,
		Role:         "user",
	}, nil)

	req := &model.LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}

	_, err := service.Login(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid username or password")
	mockStore.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupAuthService(t)

	mockStore.On("GetUserByUsername", ctx, "nonexistent").Return(nil, errors.New("user not found"))

	req := &model.LoginRequest{
		Username: "nonexistent",
		Password: "password123",
	}

	_, err := service.Login(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid username or password")
	mockStore.AssertExpectations(t)
}

func TestAuthService_ParseToken_Success(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupAuthService(t)

	passwordHash := "$2a$10$cu/yC6//Yf3oCCW8fkh5/ue6nQfR3cM19a2Onc8T1g8TD5G7vJUVC"

	mockStore.On("GetUserByUsername", ctx, "testuser").Return(&store.UserRow{
		ID:           "user-123",
		Username:     "testuser",
		PasswordHash: passwordHash,
		Role:         "admin",
	}, nil)

	// First login to get a token
	req := &model.LoginRequest{
		Username: "testuser",
		Password: "admin123",
	}
	resp, err := service.Login(ctx, req)
	require.NoError(t, err)

	// Parse the token
	claims, err := service.ParseToken(resp.AccessToken)

	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, model.UserRoleAdmin, claims.Role)
	assert.Equal(t, "access", claims.TokenType)
}

func TestAuthService_ParseToken_Invalid(t *testing.T) {
	service, _ := setupAuthService(t)

	_, err := service.ParseToken("invalid.token.here")

	assert.Error(t, err)
}

func TestAuthService_Register_Success(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupAuthService(t)

	// User does not exist
	mockStore.On("GetUserByUsername", ctx, "newuser").Return(nil, errors.New("user not found"))
	mockStore.On("CreateUser", ctx, mock.AnythingOfType("*store.UserCreate")).Return("user-new", nil)
	mockStore.On("GetUserByID", ctx, "user-new").Return(&store.UserRow{
		ID:        "user-new",
		Username:  "newuser",
		Email:     "new@example.com",
		Role:      "user",
		CreatedAt: "2024-01-01T00:00:00Z",
	}, nil)

	req := &model.RegisterRequest{
		Username: "newuser",
		Password: "password123",
		Email:    "new@example.com",
		Role:     model.UserRoleUser,
	}

	user, err := service.Register(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "user-new", user.ID)
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, "new@example.com", user.Email)
	mockStore.AssertExpectations(t)
}

func TestAuthService_Register_UsernameExists(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupAuthService(t)

	// User already exists
	mockStore.On("GetUserByUsername", ctx, "existing").Return(&store.UserRow{
		ID:       "user-existing",
		Username: "existing",
	}, nil)

	req := &model.RegisterRequest{
		Username: "existing",
		Password: "password123",
		Email:    "existing@example.com",
	}

	_, err := service.Register(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username already exists")
	mockStore.AssertExpectations(t)
}

func TestAuthService_GetUserByID_Success(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupAuthService(t)

	mockStore.On("GetUserByID", ctx, "user-123").Return(&store.UserRow{
		ID:        "user-123",
		Username:  "testuser",
		Email:     "test@example.com",
		Role:      "user",
		CreatedAt: "2024-01-01T00:00:00Z",
	}, nil)

	user, err := service.GetUserByID(ctx, "user-123")

	require.NoError(t, err)
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "testuser", user.Username)
	mockStore.AssertExpectations(t)
}

func TestDefaultAuthConfig(t *testing.T) {
	cfg := DefaultAuthConfig()

	assert.True(t, cfg.Enabled)
	assert.NotEmpty(t, cfg.JWTSecret)
	assert.Equal(t, 15*time.Minute, cfg.AccessTokenTTL)
	assert.Equal(t, 7*24*time.Hour, cfg.RefreshTokenTTL)
	assert.Equal(t, 10, cfg.BcryptCost)
}

func TestAuthService_ResolveAnonymousAuthContext(t *testing.T) {
	service := NewAuthService(nil, &AuthConfig{Enabled: false})

	authCtx := service.ResolveAnonymousAuthContext()

	require.NotNil(t, authCtx)
	assert.Equal(t, "anonymous", authCtx.UserID)
	assert.Equal(t, "anonymous", authCtx.Username)
	assert.Equal(t, model.UserRoleUser, authCtx.Role)
}

func TestAuthService_Login_Disabled(t *testing.T) {
	service := NewAuthService(nil, &AuthConfig{Enabled: false})

	_, err := service.Login(context.Background(), &model.LoginRequest{
		Username: "testuser",
		Password: "password",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication is disabled")
}
