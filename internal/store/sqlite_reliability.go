package store

import (
	"context"
	"database/sql"
	"fmt"
)

type sqliteQueryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type sqliteExecer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// TryAcquireSyncLease 尝试获取同步租约。
func (s *SQLiteStore) TryAcquireSyncLease(ctx context.Context, sourceID string, ownerID string, now string, leaseUntil string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_leases (source_id, owner_id, lease_until, updated_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(source_id) DO UPDATE SET
			owner_id = excluded.owner_id,
			lease_until = excluded.lease_until,
			updated_at = datetime('now')
		WHERE sync_leases.owner_id = excluded.owner_id OR sync_leases.lease_until <= ?
	`, sourceID, ownerID, leaseUntil, now)
	if err != nil {
		return false, fmt.Errorf("failed to acquire sync lease: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to inspect sync lease acquisition result: %w", err)
	}
	return affected > 0, nil
}

// GetSyncLease 获取同步租约。
func (s *SQLiteStore) GetSyncLease(ctx context.Context, sourceID string) (*SyncLeaseRow, error) {
	return getSQLiteSyncLease(ctx, s.db, sourceID)
}

// RenewSyncLease 续租同步租约。
func (s *SQLiteStore) RenewSyncLease(ctx context.Context, sourceID string, ownerID string, leaseUntil string) error {
	return renewSQLiteSyncLease(ctx, s.db, sourceID, ownerID, leaseUntil)
}

// ReleaseSyncLease 释放同步租约。
func (s *SQLiteStore) ReleaseSyncLease(ctx context.Context, sourceID string, ownerID string) error {
	return releaseSQLiteSyncLease(ctx, s.db, sourceID, ownerID)
}

// ForceReleaseSyncLease 强制释放同步租约。
func (s *SQLiteStore) ForceReleaseSyncLease(ctx context.Context, sourceID string) error {
	return forceReleaseSQLiteSyncLease(ctx, s.db, sourceID)
}

// EnqueueGovernanceOutboxEvent 入队治理 outbox 事件。
func (s *SQLiteStore) EnqueueGovernanceOutboxEvent(ctx context.Context, event *GovernanceOutboxEventCreate) (bool, error) {
	return enqueueSQLiteGovernanceOutboxEvent(ctx, s.db, event)
}

// GetGovernanceOutboxEvent 获取治理 outbox 事件。
func (s *SQLiteStore) GetGovernanceOutboxEvent(ctx context.Context, id string) (*GovernanceOutboxEventRow, error) {
	return getSQLiteGovernanceOutboxEventByID(ctx, s.db, id)
}

// ClaimGovernanceOutboxEvents 认领到期的治理 outbox 事件。
func (s *SQLiteStore) ClaimGovernanceOutboxEvents(ctx context.Context, ownerID string, now string, leaseUntil string, limit int) ([]*GovernanceOutboxEventRow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin governance outbox claim tx: %w", err)
	}
	defer rollbackSQLTx(tx, s.log)

	rows, err := claimSQLiteGovernanceOutboxEvents(ctx, tx, tx, ownerID, now, leaseUntil, limit)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit governance outbox claim tx: %w", err)
	}
	return rows, nil
}

// MarkGovernanceOutboxDelivered 标记治理事件已投递。
func (s *SQLiteStore) MarkGovernanceOutboxDelivered(ctx context.Context, id string, deliveredAt string) error {
	return markSQLiteGovernanceOutboxDelivered(ctx, s.db, id, deliveredAt)
}

// MarkGovernanceOutboxRetry 标记治理事件等待重试。
func (s *SQLiteStore) MarkGovernanceOutboxRetry(ctx context.Context, id string, nextAttemptAt string, lastError string) error {
	return markSQLiteGovernanceOutboxRetry(ctx, s.db, id, nextAttemptAt, lastError)
}

// MarkGovernanceOutboxDeadLetter 标记治理事件进入死信。
func (s *SQLiteStore) MarkGovernanceOutboxDeadLetter(ctx context.Context, id string, lastError string) error {
	return markSQLiteGovernanceOutboxDeadLetter(ctx, s.db, id, lastError)
}

// ReplayGovernanceOutboxEvent 重放治理事件。
func (s *SQLiteStore) ReplayGovernanceOutboxEvent(ctx context.Context, id string, nextAttemptAt string) error {
	return replaySQLiteGovernanceOutboxEvent(ctx, s.db, id, nextAttemptAt)
}

// ListGovernanceOutboxEvents 列出治理 outbox 事件。
func (s *SQLiteStore) ListGovernanceOutboxEvents(ctx context.Context, limit int) ([]*GovernanceOutboxEventRow, error) {
	return listSQLiteGovernanceOutboxEvents(ctx, s.db, limit)
}

// GetGovernanceOutboxStats 获取治理 outbox 统计。
func (s *SQLiteStore) GetGovernanceOutboxStats(ctx context.Context) (*GovernanceOutboxStatsRow, error) {
	return getSQLiteGovernanceOutboxStats(ctx, s.db)
}

// TryAcquireSyncLease 尝试获取同步租约。
func (t *SQLiteTxStore) TryAcquireSyncLease(ctx context.Context, sourceID string, ownerID string, now string, leaseUntil string) (bool, error) {
	result, err := t.tx.ExecContext(ctx, `
		INSERT INTO sync_leases (source_id, owner_id, lease_until, updated_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(source_id) DO UPDATE SET
			owner_id = excluded.owner_id,
			lease_until = excluded.lease_until,
			updated_at = datetime('now')
		WHERE sync_leases.owner_id = excluded.owner_id OR sync_leases.lease_until <= ?
	`, sourceID, ownerID, leaseUntil, now)
	if err != nil {
		return false, fmt.Errorf("failed to acquire sync lease: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to inspect sync lease acquisition result: %w", err)
	}
	return affected > 0, nil
}

// GetSyncLease 获取同步租约。
func (t *SQLiteTxStore) GetSyncLease(ctx context.Context, sourceID string) (*SyncLeaseRow, error) {
	return getSQLiteSyncLease(ctx, t.tx, sourceID)
}

// RenewSyncLease 续租同步租约。
func (t *SQLiteTxStore) RenewSyncLease(ctx context.Context, sourceID string, ownerID string, leaseUntil string) error {
	return renewSQLiteSyncLease(ctx, t.tx, sourceID, ownerID, leaseUntil)
}

// ReleaseSyncLease 释放同步租约。
func (t *SQLiteTxStore) ReleaseSyncLease(ctx context.Context, sourceID string, ownerID string) error {
	return releaseSQLiteSyncLease(ctx, t.tx, sourceID, ownerID)
}

// ForceReleaseSyncLease 强制释放同步租约。
func (t *SQLiteTxStore) ForceReleaseSyncLease(ctx context.Context, sourceID string) error {
	return forceReleaseSQLiteSyncLease(ctx, t.tx, sourceID)
}

// EnqueueGovernanceOutboxEvent 入队治理 outbox 事件。
func (t *SQLiteTxStore) EnqueueGovernanceOutboxEvent(ctx context.Context, event *GovernanceOutboxEventCreate) (bool, error) {
	return enqueueSQLiteGovernanceOutboxEvent(ctx, t.tx, event)
}

// GetGovernanceOutboxEvent 获取治理 outbox 事件。
func (t *SQLiteTxStore) GetGovernanceOutboxEvent(ctx context.Context, id string) (*GovernanceOutboxEventRow, error) {
	return getSQLiteGovernanceOutboxEventByID(ctx, t.tx, id)
}

// ClaimGovernanceOutboxEvents 认领到期的治理 outbox 事件。
func (t *SQLiteTxStore) ClaimGovernanceOutboxEvents(ctx context.Context, ownerID string, now string, leaseUntil string, limit int) ([]*GovernanceOutboxEventRow, error) {
	return claimSQLiteGovernanceOutboxEvents(ctx, t.tx, t.tx, ownerID, now, leaseUntil, limit)
}

// MarkGovernanceOutboxDelivered 标记治理事件已投递。
func (t *SQLiteTxStore) MarkGovernanceOutboxDelivered(ctx context.Context, id string, deliveredAt string) error {
	return markSQLiteGovernanceOutboxDelivered(ctx, t.tx, id, deliveredAt)
}

// MarkGovernanceOutboxRetry 标记治理事件等待重试。
func (t *SQLiteTxStore) MarkGovernanceOutboxRetry(ctx context.Context, id string, nextAttemptAt string, lastError string) error {
	return markSQLiteGovernanceOutboxRetry(ctx, t.tx, id, nextAttemptAt, lastError)
}

// MarkGovernanceOutboxDeadLetter 标记治理事件进入死信。
func (t *SQLiteTxStore) MarkGovernanceOutboxDeadLetter(ctx context.Context, id string, lastError string) error {
	return markSQLiteGovernanceOutboxDeadLetter(ctx, t.tx, id, lastError)
}

// ReplayGovernanceOutboxEvent 重放治理事件。
func (t *SQLiteTxStore) ReplayGovernanceOutboxEvent(ctx context.Context, id string, nextAttemptAt string) error {
	return replaySQLiteGovernanceOutboxEvent(ctx, t.tx, id, nextAttemptAt)
}

// ListGovernanceOutboxEvents 列出治理 outbox 事件。
func (t *SQLiteTxStore) ListGovernanceOutboxEvents(ctx context.Context, limit int) ([]*GovernanceOutboxEventRow, error) {
	return listSQLiteGovernanceOutboxEvents(ctx, t.tx, limit)
}

// GetGovernanceOutboxStats 获取治理 outbox 统计。
func (t *SQLiteTxStore) GetGovernanceOutboxStats(ctx context.Context) (*GovernanceOutboxStatsRow, error) {
	return getSQLiteGovernanceOutboxStats(ctx, t.tx)
}

func getSQLiteSyncLease(ctx context.Context, queryer sqliteQueryer, sourceID string) (*SyncLeaseRow, error) {
	row := queryer.QueryRowContext(ctx, `
		SELECT source_id, owner_id, lease_until, updated_at
		FROM sync_leases
		WHERE source_id = ?
	`, sourceID)

	lease := &SyncLeaseRow{}
	if err := row.Scan(&lease.SourceID, &lease.OwnerID, &lease.LeaseUntil, &lease.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get sync lease: %w", err)
	}
	return lease, nil
}

func renewSQLiteSyncLease(ctx context.Context, execer sqliteExecer, sourceID string, ownerID string, leaseUntil string) error {
	result, err := execer.ExecContext(ctx, `
		UPDATE sync_leases
		SET lease_until = ?, updated_at = datetime('now')
		WHERE source_id = ? AND owner_id = ?
	`, leaseUntil, sourceID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to renew sync lease: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect sync lease renewal result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("sync lease not held for source %s", sourceID)
	}
	return nil
}

func releaseSQLiteSyncLease(ctx context.Context, execer sqliteExecer, sourceID string, ownerID string) error {
	if _, err := execer.ExecContext(ctx, `DELETE FROM sync_leases WHERE source_id = ? AND owner_id = ?`, sourceID, ownerID); err != nil {
		return fmt.Errorf("failed to release sync lease: %w", err)
	}
	return nil
}

func forceReleaseSQLiteSyncLease(ctx context.Context, execer sqliteExecer, sourceID string) error {
	if _, err := execer.ExecContext(ctx, `DELETE FROM sync_leases WHERE source_id = ?`, sourceID); err != nil {
		return fmt.Errorf("failed to force release sync lease: %w", err)
	}
	return nil
}

func enqueueSQLiteGovernanceOutboxEvent(ctx context.Context, execer sqliteExecer, event *GovernanceOutboxEventCreate) (bool, error) {
	if event == nil {
		return false, fmt.Errorf("governance outbox event is nil")
	}

	status := event.Status
	if status == "" {
		status = "pending"
	}

	result, err := execer.ExecContext(ctx, `
		INSERT OR IGNORE INTO governance_outbox (
			id, event_id, event_type, trace_id, resource_type, resource_id, payload,
			status, attempt_count, next_attempt_at, last_error, delivered_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, event.ID, event.EventID, event.EventType, event.TraceID, event.ResourceType, event.ResourceID, event.Payload,
		status, event.AttemptCount, event.NextAttemptAt, event.LastError, event.DeliveredAt)
	if err != nil {
		return false, fmt.Errorf("failed to enqueue governance outbox event: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to inspect governance outbox enqueue result: %w", err)
	}
	return affected > 0, nil
}

func claimSQLiteGovernanceOutboxEvents(ctx context.Context, queryer sqliteQueryer, execer sqliteExecer, ownerID string, now string, leaseUntil string, limit int) ([]*GovernanceOutboxEventRow, error) {
	if limit <= 0 {
		limit = 1
	}

	rows, err := queryer.QueryContext(ctx, `
		SELECT id
		FROM governance_outbox
		WHERE status IN ('pending', 'processing')
		  AND next_attempt_at <= ?
		  AND (lease_until IS NULL OR lease_until <= ?)
		ORDER BY next_attempt_at ASC, created_at ASC
		LIMIT ?
	`, now, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list claimable governance outbox events: %w", err)
	}
	defer closeSQLRows(rows)

	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan governance outbox id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate governance outbox ids: %w", err)
	}

	claimed := make([]*GovernanceOutboxEventRow, 0, len(ids))
	for _, id := range ids {
		result, err := execer.ExecContext(ctx, `
			UPDATE governance_outbox
			SET status = 'processing',
			    attempt_count = attempt_count + 1,
			    lease_owner = ?,
			    lease_until = ?,
			    updated_at = datetime('now')
			WHERE id = ?
			  AND status IN ('pending', 'processing')
			  AND next_attempt_at <= ?
			  AND (lease_until IS NULL OR lease_until <= ?)
		`, ownerID, leaseUntil, id, now, now)
		if err != nil {
			return nil, fmt.Errorf("failed to claim governance outbox event %s: %w", id, err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to inspect governance outbox claim result: %w", err)
		}
		if affected == 0 {
			continue
		}

		entry, err := getSQLiteGovernanceOutboxEventByID(ctx, queryer, id)
		if err != nil {
			return nil, err
		}
		if entry != nil {
			claimed = append(claimed, entry)
		}
	}

	return claimed, nil
}

func markSQLiteGovernanceOutboxDelivered(ctx context.Context, execer sqliteExecer, id string, deliveredAt string) error {
	if _, err := execer.ExecContext(ctx, `
		UPDATE governance_outbox
		SET status = 'delivered',
		    delivered_at = ?,
		    lease_owner = NULL,
		    lease_until = NULL,
		    last_error = NULL,
		    updated_at = datetime('now')
		WHERE id = ?
	`, deliveredAt, id); err != nil {
		return fmt.Errorf("failed to mark governance outbox delivered: %w", err)
	}
	return nil
}

func markSQLiteGovernanceOutboxRetry(ctx context.Context, execer sqliteExecer, id string, nextAttemptAt string, lastError string) error {
	if _, err := execer.ExecContext(ctx, `
		UPDATE governance_outbox
		SET status = 'pending',
		    next_attempt_at = ?,
		    lease_owner = NULL,
		    lease_until = NULL,
		    last_error = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`, nextAttemptAt, lastError, id); err != nil {
		return fmt.Errorf("failed to mark governance outbox retry: %w", err)
	}
	return nil
}

func markSQLiteGovernanceOutboxDeadLetter(ctx context.Context, execer sqliteExecer, id string, lastError string) error {
	if _, err := execer.ExecContext(ctx, `
		UPDATE governance_outbox
		SET status = 'dead_letter',
		    lease_owner = NULL,
		    lease_until = NULL,
		    last_error = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`, lastError, id); err != nil {
		return fmt.Errorf("failed to mark governance outbox dead letter: %w", err)
	}
	return nil
}

func replaySQLiteGovernanceOutboxEvent(ctx context.Context, execer sqliteExecer, id string, nextAttemptAt string) error {
	result, err := execer.ExecContext(ctx, `
		UPDATE governance_outbox
		SET status = 'pending',
		    attempt_count = 0,
		    next_attempt_at = ?,
		    lease_owner = NULL,
		    lease_until = NULL,
		    last_error = NULL,
		    delivered_at = NULL,
		    updated_at = datetime('now')
		WHERE id = ?
		  AND (status = 'dead_letter' OR last_error IS NOT NULL)
	`, nextAttemptAt, id)
	if err != nil {
		return fmt.Errorf("failed to replay governance outbox event: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect governance outbox replay result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("governance outbox event is not replayable: %s", id)
	}
	return nil
}

func listSQLiteGovernanceOutboxEvents(ctx context.Context, queryer sqliteQueryer, limit int) ([]*GovernanceOutboxEventRow, error) {
	query := `
		SELECT id, event_id, event_type, trace_id, resource_type, resource_id, payload,
		       status, attempt_count, next_attempt_at, lease_owner, lease_until, last_error,
		       delivered_at, created_at, updated_at
		FROM governance_outbox
		ORDER BY created_at ASC
	`
	args := make([]interface{}, 0, 1)
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list governance outbox events: %w", err)
	}
	defer closeSQLRows(rows)

	var result []*GovernanceOutboxEventRow
	for rows.Next() {
		entry, err := scanSQLiteGovernanceOutboxEvent(rows)
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

func getSQLiteGovernanceOutboxStats(ctx context.Context, queryer sqliteQueryer) (*GovernanceOutboxStatsRow, error) {
	row := queryer.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'processing' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'delivered' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'dead_letter' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'pending' AND julianday(next_attempt_at) <= julianday('now') THEN 1 ELSE 0 END)
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

func getSQLiteGovernanceOutboxEventByID(ctx context.Context, queryer sqliteQueryer, id string) (*GovernanceOutboxEventRow, error) {
	row := queryer.QueryRowContext(ctx, `
		SELECT id, event_id, event_type, trace_id, resource_type, resource_id, payload,
		       status, attempt_count, next_attempt_at, lease_owner, lease_until, last_error,
		       delivered_at, created_at, updated_at
		FROM governance_outbox
		WHERE id = ?
	`, id)

	entry, err := scanSQLiteGovernanceOutboxEvent(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return entry, nil
}

type sqliteScanner interface {
	Scan(dest ...interface{}) error
}

func scanSQLiteGovernanceOutboxEvent(scanner sqliteScanner) (*GovernanceOutboxEventRow, error) {
	entry := &GovernanceOutboxEventRow{}
	var traceID sql.NullString
	var resourceType sql.NullString
	var resourceID sql.NullString
	var leaseOwner sql.NullString
	var leaseUntil sql.NullString
	var lastError sql.NullString
	var deliveredAt sql.NullString

	if err := scanner.Scan(
		&entry.ID, &entry.EventID, &entry.EventType, &traceID, &resourceType, &resourceID, &entry.Payload,
		&entry.Status, &entry.AttemptCount, &entry.NextAttemptAt, &leaseOwner, &leaseUntil, &lastError,
		&deliveredAt, &entry.CreatedAt, &entry.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan governance outbox event: %w", err)
	}

	if traceID.Valid {
		entry.TraceID = traceID.String
	}
	if resourceType.Valid {
		entry.ResourceType = resourceType.String
	}
	if resourceID.Valid {
		entry.ResourceID = resourceID.String
	}
	if leaseOwner.Valid {
		entry.LeaseOwner = &leaseOwner.String
	}
	if leaseUntil.Valid {
		entry.LeaseUntil = &leaseUntil.String
	}
	if lastError.Valid {
		entry.LastError = &lastError.String
	}
	if deliveredAt.Valid {
		entry.DeliveredAt = &deliveredAt.String
	}

	return entry, nil
}
