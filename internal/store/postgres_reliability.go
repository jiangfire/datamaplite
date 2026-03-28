package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgxQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type pgxExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// TryAcquireSyncLease 尝试获取同步租约。
func (s *PostgresStore) TryAcquireSyncLease(ctx context.Context, sourceID string, ownerID string, now string, leaseUntil string) (bool, error) {
	result, err := s.pool.Exec(ctx, `
		INSERT INTO sync_leases (source_id, owner_id, lease_until, updated_at)
		VALUES ($1, $2, $3::timestamptz, NOW())
		ON CONFLICT (source_id) DO UPDATE SET
			owner_id = EXCLUDED.owner_id,
			lease_until = EXCLUDED.lease_until,
			updated_at = NOW()
		WHERE sync_leases.owner_id = EXCLUDED.owner_id OR sync_leases.lease_until <= $4::timestamptz
	`, sourceID, ownerID, leaseUntil, now)
	if err != nil {
		return false, fmt.Errorf("failed to acquire sync lease: %w", err)
	}
	return result.RowsAffected() > 0, nil
}

// GetSyncLease 获取同步租约。
func (s *PostgresStore) GetSyncLease(ctx context.Context, sourceID string) (*SyncLeaseRow, error) {
	return getPostgresSyncLease(ctx, s.pool, sourceID)
}

// RenewSyncLease 续租同步租约。
func (s *PostgresStore) RenewSyncLease(ctx context.Context, sourceID string, ownerID string, leaseUntil string) error {
	return renewPostgresSyncLease(ctx, s.pool, sourceID, ownerID, leaseUntil)
}

// ReleaseSyncLease 释放同步租约。
func (s *PostgresStore) ReleaseSyncLease(ctx context.Context, sourceID string, ownerID string) error {
	return releasePostgresSyncLease(ctx, s.pool, sourceID, ownerID)
}

// ForceReleaseSyncLease 强制释放同步租约。
func (s *PostgresStore) ForceReleaseSyncLease(ctx context.Context, sourceID string) error {
	return forceReleasePostgresSyncLease(ctx, s.pool, sourceID)
}

// EnqueueGovernanceOutboxEvent 入队治理 outbox 事件。
func (s *PostgresStore) EnqueueGovernanceOutboxEvent(ctx context.Context, event *GovernanceOutboxEventCreate) (bool, error) {
	return enqueuePostgresGovernanceOutboxEvent(ctx, s.pool, event)
}

// GetGovernanceOutboxEvent 获取治理 outbox 事件。
func (s *PostgresStore) GetGovernanceOutboxEvent(ctx context.Context, id string) (*GovernanceOutboxEventRow, error) {
	return getPostgresGovernanceOutboxEvent(ctx, s.pool, id)
}

// ClaimGovernanceOutboxEvents 认领到期的治理 outbox 事件。
func (s *PostgresStore) ClaimGovernanceOutboxEvents(ctx context.Context, ownerID string, now string, leaseUntil string, limit int) ([]*GovernanceOutboxEventRow, error) {
	return claimPostgresGovernanceOutboxEvents(ctx, s.pool, ownerID, now, leaseUntil, limit)
}

// MarkGovernanceOutboxDelivered 标记治理事件已投递。
func (s *PostgresStore) MarkGovernanceOutboxDelivered(ctx context.Context, id string, deliveredAt string) error {
	return markPostgresGovernanceOutboxDelivered(ctx, s.pool, id, deliveredAt)
}

// MarkGovernanceOutboxRetry 标记治理事件等待重试。
func (s *PostgresStore) MarkGovernanceOutboxRetry(ctx context.Context, id string, nextAttemptAt string, lastError string) error {
	return markPostgresGovernanceOutboxRetry(ctx, s.pool, id, nextAttemptAt, lastError)
}

// MarkGovernanceOutboxDeadLetter 标记治理事件进入死信。
func (s *PostgresStore) MarkGovernanceOutboxDeadLetter(ctx context.Context, id string, lastError string) error {
	return markPostgresGovernanceOutboxDeadLetter(ctx, s.pool, id, lastError)
}

// ReplayGovernanceOutboxEvent 重放治理事件。
func (s *PostgresStore) ReplayGovernanceOutboxEvent(ctx context.Context, id string, nextAttemptAt string) error {
	return replayPostgresGovernanceOutboxEvent(ctx, s.pool, id, nextAttemptAt)
}

// ListGovernanceOutboxEvents 列出治理 outbox 事件。
func (s *PostgresStore) ListGovernanceOutboxEvents(ctx context.Context, limit int) ([]*GovernanceOutboxEventRow, error) {
	return listPostgresGovernanceOutboxEvents(ctx, s.pool, limit)
}

// GetGovernanceOutboxStats 获取治理 outbox 统计。
func (s *PostgresStore) GetGovernanceOutboxStats(ctx context.Context) (*GovernanceOutboxStatsRow, error) {
	return getPostgresGovernanceOutboxStats(ctx, s.pool)
}

// TryAcquireSyncLease 尝试获取同步租约。
func (t *PostgresTxStore) TryAcquireSyncLease(ctx context.Context, sourceID string, ownerID string, now string, leaseUntil string) (bool, error) {
	result, err := t.tx.Exec(ctx, `
		INSERT INTO sync_leases (source_id, owner_id, lease_until, updated_at)
		VALUES ($1, $2, $3::timestamptz, NOW())
		ON CONFLICT (source_id) DO UPDATE SET
			owner_id = EXCLUDED.owner_id,
			lease_until = EXCLUDED.lease_until,
			updated_at = NOW()
		WHERE sync_leases.owner_id = EXCLUDED.owner_id OR sync_leases.lease_until <= $4::timestamptz
	`, sourceID, ownerID, leaseUntil, now)
	if err != nil {
		return false, fmt.Errorf("failed to acquire sync lease: %w", err)
	}
	return result.RowsAffected() > 0, nil
}

// GetSyncLease 获取同步租约。
func (t *PostgresTxStore) GetSyncLease(ctx context.Context, sourceID string) (*SyncLeaseRow, error) {
	return getPostgresSyncLease(ctx, t.tx, sourceID)
}

// RenewSyncLease 续租同步租约。
func (t *PostgresTxStore) RenewSyncLease(ctx context.Context, sourceID string, ownerID string, leaseUntil string) error {
	return renewPostgresSyncLease(ctx, t.tx, sourceID, ownerID, leaseUntil)
}

// ReleaseSyncLease 释放同步租约。
func (t *PostgresTxStore) ReleaseSyncLease(ctx context.Context, sourceID string, ownerID string) error {
	return releasePostgresSyncLease(ctx, t.tx, sourceID, ownerID)
}

// ForceReleaseSyncLease 强制释放同步租约。
func (t *PostgresTxStore) ForceReleaseSyncLease(ctx context.Context, sourceID string) error {
	return forceReleasePostgresSyncLease(ctx, t.tx, sourceID)
}

// EnqueueGovernanceOutboxEvent 入队治理 outbox 事件。
func (t *PostgresTxStore) EnqueueGovernanceOutboxEvent(ctx context.Context, event *GovernanceOutboxEventCreate) (bool, error) {
	return enqueuePostgresGovernanceOutboxEvent(ctx, t.tx, event)
}

// GetGovernanceOutboxEvent 获取治理 outbox 事件。
func (t *PostgresTxStore) GetGovernanceOutboxEvent(ctx context.Context, id string) (*GovernanceOutboxEventRow, error) {
	return getPostgresGovernanceOutboxEvent(ctx, t.tx, id)
}

// ClaimGovernanceOutboxEvents 认领到期的治理 outbox 事件。
func (t *PostgresTxStore) ClaimGovernanceOutboxEvents(ctx context.Context, ownerID string, now string, leaseUntil string, limit int) ([]*GovernanceOutboxEventRow, error) {
	return claimPostgresGovernanceOutboxEvents(ctx, t.tx, ownerID, now, leaseUntil, limit)
}

// MarkGovernanceOutboxDelivered 标记治理事件已投递。
func (t *PostgresTxStore) MarkGovernanceOutboxDelivered(ctx context.Context, id string, deliveredAt string) error {
	return markPostgresGovernanceOutboxDelivered(ctx, t.tx, id, deliveredAt)
}

// MarkGovernanceOutboxRetry 标记治理事件等待重试。
func (t *PostgresTxStore) MarkGovernanceOutboxRetry(ctx context.Context, id string, nextAttemptAt string, lastError string) error {
	return markPostgresGovernanceOutboxRetry(ctx, t.tx, id, nextAttemptAt, lastError)
}

// MarkGovernanceOutboxDeadLetter 标记治理事件进入死信。
func (t *PostgresTxStore) MarkGovernanceOutboxDeadLetter(ctx context.Context, id string, lastError string) error {
	return markPostgresGovernanceOutboxDeadLetter(ctx, t.tx, id, lastError)
}

// ReplayGovernanceOutboxEvent 重放治理事件。
func (t *PostgresTxStore) ReplayGovernanceOutboxEvent(ctx context.Context, id string, nextAttemptAt string) error {
	return replayPostgresGovernanceOutboxEvent(ctx, t.tx, id, nextAttemptAt)
}

// ListGovernanceOutboxEvents 列出治理 outbox 事件。
func (t *PostgresTxStore) ListGovernanceOutboxEvents(ctx context.Context, limit int) ([]*GovernanceOutboxEventRow, error) {
	return listPostgresGovernanceOutboxEvents(ctx, t.tx, limit)
}

// GetGovernanceOutboxStats 获取治理 outbox 统计。
func (t *PostgresTxStore) GetGovernanceOutboxStats(ctx context.Context) (*GovernanceOutboxStatsRow, error) {
	return getPostgresGovernanceOutboxStats(ctx, t.tx)
}

func getPostgresSyncLease(ctx context.Context, queryer pgxQueryer, sourceID string) (*SyncLeaseRow, error) {
	row := queryer.QueryRow(ctx, `
		SELECT source_id, owner_id, lease_until::text, updated_at::text
		FROM sync_leases
		WHERE source_id = $1
	`, sourceID)

	lease := &SyncLeaseRow{}
	if err := row.Scan(&lease.SourceID, &lease.OwnerID, &lease.LeaseUntil, &lease.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get sync lease: %w", err)
	}
	return lease, nil
}

func renewPostgresSyncLease(ctx context.Context, execer pgxExecer, sourceID string, ownerID string, leaseUntil string) error {
	result, err := execer.Exec(ctx, `
		UPDATE sync_leases
		SET lease_until = $1::timestamptz, updated_at = NOW()
		WHERE source_id = $2 AND owner_id = $3
	`, leaseUntil, sourceID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to renew sync lease: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("sync lease not held for source %s", sourceID)
	}
	return nil
}

func releasePostgresSyncLease(ctx context.Context, execer pgxExecer, sourceID string, ownerID string) error {
	if _, err := execer.Exec(ctx, `DELETE FROM sync_leases WHERE source_id = $1 AND owner_id = $2`, sourceID, ownerID); err != nil {
		return fmt.Errorf("failed to release sync lease: %w", err)
	}
	return nil
}

func forceReleasePostgresSyncLease(ctx context.Context, execer pgxExecer, sourceID string) error {
	if _, err := execer.Exec(ctx, `DELETE FROM sync_leases WHERE source_id = $1`, sourceID); err != nil {
		return fmt.Errorf("failed to force release sync lease: %w", err)
	}
	return nil
}

func enqueuePostgresGovernanceOutboxEvent(ctx context.Context, execer pgxExecer, event *GovernanceOutboxEventCreate) (bool, error) {
	if event == nil {
		return false, fmt.Errorf("governance outbox event is nil")
	}

	status := event.Status
	if status == "" {
		status = "pending"
	}

	result, err := execer.Exec(ctx, `
		INSERT INTO governance_outbox (
			id, event_id, event_type, trace_id, resource_type, resource_id, payload,
			status, attempt_count, next_attempt_at, last_error, delivered_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10::timestamptz, $11, $12::timestamptz, NOW(), NOW())
		ON CONFLICT (event_id) DO NOTHING
	`, event.ID, event.EventID, event.EventType, nullableString(event.TraceID), nullableString(event.ResourceType),
		nullableString(event.ResourceID), event.Payload, status, event.AttemptCount, event.NextAttemptAt, event.LastError, event.DeliveredAt)
	if err != nil {
		return false, fmt.Errorf("failed to enqueue governance outbox event: %w", err)
	}
	return result.RowsAffected() > 0, nil
}

func claimPostgresGovernanceOutboxEvents(ctx context.Context, queryer pgxQueryer, ownerID string, now string, leaseUntil string, limit int) ([]*GovernanceOutboxEventRow, error) {
	if limit <= 0 {
		limit = 1
	}

	rows, err := queryer.Query(ctx, `
		WITH candidates AS (
			SELECT id
			FROM governance_outbox
			WHERE status IN ('pending', 'processing')
			  AND next_attempt_at <= $1::timestamptz
			  AND (lease_until IS NULL OR lease_until <= $1::timestamptz)
			ORDER BY next_attempt_at ASC, created_at ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE governance_outbox go
		SET status = 'processing',
		    attempt_count = go.attempt_count + 1,
		    lease_owner = $3,
		    lease_until = $4::timestamptz,
		    updated_at = NOW()
		FROM candidates
		WHERE go.id = candidates.id
		RETURNING go.id, go.event_id, go.event_type, go.trace_id, go.resource_type, go.resource_id,
		          go.payload::text, go.status, go.attempt_count, go.next_attempt_at::text, go.lease_owner,
		          go.lease_until::text, go.last_error, go.delivered_at::text, go.created_at::text, go.updated_at::text
	`, now, limit, ownerID, leaseUntil)
	if err != nil {
		return nil, fmt.Errorf("failed to claim governance outbox events: %w", err)
	}
	defer rows.Close()

	result := make([]*GovernanceOutboxEventRow, 0, limit)
	for rows.Next() {
		entry, err := scanPostgresGovernanceOutboxEvent(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate claimed governance outbox events: %w", err)
	}
	return result, nil
}

func getPostgresGovernanceOutboxEvent(ctx context.Context, queryer pgxQueryer, id string) (*GovernanceOutboxEventRow, error) {
	row := queryer.QueryRow(ctx, `
		SELECT id, event_id, event_type, trace_id, resource_type, resource_id, payload::text,
		       status, attempt_count, next_attempt_at::text, lease_owner, lease_until::text, last_error,
		       delivered_at::text, created_at::text, updated_at::text
		FROM governance_outbox
		WHERE id = $1
	`, id)

	entry, err := scanPostgresGovernanceOutboxEvent(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return entry, nil
}

func markPostgresGovernanceOutboxDelivered(ctx context.Context, execer pgxExecer, id string, deliveredAt string) error {
	if _, err := execer.Exec(ctx, `
		UPDATE governance_outbox
		SET status = 'delivered',
		    delivered_at = $1::timestamptz,
		    lease_owner = NULL,
		    lease_until = NULL,
		    last_error = NULL,
		    updated_at = NOW()
		WHERE id = $2
	`, deliveredAt, id); err != nil {
		return fmt.Errorf("failed to mark governance outbox delivered: %w", err)
	}
	return nil
}

func markPostgresGovernanceOutboxRetry(ctx context.Context, execer pgxExecer, id string, nextAttemptAt string, lastError string) error {
	if _, err := execer.Exec(ctx, `
		UPDATE governance_outbox
		SET status = 'pending',
		    next_attempt_at = $1::timestamptz,
		    lease_owner = NULL,
		    lease_until = NULL,
		    last_error = $2,
		    updated_at = NOW()
		WHERE id = $3
	`, nextAttemptAt, lastError, id); err != nil {
		return fmt.Errorf("failed to mark governance outbox retry: %w", err)
	}
	return nil
}

func markPostgresGovernanceOutboxDeadLetter(ctx context.Context, execer pgxExecer, id string, lastError string) error {
	if _, err := execer.Exec(ctx, `
		UPDATE governance_outbox
		SET status = 'dead_letter',
		    lease_owner = NULL,
		    lease_until = NULL,
		    last_error = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, lastError, id); err != nil {
		return fmt.Errorf("failed to mark governance outbox dead letter: %w", err)
	}
	return nil
}

func replayPostgresGovernanceOutboxEvent(ctx context.Context, execer pgxExecer, id string, nextAttemptAt string) error {
	result, err := execer.Exec(ctx, `
		UPDATE governance_outbox
		SET status = 'pending',
		    attempt_count = 0,
		    next_attempt_at = $1::timestamptz,
		    lease_owner = NULL,
		    lease_until = NULL,
		    last_error = NULL,
		    delivered_at = NULL,
		    updated_at = NOW()
		WHERE id = $2
		  AND (status = 'dead_letter' OR last_error IS NOT NULL)
	`, nextAttemptAt, id)
	if err != nil {
		return fmt.Errorf("failed to replay governance outbox event: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("governance outbox event is not replayable: %s", id)
	}
	return nil
}

func listPostgresGovernanceOutboxEvents(ctx context.Context, queryer pgxQueryer, limit int) ([]*GovernanceOutboxEventRow, error) {
	query := `
		SELECT id, event_id, event_type, trace_id, resource_type, resource_id, payload::text,
		       status, attempt_count, next_attempt_at::text, lease_owner, lease_until::text, last_error,
		       delivered_at::text, created_at::text, updated_at::text
		FROM governance_outbox
		ORDER BY created_at ASC
	`
	args := make([]interface{}, 0, 1)
	if limit > 0 {
		query += ` LIMIT $1`
		args = append(args, limit)
	}

	rows, err := queryer.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list governance outbox events: %w", err)
	}
	defer rows.Close()

	var result []*GovernanceOutboxEventRow
	for rows.Next() {
		entry, err := scanPostgresGovernanceOutboxEvent(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate governance outbox events: %w", err)
	}
	return result, nil
}

func getPostgresGovernanceOutboxStats(ctx context.Context, queryer pgxQueryer) (*GovernanceOutboxStatsRow, error) {
	row := queryer.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'delivered' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'dead_letter' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'pending' AND next_attempt_at <= NOW() THEN 1 ELSE 0 END), 0)
		FROM governance_outbox
	`)

	stats := &GovernanceOutboxStatsRow{}
	if err := row.Scan(
		&stats.PendingCount,
		&stats.ProcessingCount,
		&stats.DeliveredCount,
		&stats.DeadLetterCount,
		&stats.RetryableCount,
	); err != nil {
		return nil, fmt.Errorf("failed to get governance outbox stats: %w", err)
	}
	return stats, nil
}

func scanPostgresGovernanceOutboxEvent(scanner interface {
	Scan(dest ...interface{}) error
}) (*GovernanceOutboxEventRow, error) {
	entry := &GovernanceOutboxEventRow{}
	var traceID *string
	var resourceType *string
	var resourceID *string
	var leaseOwner *string
	var leaseUntil *string
	var lastError *string
	var deliveredAt *string

	if err := scanner.Scan(
		&entry.ID, &entry.EventID, &entry.EventType, &traceID, &resourceType, &resourceID, &entry.Payload,
		&entry.Status, &entry.AttemptCount, &entry.NextAttemptAt, &leaseOwner, &leaseUntil, &lastError,
		&deliveredAt, &entry.CreatedAt, &entry.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan governance outbox event: %w", err)
	}

	if traceID != nil {
		entry.TraceID = *traceID
	}
	if resourceType != nil {
		entry.ResourceType = *resourceType
	}
	if resourceID != nil {
		entry.ResourceID = *resourceID
	}
	entry.LeaseOwner = leaseOwner
	entry.LeaseUntil = leaseUntil
	entry.LastError = lastError
	entry.DeliveredAt = deliveredAt

	return entry, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
