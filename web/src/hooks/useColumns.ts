import { useCallback, useEffect, useState } from 'react';
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
  const [refreshTick, setRefreshTick] = useState(0);

  useEffect(() => {
    if (!columnId) {
      setColumn(null);
      return;
    }

    let active = true;
    setLoading(true);
    setError(null);
    columnService
      .getColumnDetail(columnId)
      .then((data) => {
        if (active) setColumn(data);
      })
      .catch((err) => {
        if (active)
          setError(
            err instanceof Error ? err.message : 'Failed to fetch column',
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [columnId, refreshTick]);

  const refetch = useCallback(async () => {
    setRefreshTick((t) => t + 1);
  }, []);

  return { column, loading, error, refetch };
};

export const useColumnMappings = (columnId: string | undefined) => {
  const [mappings, setMappings] = useState<ColumnMapping[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshTick, setRefreshTick] = useState(0);

  useEffect(() => {
    if (!columnId) {
      setMappings([]);
      return;
    }

    let active = true;
    setLoading(true);
    setError(null);
    columnService
      .getColumnMappings(columnId)
      .then((data) => {
        if (active) setMappings(data);
      })
      .catch((err) => {
        if (active)
          setError(
            err instanceof Error ? err.message : 'Failed to fetch mappings',
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [columnId, refreshTick]);

  const refetch = useCallback(async () => {
    setRefreshTick((t) => t + 1);
  }, []);

  const createMapping = async (data: {
    source_column_id: string;
    target_column_id: string;
    mapping_type: 'alias' | 'transform' | 'derived' | 'synonym';
    confidence?: number;
  }) => {
    if (!columnId) throw new Error('Column ID is required');
    await columnService.createColumnMapping(columnId, data);
    refetch();
  };

  const deleteMapping = async (mappingId: string) => {
    if (!columnId) throw new Error('Column ID is required');
    await columnService.deleteColumnMapping(columnId, mappingId);
    refetch();
  };

  return {
    mappings,
    loading,
    error,
    refetch,
    createMapping,
    deleteMapping,
  };
};

export const useLineage = (columnId: string | undefined) => {
  const [lineage, setLineage] = useState<LineageResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!columnId) {
      setLineage(null);
      return;
    }

    let active = true;
    setLoading(true);
    setError(null);
    columnService
      .getLineage(columnId)
      .then((data) => {
        if (active) setLineage(data);
      })
      .catch((err) => {
        if (active)
          setError(
            err instanceof Error ? err.message : 'Failed to fetch lineage',
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [columnId]);

  return { lineage, loading, error };
};

export const useImpactAnalysis = (columnId: string | undefined) => {
  const [impact, setImpact] = useState<ImpactAnalysisResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!columnId) {
      setImpact(null);
      return;
    }

    let active = true;
    setLoading(true);
    setError(null);
    columnService
      .getImpactAnalysis(columnId)
      .then((data) => {
        if (active) setImpact(data);
      })
      .catch((err) => {
        if (active)
          setError(
            err instanceof Error
              ? err.message
              : 'Failed to fetch impact analysis',
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [columnId]);

  return { impact, loading, error };
};
