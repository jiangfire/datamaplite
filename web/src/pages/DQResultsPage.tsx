import { useState } from 'react';
import { History, Filter, ChevronLeft } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import {
  Layout,
  Button,
  Card,
  CardContent,
  Input,
  DQResultCard,
} from '../components';
import { useDQResults, useDQRules } from '../hooks';

export const DQResultsPage: React.FC = () => {
  const navigate = useNavigate();
  const [ruleId, setRuleId] = useState('');
  const [batchId, setBatchId] = useState('');

  const { results, loading, error, refetch } = useDQResults({
    rule_id: ruleId || undefined,
    batch_id: batchId || undefined,
    limit: 50,
  });

  const { rules } = useDQRules();

  const getRuleName = (ruleId: string) => {
    return rules.find((r) => r.id === ruleId)?.name || ruleId;
  };

  const clearFilters = () => {
    setRuleId('');
    setBatchId('');
  };

  return (
    <Layout>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            onClick={() => navigate('/dq/rules')}
            className="p-2"
          >
            <ChevronLeft size={24} />
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">检查历史</h1>
            <p className="text-slate-500 mt-1">查看数据质量检查的历史记录</p>
          </div>
        </div>
      </div>

      {/* Filters */}
      <Card className="mb-6">
        <CardContent className="p-4">
          <div className="flex items-center gap-2 mb-4">
            <Filter size={18} className="text-slate-400" />
            <span className="text-sm font-medium text-slate-700">筛选</span>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm text-slate-600 mb-1">规则</label>
              <select
                value={ruleId}
                onChange={(e) => setRuleId(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                <option value="">全部规则</option>
                {rules.map((rule) => (
                  <option key={rule.id} value={rule.id}>
                    {rule.name}
                  </option>
                ))}
              </select>
            </div>
            <Input
              label="批次ID"
              value={batchId}
              onChange={(e) => setBatchId(e.target.value)}
              placeholder="输入批次ID"
            />
            <div className="flex items-end">
              <Button
                variant="secondary"
                onClick={clearFilters}
                className="w-full"
              >
                清除筛选
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

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
      ) : results.length === 0 ? (
        <Card>
          <CardContent className="py-16 text-center">
            <div className="w-20 h-20 rounded-2xl bg-slate-50 flex items-center justify-center mx-auto mb-6">
              <History size={40} className="text-slate-400" />
            </div>
            <h3 className="text-lg font-medium text-slate-900 mb-2">
              暂无检查记录
            </h3>
            <p className="text-slate-500 mb-6">
              在规则页面执行检查后将显示结果
            </p>
            <Button onClick={() => navigate('/dq/rules')}>返回规则页面</Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          <div className="text-sm text-slate-500 mb-4">
            共 {results.length} 条记录
          </div>
          {results.map((result) => (
            <DQResultCard
              key={result.id}
              result={result}
              ruleName={getRuleName(result.rule_id)}
            />
          ))}
        </div>
      )}
    </Layout>
  );
};
