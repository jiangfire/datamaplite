package api

import (
	"net/http"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Handler 基础处理器
type Handler struct{}

// NewHandler 创建基础处理器
func NewHandler() *Handler {
	return &Handler{}
}

// JSON 返回JSON响应
func (h *Handler) JSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    data,
	})
}

// Error 返回错误响应
func (h *Handler) Error(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, model.BaseResponse{
		Success: false,
		Error: &model.ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
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
	return true
}
