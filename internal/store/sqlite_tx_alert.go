package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	query := `
		SELECT ar.id, ar.source_id, ar.object_id, ar.name, ar.description, ar.change_types,
		       ar.notify_webhook, ar.webhook_url, ar.notify_in_app, ar.is_active, ar.created_at, ar.updated_at,
		       ds.name as source_name, so.name as object_name
		FROM alert_rules ar
		LEFT JOIN data_sources ds ON ar.source_id = ds.id
		LEFT JOIN schema_objects so ON ar.object_id = so.id
		WHERE ar.id = ?`

	row := t.tx.QueryRowContext(ctx, query, id)

	r := &AlertRuleRow{}
	var sourceName, objectName sql.NullString

	err := row.Scan(
		&r.ID, &r.SourceID, &r.ObjectID, &r.Name, &r.Description, &r.ChangeTypes,
		&r.NotifyWebhook, &r.WebhookURL, &r.NotifyInApp, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
		&sourceName, &objectName,
	)
	if err != nil {
		return nil, err
	}

	if sourceName.Valid {
		r.SourceName = &sourceName.String
	}
	if objectName.Valid {
		r.ObjectName = &objectName.String
	}

	return r, nil
}

// ListAlertRules 列出告警规则
func (t *SQLiteTxStore) ListAlertRules(ctx context.Context, sourceID *string) ([]*AlertRuleRow, error) {
	query := `
		SELECT ar.id, ar.source_id, ar.object_id, ar.name, ar.description, ar.change_types,
		       ar.notify_webhook, ar.webhook_url, ar.notify_in_app, ar.is_active, ar.created_at, ar.updated_at,
		       ds.name as source_name, so.name as object_name
		FROM alert_rules ar
		LEFT JOIN data_sources ds ON ar.source_id = ds.id
		LEFT JOIN schema_objects so ON ar.object_id = so.id`

	var args []interface{}
	if sourceID != nil {
		query += ` WHERE ar.source_id = ? OR ar.source_id IS NULL`
		args = append(args, *sourceID)
	}

	query += ` ORDER BY ar.created_at DESC`

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*AlertRuleRow
	for rows.Next() {
		r := &AlertRuleRow{}
		var sourceName, objectName sql.NullString

		err := rows.Scan(
			&r.ID, &r.SourceID, &r.ObjectID, &r.Name, &r.Description, &r.ChangeTypes,
			&r.NotifyWebhook, &r.WebhookURL, &r.NotifyInApp, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
			&sourceName, &objectName,
		)
		if err != nil {
			return nil, err
		}

		if sourceName.Valid {
			r.SourceName = &sourceName.String
		}
		if objectName.Valid {
			r.ObjectName = &objectName.String
		}
		rules = append(rules, r)
	}

	return rules, rows.Err()
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
	query := `
		SELECT ar.id, ar.source_id, ar.object_id, ar.name, ar.description, ar.change_types,
		       ar.notify_webhook, ar.webhook_url, ar.notify_in_app, ar.is_active, ar.created_at, ar.updated_at,
		       ds.name as source_name, so.name as object_name
	FROM alert_rules ar
	LEFT JOIN data_sources ds ON ar.source_id = ds.id
	LEFT JOIN schema_objects so ON ar.object_id = so.id
	WHERE ar.is_active = 1
	  AND (ar.source_id = ? OR ar.source_id IS NULL)
	  AND (
	      ar.change_types = 'all'
	      OR INSTR(',' || REPLACE(ar.change_types, ' ', '') || ',', ',' || REPLACE(?, ' ', '') || ',') > 0
	  )`

	cleanChangeType := strings.TrimSpace(changeType)

	rows, err := t.tx.QueryContext(ctx, query, sourceID, cleanChangeType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*AlertRuleRow
	for rows.Next() {
		r := &AlertRuleRow{}
		var sourceName, objectName sql.NullString

		err := rows.Scan(
			&r.ID, &r.SourceID, &r.ObjectID, &r.Name, &r.Description, &r.ChangeTypes,
			&r.NotifyWebhook, &r.WebhookURL, &r.NotifyInApp, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
			&sourceName, &objectName,
		)
		if err != nil {
			return nil, err
		}

		if sourceName.Valid {
			r.SourceName = &sourceName.String
		}
		if objectName.Valid {
			r.ObjectName = &objectName.String
		}
		rules = append(rules, r)
	}

	return rules, rows.Err()
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

	if notification.NotifyInApp {
		// 为所有用户创建未读记录
		userQuery := `SELECT id FROM users`
		rows, err := t.tx.QueryContext(ctx, userQuery)
		if err != nil {
			return id, err
		}
		userIDs := make([]string, 0)

		for rows.Next() {
			var userID string
			if err := rows.Scan(&userID); err != nil {
				_ = rows.Close()
				return id, err
			}
			userIDs = append(userIDs, userID)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return id, err
		}
		if err := rows.Close(); err != nil {
			return id, err
		}

		for _, userID := range userIDs {
			userNotifQuery := `
				INSERT INTO user_notifications (id, user_id, notification_id, is_read)
				VALUES (?, ?, ?, 0)
				ON CONFLICT (user_id, notification_id) DO NOTHING`
			if _, err := t.tx.ExecContext(ctx, userNotifQuery, uuid.New().String(), userID, id); err != nil {
				return id, err
			}
		}
	}

	return id, nil
}

// GetNotification 获取通知
func (t *SQLiteTxStore) GetNotification(ctx context.Context, id string) (*NotificationRow, error) {
	query := `
		SELECT n.id, n.rule_id, r.name as rule_name, n.change_id, n.source_id, ds.name as source_name,
		       n.title, n.message, n.change_type, n.object_type, n.object_name,
		       n.old_value, n.new_value, n.webhook_sent, n.webhook_error, n.created_at
		FROM notifications n
		LEFT JOIN alert_rules r ON n.rule_id = r.id
		JOIN data_sources ds ON n.source_id = ds.id
		WHERE n.id = ?`

	row := t.tx.QueryRowContext(ctx, query, id)

	n := &NotificationRow{}
	var ruleName sql.NullString

	err := row.Scan(
		&n.ID, &n.RuleID, &ruleName, &n.ChangeID, &n.SourceID, &n.SourceName,
		&n.Title, &n.Message, &n.ChangeType, &n.ObjectType, &n.ObjectName,
		&n.OldValue, &n.NewValue, &n.WebhookSent, &n.WebhookError, &n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if ruleName.Valid {
		ruleNameVal := ruleName.String
		n.RuleName = &ruleNameVal
	}

	return n, nil
}

// GetNotificationByRuleAndChange 根据规则和变更获取通知。
func (t *SQLiteTxStore) GetNotificationByRuleAndChange(ctx context.Context, ruleID string, changeID string) (*NotificationRow, error) {
	query := `
		SELECT n.id, n.rule_id, r.name as rule_name, n.change_id, n.source_id, ds.name as source_name,
		       n.title, n.message, n.change_type, n.object_type, n.object_name,
		       n.old_value, n.new_value, n.webhook_sent, n.webhook_error, n.created_at
		FROM notifications n
		LEFT JOIN alert_rules r ON n.rule_id = r.id
		JOIN data_sources ds ON n.source_id = ds.id
		WHERE n.rule_id = ? AND n.change_id = ?
		LIMIT 1`

	row := t.tx.QueryRowContext(ctx, query, ruleID, changeID)

	n := &NotificationRow{}
	var ruleName sql.NullString

	err := row.Scan(
		&n.ID, &n.RuleID, &ruleName, &n.ChangeID, &n.SourceID, &n.SourceName,
		&n.Title, &n.Message, &n.ChangeType, &n.ObjectType, &n.ObjectName,
		&n.OldValue, &n.NewValue, &n.WebhookSent, &n.WebhookError, &n.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if ruleName.Valid {
		ruleNameVal := ruleName.String
		n.RuleName = &ruleNameVal
	}

	return n, nil
}

// ListNotifications 列出通知
func (t *SQLiteTxStore) ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*NotificationRow, error) {
	query := `
		SELECT n.id, n.rule_id, r.name as rule_name, n.change_id, n.source_id, ds.name as source_name,
		       n.title, n.message, n.change_type, n.object_type, n.object_name,
		       n.old_value, n.new_value, n.webhook_sent, n.webhook_error, n.created_at,
		       un.is_read, un.read_at
		FROM user_notifications un
		JOIN notifications n ON un.notification_id = n.id
		LEFT JOIN alert_rules r ON n.rule_id = r.id
		JOIN data_sources ds ON n.source_id = ds.id
		WHERE un.user_id = ?`

	var args []interface{}
	args = append(args, userID)

	if unreadOnly {
		query += ` AND un.is_read = 0`
	}

	query += ` ORDER BY n.created_at DESC`

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*NotificationRow
	for rows.Next() {
		n := &NotificationRow{}
		var ruleName sql.NullString
		var readAt sql.NullString

		err := rows.Scan(
			&n.ID, &n.RuleID, &ruleName, &n.ChangeID, &n.SourceID, &n.SourceName,
			&n.Title, &n.Message, &n.ChangeType, &n.ObjectType, &n.ObjectName,
			&n.OldValue, &n.NewValue, &n.WebhookSent, &n.WebhookError, &n.CreatedAt,
			&n.IsRead, &readAt,
		)
		if err != nil {
			return nil, err
		}

		if ruleName.Valid {
			ruleNameVal := ruleName.String
			n.RuleName = &ruleNameVal
		}
		if readAt.Valid {
			n.ReadAt = &readAt.String
		}
		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

// GetNotificationStats 获取通知统计
func (t *SQLiteTxStore) GetNotificationStats(ctx context.Context, userID string) (*NotificationStatsRow, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM user_notifications WHERE user_id = ?) as total_count,
			(SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND is_read = 0) as unread_count,
			(SELECT COUNT(*) FROM user_notifications un
			 JOIN notifications n ON un.notification_id = n.id
			 WHERE un.user_id = ? AND date(n.created_at) = date('now')) as today_count`

	stats := &NotificationStatsRow{}
	err := t.tx.QueryRowContext(ctx, query, userID, userID, userID).Scan(
		&stats.TotalCount, &stats.UnreadCount, &stats.TodayCount,
	)

	return stats, err
}

// MarkNotificationAsRead 标记通知已读
func (t *SQLiteTxStore) MarkNotificationAsRead(ctx context.Context, userID string, notificationID string) error {
	query := `
		UPDATE user_notifications
		SET is_read = 1, read_at = datetime('now')
		WHERE user_id = ? AND notification_id = ?`

	_, err := t.tx.ExecContext(ctx, query, userID, notificationID)
	return err
}

// MarkAllNotificationsAsRead 标记所有通知已读
func (t *SQLiteTxStore) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	query := `
		UPDATE user_notifications
		SET is_read = 1, read_at = datetime('now')
		WHERE user_id = ? AND is_read = 0`

	_, err := t.tx.ExecContext(ctx, query, userID)
	return err
}

// UpdateNotificationWebhookStatus 更新Webhook发送状态
func (t *SQLiteTxStore) UpdateNotificationWebhookStatus(ctx context.Context, id string, sent bool, errorMsg *string) error {
	query := `UPDATE notifications SET webhook_sent = ?, webhook_error = ? WHERE id = ?`
	_, err := t.tx.ExecContext(ctx, query, sent, errorMsg, id)
	return err
}
