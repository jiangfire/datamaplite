package api

import (
	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"github.com/gin-gonic/gin"
)

// SchemaHandler Schema处理器
type SchemaHandler struct {
	*Handler
	metadataService *service.MetadataService
}

// NewSchemaHandler 创建Schema处理器
func NewSchemaHandler(metadataService *service.MetadataService) *SchemaHandler {
	return &SchemaHandler{
		Handler:         NewHandler(),
		metadataService: metadataService,
	}
}

// GetColumnDetail 获取字段详情
func (h *SchemaHandler) GetColumnDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	detail, err := h.metadataService.GetColumnDetail(c.Request.Context(), id)
	if err != nil {
		h.NotFound(c, err.Error())
		return
	}

	h.JSON(c, detail)
}

// SearchColumns 搜索字段
func (h *SchemaHandler) SearchColumns(c *gin.Context) {
	var req model.SearchColumnsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	if req.Query == "" {
		h.BadRequest(c, "query parameter 'q' is required")
		return
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}

	results, err := h.metadataService.SearchColumns(c.Request.Context(), req.Query, req.Limit)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, results)
}

// GetColumnMappings 获取字段映射
func (h *SchemaHandler) GetColumnMappings(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	mappings, err := h.metadataService.GetColumnMappings(c.Request.Context(), id)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, mappings)
}

// CreateColumnMapping 创建字段映射
func (h *SchemaHandler) CreateColumnMapping(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	var req model.ColumnMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	req.SourceColumnID = id
	if err := h.metadataService.CreateColumnMapping(c.Request.Context(), &req); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, gin.H{"success": true})
}

// DeleteColumnMapping 删除字段映射
func (h *SchemaHandler) DeleteColumnMapping(c *gin.Context) {
	mappingID := c.Param("mappingId")
	if mappingID == "" {
		h.BadRequest(c, "mappingId is required")
		return
	}

	if err := h.metadataService.DeleteColumnMapping(c.Request.Context(), mappingID); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, gin.H{"success": true})
}

// GetLineage 获取血缘关系
func (h *SchemaHandler) GetLineage(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	lineage, err := h.metadataService.GetLineage(c.Request.Context(), id)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, lineage)
}

// GetImpactAnalysis 获取影响分析
func (h *SchemaHandler) GetImpactAnalysis(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	impact, err := h.metadataService.GetImpactAnalysis(c.Request.Context(), id)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, impact)
}
