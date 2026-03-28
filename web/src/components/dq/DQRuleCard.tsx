import { useState } from 'react';
import {
  Shield,
  MoreVertical,
  Edit,
  Trash2,
  Check,
  Play,
  AlertCircle,
  CheckCircle,
  XCircle,
  Power,
} from 'lucide-react';
import { Card, CardContent, Badge } from '../ui';
import type { DQRuleWithResult, DQRuleType, DQSeverity } from '../../types';

interface DQRuleCardProps {
  rule: DQRuleWithResult;
  onEdit: (rule: DQRuleWithResult) => void;
  onDelete: (id: string) => void;
  onToggleActive: (rule: DQRuleWithResult) => void;
  onCheck: (rule: DQRuleWithResult) => void;
}

const ruleTypeLabels: Record<DQRuleType, string> = {
  not_null: '非空检查',
  unique: '唯一性',
  regex: '正则匹配',
  range: '范围检查',
  enum: '枚举值',
  custom_sql: '自定义SQL',
  referential: '引用完整性',
};

const severityColors: Record<DQSeverity, 'error' | 'warning' | 'info'> = {
  error: 'error',
  warning: 'warning',
  info: 'info',
};

const severityLabels: Record<DQSeverity, string> = {
  error: '错误',
  warning: '警告',
  info: '信息',
};

export const DQRuleCard: React.FC<DQRuleCardProps> = ({
  rule,
  onEdit,
  onDelete,
  onToggleActive,
  onCheck,
}) => {
  const [showActions, setShowActions] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const handleDelete = () => {
    if (confirmDelete) {
      onDelete(rule.id);
      setConfirmDelete(false);
    } else {
      setConfirmDelete(true);
      setTimeout(() => setConfirmDelete(false), 3000);
    }
  };

  const hasResult = !!rule.latest_result;
  const isPassed = rule.latest_result?.status === 'passed';
  const isFailed = rule.latest_result?.status === 'failed';

  return (
    <Card className={`relative group ${!rule.is_active ? 'opacity-60' : ''}`}>
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-4">
            <div
              className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                isFailed
                  ? 'bg-red-50 border border-red-100'
                  : isPassed
                    ? 'bg-emerald-50 border border-emerald-100'
                    : 'bg-slate-50 border border-slate-100'
              }`}
            >
              {isFailed ? (
                <XCircle size={24} className="text-red-500" />
              ) : isPassed ? (
                <CheckCircle size={24} className="text-emerald-500" />
              ) : (
                <Shield size={24} className="text-slate-400" />
              )}
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-lg font-semibold text-slate-900 truncate">
                {rule.name}
              </h3>
              <div className="flex flex-wrap gap-2 mt-1">
                <Badge variant="default">
                  {ruleTypeLabels[rule.rule_type]}
                </Badge>
                <Badge variant={severityColors[rule.severity]}>
                  {severityLabels[rule.severity]}
                </Badge>
                {!rule.is_active && <Badge variant="neutral">已禁用</Badge>}
              </div>
              {rule.description && (
                <p className="text-sm text-slate-600 mt-2 line-clamp-2">
                  {rule.description}
                </p>
              )}
            </div>
          </div>

          <div className="relative">
            <button
              onClick={() => setShowActions(!showActions)}
              className="p-2 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
            >
              <MoreVertical size={18} />
            </button>

            {showActions && (
              <div className="absolute right-0 top-full mt-1 w-40 bg-white rounded-lg shadow-lg border border-slate-200 py-1 z-10">
                <button
                  onClick={() => {
                    onEdit(rule);
                    setShowActions(false);
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
                >
                  <Edit size={16} />
                  编辑
                </button>
                <button
                  onClick={() => {
                    onToggleActive(rule);
                    setShowActions(false);
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
                >
                  <Power size={16} />
                  {rule.is_active ? '禁用' : '启用'}
                </button>
                <button
                  onClick={() => {
                    onCheck(rule);
                    setShowActions(false);
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
                >
                  <Play size={16} />
                  执行检查
                </button>
                <button
                  onClick={handleDelete}
                  className={`w-full px-4 py-2 text-left text-sm flex items-center gap-2 ${
                    confirmDelete
                      ? 'bg-red-50 text-red-600'
                      : 'text-red-600 hover:bg-red-50'
                  }`}
                >
                  {confirmDelete ? <Check size={16} /> : <Trash2 size={16} />}
                  {confirmDelete ? '确认删除?' : '删除'}
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Latest Result */}
        {hasResult && (
          <div className="mt-4 pt-4 border-t border-slate-100">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {isPassed ? (
                  <>
                    <CheckCircle size={16} className="text-emerald-500" />
                    <span className="text-sm text-emerald-600">通过</span>
                  </>
                ) : isFailed ? (
                  <>
                    <AlertCircle size={16} className="text-red-500" />
                    <span className="text-sm text-red-600">
                      失败 ({rule.latest_result?.failed_rows} /{' '}
                      {rule.latest_result?.total_rows})
                    </span>
                  </>
                ) : (
                  <>
                    <XCircle size={16} className="text-slate-400" />
                    <span className="text-sm text-slate-500">错误</span>
                  </>
                )}
              </div>
              <span className="text-xs text-slate-400">
                {rule.latest_result?.pass_rate.toFixed(1)}% 通过率
              </span>
            </div>
            {rule.latest_result?.checked_at && (
              <div className="text-xs text-slate-400 mt-1">
                检查于{' '}
                {new Date(rule.latest_result.checked_at).toLocaleString()}
              </div>
            )}
          </div>
        )}

        {!hasResult && (
          <div className="mt-4 pt-4 border-t border-slate-100 text-xs text-slate-400">
            尚未执行检查
          </div>
        )}
      </CardContent>
    </Card>
  );
};
