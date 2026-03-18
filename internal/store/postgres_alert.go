package store

import (
	"context"
	"fmt"
	"strings"
)

// CreateAlertRule 创建告警规则
func (s *PostgresStore) CreateAlertRule(ctx context.Context, rule *AlertRuleCreate) (string, error) {
	query := `
		INSERT INTO alert_rules (source_id, object_id, name, description, change_types, notify_webhook, webhook_url, notify_in_app, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id string
	err := s.pool.QueryRow(ctx, query,
		rule.SourceID,
		rule.ObjectID,
		rule.Name,
		rule.Description,
		rule.ChangeTypes,
		rule.NotifyWebhook,
		rule.WebhookURL,
		rule.NotifyInApp,
		rule.IsActive,
	).Scan(&id)

	return id, err
}

// GetAlertRule 获取告警规则
func (s *PostgresStore) GetAlertRule(ctx context.Context, id string) (*AlertRuleRow, error) {
	query := `
		SELECT ar.id, ar.source_id, ar.object_id, ar.name, ar.description, ar.change_types,
		       ar.notify_webhook, ar.webhook_url, ar.notify_in_app, ar.is_active, ar.created_at, ar.updated_at,
		       ds.name as source_name, so.name as object_name
		FROM alert_rules ar
		LEFT JOIN data_sources ds ON ar.source_id = ds.id
		LEFT JOIN schema_objects so ON ar.object_id = so.id
		WHERE ar.id = $1`

	row := s.pool.QueryRow(ctx, query, id)

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
func (s *PostgresStore) ListAlertRules(ctx context.Context, sourceID *string) ([]*AlertRuleRow, error) {
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

	rows, err := s.pool.Query(ctx, query, args...)
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

// ListMatchingAlertRules 列出匹配变更类型的告警规则
func (s *PostgresStore) ListMatchingAlertRules(ctx context.Context, sourceID string, changeType string) ([]*AlertRuleRow, error) {
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

	rows, err := s.pool.Query(ctx, query, sourceID, cleanChangeType)
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
func (s *PostgresStore) UpdateAlertRule(ctx context.Context, id string, updates *AlertRuleUpdate) error {
	query := `UPDATE alert_rules SET `
	var setParts []string
	var args []interface{}
	argIdx := 1

	if updates.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *updates.Name)
		argIdx++
	}
	if updates.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *updates.Description)
		argIdx++
	}
	if updates.SourceID != nil {
		setParts = append(setParts, fmt.Sprintf("source_id = $%d", argIdx))
		args = append(args, *updates.SourceID)
		argIdx++
	}
	if updates.ObjectID != nil {
		setParts = append(setParts, fmt.Sprintf("object_id = $%d", argIdx))
		args = append(args, *updates.ObjectID)
		argIdx++
	}
	if updates.ChangeTypes != nil {
		setParts = append(setParts, fmt.Sprintf("change_types = $%d", argIdx))
		args = append(args, *updates.ChangeTypes)
		argIdx++
	}
	if updates.NotifyWebhook != nil {
		setParts = append(setParts, fmt.Sprintf("notify_webhook = $%d", argIdx))
		args = append(args, *updates.NotifyWebhook)
		argIdx++
	}
	if updates.WebhookURL != nil {
		setParts = append(setParts, fmt.Sprintf("webhook_url = $%d", argIdx))
		args = append(args, *updates.WebhookURL)
		argIdx++
	}
	if updates.NotifyInApp != nil {
		setParts = append(setParts, fmt.Sprintf("notify_in_app = $%d", argIdx))
		args = append(args, *updates.NotifyInApp)
		argIdx++
	}
	if updates.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *updates.IsActive)
		argIdx++
	}

	if len(setParts) == 0 {
		return nil
	}

	query += strings.Join(setParts, ", ")
	query += fmt.Sprintf(", updated_at = NOW() WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

// DeleteAlertRule 删除告警规则
func (s *PostgresStore) DeleteAlertRule(ctx context.Context, id string) error {
	query := `DELETE FROM alert_rules WHERE id = $1`
	_, err := s.pool.Exec(ctx, query, id)
	return err
}

// CreateNotification 创建通知
func (s *PostgresStore) CreateNotification(ctx context.Context, notification *NotificationCreate) (string, error) {
	query := `
		INSERT INTO notifications (rule_id, change_id, source_id, title, message, change_type, object_type, object_name, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id string
	err := s.pool.QueryRow(ctx, query,
		notification.RuleID,
		notification.ChangeID,
		notification.SourceID,
		notification.Title,
		notification.Message,
		notification.ChangeType,
		notification.ObjectType,
		notification.ObjectName,
		notification.OldValue,
		notification.NewValue,
	).Scan(&id)

	if err != nil {
		return "", err
	}

	// 为所有用户创建未读记录
	userQuery := `SELECT id FROM users`
	rows, err := s.pool.Query(ctx, userQuery)
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
			INSERT INTO user_notifications (user_id, notification_id, is_read)
			VALUES ($1, $2, false)
			ON CONFLICT (user_id, notification_id) DO NOTHING`
		s.pool.Exec(ctx, userNotifQuery, userID, id)
	}

	return id, nil
}

// GetNotification 获取通知
func (s *PostgresStore) GetNotification(ctx context.Context, id string) (*NotificationRow, error) {
	query := `
		SELECT n.id, n.rule_id, r.name as rule_name, n.change_id, n.source_id, ds.name as source_name,
		       n.title, n.message, n.change_type, n.object_type, n.object_name,
		       n.old_value, n.new_value, n.webhook_sent, n.webhook_error, n.created_at
		FROM notifications n
		LEFT JOIN alert_rules r ON n.rule_id = r.id
		JOIN data_sources ds ON n.source_id = ds.id
		WHERE n.id = $1`

	row := s.pool.QueryRow(ctx, query, id)

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

// ListNotifications 列出通知
func (s *PostgresStore) ListNotifications(ctx context.Context, userID string, unreadOnly bool, limit int) ([]*NotificationRow, error) {
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

	rows, err := s.pool.Query(ctx, query, args...)
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
func (s *PostgresStore) GetNotificationStats(ctx context.Context, userID string) (*NotificationStatsRow, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM user_notifications WHERE user_id = $1) as total_count,
			(SELECT COUNT(*) FROM user_notifications WHERE user_id = $1 AND is_read = false) as unread_count,
			(SELECT COUNT(*) FROM user_notifications un
			 JOIN notifications n ON un.notification_id = n.id
			 WHERE un.user_id = $1 AND n.created_at >= CURRENT_DATE) as today_count`

	stats := &NotificationStatsRow{}
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&stats.TotalCount, &stats.UnreadCount, &stats.TodayCount,
	)

	return stats, err
}

// MarkNotificationAsRead 标记通知已读
func (s *PostgresStore) MarkNotificationAsRead(ctx context.Context, userID string, notificationID string) error {
	query := `
		UPDATE user_notifications
		SET is_read = true, read_at = NOW()
		WHERE user_id = $1 AND notification_id = $2`

	_, err := s.pool.Exec(ctx, query, userID, notificationID)
	return err
}

// MarkAllNotificationsAsRead 标记所有通知已读
func (s *PostgresStore) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	query := `
		UPDATE user_notifications
		SET is_read = true, read_at = NOW()
		WHERE user_id = $1 AND is_read = false`

	_, err := s.pool.Exec(ctx, query, userID)
	return err
}

// UpdateNotificationWebhookStatus 更新Webhook发送状态
func (s *PostgresStore) UpdateNotificationWebhookStatus(ctx context.Context, id string, sent bool, errorMsg *string) error {
	query := `UPDATE notifications SET webhook_sent = $1, webhook_error = $2 WHERE id = $3`
	_, err := s.pool.Exec(ctx, query, sent, errorMsg, id)
	return err
}
