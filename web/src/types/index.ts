// DataMap-Lite Frontend Types

// ============ DataSource Types ============
export type DataSourceType = 'mysql' | 'postgres' | 'mongodb';
export type DataSourceStatus = 'active' | 'inactive' | 'error' | 'syncing';

export interface DataSource {
  id: string;
  name: string;
  description?: string;
  type: DataSourceType;
  host: string;
  port: number;
  database: string;
  status: DataSourceStatus;
  last_sync_at?: string;
  last_sync_error?: string;
  created_at: string;
  updated_at: string;
}

export interface DataSourceCreate {
  name: string;
  description?: string;
  type: DataSourceType;
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
  ssl_mode?: string;
}

export interface DataSourceUpdate {
  name?: string;
  description?: string;
  host?: string;
  port?: number;
  database?: string;
  username?: string;
  password?: string;
}

// ============ Schema Types ============
export type ObjectType = 'table' | 'view' | 'collection';

export interface SchemaObject {
  id: string;
  name: string;
  type: ObjectType;
  schema?: string;
  description?: string;
  row_count?: number;
  size_bytes?: number;
  column_count: number;
}

export interface SchemaObjectWithColumns extends SchemaObject {
  columns: Column[];
}

export interface SchemaTree {
  source_id: string;
  objects: SchemaObjectWithColumns[];
}

// ============ Column Types ============
export interface Column {
  id: string;
  name: string;
  data_type: string;
  full_data_type: string;
  is_nullable: boolean;
  default_value?: string;
  is_primary_key: boolean;
  ordinal_position: number;
  description?: string;
}

export interface ColumnDetail extends Column {
  confidence: number;
  parent_column_path?: string;
  object: ObjectSummary;
  source: SourceSummary;
  term?: TermSummary;
  mapped_columns?: MappedColumn[];
  tags?: Tag[];
}

export interface ObjectSummary {
  id: string;
  name: string;
  type: string;
}

export interface SourceSummary {
  id: string;
  name: string;
  type: string;
}

export interface TermSummary {
  id: string;
  name: string;
}

export interface MappedColumn {
  id: string;
  name: string;
  object_name: string;
  source_name: string;
  mapping_type: string;
  confidence: number;
}

export interface ColumnSearchResult {
  id: string;
  name: string;
  data_type: string;
  object_name: string;
  source_id: string;
  source_name: string;
  source_type: string;
  confidence: number;
  parent_column_path?: string;
}

// ============ Mapping Types ============
export interface ColumnMapping {
  id: string;
  source_column_id: string;
  target_column_id: string;
  mapping_type: 'alias' | 'transform' | 'derived' | 'synonym';
  confidence: number;
  created_at: string;
  target_column: ColumnSummary;
}

export interface ColumnMappingCreate {
  source_column_id: string;
  target_column_id: string;
  mapping_type: 'alias' | 'transform' | 'derived' | 'synonym';
  confidence?: number;
}

export interface ColumnSummary {
  id: string;
  name: string;
  data_type: string;
  object_name: string;
  source_name: string;
}

// ============ Lineage Types ============
export interface LineageNode {
  id: string;
  name: string;
  type: 'column' | 'object';
  data_type?: string;
  source: string;
}

export interface LineageEdge {
  source: LineageNode;
  target: LineageNode;
  transform_sql?: string;
  job_name?: string;
}

export interface LineageResponse {
  column_id: string;
  upward: LineageEdge[];
  downward: LineageEdge[];
}

// ============ Impact Analysis Types ============
export interface ImpactObject {
  id: string;
  name: string;
  type: 'object' | 'column';
  object_name?: string;
  source_name: string;
  impact_path: string;
  distance: number;
}

export interface ImpactAnalysisResponse {
  column_id: string;
  impact_objects: ImpactObject[];
  total_objects: number;
}

// ============ Business Term Types ============
export interface BusinessTerm {
  id: string;
  name: string;
  description?: string;
  category?: string;
  standard_code?: string;
  domain?: string;
  data_type_standard?: string;
  validation_rule?: string;
  owner?: string;
  status?: string;
  created_at: string;
  updated_at: string;
}

export interface BusinessTermCreate {
  name: string;
  description?: string;
  category?: string;
  standard_code?: string;
  domain?: string;
  data_type_standard?: string;
  validation_rule?: string;
  owner?: string;
  status?: string;
}

export interface AssignTermRequest {
  term_id: string | null;
}

// ============ DDL Types ============
export interface DDLGenerateRequest {
  object_id: string;
  target_type: 'mysql' | 'postgres';
}

export interface DDLGenerateResponse {
  object_id: string;
  sql: string;
}

// ============ Schema Change Types ============
export interface SchemaChange {
  id: string;
  object_id?: string;
  change_type: string;
  object_type: string;
  object_name: string;
  old_value?: string;
  new_value?: string;
  detected_at: string;
  acknowledged: boolean;
}

// ============ Connection Test Types ============
export interface ConnectionTestRequest {
  type: DataSourceType;
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
}

// ============ Sync Types ============
export interface SyncResponse {
  source_id: string;
  started_at: string;
  objects_count: number;
}

// ============ API Response Types ============
export interface ApiResponse<T> {
  code: number;
  message?: string;
  error_code?: string;
  data?: T;
}

export interface ListResponse<T> {
  total: number;
  items: T[];
}

// ============ Auth Types ============
export type UserRole = 'admin' | 'user';

export interface UserInfo {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  created_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: UserInfo;
}

export interface UserCreateRequest {
  username: string;
  password: string;
  email: string;
  role?: UserRole;
}

export interface UserUpdateRequest {
  email?: string;
  password?: string;
  role?: UserRole;
}

// ============ Data Quality Types ============
export type DQRuleType =
  | 'not_null'
  | 'unique'
  | 'regex'
  | 'range'
  | 'enum'
  | 'custom_sql'
  | 'referential';
export type DQSeverity = 'error' | 'warning' | 'info';
export type DQResultStatus = 'passed' | 'failed' | 'error';

export interface DQRule {
  id: string;
  source_id?: string;
  object_id?: string;
  column_id?: string;
  name: string;
  description?: string;
  rule_type: DQRuleType;
  rule_config: Record<string, unknown>;
  severity: DQSeverity;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface DQRuleWithResult extends DQRule {
  latest_result?: DQResult;
}

export interface DQRuleCreate {
  source_id?: string;
  object_id?: string;
  column_id?: string;
  name: string;
  description?: string;
  rule_type: DQRuleType;
  rule_config: Record<string, unknown>;
  severity: DQSeverity;
  is_active?: boolean;
}

export interface DQRuleFilter {
  source_id?: string;
  object_id?: string;
  column_id?: string;
  rule_type?: DQRuleType;
  is_active?: boolean;
}

export interface DQResult {
  id: string;
  rule_id: string;
  check_batch_id: string;
  column_id?: string;
  status: DQResultStatus;
  total_rows: number;
  failed_rows: number;
  pass_rate: number;
  sample_errors?: Record<string, unknown>[];
  error_message?: string;
  checked_at: string;
}

export interface DQCheckRequest {
  rule_ids?: string[];
  source_id?: string;
  object_id?: string;
  column_id?: string;
  check_all?: boolean;
  sample_limit?: number;
}

export interface DQCheckResponse {
  batch_id: string;
  total_rules: number;
  passed_rules: number;
  failed_rules: number;
  results: DQResult[];
  checked_at: string;
}

export interface DQStats {
  total_rules: number;
  active_rules: number;
  total_checks: number;
  passed_checks: number;
  failed_checks: number;
  overall_pass_rate: number;
}

// Rule Config Types
export interface RegexRuleConfig {
  pattern: string;
  flags?: string;
}

export interface RangeRuleConfig {
  min?: number;
  max?: number;
}

export interface EnumRuleConfig {
  values: string[];
}

export interface CustomSQLRuleConfig {
  sql: string;
}

export interface ReferentialRuleConfig {
  ref_source_id?: string;
  ref_object_id: string;
  ref_column_id: string;
}

// ============ Tag Types ============
export interface Tag {
  id: string;
  name: string;
  color: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface TagCreate {
  name: string;
  color: string;
  description?: string;
}

// ============ Alert Rule Types ============
export interface AlertRule {
  id: string;
  source_id?: string;
  object_id?: string;
  source_name?: string;
  object_name?: string;
  name: string;
  description?: string;
  change_types: string;
  notify_webhook: boolean;
  webhook_url?: string;
  notify_in_app: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface AlertRuleCreate {
  source_id?: string;
  object_id?: string;
  name: string;
  description?: string;
  change_types: string;
  notify_webhook: boolean;
  webhook_url?: string;
  notify_in_app: boolean;
  is_active: boolean;
}

// ============ Notification Types ============
export interface Notification {
  id: string;
  rule_id?: string;
  rule_name?: string;
  change_id: string;
  source_id: string;
  source_name: string;
  title: string;
  message: string;
  change_type: string;
  object_type: string;
  object_name: string;
  old_value?: string;
  new_value?: string;
  webhook_sent: boolean;
  webhook_error?: string;
  is_read: boolean;
  created_at: string;
}

export interface NotificationStats {
  total_count: number;
  unread_count: number;
  today_count: number;
}

export interface MarkAsReadRequest {
  notification_ids?: string[];
  mark_all?: boolean;
}
