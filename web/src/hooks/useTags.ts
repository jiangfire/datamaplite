import { useState, useEffect, useCallback } from 'react';
import { tagService } from '../services';
import { useToastContext } from '../components/ToastProvider';
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

export const useTag = (tagId: string | undefined) => {
  const [tag, setTag] = useState<Tag | null>(null);
  const [columns, setColumns] = useState<ColumnSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!tagId) {
      setTag(null);
      setColumns([]);
      return;
    }

    let active = true;
    setLoading(true);
    setError(null);

    Promise.all([tagService.getTag(tagId), tagService.getColumnsByTag(tagId)])
      .then(([tagData, colsData]) => {
        if (active) {
          setTag(tagData);
          setColumns(colsData);
        }
      })
      .catch((err) => {
        if (active)
          setError(err instanceof Error ? err.message : 'Failed to fetch tag');
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [tagId]);

  return { tag, columns, loading, error };
};
