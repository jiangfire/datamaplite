import { useState, useEffect, useCallback } from 'react';
import { alertService } from '../services';
import { useToastContext } from './useToastContext';
import type { AlertRule, AlertRuleCreate } from '../types';

export const useAlerts = () => {
  const { toast } = useToastContext();
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
      setError(
        err instanceof Error ? err.message : 'Failed to fetch alert rules',
      );
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
      toast('告警规则创建成功', 'success');
      return newRule;
    } catch (err) {
      toast(err instanceof Error ? err.message : '创建告警规则失败', 'error');
      throw err;
    }
  };

  const updateRule = async (id: string, data: AlertRuleCreate) => {
    try {
      await alertService.updateRule(id, data);
      await fetchRules();
      toast('告警规则更新成功', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : '更新告警规则失败', 'error');
      throw err;
    }
  };

  const deleteRule = async (id: string) => {
    try {
      await alertService.deleteRule(id);
      setRules((prev) => prev.filter((r) => r.id !== id));
      toast('告警规则已删除', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : '删除告警规则失败', 'error');
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
