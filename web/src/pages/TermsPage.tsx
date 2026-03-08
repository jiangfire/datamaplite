import { useState } from 'react';
import { Plus, BookOpen } from 'lucide-react';
import { Layout, Button, Card, CardContent, TermCard, TermForm } from '../components';
import { useTerms } from '../hooks';
import type { BusinessTerm, BusinessTermCreate } from '../types';

export const TermsPage: React.FC = () => {
  const { terms, loading, error, refetch, createTerm, updateTerm, deleteTerm } = useTerms();
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingTerm, setEditingTerm] = useState<BusinessTerm | null>(null);

  const handleCreate = async (data: BusinessTermCreate) => {
    await createTerm(data);
    setShowCreateForm(false);
  };

  const handleEdit = async (data: BusinessTermCreate) => {
    if (!editingTerm) return;
    await updateTerm(editingTerm.id, data);
    setEditingTerm(null);
  };

  return (
    <Layout>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">业务术语</h1>
          <p className="text-slate-500 mt-1">
            定义数据标准，建立业务与技术之间的桥梁
          </p>
        </div>
        <Button onClick={() => setShowCreateForm(true)}>
          <Plus size={18} className="mr-2" />
          添加术语
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
      ) : terms.length === 0 ? (
        <Card>
          <CardContent className="py-16 text-center">
            <div className="w-20 h-20 rounded-2xl bg-emerald-50 flex items-center justify-center mx-auto mb-6">
              <BookOpen size={40} className="text-emerald-400" />
            </div>
            <h3 className="text-lg font-medium text-slate-900 mb-2">
              暂无业务术语
            </h3>
            <p className="text-slate-500 mb-6">
              添加业务术语来标准化数据字段描述
            </p>
            <Button onClick={() => setShowCreateForm(true)}>
              <Plus size={18} className="mr-2" />
              添加术语
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {terms.map((term) => (
            <TermCard
              key={term.id}
              term={term}
              onEdit={setEditingTerm}
              onDelete={deleteTerm}
            />
          ))}
        </div>
      )}

      {/* Create Form Modal */}
      <TermForm
        isOpen={showCreateForm}
        onClose={() => setShowCreateForm(false)}
        onSubmit={handleCreate}
        mode="create"
      />

      {/* Edit Form Modal */}
      <TermForm
        isOpen={!!editingTerm}
        onClose={() => setEditingTerm(null)}
        onSubmit={handleEdit}
        initialData={editingTerm || undefined}
        mode="edit"
      />
    </Layout>
  );
};
