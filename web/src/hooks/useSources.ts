import { useState, useEffect, useCallback } from 'react';
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
    try {
      const newSource = await sourceService.createSource(data);
      setSources((prev) => [...prev, newSource]);
      return newSource;
    } catch (err) {
      throw err;
    }
  };

  const updateSource = async (id: string, data: DataSourceUpdate) => {
    try {
      await sourceService.updateSource(id, data);
      await fetchSources();
    } catch (err) {
      throw err;
    }
  };

  const deleteSource = async (id: string) => {
    try {
      await sourceService.deleteSource(id);
      setSources((prev) => prev.filter((s) => s.id !== id));
    } catch (err) {
      throw err;
    }
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

export const useSource = (id: string | undefined) => {
  const [source, setSource] = useState<DataSource | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;

    const fetchSource = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await sourceService.getSource(id);
        setSource(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch source');
      } finally {
        setLoading(false);
      }
    };

    fetchSource();
  }, [id]);

  return { source, loading, error };
};
