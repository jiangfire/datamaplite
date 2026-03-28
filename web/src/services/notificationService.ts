import { api } from './api';
import type {
  Notification,
  NotificationStats,
  MarkAsReadRequest,
} from '../types';

export const notificationService = {
  listNotifications: (unreadOnly?: boolean) =>
    api.get<Notification[]>(
      '/notifications',
      unreadOnly ? { unread_only: true } : undefined,
    ),
  getStats: () => api.get<NotificationStats>('/notifications/stats'),
  markAsRead: (data: MarkAsReadRequest) =>
    api.post<void>('/notifications/read', data),
};
