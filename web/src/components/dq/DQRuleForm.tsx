import { useEffect, useMemo, useState } from 'react';
import { sourceService } from '../../services';
import type {
  DataSource,
  DQRuleCreate,
  DQRuleType,
  DQSeverity,
  SchemaTree,
} from '../../types';
import { Button, Input, Modal, Select } from '../ui';

interface DQRuleFormProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: DQRuleCreate) => void;
  initialData?: Partial<DQRuleCreate>;
  mode: 'create' | 'edit';
}

const ruleTypes: { value: DQRuleType; label: string; description: string }[] = [
  { value: 'not_null', label: '非空检查', description: '检查字段值是否为空' },
  { value: 'unique', label: '唯一性', description: '检查字段值是否唯一' },
  { value: 'regex', label: '正则匹配', description: '使用正则表达式验证格式' },
  {
    value: 'range',
    label: '范围检查',
    description: '检查数值是否在指定范围内',
  },
  { value: 'enum', label: '枚举值', description: '检查值是否在预定义列表中' },
  {
    value: 'custom_sql',
    label: '自定义SQL',
    description: '执行自定义 SQL 返回失败样本',
  },
  {
    value: 'referential',
    label: '引用完整性',
    description: '检查外键引用是否有效',
  },
];

const severities: { value: DQSeverity; label: string; color: string }[] = [
  {
    value: 'error',
    label: '错误',
    color: 'text-red-600 bg-red-50 border-red-200',
  },
  {
    value: 'warning',
    label: '警告',
    color: 'text-amber-600 bg-amber-50 border-amber-200',
  },
  {
    value: 'info',
    label: '信息',
    color: 'text-blue-600 bg-blue-50 border-blue-200',
  },
];

const buildInitialFormData = (
  initialData?: Partial<DQRuleCreate>,
): DQRuleCreate => ({
  source_id: '',
  object_id: '',
  column_id: '',
  name: '',
  description: '',
  rule_type: 'not_null',
  rule_config: {},
  severity: 'error',
  is_active: true,
  ...initialData,
});

const buildInitialRuleConfigState = (initialData?: Partial<DQRuleCreate>) => {
  const config = initialData?.rule_config ?? {};

  return {
    regexPattern:
      initialData?.rule_type === 'regex'
        ? ((config.pattern as string) ?? '')
        : '',
    regexFlags:
      initialData?.rule_type === 'regex'
        ? ((config.flags as string) ?? '')
        : '',
    rangeMin:
      initialData?.rule_type === 'range' && config.min != null
        ? String(config.min)
        : '',
    rangeMax:
      initialData?.rule_type === 'range' && config.max != null
        ? String(config.max)
        : '',
    enumValues:
      initialData?.rule_type === 'enum'
        ? ((config.values as string[]) ?? []).join(',')
        : '',
    customSQL:
      initialData?.rule_type === 'custom_sql'
        ? ((config.sql as string) ?? '')
        : '',
    refObjectId:
      initialData?.rule_type === 'referential'
        ? ((config.ref_object_id as string) ?? '')
        : '',
    refColumnId:
      initialData?.rule_type === 'referential'
        ? ((config.ref_column_id as string) ?? '')
        : '',
  };
};

const needsColumnContext = (ruleType: DQRuleType) =>
  ruleType !== 'custom_sql';

export const DQRuleForm: React.FC<DQRuleFormProps> = ({
  isOpen,
  onClose,
  onSubmit,
  initialData,
  mode,
}) => {
  const initialRuleConfigState = buildInitialRuleConfigState(initialData);
  const [formData, setFormData] = useState<DQRuleCreate>(() =>
    buildInitialFormData(initialData),
  );
  const [errors, setErrors] = useState<Record<string, string>>({});

  const [regexPattern, setRegexPattern] = useState(
    initialRuleConfigState.regexPattern,
  );
  const [regexFlags, setRegexFlags] = useState(
    initialRuleConfigState.regexFlags,
  );
  const [rangeMin, setRangeMin] = useState(initialRuleConfigState.rangeMin);
  const [rangeMax, setRangeMax] = useState(initialRuleConfigState.rangeMax);
  const [enumValues, setEnumValues] = useState(
    initialRuleConfigState.enumValues,
  );
  const [customSQL, setCustomSQL] = useState(initialRuleConfigState.customSQL);
  const [refObjectId, setRefObjectId] = useState(
    initialRuleConfigState.refObjectId,
  );
  const [refColumnId, setRefColumnId] = useState(
    initialRuleConfigState.refColumnId,
  );

  const [sources, setSources] = useState<DataSource[]>([]);
  const [schemaTree, setSchemaTree] = useState<SchemaTree | null>(null);
  const [sourcesLoading, setSourcesLoading] = useState(false);
  const [schemaLoading, setSchemaLoading] = useState(false);

  useEffect(() => {
    if (!isOpen) return;

    setFormData(buildInitialFormData(initialData));
    const nextConfig = buildInitialRuleConfigState(initialData);
    setRegexPattern(nextConfig.regexPattern);
    setRegexFlags(nextConfig.regexFlags);
    setRangeMin(nextConfig.rangeMin);
    setRangeMax(nextConfig.rangeMax);
    setEnumValues(nextConfig.enumValues);
    setCustomSQL(nextConfig.customSQL);
    setRefObjectId(nextConfig.refObjectId);
    setRefColumnId(nextConfig.refColumnId);
    setErrors({});
  }, [initialData, isOpen]);

  useEffect(() => {
    if (!isOpen) return;

    const fetchSources = async () => {
      setSourcesLoading(true);
      try {
        const data = await sourceService.listSources();
        setSources(data);
      } finally {
        setSourcesLoading(false);
      }
    };

    fetchSources();
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen || !formData.source_id) {
      setSchemaTree(null);
      return;
    }

    const fetchSchemaTree = async () => {
      setSchemaLoading(true);
      try {
        const data = await sourceService.getSchemaTree(formData.source_id!);
        setSchemaTree(data);
      } catch {
        setSchemaTree(null);
      } finally {
        setSchemaLoading(false);
      }
    };

    fetchSchemaTree();
  }, [formData.source_id, isOpen]);

  const objectOptions = useMemo(
    () => [
      { value: '', label: schemaLoading ? '加载对象中...' : '请选择对象' },
      ...((schemaTree?.objects ?? []).map((object) => ({
        value: object.id,
        label: `${object.schema ? `${object.schema}.` : ''}${object.name}`,
      })) ?? []),
    ],
    [schemaLoading, schemaTree],
  );

  const selectedObject = useMemo(
    () =>
      schemaTree?.objects.find((object) => object.id === formData.object_id) ??
      null,
    [formData.object_id, schemaTree],
  );

  const columnOptions = useMemo(
    () => [
      { value: '', label: '请选择字段' },
      ...((selectedObject?.columns ?? []).map((column) => ({
        value: column.id,
        label: `${column.name} (${column.data_type})`,
      })) ?? []),
    ],
    [selectedObject],
  );

  const selectedRefObject = useMemo(
    () => schemaTree?.objects.find((object) => object.id === refObjectId) ?? null,
    [refObjectId, schemaTree],
  );

  const refColumnOptions = useMemo(
    () => [
      { value: '', label: '请选择引用字段' },
      ...((selectedRefObject?.columns ?? []).map((column) => ({
        value: column.id,
        label: `${column.name} (${column.data_type})`,
      })) ?? []),
    ],
    [selectedRefObject],
  );

  const buildRuleConfig = (): Record<string, unknown> => {
    switch (formData.rule_type) {
      case 'regex':
        return { pattern: regexPattern, flags: regexFlags || undefined };
      case 'range':
        return {
          min: rangeMin ? parseFloat(rangeMin) : undefined,
          max: rangeMax ? parseFloat(rangeMax) : undefined,
        };
      case 'enum':
        return {
          values: enumValues
            .split(',')
            .map((value) => value.trim())
            .filter(Boolean),
        };
      case 'custom_sql':
        return { sql: customSQL };
      case 'referential':
        return { ref_object_id: refObjectId, ref_column_id: refColumnId };
      default:
        return {};
    }
  };

  const validate = () => {
    const newErrors: Record<string, string> = {};
    if (!formData.name.trim()) newErrors.name = '规则名称不能为空';
    if (!formData.source_id) newErrors.source_id = '数据源不能为空';

    if (needsColumnContext(formData.rule_type)) {
      if (!formData.object_id) newErrors.object_id = '对象不能为空';
      if (!formData.column_id) newErrors.column_id = '字段不能为空';
    }

    switch (formData.rule_type) {
      case 'regex':
        if (!regexPattern.trim()) newErrors.regexPattern = '正则表达式不能为空';
        break;
      case 'range':
        if (rangeMin && Number.isNaN(parseFloat(rangeMin))) {
          newErrors.rangeMin = '最小值必须是数字';
        }
        if (rangeMax && Number.isNaN(parseFloat(rangeMax))) {
          newErrors.rangeMax = '最大值必须是数字';
        }
        break;
      case 'enum':
        if (!enumValues.trim()) newErrors.enumValues = '枚举值不能为空';
        break;
      case 'custom_sql':
        if (!customSQL.trim()) newErrors.customSQL = 'SQL语句不能为空';
        break;
      case 'referential':
        if (!refObjectId) newErrors.refObjectId = '引用对象不能为空';
        if (!refColumnId) newErrors.refColumnId = '引用字段不能为空';
        break;
      default:
        break;
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    onSubmit({
      ...formData,
      source_id: formData.source_id || undefined,
      object_id: formData.object_id || undefined,
      column_id: formData.column_id || undefined,
      rule_config: buildRuleConfig(),
    });
  };

  const renderRuleConfig = () => {
    switch (formData.rule_type) {
      case 'regex':
        return (
          <div className="space-y-3">
            <Input
              label="正则表达式 *"
              value={regexPattern}
              onChange={(e) => setRegexPattern(e.target.value)}
              error={errors.regexPattern}
              placeholder="例如: ^[a-zA-Z0-9]+$"
            />
            <Input
              label="标志 (可选)"
              value={regexFlags}
              onChange={(e) => setRegexFlags(e.target.value)}
              placeholder="例如: i"
            />
          </div>
        );
      case 'range':
        return (
          <div className="grid grid-cols-2 gap-3">
            <Input
              label="最小值"
              type="number"
              value={rangeMin}
              onChange={(e) => setRangeMin(e.target.value)}
              error={errors.rangeMin}
              placeholder="无限制"
            />
            <Input
              label="最大值"
              type="number"
              value={rangeMax}
              onChange={(e) => setRangeMax(e.target.value)}
              error={errors.rangeMax}
              placeholder="无限制"
            />
          </div>
        );
      case 'enum':
        return (
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-slate-700">
              枚举值 *
            </label>
            <textarea
              value={enumValues}
              onChange={(e) => setEnumValues(e.target.value)}
              className={`w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
                errors.enumValues ? 'border-red-500' : 'border-slate-300'
              }`}
              rows={3}
              placeholder="输入枚举值，用逗号分隔，例如: 男,女,未知"
            />
            {errors.enumValues && (
              <p className="text-sm text-red-500">{errors.enumValues}</p>
            )}
          </div>
        );
      case 'custom_sql':
        return (
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-slate-700">
              自定义 SQL *
            </label>
            <textarea
              value={customSQL}
              onChange={(e) => setCustomSQL(e.target.value)}
              className={`w-full px-3 py-2 border rounded-lg font-mono text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 ${
                errors.customSQL ? 'border-red-500' : 'border-slate-300'
              }`}
              rows={6}
              placeholder="返回失败记录的 SQL，例如：SELECT * FROM users WHERE email IS NULL"
            />
            {errors.customSQL && (
              <p className="text-sm text-red-500">{errors.customSQL}</p>
            )}
          </div>
        );
      case 'referential':
        return (
          <div className="space-y-3">
            <Select
              label="引用对象 *"
              value={refObjectId}
              onChange={(e) => {
                setRefObjectId(e.target.value);
                setRefColumnId('');
              }}
              error={errors.refObjectId}
              options={objectOptions}
              disabled={!formData.source_id}
            />
            <Select
              label="引用字段 *"
              value={refColumnId}
              onChange={(e) => setRefColumnId(e.target.value)}
              error={errors.refColumnId}
              options={refColumnOptions}
              disabled={!refObjectId}
            />
          </div>
        );
      default:
        return (
          <div className="text-sm text-slate-500 bg-slate-50 p-3 rounded-lg">
            此规则类型无需额外配置
          </div>
        );
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={mode === 'create' ? '创建数据质量规则' : '编辑数据质量规则'}
    >
      <form
        onSubmit={handleSubmit}
        className="space-y-4 max-h-[70vh] overflow-y-auto pr-2"
      >
        <Input
          label="规则名称 *"
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
          placeholder="可选，描述此规则的用途"
        />

        <Select
          label="数据源 *"
          value={formData.source_id || ''}
          onChange={(e) => {
            setFormData({
              ...formData,
              source_id: e.target.value || '',
              object_id: '',
              column_id: '',
            });
            setRefObjectId('');
            setRefColumnId('');
          }}
          error={errors.source_id}
          options={[
            {
              value: '',
              label: sourcesLoading ? '加载数据源中...' : '请选择数据源',
            },
            ...sources.map((source) => ({
              value: source.id,
              label: `${source.name} (${source.type})`,
            })),
          ]}
          required
        />

        <div className="grid grid-cols-2 gap-4">
          <Select
            label={needsColumnContext(formData.rule_type) ? '对象 *' : '对象'}
            value={formData.object_id || ''}
            onChange={(e) =>
              setFormData({
                ...formData,
                object_id: e.target.value || '',
                column_id: '',
              })
            }
            error={errors.object_id}
            options={objectOptions}
            disabled={!formData.source_id}
            required={needsColumnContext(formData.rule_type)}
          />
          <Select
            label={needsColumnContext(formData.rule_type) ? '字段 *' : '字段'}
            value={formData.column_id || ''}
            onChange={(e) =>
              setFormData({ ...formData, column_id: e.target.value || '' })
            }
            error={errors.column_id}
            options={columnOptions}
            disabled={!formData.object_id}
            required={needsColumnContext(formData.rule_type)}
          />
        </div>

        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-slate-700">
            规则类型 *
          </label>
          <div className="grid grid-cols-1 gap-2">
            {ruleTypes.map((type) => (
              <button
                key={type.value}
                type="button"
                onClick={() =>
                  setFormData({ ...formData, rule_type: type.value })
                }
                className={`flex items-center p-3 rounded-lg border text-left transition-colors ${
                  formData.rule_type === type.value
                    ? 'border-indigo-500 bg-indigo-50'
                    : 'border-slate-200 hover:border-slate-300 hover:bg-slate-50'
                }`}
              >
                <div className="flex-1">
                  <div className="font-medium text-slate-900">{type.label}</div>
                  <div className="text-xs text-slate-500">
                    {type.description}
                  </div>
                </div>
                {formData.rule_type === type.value && (
                  <div className="w-5 h-5 rounded-full bg-indigo-500 flex items-center justify-center">
                    <svg
                      className="w-3 h-3 text-white"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fillRule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </div>
                )}
              </button>
            ))}
          </div>
        </div>

        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-slate-700">
            规则配置
          </label>
          {renderRuleConfig()}
        </div>

        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-slate-700">
            严重级别 *
          </label>
          <div className="flex flex-wrap gap-2">
            {severities.map((severity) => (
              <button
                key={severity.value}
                type="button"
                onClick={() =>
                  setFormData({ ...formData, severity: severity.value })
                }
                className={`px-4 py-2 rounded-lg text-sm font-medium border transition-colors ${
                  formData.severity === severity.value
                    ? severity.color
                    : 'bg-slate-100 text-slate-600 hover:bg-slate-200 border-transparent'
                }`}
              >
                {severity.label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="is_active"
            checked={formData.is_active}
            onChange={(e) =>
              setFormData({ ...formData, is_active: e.target.checked })
            }
            className="w-4 h-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
          />
          <label htmlFor="is_active" className="text-sm text-slate-700">
            启用此规则
          </label>
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
