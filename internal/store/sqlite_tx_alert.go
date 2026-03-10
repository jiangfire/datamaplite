package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CreateAlertRule 创建告警规则
func (t *SQLiteTxStore) CreateAlertRule(ctx context.Context, rule *AlertRuleCreate) (string, error) {
	query := `
		INSERT INTO alert_rules (id, source_id, object_id, name, description, change_types, notify_webhook, webhook_url, notify_in_app, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, rule.SourceID, rule.ObjectID, rule.Name, rule.Description,
		rule.ChangeTypes, rule.NotifyWebhook, rule.WebhookURL, rule.NotifyInApp, rule.IsActive)
	if err != nil {
		return "", fmt.Errorf("failed to create alert rule: %w", err)
	}
	return id, nil
}

// GetAlertRule 获取告警规则
func (t *SQLiteTxStore) GetAlertRule(ctx context.Context, id string) (*AlertRuleRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// ListAlertRules 列出告警规则
func (t *SQLiteTxStore) ListAlertRules(ctx context.Context, sourceID *string) ([]*AlertRuleRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// UpdateAlertRule 更新告警规则
func (t *SQLiteTxStore) UpdateAlertRule(ctx context.Context, id string, updates *AlertRuleUpdate) error {
	query := `UPDATE alert_rules SET updated_at = datetime('now')`
	var args []interface{}

	if updates.Name != nil {
		query += `, name = ?`
		args = append(args, *updates.Name)
	}
	if updates.Description != nil {
		query += `, description = ?`
		args = append(args, *updates.Description)
	}
	if updates.SourceID != nil {
		query += `, source_id = ?`
		args = append(args, *updates.SourceID)
	}
	if updates.ObjectID != nil {
		query += `, object_id = ?`
		args = append(args, *updates.ObjectID)
	}
	if updates.ChangeTypes != nil {
		query += `, change_types = ?`
		args = append(args, *updates.ChangeTypes)
	}
	if updates.NotifyWebhook != nil {
		query += `, notify_webhook = ?`
		args = append(args, *updates.NotifyWebhook)
	}
	if updates.WebhookURL != nil {
		query += `, webhook_url = ?`
		args = append(args, *updates.WebhookURL)
	}
	if updates.NotifyInApp != nil {
		query += `, notify_in_app = ?`
		args = append(args, *updates.NotifyInApp)
	}
	if updates.IsActive != nil {
		query += `, is_active = ?`
		args = append(args, *updates.IsActive)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update alert rule: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("alert rule not found: %s", id)
	}
	return nil
}

// DeleteAlertRule 删除告警规则
func (t *SQLiteTxStore) DeleteAlertRule(ctx context.Context, id string) error {
	query := `DELETE FROM alert_rules WHERE id = ?`
	result, err := t.tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("alert rule not found: %s", id)
	}
	return nil
}

// ListMatchingAlertRules 列出匹配变更类型的告警规则
func (t *SQLiteTxStore) ListMatchingAlertRules(ctx context.Context, sourceID string, changeType string) ([]*AlertRuleRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// CreateNotification 创建通知
func (t *SQLiteTxStore) CreateNotification(ctx context.Context, notification *NotificationCreate) (string, error) {
	query := `
		INSERT INTO notifications (id, rule_id, change_id, source_id, title, message, change_type, object_type, object_name, old_value, new_value)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	id := uuid.New().String()
	_, err := t.tx.ExecContext(ctx, query,
		id, notification.RuleID, notification.ChangeID, notification.SourceID,
		notification.Title, notification.Message, notification.ChangeType,
		notification.ObjectType, notification.ObjectName, notification.OldValue, notification.NewValue)
	if err != nil {
		return "", fmt.Errorf("failed to create notification: %w", err)
	}

	// 为所有用户创建未读记录
	userQuery := `SELECT id FROM users`
	rows, err := t.tx.QueryContext(ctx, userQuery)
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
			VALUES (?, ?, ?, 0)
			ON CONFLICT (user_id, notification_id) DO NOTHING`
		t.tx.ExecContext(ctx, userNotifQuery, uuid.New().String(), userID, id)
	}

	return id, nil
}

// GetNotification 获取通知
func (t *SQLiteTxStore) GetNotification(ctx context.Context, id string) (*NotificationRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// ListNotifications 列出通知
func (t *SQLiteTxStore) ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*NotificationRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// GetNotificationStats 获取通知统计
func (t *SQLiteTxStore) GetNotificationStats(ctx context.Context, userID string) (*NotificationStatsRow, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

// MarkNotificationAsRead 标记通知已读
func (t *SQLiteTxStore) MarkNotificationAsRead(ctx context.Context, userID string, notificationID string) error {
	return fmt.Errorf("not implemented in transaction")
}

// MarkAllNotificationsAsRead 标记所有通知已读
func (t *SQLiteTxStore) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	return fmt.Errorf("not implemented in transaction")
}

// UpdateNotificationWebhookStatus 更新Webhook发送状态
func (t *SQLiteTxStore) UpdateNotificationWebhookStatus(ctx context.Context, id string, sent bool, errorMsg *string) error {
	query := `UPDATE notifications SET webhook_sent = ?, webhook_error = ? WHERE id = ?`
	_, err := t.tx.ExecContext(ctx, query, sent, errorMsg, id)
	return err
}
