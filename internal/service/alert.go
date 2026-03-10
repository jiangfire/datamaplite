package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"go.uber.org/zap"
)

// AlertService 告警服务
type AlertService struct {
	store  store.Store
	logger *zap.Logger
	client *http.Client
}

// NewAlertService 创建告警服务
func NewAlertService(store store.Store, logger *zap.Logger) *AlertService {
	return &AlertService{
		store:  store,
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateAlertRule 创建告警规则
func (s *AlertService) CreateAlertRule(ctx context.Context, req *model.AlertRuleRequest) (*model.AlertRuleResponse, error) {
	// 验证SourceID
	if req.SourceID != nil {
		_, err := s.store.GetDataSource(ctx, *req.SourceID)
		if err != nil {
			return nil, fmt.Errorf("data source not found: %w", err)
		}
	}

	// 验证ObjectID
	if req.ObjectID != nil {
		_, err := s.store.GetSchemaObject(ctx, *req.ObjectID)
		if err != nil {
			return nil, fmt.Errorf("schema object not found: %w", err)
		}
	}

	create := &store.AlertRuleCreate{
		SourceID:      req.SourceID,
		ObjectID:      req.ObjectID,
		Name:          req.Name,
		Description:   &req.Description,
		ChangeTypes:   req.ChangeTypes,
		NotifyWebhook: req.NotifyWebhook,
		WebhookURL:    &req.WebhookURL,
		NotifyInApp:   req.NotifyInApp,
		IsActive:      req.IsActive,
	}

	id, err := s.store.CreateAlertRule(ctx, create)
	if err != nil {
		s.logger.Error("failed to create alert rule", zap.Error(err))
		return nil, fmt.Errorf("failed to create alert rule: %w", err)
	}

	rule, err := s.store.GetAlertRule(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toAlertRuleResponse(rule), nil
}

// GetAlertRule 获取告警规则
func (s *AlertService) GetAlertRule(ctx context.Context, id string) (*model.AlertRuleResponse, error) {
	rule, err := s.store.GetAlertRule(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("alert rule not found: %w", err)
	}
	return s.toAlertRuleResponse(rule), nil
}

// ListAlertRules 列出告警规则
func (s *AlertService) ListAlertRules(ctx context.Context, sourceID *string) ([]*model.AlertRuleResponse, error) {
	rules, err := s.store.ListAlertRules(ctx, sourceID)
	if err != nil {
		s.logger.Error("failed to list alert rules", zap.Error(err))
		return nil, err
	}

	var responses []*model.AlertRuleResponse
	for _, rule := range rules {
		responses = append(responses, s.toAlertRuleResponse(rule))
	}
	return responses, nil
}

// UpdateAlertRule 更新告警规则
func (s *AlertService) UpdateAlertRule(ctx context.Context, id string, req *model.AlertRuleRequest) error {
	// 检查规则是否存在
	_, err := s.store.GetAlertRule(ctx, id)
	if err != nil {
		return fmt.Errorf("alert rule not found: %w", err)
	}

	// 验证SourceID
	if req.SourceID != nil {
		_, err := s.store.GetDataSource(ctx, *req.SourceID)
		if err != nil {
			return fmt.Errorf("data source not found: %w", err)
		}
	}

	// 验证ObjectID
	if req.ObjectID != nil {
		_, err := s.store.GetSchemaObject(ctx, *req.ObjectID)
		if err != nil {
			return fmt.Errorf("schema object not found: %w", err)
		}
	}

	updates := &store.AlertRuleUpdate{
		Name:          &req.Name,
		Description:   &req.Description,
		ChangeTypes:   &req.ChangeTypes,
		NotifyWebhook: &req.NotifyWebhook,
		NotifyInApp:   &req.NotifyInApp,
		IsActive:      &req.IsActive,
	}

	if req.SourceID != nil {
		updates.SourceID = req.SourceID
	}
	if req.ObjectID != nil {
		updates.ObjectID = req.ObjectID
	}
	if req.WebhookURL != "" {
		updates.WebhookURL = &req.WebhookURL
	} else {
		emptyURL := ""
		updates.WebhookURL = &emptyURL
	}

	if err := s.store.UpdateAlertRule(ctx, id, updates); err != nil {
		s.logger.Error("failed to update alert rule", zap.Error(err))
		return fmt.Errorf("failed to update alert rule: %w", err)
	}

	return nil
}

// DeleteAlertRule 删除告警规则
func (s *AlertService) DeleteAlertRule(ctx context.Context, id string) error {
	// 检查规则是否存在
	_, err := s.store.GetAlertRule(ctx, id)
	if err != nil {
		return fmt.Errorf("alert rule not found: %w", err)
	}

	if err := s.store.DeleteAlertRule(ctx, id); err != nil {
		s.logger.Error("failed to delete alert rule", zap.Error(err))
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	return nil
}

// SchemaChangeInfo Schema变更信息
type SchemaChangeInfo struct {
	ID         string
	SourceID   string
	ObjectID   *string
	ChangeType string
	ObjectType string
	ObjectName string
	OldValue   *string
	NewValue   *string
	DetectedAt string
}

// ProcessSchemaChange 处理Schema变更，触发告警
func (s *AlertService) ProcessSchemaChange(ctx context.Context, change *SchemaChangeInfo) error {
	// 查找匹配的告警规则
	rules, err := s.store.ListMatchingAlertRules(ctx, change.SourceID, change.ChangeType)
	if err != nil {
		s.logger.Error("failed to list matching alert rules", zap.Error(err))
		return err
	}

	if len(rules) == 0 {
		s.logger.Debug("no matching alert rules for change", zap.String("changeType", change.ChangeType))
		return nil
	}

	// 获取数据源信息
	source, err := s.store.GetDataSource(ctx, change.SourceID)
	if err != nil {
		s.logger.Error("failed to get data source", zap.Error(err))
		return err
	}

	// 为每个匹配的规则创建通知
	for _, rule := range rules {
		// 检查是否指定了对象，变更是否匹配
		if rule.ObjectID != nil && (change.ObjectID == nil || *rule.ObjectID != *change.ObjectID) {
			continue
		}

		if err := s.createNotification(ctx, rule, change, source); err != nil {
			s.logger.Error("failed to create notification",
				zap.String("ruleId", rule.ID),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// createNotification 创建通知
func (s *AlertService) createNotification(ctx context.Context, rule *store.AlertRuleRow, change *SchemaChangeInfo, source *store.DataSourceRow) error {
	// 构建通知标题和消息
	title := s.buildNotificationTitle(change, source)
	message := s.buildNotificationMessage(change, source)

	create := &store.NotificationCreate{
		RuleID:     &rule.ID,
		ChangeID:   change.ID,
		SourceID:   change.SourceID,
		Title:      title,
		Message:    message,
		ChangeType: change.ChangeType,
		ObjectType: change.ObjectType,
		ObjectName: change.ObjectName,
		OldValue:   change.OldValue,
		NewValue:   change.NewValue,
	}

	notificationID, err := s.store.CreateNotification(ctx, create)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	s.logger.Info("notification created",
		zap.String("notificationId", notificationID),
		zap.String("ruleId", rule.ID))

	// 发送Webhook通知
	if rule.NotifyWebhook && rule.WebhookURL != nil && *rule.WebhookURL != "" {
		if err := s.sendWebhook(ctx, rule, change, source, notificationID); err != nil {
			s.logger.Error("failed to send webhook", zap.Error(err))
			// 记录Webhook失败但不影响主流程
			s.store.UpdateNotificationWebhookStatus(ctx, notificationID, false, &[]string{err.Error()}[0])
		} else {
			s.store.UpdateNotificationWebhookStatus(ctx, notificationID, true, nil)
		}
	}

	return nil
}

// sendWebhook 发送Webhook通知
func (s *AlertService) sendWebhook(ctx context.Context, rule *store.AlertRuleRow, change *SchemaChangeInfo, source *store.DataSourceRow, notificationID string) error {
	if rule.WebhookURL == nil || *rule.WebhookURL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	payload := &model.WebhookPayload{
		ID:         notificationID,
		Event:      "schema_change",
		Timestamp:  time.Now(),
		ChangeType: change.ChangeType,
		ObjectType: change.ObjectType,
		ObjectName: change.ObjectName,
		SourceID:   change.SourceID,
		SourceName: source.Name,
		OldValue:   change.OldValue,
		NewValue:   change.NewValue,
		Message:    s.buildNotificationMessage(change, source),
		RuleID:     &rule.ID,
		RuleName:   rule.Name,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", *rule.WebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DataMap-Event", "schema_change")
	req.Header.Set("X-DataMap-Signature", "") // 可添加签名验证

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	s.logger.Info("webhook sent successfully",
		zap.String("notificationId", notificationID),
		zap.String("webhookUrl", *rule.WebhookURL))

	return nil
}

// buildNotificationTitle 构建通知标题
func (s *AlertService) buildNotificationTitle(change *SchemaChangeInfo, source *store.DataSourceRow) string {
	switch change.ChangeType {
	case "add_object":
		return fmt.Sprintf("[%s] 新增对象: %s", source.Name, change.ObjectName)
	case "drop_object":
		return fmt.Sprintf("[%s] 删除对象: %s", source.Name, change.ObjectName)
	case "add_column":
		return fmt.Sprintf("[%s] 新增字段: %s", source.Name, change.ObjectName)
	case "drop_column":
		return fmt.Sprintf("[%s] 删除字段: %s", source.Name, change.ObjectName)
	case "alter_column":
		return fmt.Sprintf("[%s] 修改字段: %s", source.Name, change.ObjectName)
	case "change_type":
		return fmt.Sprintf("[%s] 类型变更: %s", source.Name, change.ObjectName)
	default:
		return fmt.Sprintf("[%s] Schema变更: %s", source.Name, change.ObjectName)
	}
}

// buildNotificationMessage 构建通知消息
func (s *AlertService) buildNotificationMessage(change *SchemaChangeInfo, source *store.DataSourceRow) string {
	var details string
	if change.OldValue != nil && change.NewValue != nil {
		details = fmt.Sprintf("\n变更详情: %s → %s", *change.OldValue, *change.NewValue)
	} else if change.NewValue != nil {
		details = fmt.Sprintf("\n新值: %s", *change.NewValue)
	}
	return fmt.Sprintf("数据源 [%s] 发生 %s 变更: %s%s", source.Name, change.ChangeType, change.ObjectName, details)
}

// toAlertRuleResponse 转换为响应模型
func (s *AlertService) toAlertRuleResponse(rule *store.AlertRuleRow) *model.AlertRuleResponse {
	resp := &model.AlertRuleResponse{
		ID:            rule.ID,
		SourceID:      rule.SourceID,
		ObjectID:      rule.ObjectID,
		SourceName:    rule.SourceName,
		ObjectName:    rule.ObjectName,
		Name:          rule.Name,
		ChangeTypes:   rule.ChangeTypes,
		NotifyWebhook: rule.NotifyWebhook,
		NotifyInApp:   rule.NotifyInApp,
		IsActive:      rule.IsActive,
		CreatedAt:     rule.CreatedAt,
		UpdatedAt:     rule.UpdatedAt,
	}
	if rule.Description != nil {
		resp.Description = *rule.Description
	}
	if rule.WebhookURL != nil {
		resp.WebhookURL = *rule.WebhookURL
	}
	return resp
}

// NotificationService 通知服务
type NotificationService struct {
	store  store.Store
	logger *zap.Logger
}

// NewNotificationService 创建通知服务
func NewNotificationService(store store.Store, logger *zap.Logger) *NotificationService {
	return &NotificationService{
		store:  store,
		logger: logger,
	}
}

// ListNotifications 列出用户的通知
func (s *NotificationService) ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*model.NotificationResponse, error) {
	notifications, err := s.store.ListNotifications(ctx, userID, unreadOnly, limit)
	if err != nil {
		s.logger.Error("failed to list notifications", zap.Error(err))
		return nil, err
	}

	var responses []*model.NotificationResponse
	for _, n := range notifications {
		responses = append(responses, s.toNotificationResponse(n))
	}
	return responses, nil
}

// GetNotificationStats 获取通知统计
func (s *NotificationService) GetNotificationStats(ctx context.Context, userID string) (*model.NotificationStats, error) {
	stats, err := s.store.GetNotificationStats(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get notification stats", zap.Error(err))
		return nil, err
	}

	return &model.NotificationStats{
		TotalCount:  stats.TotalCount,
		UnreadCount: stats.UnreadCount,
		TodayCount:  stats.TodayCount,
	}, nil
}

// MarkAsRead 标记通知已读
func (s *NotificationService) MarkAsRead(ctx context.Context, userID string, notificationID string) error {
	if err := s.store.MarkNotificationAsRead(ctx, userID, notificationID); err != nil {
		s.logger.Error("failed to mark notification as read", zap.Error(err))
		return err
	}
	return nil
}

// MarkAllAsRead 标记所有通知已读
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	if err := s.store.MarkAllNotificationsAsRead(ctx, userID); err != nil {
		s.logger.Error("failed to mark all notifications as read", zap.Error(err))
		return err
	}
	return nil
}

// toNotificationResponse 转换为响应模型
func (s *NotificationService) toNotificationResponse(n *store.NotificationRow) *model.NotificationResponse {
	resp := &model.NotificationResponse{
		ID:          n.ID,
		RuleID:      n.RuleID,
		ChangeID:    n.ChangeID,
		SourceID:    n.SourceID,
		SourceName:  n.SourceName,
		Title:       n.Title,
		Message:     n.Message,
		ChangeType:  n.ChangeType,
		ObjectType:  n.ObjectType,
		ObjectName:  n.ObjectName,
		WebhookSent: n.WebhookSent,
		IsRead:      n.IsRead,
		CreatedAt:   n.CreatedAt,
	}
	if n.RuleName != nil {
		resp.RuleName = *n.RuleName
	}
	if n.OldValue != nil {
		resp.OldValue = *n.OldValue
	}
	if n.NewValue != nil {
		resp.NewValue = *n.NewValue
	}
	if n.WebhookError != nil {
		resp.WebhookError = *n.WebhookError
	}
	return resp
}
