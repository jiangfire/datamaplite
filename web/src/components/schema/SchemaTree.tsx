import { useState } from 'react';
import { ChevronRight, ChevronDown, Table, Key, Hash } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { SchemaObjectWithColumns, Column } from '../../types';

interface SchemaTreeProps {
  objects: SchemaObjectWithColumns[];
}

interface ObjectNodeProps {
  object: SchemaObjectWithColumns;
  expanded: boolean;
  onToggle: () => void;
}

const typeIcons: Record<string, string> = {
  table: '📋',
  view: '👁️',
  collection: '🍃',
};

const ColumnItem: React.FC<{ column: Column }> = ({ column }) => {
  return (
    <div className="flex items-center gap-2 py-1.5 px-3 text-sm hover:bg-slate-50 rounded-lg group">
      <span className="text-slate-400">
        {column.is_primary_key ? (
          <Key size={14} className="text-amber-500" />
        ) : (
          <Hash size={14} />
        )}
      </span>
      <Link
        to={`/columns/${column.id}`}
        className="flex-1 text-slate-700 hover:text-indigo-600 transition-colors"
      >
        {column.name}
      </Link>
      <span className="text-xs text-slate-400 font-mono">
        {column.data_type}
      </span>
      {!column.is_nullable && (
        <span className="text-xs text-slate-300">NOT NULL</span>
      )}
    </div>
  );
};

const ObjectNode: React.FC<ObjectNodeProps> = ({
  object,
  expanded,
  onToggle,
}) => {
  return (
    <div className="border-b border-slate-100 last:border-0">
      <button
        onClick={onToggle}
        className="w-full flex items-center gap-2 py-2.5 px-3 text-left hover:bg-slate-50 transition-colors"
      >
        {expanded ? (
          <ChevronDown size={16} className="text-slate-400" />
        ) : (
          <ChevronRight size={16} className="text-slate-400" />
        )}
        <span className="text-lg">{typeIcons[object.type] || '📋'}</span>
        <span className="font-medium text-slate-900">{object.name}</span>
        {object.schema && (
          <span className="text-xs text-slate-400">({object.schema})</span>
        )}
        <span className="ml-auto text-xs text-slate-400">
          {object.column_count} 字段
        </span>
      </button>

      {expanded && (
        <div className="ml-9 border-l border-slate-200 pl-2 pb-2">
          {object.columns.map((column) => (
            <ColumnItem key={column.id} column={column} />
          ))}
        </div>
      )}
    </div>
  );
};

export const SchemaTree: React.FC<SchemaTreeProps> = ({ objects }) => {
  const [expandedObjects, setExpandedObjects] = useState<Set<string>>(
    new Set(),
  );

  const toggleObject = (id: string) => {
    setExpandedObjects((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const expandAll = () => {
    setExpandedObjects(new Set(objects.map((o) => o.id)));
  };

  const collapseAll = () => {
    setExpandedObjects(new Set());
  };

  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-100 bg-slate-50/50">
        <div className="flex items-center gap-2 text-sm text-slate-600">
          <Table size={16} />
          <span>{objects.length} 个对象</span>
        </div>
        <div className="flex gap-2">
          <button
            onClick={expandAll}
            className="text-xs text-indigo-600 hover:text-indigo-700 font-medium"
          >
            展开全部
          </button>
          <span className="text-slate-300">|</span>
          <button
            onClick={collapseAll}
            className="text-xs text-indigo-600 hover:text-indigo-700 font-medium"
          >
            收起全部
          </button>
        </div>
      </div>

      <div className="divide-y divide-slate-100">
        {objects.map((object) => (
          <ObjectNode
            key={object.id}
            object={object}
            expanded={expandedObjects.has(object.id)}
            onToggle={() => toggleObject(object.id)}
          />
        ))}
      </div>

      {objects.length === 0 && (
        <div className="py-12 text-center text-slate-400">
          <Table size={48} className="mx-auto mb-4 opacity-30" />
          <p>暂无Schema对象</p>
          <p className="text-sm mt-1">点击"同步元数据"按钮开始采集</p>
        </div>
      )}
    </div>
  );
};
