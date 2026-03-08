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
    try {
      const newTerm = await termService.createTerm(data);
      setTerms((prev) => [...prev, newTerm]);
      return newTerm;
    } catch (err) {
      throw err;
    }
  };

  const updateTerm = async (id: string, data: BusinessTermCreate) => {
    try {
      await termService.updateTerm(id, data);
      await fetchTerms();
    } catch (err) {
      throw err;
    }
  };

  const deleteTerm = async (id: string) => {
    try {
      await termService.deleteTerm(id);
      setTerms((prev) => prev.filter((t) => t.id !== id));
    } catch (err) {
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

export const useDDLGeneration = () => {
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const generateDDL = async (objectId: string, targetType: 'mysql' | 'postgres') => {
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
