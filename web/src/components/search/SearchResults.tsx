import { Link } from 'react-router-dom';
import { Database, Table, Hash, AlertCircle } from 'lucide-react';
import type { ColumnSearchResult } from '../../types';

interface SearchResultsProps {
  results: ColumnSearchResult[];
  query: string;
  loading: boolean;
}

const highlightText = (text: string, query: string) => {
  if (!query.trim()) return text;

  const regex = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
  const parts = text.split(regex);

  return parts.map((part, i) =>
    regex.test(part) ? (
      <mark key={i} className="bg-indigo-100 text-indigo-900 rounded px-0.5">
        {part}
      </mark>
    ) : (
      part
    )
  );
};

const ResultItem: React.FC<{ result: ColumnSearchResult; query: string }> = ({ result, query }) => {
  const typeIcons: Record<string, string> = {
    mysql: '🔷',
    postgres: '🐘',
    mongodb: '🍃',
    oracle: '🔴',
    mssql: '📊',
  };

  return (
    <Link
      to={`/columns/${result.id}`}
      className="block p-4 hover:bg-slate-50 transition-colors border-b border-slate-100 last:border-0"
    >
      <div className="flex items-start gap-4">
        <div className="w-10 h-10 rounded-lg bg-indigo-50 flex items-center justify-center shrink-0">
          <Hash size={20} className="text-indigo-600" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="font-medium text-slate-900 truncate">
              {highlightText(result.name, query)}
            </h3>
            <span className="text-xs font-mono text-slate-500 bg-slate-100 px-2 py-0.5 rounded">
              {result.data_type}
            </span>
          </div>
          <div className="flex items-center gap-3 mt-1 text-sm text-slate-500">
            <span className="flex items-center gap-1">
              <Table size={14} />
              {result.object_name}
            </span>
            <span className="text-slate-300">•</span>
            <span className="flex items-center gap-1">
              <span>{typeIcons[result.source_type] || '🔹'}</span>
              {result.source_name}
            </span>
          </div>
          {result.parent_column_path && (
            <p className="text-xs text-slate-400 mt-1">
              路径: {result.parent_column_path}
            </p>
          )}
        </div>
        <div className="shrink-0">
          <div className="text-right">
            <span className={`text-sm font-medium ${
              result.confidence >= 0.8 ? 'text-emerald-600' : 'text-amber-600'
            }`}>
              {Math.round(result.confidence * 100)}%
            </span>
            <p className="text-xs text-slate-400">匹配度</p>
          </div>
        </div>
      </div>
    </Link>
  );
};

export const SearchResults: React.FC<SearchResultsProps> = ({ results, query, loading }) => {
  if (loading) {
    return (
      <div className="py-12 text-center">
        <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mx-auto" />
        <p className="text-slate-500 mt-4">搜索中...</p>
      </div>
    );
  }

  if (!query.trim()) {
    return (
      <div className="py-12 text-center text-slate-400">
        <Database size={48} className="mx-auto mb-4 opacity-30" />
        <p>输入关键词开始搜索字段</p>
        <p className="text-sm mt-2">支持模糊匹配，例如：user_id, user, id</p>
      </div>
    );
  }

  if (results.length === 0) {
    return (
      <div className="py-12 text-center text-slate-400">
        <AlertCircle size={48} className="mx-auto mb-4 opacity-30" />
        <p>未找到匹配 "{query}" 的字段</p>
        <p className="text-sm mt-2">尝试使用其他关键词或检查拼写</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-100 bg-slate-50/50 flex items-center justify-between">
        <span className="text-sm text-slate-600">
          找到 {results.length} 个结果
        </span>
        <span className="text-xs text-slate-400">
          搜索: "{query}"
        </span>
      </div>
      <div className="divide-y divide-slate-100">
        {results.map((result) => (
          <ResultItem key={result.id} result={result} query={query} />
        ))}
      </div>
    </div>
  );
};
