import { useState } from 'react';
import { Plus, Bell, Edit2, Trash2, Webhook, Mail } from 'lucide-react';
import { Layout, Button, Card, CardContent } from '../components';
import { useAlerts } from '../hooks';
import type { AlertRuleCreate } from '../types';

const CHANGE_TYPE_OPTIONS = [
  { value: 'all', label: '全部变更' },
  { value: 'add_object', label: '新增对象' },
  { value: 'drop_object', label: '删除对象' },
  { value: 'add_column', label: '新增字段' },
  { value: 'drop_column', label: '删除字段' },
  { value: 'alter_column', label: '修改字段' },
  { value: 'change_type', label: '类型变更' },
];

export const AlertRulesPage: React.FC = () => {
  const { rules, loading, error, createRule, updateRule, deleteRule } = useAlerts();
  const [showForm, setShowForm] = useState(false);
  const [editingRule, setEditingRule] = useState<string | null>(null);
  const [formData, setFormData] = useState<AlertRuleCreate>({
    name: '',
    description: '',
    change_types: 'all',
    notify_webhook: false,
    webhook_url: '',
    notify_in_app: true,
    is_active: true,
  });

  const handleSubmit = async () => {
    if (!formData.name.trim()) return;

    const data: AlertRuleCreate = {
      ...formData,
      webhook_url: formData.notify_webhook ? formData.webhook_url : undefined,
    };

    if (editingRule) {
      await updateRule(editingRule, data);
      setEditingRule(null);
    } else {
      await createRule(data);
    }

    setShowForm(false);
    setFormData({
      name: '',
      description: '',
      change_types: 'all',
      notify_webhook: false,
      webhook_url: '',
      notify_in_app: true,
      is_active: true,
    });
  };

  const handleEdit = (rule: typeof rules[0]) => {
    setEditingRule(rule.id);
    setFormData({
      name: rule.name,
      description: rule.description || '',
      change_types: rule.change_types,
      notify_webhook: rule.notify_webhook,
      webhook_url: rule.webhook_url || '',
      notify_in_app: rule.notify_in_app,
      is_active: rule.is_active,
    });
    setShowForm(true);
  };

  const getChangeTypeLabel = (types: string) => {
    if (types === 'all') return '全部变更';
    return types.split(',').map(t => {
      const option = CHANGE_TYPE_OPTIONS.find(o => o.value === t.trim());
      return option?.label || t;
    }).join(', ');
  };

  return (
    <Layout>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">告警规则</h1>
          <p className="text-slate-500 mt-1">管理Schema变更告警规则</p>
        </div>
        <Button onClick={() => { setEditingRule(null); setShowForm(true); }}>
          <Plus size={18} className="mr-2" />
          创建规则
        </Button>
      </div>

      {showForm && (
        <Card className="mb-6">
          <CardContent className="p-6">
            <h3 className="font-medium mb-4">
              {editingRule ? '编辑规则' : '创建新规则'}
            </h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-slate-600 mb-1">规则名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border rounded-lg"
                  placeholder="例如：生产库表结构变更告警"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-600 mb-1">描述</label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-3 py-2 border rounded-lg"
                  placeholder="可选描述"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-600 mb-1">监控变更类型</label>
                <select
                  value={formData.change_types}
                  onChange={(e) => setFormData({ ...formData, change_types: e.target.value })}
                  className="w-full px-3 py-2 border rounded-lg"
                >
                  {CHANGE_TYPE_OPTIONS.map(opt => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              </div>
              <div className="flex gap-4">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.notify_in_app}
                    onChange={(e) => setFormData({ ...formData, notify_in_app: e.target.checked })}
                    className="rounded"
                  />
                  <span className="text-sm">站内通知</span>
                </label>
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.notify_webhook}
                    onChange={(e) => setFormData({ ...formData, notify_webhook: e.target.checked })}
                    className="rounded"
                  />
                  <span className="text-sm">Webhook</span>
                </label>
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.is_active}
                    onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                    className="rounded"
                  />
                  <span className="text-sm">启用</span>
                </label>
              </div>
              {formData.notify_webhook && (
                <div>
                  <label className="block text-sm text-slate-600 mb-1">Webhook URL</label>
                  <input
                    type="url"
                    value={formData.webhook_url}
                    onChange={(e) => setFormData({ ...formData, webhook_url: e.target.value })}
                    className="w-full px-3 py-2 border rounded-lg"
                    placeholder="https://example.com/webhook"
                  />
                </div>
              )}
              <div className="flex gap-2 pt-2">
                <Button variant="secondary" onClick={() => setShowForm(false)}>取消</Button>
                <Button onClick={handleSubmit}>{editingRule ? '保存' : '创建'}</Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {loading ? (
        <div className="text-center py-12">加载中...</div>
      ) : error ? (
        <Card><CardContent className="p-8 text-center text-red-500">{error}</CardContent></Card>
      ) : rules.length === 0 ? (
        <Card>
          <CardContent className="py-16 text-center">
            <div className="w-16 h-16 rounded-full bg-slate-100 flex items-center justify-center mx-auto mb-4">
              <Bell size={32} className="text-slate-400" />
            </div>
            <p className="text-slate-500">暂无告警规则，创建一个吧</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {rules.map((rule) => (
            <Card key={rule.id} className={!rule.is_active ? 'opacity-60' : ''}>
              <CardContent className="p-4">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-medium">{rule.name}</h3>
                      {!rule.is_active && (
                        <span className="px-2 py-0.5 text-xs bg-slate-200 text-slate-600 rounded">已禁用</span>
                      )}
                    </div>
                    {rule.description && (
                      <p className="text-sm text-slate-500 mb-2">{rule.description}</p>
                    )}
                    <div className="flex flex-wrap gap-2 text-xs">
                      <span className="px-2 py-1 bg-indigo-50 text-indigo-700 rounded">
                        {getChangeTypeLabel(rule.change_types)}
                      </span>
                      {rule.notify_in_app && (
                        <span className="flex items-center gap-1 px-2 py-1 bg-blue-50 text-blue-700 rounded">
                          <Mail size={12} /> 站内通知
                        </span>
                      )}
                      {rule.notify_webhook && (
                        <span className="flex items-center gap-1 px-2 py-1 bg-green-50 text-green-700 rounded">
                          <Webhook size={12} /> Webhook
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-1">
                    <button
                      onClick={() => handleEdit(rule)}
                      className="p-2 text-slate-400 hover:text-indigo-600 rounded-lg hover:bg-indigo-50"
                    >
                      <Edit2 size={16} />
                    </button>
                    <button
                      onClick={() => deleteRule(rule.id)}
                      className="p-2 text-slate-400 hover:text-red-600 rounded-lg hover:bg-red-50"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </Layout>
  );
};
