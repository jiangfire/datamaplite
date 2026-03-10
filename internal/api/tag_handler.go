package api

import (
	"net/http"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"github.com/gin-gonic/gin"
)

// TagHandler 标签处理器
type TagHandler struct {
	tagService *service.TagService
}

// NewTagHandler 创建标签处理器
func NewTagHandler(tagService *service.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

// ListTags 列出所有标签
func (h *TagHandler) ListTags(c *gin.Context) {
	tags, err := h.tagService.ListTags(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "LIST_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    tags,
	})
}

// CreateTag 创建标签
func (h *TagHandler) CreateTag(c *gin.Context) {
	var req model.TagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	tag, err := h.tagService.CreateTag(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "CREATE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, model.BaseResponse{
		Success: true,
		Data:    tag,
	})
}

// GetTag 获取标签详情
func (h *TagHandler) GetTag(c *gin.Context) {
	id := c.Param("id")

	tag, err := h.tagService.GetTag(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "NOT_FOUND",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    tag,
	})
}

// UpdateTag 更新标签
func (h *TagHandler) UpdateTag(c *gin.Context) {
	id := c.Param("id")

	var req model.TagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	if err := h.tagService.UpdateTag(c.Request.Context(), id, &req); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "UPDATE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
	})
}

// DeleteTag 删除标签
func (h *TagHandler) DeleteTag(c *gin.Context) {
	id := c.Param("id")

	if err := h.tagService.DeleteTag(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "DELETE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
	})
}

// GetColumnsByTag 获取带有指定标签的字段列表
func (h *TagHandler) GetColumnsByTag(c *gin.Context) {
	id := c.Param("id")

	columns, err := h.tagService.GetColumnsByTag(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "LIST_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    columns,
	})
}
