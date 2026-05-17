import { useCallback, useEffect, useState } from 'react';
import { sourceService } from '../services';
import type { SchemaTree, SchemaChange } from '../types';

export const useSchemaTree = (sourceId: string | undefined) => {
  const [schemaTree, setSchemaTree] = useState<SchemaTree | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshTick, setRefreshTick] = useState(0);

  useEffect(() => {
    if (!sourceId) {
      setSchemaTree(null);
      return;
    }

    let active = true;
    setLoading(true);
    setError(null);
    sourceService
      .getSchemaTree(sourceId)
      .then((data) => {
        if (active) setSchemaTree(data);
      })
      .catch((err) => {
        if (active)
          setError(
            err instanceof Error ? err.message : 'Failed to fetch schema',
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [sourceId, refreshTick]);

  const refetch = useCallback(async () => {
    setRefreshTick((t) => t + 1);
  }, []);

  return { schemaTree, loading, error, refetch };
};

export const useSchemaChanges = (sourceId: string | undefined) => {
  const [changes, setChanges] = useState<SchemaChange[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!sourceId) {
      setChanges([]);
      return;
    }

    let active = true;
    setLoading(true);
    setError(null);
    sourceService
      .getSchemaChanges(sourceId)
      .then((data) => {
        if (active) setChanges(data);
      })
      .catch((err) => {
        if (active)
          setError(
            err instanceof Error ? err.message : 'Failed to fetch changes',
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
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
