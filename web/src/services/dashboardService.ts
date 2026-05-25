import { api } from './api';

export interface DashboardStats {
  total_sources: number;
  total_objects: number;
  total_columns: number;
  total_terms: number;
  total_mappings: number;
  total_dq_rules: number;
  total_tags: number;
  recent_changes: number;
  total_alert_rules: number;
  total_users: number;
  active_dq_rules: number;
  overall_pass_rate: number;
  unread_notifications: number;
}

export const dashboardService = {
  getStats: (): Promise<DashboardStats> =>
    api.get<DashboardStats>('/dashboard/stats'),
};
