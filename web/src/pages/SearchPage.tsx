import { useState, useCallback } from 'react';
import { Search } from 'lucide-react';
import { Layout, SearchBar, SearchResults } from '../components';
import { useColumnSearch } from '../hooks';

export const SearchPage: React.FC = () => {
  const [query, setQuery] = useState('');
  const { results, loading, search } = useColumnSearch();

  const handleSearch = useCallback(
    (value: string) => {
      search(value);
    },
    [search],
  );

  return (
    <Layout>
      {/* Header */}
      <div className="text-center mb-8">
        <h1 className="text-3xl font-bold text-slate-900 mb-2">字段搜索</h1>
        <p className="text-slate-500">
          跨所有数据源搜索字段，快速找到"同义不同名"的字段
        </p>
      </div>

      {/* Search Box */}
      <div className="max-w-2xl mx-auto mb-8">
        <SearchBar
          value={query}
          onChange={setQuery}
          onSubmit={handleSearch}
          placeholder="搜索字段名称，例如：user_id, plate_no..."
          loading={loading}
          autoFocus
        />
      </div>

      {/* Results */}
      <SearchResults results={results} query={query} loading={loading} />

      {/* Tips */}
      {!query && (
        <div className="mt-12 grid grid-cols-3 gap-6 max-w-3xl mx-auto">
          <div className="text-center p-4">
            <div className="w-12 h-12 rounded-xl bg-indigo-50 flex items-center justify-center mx-auto mb-3">
              <Search size={24} className="text-indigo-500" />
            </div>
            <h3 className="font-medium text-slate-900 mb-1">模糊匹配</h3>
            <p className="text-sm text-slate-500">
              支持字段名模糊搜索，不区分大小写
            </p>
          </div>
          <div className="text-center p-4">
            <div className="w-12 h-12 rounded-xl bg-emerald-50 flex items-center justify-center mx-auto mb-3">
              <span className="text-2xl">🔍</span>
            </div>
            <h3 className="font-medium text-slate-900 mb-1">跨源搜索</h3>
            <p className="text-sm text-slate-500">
              一次搜索覆盖所有已连接的数据源
            </p>
          </div>
          <div className="text-center p-4">
            <div className="w-12 h-12 rounded-xl bg-violet-50 flex items-center justify-center mx-auto mb-3">
              <span className="text-2xl">📊</span>
            </div>
            <h3 className="font-medium text-slate-900 mb-1">置信度评分</h3>
            <p className="text-sm text-slate-500">根据匹配程度显示置信度分数</p>
          </div>
        </div>
      )}
    </Layout>
  );
};
