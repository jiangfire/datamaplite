import { useEffect, useMemo, useState } from 'react';
import { authService } from '../services';
import type { UserInfo } from '../types';
import {
  AUTH_SESSION_EVENT,
  clearStoredSession,
  getStoredSession,
  setStoredSession,
} from '../services/api';
import { AuthContext, type AuthContextValue } from './context';

const syncUserFromStorage = () => getStoredSession()?.user ?? null;

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [user, setUser] = useState<UserInfo | null>(syncUserFromStorage);
  const [loading, setLoading] = useState(() => !!getStoredSession()?.accessToken);

  useEffect(() => {
    const handleSessionChange = () => {
      setUser(syncUserFromStorage());
      setLoading(false);
    };

    window.addEventListener(AUTH_SESSION_EVENT, handleSessionChange);
    return () => {
      window.removeEventListener(AUTH_SESSION_EVENT, handleSessionChange);
    };
  }, []);

  useEffect(() => {
    const session = getStoredSession();
    if (!session?.accessToken) {
      return;
    }

    let active = true;
    authService
      .getCurrentUser()
      .then((currentUser) => {
        if (!active) return;
        setStoredSession({
          ...session,
          user: currentUser,
        });
        setUser(currentUser);
      })
      .catch(() => {
        if (!active) return;
        clearStoredSession();
        setUser(null);
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

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
