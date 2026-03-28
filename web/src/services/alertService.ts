import { api } from './api';
import type { AlertRule, AlertRuleCreate } from '../types';

export const alertService = {
  listRules: () => api.get<AlertRule[]>('/alerts/rules'),
  createRule: (data: AlertRuleCreate) =>
    api.post<AlertRule>('/alerts/rules', data),
  getRule: (id: string) => api.get<AlertRule>(`/alerts/rules/${id}`),
  updateRule: (id: string, data: AlertRuleCreate) =>
    api.put<void>(`/alerts/rules/${id}`, data),
  deleteRule: (id: string) => api.delete<void>(`/alerts/rules/${id}`),
};
