import { api } from './api';
import type {
  DataSource,
  DataSourceCreate,
  DataSourceUpdate,
  SchemaTree,
  SchemaChange,
  ConnectionTestRequest,
  ConnectionTestResponse,
  SyncResponse,
} from '../types';

export const sourceService = {
  // List all data sources
  listSources: () => api.get<DataSource[]>('/sources'),

  // Get data source by ID
  getSource: (id: string) => api.get<DataSource>(`/sources/${id}`),

  // Create new data source
  createSource: (data: DataSourceCreate) =>
    api.post<DataSource>('/sources', data),

  // Update data source
  updateSource: (id: string, data: DataSourceUpdate) =>
    api.put<void>(`/sources/${id}`, data),

  // Delete data source
  deleteSource: (id: string) => api.delete<void>(`/sources/${id}`),

  // Test connection
  testConnection: (id: string, config?: ConnectionTestRequest) =>
    api.post<ConnectionTestResponse>(`/sources/${id}/test`, config),

  // Trigger sync
  triggerSync: (id: string) => api.post<SyncResponse>(`/sources/${id}/sync`),

  // Get schema tree
  getSchemaTree: (id: string) => api.get<SchemaTree>(`/sources/${id}/schema`),

  // Get schema changes
  getSchemaChanges: (id: string) =>
    api.get<SchemaChange[]>(`/sources/${id}/changes`),
};
