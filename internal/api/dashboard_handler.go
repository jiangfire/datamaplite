package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/service"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/jiangfire/datamaplite/pkg/response"
)

// DashboardHandler 仪表盘 HTTP 处理器
type DashboardHandler struct {
	dashboardService *service.DashboardService
	store            store.Store
}

// NewDashboardHandler 创建仪表盘处理器
func NewDashboardHandler(dashboardService *service.DashboardService, st store.Store) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		store:            st,
	}
}

// GetStats 获取仪表盘统计数据
func (h *DashboardHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	userID := ""
	if authCtx, exists := GetAuthContext(c); exists {
		userID = authCtx.UserID
	}

	stats, err := h.dashboardService.GetStats(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(stats))
}

// GetChangeTrend 获取变更趋势数据
func (h *DashboardHandler) GetChangeTrend(c *gin.Context) {
	ctx := c.Request.Context()

	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	trend, err := h.store.GetChangeTrend(ctx, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(trend))
}
