import { describe, it, expect, vi, beforeEach } from 'vitest';

// Create hoisted mock functions
const { mockGet, mockPost, mockPut, mockDelete } = vi.hoisted(() => ({
  mockGet: vi.fn(),
  mockPost: vi.fn(),
  mockPut: vi.fn(),
  mockDelete: vi.fn(),
}));

// Mock axios with hoisted mock
vi.mock('axios', async () => {
  return {
    default: {
      create: vi.fn(() => ({
        get: mockGet,
        post: mockPost,
        put: mockPut,
        delete: mockDelete,
        interceptors: {
          request: { use: vi.fn() },
          response: { use: vi.fn() },
        },
      })),
    },
  };
});

// Import after mock
import { api } from '../api';

describe('API Service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('api.get', () => {
    it('should return data on successful response', async () => {
      const mockData = { id: '1', name: 'Test' };
      mockGet.mockResolvedValue({
        data: { code: 0, data: mockData },
      });

      const result = await api.get('/test');

      expect(mockGet).toHaveBeenCalledWith('/test', { params: undefined });
      expect(result).toEqual(mockData);
    });

    it('should throw error when response is not successful', async () => {
      mockGet.mockResolvedValue({
        data: { code: 404, message: 'Not found', error_code: 'NOT_FOUND' },
      });

      await expect(api.get('/test')).rejects.toThrow('Not found');
    });

    it('should throw generic error when error message is missing', async () => {
      mockGet.mockResolvedValue({
        data: { code: 500 },
      });

      await expect(api.get('/test')).rejects.toThrow('Request failed');
    });
  });

  describe('api.post', () => {
    it('should return data on successful post', async () => {
      const mockData = { id: '1', name: 'Test' };
      const postData = { name: 'Test' };
      mockPost.mockResolvedValue({
        data: { code: 0, data: mockData },
      });

      const result = await api.post('/test', postData);

      expect(mockPost).toHaveBeenCalledWith('/test', postData);
      expect(result).toEqual(mockData);
    });

    it('should throw error on failed post', async () => {
      mockPost.mockResolvedValue({
        data: { code: 400, message: 'Invalid data', error_code: 'BAD_REQUEST' },
      });

      await expect(api.post('/test', {})).rejects.toThrow('Invalid data');
    });
  });

  describe('api.put', () => {
    it('should return data on successful put', async () => {
      const mockData = { id: '1', name: 'Updated' };
      mockPut.mockResolvedValue({
        data: { code: 0, data: mockData },
      });

      const result = await api.put('/test/1', { name: 'Updated' });

      expect(mockPut).toHaveBeenCalledWith('/test/1', { name: 'Updated' });
      expect(result).toEqual(mockData);
    });
  });

  describe('api.delete', () => {
    it('should return data on successful delete', async () => {
      mockDelete.mockResolvedValue({
        data: { code: 0, data: null },
      });

      const result = await api.delete('/test/1');

      expect(mockDelete).toHaveBeenCalledWith('/test/1');
      expect(result).toBeNull();
    });
  });
});
