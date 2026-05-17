import { CheckCircle, XCircle, AlertCircle } from 'lucide-react';
import { Card, CardContent, Badge } from '../ui';
import type { DQResult, DQResultStatus } from '../../types';

interface DQResultCardProps {
  result: DQResult;
  ruleName?: string;
}

const statusConfig: Record<
  DQResultStatus,
  { icon: React.ReactNode; label: string; color: string; bgColor: string }
> = {
  passed: {
    icon: <CheckCircle size={20} />,
    label: '通过',
    color: 'text-emerald-600',
    bgColor: 'bg-emerald-50 border-emerald-100',
  },
  failed: {
    icon: <XCircle size={20} />,
    label: '失败',
    color: 'text-red-600',
    bgColor: 'bg-red-50 border-red-100',
  },
  error: {
    icon: <AlertCircle size={20} />,
    label: '错误',
    color: 'text-amber-600',
    bgColor: 'bg-amber-50 border-amber-100',
  },
};

export const DQResultCard: React.FC<DQResultCardProps> = ({
  result,
  ruleName,
}) => {
  const config = statusConfig[result.status];
  const passRatePercent = Math.round(result.pass_rate);

  return (
    <Card className="overflow-hidden">
      <CardContent className="p-0">
        {/* Header */}
        <div
          className={`flex items-center gap-3 p-4 border-b ${config.bgColor}`}
        >
          <div className={config.color}>{config.icon}</div>
          <div className="flex-1 min-w-0">
            {ruleName && (
              <h4 className="font-semibold text-slate-900 truncate">
                {ruleName}
              </h4>
            )}
            <div className="flex items-center gap-2 text-sm">
              <span className={config.color}>{config.label}</span>
              <span className="text-slate-400">•</span>
              <span className="text-slate-500">
                {new Date(result.checked_at).toLocaleString()}
              </span>
            </div>
          </div>
          <Badge
            variant={
              result.status === 'passed'
                ? 'success'
                : result.status === 'failed'
                  ? 'error'
                  : 'warning'
            }
          >
            {passRatePercent}% 通过
          </Badge>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 divide-x border-b">
          <div className="p-4 text-center">
            <div className="text-2xl font-bold text-slate-900">
              {result.total_rows.toLocaleString()}
            </div>
            <div className="text-xs text-slate-500 mt-1">总行数</div>
          </div>
          <div className="p-4 text-center">
            <div
              className={`text-2xl font-bold ${
                result.failed_rows > 0 ? 'text-red-600' : 'text-emerald-600'
              }`}
            >
              {result.failed_rows.toLocaleString()}
            </div>
            <div className="text-xs text-slate-500 mt-1">失败行数</div>
          </div>
          <div className="p-4 text-center">
            <div className="text-2xl font-bold text-slate-900">
              {(result.total_rows - result.failed_rows).toLocaleString()}
            </div>
            <div className="text-xs text-slate-500 mt-1">通过行数</div>
          </div>
        </div>

        {/* Error Message */}
        {result.error_message && (
          <div className="p-4 bg-red-50 border-b">
            <div className="text-sm text-red-600">
              <span className="font-medium">错误信息：</span>
              {result.error_message}
            </div>
          </div>
        )}

        {/* Sample Errors */}
        {result.sample_errors && result.sample_errors.length > 0 && (
          <div className="p-4">
            <div className="text-sm font-medium text-slate-700 mb-2">
              错误样本
            </div>
            <div className="space-y-2 max-h-40 overflow-y-auto">
              {result.sample_errors.map((error, index) => (
                <div
                  key={index}
                  className="text-xs font-mono bg-slate-50 p-2 rounded border"
                >
                  <pre className="whitespace-pre-wrap break-all">
                    {JSON.stringify(error, null, 2)}
                  </pre>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Batch ID */}
        <div className="px-4 py-2 bg-slate-50 text-xs text-slate-400 border-t">
          批次ID: {result.check_batch_id}
        </div>
      </CardContent>
    </Card>
  );
};
