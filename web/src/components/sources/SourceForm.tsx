import { useState, useEffect } from 'react';
import { Button, Input, Select, Modal } from '../ui';
import { sourceService } from '../../services';
import type {
  DataSourceCreate,
  DataSourceUpdate,
  DataSourceType,
} from '../../types';

type SourceFormProps =
  | {
      isOpen: boolean;
      onClose: () => void;
      onSubmit: (data: DataSourceCreate) => void;
      initialData?: Partial<DataSourceCreate>;
      mode: 'create';
    }
  | {
      isOpen: boolean;
      onClose: () => void;
      onSubmit: (data: DataSourceUpdate) => void;
      initialData?: Partial<DataSourceCreate>;
      mode: 'edit';
    };

const dataSourceTypes = [
  { value: 'mysql', label: 'MySQL' },
  { value: 'postgres', label: 'PostgreSQL' },
  { value: 'mongodb', label: 'MongoDB' },
];

const defaultPorts: Record<DataSourceType, number> = {
  mysql: 3306,
  postgres: 5432,
  mongodb: 27017,
};

export const SourceForm: React.FC<SourceFormProps> = ({
  isOpen,
  onClose,
  onSubmit,
  initialData,
  mode,
}) => {
  const [formData, setFormData] = useState<DataSourceCreate>({
    name: '',
    type: 'mysql',
    host: '',
    port: 3306,
    database: '',
    username: '',
    password: '',
    description: '',
  });
  const [testing, setTesting] = useState(false);
  const [connectionCheck, setConnectionCheck] = useState<{
    tone: 'positive' | 'negative';
    message: string;
  } | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (initialData) {
      setFormData((prev) => ({ ...prev, ...initialData }));
    }
  }, [initialData]);

  const handleTypeChange = (type: DataSourceType) => {
    setFormData((prev) => ({
      ...prev,
      type,
      port: defaultPorts[type],
    }));
  };

  const validate = () => {
    const newErrors: Record<string, string> = {};
    if (!formData.name.trim()) newErrors.name = '名称不能为空';
    if (!formData.host.trim()) newErrors.host = '主机地址不能为空';
    if (!formData.port || formData.port < 1 || formData.port > 65535) {
      newErrors.port = '端口号必须在 1-65535 之间';
    }
    if (!formData.database.trim()) newErrors.database = '数据库名不能为空';
    if (!formData.username.trim()) newErrors.username = '用户名不能为空';
    if (mode === 'create' && !formData.password)
      newErrors.password = '密码不能为空';

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    if (mode === 'edit') {
      // 编辑模式下不发送空的凭据字段，避免后端误解释为"清空"
      const { username, password, ...rest } = formData;
      const payload: DataSourceUpdate = { ...rest };
      if (username) payload.username = username;
      if (password) payload.password = password;
      onSubmit(payload);
    } else {
      onSubmit(formData);
    }
  };

  const handleTestConnection = async () => {
    if (!validate()) return;

    setTesting(true);
    setConnectionCheck(null);
    try {
      await sourceService.testConnection({
        type: formData.type,
        host: formData.host,
        port: formData.port,
        database: formData.database,
        username: formData.username,
        password: formData.password,
      });
      setConnectionCheck({
        tone: 'positive',
        message: '连接测试成功',
      });
    } catch (err) {
      setConnectionCheck({
        tone: 'negative',
        message: err instanceof Error ? err.message : '连接测试失败',
      });
    } finally {
      setTesting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={mode === 'create' ? '创建数据源' : '编辑数据源'}
      size="lg"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <Input
            label="名称"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            error={errors.name}
            required
          />
          <Select
            label="类型"
            value={formData.type}
            onChange={(e) => handleTypeChange(e.target.value as DataSourceType)}
            options={dataSourceTypes}
            required
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Input
            label="主机地址"
            value={formData.host}
            onChange={(e) => setFormData({ ...formData, host: e.target.value })}
            error={errors.host}
            required
          />
          <Input
            label="端口"
            type="number"
            value={formData.port}
            onChange={(e) =>
              setFormData({ ...formData, port: parseInt(e.target.value) })
            }
            error={errors.port}
            required
          />
        </div>

        <Input
          label="数据库名"
          value={formData.database}
          onChange={(e) =>
            setFormData({ ...formData, database: e.target.value })
          }
          error={errors.database}
          required
        />

        <div className="grid grid-cols-2 gap-4">
          <Input
            label="用户名"
            value={formData.username}
            onChange={(e) =>
              setFormData({ ...formData, username: e.target.value })
            }
            error={errors.username}
            required
          />
          <Input
            label="密码"
            type="password"
            value={formData.password}
            onChange={(e) =>
              setFormData({ ...formData, password: e.target.value })
            }
            error={errors.password}
            required={mode === 'create'}
            helperText={mode === 'edit' ? '留空表示不修改密码' : undefined}
          />
        </div>

        <Input
          label="描述"
          value={formData.description || ''}
          onChange={(e) =>
            setFormData({ ...formData, description: e.target.value })
          }
        />

        {connectionCheck && (
          <div
            className={`p-3 rounded-lg text-sm ${
              connectionCheck.tone === 'positive'
                ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                : 'bg-red-50 text-red-700 border border-red-200'
            }`}
          >
            {connectionCheck.tone === 'positive' ? '✓ ' : '✗ '}
            {connectionCheck.message}
          </div>
        )}

        <div className="flex justify-between pt-4 border-t">
          <Button
            type="button"
            variant="secondary"
            onClick={handleTestConnection}
            loading={testing}
          >
            测试连接
          </Button>
          <div className="flex gap-3">
            <Button type="button" variant="ghost" onClick={onClose}>
              取消
            </Button>
            <Button type="submit">{mode === 'create' ? '创建' : '保存'}</Button>
          </div>
        </div>
      </form>
    </Modal>
  );
};
