import { useState, useEffect, useCallback } from 'react';
import { termService } from '../services';
import type { BusinessTerm, BusinessTermCreate } from '../types';

export const useTerms = (category?: string) => {
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
    const newTerm = await termService.createTerm(data);
    setTerms((prev) => [...prev, newTerm]);
    return newTerm;
  };

  const updateTerm = async (id: string, data: BusinessTermCreate) => {
    await termService.updateTerm(id, data);
    await fetchTerms();
  };

  const deleteTerm = async (id: string) => {
    await termService.deleteTerm(id);
    setTerms((prev) => prev.filter((t) => t.id !== id));
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

export const useTerm = (id: string | undefined) => {
  const [term, setTerm] = useState<BusinessTerm | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;

    const fetchTerm = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await termService.getTerm(id);
        setTerm(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch term');
      } finally {
        setLoading(false);
      }
    };

    fetchTerm();
  }, [id]);

  return { term, loading, error };
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
