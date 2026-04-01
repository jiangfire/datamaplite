package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/config"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type governanceAuditContextKey string

const governanceAuditMetaKey governanceAuditContextKey = "governance_audit_meta"

// GovernanceEvent 标准治理事件。
type GovernanceEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	OccurredAt   string                 `json:"occurred_at"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	ActorID      string                 `json:"actor_id,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
}

// GovernanceAuditMeta 描述一次治理动作的审计上下文。
type GovernanceAuditMeta struct {
	ActorID   string
	TraceID   string
	Origin    string
	Operation string
}

// GovernanceEventService 负责向 cornerstone 发送治理事件。
type GovernanceEventService struct {
	enabled                bool
	endpoint               string
	integrationToken       string
	sourceSystem           string
	client                 *http.Client
	logger                 *zap.Logger
	store                  store.Store
	outboxOwnerID          string
	outboxLeaseTTL         time.Duration
	outboxRetryBaseDelay   time.Duration
	outboxDispatchInterval time.Duration
	outboxMaxAttempts      int
}

// NewGovernanceEventService 创建治理事件发送服务。
func NewGovernanceEventService(cfg config.GovernanceConfig, logger *zap.Logger) *GovernanceEventService {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &GovernanceEventService{
		enabled:                cfg.Enabled,
		endpoint:               strings.TrimSpace(cfg.Endpoint),
		integrationToken:       strings.TrimSpace(cfg.IntegrationToken),
		sourceSystem:           strings.TrimSpace(cfg.SourceSystem),
		client:                 &http.Client{Timeout: timeout},
		logger:                 logger,
		outboxOwnerID:          "governance_outbox_" + uuid.NewString(),
		outboxLeaseTTL:         15 * time.Second,
		outboxRetryBaseDelay:   time.Second,
		outboxDispatchInterval: 2 * time.Second,
		outboxMaxAttempts:      5,
	}
}

// SetStore 为治理事件服务挂载持久化 store，启用 outbox 模式。
func (s *GovernanceEventService) SetStore(st store.Store) {
	if s == nil {
		return
	}
	s.store = st
}

// StartOutboxDispatcher 启动后台 outbox 补偿投递。
func (s *GovernanceEventService) StartOutboxDispatcher(ctx context.Context, interval time.Duration) {
	if s == nil || !s.Enabled() || s.store == nil {
		return
	}
	if interval <= 0 {
		interval = s.outboxDispatchInterval
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.ProcessOutbox(context.WithoutCancel(ctx), 16); err != nil && s.logger != nil {
					s.logger.Warn("process governance outbox failed", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// ListOutbox 列出治理 outbox 事件。
func (s *GovernanceEventService) ListOutbox(ctx context.Context, limit int) ([]*store.GovernanceOutboxEventRow, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("governance outbox store is not configured")
	}
	return s.store.ListGovernanceOutboxEvents(ctx, limit)
}

// GetOutboxStats 获取治理 outbox 统计。
func (s *GovernanceEventService) GetOutboxStats(ctx context.Context) (*store.GovernanceOutboxStatsRow, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("governance outbox store is not configured")
	}
	return s.store.GetGovernanceOutboxStats(ctx)
}

// ReplayOutboxEvent 手工重放治理 outbox 事件。
func (s *GovernanceEventService) ReplayOutboxEvent(ctx context.Context, id string) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("governance outbox store is not configured")
	}
	return s.store.ReplayGovernanceOutboxEvent(ctx, id, time.Now().UTC().Format(time.RFC3339Nano))
}

// WithGovernanceAuditMeta 将治理审计上下文放入 context。
func WithGovernanceAuditMeta(ctx context.Context, meta GovernanceAuditMeta) context.Context {
	return context.WithValue(ctx, governanceAuditMetaKey, meta)
}

// GovernanceAuditMetaFromContext 从 context 中提取治理审计上下文。
func GovernanceAuditMetaFromContext(ctx context.Context) GovernanceAuditMeta {
	if ctx == nil {
		return GovernanceAuditMeta{}
	}

	meta, ok := ctx.Value(governanceAuditMetaKey).(GovernanceAuditMeta)
	if !ok {
		return GovernanceAuditMeta{}
	}
	return meta
}

// Enabled 返回治理事件发送是否启用。
func (s *GovernanceEventService) Enabled() bool {
	return s != nil && s.enabled
}

// Publish 发送治理事件到 cornerstone。
func (s *GovernanceEventService) Publish(ctx context.Context, event GovernanceEvent) error {
	if !s.Enabled() {
		return nil
	}

	normalized, err := s.normalizeEvent(ctx, event)
	if err != nil {
		return err
	}

	if s.store == nil {
		return s.sendEvent(ctx, normalized)
	}

	if err := s.enqueueEvent(context.WithoutCancel(ctx), normalized); err != nil {
		return err
	}

	if err := s.ProcessOutbox(context.WithoutCancel(ctx), 8); err != nil && s.logger != nil {
		s.logger.Warn("best-effort governance outbox flush failed",
			zap.String("eventID", normalized.EventID),
			zap.String("eventType", normalized.EventType),
			zap.Error(err),
		)
	}

	return nil
}

// ProcessOutbox 处理治理事件 outbox。
func (s *GovernanceEventService) ProcessOutbox(ctx context.Context, limit int) error {
	if !s.Enabled() || s.store == nil {
		return nil
	}
	if limit <= 0 {
		limit = 1
	}

	now := time.Now().UTC()
	leaseUntil := now.Add(s.outboxLeaseTTL).Format(time.RFC3339Nano)
	rows, err := s.store.ClaimGovernanceOutboxEvents(ctx, s.outboxOwnerID, now.Format(time.RFC3339Nano), leaseUntil, limit)
	if err != nil {
		return err
	}

	var firstErr error
	for _, row := range rows {
		var event GovernanceEvent
		if err := json.Unmarshal([]byte(row.Payload), &event); err != nil {
			s.handleOutboxFailure(ctx, row, fmt.Errorf("unmarshal governance outbox payload failed: %w", err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := s.sendEvent(context.WithoutCancel(ctx), event); err != nil {
			s.handleOutboxFailure(ctx, row, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		deliveredAt := time.Now().UTC().Format(time.RFC3339Nano)
		if err := s.store.MarkGovernanceOutboxDelivered(ctx, row.ID, deliveredAt); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (s *GovernanceEventService) enqueueEvent(ctx context.Context, event GovernanceEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		s.logError("marshal governance outbox payload failed", event, err)
		return fmt.Errorf("marshal governance outbox payload failed: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.store.EnqueueGovernanceOutboxEvent(ctx, &store.GovernanceOutboxEventCreate{
		ID:            "outbox_" + uuid.NewString(),
		EventID:       event.EventID,
		EventType:     event.EventType,
		TraceID:       event.TraceID,
		ResourceType:  event.ResourceType,
		ResourceID:    event.ResourceID,
		Payload:       string(payload),
		Status:        "pending",
		AttemptCount:  0,
		NextAttemptAt: now,
	})
	if err != nil {
		s.logError("enqueue governance outbox event failed", event, err)
		return fmt.Errorf("enqueue governance outbox event failed: %w", err)
	}
	return nil
}

func (s *GovernanceEventService) normalizeEvent(ctx context.Context, event GovernanceEvent) (GovernanceEvent, error) {
	if strings.TrimSpace(event.EventType) == "" {
		return GovernanceEvent{}, fmt.Errorf("governance event_type is required")
	}

	auditMeta := GovernanceAuditMetaFromContext(ctx)
	if strings.TrimSpace(event.EventID) == "" {
		event.EventID = "evt_" + uuid.NewString()
	}
	if strings.TrimSpace(event.TraceID) == "" && strings.TrimSpace(auditMeta.TraceID) != "" {
		event.TraceID = auditMeta.TraceID
	}
	if strings.TrimSpace(event.TraceID) == "" {
		event.TraceID = "trc_" + uuid.NewString()
	}
	if strings.TrimSpace(event.ActorID) == "" && strings.TrimSpace(auditMeta.ActorID) != "" {
		event.ActorID = auditMeta.ActorID
	}
	if strings.TrimSpace(event.ActorID) == "" {
		event.ActorID = "system"
	}
	if strings.TrimSpace(event.OccurredAt) == "" {
		event.OccurredAt = time.Now().UTC().Format(time.RFC3339)
	}
	if event.Payload == nil {
		event.Payload = map[string]interface{}{}
	}
	if _, exists := event.Payload["audit_origin"]; !exists && strings.TrimSpace(auditMeta.Origin) != "" {
		event.Payload["audit_origin"] = auditMeta.Origin
	}
	if _, exists := event.Payload["audit_operation"]; !exists && strings.TrimSpace(auditMeta.Operation) != "" {
		event.Payload["audit_operation"] = auditMeta.Operation
	}

	return event, nil
}

func (s *GovernanceEventService) sendEvent(ctx context.Context, event GovernanceEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		s.logError("marshal governance event failed", event, err)
		return fmt.Errorf("marshal governance event failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewBuffer(body))
	if err != nil {
		s.logError("create governance event request failed", event, err)
		return fmt.Errorf("create governance event request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.integrationToken)
	req.Header.Set("X-Source-System", s.sourceSystem)
	req.Header.Set("X-Trace-ID", event.TraceID)
	req.Header.Set("Idempotency-Key", event.EventID)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logError("send governance event failed", event, err)
		return fmt.Errorf("send governance event failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && s.logger != nil {
			s.logger.Warn("close governance response body failed",
				zap.String("eventID", event.EventID),
				zap.Error(closeErr),
			)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusAlreadyReported {
			if s.logger != nil {
				s.logger.Info("governance event duplicate acknowledged as success",
					zap.String("eventID", event.EventID),
					zap.Int("statusCode", resp.StatusCode),
				)
			}
			return nil
		}
		err = fmt.Errorf("governance event endpoint returned status %d", resp.StatusCode)
		s.logError("governance event endpoint returned non-2xx status", event, err)
		return err
	}

	if s.logger != nil {
		s.logger.Info("governance event sent",
			zap.String("eventType", event.EventType),
			zap.String("eventID", event.EventID),
			zap.String("resourceType", event.ResourceType),
			zap.String("resourceID", event.ResourceID),
			zap.String("traceID", event.TraceID),
		)
	}

	return nil
}

func (s *GovernanceEventService) handleOutboxFailure(ctx context.Context, row *store.GovernanceOutboxEventRow, err error) {
	if row == nil {
		return
	}

	if s.outboxMaxAttempts > 0 && row.AttemptCount >= s.outboxMaxAttempts {
		if markErr := s.store.MarkGovernanceOutboxDeadLetter(ctx, row.ID, err.Error()); markErr != nil && s.logger != nil {
			s.logger.Warn("mark governance outbox dead letter failed",
				zap.String("outboxID", row.ID),
				zap.String("eventID", row.EventID),
				zap.Error(markErr),
			)
		}
		return
	}

	nextAttemptAt := s.nextOutboxAttemptAt(row.AttemptCount).Format(time.RFC3339Nano)
	if markErr := s.store.MarkGovernanceOutboxRetry(ctx, row.ID, nextAttemptAt, err.Error()); markErr != nil && s.logger != nil {
		s.logger.Warn("mark governance outbox retry failed",
			zap.String("outboxID", row.ID),
			zap.String("eventID", row.EventID),
			zap.Error(markErr),
		)
	}
}

func (s *GovernanceEventService) nextOutboxAttemptAt(attempt int) time.Time {
	if attempt < 1 {
		attempt = 1
	}

	delay := s.outboxRetryBaseDelay
	if delay <= 0 {
		delay = time.Second
	}
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= 5*time.Minute {
			delay = 5 * time.Minute
			break
		}
	}
	return time.Now().UTC().Add(delay)
}

func (s *GovernanceEventService) logError(message string, event GovernanceEvent, err error) {
	if s.logger == nil {
		return
	}
	s.logger.Error(message,
		zap.String("eventType", event.EventType),
		zap.String("eventID", event.EventID),
		zap.String("resourceType", event.ResourceType),
		zap.String("resourceID", event.ResourceID),
		zap.String("traceID", event.TraceID),
		zap.Error(err),
	)
}
