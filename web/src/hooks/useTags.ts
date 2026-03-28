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
    const newTag = await tagService.createTag(data);
    setTags((prev) => [...prev, newTag]);
    return newTag;
  };

  const updateTag = async (id: string, data: TagCreate) => {
    await tagService.updateTag(id, data);
    await fetchTags();
  };

  const deleteTag = async (id: string) => {
    await tagService.deleteTag(id);
    setTags((prev) => prev.filter((t) => t.id !== id));
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
