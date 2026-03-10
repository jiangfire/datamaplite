import '@testing-library/jest-dom/vitest';
import { vi, beforeAll, afterAll, afterEach } from 'vitest';

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

// Mock import.meta.env
vi.mock('import.meta.env', () => ({
  VITE_API_URL: 'http://localhost:8080/api/v1',
}));

// Clean up after each test
afterEach(() => {
  vi.clearAllMocks();
});
