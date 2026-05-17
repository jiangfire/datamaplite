import { describe, it, expect, vi, beforeEach } from 'vitest';
import { sourceService } from '../sourceService';
import { api } from '../api';
import type {
  DataSource,
  DataSourceCreate,
  DataSourceUpdate,
  SchemaTree,
  SchemaChange,
  SyncResponse,
} from '../../types';

// Mock the api module
vi.mock('../api', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('SourceService', () => {
  const mockSource: DataSource = {
    id: '1',
    name: 'Test Source',
    type: 'mysql',
    host: 'localhost',
    port: 3306,
    database: 'test_db',
    status: 'active',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('listSources', () => {
    it('should return list of data sources', async () => {
      const mockSources = [mockSource];
      vi.mocked(api.get).mockResolvedValue(mockSources);

      const result = await sourceService.listSources();

      expect(api.get).toHaveBeenCalledWith('/sources');
      expect(result).toEqual(mockSources);
    });

    it('should handle empty list', async () => {
      vi.mocked(api.get).mockResolvedValue([]);

      const result = await sourceService.listSources();

      expect(result).toEqual([]);
    });
  });

  describe('getSource', () => {
    it('should return a data source by id', async () => {
      vi.mocked(api.get).mockResolvedValue(mockSource);

      const result = await sourceService.getSource('1');

      expect(api.get).toHaveBeenCalledWith('/sources/1');
      expect(result).toEqual(mockSource);
    });
  });

  describe('createSource', () => {
    it('should create a new data source', async () => {
      const createData: DataSourceCreate = {
        name: 'New Source',
        type: 'mysql',
        host: 'localhost',
        port: 3306,
        database: 'new_db',
        username: 'admin',
        password: 'secret',
      };
      vi.mocked(api.post).mockResolvedValue(mockSource);

      const result = await sourceService.createSource(createData);

      expect(api.post).toHaveBeenCalledWith('/sources', createData);
      expect(result).toEqual(mockSource);
    });
  });

  describe('updateSource', () => {
    it('should update a data source', async () => {
      const updateData: DataSourceUpdate = {
        name: 'Updated Source',
      };
      vi.mocked(api.put).mockResolvedValue(undefined);

      await sourceService.updateSource('1', updateData);

      expect(api.put).toHaveBeenCalledWith('/sources/1', updateData);
    });
  });

  describe('deleteSource', () => {
    it('should delete a data source', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined);

      await sourceService.deleteSource('1');

      expect(api.delete).toHaveBeenCalledWith('/sources/1');
    });
  });

  describe('testConnection', () => {
    it('should test connection with config', async () => {
      const config = {
        type: 'mysql' as const,
        host: 'localhost',
        port: 3306,
        database: 'test',
        username: 'admin',
        password: 'secret',
      };
      vi.mocked(api.post).mockResolvedValue(undefined);

      const result = await sourceService.testConnection(config);

      expect(api.post).toHaveBeenCalledWith('/sources/test-connection', config);
      expect(result).toBeUndefined();
    });
  });

  describe('triggerSync', () => {
    it('should trigger sync for a source', async () => {
      const mockResponse: SyncResponse = {
        source_id: '1',
        started_at: '2024-01-01T00:00:00Z',
        objects_count: 10,
      };
      vi.mocked(api.post).mockResolvedValue(mockResponse);

      const result = await sourceService.triggerSync('1');

      expect(api.post).toHaveBeenCalledWith('/sources/1/sync');
      expect(result).toEqual(mockResponse);
    });
  });

  describe('getSchemaTree', () => {
    it('should return schema tree for a source', async () => {
      const mockSchemaTree: SchemaTree = {
        source_id: '1',
        objects: [
          {
            id: 'obj1',
            name: 'users',
            type: 'table',
            column_count: 5,
            columns: [],
          },
        ],
      };
      vi.mocked(api.get).mockResolvedValue(mockSchemaTree);

      const result = await sourceService.getSchemaTree('1');

      expect(api.get).toHaveBeenCalledWith('/sources/1/schema');
      expect(result).toEqual(mockSchemaTree);
    });
  });

  describe('getSchemaChanges', () => {
    it('should return schema changes for a source', async () => {
      const mockChanges: SchemaChange[] = [
        {
          id: 'chg1',
          change_type: 'column_added',
          object_type: 'table',
          object_name: 'users',
          detected_at: '2024-01-01T00:00:00Z',
          acknowledged: false,
        },
      ];
      vi.mocked(api.get).mockResolvedValue(mockChanges);

      const result = await sourceService.getSchemaChanges('1');

      expect(api.get).toHaveBeenCalledWith('/sources/1/changes');
      expect(result).toEqual(mockChanges);
    });
  });
});
