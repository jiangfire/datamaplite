import { api } from './api';
import type {
  UserInfo,
  UserCreateRequest,
  UserUpdateRequest,
} from '../types';

export const userService = {
  listUsers: () => api.get<UserInfo[]>('/auth/users'),

  createUser: (data: UserCreateRequest) =>
    api.post<UserInfo>('/auth/register', data),

  updateUser: (id: string, data: UserUpdateRequest) =>
    api.put<UserInfo>(`/auth/users/${id}`, data),

  deleteUser: (id: string) => api.delete<void>(`/auth/users/${id}`),
};
