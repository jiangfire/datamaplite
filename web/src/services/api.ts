import axios from 'axios';
import type {
  AxiosError,
  AxiosInstance,
  InternalAxiosRequestConfig,
} from 'axios';
import type { ApiResponse, LoginResponse, UserInfo } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';
const AUTH_STORAGE_KEY = 'datamap.auth';

interface StoredAuthSession {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: UserInfo | null;
}

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean;
}

const readAuthStorage = (): StoredAuthSession | null => {
  const raw = localStorage.getItem(AUTH_STORAGE_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as StoredAuthSession;
  } catch {
    localStorage.removeItem(AUTH_STORAGE_KEY);
    return null;
  }
};

export const getStoredSession = () => readAuthStorage();

export const setStoredSession = (session: StoredAuthSession) => {
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(session));
};

export const clearStoredSession = () => {
  localStorage.removeItem(AUTH_STORAGE_KEY);
};

const unwrapResponse = <T>(payload: ApiResponse<T>) => {
  if (payload.code !== 0) {
    throw new Error(payload.message || 'Request failed');
  }
  return payload.data as T;
};

const refreshClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

let refreshPromise: Promise<string | null> | null = null;

const refreshAccessToken = async (): Promise<string | null> => {
  const session = getStoredSession();
  if (!session?.refreshToken) {
    clearStoredSession();
    return null;
  }

  if (!refreshPromise) {
    refreshPromise = refreshClient
      .post<ApiResponse<LoginResponse>>('/auth/refresh', {
        refresh_token: session.refreshToken,
      })
      .then((response) => {
        const refreshed = unwrapResponse(response.data);
        setStoredSession({
          accessToken: refreshed.access_token,
          refreshToken: refreshed.refresh_token,
          expiresIn: refreshed.expires_in,
          user: refreshed.user,
        });
        return refreshed.access_token;
      })
      .catch(() => {
        clearStoredSession();
        return null;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }

  return refreshPromise;
};

// Request interceptor
apiClient.interceptors.request.use(
  (config) => {
    const token = getStoredSession()?.accessToken;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error),
);

// Response interceptor
apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiResponse<unknown>>) => {
    const originalRequest = error.config as RetryableRequestConfig | undefined;
    const requestPath = originalRequest?.url || '';

    if (
      error.response?.status === 401 &&
      originalRequest &&
      !originalRequest._retry &&
      !requestPath.includes('/auth/login') &&
      !requestPath.includes('/auth/refresh')
    ) {
      originalRequest._retry = true;
      const newAccessToken = await refreshAccessToken();

      if (newAccessToken) {
        originalRequest.headers = originalRequest.headers || {};
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
        return apiClient(originalRequest);
      }
    }

    if (error.response?.data) {
      return Promise.reject(new Error(error.response.data.message || 'Request failed'));
    }
    return Promise.reject(error);
  },
);

// Generic API methods
export const api = {
  get: <T>(url: string, params?: Record<string, unknown>) =>
    apiClient
      .get<ApiResponse<T>>(url, { params })
      .then((res) => unwrapResponse(res.data)),

  post: <T>(url: string, data?: unknown) =>
    apiClient.post<ApiResponse<T>>(url, data).then((res) => unwrapResponse(res.data)),

  put: <T>(url: string, data?: unknown) =>
    apiClient.put<ApiResponse<T>>(url, data).then((res) => unwrapResponse(res.data)),

  delete: <T>(url: string) =>
    apiClient.delete<ApiResponse<T>>(url).then((res) => unwrapResponse(res.data)),
};

export default apiClient;
