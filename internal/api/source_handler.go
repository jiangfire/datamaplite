package api

import (
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/scanner"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"github.com/gin-gonic/gin"
)

// SourceHandler 数据源处理器
type SourceHandler struct {
	*Handler
	sourceService   SourceService
	metadataService MetadataService
}

// NewSourceHandler 创建数据源处理器
func NewSourceHandler(sourceService *service.SourceService, metadataService *service.MetadataService) *SourceHandler {
	return &SourceHandler{
		Handler:         NewHandler(),
		sourceService:   sourceService,
		metadataService: metadataService,
	}
}

// ListSources 列出所有数据源
func (h *SourceHandler) ListSources(c *gin.Context) {
	sources, err := h.sourceService.ListSources(c.Request.Context())
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}
	h.JSON(c, sources)
}

// CreateSource 创建数据源
func (h *SourceHandler) CreateSource(c *gin.Context) {
	var req model.CreateSourceRequest
	if !h.BindJSON(c, &req) {
		return
	}

	source, err := h.sourceService.CreateSource(c.Request.Context(), &req)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, source)
}

// GetSource 获取数据源详情
func (h *SourceHandler) GetSource(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	source, err := h.sourceService.GetSource(c.Request.Context(), id)
	if err != nil {
		h.NotFound(c, err.Error())
		return
	}

	h.JSON(c, source)
}

// UpdateSource 更新数据源
func (h *SourceHandler) UpdateSource(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	var req model.UpdateSourceRequest
	if !h.BindJSON(c, &req) {
		return
	}

	if err := h.sourceService.UpdateSource(c.Request.Context(), id, &req); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, gin.H{"message": "updated"})
}

// DeleteSource 删除数据源
func (h *SourceHandler) DeleteSource(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	if err := h.sourceService.DeleteSource(c.Request.Context(), id); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, gin.H{"message": "deleted"})
}

// TestConnection 测试连接
func (h *SourceHandler) TestConnection(c *gin.Context) {
	var req model.ConnectionTestRequest
	if !h.BindJSON(c, &req) {
		return
	}

	config := scanner.ConnectionConfig{
		Host:     req.Host,
		Port:     req.Port,
		Database: req.Database,
		Username: req.Username,
		Password: req.Password,
	}

	if err := h.sourceService.TestConnection(c.Request.Context(), string(req.Type), config); err != nil {
		h.JSON(c, model.ConnectionTestResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	h.JSON(c, model.ConnectionTestResponse{
		Success: true,
		Message: "Connection successful",
	})
}

// TriggerSync 触发同步
func (h *SourceHandler) TriggerSync(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	if err := h.sourceService.TriggerSync(c.Request.Context(), id); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, model.SyncResponse{
		SourceID:  id,
		StartedAt: time.Now(),
		ObjectsCount: 0,
	})
}

// GetSchemaTree 获取Schema树
func (h *SourceHandler) GetSchemaTree(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	tree, err := h.metadataService.GetSchemaTree(c.Request.Context(), id)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, tree)
}

// ListSchemaChanges 获取变更历史
func (h *SourceHandler) ListSchemaChanges(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	var query model.PaginationQuery
	h.BindQuery(c, &query)

	changes, err := h.metadataService.ListSchemaChanges(c.Request.Context(), id, query.GetLimit())
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, changes)
}
