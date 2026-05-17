import axios from 'axios';
import type {
  AxiosError,
  AxiosInstance,
  InternalAxiosRequestConfig,
} from 'axios';
import type { ApiResponse, LoginResponse, UserInfo } from '../types';

export const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';
const AUTH_STORAGE_KEY = 'datamap.auth';
export const AUTH_SESSION_EVENT = 'datamap-auth-session-changed';

interface StoredAuthSession {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: UserInfo | null;
}

interface PersistedAuthData {
  refreshToken: string;
  expiresIn: number;
  user: UserInfo | null;
}

// Access token 保存在内存中，降低 XSS 泄露风险（B16）。
let accessTokenMemory: string | null = null;

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean;
}

const readPersistedData = (): PersistedAuthData | null => {
  const raw = localStorage.getItem(AUTH_STORAGE_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<StoredAuthSession> & Partial<PersistedAuthData>;
    // 兼容旧格式：旧格式同时存了 accessToken，迁移时忽略它
    if (!parsed.refreshToken) return null;
    return {
      refreshToken: parsed.refreshToken,
      expiresIn: parsed.expiresIn ?? 0,
      user: parsed.user ?? null,
    };
  } catch {
    localStorage.removeItem(AUTH_STORAGE_KEY);
    return null;
  }
};

const notifyAuthSessionChanged = () => {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(AUTH_SESSION_EVENT));
  }
};

export const getStoredSession = (): StoredAuthSession | null => {
  const persisted = readPersistedData();
  if (!persisted || !accessTokenMemory) return null;
  return {
    accessToken: accessTokenMemory,
    refreshToken: persisted.refreshToken,
    expiresIn: persisted.expiresIn,
    user: persisted.user,
  };
};

export const setStoredSession = (session: StoredAuthSession) => {
  accessTokenMemory = session.accessToken;
  const persisted: PersistedAuthData = {
    refreshToken: session.refreshToken,
    expiresIn: session.expiresIn,
    user: session.user,
  };
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(persisted));
  notifyAuthSessionChanged();
};

export const clearStoredSession = () => {
  accessTokenMemory = null;
  localStorage.removeItem(AUTH_STORAGE_KEY);
  notifyAuthSessionChanged();
};

// 页面刷新后尝试用 refresh_token 恢复 access_token
export const restoreSession = async (): Promise<StoredAuthSession | null> => {
  const persisted = readPersistedData();
  if (!persisted?.refreshToken) return null;
  if (accessTokenMemory) return getStoredSession();
  const newToken = await refreshAccessToken();
  if (!newToken) return null;
  return getStoredSession();
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

const redirectToLogin = () => {
  if (typeof window === 'undefined') {
    return;
  }
  const { pathname, search } = window.location;
  if (pathname === '/login') {
    return;
  }
  const next = encodeURIComponent(`${pathname}${search}`);
  window.location.assign(`/login?next=${next}`);
};

const refreshAccessToken = async (): Promise<string | null> => {
  const persisted = readPersistedData();
  if (!persisted?.refreshToken) {
    clearStoredSession();
    redirectToLogin();
    return null;
  }

  if (!refreshPromise) {
    refreshPromise = refreshClient
      .post<ApiResponse<LoginResponse>>('/auth/refresh', {
        refresh_token: persisted.refreshToken,
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
        redirectToLogin();
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
