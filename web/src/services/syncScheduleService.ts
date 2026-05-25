import { api } from './api';

export interface SyncSchedule {
  id: string;
  source_id: string;
  name: string;
  description?: string;
  cron_expression: string;
  is_active: boolean;
  last_run_at?: string;
  last_run_status?: string;
  last_run_error?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

export interface SyncScheduleCreate {
  source_id: string;
  name: string;
  description?: string;
  cron_expression: string;
  is_active: boolean;
}

export interface SyncScheduleUpdate {
  name?: string;
  description?: string;
  cron_expression?: string;
  is_active?: boolean;
}

export const syncScheduleService = {
  listSchedules: (): Promise<SyncSchedule[]> =>
    api.get<SyncSchedule[]>('/sync/schedules'),

  getSchedule: (id: string): Promise<SyncSchedule> =>
    api.get<SyncSchedule>(`/sync/schedules/${id}`),

  createSchedule: (data: SyncScheduleCreate): Promise<{ id: string }> =>
    api.post<{ id: string }>('/sync/schedules', data),

  updateSchedule: (id: string, data: SyncScheduleUpdate): Promise<void> =>
    api.put<void>(`/sync/schedules/${id}`, data),

  deleteSchedule: (id: string): Promise<void> =>
    api.delete<void>(`/sync/schedules/${id}`),
};
