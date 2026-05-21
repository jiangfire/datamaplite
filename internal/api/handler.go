package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jiangfire/datamaplite/internal/model"
	responsepkg "github.com/jiangfire/datamaplite/pkg/response"
)

var validate = validator.New()

func init() {
	_ = validate.RegisterValidation("hexcolor", func(fl validator.FieldLevel) bool {
		color := fl.Field().String()
		match, _ := regexp.MatchString(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`, color)
		return match
	})
}

// Handler 基础处理器
type Handler struct{}

// NewHandler 创建基础处理器
func NewHandler() *Handler {
	return &Handler{}
}

// JSON 返回JSON响应
func (h *Handler) JSON(c *gin.Context, data interface{}) {
	h.JSONWithStatus(c, http.StatusOK, data)
}

// JSONWithStatus 返回指定状态码的JSON响应
func (h *Handler) JSONWithStatus(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, responsepkg.Success(data))
}

// Success 返回无数据成功响应
func (h *Handler) Success(c *gin.Context) {
	c.JSON(http.StatusOK, responsepkg.Success(nil))
}

// Created 返回201响应
func (h *Handler) Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, responsepkg.SuccessWithMessage("created", data))
}

// Error 返回错误响应
func (h *Handler) Error(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, responsepkg.Error(statusCode, code, message))
}

// BadRequest 返回400错误
func (h *Handler) BadRequest(c *gin.Context, message string) {
	h.Error(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

// NotFound 返回404错误
func (h *Handler) NotFound(c *gin.Context, message string) {
	h.Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

// InternalError 返回500错误
func (h *Handler) InternalError(c *gin.Context, message string) {
	h.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// Unauthorized 返回401错误
func (h *Handler) Unauthorized(c *gin.Context, message string) {
	h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden 返回403错误
func (h *Handler) Forbidden(c *gin.Context, message string) {
	h.Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

// ServiceUnavailable 返回503错误
func (h *Handler) ServiceUnavailable(c *gin.Context, code string, message string) {
	h.Error(c, http.StatusServiceUnavailable, code, message)
}

// BindJSON 绑定并验证JSON请求
func (h *Handler) BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		h.BadRequest(c, "Invalid request body: "+err.Error())
		return false
	}
	if err := validate.Struct(obj); err != nil {
		h.BadRequest(c, "Validation failed: "+err.Error())
		return false
	}
	return true
}

// BindQuery 绑定查询参数
func (h *Handler) BindQuery(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		h.BadRequest(c, "Invalid query parameters: "+err.Error())
		return false
	}
	if err := validate.Struct(obj); err != nil {
		h.BadRequest(c, "Validation failed: "+err.Error())
		return false
	}
	return true
}

// RequireAuthContext 获取认证上下文，不存在时直接返回401响应。
func (h *Handler) RequireAuthContext(c *gin.Context) (*model.AuthContext, bool) {
	authCtx, exists := GetAuthContext(c)
	if !exists {
		h.Unauthorized(c, "User not authenticated")
		return nil, false
	}
	return authCtx, true
}
