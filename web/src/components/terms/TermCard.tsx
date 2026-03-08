import { useState } from 'react';
import { BookOpen, MoreVertical, Edit, Trash2, Check } from 'lucide-react';
import { Card, CardContent, Badge } from '../ui';
import type { BusinessTerm } from '../../types';

interface TermCardProps {
  term: BusinessTerm;
  onEdit: (term: BusinessTerm) => void;
  onDelete: (id: string) => void;
}

export const TermCard: React.FC<TermCardProps> = ({ term, onEdit, onDelete }) => {
  const [showActions, setShowActions] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const handleDelete = () => {
    if (confirmDelete) {
      onDelete(term.id);
      setConfirmDelete(false);
    } else {
      setConfirmDelete(true);
      setTimeout(() => setConfirmDelete(false), 3000);
    }
  };

  return (
    <Card className="relative group">
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-4">
            <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-emerald-50 to-teal-50 border border-emerald-100 flex items-center justify-center">
              <BookOpen size={24} className="text-emerald-600" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-slate-900">{term.name}</h3>
              {term.category && (
                <Badge variant="info" className="mt-1">
                  {term.category}
                </Badge>
              )}
              {term.description && (
                <p className="text-sm text-slate-600 mt-2 line-clamp-2">{term.description}</p>
              )}
            </div>
          </div>

          <div className="relative">
            <button
              onClick={() => setShowActions(!showActions)}
              className="p-2 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
            >
              <MoreVertical size={18} />
            </button>

            {showActions && (
              <div className="absolute right-0 top-full mt-1 w-32 bg-white rounded-lg shadow-lg border border-slate-200 py-1 z-10">
                <button
                  onClick={() => {
                    onEdit(term);
                    setShowActions(false);
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-slate-700 hover:bg-slate-50 flex items-center gap-2"
                >
                  <Edit size={16} />
                  编辑
                </button>
                <button
                  onClick={handleDelete}
                  className={`w-full px-4 py-2 text-left text-sm flex items-center gap-2 ${
                    confirmDelete ? 'bg-red-50 text-red-600' : 'text-red-600 hover:bg-red-50'
                  }`}
                >
                  {confirmDelete ? <Check size={16} /> : <Trash2 size={16} />}
                  {confirmDelete ? '确认删除?' : '删除'}
                </button>
              </div>
            )}
          </div>
        </div>

        <div className="mt-4 pt-4 border-t border-slate-100 text-xs text-slate-400">
          <span>更新于 {new Date(term.updated_at).toLocaleDateString()}</span>
        </div>
      </CardContent>
    </Card>
  );
};
