package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/service"
	"github.com/jiangfire/datamaplite/pkg/response"
)

// DashboardHandler 仪表盘 HTTP 处理器
type DashboardHandler struct {
	dashboardService *service.DashboardService
}

// NewDashboardHandler 创建仪表盘处理器
func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetStats 获取仪表盘统计数据
func (h *DashboardHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.dashboardService.GetStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(stats))
}
