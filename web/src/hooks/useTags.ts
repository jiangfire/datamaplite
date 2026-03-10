import { useState, useEffect, useCallback } from 'react';
import { tagService } from '../services';
import type { Tag, TagCreate } from '../types';

export const useTags = () => {
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
      return newTag;
    } catch (err) {
      throw err;
    }
  };

  const updateTag = async (id: string, data: TagCreate) => {
    try {
      await tagService.updateTag(id, data);
      await fetchTags();
    } catch (err) {
      throw err;
    }
  };

  const deleteTag = async (id: string) => {
    try {
      await tagService.deleteTag(id);
      setTags((prev) => prev.filter((t) => t.id !== id));
    } catch (err) {
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

  const fetchColumnTags = useCallback(async () => {
    if (!columnId) return;
    setLoading(true);
    try {
      // 通过columnService获取字段详情，其中包含标签
      // 这里简化处理，实际应该调用专门的API
      setTags([]);
    } catch (err) {
      console.error('Failed to fetch column tags:', err);
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
  };

  const removeTag = async (tagId: string) => {
    if (!columnId) return;
    await tagService.removeTagFromColumn(columnId, tagId);
  };

  return { tags, loading, addTag, removeTag, refetch: fetchColumnTags };
};
