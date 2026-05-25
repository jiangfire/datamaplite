-- DataMap-Lite Phase 1 数据库初始化脚本

-- 启用 pgcrypto 扩展用于 UUID
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 启用 pg_trgm 扩展用于模糊搜索
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 1. 数据源表
CREATE TABLE IF NOT EXISTS data_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL CHECK (type IN ('mysql', 'postgres', 'mongodb')),
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL CHECK (port > 0 AND port <= 65535),
    database VARCHAR(255) NOT NULL,
    connection_config TEXT NOT NULL, -- AES加密的JSON配置
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'error', 'syncing')),
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_sync_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE data_sources IS '数据源注册表';
COMMENT ON COLUMN data_sources.connection_config IS 'AES-256-GCM加密的连接配置JSON';

CREATE INDEX idx_data_sources_type ON data_sources(type);
CREATE INDEX idx_data_sources_status ON data_sources(status);

-- 2. 业务术语表
CREATE TABLE IF NOT EXISTS business_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    category VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE business_terms IS '业务术语表，用于数据标准化';

CREATE INDEX idx_business_terms_category ON business_terms(category);

-- 3. Schema对象表（表/视图/集合）
CREATE TABLE IF NOT EXISTS schema_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('table', 'view', 'collection')),
    schema VARCHAR(255), -- MySQL/PostgreSQL的schema，MongoDB为null
    description TEXT,
    row_count BIGINT,
    size_bytes BIGINT,
    column_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(source_id, name, schema)
);

COMMENT ON TABLE schema_objects IS 'Schema对象：表、视图、集合等';

CREATE INDEX idx_schema_objects_source ON schema_objects(source_id);
CREATE INDEX idx_schema_objects_name ON schema_objects(name);
CREATE INDEX idx_schema_objects_type ON schema_objects(type);

-- 4. 字段表
CREATE TABLE IF NOT EXISTS columns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id UUID NOT NULL REFERENCES schema_objects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    data_type VARCHAR(100) NOT NULL,
    full_data_type VARCHAR(255), -- 如 varchar(255)
    is_nullable BOOLEAN NOT NULL DEFAULT true,
    default_value TEXT,
    is_primary_key BOOLEAN NOT NULL DEFAULT false,
    is_unique BOOLEAN NOT NULL DEFAULT false,
    ordinal_position INTEGER NOT NULL,
    description TEXT,
    term_id UUID REFERENCES business_terms(id),
    confidence DECIMAL(3,2) DEFAULT 1.0, -- MongoDB抽样推断的置信度
    parent_column_path VARCHAR(500), -- MongoDB嵌套字段的父路径
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(object_id, name)
);

COMMENT ON TABLE columns IS '字段元数据表';
COMMENT ON COLUMN columns.confidence IS 'MongoDB抽样推断的置信度，0-1之间';
COMMENT ON COLUMN columns.parent_column_path IS 'MongoDB嵌套字段的父路径，如 address.city';

CREATE INDEX idx_columns_object ON columns(object_id);
CREATE INDEX idx_columns_term ON columns(term_id);
CREATE INDEX idx_columns_name ON columns(name);
CREATE INDEX idx_columns_name_trgm ON columns USING gin(name gin_trgm_ops);

-- 5. 字段映射表
CREATE TABLE IF NOT EXISTS column_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_column_id UUID NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    target_column_id UUID NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    mapping_type VARCHAR(50) NOT NULL DEFAULT 'alias' CHECK (mapping_type IN ('alias', 'transform', 'derived', 'synonym')),
    confidence DECIMAL(3,2) DEFAULT 1.0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(source_column_id, target_column_id)
);

COMMENT ON TABLE column_mappings IS '跨系统字段映射关系';

CREATE INDEX idx_mappings_source ON column_mappings(source_column_id);
CREATE INDEX idx_mappings_target ON column_mappings(target_column_id);

-- 6. 数据血缘表
CREATE TABLE IF NOT EXISTS lineage_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL, -- 可以是column或object的ID
    target_id UUID NOT NULL, -- 可以是column或object的ID
    source_type VARCHAR(20) NOT NULL CHECK (source_type IN ('column', 'object')),
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('column', 'object')),
    transform_sql TEXT, -- 转换SQL或逻辑
    job_name VARCHAR(255), -- ETL任务名称
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE lineage_edges IS '数据血缘关系（表级和字段级）';

CREATE INDEX idx_lineage_source ON lineage_edges(source_id, source_type);
CREATE INDEX idx_lineage_target ON lineage_edges(target_id, target_type);

-- 7. Schema变更审计表
CREATE TABLE IF NOT EXISTS schema_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    object_id UUID REFERENCES schema_objects(id) ON DELETE CASCADE,
    change_type VARCHAR(50) NOT NULL CHECK (change_type IN ('add_object', 'drop_object', 'add_column', 'drop_column', 'alter_column', 'change_type')),
    object_type VARCHAR(20) NOT NULL CHECK (object_type IN ('column', 'object')),
    object_name VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    acknowledged BOOLEAN NOT NULL DEFAULT false
);

COMMENT ON TABLE schema_changes IS 'Schema变更审计记录';

CREATE INDEX idx_schema_changes_source ON schema_changes(source_id);
CREATE INDEX idx_schema_changes_detected ON schema_changes(detected_at DESC);
CREATE INDEX idx_schema_changes_ack ON schema_changes(acknowledged);

-- 8. 用户表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE users IS '系统用户表';

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);


-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为所有表添加更新时间触发器
CREATE TRIGGER update_data_sources_updated_at BEFORE UPDATE ON data_sources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_business_terms_updated_at BEFORE UPDATE ON business_terms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_schema_objects_updated_at BEFORE UPDATE ON schema_objects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_columns_updated_at BEFORE UPDATE ON columns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 9. 数据质量规则表
CREATE TABLE IF NOT EXISTS dq_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID REFERENCES data_sources(id) ON DELETE CASCADE,
    object_id UUID REFERENCES schema_objects(id) ON DELETE CASCADE,
    column_id UUID REFERENCES columns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL CHECK (rule_type IN ('not_null', 'unique', 'regex', 'range', 'enum', 'custom_sql', 'referential')),
    rule_config JSONB NOT NULL DEFAULT '{}',
    severity VARCHAR(20) NOT NULL DEFAULT 'error' CHECK (severity IN ('error', 'warning', 'info')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE dq_rules IS '数据质量规则定义表';
COMMENT ON COLUMN dq_rules.rule_config IS '规则配置JSON，包含正则表达式、范围值、枚举值等';

CREATE INDEX idx_dq_rules_source ON dq_rules(source_id);
CREATE INDEX idx_dq_rules_object ON dq_rules(object_id);
CREATE INDEX idx_dq_rules_column ON dq_rules(column_id);
CREATE INDEX idx_dq_rules_type ON dq_rules(rule_type);
CREATE INDEX idx_dq_rules_active ON dq_rules(is_active);

-- 10. 数据质量检测结果表
CREATE TABLE IF NOT EXISTS dq_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES dq_rules(id) ON DELETE CASCADE,
    check_batch_id UUID NOT NULL,
    column_id UUID REFERENCES columns(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('passed', 'failed', 'error')),
    total_rows BIGINT NOT NULL DEFAULT 0,
    failed_rows BIGINT NOT NULL DEFAULT 0,
    pass_rate DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    sample_errors JSONB DEFAULT '[]',
    error_message TEXT,
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE dq_results IS '数据质量检测结果';
COMMENT ON COLUMN dq_results.sample_errors IS '样本错误数据，最多存储5条';
COMMENT ON COLUMN dq_results.check_batch_id IS '同一批次检查的唯一标识';

CREATE INDEX idx_dq_results_rule ON dq_results(rule_id);
CREATE INDEX idx_dq_results_batch ON dq_results(check_batch_id);
CREATE INDEX idx_dq_results_status ON dq_results(status);
CREATE INDEX idx_dq_results_checked_at ON dq_results(checked_at DESC);

-- 为dq_rules和dq_results添加更新时间触发器
CREATE TRIGGER update_dq_rules_updated_at BEFORE UPDATE ON dq_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 11. 标签表
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(7) NOT NULL DEFAULT '#6366f1',
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE tags IS '字段标签表';

CREATE INDEX idx_tags_name ON tags(name);

-- 12. 字段标签关联表（多对多）
CREATE TABLE IF NOT EXISTS column_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    column_id UUID NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(column_id, tag_id)
);

COMMENT ON TABLE column_tags IS '字段与标签的关联表';

CREATE INDEX idx_column_tags_column ON column_tags(column_id);
CREATE INDEX idx_column_tags_tag ON column_tags(tag_id);

-- 为tags表添加更新时间触发器
CREATE TRIGGER update_tags_updated_at BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 13. 告警规则表
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID REFERENCES data_sources(id) ON DELETE CASCADE,
    object_id UUID REFERENCES schema_objects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    change_types VARCHAR(255) NOT NULL DEFAULT 'all', -- 逗号分隔: add_object,drop_object,add_column,drop_column,alter_column,change_type
    notify_webhook BOOLEAN NOT NULL DEFAULT false,
    webhook_url TEXT,
    notify_in_app BOOLEAN NOT NULL DEFAULT true,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE alert_rules IS 'Schema变更告警规则';
COMMENT ON COLUMN alert_rules.change_types IS '监控的变更类型，all表示全部，或逗号分隔指定类型';
COMMENT ON COLUMN alert_rules.notify_webhook IS '是否启用Webhook通知';
COMMENT ON COLUMN alert_rules.webhook_url IS 'Webhook URL地址';

CREATE INDEX idx_alert_rules_source ON alert_rules(source_id);
CREATE INDEX idx_alert_rules_object ON alert_rules(object_id);
CREATE INDEX idx_alert_rules_active ON alert_rules(is_active);

-- 14. 通知记录表
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES alert_rules(id) ON DELETE CASCADE,
    change_id UUID NOT NULL REFERENCES schema_changes(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    change_type VARCHAR(50) NOT NULL,
    object_type VARCHAR(20) NOT NULL,
    object_name VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    webhook_sent BOOLEAN NOT NULL DEFAULT false,
    webhook_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE notifications IS '告警通知记录';

CREATE INDEX idx_notifications_rule ON notifications(rule_id);
CREATE INDEX idx_notifications_change ON notifications(change_id);
CREATE INDEX idx_notifications_source ON notifications(source_id);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- 15. 用户通知状态表
CREATE TABLE IF NOT EXISTS user_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    is_read BOOLEAN NOT NULL DEFAULT false,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, notification_id)
);

COMMENT ON TABLE user_notifications IS '用户通知读取状态';

CREATE INDEX idx_user_notifications_user ON user_notifications(user_id);
CREATE INDEX idx_user_notifications_notification ON user_notifications(notification_id);
CREATE INDEX idx_user_notifications_unread ON user_notifications(user_id, is_read);

-- 为告警规则表添加更新时间触发器
CREATE TRIGGER update_alert_rules_updated_at BEFORE UPDATE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
