import { useState } from 'react';
import { Plus, Tag, Edit2, Trash2 } from 'lucide-react';
import { Layout, Button, Card, CardContent } from '../components';
import { useTags } from '../hooks';
import type { TagCreate } from '../types';

const PRESET_COLORS = [
  '#ef4444', '#f97316', '#f59e0b', '#84cc16', '#22c55e',
  '#10b981', '#14b8a6', '#06b6d4', '#0ea5e9', '#3b82f6',
  '#6366f1', '#8b5cf6', '#a855f7', '#d946ef', '#ec4899',
];

export const TagsPage: React.FC = () => {
  const { tags, loading, error, refetch, createTag, deleteTag } = useTags();
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newTagName, setNewTagName] = useState('');
  const [newTagDesc, setNewTagDesc] = useState('');
  const [selectedColor, setSelectedColor] = useState(PRESET_COLORS[10]);

  const handleCreate = async () => {
    if (!newTagName.trim()) return;
    const data: TagCreate = {
      name: newTagName.trim(),
      color: selectedColor,
      description: newTagDesc.trim() || undefined,
    };
    await createTag(data);
    setNewTagName('');
    setNewTagDesc('');
    setShowCreateForm(false);
  };

  return (
    <Layout>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">字段标签</h1>
          <p className="text-slate-500 mt-1">管理字段分类标签</p>
        </div>
        <Button onClick={() => setShowCreateForm(true)}>
          <Plus size={18} className="mr-2" />
          创建标签
        </Button>
      </div>

      {showCreateForm && (
        <Card className="mb-6">
          <CardContent className="p-4">
            <h3 className="font-medium mb-4">创建新标签</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-slate-600 mb-1">名称</label>
                <input
                  type="text"
                  value={newTagName}
                  onChange={(e) => setNewTagName(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg"
                  placeholder="例如：PII"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-600 mb-1">描述</label>
                <input
                  type="text"
                  value={newTagDesc}
                  onChange={(e) => setNewTagDesc(e.target.value)}
                  className="w-full px-3 py-2 border rounded-lg"
                  placeholder="可选描述"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-600 mb-2">颜色</label>
                <div className="flex flex-wrap gap-2">
                  {PRESET_COLORS.map((color) => (
                    <button
                      key={color}
                      onClick={() => setSelectedColor(color)}
                      className={`w-8 h-8 rounded-full ${selectedColor === color ? 'ring-2 ring-offset-2 ring-slate-400' : ''}`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
              </div>
              <div className="flex gap-2">
                <Button variant="secondary" onClick={() => setShowCreateForm(false)}>取消</Button>
                <Button onClick={handleCreate}>创建</Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {loading ? (
        <div className="text-center py-12">加载中...</div>
      ) : error ? (
        <Card><CardContent className="p-8 text-center text-red-500">{error}</CardContent></Card>
      ) : tags.length === 0 ? (
        <Card>
          <CardContent className="py-16 text-center">
            <div className="w-16 h-16 rounded-full bg-slate-100 flex items-center justify-center mx-auto mb-4">
              <Tag size={32} className="text-slate-400" />
            </div>
            <p className="text-slate-500">暂无标签，创建一个吧</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {tags.map((tag) => (
            <Card key={tag.id}>
              <CardContent className="p-4">
                <div className="flex items-center gap-3">
                  <span
                    className="w-4 h-4 rounded-full"
                    style={{ backgroundColor: tag.color }}
                  />
                  <div className="flex-1">
                    <h3 className="font-medium">{tag.name}</h3>
                    {tag.description && (
                      <p className="text-sm text-slate-500">{tag.description}</p>
                    )}
                  </div>
                  <button
                    onClick={() => deleteTag(tag.id)}
                    className="p-2 text-slate-400 hover:text-red-500"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </Layout>
  );
};
