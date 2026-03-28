package api

import (
	"net/http"
	"strings"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	responsepkg "git.neolidy.top/neo/fuckcmdb/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// AuthContextKey 认证上下文在gin.Context中的key
	AuthContextKey = "auth_context"
)

func abortWithError(c *gin.Context, statusCode int, code string, message string) {
	c.AbortWithStatusJSON(statusCode, responsepkg.Error(statusCode, code, message))
}

func authenticateRequest(c *gin.Context, authService *service.AuthService, allowAnonymousReadWhenDisabled bool, disabledMessage string) bool {
	if !authService.IsEnabled() {
		if allowAnonymousReadWhenDisabled && isReadOnlyMethod(c.Request.Method) {
			c.Set(AuthContextKey, authService.ResolveAnonymousAuthContext())
			return true
		}

		abortWithError(c, http.StatusServiceUnavailable, "AUTH_DISABLED", disabledMessage)
		return false
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		abortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		abortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
		return false
	}

	claims, err := authService.ParseToken(parts[1])
	if err != nil {
		abortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
		return false
	}

	if claims.TokenType != "access" {
		abortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token type")
		return false
	}

	c.Set(AuthContextKey, &model.AuthContext{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	})
	return true
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authenticateRequest(c, authService, true, "Authentication is disabled; write operations are unavailable") {
			return
		}
		c.Next()
	}
}

// MCPAuthMiddleware 要求 MCP 入口始终携带 Bearer Token。
func MCPAuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authenticateRequest(c, authService, false, "Authentication is required for MCP endpoint") {
			return
		}
		c.Next()
	}
}

func isReadOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// GovernanceAuditMiddleware 将 HTTP 请求审计上下文写入 request context。
func GovernanceAuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, _ := GetAuthContext(c)

		actorID := "anonymous"
		if authCtx != nil && authCtx.UserID != "" {
			actorID = authCtx.UserID
		}

		traceID := strings.TrimSpace(c.GetHeader("X-Trace-ID"))
		if traceID == "" {
			traceID = "http_" + uuid.NewString()
		}

		operationPath := c.FullPath()
		if operationPath == "" {
			operationPath = c.Request.URL.Path
		}

		c.Request = c.Request.WithContext(service.WithGovernanceAuditMeta(c.Request.Context(), service.GovernanceAuditMeta{
			ActorID:   actorID,
			TraceID:   traceID,
			Origin:    "http",
			Operation: c.Request.Method + " " + operationPath,
		}))

		c.Next()
	}
}

// AdminMiddleware 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, exists := GetAuthContext(c)
		if !exists {
			abortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}

		if authCtx.Role != model.UserRoleAdmin {
			abortWithError(c, http.StatusForbidden, "FORBIDDEN", "Admin permission required")
			return
		}

		c.Next()
	}
}

// GetAuthContext 从gin.Context获取认证上下文
func GetAuthContext(c *gin.Context) (*model.AuthContext, bool) {
	val, exists := c.Get(AuthContextKey)
	if !exists {
		return nil, false
	}

	authCtx, ok := val.(*model.AuthContext)
	if !ok {
		return nil, false
	}

	return authCtx, true
}
