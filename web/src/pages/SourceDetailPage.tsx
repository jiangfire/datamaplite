import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, RefreshCw, Database, History } from 'lucide-react';
import {
  Layout,
  Button,
  Card,
  CardContent,
  SchemaTree,
  Badge,
} from '../components';
import { useSource, useSchemaTree, useSchemaChanges, useSyncSource } from '../hooks';

const typeLabels: Record<string, string> = {
  mysql: 'MySQL',
  postgres: 'PostgreSQL',
  mongodb: 'MongoDB',
};

const statusLabels: Record<string, string> = {
  active: '正常',
  inactive: '未激活',
  error: '错误',
  syncing: '同步中',
};

const statusVariants: Record<
  string,
  'success' | 'warning' | 'error' | 'neutral'
> = {
  active: 'success',
  inactive: 'neutral',
  error: 'error',
  syncing: 'warning',
};

export const SourceDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { source, loading: sourceLoading, error: sourceError } = useSource(id);
  const { schemaTree, loading: schemaLoading, refetch } = useSchemaTree(id);
  const { changes, loading: changesLoading } = useSchemaChanges(id);
  const { sync, syncing } = useSyncSource();

  const handleSync = async () => {
    if (!id) return;
    await sync(id);
    refetch();
  };

  if (sourceLoading) {
    return (
      <Layout>
        <div className="py-12 text-center">
          <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto" />
          <p className="text-slate-500 mt-4">加载中...</p>
        </div>
      </Layout>
    );
  }

  if (sourceError || !source) {
    return (
      <Layout>
        <Card>
          <CardContent className="py-12 text-center text-red-500">
            <p>加载失败: {sourceError || '数据源不存在'}</p>
            <Link to="/">
              <Button variant="secondary" className="mt-4">
                返回列表
              </Button>
            </Link>
          </CardContent>
        </Card>
      </Layout>
    );
  }

  return (
    <Layout>
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link
          to="/"
          className="inline-flex items-center text-sm text-slate-500 hover:text-indigo-600"
        >
          <ArrowLeft size={16} className="mr-1" />
          返回数据源列表
        </Link>
      </div>

      {/* Header */}
      <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-6 mb-6">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-4">
            <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center text-2xl shadow-lg shadow-indigo-500/20">
              🔷
            </div>
            <div>
              <h1 className="text-2xl font-bold text-slate-900">
                {source.name}
              </h1>
              <div className="flex items-center gap-3 mt-2">
                <Badge variant={statusVariants[source.status]}>
                  {statusLabels[source.status]}
                </Badge>
                <span className="text-sm text-slate-500">
                  {typeLabels[source.type]} · {source.host}:{source.port}
                </span>
              </div>
              {source.description && (
                <p className="text-slate-600 mt-2">{source.description}</p>
              )}
            </div>
          </div>
          <Button onClick={handleSync} loading={syncing} disabled={syncing}>
            <RefreshCw
              size={18}
              className={syncing ? 'animate-spin mr-2' : 'mr-2'}
            />
            同步元数据
          </Button>
        </div>

        {/* Info Grid */}
        <div className="grid grid-cols-4 gap-4 mt-6 pt-6 border-t border-slate-100">
          <div>
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              数据库
            </p>
            <p className="font-medium text-slate-900 mt-1">{source.database}</p>
          </div>
          <div>
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              状态
            </p>
            <p className="font-medium text-slate-900 mt-1">
              {statusLabels[source.status]}
            </p>
          </div>
          <div>
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              上次同步
            </p>
            <p className="font-medium text-slate-900 mt-1">
              {source.last_sync_at
                ? new Date(source.last_sync_at).toLocaleString()
                : '从未'}
            </p>
          </div>
          <div>
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              创建时间
            </p>
            <p className="font-medium text-slate-900 mt-1">
              {new Date(source.created_at).toLocaleDateString()}
            </p>
          </div>
        </div>
      </div>

      {/* Schema Tree */}
      <div>
        <h2 className="text-lg font-semibold text-slate-900 mb-4">
          Schema 浏览器
        </h2>
        {schemaLoading ? (
          <Card>
            <CardContent className="py-12 text-center">
              <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto" />
              <p className="text-slate-500 mt-4">加载 Schema...</p>
            </CardContent>
          </Card>
        ) : schemaTree ? (
          <SchemaTree objects={schemaTree.objects} />
        ) : (
          <Card>
            <CardContent className="py-12 text-center text-slate-400">
              <Database size={48} className="mx-auto mb-4 opacity-30" />
              <p>暂无 Schema 数据</p>
              <p className="text-sm mt-2">点击上方"同步元数据"按钮开始采集</p>
            </CardContent>
          </Card>
        )}
      </div>

      <div className="mt-8">
        <h2 className="text-lg font-semibold text-slate-900 mb-4 flex items-center gap-2">
          <History size={18} />
          最近变更
        </h2>
        {changesLoading ? (
          <Card>
            <CardContent className="py-8 text-center text-slate-500">
              加载变更记录中...
            </CardContent>
          </Card>
        ) : changes.length === 0 ? (
          <Card>
            <CardContent className="py-8 text-center text-slate-500">
              暂无变更记录
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {changes.map((change) => (
              <Card key={change.id}>
                <CardContent className="p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="font-medium text-slate-900">
                        {change.object_name}
                      </p>
                      <p className="text-sm text-slate-500 mt-1">
                        {change.change_type} · {change.object_type}
                      </p>
                      {(change.old_value || change.new_value) && (
                        <p className="text-xs text-slate-500 mt-2">
                          {change.old_value || '-'} → {change.new_value || '-'}
                        </p>
                      )}
                    </div>
                    <span className="text-xs text-slate-400">
                      {new Date(change.detected_at).toLocaleString()}
                    </span>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </Layout>
  );
};
