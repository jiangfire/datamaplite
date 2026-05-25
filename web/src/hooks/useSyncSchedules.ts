import { useState, useEffect } from 'react';
import { syncScheduleService, type SyncSchedule } from '../services/syncScheduleService';
import { useToast } from './useToast';

export function useSyncSchedules() {
  const [schedules, setSchedules] = useState<SyncSchedule[]>([]);
  const [loading, setLoading] = useState(false);
  const { add } = useToast();

  const fetchSchedules = async () => {
    setLoading(true);
    try {
      const data = await syncScheduleService.listSchedules();
      setSchedules(data);
    } catch (err) {
      add(err instanceof Error ? err.message : 'Failed to load schedules', 'error');
    } finally {
      setLoading(false);
    }
  };

  const createSchedule = async (data: Parameters<typeof syncScheduleService.createSchedule>[0]) => {
    try {
      await syncScheduleService.createSchedule(data);
      add('Schedule created successfully', 'success');
      await fetchSchedules();
    } catch (err) {
      add(err instanceof Error ? err.message : 'Failed to create schedule', 'error');
      throw err;
    }
  };

  const updateSchedule = async (id: string, data: Parameters<typeof syncScheduleService.updateSchedule>[1]) => {
    try {
      await syncScheduleService.updateSchedule(id, data);
      add('Schedule updated successfully', 'success');
      await fetchSchedules();
    } catch (err) {
      add(err instanceof Error ? err.message : 'Failed to update schedule', 'error');
      throw err;
    }
  };

  const deleteSchedule = async (id: string) => {
    try {
      await syncScheduleService.deleteSchedule(id);
      add('Schedule deleted successfully', 'success');
      await fetchSchedules();
    } catch (err) {
      add(err instanceof Error ? err.message : 'Failed to delete schedule', 'error');
      throw err;
    }
  };

  useEffect(() => {
    fetchSchedules();
  }, []);

  return {
    schedules,
    loading,
    fetchSchedules,
    createSchedule,
    updateSchedule,
    deleteSchedule,
  };
}
