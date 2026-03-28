package api

import (
	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"github.com/gin-gonic/gin"
)

// TagHandler 标签处理器
type TagHandler struct {
	*Handler
	tagService *service.TagService
}

// NewTagHandler 创建标签处理器
func NewTagHandler(tagService *service.TagService) *TagHandler {
	return &TagHandler{
		Handler:    NewHandler(),
		tagService: tagService,
	}
}

// ListTags 列出所有标签
func (h *TagHandler) ListTags(c *gin.Context) {
	tags, err := h.tagService.ListTags(c.Request.Context())
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, tags)
}

// CreateTag 创建标签
func (h *TagHandler) CreateTag(c *gin.Context) {
	var req model.TagRequest
	if !h.BindJSON(c, &req) {
		return
	}

	tag, err := h.tagService.CreateTag(c.Request.Context(), &req)
	if err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	h.Created(c, tag)
}

// GetTag 获取标签详情
func (h *TagHandler) GetTag(c *gin.Context) {
	id := c.Param("id")

	tag, err := h.tagService.GetTag(c.Request.Context(), id)
	if err != nil {
		h.NotFound(c, err.Error())
		return
	}

	h.JSON(c, tag)
}

// UpdateTag 更新标签
func (h *TagHandler) UpdateTag(c *gin.Context) {
	id := c.Param("id")

	var req model.TagRequest
	if !h.BindJSON(c, &req) {
		return
	}

	if err := h.tagService.UpdateTag(c.Request.Context(), id, &req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	h.Success(c)
}

// DeleteTag 删除标签
func (h *TagHandler) DeleteTag(c *gin.Context) {
	id := c.Param("id")

	if err := h.tagService.DeleteTag(c.Request.Context(), id); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	h.Success(c)
}

// GetColumnsByTag 获取带有指定标签的字段列表
func (h *TagHandler) GetColumnsByTag(c *gin.Context) {
	id := c.Param("id")

	columns, err := h.tagService.GetColumnsByTag(c.Request.Context(), id)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, columns)
}

// AssignTagsToColumn 为字段分配标签
func (h *TagHandler) AssignTagsToColumn(c *gin.Context) {
	columnID := c.Param("id")

	var req struct {
		TagID  string   `json:"tag_id"`
		TagIDs []string `json:"tag_ids"`
	}
	if !h.BindJSON(c, &req) {
		return
	}

	if req.TagID != "" {
		req.TagIDs = append(req.TagIDs, req.TagID)
	}
	if len(req.TagIDs) == 0 {
		h.BadRequest(c, "tag_id or tag_ids is required")
		return
	}

	for _, tagID := range req.TagIDs {
		if err := h.tagService.AddTagToColumn(c.Request.Context(), columnID, tagID); err != nil {
			h.InternalError(c, err.Error())
			return
		}
	}

	h.Success(c)
}

// RemoveTagFromColumn 从字段移除标签
func (h *TagHandler) RemoveTagFromColumn(c *gin.Context) {
	columnID := c.Param("id")
	tagID := c.Param("tagId")

	if columnID == "" || tagID == "" {
		h.BadRequest(c, "column id and tag id are required")
		return
	}

	if err := h.tagService.RemoveTagFromColumn(c.Request.Context(), columnID, tagID); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.Success(c)
}

// GetColumnTags 获取字段的所有标签
func (h *TagHandler) GetColumnTags(c *gin.Context) {
	columnID := c.Param("id")

	tags, err := h.tagService.GetColumnTags(c.Request.Context(), columnID)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, tags)
}
