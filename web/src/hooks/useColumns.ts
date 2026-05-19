import { useCallback, useEffect, useReducer, useState } from 'react';
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

interface ColumnDetailState {
  column: ColumnDetail | null;
  loading: boolean;
  error: string | null;
}

type ColumnDetailAction =
  | { type: 'start' }
  | { type: 'success'; column: ColumnDetail }
  | { type: 'error'; error: string };

function columnDetailReducer(
  state: ColumnDetailState,
  action: ColumnDetailAction,
): ColumnDetailState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return { column: action.column, loading: false, error: null };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useColumnDetail = (columnId: string | undefined) => {
  const [state, dispatch] = useReducer(columnDetailReducer, {
    column: null,
    loading: false,
    error: null,
  });
  const [refreshTick, setRefreshTick] = useState(0);

  useEffect(() => {
    if (!columnId) return;

    let active = true;
    dispatch({ type: 'start' });
    columnService
      .getColumnDetail(columnId)
      .then((data) => {
        if (active) dispatch({ type: 'success', column: data });
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error:
              err instanceof Error ? err.message : 'Failed to fetch column',
          });
      });

    return () => {
      active = false;
    };
  }, [columnId, refreshTick]);

  const refetch = useCallback(async () => {
    setRefreshTick((t) => t + 1);
  }, []);

  return { ...state, refetch };
};

interface ColumnMappingsState {
  mappings: ColumnMapping[];
  loading: boolean;
  error: string | null;
}

type ColumnMappingsAction =
  | { type: 'start' }
  | { type: 'success'; mappings: ColumnMapping[] }
  | { type: 'error'; error: string };

function columnMappingsReducer(
  state: ColumnMappingsState,
  action: ColumnMappingsAction,
): ColumnMappingsState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return { mappings: action.mappings, loading: false, error: null };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useColumnMappings = (columnId: string | undefined) => {
  const [state, dispatch] = useReducer(columnMappingsReducer, {
    mappings: [],
    loading: false,
    error: null,
  });
  const [refreshTick, setRefreshTick] = useState(0);

  useEffect(() => {
    if (!columnId) return;

    let active = true;
    dispatch({ type: 'start' });
    columnService
      .getColumnMappings(columnId)
      .then((data) => {
        if (active) dispatch({ type: 'success', mappings: data });
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error:
              err instanceof Error ? err.message : 'Failed to fetch mappings',
          });
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
    ...state,
    refetch,
    createMapping,
    deleteMapping,
  };
};

interface LineageState {
  lineage: LineageResponse | null;
  loading: boolean;
  error: string | null;
}

type LineageAction =
  | { type: 'start' }
  | { type: 'success'; lineage: LineageResponse }
  | { type: 'error'; error: string };

function lineageReducer(
  state: LineageState,
  action: LineageAction,
): LineageState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return { lineage: action.lineage, loading: false, error: null };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useLineage = (columnId: string | undefined) => {
  const [state, dispatch] = useReducer(lineageReducer, {
    lineage: null,
    loading: false,
    error: null,
  });

  useEffect(() => {
    if (!columnId) return;

    let active = true;
    dispatch({ type: 'start' });
    columnService
      .getLineage(columnId)
      .then((data) => {
        if (active) dispatch({ type: 'success', lineage: data });
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error:
              err instanceof Error ? err.message : 'Failed to fetch lineage',
          });
      });

    return () => {
      active = false;
    };
  }, [columnId]);

  return state;
};

interface ImpactState {
  impact: ImpactAnalysisResponse | null;
  loading: boolean;
  error: string | null;
}

type ImpactAction =
  | { type: 'start' }
  | { type: 'success'; impact: ImpactAnalysisResponse }
  | { type: 'error'; error: string };

function impactReducer(
  state: ImpactState,
  action: ImpactAction,
): ImpactState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return { impact: action.impact, loading: false, error: null };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useImpactAnalysis = (columnId: string | undefined) => {
  const [state, dispatch] = useReducer(impactReducer, {
    impact: null,
    loading: false,
    error: null,
  });

  useEffect(() => {
    if (!columnId) return;

    let active = true;
    dispatch({ type: 'start' });
    columnService
      .getImpactAnalysis(columnId)
      .then((data) => {
        if (active) dispatch({ type: 'success', impact: data });
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error:
              err instanceof Error
                ? err.message
                : 'Failed to fetch impact analysis',
          });
      });

    return () => {
      active = false;
    };
  }, [columnId]);

  return state;
};
