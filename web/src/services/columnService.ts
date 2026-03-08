import { api } from './api';
import type {
  ColumnDetail,
  ColumnSearchResult,
  ColumnMapping,
  ColumnMappingCreate,
  LineageResponse,
  ImpactAnalysisResponse,
  AssignTermRequest,
} from '../types';

export const columnService = {
  // Search columns globally
  searchColumns: (query: string, limit?: number) =>
    api.get<ColumnSearchResult[]>('/columns/search', { q: query, limit }),

  // Get column detail
  getColumnDetail: (id: string) => api.get<ColumnDetail>(`/columns/${id}`),

  // Get column mappings
  getColumnMappings: (id: string) =>
    api.get<ColumnMapping[]>(`/columns/${id}/mappings`),

  // Create column mapping
  createColumnMapping: (id: string, data: ColumnMappingCreate) =>
    api.post<ColumnMapping>(`/columns/${id}/mappings`, data),

  // Delete column mapping
  deleteColumnMapping: (columnId: string, mappingId: string) =>
    api.delete<void>(`/columns/${columnId}/mappings/${mappingId}`),

  // Get lineage
  getLineage: (id: string) => api.get<LineageResponse>(`/columns/${id}/lineage`),

  // Get impact analysis
  getImpactAnalysis: (id: string) =>
    api.get<ImpactAnalysisResponse>(`/columns/${id}/impact`),

  // Assign term to column
  assignTerm: (id: string, data: AssignTermRequest) =>
    api.post<void>(`/columns/${id}/term`, data),
};
