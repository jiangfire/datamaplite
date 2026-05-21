import { useState, useRef, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { GitBranch, Search } from 'lucide-react';
import {
  Layout,
  Card,
  CardContent,
  Button,
  LineageGraph,
  ImpactAnalysis,
} from '../components';
import { useColumnSearch, useLineage, useImpactAnalysis, useColumnDetail } from '../hooks';

export const LineagePage: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const columnId = searchParams.get('column') || '';
  const [searchQuery, setSearchQuery] = useState('');
  const { results, search } = useColumnSearch();
  const { lineage } = useLineage(columnId || undefined);
  const { impact } = useImpactAnalysis(columnId || undefined);
  const { column } = useColumnDetail(columnId || undefined);
  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleSearch = useCallback(
    (query: string) => {
      setSearchQuery(query);
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
      searchTimeoutRef.current = setTimeout(() => {
        search(query, 10);
      }, 300);
    },
    [search],
  );

  const handleSelectColumn = (id: string) => {
    setSearchQuery('');
    setSearchParams({ column: id });
  };

  return (
    <Layout>
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-slate-900 mb-2">
          <GitBranch size={28} className="inline mr-2" />
          血缘分析
        </h1>
        <p className="text-slate-500">追踪数据流转路径，分析字段间的依赖关系</p>
      </div>

      {/* Column Selector */}
      <Card className="mb-6">
        <CardContent className="p-4">
          <div className="relative">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
              size={18}
            />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => handleSearch(e.target.value)}
              placeholder="搜索字段开始分析血缘..."
              className="w-full pl-10 pr-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
            />
          </div>

          {/* Search Results Dropdown */}
          {searchQuery && results.length > 0 && (
            <div className="absolute z-10 w-full max-w-2xl mt-1 bg-white rounded-lg shadow-lg border border-slate-200 max-h-64 overflow-y-auto">
              {results.map((result) => (
                <button
                  key={result.id}
                  onClick={() => handleSelectColumn(result.id)}
                  className="w-full text-left px-4 py-3 hover:bg-slate-50 border-b border-slate-100 last:border-0"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-medium text-slate-900">
                        {result.name}
                      </p>
                      <p className="text-sm text-slate-500">
                        {result.object_name} · {result.source_name}
                      </p>
                    </div>
                    <span className="text-sm text-slate-400">
                      {Math.round(result.confidence * 100)}% 匹配
                    </span>
                  </div>
                </button>
              ))}
            </div>
          )}

          {columnId && (
            <div className="mt-4 p-3 bg-indigo-50 rounded-lg flex items-center justify-between">
              <div>
                <p className="text-sm text-slate-600">当前分析字段</p>
                <p className="font-medium text-indigo-900">{column?.name || columnId}</p>
                {column && (
                  <p className="text-xs text-slate-500">{column.object.name} · {column.source.name}</p>
                )}
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setSearchParams({});
                }}
              >
                清除
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Lineage Graph */}
      {columnId ? (
        lineage ? (
          <LineageGraph lineage={lineage} columnName={column?.name} />
        ) : (
          <Card>
            <CardContent className="py-12 text-center">
              <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto" />
              <p className="text-slate-500 mt-4">加载血缘数据...</p>
            </CardContent>
          </Card>
        )
      ) : (
        <Card>
          <CardContent className="py-16 text-center text-slate-400">
            <GitBranch size={64} className="mx-auto mb-4 opacity-30" />
            <h3 className="text-lg font-medium text-slate-900 mb-2">
              选择字段开始分析
            </h3>
            <p className="text-slate-500">
              在上方搜索框中输入字段名称，查看其血缘关系
            </p>
          </CardContent>
        </Card>
      )}

      {/* Impact Analysis */}
      {columnId && impact && impact.total_objects > 0 && (
        <div className="mt-6">
          <h2 className="text-lg font-semibold text-slate-900 mb-4">
            影响分析
          </h2>
          <ImpactAnalysis impact={impact} />
        </div>
      )}
    </Layout>
  );
};
