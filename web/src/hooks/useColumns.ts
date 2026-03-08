import { useState, useEffect, useCallback } from 'react';
import { columnService } from '../services';
import type {
  ColumnDetail,
  ColumnSearchResult,
  ColumnMapping,
  LineageResponse,
  ImpactAnalysisResponse,
} from '../types';

export const useColumnSearch = () => {
  const [results, setResults] = useState<ColumnSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const search = useCallback(async (query: string, limit?: number) => {
    if (!query.trim()) {
      setResults([]);
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const data = await columnService.searchColumns(query, limit);
      setResults(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  }, []);

  return { results, loading, error, search };
};

export const useColumnDetail = (columnId: string | undefined) => {
  const [column, setColumn] = useState<ColumnDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!columnId) return;

    const fetchColumn = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await columnService.getColumnDetail(columnId);
        setColumn(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch column');
      } finally {
        setLoading(false);
      }
    };

    fetchColumn();
  }, [columnId]);

  return { column, loading, error };
};

export const useColumnMappings = (columnId: string | undefined) => {
  const [mappings, setMappings] = useState<ColumnMapping[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchMappings = useCallback(async () => {
    if (!columnId) return;

    setLoading(true);
    setError(null);
    try {
      const data = await columnService.getColumnMappings(columnId);
      setMappings(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch mappings');
    } finally {
      setLoading(false);
    }
  }, [columnId]);

  useEffect(() => {
    fetchMappings();
  }, [fetchMappings]);

  const createMapping = async (data: {
    source_column_id: string;
    target_column_id: string;
    mapping_type: 'alias' | 'transform' | 'derived' | 'synonym';
    confidence?: number;
  }) => {
    if (!columnId) throw new Error('Column ID is required');

    try {
      const newMapping = await columnService.createColumnMapping(columnId, data);
      setMappings((prev) => [...prev, newMapping]);
      return newMapping;
    } catch (err) {
      throw err;
    }
  };

  const deleteMapping = async (mappingId: string) => {
    if (!columnId) throw new Error('Column ID is required');

    try {
      await columnService.deleteColumnMapping(columnId, mappingId);
      setMappings((prev) => prev.filter((m) => m.id !== mappingId));
    } catch (err) {
      throw err;
    }
  };

  return { mappings, loading, error, refetch: fetchMappings, createMapping, deleteMapping };
};

export const useLineage = (columnId: string | undefined) => {
  const [lineage, setLineage] = useState<LineageResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!columnId) return;

    const fetchLineage = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await columnService.getLineage(columnId);
        setLineage(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch lineage');
      } finally {
        setLoading(false);
      }
    };

    fetchLineage();
  }, [columnId]);

  return { lineage, loading, error };
};

export const useImpactAnalysis = (columnId: string | undefined) => {
  const [impact, setImpact] = useState<ImpactAnalysisResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!columnId) return;

    const fetchImpact = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await columnService.getImpactAnalysis(columnId);
        setImpact(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch impact analysis');
      } finally {
        setLoading(false);
      }
    };

    fetchImpact();
  }, [columnId]);

  return { impact, loading, error };
};
