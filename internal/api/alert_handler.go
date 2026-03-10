package api

import (
	"net/http"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	alertService *service.AlertService
	notifService *service.NotificationService
	logger       *zap.Logger
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler(alertService *service.AlertService, notifService *service.NotificationService, logger *zap.Logger) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
		notifService: notifService,
		logger:       logger,
	}
}

// CreateAlertRule 创建告警规则
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
	var req model.AlertRuleRequest
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

	rule, err := h.alertService.CreateAlertRule(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create alert rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, model.BaseResponse{
		Success: true,
		Data:    rule,
	})
}

// GetAlertRule 获取告警规则详情
func (h *AlertHandler) GetAlertRule(c *gin.Context) {
	id := c.Param("id")

	rule, err := h.alertService.GetAlertRule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "NOT_FOUND",
				Message: "Alert rule not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    rule,
	})
}

// ListAlertRules 列出告警规则
func (h *AlertHandler) ListAlertRules(c *gin.Context) {
	var sourceID *string
	if sid := c.Query("source_id"); sid != "" {
		sourceID = &sid
	}

	rules, err := h.alertService.ListAlertRules(c.Request.Context(), sourceID)
	if err != nil {
		h.logger.Error("failed to list alert rules", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    rules,
	})
}

// UpdateAlertRule 更新告警规则
func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
	id := c.Param("id")

	var req model.AlertRuleRequest
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

	if err := h.alertService.UpdateAlertRule(c.Request.Context(), id, &req); err != nil {
		h.logger.Error("failed to update alert rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
	})
}

// DeleteAlertRule 删除告警规则
func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
	id := c.Param("id")

	if err := h.alertService.DeleteAlertRule(c.Request.Context(), id); err != nil {
		h.logger.Error("failed to delete alert rule", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
	})
}

// NotificationHandler 通知处理器
type NotificationHandler struct {
	notifService *service.NotificationService
	logger       *zap.Logger
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(notifService *service.NotificationService, logger *zap.Logger) *NotificationHandler {
	return &NotificationHandler{
		notifService: notifService,
		logger:       logger,
	}
}

// ListNotifications 列出通知
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
		})
		return
	}

	unreadOnly := c.Query("unread_only") == "true"
	limit := 50

	notifications, err := h.notifService.ListNotifications(c.Request.Context(), userID, unreadOnly, limit)
	if err != nil {
		h.logger.Error("failed to list notifications", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    notifications,
	})
}

// GetNotificationStats 获取通知统计
func (h *NotificationHandler) GetNotificationStats(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
		})
		return
	}

	stats, err := h.notifService.GetNotificationStats(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get notification stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    stats,
	})
}

// MarkAsRead 标记通知已读
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
		})
		return
	}

	var req model.MarkAsReadRequest
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

	if req.MarkAll {
		if err := h.notifService.MarkAllAsRead(c.Request.Context(), userID); err != nil {
			h.logger.Error("failed to mark all as read", zap.Error(err))
			c.JSON(http.StatusInternalServerError, model.BaseResponse{
				Success: false,
				Error: &model.ErrorInfo{
					Code:    "INTERNAL_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
	} else if len(req.NotificationIDs) > 0 {
		for _, id := range req.NotificationIDs {
			if err := h.notifService.MarkAsRead(c.Request.Context(), userID, id); err != nil {
				h.logger.Error("failed to mark as read", zap.Error(err))
				c.JSON(http.StatusInternalServerError, model.BaseResponse{
					Success: false,
					Error: &model.ErrorInfo{
						Code:    "INTERNAL_ERROR",
						Message: err.Error(),
					},
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
	})
}
