import { Shield, CheckCircle, XCircle, Activity } from 'lucide-react';
import { Card, CardContent } from '../ui';
import type { DQStats } from '../../types';

interface DQStatsCardProps {
  stats: DQStats;
}

export const DQStatsCard: React.FC<DQStatsCardProps> = ({ stats }) => {
  const passRatePercent = Math.round(stats.overall_pass_rate);

  const items = [
    {
      icon: <Shield size={24} className="text-indigo-500" />,
      label: '总规则数',
      value: stats.total_rules,
      color: 'bg-indigo-50 border-indigo-100',
    },
    {
      icon: <Activity size={24} className="text-blue-500" />,
      label: '活跃规则',
      value: stats.active_rules,
      color: 'bg-blue-50 border-blue-100',
    },
    {
      icon: <CheckCircle size={24} className="text-emerald-500" />,
      label: '通过检查',
      value: stats.passed_checks.toLocaleString(),
      color: 'bg-emerald-50 border-emerald-100',
    },
    {
      icon: <XCircle size={24} className="text-red-500" />,
      label: '失败检查',
      value: stats.failed_checks.toLocaleString(),
      color: 'bg-red-50 border-red-100',
    },
  ];

  return (
    <div className="space-y-4">
      {/* Overall Pass Rate */}
      <Card className="bg-gradient-to-r from-indigo-500 to-purple-600 text-white">
        <CardContent className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-indigo-100 text-sm mb-1">总体通过率</div>
              <div className="text-4xl font-bold">{passRatePercent}%</div>
              <div className="text-indigo-100 text-sm mt-1">
                共 {stats.total_checks.toLocaleString()} 次检查
              </div>
            </div>
            <div className="w-20 h-20 rounded-full bg-white/20 flex items-center justify-center">
              <Activity size={40} className="text-white" />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Stats Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {items.map((item) => (
          <Card key={item.label} className={`${item.color} border`}>
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-white/80 flex items-center justify-center">
                  {item.icon}
                </div>
                <div>
                  <div className="text-2xl font-bold text-slate-900">
                    {item.value}
                  </div>
                  <div className="text-xs text-slate-500">{item.label}</div>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
};
