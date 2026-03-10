import { api } from './api';
import type {
  DQRule,
  DQRuleWithResult,
  DQRuleCreate,
  DQRuleFilter,
  DQResult,
  DQCheckRequest,
  DQCheckResponse,
  DQStats,
} from '../types';

export const dqService = {
  // 创建规则
  createRule: (data: DQRuleCreate) =>
    api.post<DQRule>('/dq/rules', data),

  // 列出规则
  listRules: (filter?: DQRuleFilter) =>
    api.get<DQRuleWithResult[]>('/dq/rules', filter as Record<string, unknown>),

  // 获取规则详情
  getRule: (id: string) =>
    api.get<DQRule>(`/dq/rules/${id}`),

  // 更新规则
  updateRule: (id: string, data: DQRuleCreate) =>
    api.put<void>(`/dq/rules/${id}`, data),

  // 删除规则
  deleteRule: (id: string) =>
    api.delete<void>(`/dq/rules/${id}`),

  // 执行质量检查
  checkRules: (data: DQCheckRequest) =>
    api.post<DQCheckResponse>('/dq/check', data),

  // 获取检测结果
  getResults: (params?: { rule_id?: string; batch_id?: string; limit?: number }) =>
    api.get<DQResult[]>('/dq/results', params),

  // 获取统计数据
  getStats: () =>
    api.get<DQStats>('/dq/stats'),
};
