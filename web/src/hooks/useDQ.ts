import { useState, useEffect, useCallback } from 'react';
import { dqService } from '../services';
import type {
  DQRule,
  DQRuleWithResult,
  DQRuleCreate,
  DQRuleFilter,
  DQResult,
  DQCheckRequest,
  DQCheckResponse,
  DQStats,
} from '../types';

// 规则列表Hook
export const useDQRules = (filter?: DQRuleFilter) => {
  const [rules, setRules] = useState<DQRuleWithResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRules = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await dqService.listRules(filter);
      setRules(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch rules');
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const createRule = async (data: DQRuleCreate) => {
    const newRule = await dqService.createRule(data);
    await fetchRules();
    return newRule;
  };

  const updateRule = async (id: string, data: DQRuleCreate) => {
    await dqService.updateRule(id, data);
    await fetchRules();
  };

  const deleteRule = async (id: string) => {
    await dqService.deleteRule(id);
    setRules((prev) => prev.filter((r) => r.id !== id));
  };

  const toggleRuleActive = async (rule: DQRuleWithResult) => {
    await updateRule(rule.id, { ...rule, is_active: !rule.is_active });
  };

  return {
    rules,
    loading,
    error,
    refetch: fetchRules,
    createRule,
    updateRule,
    deleteRule,
    toggleRuleActive,
  };
};

// 规则详情Hook
export const useDQRule = (id: string | null) => {
  const [rule, setRule] = useState<DQRule | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    const fetchRule = async () => {
      setLoading(true);
      try {
        const data = await dqService.getRule(id);
        setRule(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch rule');
      } finally {
        setLoading(false);
      }
    };
    fetchRule();
  }, [id]);

  return { rule, loading, error };
};

// 检查结果Hook
export const useDQResults = (params?: { rule_id?: string; batch_id?: string; limit?: number }) => {
  const [results, setResults] = useState<DQResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchResults = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await dqService.getResults(params);
      setResults(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch results');
    } finally {
      setLoading(false);
    }
  }, [params?.rule_id, params?.batch_id, params?.limit]);

  useEffect(() => {
    fetchResults();
  }, [fetchResults]);

  return { results, loading, error, refetch: fetchResults };
};

// 质量检查执行Hook
export const useDQCheck = () => {
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastResult, setLastResult] = useState<DQCheckResponse | null>(null);

  const checkRules = async (data: DQCheckRequest) => {
    setChecking(true);
    setError(null);
    try {
      const result = await dqService.checkRules(data);
      setLastResult(result);
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Check failed');
      throw err;
    } finally {
      setChecking(false);
    }
  };

  return { checkRules, checking, error, lastResult };
};

// 统计Hook
export const useDQStats = () => {
  const [stats, setStats] = useState<DQStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await dqService.getStats();
      setStats(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch stats');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  return { stats, loading, error, refetch: fetchStats };
};
