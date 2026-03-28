import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, BookOpen, Hash, ShieldCheck, User } from 'lucide-react';
import { Badge, Button, Card, CardContent, Layout } from '../components';
import { useTerm } from '../hooks';

export const TermDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { term, loading, error } = useTerm(id);

  if (loading) {
    return (
      <Layout>
        <div className="py-12 text-center">
          <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto" />
          <p className="text-slate-500 mt-4">加载中...</p>
        </div>
      </Layout>
    );
  }

  if (error || !term) {
    return (
      <Layout>
        <Card>
          <CardContent className="py-12 text-center text-red-500">
            <p>加载失败: {error || '术语不存在'}</p>
            <Link to="/terms">
              <Button variant="secondary" className="mt-4">
                返回术语列表
              </Button>
            </Link>
          </CardContent>
        </Card>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="mb-6">
        <Link
          to="/terms"
          className="inline-flex items-center text-sm text-slate-500 hover:text-indigo-600"
        >
          <ArrowLeft size={16} className="mr-1" />
          返回术语列表
        </Link>
      </div>

      <Card>
        <CardContent className="p-6 space-y-6">
          <div className="flex items-start gap-4">
            <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-emerald-50 to-teal-50 border border-emerald-100 flex items-center justify-center">
              <BookOpen size={28} className="text-emerald-600" />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2 flex-wrap">
                <h1 className="text-2xl font-bold text-slate-900">{term.name}</h1>
                {term.category && <Badge variant="info">{term.category}</Badge>}
                {term.status && (
                  <Badge
                    variant={term.status === 'deprecated' ? 'error' : 'success'}
                  >
                    {term.status}
                  </Badge>
                )}
              </div>
              {term.description && (
                <p className="text-slate-600 mt-2">{term.description}</p>
              )}
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <Card>
              <CardContent className="p-4">
                <p className="text-xs text-slate-500 uppercase tracking-wider">
                  标准编码
                </p>
                <p className="mt-2 font-medium text-slate-900 flex items-center gap-2">
                  <Hash size={16} className="text-slate-400" />
                  {term.standard_code || '未设置'}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="p-4">
                <p className="text-xs text-slate-500 uppercase tracking-wider">
                  业务域
                </p>
                <p className="mt-2 font-medium text-slate-900">
                  {term.domain || '未设置'}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="p-4">
                <p className="text-xs text-slate-500 uppercase tracking-wider">
                  负责人
                </p>
                <p className="mt-2 font-medium text-slate-900 flex items-center gap-2">
                  <User size={16} className="text-slate-400" />
                  {term.owner || '未设置'}
                </p>
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardContent className="p-4">
                <p className="text-xs text-slate-500 uppercase tracking-wider">
                  标准数据类型
                </p>
                <p className="mt-2 font-medium text-slate-900">
                  {term.data_type_standard || '未设置'}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="p-4">
                <p className="text-xs text-slate-500 uppercase tracking-wider">
                  校验规则
                </p>
                <p className="mt-2 text-slate-900 flex items-start gap-2">
                  <ShieldCheck size={16} className="text-slate-400 mt-0.5" />
                  <span>{term.validation_rule || '未设置'}</span>
                </p>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>
    </Layout>
  );
};
