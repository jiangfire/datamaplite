import { useEffect, useMemo, useState } from 'react';
import { authService } from '../services';
import type { UserInfo } from '../types';
import {
  AUTH_SESSION_EVENT,
  clearStoredSession,
  getStoredSession,
  setStoredSession,
  restoreSession,
} from '../services/api';
import { AuthContext, type AuthContextValue } from './context';

const syncUserFromStorage = () => getStoredSession()?.user ?? null;

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [user, setUser] = useState<UserInfo | null>(syncUserFromStorage);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const handleSessionChange = () => {
      setUser(syncUserFromStorage());
    };

    window.addEventListener(AUTH_SESSION_EVENT, handleSessionChange);
    return () => {
      window.removeEventListener(AUTH_SESSION_EVENT, handleSessionChange);
    };
  }, []);

  useEffect(() => {
    let active = true;

    const bootstrap = async () => {
      let session = getStoredSession();
      // 页面刷新后 accessToken 在内存中丢失，尝试用 refresh_token 恢复
      if (!session) {
        session = await restoreSession();
      }
      if (!session?.accessToken) {
        if (active) setLoading(false);
        return;
      }

      try {
        const currentUser = await authService.getCurrentUser();
        if (!active) return;
        setStoredSession({
          ...session,
          user: currentUser,
        });
        setUser(currentUser);
      } catch {
        if (!active) return;
        clearStoredSession();
        setUser(null);
      } finally {
        if (active) setLoading(false);
      }
    };

    bootstrap();
    return () => {
      active = false;
    };
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated: !!user,
      loading,
      login: async (credentials) => {
        const session = await authService.login(credentials);
        setUser(session.user);
        return session;
      },
      logout: () => {
        authService.logout();
        setUser(null);
      },
    }),
    [loading, user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
