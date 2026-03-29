import { createContext } from 'react';
import type { LoginRequest, LoginResponse, UserInfo } from '../types';

export interface AuthContextValue {
  user: UserInfo | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (credentials: LoginRequest) => Promise<LoginResponse>;
  logout: () => void;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
