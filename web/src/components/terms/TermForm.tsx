import { useState, useEffect } from 'react';
import { Button, Input, Modal } from '../ui';
import type { BusinessTermCreate } from '../../types';

interface TermFormProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: BusinessTermCreate) => void;
  initialData?: Partial<BusinessTermCreate>;
  mode: 'create' | 'edit';
}

const categories = [
  '主数据',
  '业务指标',
  '技术字段',
  '维度属性',
  '度量值',
  '其他',
];

export const TermForm: React.FC<TermFormProps> = ({
  isOpen,
  onClose,
  onSubmit,
  initialData,
  mode,
}) => {
  const [formData, setFormData] = useState<BusinessTermCreate>({
    name: '',
    description: '',
    category: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (initialData) {
      setFormData((prev) => ({ ...prev, ...initialData }));
    }
  }, [initialData]);

  const validate = () => {
    const newErrors: Record<string, string> = {};
    if (!formData.name.trim()) newErrors.name = '名称不能为空';
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validate()) {
      onSubmit(formData);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={mode === 'create' ? '创建业务术语' : '编辑业务术语'}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="名称"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          error={errors.name}
          required
        />

        <Input
          label="描述"
          value={formData.description || ''}
          onChange={(e) => setFormData({ ...formData, description: e.target.value })}
        />

        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-slate-700">分类</label>
          <div className="flex flex-wrap gap-2">
            {categories.map((cat) => (
              <button
                key={cat}
                type="button"
                onClick={() => setFormData({ ...formData, category: cat })}
                className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
                  formData.category === cat
                    ? 'bg-indigo-100 text-indigo-700 border border-indigo-300'
                    : 'bg-slate-100 text-slate-600 hover:bg-slate-200 border border-transparent'
                }`}
              >
                {cat}
              </button>
            ))}
            <button
              type="button"
              onClick={() => setFormData({ ...formData, category: '' })}
              className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
                !formData.category
                  ? 'bg-indigo-100 text-indigo-700 border border-indigo-300'
                  : 'bg-slate-100 text-slate-600 hover:bg-slate-200 border border-transparent'
              }`}
            >
              无分类
            </button>
          </div>
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t">
          <Button type="button" variant="ghost" onClick={onClose}>
            取消
          </Button>
          <Button type="submit">{mode === 'create' ? '创建' : '保存'}</Button>
        </div>
      </form>
    </Modal>
  );
};
