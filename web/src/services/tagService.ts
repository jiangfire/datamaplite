import { api } from './api';
import type { Tag, TagCreate, ColumnSearchResult } from '../types';

export const tagService = {
  // 列出所有标签
  listTags: () => api.get<Tag[]>('/tags'),

  // 创建标签
  createTag: (data: TagCreate) => api.post<Tag>('/tags', data),

  // 获取标签详情
  getTag: (id: string) => api.get<Tag>(`/tags/${id}`),

  // 更新标签
  updateTag: (id: string, data: TagCreate) => api.put<void>(`/tags/${id}`, data),

  // 删除标签
  deleteTag: (id: string) => api.delete<void>(`/tags/${id}`),

  // 获取标签关联的字段
  getColumnsByTag: (id: string) => api.get<ColumnSearchResult[]>(`/tags/${id}/columns`),

  // 给字段添加标签
  addTagToColumn: (columnId: string, tagId: string) =>
    api.post<void>(`/columns/${columnId}/tags`, { tag_id: tagId }),

  // 从字段移除标签
  removeTagFromColumn: (columnId: string, tagId: string) =>
    api.delete<void>(`/columns/${columnId}/tags/${tagId}`),
};
