// DataMap-Lite Frontend Types

// ============ DataSource Types ============
export type DataSourceType = 'mysql' | 'postgres' | 'mongodb' | 'oracle' | 'mssql';
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
  created_at: string;
  updated_at: string;
}

export interface BusinessTermCreate {
  name: string;
  description?: string;
  category?: string;
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

export interface ConnectionTestResponse {
  success: boolean;
  message: string;
}

// ============ Sync Types ============
export interface SyncResponse {
  source_id: string;
  started_at: string;
  objects_count: number;
}

// ============ API Response Types ============
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

export interface ListResponse<T> {
  total: number;
  items: T[];
}
