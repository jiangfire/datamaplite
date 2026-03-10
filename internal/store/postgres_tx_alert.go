package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CreateAlertRule 创建告警规则
func (t *PostgresTxStore) CreateAlertRule(ctx context.Context, rule *AlertRuleCreate) (string, error) {
	query := `
		INSERT INTO alert_rules (id, source_id, object_id, name, description, change_types, notify_webhook, webhook_url, notify_in_app, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, rule.SourceID, rule.ObjectID, rule.Name, rule.Description,
		rule.ChangeTypes, rule.NotifyWebhook, rule.WebhookURL, rule.NotifyInApp, rule.IsActive)
	if err != nil {
		return "", fmt.Errorf("failed to create alert rule: %w", err)
	}
	return id, nil
}

// GetAlertRule 获取告警规则
func (t *PostgresTxStore) GetAlertRule(ctx context.Context, id string) (*AlertRuleRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// ListAlertRules 列出告警规则
func (t *PostgresTxStore) ListAlertRules(ctx context.Context, sourceID *string) ([]*AlertRuleRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// UpdateAlertRule 更新告警规则
func (t *PostgresTxStore) UpdateAlertRule(ctx context.Context, id string, updates *AlertRuleUpdate) error {
	query := `UPDATE alert_rules SET updated_at = NOW()`
	var args []interface{}
	argCount := 1

	if updates.Name != nil {
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *updates.Name)
		argCount++
	}
	if updates.Description != nil {
		query += fmt.Sprintf(", description = $%d", argCount)
		args = append(args, *updates.Description)
		argCount++
	}
	if updates.SourceID != nil {
		query += fmt.Sprintf(", source_id = $%d", argCount)
		args = append(args, *updates.SourceID)
		argCount++
	}
	if updates.ObjectID != nil {
		query += fmt.Sprintf(", object_id = $%d", argCount)
		args = append(args, *updates.ObjectID)
		argCount++
	}
	if updates.ChangeTypes != nil {
		query += fmt.Sprintf(", change_types = $%d", argCount)
		args = append(args, *updates.ChangeTypes)
		argCount++
	}
	if updates.NotifyWebhook != nil {
		query += fmt.Sprintf(", notify_webhook = $%d", argCount)
		args = append(args, *updates.NotifyWebhook)
		argCount++
	}
	if updates.WebhookURL != nil {
		query += fmt.Sprintf(", webhook_url = $%d", argCount)
		args = append(args, *updates.WebhookURL)
		argCount++
	}
	if updates.NotifyInApp != nil {
		query += fmt.Sprintf(", notify_in_app = $%d", argCount)
		args = append(args, *updates.NotifyInApp)
		argCount++
	}
	if updates.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argCount)
		args = append(args, *updates.IsActive)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	result, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update alert rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("alert rule not found: %s", id)
	}
	return nil
}

// DeleteAlertRule 删除告警规则
func (t *PostgresTxStore) DeleteAlertRule(ctx context.Context, id string) error {
	query := `DELETE FROM alert_rules WHERE id = $1`
	result, err := t.tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("alert rule not found: %s", id)
	}
	return nil
}

// ListMatchingAlertRules 列出匹配变更类型的告警规则
func (t *PostgresTxStore) ListMatchingAlertRules(ctx context.Context, sourceID string, changeType string) ([]*AlertRuleRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// CreateNotification 创建通知
func (t *PostgresTxStore) CreateNotification(ctx context.Context, notification *NotificationCreate) (string, error) {
	query := `
		INSERT INTO notifications (id, rule_id, change_id, source_id, title, message, change_type, object_type, object_name, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	id := uuid.New().String()
	_, err := t.tx.Exec(ctx, query,
		id, notification.RuleID, notification.ChangeID, notification.SourceID,
		notification.Title, notification.Message, notification.ChangeType,
		notification.ObjectType, notification.ObjectName, notification.OldValue, notification.NewValue)
	if err != nil {
		return "", fmt.Errorf("failed to create notification: %w", err)
	}

	// 为所有用户创建未读记录
	userQuery := `SELECT id FROM users`
	rows, err := t.tx.Query(ctx, userQuery)
	if err != nil {
		return id, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		userNotifQuery := `
			INSERT INTO user_notifications (id, user_id, notification_id, is_read)
			VALUES ($1, $2, $3, false)
			ON CONFLICT (user_id, notification_id) DO NOTHING`
		t.tx.Exec(ctx, userNotifQuery, uuid.New().String(), userID, id)
	}

	return id, nil
}

// GetNotification 获取通知
func (t *PostgresTxStore) GetNotification(ctx context.Context, id string) (*NotificationRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// ListNotifications 列出通知
func (t *PostgresTxStore) ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*NotificationRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// GetNotificationStats 获取通知统计
func (t *PostgresTxStore) GetNotificationStats(ctx context.Context, userID string) (*NotificationStatsRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// MarkNotificationAsRead 标记通知已读
func (t *PostgresTxStore) MarkNotificationAsRead(ctx context.Context, userID string, notificationID string) error {
	return fmt.Errorf("not implemented in transaction")
}

// MarkAllNotificationsAsRead 标记所有通知已读
func (t *PostgresTxStore) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	return fmt.Errorf("not implemented in transaction")
}

// UpdateNotificationWebhookStatus 更新Webhook发送状态
func (t *PostgresTxStore) UpdateNotificationWebhookStatus(ctx context.Context, id string, sent bool, errorMsg *string) error {
	query := `UPDATE notifications SET webhook_sent = $1, webhook_error = $2 WHERE id = $3`
	_, err := t.tx.Exec(ctx, query, sent, errorMsg, id)
	return err
}
