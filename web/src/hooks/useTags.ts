import { useState, useEffect, useCallback, useReducer } from 'react';
import { tagService } from '../services';
import { useToastContext } from './useToastContext';
import type { Tag, TagCreate, ColumnSearchResult } from '../types';

export const useTags = () => {
  const { toast } = useToastContext();
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTags = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await tagService.listTags();
      setTags(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch tags');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTags();
  }, [fetchTags]);

  const createTag = async (data: TagCreate) => {
    try {
      const newTag = await tagService.createTag(data);
      setTags((prev) => [...prev, newTag]);
      toast('标签创建成功', 'success');
      return newTag;
    } catch (err) {
      toast(err instanceof Error ? err.message : '创建标签失败', 'error');
      throw err;
    }
  };

  const updateTag = async (id: string, data: TagCreate) => {
    try {
      await tagService.updateTag(id, data);
      await fetchTags();
      toast('标签更新成功', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : '更新标签失败', 'error');
      throw err;
    }
  };

  const deleteTag = async (id: string) => {
    try {
      await tagService.deleteTag(id);
      setTags((prev) => prev.filter((t) => t.id !== id));
      toast('标签已删除', 'success');
    } catch (err) {
      toast(err instanceof Error ? err.message : '删除标签失败', 'error');
      throw err;
    }
  };

  return {
    tags,
    loading,
    error,
    refetch: fetchTags,
    createTag,
    updateTag,
    deleteTag,
  };
};

export const useColumnTags = (columnId: string | null) => {
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchColumnTags = useCallback(async () => {
    if (!columnId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await tagService.getColumnTags(columnId);
      setTags(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to fetch column tags',
      );
    } finally {
      setLoading(false);
    }
  }, [columnId]);

  useEffect(() => {
    fetchColumnTags();
  }, [fetchColumnTags]);

  const addTag = async (tagId: string) => {
    if (!columnId) return;
    await tagService.addTagToColumn(columnId, tagId);
    await fetchColumnTags();
  };

  const removeTag = async (tagId: string) => {
    if (!columnId) return;
    await tagService.removeTagFromColumn(columnId, tagId);
    await fetchColumnTags();
  };

  return { tags, loading, error, addTag, removeTag, refetch: fetchColumnTags };
};

interface TagState {
  tag: Tag | null;
  columns: ColumnSearchResult[];
  loading: boolean;
  error: string | null;
}

type TagAction =
  | { type: 'start' }
  | { type: 'success'; tag: Tag; columns: ColumnSearchResult[] }
  | { type: 'error'; error: string };

function tagReducer(state: TagState, action: TagAction): TagState {
  switch (action.type) {
    case 'start':
      return { ...state, loading: true, error: null };
    case 'success':
      return {
        tag: action.tag,
        columns: action.columns,
        loading: false,
        error: null,
      };
    case 'error':
      return { ...state, loading: false, error: action.error };
    default:
      return state;
  }
}

export const useTag = (tagId: string | undefined) => {
  const [state, dispatch] = useReducer(tagReducer, {
    tag: null,
    columns: [],
    loading: false,
    error: null,
  });

  useEffect(() => {
    if (!tagId) return;

    let active = true;
    dispatch({ type: 'start' });

    Promise.all([tagService.getTag(tagId), tagService.getColumnsByTag(tagId)])
      .then(([tagData, colsData]) => {
        if (active) {
          dispatch({ type: 'success', tag: tagData, columns: colsData });
        }
      })
      .catch((err) => {
        if (active)
          dispatch({
            type: 'error',
            error: err instanceof Error ? err.message : 'Failed to fetch tag',
          });
      });

    return () => {
      active = false;
    };
  }, [tagId]);

  return state;
};
