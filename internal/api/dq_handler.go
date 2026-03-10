package api

import (
	"net/http"
	"strconv"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/service"
	"github.com/gin-gonic/gin"
)

// parseInt 解析字符串为整数
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// DQHandler 数据质量处理器
type DQHandler struct {
	dqService *service.DQService
}

// NewDQHandler 创建数据质量处理器
func NewDQHandler(dqService *service.DQService) *DQHandler {
	return &DQHandler{dqService: dqService}
}

// CreateRule 创建数据质量规则
func (h *DQHandler) CreateRule(c *gin.Context) {
	var req model.DQRuleRequest
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

	rule, err := h.dqService.CreateRule(c.Request.Context(), &req)
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
		Data:    rule,
	})
}

// ListRules 列出数据质量规则
func (h *DQHandler) ListRules(c *gin.Context) {
	filter := &model.DQRuleFilter{}

	if sourceID := c.Query("source_id"); sourceID != "" {
		filter.SourceID = &sourceID
	}
	if objectID := c.Query("object_id"); objectID != "" {
		filter.ObjectID = &objectID
	}
	if columnID := c.Query("column_id"); columnID != "" {
		filter.ColumnID = &columnID
	}
	if ruleType := c.Query("rule_type"); ruleType != "" {
		t := model.DQRuleType(ruleType)
		filter.RuleType = &t
	}
	if isActive := c.Query("is_active"); isActive != "" {
		active := isActive == "true"
		filter.IsActive = &active
	}

	rules, err := h.dqService.ListRules(c.Request.Context(), filter)
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
		Data:    rules,
	})
}

// GetRule 获取数据质量规则
func (h *DQHandler) GetRule(c *gin.Context) {
	ruleID := c.Param("id")

	rule, err := h.dqService.GetRule(c.Request.Context(), ruleID)
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
		Data:    rule,
	})
}

// UpdateRule 更新数据质量规则
func (h *DQHandler) UpdateRule(c *gin.Context) {
	ruleID := c.Param("id")

	var req model.DQRuleRequest
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

	if err := h.dqService.UpdateRule(c.Request.Context(), ruleID, &req); err != nil {
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

// DeleteRule 删除数据质量规则
func (h *DQHandler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("id")

	if err := h.dqService.DeleteRule(c.Request.Context(), ruleID); err != nil {
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

// CheckRules 执行数据质量检查
func (h *DQHandler) CheckRules(c *gin.Context) {
	var req model.DQCheckRequest
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

	resp, err := h.dqService.CheckRules(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "CHECK_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, model.BaseResponse{
		Success: true,
		Data:    resp,
	})
}

// GetResults 获取检测结果
func (h *DQHandler) GetResults(c *gin.Context) {
	var ruleID, batchID *string

	if r := c.Query("rule_id"); r != "" {
		ruleID = &r
	}
	if b := c.Query("batch_id"); b != "" {
		batchID = &b
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			if parsed > 100 {
				parsed = 100
			}
			limit = parsed
		}
	}

	results, err := h.dqService.GetResults(c.Request.Context(), ruleID, batchID, limit)
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
		Data:    results,
	})
}

// GetStats 获取统计信息
func (h *DQHandler) GetStats(c *gin.Context) {
	stats, err := h.dqService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponse{
			Success: false,
			Error: &model.ErrorInfo{
				Code:    "STATS_FAILED",
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
