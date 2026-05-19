import { useCallback, useEffect, useReducer, useState } from 'react';
import { sourceService } from '../services';
import type { SchemaTree, SchemaChange } from '../types';

interface SchemaTreeState {
  schemaTree: SchemaTree | null;
  loading: boolean;
  error: string | null;
}

type SchemaTreeAction =
  | { type: 'start' }
  | { type: 'success'; schemaTree: SchemaTree }
  | { type: 'error'; error: string };

function schemaTreeReducer(
  state: SchemaTreeState,
  action: SchemaTreeAction,
): SchemaTreeState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return { schemaTree: action.schemaTree, loading: false, error: null };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useSchemaTree = (sourceId: string | undefined) => {
  const [state, dispatch] = useReducer(schemaTreeReducer, {
    schemaTree: null,
    loading: false,
    error: null,
  });
  const [refreshTick, setRefreshTick] = useState(0);

  useEffect(() => {
    if (!sourceId) return;

    let active = true;
    dispatch({ type: 'start' });
    sourceService
      .getSchemaTree(sourceId)
      .then((data) => {
        if (active) dispatch({ type: 'success', schemaTree: data });
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error:
              err instanceof Error ? err.message : 'Failed to fetch schema',
          });
      });

    return () => {
      active = false;
    };
  }, [sourceId, refreshTick]);

  const refetch = useCallback(async () => {
    setRefreshTick((t) => t + 1);
  }, []);

  return { ...state, refetch };
};

interface SchemaChangesState {
  changes: SchemaChange[];
  loading: boolean;
  error: string | null;
}

type SchemaChangesAction =
  | { type: 'start' }
  | { type: 'success'; changes: SchemaChange[] }
  | { type: 'error'; error: string };

function schemaChangesReducer(
  state: SchemaChangesState,
  action: SchemaChangesAction,
): SchemaChangesState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return { changes: action.changes, loading: false, error: null };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useSchemaChanges = (sourceId: string | undefined) => {
  const [state, dispatch] = useReducer(schemaChangesReducer, {
    changes: [],
    loading: false,
    error: null,
  });

  useEffect(() => {
    if (!sourceId) return;

    let active = true;
    dispatch({ type: 'start' });
    sourceService
      .getSchemaChanges(sourceId)
      .then((data) => {
        if (active) dispatch({ type: 'success', changes: data });
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error:
              err instanceof Error ? err.message : 'Failed to fetch changes',
          });
      });

    return () => {
      active = false;
    };
  }, [sourceId]);

  return state;
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
