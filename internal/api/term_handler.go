package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/service"
)

// TermHandler 业务术语处理器
type TermHandler struct {
	*Handler
	termService TermService
	ddlService  DDLService
}

// NewTermHandler 创建业务术语处理器
func NewTermHandler(termService *service.TermService, ddlService *service.DDLService) *TermHandler {
	return &TermHandler{
		Handler:     NewHandler(),
		termService: termService,
		ddlService:  ddlService,
	}
}

// CreateTerm 创建业务术语
func (h *TermHandler) CreateTerm(c *gin.Context) {
	var req model.BusinessTermRequest
	if !h.BindJSON(c, &req) {
		return
	}

	term, err := h.termService.CreateTerm(c.Request.Context(), &req)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, term)
}

// ListTerms 列出业务术语
func (h *TermHandler) ListTerms(c *gin.Context) {
	category := c.Query("category")

	terms, err := h.termService.ListTerms(c.Request.Context(), category)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, terms)
}

// GetTerm 获取业务术语
func (h *TermHandler) GetTerm(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	term, err := h.termService.GetTerm(c.Request.Context(), id)
	if err != nil {
		h.NotFound(c, err.Error())
		return
	}

	h.JSON(c, term)
}

// UpdateTerm 更新业务术语
func (h *TermHandler) UpdateTerm(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	var req model.BusinessTermRequest
	if !h.BindJSON(c, &req) {
		return
	}

	if err := h.termService.UpdateTerm(c.Request.Context(), id, &req); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.Success(c)
}

// DeleteTerm 删除业务术语
func (h *TermHandler) DeleteTerm(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.BadRequest(c, "id is required")
		return
	}

	if err := h.termService.DeleteTerm(c.Request.Context(), id); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.Success(c)
}

// AssignTermToColumn 分配术语到字段
func (h *TermHandler) AssignTermToColumn(c *gin.Context) {
	columnID := c.Param("id")
	if columnID == "" {
		h.BadRequest(c, "column id is required")
		return
	}

	var req model.AssignTermRequest
	if !h.BindJSON(c, &req) {
		return
	}

	if err := h.termService.AssignTermToColumn(c.Request.Context(), columnID, &req); err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.Success(c)
}

// GenerateDDL 生成DDL
func (h *TermHandler) GenerateDDL(c *gin.Context) {
	var req model.DDLGenerateRequest
	if !h.BindJSON(c, &req) {
		return
	}

	resp, err := h.ddlService.GenerateDDL(c.Request.Context(), req.ObjectID, req.TargetType)
	if err != nil {
		h.InternalError(c, err.Error())
		return
	}

	h.JSON(c, resp)
}
