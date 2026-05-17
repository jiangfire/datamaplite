import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { AuthProvider } from './AuthContext';
import * as api from '../services/api';

vi.mock('../services/api', async () => {
  const actual = await vi.importActual<typeof import('../services/api')>(
    '../services/api',
  );
  return {
    ...actual,
    restoreSession: vi.fn(),
    getStoredSession: vi.fn(),
    setStoredSession: vi.fn(),
    clearStoredSession: vi.fn(),
  };
});

vi.mock('../services', () => ({
  authService: {
    getCurrentUser: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
  },
}));

const TestChild = () => <div data-testid="child">Child</div>;

describe('AuthProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders children after bootstrap', async () => {
    vi.mocked(api.getStoredSession).mockReturnValue(null);
    vi.mocked(api.restoreSession).mockResolvedValue(null);

    render(
      <AuthProvider>
        <TestChild />
      </AuthProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('child')).toBeInTheDocument();
    });
  });

  it('restores session on mount when refresh token exists', async () => {
    vi.mocked(api.getStoredSession).mockReturnValue(null);
    vi.mocked(api.restoreSession).mockResolvedValue({
      accessToken: 'token',
      refreshToken: 'refresh',
      expiresIn: 900,
      user: null,
    });

    render(
      <AuthProvider>
        <TestChild />
      </AuthProvider>,
    );

    await waitFor(() => {
      expect(api.restoreSession).toHaveBeenCalled();
    });
  });
});
