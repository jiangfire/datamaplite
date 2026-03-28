import { useState } from 'react';
import { Button, Input, Modal, Select } from '../ui';
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
  const [formData, setFormData] = useState<BusinessTermCreate>(() => ({
    name: '',
    description: '',
    category: '',
    standard_code: '',
    domain: '',
    data_type_standard: '',
    validation_rule: '',
    owner: '',
    status: 'active',
    ...initialData,
  }));
  const [errors, setErrors] = useState<Record<string, string>>({});

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
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={mode === 'create' ? '创建业务术语' : '编辑业务术语'}
    >
      <form onSubmit={handleSubmit} className="space-y-4 max-h-[70vh] overflow-y-auto pr-2">
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
          onChange={(e) =>
            setFormData({ ...formData, description: e.target.value })
          }
        />

        <div className="grid grid-cols-2 gap-4">
          <Input
            label="标准编码"
            value={formData.standard_code || ''}
            onChange={(e) =>
              setFormData({ ...formData, standard_code: e.target.value })
            }
            placeholder="例如：STD-001"
          />
          <Input
            label="业务域"
            value={formData.domain || ''}
            onChange={(e) =>
              setFormData({ ...formData, domain: e.target.value })
            }
            placeholder="例如：制造/财务"
          />
        </div>

        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-slate-700">
            分类
          </label>
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

        <div className="grid grid-cols-2 gap-4">
          <Input
            label="标准数据类型"
            value={formData.data_type_standard || ''}
            onChange={(e) =>
              setFormData({
                ...formData,
                data_type_standard: e.target.value,
              })
            }
            placeholder="例如：varchar(64)"
          />
          <Input
            label="负责人"
            value={formData.owner || ''}
            onChange={(e) =>
              setFormData({ ...formData, owner: e.target.value })
            }
            placeholder="例如：数据治理组"
          />
        </div>

        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-slate-700">
            校验规则
          </label>
          <textarea
            value={formData.validation_rule || ''}
            onChange={(e) =>
              setFormData({
                ...formData,
                validation_rule: e.target.value,
              })
            }
            className="w-full px-3 py-2 border rounded-lg border-slate-300 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            rows={3}
            placeholder="例如：非空，且满足长度 <= 64"
          />
        </div>

        <Select
          label="状态"
          value={formData.status || 'active'}
          onChange={(e) => setFormData({ ...formData, status: e.target.value })}
          options={[
            { value: 'active', label: '启用' },
            { value: 'draft', label: '草稿' },
            { value: 'deprecated', label: '废弃' },
          ]}
        />

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
