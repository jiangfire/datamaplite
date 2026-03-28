package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	query := `
		SELECT ar.id, ar.source_id, ar.object_id, ar.name, ar.description, ar.change_types,
		       ar.notify_webhook, ar.webhook_url, ar.notify_in_app, ar.is_active, ar.created_at, ar.updated_at,
		       ds.name as source_name, so.name as object_name
		FROM alert_rules ar
		LEFT JOIN data_sources ds ON ar.source_id = ds.id
		LEFT JOIN schema_objects so ON ar.object_id = so.id
		WHERE ar.id = $1`

	row := t.tx.QueryRow(ctx, query, id)

	r := &AlertRuleRow{}
	var sourceName, objectName *string

	err := row.Scan(
		&r.ID, &r.SourceID, &r.ObjectID, &r.Name, &r.Description, &r.ChangeTypes,
		&r.NotifyWebhook, &r.WebhookURL, &r.NotifyInApp, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
		&sourceName, &objectName,
	)
	if err != nil {
		return nil, err
	}

	r.SourceName = sourceName
	r.ObjectName = objectName

	return r, nil
}

// ListAlertRules 列出告警规则
func (t *PostgresTxStore) ListAlertRules(ctx context.Context, sourceID *string) ([]*AlertRuleRow, error) {
	query := `
		SELECT ar.id, ar.source_id, ar.object_id, ar.name, ar.description, ar.change_types,
		       ar.notify_webhook, ar.webhook_url, ar.notify_in_app, ar.is_active, ar.created_at, ar.updated_at,
		       ds.name as source_name, so.name as object_name
		FROM alert_rules ar
		LEFT JOIN data_sources ds ON ar.source_id = ds.id
		LEFT JOIN schema_objects so ON ar.object_id = so.id`

	var args []interface{}
	if sourceID != nil {
		query += ` WHERE ar.source_id = $1 OR ar.source_id IS NULL`
		args = append(args, *sourceID)
	}

	query += ` ORDER BY ar.created_at DESC`

	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*AlertRuleRow
	for rows.Next() {
		r := &AlertRuleRow{}
		var sourceName, objectName *string

		err := rows.Scan(
			&r.ID, &r.SourceID, &r.ObjectID, &r.Name, &r.Description, &r.ChangeTypes,
			&r.NotifyWebhook, &r.WebhookURL, &r.NotifyInApp, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
			&sourceName, &objectName,
		)
		if err != nil {
			return nil, err
		}

		r.SourceName = sourceName
		r.ObjectName = objectName
		rules = append(rules, r)
	}

	return rules, rows.Err()
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
	query := `
		SELECT ar.id, ar.source_id, ar.object_id, ar.name, ar.description, ar.change_types,
		       ar.notify_webhook, ar.webhook_url, ar.notify_in_app, ar.is_active, ar.created_at, ar.updated_at,
		       ds.name as source_name, so.name as object_name
		FROM alert_rules ar
		LEFT JOIN data_sources ds ON ar.source_id = ds.id
		LEFT JOIN schema_objects so ON ar.object_id = so.id
		WHERE ar.is_active = true
		  AND (ar.source_id = $1 OR ar.source_id IS NULL)
		  AND (ar.change_types = 'all' OR $2 = ANY(string_to_array(ar.change_types, ',')))`

	// Normalize changeType to handle whitespace
	cleanChangeType := strings.TrimSpace(changeType)

	rows, err := t.tx.Query(ctx, query, sourceID, cleanChangeType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*AlertRuleRow
	for rows.Next() {
		r := &AlertRuleRow{}
		var sourceName, objectName *string

		err := rows.Scan(
			&r.ID, &r.SourceID, &r.ObjectID, &r.Name, &r.Description, &r.ChangeTypes,
			&r.NotifyWebhook, &r.WebhookURL, &r.NotifyInApp, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
			&sourceName, &objectName,
		)
		if err != nil {
			return nil, err
		}

		r.SourceName = sourceName
		r.ObjectName = objectName
		rules = append(rules, r)
	}

	return rules, rows.Err()
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

	if notification.NotifyInApp {
		// 为所有用户创建未读记录
		userQuery := `SELECT id FROM users`
		rows, err := t.tx.Query(ctx, userQuery)
		if err != nil {
			return id, err
		}
		userIDs := make([]string, 0)

		for rows.Next() {
			var userID string
			if err := rows.Scan(&userID); err != nil {
				rows.Close()
				return id, err
			}
			userIDs = append(userIDs, userID)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return id, err
		}
		rows.Close()

		for _, userID := range userIDs {
			userNotifQuery := `
				INSERT INTO user_notifications (id, user_id, notification_id, is_read)
				VALUES ($1, $2, $3, false)
				ON CONFLICT (user_id, notification_id) DO NOTHING`
			if _, err := t.tx.Exec(ctx, userNotifQuery, uuid.New().String(), userID, id); err != nil {
				return id, err
			}
		}
	}

	return id, nil
}

// GetNotification 获取通知
func (t *PostgresTxStore) GetNotification(ctx context.Context, id string) (*NotificationRow, error) {
	query := `
		SELECT n.id, n.rule_id, r.name as rule_name, n.change_id, n.source_id, ds.name as source_name,
		       n.title, n.message, n.change_type, n.object_type, n.object_name,
		       n.old_value, n.new_value, n.webhook_sent, n.webhook_error, n.created_at
		FROM notifications n
		LEFT JOIN alert_rules r ON n.rule_id = r.id
		JOIN data_sources ds ON n.source_id = ds.id
		WHERE n.id = $1`

	row := t.tx.QueryRow(ctx, query, id)

	n := &NotificationRow{}
	var ruleName *string

	err := row.Scan(
		&n.ID, &n.RuleID, &ruleName, &n.ChangeID, &n.SourceID, &n.SourceName,
		&n.Title, &n.Message, &n.ChangeType, &n.ObjectType, &n.ObjectName,
		&n.OldValue, &n.NewValue, &n.WebhookSent, &n.WebhookError, &n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	n.RuleName = ruleName

	return n, nil
}

// GetNotificationByRuleAndChange 根据规则和变更获取通知。
func (t *PostgresTxStore) GetNotificationByRuleAndChange(ctx context.Context, ruleID string, changeID string) (*NotificationRow, error) {
	query := `
		SELECT n.id, n.rule_id, r.name as rule_name, n.change_id, n.source_id, ds.name as source_name,
		       n.title, n.message, n.change_type, n.object_type, n.object_name,
		       n.old_value, n.new_value, n.webhook_sent, n.webhook_error, n.created_at
		FROM notifications n
		LEFT JOIN alert_rules r ON n.rule_id = r.id
		JOIN data_sources ds ON n.source_id = ds.id
		WHERE n.rule_id = $1 AND n.change_id = $2
		LIMIT 1`

	row := t.tx.QueryRow(ctx, query, ruleID, changeID)

	n := &NotificationRow{}
	var ruleName *string

	err := row.Scan(
		&n.ID, &n.RuleID, &ruleName, &n.ChangeID, &n.SourceID, &n.SourceName,
		&n.Title, &n.Message, &n.ChangeType, &n.ObjectType, &n.ObjectName,
		&n.OldValue, &n.NewValue, &n.WebhookSent, &n.WebhookError, &n.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	n.RuleName = ruleName
	return n, nil
}

// ListNotifications 列出通知
func (t *PostgresTxStore) ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*NotificationRow, error) {
	query := `
		SELECT n.id, n.rule_id, r.name as rule_name, n.change_id, n.source_id, ds.name as source_name,
		       n.title, n.message, n.change_type, n.object_type, n.object_name,
		       n.old_value, n.new_value, n.webhook_sent, n.webhook_error, n.created_at,
		       un.is_read, un.read_at
		FROM user_notifications un
		JOIN notifications n ON un.notification_id = n.id
		LEFT JOIN alert_rules r ON n.rule_id = r.id
		JOIN data_sources ds ON n.source_id = ds.id
		WHERE un.user_id = $1`

	args := []interface{}{userID}
	argIdx := 2

	if unreadOnly {
		query += ` AND un.is_read = false`
	}

	query += ` ORDER BY n.created_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}

	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*NotificationRow
	for rows.Next() {
		n := &NotificationRow{}
		var ruleName *string
		var readAt *string

		err := rows.Scan(
			&n.ID, &n.RuleID, &ruleName, &n.ChangeID, &n.SourceID, &n.SourceName,
			&n.Title, &n.Message, &n.ChangeType, &n.ObjectType, &n.ObjectName,
			&n.OldValue, &n.NewValue, &n.WebhookSent, &n.WebhookError, &n.CreatedAt,
			&n.IsRead, &readAt,
		)
		if err != nil {
			return nil, err
		}

		n.RuleName = ruleName
		n.ReadAt = readAt
		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

// GetNotificationStats 获取通知统计
func (t *PostgresTxStore) GetNotificationStats(ctx context.Context, userID string) (*NotificationStatsRow, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM user_notifications WHERE user_id = $1) as total_count,
			(SELECT COUNT(*) FROM user_notifications WHERE user_id = $1 AND is_read = false) as unread_count,
			(SELECT COUNT(*) FROM user_notifications un
			 JOIN notifications n ON un.notification_id = n.id
			 WHERE un.user_id = $1 AND n.created_at >= CURRENT_DATE) as today_count`

	stats := &NotificationStatsRow{}
	err := t.tx.QueryRow(ctx, query, userID).Scan(
		&stats.TotalCount, &stats.UnreadCount, &stats.TodayCount,
	)

	return stats, err
}

// MarkNotificationAsRead 标记通知已读
func (t *PostgresTxStore) MarkNotificationAsRead(ctx context.Context, userID string, notificationID string) error {
	query := `
		UPDATE user_notifications
		SET is_read = true, read_at = NOW()
		WHERE user_id = $1 AND notification_id = $2`

	_, err := t.tx.Exec(ctx, query, userID, notificationID)
	return err
}

// MarkAllNotificationsAsRead 标记所有通知已读
func (t *PostgresTxStore) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	query := `
		UPDATE user_notifications
		SET is_read = true, read_at = NOW()
		WHERE user_id = $1 AND is_read = false`

	_, err := t.tx.Exec(ctx, query, userID)
	return err
}

// UpdateNotificationWebhookStatus 更新Webhook发送状态
func (t *PostgresTxStore) UpdateNotificationWebhookStatus(ctx context.Context, id string, sent bool, errorMsg *string) error {
	query := `UPDATE notifications SET webhook_sent = $1, webhook_error = $2 WHERE id = $3`
	_, err := t.tx.Exec(ctx, query, sent, errorMsg, id)
	return err
}
