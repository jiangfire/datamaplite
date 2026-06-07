DELETE FROM user_notifications un
USING notifications n, notifications keep_n
WHERE un.notification_id = n.id
  AND n.rule_id IS NOT NULL
  AND keep_n.rule_id = n.rule_id
  AND keep_n.change_id = n.change_id
  AND (
    keep_n.created_at < n.created_at OR
    (keep_n.created_at = n.created_at AND keep_n.id < n.id)
  );

DELETE FROM notifications n
USING notifications keep_n
WHERE n.rule_id IS NOT NULL
  AND keep_n.rule_id = n.rule_id
  AND keep_n.change_id = n.change_id
  AND (
    keep_n.created_at < n.created_at OR
    (keep_n.created_at = n.created_at AND keep_n.id < n.id)
  );

CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_rule_change
ON notifications(rule_id, change_id)
WHERE rule_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS sync_leases (
    source_id UUID PRIMARY KEY REFERENCES data_sources(id) ON DELETE CASCADE,
    owner_id TEXT NOT NULL,
    lease_until TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_leases_owner ON sync_leases(owner_id);
CREATE INDEX IF NOT EXISTS idx_sync_leases_until ON sync_leases(lease_until);

CREATE TABLE IF NOT EXISTS governance_outbox (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    trace_id TEXT,
    resource_type TEXT,
    resource_id TEXT,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    lease_owner TEXT,
    lease_until TIMESTAMPTZ,
    last_error TEXT,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_governance_outbox_status_next_attempt
ON governance_outbox(status, next_attempt_at);

CREATE INDEX IF NOT EXISTS idx_governance_outbox_event_id
ON governance_outbox(event_id);
