import { Link } from 'react-router-dom';
import {
  Database,
  Table,
  ArrowRight,
  GitBranch,
  AlertTriangle,
} from 'lucide-react';
import { Card, CardHeader, CardTitle, CardContent, Badge } from '../ui';
import type {
  ColumnDetail as ColumnDetailType,
  MappedColumn,
} from '../../types';

interface ColumnDetailProps {
  column: ColumnDetailType;
}

const MappingItem: React.FC<{ mapping: MappedColumn }> = ({ mapping }) => {
  const mappingTypeLabels: Record<string, string> = {
    alias: '别名',
    transform: '转换',
    derived: '派生',
    synonym: '同义词',
  };

  return (
    <div className="flex items-center gap-3 py-2 border-b border-slate-100 last:border-0">
      <Badge variant="info">
        {mappingTypeLabels[mapping.mapping_type] || mapping.mapping_type}
      </Badge>
      <span className="text-sm font-medium text-slate-900">{mapping.name}</span>
      <span className="text-xs text-slate-400">{mapping.object_name}</span>
      <ArrowRight size={14} className="text-slate-300" />
      <span className="text-xs text-slate-500">{mapping.source_name}</span>
      <div className="ml-auto">
        <div className="w-16 h-1.5 bg-slate-100 rounded-full overflow-hidden">
          <div
            className="h-full bg-indigo-500 rounded-full"
            style={{ width: `${mapping.confidence * 100}%` }}
          />
        </div>
      </div>
    </div>
  );
};

export const ColumnDetailCard: React.FC<ColumnDetailProps> = ({ column }) => {
  return (
    <div className="space-y-6">
      {/* Header Info */}
      <div className="bg-gradient-to-r from-indigo-50 to-violet-50 rounded-xl p-6 border border-indigo-100">
        <div className="flex items-start justify-between">
          <div>
            <h2 className="text-2xl font-bold text-slate-900">{column.name}</h2>
            <div className="flex items-center gap-4 mt-2 text-sm text-slate-600">
              <Link
                to={`/sources/${column.source.id}`}
                className="flex items-center gap-1 hover:text-indigo-600"
              >
                <Database size={14} />
                {column.source.name}
              </Link>
              <span className="text-slate-300">/</span>
              <Link
                to={`/sources/${column.source.id}#${column.object.id}`}
                className="flex items-center gap-1 hover:text-indigo-600"
              >
                <Table size={14} />
                {column.object.name}
              </Link>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {column.is_primary_key && <Badge variant="warning">主键</Badge>}
            {!column.is_nullable && <Badge variant="error">NOT NULL</Badge>}
          </div>
        </div>

        {column.description && (
          <p className="mt-4 text-slate-700">{column.description}</p>
        )}
      </div>

      {/* Properties Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              数据类型
            </p>
            <p className="text-lg font-semibold text-slate-900 mt-1">
              {column.data_type}
            </p>
            <p className="text-xs text-slate-400 font-mono mt-1">
              {column.full_data_type}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              位置
            </p>
            <p className="text-lg font-semibold text-slate-900 mt-1">
              #{column.ordinal_position}
            </p>
            <p className="text-xs text-slate-400 mt-1">字段顺序</p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              置信度
            </p>
            <p className="text-lg font-semibold text-slate-900 mt-1">
              {Math.round(column.confidence * 100)}%
            </p>
            <div className="w-full h-1.5 bg-slate-100 rounded-full overflow-hidden mt-2">
              <div
                className={`h-full rounded-full ${
                  column.confidence >= 0.8 ? 'bg-emerald-500' : 'bg-amber-500'
                }`}
                style={{ width: `${column.confidence * 100}%` }}
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              业务术语
            </p>
            {column.term ? (
              <Link
                to={`/terms/${column.term.id}`}
                className="text-lg font-semibold text-indigo-600 hover:text-indigo-700 mt-1 block"
              >
                {column.term.name}
              </Link>
            ) : (
              <div className="flex items-center gap-2 mt-1">
                <AlertTriangle size={14} className="text-amber-500" />
                <span className="text-sm text-slate-500">未分配术语</span>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Default Value */}
      {column.default_value && (
        <Card>
          <CardContent className="p-4">
            <p className="text-xs text-slate-500 uppercase tracking-wider">
              默认值
            </p>
            <code className="mt-2 block bg-slate-50 px-3 py-2 rounded text-sm font-mono text-slate-700">
              {column.default_value}
            </code>
          </CardContent>
        </Card>
      )}

      {/* Mappings */}
      {column.mapped_columns && column.mapped_columns.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GitBranch size={18} />
              字段映射 ({column.mapped_columns.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            {column.mapped_columns.map((mapping) => (
              <MappingItem key={mapping.id} mapping={mapping} />
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
};
