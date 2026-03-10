package api

import (
	"net/http"
	"strings"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	// AuthContextKey 认证上下文在gin.Context中的key
	AuthContextKey = "auth_context"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header中获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.BaseResponse{
				Success: false,
				Error: &model.ErrorInfo{
					Code:    "UNAUTHORIZED",
					Message: "Authorization header required",
				},
			})
			c.Abort()
			return
		}

		// 解析Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, model.BaseResponse{
				Success: false,
				Error: &model.ErrorInfo{
					Code:    "UNAUTHORIZED",
					Message: "Invalid authorization header format",
				},
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证Token
		claims, err := authService.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.BaseResponse{
				Success: false,
				Error: &model.ErrorInfo{
					Code:    "UNAUTHORIZED",
					Message: "Invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		// 检查Token类型
		if claims.TokenType != "access" {
			c.JSON(http.StatusUnauthorized, model.BaseResponse{
				Success: false,
				Error: &model.ErrorInfo{
					Code:    "UNAUTHORIZED",
					Message: "Invalid token type",
				},
			})
			c.Abort()
			return
		}

		// 将认证信息存入上下文
		authCtx := &model.AuthContext{
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		}
		c.Set(AuthContextKey, authCtx)

		c.Next()
	}
}

// AdminMiddleware 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, exists := GetAuthContext(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, model.BaseResponse{
				Success: false,
				Error: &model.ErrorInfo{
					Code:    "UNAUTHORIZED",
					Message: "Authentication required",
				},
			})
			c.Abort()
			return
		}

		if authCtx.Role != model.UserRoleAdmin {
			c.JSON(http.StatusForbidden, model.BaseResponse{
				Success: false,
				Error: &model.ErrorInfo{
					Code:    "FORBIDDEN",
					Message: "Admin permission required",
				},
			})
			c.Abort()
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
