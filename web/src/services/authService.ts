import axios from 'axios';
import type { ApiResponse, LoginRequest, LoginResponse, UserInfo } from '../types';
import { API_BASE_URL, api, clearStoredSession, setStoredSession } from './api';

const authClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

const unwrapResponse = <T>(payload: ApiResponse<T>) => {
  if (payload.code !== 0) {
    throw new Error(payload.message || 'Request failed');
  }
  return payload.data as T;
};

export const authService = {
  login: async (data: LoginRequest) => {
    const response = await authClient.post<ApiResponse<LoginResponse>>('/auth/login', data);
    const session = unwrapResponse(response.data);
    setStoredSession({
      accessToken: session.access_token,
      refreshToken: session.refresh_token,
      expiresIn: session.expires_in,
      user: session.user,
    });
    return session;
  },

  getCurrentUser: () => api.get<UserInfo>('/auth/me'),

  logout: () => {
    clearStoredSession();
  },
};
