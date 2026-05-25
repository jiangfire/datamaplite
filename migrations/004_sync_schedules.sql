-- DataMap-Lite Phase 4: 定时同步调度表

CREATE TABLE IF NOT EXISTS sync_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMP WITH TIME ZONE,
    last_run_status VARCHAR(20) CHECK (last_run_status IN ('success', 'failed', 'running')),
    last_run_error TEXT,
    next_run_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE sync_schedules IS '数据源定时同步调度配置';
COMMENT ON COLUMN sync_schedules.cron_expression IS 'Cron 表达式，如 "0 2 * * *" 表示每天凌晨 2 点';
COMMENT ON COLUMN sync_schedules.last_run_status IS '上次执行状态: success/failed/running';

CREATE INDEX idx_sync_schedules_source ON sync_schedules(source_id);
CREATE INDEX idx_sync_schedules_active ON sync_schedules(is_active);
CREATE INDEX idx_sync_schedules_next_run ON sync_schedules(next_run_at);

CREATE TRIGGER update_sync_schedules_updated_at BEFORE UPDATE ON sync_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
