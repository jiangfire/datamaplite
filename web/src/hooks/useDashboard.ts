import { useState, useEffect } from 'react';
import { dashboardService, type DashboardStats, type ChangeTrendPoint } from '../services/dashboardService';
import { useToast } from './useToast';

export function useDashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [changeTrend, setChangeTrend] = useState<ChangeTrendPoint[]>([]);
  const [loading, setLoading] = useState(false);
  const { add } = useToast();

  const fetchStats = async () => {
    setLoading(true);
    try {
      const [statsData, trendData] = await Promise.all([
        dashboardService.getStats(),
        dashboardService.getChangeTrend(30),
      ]);
      setStats(statsData);
      setChangeTrend(trendData);
    } catch (err) {
      add(err instanceof Error ? err.message : 'Failed to load dashboard', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  return { stats, changeTrend, loading, fetchStats };
}
