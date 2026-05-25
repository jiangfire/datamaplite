import { Layout } from '../components/Layout';
import { useDashboard } from '../hooks/useDashboard';
import {
  Database,
  Table2,
  Columns,
  BookOpen,
  GitCompare,
  ShieldCheck,
  Tag,
  Bell,
  Users,
  Activity,
  TrendingUp,
  AlertTriangle,
} from 'lucide-react';

interface StatCardProps {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  color: string;
  subtitle?: string;
}

function StatCard({ title, value, icon, color, subtitle }: StatCardProps) {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-slate-100 p-6 hover:shadow-md transition-shadow">
      <div className="flex items-center justify-between">
        <div className={`p-3 rounded-lg ${color}`}>
          {icon}
        </div>
        <div className="text-right">
          <div className="text-2xl font-bold text-slate-800">{value}</div>
          <div className="text-sm text-slate-500">{title}</div>
        </div>
      </div>
      {subtitle && (
        <div className="mt-2 text-xs text-slate-400">{subtitle}</div>
      )}
    </div>
  );
}

export function DashboardPage() {
  const { stats, loading } = useDashboard();

  if (loading) {
    return (
      <Layout>
        <div className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="bg-white rounded-xl shadow-sm border border-slate-100 p-6 animate-pulse">
                <div className="h-12 bg-slate-200 rounded mb-4"></div>
                <div className="h-6 bg-slate-200 rounded w-1/2"></div>
              </div>
            ))}
          </div>
        </div>
      </Layout>
    );
  }

  if (!stats) {
    return (
      <Layout>
        <div className="p-6 text-center text-slate-500">Failed to load dashboard data</div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="p-6">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-slate-800">Dashboard</h1>
          <p className="text-slate-500 mt-1">DataMap Lite 数据资产总览</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <StatCard
            title="数据源"
            value={stats.total_sources}
            icon={<Database className="w-6 h-6 text-white" />}
            color="bg-blue-500"
            subtitle="已注册的数据源"
          />
          <StatCard
            title="Schema 对象"
            value={stats.total_objects}
            icon={<Table2 className="w-6 h-6 text-white" />}
            color="bg-indigo-500"
            subtitle="表、视图、集合"
          />
          <StatCard
            title="字段"
            value={stats.total_columns}
            icon={<Columns className="w-6 h-6 text-white" />}
            color="bg-violet-500"
            subtitle="已采集的字段"
          />
          <StatCard
            title="业务术语"
            value={stats.total_terms}
            icon={<BookOpen className="w-6 h-6 text-white" />}
            color="bg-emerald-500"
            subtitle="标准化术语"
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <StatCard
            title="字段映射"
            value={stats.total_mappings}
            icon={<GitCompare className="w-6 h-6 text-white" />}
            color="bg-amber-500"
          />
          <StatCard
            title="数据质量规则"
            value={stats.total_dq_rules}
            icon={<ShieldCheck className="w-6 h-6 text-white" />}
            color="bg-rose-500"
            subtitle={`${stats.active_dq_rules} 个活跃中`}
          />
          <StatCard
            title="标签"
            value={stats.total_tags}
            icon={<Tag className="w-6 h-6 text-white" />}
            color="bg-cyan-500"
          />
          <StatCard
            title="Schema 变更"
            value={stats.recent_changes}
            icon={<Activity className="w-6 h-6 text-white" />}
            color="bg-orange-500"
            subtitle="近期变更记录"
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <StatCard
            title="告警规则"
            value={stats.total_alert_rules}
            icon={<Bell className="w-6 h-6 text-white" />}
            color="bg-red-500"
          />
          <StatCard
            title="系统用户"
            value={stats.total_users}
            icon={<Users className="w-6 h-6 text-white" />}
            color="bg-teal-500"
          />
          <StatCard
            title="质量通过率"
            value={`${stats.overall_pass_rate.toFixed(1)}%`}
            icon={<TrendingUp className="w-6 h-6 text-white" />}
            color={stats.overall_pass_rate >= 90 ? 'bg-green-500' : stats.overall_pass_rate >= 70 ? 'bg-yellow-500' : 'bg-red-500'}
          />
        </div>

        {stats.total_dq_rules > 0 && stats.overall_pass_rate < 70 && (
          <div className="mt-6 p-4 bg-amber-50 border border-amber-200 rounded-lg flex items-center gap-3">
            <AlertTriangle className="w-5 h-5 text-amber-600" />
            <span className="text-amber-800 text-sm">
              数据质量整体通过率较低 ({stats.overall_pass_rate.toFixed(1)}%)，建议检查 DQ 规则配置
            </span>
          </div>
        )}
      </div>
    </Layout>
  );
}
