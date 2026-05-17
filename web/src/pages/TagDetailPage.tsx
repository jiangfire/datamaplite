import { useParams, useNavigate } from 'react-router-dom';
import { Tag, ArrowLeft, Hash, Table, Database } from 'lucide-react';
import { Layout, Card, CardContent } from '../components';
import { useTag } from '../hooks';

export const TagDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { tag, columns, loading, error } = useTag(id);

  return (
    <Layout>
      <button
        onClick={() => navigate('/tags')}
        className="flex items-center text-sm text-slate-500 hover:text-slate-700 mb-4"
      >
        <ArrowLeft size={16} className="mr-1" />
        返回标签列表
      </button>

      {loading ? (
        <div className="text-center py-12">加载中...</div>
      ) : error ? (
        <Card>
          <CardContent className="p-8 text-center text-red-500">{error}</CardContent>
        </Card>
      ) : !tag ? (
        <Card>
          <CardContent className="py-16 text-center text-slate-400">
            <Tag size={48} className="mx-auto mb-4 opacity-30" />
            <p>标签不存在或已被删除</p>
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="mb-6">
            <div className="flex items-center gap-3">
              <span
                className="w-5 h-5 rounded-full"
                style={{ backgroundColor: tag.color }}
              />
              <h1 className="text-2xl font-bold text-slate-900">{tag.name}</h1>
            </div>
            {tag.description && (
              <p className="text-slate-500 mt-1">{tag.description}</p>
            )}
          </div>

          <h2 className="text-lg font-semibold text-slate-900 mb-4">
            关联字段 ({columns.length})
          </h2>

          {columns.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center text-slate-400">
                <Hash size={48} className="mx-auto mb-4 opacity-30" />
                <p>暂无关联字段</p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-3">
              {columns.map((col) => (
                <Card
                  key={col.id}
                  className="cursor-pointer hover:shadow-md transition-shadow"
                  onClick={() => navigate(`/columns/${col.id}`)}
                >
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="w-9 h-9 rounded-lg bg-indigo-50 flex items-center justify-center">
                          <Hash size={18} className="text-indigo-600" />
                        </div>
                        <div>
                          <p className="font-medium text-slate-900">{col.name}</p>
                          <div className="flex items-center gap-2 text-sm text-slate-500">
                            <span className="flex items-center gap-1">
                              <Table size={14} />
                              {col.object_name}
                            </span>
                            <span>·</span>
                            <span className="flex items-center gap-1">
                              <Database size={14} />
                              {col.source_name}
                            </span>
                          </div>
                        </div>
                      </div>
                      <span className="text-xs font-mono text-slate-500 bg-slate-100 px-2 py-0.5 rounded">
                        {col.data_type}
                      </span>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </>
      )}
    </Layout>
  );
};
