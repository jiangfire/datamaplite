package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	responsepkg "git.neolidy.top/neo/fuckcmdb/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_DisabledAllowsReadOnlyRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authService := service.NewAuthService(nil, &service.AuthConfig{Enabled: false})

	router.GET("/protected", AuthMiddleware(authService), func(c *gin.Context) {
		authCtx, exists := GetAuthContext(c)
		require.True(t, exists)
		require.NotNil(t, authCtx)
		c.JSON(http.StatusOK, responsepkg.Success(gin.H{
			"user_id": authCtx.UserID,
			"role":    authCtx.Role,
		}))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, successCode, resp.Code)
}

func TestAuthMiddleware_DisabledBlocksWriteRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authService := service.NewAuthService(nil, &service.AuthConfig{Enabled: false})

	router.POST("/protected", AuthMiddleware(authService), func(c *gin.Context) {
		c.JSON(http.StatusOK, responsepkg.Success(nil))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/protected", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Equal(t, "AUTH_DISABLED", resp.ErrorCode)
}

func TestGovernanceAuditMiddleware_InjectsAuditMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.POST("/sources/:id/sync", func(c *gin.Context) {
		c.Set(AuthContextKey, &model.AuthContext{
			UserID:   "user-123",
			Username: "tester",
			Role:     model.UserRoleAdmin,
		})
		c.Next()
	}, GovernanceAuditMiddleware(), func(c *gin.Context) {
		meta := service.GovernanceAuditMetaFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{
			"actor_id":  meta.ActorID,
			"trace_id":  meta.TraceID,
			"origin":    meta.Origin,
			"operation": meta.Operation,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/sources/source-1/sync", nil)
	req.Header.Set("X-Trace-ID", "trace-http-123")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, "user-123", payload["actor_id"])
	assert.Equal(t, "trace-http-123", payload["trace_id"])
	assert.Equal(t, "http", payload["origin"])
	assert.Equal(t, "POST /sources/:id/sync", payload["operation"])
}

func TestMCPAuthMiddleware_RequiresTokenWhenAuthDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authService := service.NewAuthService(nil, &service.AuthConfig{Enabled: false})

	router.Any("/mcp", MCPAuthMiddleware(authService), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/mcp", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Equal(t, "AUTH_DISABLED", resp.ErrorCode)
	assert.Equal(t, "Authentication is required for MCP endpoint", resp.Message)
}

func TestMCPAuthMiddleware_RejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authService := service.NewAuthService(nil, &service.AuthConfig{
		Enabled:         true,
		JWTSecret:       "mcp-test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: time.Hour,
		BcryptCost:      4,
	})

	router.Any("/mcp", MCPAuthMiddleware(authService), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/mcp", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)

	resp := decodeHTTPResult(t, w.Body.Bytes())
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Equal(t, "UNAUTHORIZED", resp.ErrorCode)
	assert.Equal(t, "Authorization header required", resp.Message)
}

func TestMCPAuthMiddleware_AllowsValidAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authService := service.NewAuthService(nil, &service.AuthConfig{
		Enabled:         true,
		JWTSecret:       "mcp-test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: time.Hour,
		BcryptCost:      4,
	})

	router.Any("/mcp", MCPAuthMiddleware(authService), func(c *gin.Context) {
		authCtx, exists := GetAuthContext(c)
		require.True(t, exists)
		require.NotNil(t, authCtx)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  authCtx.UserID,
			"username": authCtx.Username,
			"role":     authCtx.Role,
		})
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    "user-mcp-1",
		"username":   "mcp-user",
		"role":       string(model.UserRoleAdmin),
		"token_type": "access",
		"exp":        time.Now().Add(15 * time.Minute).Unix(),
		"iat":        time.Now().Unix(),
	})
	tokenString, err := token.SignedString([]byte("mcp-test-secret"))
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, "user-mcp-1", payload["user_id"])
	assert.Equal(t, "mcp-user", payload["username"])
	assert.Equal(t, string(model.UserRoleAdmin), payload["role"])
}
