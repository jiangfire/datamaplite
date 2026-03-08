import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowUp, ArrowDown, Database, Table, Hash, ArrowRight } from 'lucide-react';
import { Card, CardHeader, CardTitle, CardContent } from '../ui';
import type { LineageResponse, LineageEdge } from '../../types';

interface LineageGraphProps {
  lineage: LineageResponse;
}

interface LineageNodeProps {
  node: {
    id: string;
    name: string;
    type: 'column' | 'object';
    data_type?: string;
    source: string;
  };
  direction: 'up' | 'down';
  isCurrent?: boolean;
}

const LineageNode: React.FC<LineageNodeProps> = ({ node, isCurrent }) => {
  const icons = {
    column: <Hash size={16} />,
    object: <Table size={16} />,
  };

  return (
    <div
      className={`flex items-center gap-3 p-3 rounded-lg border ${
        isCurrent
          ? 'bg-indigo-50 border-indigo-300 shadow-sm'
          : 'bg-white border-slate-200 hover:border-indigo-200'
      }`}
    >
      <div
        className={`w-8 h-8 rounded-lg flex items-center justify-center ${
          isCurrent ? 'bg-indigo-100 text-indigo-600' : 'bg-slate-100 text-slate-500'
        }`}
      >
        {icons[node.type]}
      </div>
      <div className="flex-1 min-w-0">
        <Link
          to={node.type === 'column' ? `/columns/${node.id}` : `#`}
          className={`font-medium text-sm truncate block ${
            isCurrent ? 'text-indigo-900' : 'text-slate-900 hover:text-indigo-600'
          }`}
        >
          {node.name}
        </Link>
        <p className="text-xs text-slate-400 truncate">{node.source}</p>
      </div>
      {node.data_type && (
        <span className="text-xs font-mono text-slate-400 bg-slate-50 px-2 py-0.5 rounded">
          {node.data_type}
        </span>
      )}
    </div>
  );
};

const EdgeConnector: React.FC<{ edge: LineageEdge; direction: 'up' | 'down' }> = ({
  edge,
  direction,
}) => {
  const [showDetails, setShowDetails] = useState(false);

  return (
    <div className="flex flex-col items-center py-2">
      <div
        className="relative flex items-center justify-center w-8 h-8 rounded-full bg-slate-100 text-slate-400 cursor-pointer hover:bg-indigo-50 hover:text-indigo-500 transition-colors"
        onClick={() => setShowDetails(!showDetails)}
      >
        {direction === 'up' ? <ArrowUp size={16} /> : <ArrowDown size={16} />}
      </div>

      {showDetails && (edge.transform_sql || edge.job_name) && (
        <div className="mt-2 p-3 bg-slate-50 rounded-lg text-xs text-slate-600 max-w-xs">
          {edge.job_name && (
            <p className="font-medium mb-1">Job: {edge.job_name}</p>
          )}
          {edge.transform_sql && (
            <pre className="bg-slate-800 text-slate-200 p-2 rounded overflow-x-auto">
              {edge.transform_sql}
            </pre>
          )}
        </div>
      )}

      {!showDetails && (edge.transform_sql || edge.job_name) && (
        <div className="w-1.5 h-1.5 rounded-full bg-indigo-400 mt-1" />
      )}
    </div>
  );
};

export const LineageGraph: React.FC<LineageGraphProps> = ({ lineage }) => {
  const { upward, downward, column_id } = lineage;

  return (
    <div className="space-y-6">
      {/* Upstream */}
      {upward.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <ArrowUp size={18} className="text-emerald-500" />
              上游血缘 ({upward.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {upward.map((edge, index) => (
                <div key={`up-${index}`}>
                  <LineageNode node={edge.source} direction="up" />
                  {index < upward.length - 1 && (
                    <EdgeConnector edge={edge} direction="up" />
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Current Column */}
      <Card className="border-indigo-300 shadow-lg shadow-indigo-100">
        <CardContent className="p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center text-white">
              <Hash size={20} />
            </div>
            <div>
              <p className="text-sm text-slate-500">当前字段</p>
              <p className="text-lg font-semibold text-slate-900">{column_id}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Downstream */}
      {downward.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <ArrowDown size={18} className="text-amber-500" />
              下游血缘 ({downward.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {downward.map((edge, index) => (
                <div key={`down-${index}`}>
                  <LineageNode node={edge.target} direction="down" />
                  {index < downward.length - 1 && (
                    <EdgeConnector edge={edge} direction="down" />
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {upward.length === 0 && downward.length === 0 && (
        <div className="py-12 text-center text-slate-400">
          <Database size={48} className="mx-auto mb-4 opacity-30" />
          <p>暂无血缘关系数据</p>
          <p className="text-sm mt-2">该字段尚未建立任何血缘连接</p>
        </div>
      )}
    </div>
  );
};

export const ImpactAnalysis: React.FC<{
  impact: {
    column_id: string;
    impact_objects: Array<{
      id: string;
      name: string;
      type: string;
      object_name?: string;
      source_name: string;
      impact_path: string;
      distance: number;
    }>;
    total_objects: number;
  };
}> = ({ impact }) => {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ArrowRight size={18} className="text-red-500" />
          影响分析
          <span className="text-sm font-normal text-slate-500">
            ({impact.total_objects} 个对象)
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {impact.impact_objects.map((obj) => (
            <div
              key={obj.id}
              className="flex items-center gap-3 p-3 bg-slate-50 rounded-lg"
            >
              <div className="w-6 h-6 rounded-full bg-indigo-100 text-indigo-600 flex items-center justify-center text-xs font-bold">
                {obj.distance}
              </div>
              <div className="flex-1">
                <p className="font-medium text-sm text-slate-900">{obj.name}</p>
                <p className="text-xs text-slate-500">
                  {obj.type === 'column' ? `字段 · ${obj.object_name}` : '表'}
                  {' · '}
                  {obj.source_name}
                </p>
              </div>
              <div className="text-xs text-slate-400 font-mono">
                {obj.impact_path}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
};
