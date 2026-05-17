package model

import "time"

// AlertRule 告警规则
type AlertRule struct {
	ID            string    `json:"id" db:"id"`
	SourceID      *string   `json:"source_id,omitempty" db:"source_id"`
	ObjectID      *string   `json:"object_id,omitempty" db:"object_id"`
	Name          string    `json:"name" db:"name"`
	Description   *string   `json:"description,omitempty" db:"description"`
	ChangeTypes   string    `json:"change_types" db:"change_types"`
	NotifyWebhook bool      `json:"notify_webhook" db:"notify_webhook"`
	WebhookURL    *string   `json:"webhook_url,omitempty" db:"webhook_url"`
	NotifyInApp   bool      `json:"notify_in_app" db:"notify_in_app"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ChangeType 变更类型
type ChangeType string

const (
	ChangeTypeAddObject   ChangeType = "add_object"
	ChangeTypeDropObject  ChangeType = "drop_object"
	ChangeTypeAddColumn   ChangeType = "add_column"
	ChangeTypeDropColumn  ChangeType = "drop_column"
	ChangeTypeAlterColumn ChangeType = "alter_column"
	ChangeTypeChangeType  ChangeType = "change_type"
	ChangeTypeAll         ChangeType = "all"
)

// ValidChangeTypes 有效的变更类型列表
var ValidChangeTypes = []ChangeType{
	ChangeTypeAddObject,
	ChangeTypeDropObject,
	ChangeTypeAddColumn,
	ChangeTypeDropColumn,
	ChangeTypeAlterColumn,
	ChangeTypeChangeType,
}

// Notification 通知记录
type Notification struct {
	ID           string    `json:"id" db:"id"`
	RuleID       *string   `json:"rule_id,omitempty" db:"rule_id"`
	ChangeID     string    `json:"change_id" db:"change_id"`
	SourceID     string    `json:"source_id" db:"source_id"`
	Title        string    `json:"title" db:"title"`
	Message      string    `json:"message" db:"message"`
	ChangeType   string    `json:"change_type" db:"change_type"`
	ObjectType   string    `json:"object_type" db:"object_type"`
	ObjectName   string    `json:"object_name" db:"object_name"`
	OldValue     *string   `json:"old_value,omitempty" db:"old_value"`
	NewValue     *string   `json:"new_value,omitempty" db:"new_value"`
	WebhookSent  bool      `json:"webhook_sent" db:"webhook_sent"`
	WebhookError *string   `json:"webhook_error,omitempty" db:"webhook_error"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// UserNotification 用户通知状态
type UserNotification struct {
	ID             string     `json:"id" db:"id"`
	UserID         string     `json:"user_id" db:"user_id"`
	NotificationID string     `json:"notification_id" db:"notification_id"`
	IsRead         bool       `json:"is_read" db:"is_read"`
	ReadAt         *time.Time `json:"read_at,omitempty" db:"read_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	// 关联的通知详情
	Notification *Notification `json:"notification,omitempty" db:"-"`
}

// AlertRuleRequest 创建/更新告警规则请求。
// Description 和 WebhookURL 使用指针：nil 表示"不修改"，非 nil（含空串）表示"显式设置/清空"。
type AlertRuleRequest struct {
	SourceID      *string `json:"source_id,omitempty" validate:"omitempty,uuid"`
	ObjectID      *string `json:"object_id,omitempty" validate:"omitempty,uuid"`
	Name          string  `json:"name" validate:"required,max=255"`
	Description   *string `json:"description,omitempty"`
	ChangeTypes   string  `json:"change_types" validate:"required"`
	NotifyWebhook bool    `json:"notify_webhook"`
	WebhookURL    *string `json:"webhook_url,omitempty" validate:"omitempty,url"`
	NotifyInApp   bool    `json:"notify_in_app"`
	IsActive      bool    `json:"is_active"`
}

// AlertRuleResponse 告警规则响应
type AlertRuleResponse struct {
	ID            string  `json:"id"`
	SourceID      *string `json:"source_id,omitempty"`
	ObjectID      *string `json:"object_id,omitempty"`
	SourceName    *string `json:"source_name,omitempty"`
	ObjectName    *string `json:"object_name,omitempty"`
	Name          string  `json:"name"`
	Description   string  `json:"description,omitempty"`
	ChangeTypes   string  `json:"change_types"`
	NotifyWebhook bool    `json:"notify_webhook"`
	WebhookURL    string  `json:"webhook_url,omitempty"`
	NotifyInApp   bool    `json:"notify_in_app"`
	IsActive      bool    `json:"is_active"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// NotificationResponse 通知响应
type NotificationResponse struct {
	ID           string  `json:"id"`
	RuleID       *string `json:"rule_id,omitempty"`
	RuleName     string  `json:"rule_name,omitempty"`
	ChangeID     string  `json:"change_id"`
	SourceID     string  `json:"source_id"`
	SourceName   string  `json:"source_name"`
	Title        string  `json:"title"`
	Message      string  `json:"message"`
	ChangeType   string  `json:"change_type"`
	ObjectType   string  `json:"object_type"`
	ObjectName   string  `json:"object_name"`
	OldValue     string  `json:"old_value,omitempty"`
	NewValue     string  `json:"new_value,omitempty"`
	WebhookSent  bool    `json:"webhook_sent"`
	WebhookError string  `json:"webhook_error,omitempty"`
	IsRead       bool    `json:"is_read"`
	CreatedAt    string  `json:"created_at"`
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	RuleID     *string `json:"rule_id,omitempty"`
	ChangeID   string  `json:"change_id" validate:"required"`
	SourceID   string  `json:"source_id" validate:"required"`
	Title      string  `json:"title" validate:"required"`
	Message    string  `json:"message" validate:"required"`
	ChangeType string  `json:"change_type" validate:"required"`
	ObjectType string  `json:"object_type" validate:"required"`
	ObjectName string  `json:"object_name" validate:"required"`
	OldValue   *string `json:"old_value,omitempty"`
	NewValue   *string `json:"new_value,omitempty"`
}

// WebhookPayload Webhook通知载荷
type WebhookPayload struct {
	ID             string    `json:"id"`
	NotificationID string    `json:"notification_id,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	Event          string    `json:"event"`
	Timestamp      time.Time `json:"timestamp"`
	ChangeType     string    `json:"change_type"`
	ObjectType     string    `json:"object_type"`
	ObjectName     string    `json:"object_name"`
	SourceID       string    `json:"source_id"`
	SourceName     string    `json:"source_name,omitempty"`
	OldValue       *string   `json:"old_value,omitempty"`
	NewValue       *string   `json:"new_value,omitempty"`
	Message        string    `json:"message"`
	RuleID         *string   `json:"rule_id,omitempty"`
	RuleName       string    `json:"rule_name,omitempty"`
}

// NotificationStats 通知统计
type NotificationStats struct {
	TotalCount  int64 `json:"total_count"`
	UnreadCount int64 `json:"unread_count"`
	TodayCount  int64 `json:"today_count"`
}

// MarkAsReadRequest 标记已读请求
type MarkAsReadRequest struct {
	NotificationIDs []string `json:"notification_ids,omitempty"`
	MarkAll         bool     `json:"mark_all"`
}
