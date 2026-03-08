import { useState } from 'react';
import { Plus, Database } from 'lucide-react';
import {
  Layout,
  Button,
  Card,
  CardContent,
  SourceCard,
  SourceForm,
} from '../components';
import { useSources, useSyncSource } from '../hooks';

export const SourcesPage: React.FC = () => {
  const { sources, loading, error, refetch, createSource, deleteSource } = useSources();
  const { sync, syncing } = useSyncSource();
  const [showCreateForm, setShowCreateForm] = useState(false);

  const handleCreate = async (data: Parameters<typeof createSource>[0]) => {
    await createSource(data);
    setShowCreateForm(false);
  };

  const handleSync = async (id: string) => {
    await sync(id);
    refetch();
  };

  return (
    <Layout>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">数据源管理</h1>
          <p className="text-slate-500 mt-1">
            管理您的数据库连接并同步元数据
          </p>
        </div>
        <Button onClick={() => setShowCreateForm(true)}>
          <Plus size={18} className="mr-2" />
          添加数据源
        </Button>
      </div>

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
      ) : sources.length === 0 ? (
        <Card>
          <CardContent className="py-16 text-center">
            <div className="w-20 h-20 rounded-2xl bg-indigo-50 flex items-center justify-center mx-auto mb-6">
              <Database size={40} className="text-indigo-400" />
            </div>
            <h3 className="text-lg font-medium text-slate-900 mb-2">
              暂无数据源
            </h3>
            <p className="text-slate-500 mb-6">
              添加您的第一个数据源开始元数据管理
            </p>
            <Button onClick={() => setShowCreateForm(true)}>
              <Plus size={18} className="mr-2" />
              添加数据源
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {sources.map((source) => (
            <SourceCard
              key={source.id}
              source={source}
              onDelete={deleteSource}
              onSync={handleSync}
              syncing={syncing}
            />
          ))}
        </div>
      )}

      {/* Create Form Modal */}
      <SourceForm
        isOpen={showCreateForm}
        onClose={() => setShowCreateForm(false)}
        onSubmit={handleCreate}
        mode="create"
      />
    </Layout>
  );
};
