import { useState, useEffect, useCallback, useReducer } from 'react';
import { termService } from '../services';
import { useToastContext } from './useToastContext';
import type { BusinessTerm, BusinessTermCreate } from '../types';

export const useTerms = (category?: string) => {
  const { toast } = useToastContext();
  const [terms, setTerms] = useState<BusinessTerm[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTerms = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await termService.listTerms(category);
      setTerms(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch terms');
    } finally {
      setLoading(false);
    }
  }, [category]);

  useEffect(() => {
    fetchTerms();
  }, [fetchTerms]);

  const createTerm = async (data: BusinessTermCreate) => {
    try {
      const newTerm = await termService.createTerm(data);
      setTerms((prev) => [...prev, newTerm]);
      toast('业务术语创建成功', 'success');
      return newTerm;
    } catch (err) {
      toast(err instanceof Error ? err.message : '创建业务术语失败', 'error');
      throw err;
    }
  };

  const updateTerm = async (id: string, data: BusinessTermCreate) => {
    try {
      await termService.updateTerm(id, data);
      await fetchTerms();
      toast('业务术语更新成功', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : '更新业务术语失败', 'error');
      throw err;
    }
  };

  const deleteTerm = async (id: string) => {
    try {
      await termService.deleteTerm(id);
      setTerms((prev) => prev.filter((t) => t.id !== id));
      toast('业务术语已删除', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : '删除业务术语失败', 'error');
      throw err;
    }
  };

  return {
    terms,
    loading,
    error,
    refetch: fetchTerms,
    createTerm,
    updateTerm,
    deleteTerm,
  };
};

interface TermState {
  term: BusinessTerm | null;
  loading: boolean;
  error: string | null;
}

type TermAction =
  | { type: 'start' }
  | { type: 'success'; term: BusinessTerm }
  | { type: 'error'; error: string };

function termReducer(state: TermState, action: TermAction): TermState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return { term: action.term, loading: false, error: null };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useTerm = (id: string | undefined) => {
  const [state, dispatch] = useReducer(termReducer, {
    term: null,
    loading: false,
    error: null,
  });

  useEffect(() => {
    if (!id) return;

    let active = true;
    dispatch({ type: 'start' });
    termService
      .getTerm(id)
      .then((data) => {
        if (active) dispatch({ type: 'success', term: data });
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error:
              err instanceof Error ? err.message : 'Failed to fetch term',
          });
      });

    return () => {
      active = false;
    };
  }, [id]);

  return state;
};

export const useDDLGeneration = () => {
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const generateDDL = async (
    objectId: string,
    targetType: 'mysql' | 'postgres',
  ) => {
    setGenerating(true);
    setError(null);
    try {
      const result = await termService.generateDDL({
        object_id: objectId,
        target_type: targetType,
      });
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate DDL');
      throw err;
    } finally {
      setGenerating(false);
    }
  };

  return { generateDDL, generating, error };
};
