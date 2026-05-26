package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/datamaplite/internal/service"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/jiangfire/datamaplite/pkg/response"
)

// SyncScheduleHandler 定时同步配置 HTTP 处理器
type SyncScheduleHandler struct {
	store            store.Store
	schedulerService *service.SyncSchedulerService
}

// NewSyncScheduleHandler 创建定时同步配置处理器
func NewSyncScheduleHandler(st store.Store, schedulerService *service.SyncSchedulerService) *SyncScheduleHandler {
	return &SyncScheduleHandler{
		store:            st,
		schedulerService: schedulerService,
	}
}

// RegisterRoutes 注册路由
func (h *SyncScheduleHandler) RegisterRoutes(router *gin.RouterGroup) {
	schedules := router.Group("/sync/schedules")
	{
		schedules.GET("", h.ListSchedules)
		schedules.POST("", h.CreateSchedule)
		schedules.GET("/:id", h.GetSchedule)
		schedules.PUT("/:id", h.UpdateSchedule)
		schedules.DELETE("/:id", h.DeleteSchedule)
	}
}

// SyncScheduleCreateRequest 创建请求
type SyncScheduleCreateRequest struct {
	SourceID       string `json:"source_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	CronExpression string `json:"cron_expression" binding:"required"`
	IsActive       bool   `json:"is_active"`
}

// SyncScheduleUpdateRequest 更新请求
type SyncScheduleUpdateRequest struct {
	Name           *string `json:"name"`
	Description    *string `json:"description"`
	CronExpression *string `json:"cron_expression"`
	IsActive       *bool   `json:"is_active"`
}

// ListSchedules 列出所有定时同步配置
func (h *SyncScheduleHandler) ListSchedules(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := h.store.ListSyncSchedules(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(rows))
}

// GetSchedule 获取单个定时同步配置
func (h *SyncScheduleHandler) GetSchedule(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	row, err := h.store.GetSyncSchedule(ctx, id)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, response.Error(http.StatusNotFound, "NOT_FOUND", "sync schedule not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(row))
}

// CreateSchedule 创建定时同步配置
func (h *SyncScheduleHandler) CreateSchedule(c *gin.Context) {
	ctx := c.Request.Context()
	var req SyncScheduleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(http.StatusBadRequest, "BAD_REQUEST", err.Error()))
		return
	}

	// 验证数据源是否存在
	if _, err := h.store.GetDataSource(ctx, req.SourceID); err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusBadRequest, response.Error(http.StatusBadRequest, "BAD_REQUEST", "source not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}

	// 验证 cron 表达式
	if err := service.ValidateCronExpression(req.CronExpression); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(http.StatusBadRequest, "BAD_REQUEST", "invalid cron expression: "+err.Error()))
		return
	}

	id, err := h.store.CreateSyncSchedule(ctx, &store.SyncScheduleCreate{
		SourceID:       req.SourceID,
		Name:           req.Name,
		Description:    strPtr(req.Description),
		CronExpression: req.CronExpression,
		IsActive:       req.IsActive,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}

	// 刷新调度器
	if h.schedulerService != nil {
		if err := h.schedulerService.RefreshSchedules(ctx); err != nil {
			// 不阻断创建成功，只记录日志
			_ = err // TODO: add logging
		}
	}

	c.JSON(http.StatusCreated, response.Success(gin.H{"id": id}))
}

// UpdateSchedule 更新定时同步配置
func (h *SyncScheduleHandler) UpdateSchedule(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	var req SyncScheduleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(http.StatusBadRequest, "BAD_REQUEST", err.Error()))
		return
	}

	// 如果更新了 cron 表达式，验证其有效性
	if req.CronExpression != nil && *req.CronExpression != "" {
		if err := service.ValidateCronExpression(*req.CronExpression); err != nil {
			c.JSON(http.StatusBadRequest, response.Error(http.StatusBadRequest, "BAD_REQUEST", "invalid cron expression: "+err.Error()))
			return
		}
	}

	if err := h.store.UpdateSyncSchedule(ctx, id, &store.SyncScheduleUpdate{
		Name:           req.Name,
		Description:    req.Description,
		CronExpression: req.CronExpression,
		IsActive:       req.IsActive,
	}); err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, response.Error(http.StatusNotFound, "NOT_FOUND", "sync schedule not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}

	// 刷新调度器
	if h.schedulerService != nil {
		if err := h.schedulerService.RefreshSchedules(ctx); err != nil {
			_ = err // 不阻断更新成功
		}
	}

	c.JSON(http.StatusOK, response.Success(nil))
}

// DeleteSchedule 删除定时同步配置
func (h *SyncScheduleHandler) DeleteSchedule(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	if err := h.store.DeleteSyncSchedule(ctx, id); err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, response.Error(http.StatusNotFound, "NOT_FOUND", "sync schedule not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(http.StatusInternalServerError, "INTERNAL_ERROR", err.Error()))
		return
	}

	// 刷新调度器
	if h.schedulerService != nil {
		if err := h.schedulerService.RefreshSchedules(ctx); err != nil {
			_ = err // 不阻断删除成功
		}
	}

	c.JSON(http.StatusOK, response.Success(nil))
}

// isNotFoundError 检查错误是否为 not found 类型
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found")
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
