import { useState, useEffect, useCallback } from 'react';
import { notificationService } from '../services';
import type { Notification, NotificationStats } from '../types';

export const useNotifications = (unreadOnly?: boolean) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [stats, setStats] = useState<NotificationStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchNotifications = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [data, statsData] = await Promise.all([
        notificationService.listNotifications(unreadOnly),
        notificationService.getStats(),
      ]);
      setNotifications(data);
      setStats(statsData);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to fetch notifications',
      );
    } finally {
      setLoading(false);
    }
  }, [unreadOnly]);

  useEffect(() => {
    fetchNotifications();
  }, [fetchNotifications]);

  const markAsRead = async (notificationIds: string[]) => {
    await notificationService.markAsRead({
      notification_ids: notificationIds,
    });
    await fetchNotifications();
  };

  const markAllAsRead = async () => {
    await notificationService.markAsRead({ mark_all: true });
    await fetchNotifications();
  };

  return {
    notifications,
    stats,
    loading,
    error,
    refetch: fetchNotifications,
    markAsRead,
    markAllAsRead,
  };
};

export const useNotificationStats = () => {
  const [stats, setStats] = useState<NotificationStats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchStats = useCallback(async () => {
    setLoading(true);
    try {
      const data = await notificationService.getStats();
      setStats(data);
    } catch (err) {
      console.error('Failed to fetch notification stats:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  return { stats, loading, refetch: fetchStats };
};
