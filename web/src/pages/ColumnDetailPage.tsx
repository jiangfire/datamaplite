import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  BookOpen,
  FileCode,
  GitBranch,
  Link2,
  Search,
  Trash2,
  Zap,
} from 'lucide-react';
import {
  Layout,
  Card,
  CardContent,
  Button,
  Badge,
  ColumnDetailCard,
  LineageGraph,
  ImpactAnalysis,
  Modal,
} from '../components';
import {
  useColumnDetail,
  useLineage,
  useImpactAnalysis,
  useDDLGeneration,
  useColumnTags,
  useTags,
  useTerms,
  useColumnMappings,
  useColumnSearch,
} from '../hooks';
import { columnService } from '../services';
import type { ColumnSearchResult } from '../types';

const mappingTypeLabels: Record<string, string> = {
  alias: '别名',
  transform: '转换',
  derived: '派生',
  synonym: '同义词',
};

export const ColumnDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const {
    column,
    loading: columnLoading,
    error: columnError,
    refetch: refetchColumn,
  } = useColumnDetail(id);
  const { lineage } = useLineage(id);
  const { impact } = useImpactAnalysis(id);
  const { generateDDL, generating } = useDDLGeneration();
  const { tags, addTag, removeTag, loading: tagsLoading } = useColumnTags(
    id || null,
  );
  const { tags: allTags } = useTags();
  const {
    terms,
    loading: termsLoading,
    error: termsError,
  } = useTerms();
  const {
    mappings,
    loading: mappingsLoading,
    error: mappingsError,
    createMapping,
    deleteMapping,
  } = useColumnMappings(id);
  const {
    results: targetResults,
    loading: targetSearchLoading,
    error: targetSearchError,
    search: searchColumns,
  } = useColumnSearch();
  const [showDDL, setShowDDL] = useState(false);
  const [ddlContent, setDdlContent] = useState('');
  const [ddlTarget, setDdlTarget] = useState<'mysql' | 'postgres'>('mysql');
  const [selectedTagId, setSelectedTagId] = useState('');
  const [selectedTermId, setSelectedTermId] = useState('');
  const [termSaving, setTermSaving] = useState(false);
  const [termError, setTermError] = useState<string | null>(null);
  const [mappingQuery, setMappingQuery] = useState('');
  const [selectedTarget, setSelectedTarget] =
    useState<ColumnSearchResult | null>(null);
  const [mappingType, setMappingType] = useState<
    'alias' | 'transform' | 'derived' | 'synonym'
  >('alias');
  const [mappingConfidence, setMappingConfidence] = useState('1');
  const [mappingSaving, setMappingSaving] = useState(false);
  const [mappingError, setMappingError] = useState<string | null>(null);
  const [deletingMappingId, setDeletingMappingId] = useState<string | null>(
    null,
  );

  useEffect(() => {
    setSelectedTermId(column?.term?.id ?? '');
  }, [column?.term?.id]);

  const handleGenerateDDL = async () => {
    if (!column) return;
    const result = await generateDDL(column.object.id, ddlTarget);
    setDdlContent(result.sql);
    setShowDDL(true);
  };

  const handleAssignTerm = async (termId: string | null) => {
    if (!id) return;

    setTermSaving(true);
    setTermError(null);
    try {
      await columnService.assignTerm(id, { term_id: termId });
      await refetchColumn();
      setSelectedTermId(termId ?? '');
    } catch (err) {
      setTermError(err instanceof Error ? err.message : '术语保存失败');
    } finally {
      setTermSaving(false);
    }
  };

  const handleSearchTarget = (value: string) => {
    setMappingQuery(value);
    setSelectedTarget(null);
    setMappingError(null);
    void searchColumns(value, 8);
  };

  const handleSelectTarget = (target: ColumnSearchResult) => {
    setSelectedTarget(target);
    setMappingQuery(`${target.name} · ${target.object_name} · ${target.source_name}`);
    void searchColumns('', 8);
  };

  const handleCreateMapping = async () => {
    if (!id) return;
    if (!selectedTarget) {
      setMappingError('请选择目标字段');
      return;
    }

    const confidence =
      mappingConfidence.trim() === '' ? undefined : Number(mappingConfidence);
    if (
      confidence !== undefined &&
      (Number.isNaN(confidence) || confidence < 0 || confidence > 1)
    ) {
      setMappingError('置信度必须在 0 到 1 之间');
      return;
    }

    setMappingSaving(true);
    setMappingError(null);
    try {
      await createMapping({
        source_column_id: id,
        target_column_id: selectedTarget.id,
        mapping_type: mappingType,
        confidence,
      });
      await refetchColumn();
      setSelectedTarget(null);
      setMappingQuery('');
      setMappingType('alias');
      setMappingConfidence('1');
      void searchColumns('', 8);
    } catch (err) {
      setMappingError(err instanceof Error ? err.message : '映射创建失败');
    } finally {
      setMappingSaving(false);
    }
  };

  const handleDeleteMapping = async (mappingId: string) => {
    setDeletingMappingId(mappingId);
    setMappingError(null);
    try {
      await deleteMapping(mappingId);
      await refetchColumn();
    } catch (err) {
      setMappingError(err instanceof Error ? err.message : '映射删除失败');
    } finally {
      setDeletingMappingId(null);
    }
  };

  if (columnLoading) {
    return (
      <Layout>
        <div className="py-12 text-center">
          <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto" />
          <p className="text-slate-500 mt-4">加载中...</p>
        </div>
      </Layout>
    );
  }

  if (columnError || !column) {
    return (
      <Layout>
        <Card>
          <CardContent className="py-12 text-center text-red-500">
            <p>加载失败: {columnError || '字段不存在'}</p>
            <Link to="/search">
              <Button variant="secondary" className="mt-4">
                返回搜索
              </Button>
            </Link>
          </CardContent>
        </Card>
      </Layout>
    );
  }

  const existingTargetIDs = new Set(mappings.map((mapping) => mapping.target_column_id));
  const availableTargetResults = column
    ? targetResults.filter(
        (result) => result.id !== column.id && !existingTargetIDs.has(result.id),
      )
    : [];

  return (
    <Layout>
      {/* Breadcrumb */}
      <div className="mb-6">
        <Link
          to={`/sources/${column.source.id}`}
          className="inline-flex items-center text-sm text-slate-500 hover:text-indigo-600"
        >
          <ArrowLeft size={16} className="mr-1" />
          返回 {column.object.name}
        </Link>
      </div>

      {/* Main Content */}
      <ColumnDetailCard column={column} />

      {/* Actions */}
      <div className="flex gap-3 my-6">
        <Link to={`/lineage?column=${column.id}`}>
          <Button variant="secondary">
            <GitBranch size={18} className="mr-2" />
            查看血缘
          </Button>
        </Link>
        <div className="flex gap-2">
          <select
            value={ddlTarget}
            onChange={(e) =>
              setDdlTarget(e.target.value as 'mysql' | 'postgres')
            }
            aria-label="DDL目标数据库"
            className="px-3 py-2 rounded-lg border border-slate-300 text-sm"
          >
            <option value="mysql">MySQL</option>
            <option value="postgres">PostgreSQL</option>
          </select>
          <Button
            variant="secondary"
            onClick={handleGenerateDDL}
            loading={generating}
          >
            <FileCode size={18} className="mr-2" />
            生成DDL
          </Button>
        </div>
      </div>

      <Card>
        <CardContent className="p-4 space-y-4">
          <div className="flex items-center justify-between gap-4">
            <div>
              <h2 className="text-lg font-semibold text-slate-900">字段标签</h2>
              <p className="text-sm text-slate-500">
                为字段补充分类与治理标记
              </p>
            </div>
            <div className="flex gap-2">
              <select
                value={selectedTagId}
                onChange={(e) => setSelectedTagId(e.target.value)}
                aria-label="字段标签选择"
                className="px-3 py-2 rounded-lg border border-slate-300 text-sm min-w-52"
              >
                <option value="">选择标签</option>
                {allTags
                  .filter((tag) => !tags.some((current) => current.id === tag.id))
                  .map((tag) => (
                    <option key={tag.id} value={tag.id}>
                      {tag.name}
                    </option>
                  ))}
              </select>
              <Button
                variant="secondary"
                disabled={!selectedTagId}
                onClick={async () => {
                  await addTag(selectedTagId);
                  setSelectedTagId('');
                }}
              >
                添加标签
              </Button>
            </div>
          </div>

          {tagsLoading ? (
            <p className="text-sm text-slate-500">加载标签中...</p>
          ) : tags.length === 0 ? (
            <p className="text-sm text-slate-500">当前字段还没有标签</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {tags.map((tag) => (
                <div
                  key={tag.id}
                  className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3 py-1.5"
                >
                  <span
                    className="h-2.5 w-2.5 rounded-full"
                    style={{ backgroundColor: tag.color }}
                  />
                  <Badge variant="neutral">{tag.name}</Badge>
                  <button
                    onClick={() => removeTag(tag.id)}
                    className="text-xs text-slate-400 hover:text-red-600"
                    title="移除标签"
                  >
                    删除
                  </button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-6 mt-6 xl:grid-cols-[1.1fr,1.4fr]">
        <Card>
          <CardContent className="p-5 space-y-4">
            <div className="flex items-start gap-3">
              <div className="mt-0.5 rounded-lg bg-indigo-50 p-2 text-indigo-600">
                <BookOpen size={18} />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-slate-900">
                  术语绑定
                </h2>
                <p className="text-sm text-slate-500">
                  将字段映射到标准业务术语，便于统一口径治理
                </p>
              </div>
            </div>

            <div className="space-y-3">
              <select
                value={selectedTermId}
                onChange={(e) => setSelectedTermId(e.target.value)}
                aria-label="字段术语选择"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
                disabled={termsLoading || termSaving}
              >
                <option value="">未分配术语</option>
                {terms.map((term) => (
                  <option key={term.id} value={term.id}>
                    {term.name}
                    {term.category ? ` · ${term.category}` : ''}
                  </option>
                ))}
              </select>

              <div className="flex flex-wrap gap-2">
                <Button
                  variant="secondary"
                  loading={termSaving}
                  disabled={
                    termsLoading ||
                    termSaving ||
                    selectedTermId === (column.term?.id ?? '')
                  }
                  onClick={() => handleAssignTerm(selectedTermId || null)}
                >
                  保存术语绑定
                </Button>
                <Button
                  variant="ghost"
                  disabled={!column.term || termSaving}
                  onClick={() => handleAssignTerm(null)}
                >
                  解除绑定
                </Button>
              </div>
            </div>

            {column.term ? (
              <p className="text-sm text-slate-600">
                当前术语：
                <Link
                  to={`/terms/${column.term.id}`}
                  className="ml-1 font-medium text-indigo-600 hover:text-indigo-700"
                >
                  {column.term.name}
                </Link>
              </p>
            ) : (
              <p className="text-sm text-slate-500">当前字段尚未绑定业务术语</p>
            )}

            {(termsError || termError) && (
              <p className="text-sm text-red-500">{termsError || termError}</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5 space-y-4">
            <div className="flex items-start gap-3">
              <div className="mt-0.5 rounded-lg bg-emerald-50 p-2 text-emerald-600">
                <Link2 size={18} />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-slate-900">
                  字段映射治理
                </h2>
                <p className="text-sm text-slate-500">
                  维护目标字段映射，补齐同义、派生和转换关系
                </p>
              </div>
            </div>

            <div className="grid gap-3 lg:grid-cols-[1.6fr,0.8fr,0.6fr,auto]">
              <div className="relative">
                <Search
                  size={16}
                  className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                />
                <input
                  value={mappingQuery}
                  onChange={(e) => handleSearchTarget(e.target.value)}
                  placeholder="搜索目标字段名称"
                  aria-label="目标字段搜索"
                  className="w-full rounded-lg border border-slate-300 py-2 pl-9 pr-3 text-sm"
                />

                {mappingQuery && !selectedTarget && availableTargetResults.length > 0 && (
                  <div className="absolute z-10 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-slate-200 bg-white shadow-lg">
                    {availableTargetResults.map((result) => (
                      <button
                        key={result.id}
                        type="button"
                        onClick={() => handleSelectTarget(result)}
                        className="w-full border-b border-slate-100 px-4 py-3 text-left last:border-0 hover:bg-slate-50"
                      >
                        <p className="font-medium text-slate-900">
                          {result.name}
                        </p>
                        <p className="text-xs text-slate-500">
                          {result.object_name} · {result.source_name}
                        </p>
                      </button>
                    ))}
                  </div>
                )}
              </div>

              <select
                value={mappingType}
                onChange={(e) =>
                  setMappingType(
                    e.target.value as 'alias' | 'transform' | 'derived' | 'synonym',
                  )
                }
                aria-label="映射类型选择"
                className="rounded-lg border border-slate-300 px-3 py-2 text-sm"
              >
                {Object.entries(mappingTypeLabels).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>

              <input
                value={mappingConfidence}
                onChange={(e) => setMappingConfidence(e.target.value)}
                type="number"
                min="0"
                max="1"
                step="0.1"
                aria-label="映射置信度"
                className="rounded-lg border border-slate-300 px-3 py-2 text-sm"
                placeholder="1.0"
              />

              <Button
                variant="secondary"
                loading={mappingSaving}
                disabled={!selectedTarget || mappingSaving}
                onClick={handleCreateMapping}
              >
                新增映射
              </Button>
            </div>

            {selectedTarget && (
              <div className="rounded-lg border border-emerald-200 bg-emerald-50/70 px-4 py-3 text-sm text-slate-700">
                已选择目标字段：
                <span className="ml-1 font-medium text-slate-900">
                  {selectedTarget.name}
                </span>
                <span className="ml-2 text-slate-500">
                  {selectedTarget.object_name} · {selectedTarget.source_name}
                </span>
              </div>
            )}

            {targetSearchLoading && (
              <p className="text-sm text-slate-500">正在搜索目标字段...</p>
            )}

            {(mappingError || mappingsError || targetSearchError) && (
              <p className="text-sm text-red-500">
                {mappingError || mappingsError || targetSearchError}
              </p>
            )}

            <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50/80 p-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-slate-900">
                  已维护映射
                </h3>
                <span className="text-xs text-slate-500">
                  {mappings.length} 条
                </span>
              </div>

              {mappingsLoading ? (
                <p className="text-sm text-slate-500">加载映射中...</p>
              ) : mappings.length === 0 ? (
                <p className="text-sm text-slate-500">当前字段还没有映射关系</p>
              ) : (
                mappings.map((mapping) => (
                  <div
                    key={mapping.id}
                    className="flex flex-wrap items-center gap-3 rounded-lg border border-slate-200 bg-white px-4 py-3"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="rounded-full bg-indigo-50 px-2.5 py-1 text-xs font-medium text-indigo-700">
                          {mappingTypeLabels[mapping.mapping_type] ||
                            mapping.mapping_type}
                        </span>
                        <Link
                          to={`/columns/${mapping.target_column.id}`}
                          className="truncate text-sm font-medium text-slate-900 hover:text-indigo-600"
                        >
                          {mapping.target_column.name}
                        </Link>
                      </div>
                      <p className="mt-1 text-xs text-slate-500">
                        {mapping.target_column.object_name} ·{' '}
                        {mapping.target_column.source_name}
                      </p>
                    </div>

                    <div className="text-right text-xs text-slate-500">
                      <p>{Math.round(mapping.confidence * 100)}% 置信度</p>
                      <p>{mapping.created_at}</p>
                    </div>

                    <Button
                      variant="ghost"
                      size="sm"
                      loading={deletingMappingId === mapping.id}
                      onClick={() => handleDeleteMapping(mapping.id)}
                    >
                      <Trash2 size={14} className="mr-1" />
                      删除
                    </Button>
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Lineage Preview */}
      {lineage && (
        <div className="mt-8">
          <h2 className="text-lg font-semibold text-slate-900 mb-4">
            <GitBranch size={20} className="inline mr-2" />
            血缘关系预览
          </h2>
          <LineageGraph lineage={lineage} />
        </div>
      )}

      {/* Impact Preview */}
      {impact && impact.total_objects > 0 && (
        <div className="mt-8">
          <h2 className="text-lg font-semibold text-slate-900 mb-4">
            <Zap size={20} className="inline mr-2" />
            影响分析
          </h2>
          <ImpactAnalysis impact={impact} />
        </div>
      )}

      {/* DDL Modal */}
      <Modal
        isOpen={showDDL}
        onClose={() => setShowDDL(false)}
        title={`DDL (${ddlTarget})`}
      >
        <div className="space-y-4">
          <pre className="bg-slate-900 text-slate-100 p-4 rounded-lg overflow-x-auto text-sm font-mono">
            {ddlContent}
          </pre>
          <div className="flex justify-end">
            <Button
              variant="secondary"
              onClick={() => navigator.clipboard.writeText(ddlContent)}
            >
              复制到剪贴板
            </Button>
          </div>
        </div>
      </Modal>
    </Layout>
  );
};
