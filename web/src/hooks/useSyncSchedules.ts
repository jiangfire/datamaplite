import { useState, useEffect } from 'react';
import { syncScheduleService, type SyncSchedule } from '../services/syncScheduleService';
import { useToast } from './useToast';

export function useSyncSchedules() {
  const [schedules, setSchedules] = useState<SyncSchedule[]>([]);
  const [loading, setLoading] = useState(false);
  const { showError, showSuccess } = useToast();

  const fetchSchedules = async () => {
    setLoading(true);
    try {
      const data = await syncScheduleService.listSchedules();
      setSchedules(data);
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to load schedules');
    } finally {
      setLoading(false);
    }
  };

  const createSchedule = async (data: Parameters<typeof syncScheduleService.createSchedule>[0]) => {
    try {
      await syncScheduleService.createSchedule(data);
      showSuccess('Schedule created successfully');
      await fetchSchedules();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to create schedule');
      throw err;
    }
  };

  const updateSchedule = async (id: string, data: Parameters<typeof syncScheduleService.updateSchedule>[1]) => {
    try {
      await syncScheduleService.updateSchedule(id, data);
      showSuccess('Schedule updated successfully');
      await fetchSchedules();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to update schedule');
      throw err;
    }
  };

  const deleteSchedule = async (id: string) => {
    try {
      await syncScheduleService.deleteSchedule(id);
      showSuccess('Schedule deleted successfully');
      await fetchSchedules();
    } catch (err) {
      showError(err instanceof Error ? err.message : 'Failed to delete schedule');
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
