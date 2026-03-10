import { useState, useEffect, useCallback } from 'react';
import { alertService } from '../services';
import type { AlertRule, AlertRuleCreate } from '../types';

export const useAlerts = () => {
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRules = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await alertService.listRules();
      setRules(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch alert rules');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const createRule = async (data: AlertRuleCreate) => {
    try {
      const newRule = await alertService.createRule(data);
      setRules((prev) => [...prev, newRule]);
      return newRule;
    } catch (err) {
      throw err;
    }
  };

  const updateRule = async (id: string, data: AlertRuleCreate) => {
    try {
      await alertService.updateRule(id, data);
      await fetchRules();
    } catch (err) {
      throw err;
    }
  };

  const deleteRule = async (id: string) => {
    try {
      await alertService.deleteRule(id);
      setRules((prev) => prev.filter((r) => r.id !== id));
    } catch (err) {
      throw err;
    }
  };

  return {
    rules,
    loading,
    error,
    refetch: fetchRules,
    createRule,
    updateRule,
    deleteRule,
  };
};
