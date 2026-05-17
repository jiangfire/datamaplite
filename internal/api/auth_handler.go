package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/service"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	*Handler
	authService *service.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		Handler:     NewHandler(),
		authService: authService,
	}
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	if !h.authService.IsEnabled() {
		h.ServiceUnavailable(c, "AUTH_DISABLED", "Authentication is disabled")
		return
	}

	var req model.LoginRequest
	if !h.BindJSON(c, &req) {
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		h.Unauthorized(c, err.Error())
		return
	}

	h.JSON(c, resp)
}

// RefreshToken 刷新Token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	if !h.authService.IsEnabled() {
		h.ServiceUnavailable(c, "AUTH_DISABLED", "Authentication is disabled")
		return
	}

	var req model.RefreshTokenRequest
	if !h.BindJSON(c, &req) {
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.Unauthorized(c, err.Error())
		return
	}

	h.JSON(c, resp)
}

// Register 用户注册（需要管理员权限）
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if !h.BindJSON(c, &req) {
		return
	}

	user, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	h.Created(c, user)
}

// GetCurrentUser 获取当前用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	if !h.authService.IsEnabled() {
		h.ServiceUnavailable(c, "AUTH_DISABLED", "Authentication is disabled")
		return
	}

	authCtx, ok := h.RequireAuthContext(c)
	if !ok {
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), authCtx.UserID)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, user)
}
