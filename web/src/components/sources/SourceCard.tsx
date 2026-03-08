import { Link } from 'react-router-dom';
import { Database, MoreVertical, RefreshCw, Trash2, Edit, Check } from 'lucide-react';
import { Card, CardContent, Badge } from '../ui';
import type { DataSource, DataSourceType, DataSourceStatus } from '../../types';
import { useState } from 'react';

interface SourceCardProps {
  source: DataSource;
  onDelete: (id: string) => void;
  onSync: (id: string) => void;
  syncing: boolean;
}

const typeIcons: Record<DataSourceType, string> = {
  mysql: '🔷',
  postgres: '🐘',
  mongodb: '🍃',
  oracle: '🔴',
  mssql: '📊',
};

const typeLabels: Record<DataSourceType, string> = {
  mysql: 'MySQL',
  postgres: 'PostgreSQL',
  mongodb: 'MongoDB',
  oracle: 'Oracle',
  mssql: 'SQL Server',
};

const statusVariants: Record<DataSourceStatus, 'success' | 'warning' | 'error' | 'neutral'> = {
  active: 'success',
  inactive: 'neutral',
  error: 'error',
  syncing: 'warning',
};

const statusLabels: Record<DataSourceStatus, string> = {
  active: '正常',
  inactive: '未激活',
  error: '错误',
  syncing: '同步中',
};

export const SourceCard: React.FC<SourceCardProps> = ({
  source,
  onDelete,
  onSync,
  syncing,
}) => {
  const [showActions, setShowActions] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const handleDelete = () => {
    if (confirmDelete) {
      onDelete(source.id);
      setConfirmDelete(false);
    } else {
      setConfirmDelete(true);
      setTimeout(() => setConfirmDelete(false), 3000);
    }
  };

  return (
    <Card className="relative group">
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          {/* Icon & Info */}
          <div className="flex items-start gap-4">
            <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-indigo-50 to-violet-50 border border-indigo-100 flex items-center justify-center text-2xl">
              {typeIcons[source.type] || <Database size={24} className="text-indigo-600" />}
            </div>
            <div>
              <Link
                to={`/sources/${source.id}`}
                className="text-lg font-semibold text-slate-900 hover:text-indigo-600 transition-colors"
              >
                {source.name}
              </Link>
              <p className="text-sm text-slate-500 mt-1">
                {typeLabels[source.type]} · {source.host}:{source.port}
              </p>
              <div className="flex items-center gap-2 mt-2">
                <Badge variant={statusVariants[source.status]}>
                  {statusLabels[source.status]}
                </Badge>
                <span className="text-xs text-slate-400">{source.database}</span>
              </div>
            </div>
          </div>

          {/* Actions */}
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
                    onSync(source.id);
                    setShowActions(false);
                  }}
                  disabled={syncing || source.status === 'syncing'}
                  className="w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2 disabled:opacity-50"
                >
                  <RefreshCw size={16} className={syncing ? 'animate-spin' : ''} />
                  同步元数据
                </button>
                <Link
                  to={`/sources/${source.id}/edit`}
                  onClick={() => setShowActions(false)}
                  className="w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
                >
                  <Edit size={16} />
                  编辑
                </Link>
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

        {/* Footer Info */}
        <div className="mt-4 pt-4 border-t border-slate-100 flex items-center justify-between text-xs text-slate-400">
          <span>创建于 {new Date(source.created_at).toLocaleDateString()}</span>
          {source.last_sync_at && (
            <span>上次同步 {new Date(source.last_sync_at).toLocaleString()}</span>
          )}
        </div>
      </CardContent>
    </Card>
  );
};
