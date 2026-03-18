import { useState } from 'react';
import { Plus, Shield, Play, CheckCircle } from 'lucide-react';
import {
  Layout,
  Button,
  Card,
  CardContent,
  DQRuleCard,
  DQRuleForm,
  DQStatsCard,
} from '../components';
import { useDQRules, useDQStats, useDQCheck } from '../hooks';
import type { DQRuleWithResult, DQRuleCreate } from '../types';

export const DQRulesPage: React.FC = () => {
  const { rules, loading, error, refetch, createRule, updateRule, deleteRule, toggleRuleActive } =
    useDQRules();
  const { stats, refetch: refetchStats } = useDQStats();
  const { checkRules, checking } = useDQCheck();

  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingRule, setEditingRule] = useState<DQRuleWithResult | null>(null);
  const [checkResult, setCheckResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  const handleCreate = async (data: DQRuleCreate) => {
    await createRule(data);
    setShowCreateForm(false);
    refetchStats();
  };

  const handleEdit = async (data: DQRuleCreate) => {
    if (!editingRule) return;
    await updateRule(editingRule.id, data);
    setEditingRule(null);
  };

  const handleCheck = async (rule: DQRuleWithResult) => {
    try {
      const result = await checkRules({ rule_ids: [rule.id] });
      setCheckResult({
        success: true,
        message: `检查完成: ${result.passed_rules} 通过, ${result.failed_rules} 失败`,
      });
      refetch();
      refetchStats();
      setTimeout(() => setCheckResult(null), 5000);
    } catch (err) {
      setCheckResult({
        success: false,
        message: err instanceof Error ? err.message : '检查失败',
      });
      setTimeout(() => setCheckResult(null), 5000);
    }
  };

  const handleCheckAll = async () => {
    const activeRules = rules.filter((r) => r.is_active);
    if (activeRules.length === 0) {
      setCheckResult({
        success: false,
        message: '没有启用的规则可检查',
      });
      setTimeout(() => setCheckResult(null), 5000);
      return;
    }

    try {
      const result = await checkRules({ check_all: true });
      setCheckResult({
        success: true,
        message: `批量检查完成: ${result.passed_rules} 通过, ${result.failed_rules} 失败`,
      });
      refetch();
      refetchStats();
      setTimeout(() => setCheckResult(null), 5000);
    } catch (err) {
      setCheckResult({
        success: false,
        message: err instanceof Error ? err.message : '检查失败',
      });
      setTimeout(() => setCheckResult(null), 5000);
    }
  };

  return (
    <Layout>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">数据质量</h1>
          <p className="text-slate-500 mt-1">
            定义数据质量规则，监控数据健康状况
          </p>
        </div>
        <div className="flex gap-3">
          <Button
            variant="secondary"
            onClick={handleCheckAll}
            disabled={checking || rules.length === 0}
          >
            <Play size={18} className="mr-2" />
            {checking ? '检查中...' : '检查全部'}
          </Button>
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus size={18} className="mr-2" />
            创建规则
          </Button>
        </div>
      </div>

      {/* Check Result Toast */}
      {checkResult && (
        <div
          className={`mb-6 p-4 rounded-lg flex items-center gap-2 ${
            checkResult.success
              ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
              : 'bg-red-50 text-red-700 border border-red-200'
          }`}
        >
          {checkResult.success ? (
            <CheckCircle size={20} />
          ) : (
            <div className="w-5 h-5 rounded-full bg-red-500 text-white flex items-center justify-center text-xs">
              !
            </div>
          )}
          {checkResult.message}
        </div>
      )}

      {/* Stats */}
      {stats && <div className="mb-8"><DQStatsCard stats={stats} /></div>}

      {/* Content */}
      {loading ? (
        <div className="py-12 text-center">
          <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto" />
          <p className="text-slate-500 mt-4">加载中...</p>
        </div>
      ) : error ? (
        <Card>
          <CardContent className="py-12 text-center text-red-500">
            <p>加载失败: {error}</p>
            <Button variant="secondary" onClick={refetch} className="mt-4">
              重试
            </Button>
          </CardContent>
        </Card>
      ) : rules.length === 0 ? (
        <Card>
          <CardContent className="py-16 text-center">
            <div className="w-20 h-20 rounded-2xl bg-indigo-50 flex items-center justify-center mx-auto mb-6">
              <Shield size={40} className="text-indigo-400" />
            </div>
            <h3 className="text-lg font-medium text-slate-900 mb-2">
              暂无数据质量规则
            </h3>
            <p className="text-slate-500 mb-6">
              创建规则来监控和保证数据质量
            </p>
            <Button onClick={() => setShowCreateForm(true)}>
              <Plus size={18} className="mr-2" />
              创建规则
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {rules.map((rule) => (
            <DQRuleCard
              key={rule.id}
              rule={rule}
              onEdit={setEditingRule}
              onDelete={deleteRule}
              onToggleActive={toggleRuleActive}
              onCheck={handleCheck}
            />
          ))}
        </div>
      )}

      {/* Create Form Modal */}
      <DQRuleForm
        isOpen={showCreateForm}
        onClose={() => setShowCreateForm(false)}
        onSubmit={handleCreate}
        mode="create"
      />

      {/* Edit Form Modal */}
      <DQRuleForm
        isOpen={!!editingRule}
        onClose={() => setEditingRule(null)}
        onSubmit={handleEdit}
        initialData={editingRule || undefined}
        mode="edit"
      />
    </Layout>
  );
};
