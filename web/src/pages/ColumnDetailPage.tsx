import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, GitBranch, Zap, FileCode } from 'lucide-react';
import {
  Layout,
  Card,
  CardContent,
  Button,
  ColumnDetailCard,
  LineageGraph,
  ImpactAnalysis,
  Modal,
} from '../components';
import { useColumnDetail, useLineage, useImpactAnalysis, useDDLGeneration } from '../hooks';
import { useState } from 'react';

export const ColumnDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { column, loading: columnLoading, error: columnError } = useColumnDetail(id);
  const { lineage } = useLineage(id);
  const { impact } = useImpactAnalysis(id);
  const { generateDDL, generating } = useDDLGeneration();
  const [showDDL, setShowDDL] = useState(false);
  const [ddlContent, setDdlContent] = useState('');
  const [ddlTarget, setDdlTarget] = useState<'mysql' | 'postgres'>('mysql');

  const handleGenerateDDL = async () => {
    if (!column) return;
    const result = await generateDDL(column.object.id, ddlTarget);
    setDdlContent(result.sql);
    setShowDDL(true);
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
            onChange={(e) => setDdlTarget(e.target.value as 'mysql' | 'postgres')}
            className="px-3 py-2 rounded-lg border border-slate-300 text-sm"
          >
            <option value="mysql">MySQL</option>
            <option value="postgres">PostgreSQL</option>
          </select>
          <Button variant="secondary" onClick={handleGenerateDDL} loading={generating}>
            <FileCode size={18} className="mr-2" />
            生成DDL
          </Button>
        </div>
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
      <Modal isOpen={showDDL} onClose={() => setShowDDL(false)} title={`DDL (${ddlTarget})`}>
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
