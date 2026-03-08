import { useState, useEffect, useCallback } from 'react';
import { sourceService } from '../services';
import type { SchemaTree, SchemaChange } from '../types';

export const useSchemaTree = (sourceId: string | undefined) => {
  const [schemaTree, setSchemaTree] = useState<SchemaTree | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSchemaTree = useCallback(async () => {
    if (!sourceId) return;

    setLoading(true);
    setError(null);
    try {
      const data = await sourceService.getSchemaTree(sourceId);
      setSchemaTree(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch schema');
    } finally {
      setLoading(false);
    }
  }, [sourceId]);

  useEffect(() => {
    fetchSchemaTree();
  }, [fetchSchemaTree]);

  return { schemaTree, loading, error, refetch: fetchSchemaTree };
};

export const useSchemaChanges = (sourceId: string | undefined) => {
  const [changes, setChanges] = useState<SchemaChange[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!sourceId) return;

    const fetchChanges = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await sourceService.getSchemaChanges(sourceId);
        setChanges(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch changes');
      } finally {
        setLoading(false);
      }
    };

    fetchChanges();
  }, [sourceId]);

  return { changes, loading, error };
};

export const useSyncSource = () => {
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sync = async (sourceId: string) => {
    setSyncing(true);
    setError(null);
    try {
      const result = await sourceService.triggerSync(sourceId);
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sync failed');
      throw err;
    } finally {
      setSyncing(false);
    }
  };

  return { sync, syncing, error };
};
