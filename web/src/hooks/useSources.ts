import { useState, useEffect, useCallback, useReducer } from 'react';
import { sourceService } from '../services';
import type { DataSource, DataSourceCreate, DataSourceUpdate } from '../types';

export const useSources = () => {
  const [sources, setSources] = useState<DataSource[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSources = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await sourceService.listSources();
      setSources(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch sources');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSources();
  }, [fetchSources]);

  const createSource = async (data: DataSourceCreate) => {
    const newSource = await sourceService.createSource(data);
    setSources((prev) => [...prev, newSource]);
    return newSource;
  };

  const updateSource = async (id: string, data: DataSourceUpdate) => {
    await sourceService.updateSource(id, data);
    await fetchSources();
  };

  const deleteSource = async (id: string) => {
    await sourceService.deleteSource(id);
    setSources((prev) => prev.filter((s) => s.id !== id));
  };

  return {
    sources,
    loading,
    error,
    refetch: fetchSources,
    createSource,
    updateSource,
    deleteSource,
  };
};

interface SourceState {
  source: DataSource | null;
  loading: boolean;
  error: string | null;
}

type SourceAction =
  | { type: 'start' }
  | { type: 'success'; source: DataSource }
  | { type: 'error'; error: string };

function sourceReducer(state: SourceState, action: SourceAction): SourceState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return { source: action.source, loading: false, error: null };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useSource = (id: string | undefined) => {
  const [state, dispatch] = useReducer(sourceReducer, {
    source: null,
    loading: false,
    error: null,
  });

  useEffect(() => {
    if (!id) return;

    let active = true;
    dispatch({ type: 'start' });
    sourceService
      .getSource(id)
      .then((data) => {
        if (active) dispatch({ type: 'success', source: data });
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error:
              err instanceof Error ? err.message : 'Failed to fetch source',
          });
      });

    return () => {
      active = false;
    };
  }, [id]);

  return state;
};
