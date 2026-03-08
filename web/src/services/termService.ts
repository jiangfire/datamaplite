import { api } from './api';
import type {
  BusinessTerm,
  BusinessTermCreate,
  DDLGenerateRequest,
  DDLGenerateResponse,
} from '../types';

export const termService = {
  // List all business terms
  listTerms: (category?: string) =>
    api.get<BusinessTerm[]>('/terms', category ? { category } : undefined),

  // Get business term by ID
  getTerm: (id: string) => api.get<BusinessTerm>(`/terms/${id}`),

  // Create new business term
  createTerm: (data: BusinessTermCreate) =>
    api.post<BusinessTerm>('/terms', data),

  // Update business term
  updateTerm: (id: string, data: BusinessTermCreate) =>
    api.put<void>(`/terms/${id}`, data),

  // Delete business term
  deleteTerm: (id: string) => api.delete<void>(`/terms/${id}`),

  // Generate DDL
  generateDDL: (data: DDLGenerateRequest) =>
    api.post<DDLGenerateResponse>('/ddl/generate', data),
};
