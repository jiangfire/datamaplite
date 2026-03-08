import axios from 'axios';
import type { AxiosInstance, AxiosError } from 'axios';
import type { ApiResponse } from '../types';

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor
apiClient.interceptors.request.use(
  (config) => {
    // Add auth token if needed
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiResponse<unknown>>) => {
    if (error.response?.data?.error) {
      return Promise.reject(new Error(error.response.data.error.message));
    }
    return Promise.reject(error);
  }
);

// Generic API methods
export const api = {
  get: <T>(url: string, params?: Record<string, unknown>) =>
    apiClient.get<ApiResponse<T>>(url, { params }).then((res) => {
      if (!res.data.success) {
        throw new Error(res.data.error?.message || 'Request failed');
      }
      return res.data.data as T;
    }),

  post: <T>(url: string, data?: unknown) =>
    apiClient.post<ApiResponse<T>>(url, data).then((res) => {
      if (!res.data.success) {
        throw new Error(res.data.error?.message || 'Request failed');
      }
      return res.data.data as T;
    }),

  put: <T>(url: string, data?: unknown) =>
    apiClient.put<ApiResponse<T>>(url, data).then((res) => {
      if (!res.data.success) {
        throw new Error(res.data.error?.message || 'Request failed');
      }
      return res.data.data as T;
    }),

  delete: <T>(url: string) =>
    apiClient.delete<ApiResponse<T>>(url).then((res) => {
      if (!res.data.success) {
        throw new Error(res.data.error?.message || 'Request failed');
      }
      return res.data.data as T;
    }),
};

export default apiClient;
