import type { ReactNode } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Mock } from 'vitest';
import type {
  BusinessTerm,
  ColumnDetail,
  ColumnMapping,
  ColumnSearchResult,
  DDLGenerateResponse,
} from '../types';

vi.mock('../hooks', () => ({
  useColumnDetail: vi.fn(),
  useLineage: vi.fn(),
  useImpactAnalysis: vi.fn(),
  useDDLGeneration: vi.fn(),
  useColumnTags: vi.fn(),
  useTags: vi.fn(),
  useTerms: vi.fn(),
  useColumnMappings: vi.fn(),
  useColumnSearch: vi.fn(),
}));

vi.mock('../services', () => ({
  columnService: {
    assignTerm: vi.fn(),
  },
}));

vi.mock('../components', async () => {
  const actual = await vi.importActual<typeof import('../components')>(
    '../components',
  );

  return {
    ...actual,
    Layout: ({ children }: { children: ReactNode }) => (
      <div data-testid="layout">{children}</div>
    ),
    ColumnDetailCard: ({ column }: { column: ColumnDetail }) => (
      <div data-testid="column-detail-card">{column.name}</div>
    ),
    LineageGraph: () => <div data-testid="lineage-graph" />,
    ImpactAnalysis: ({
      impact,
    }: {
      impact: { total_objects: number };
    }) => <div data-testid="impact-analysis">{impact.total_objects}</div>,
    Modal: ({
      children,
      isOpen,
      title,
    }: {
      children: ReactNode;
      isOpen: boolean;
      title: string;
    }) =>
      isOpen ? (
        <div data-testid="modal">
          <h2>{title}</h2>
          {children}
        </div>
      ) : null,
  };
});

import { columnService } from '../services';
import {
  useColumnDetail,
  useColumnMappings,
  useColumnSearch,
  useColumnTags,
  useDDLGeneration,
  useImpactAnalysis,
  useLineage,
  useTags,
  useTerms,
} from '../hooks';
import { ColumnDetailPage } from './ColumnDetailPage';

const createColumn = (overrides?: Partial<ColumnDetail>): ColumnDetail => ({
  id: 'col-1',
  name: 'source_column',
  data_type: 'varchar',
  full_data_type: 'varchar(255)',
  is_nullable: false,
  is_primary_key: false,
  ordinal_position: 1,
  confidence: 0.96,
  object: {
    id: 'obj-1',
    name: 'users',
    type: 'table',
  },
  source: {
    id: 'src-1',
    name: 'mysql-prod',
    type: 'mysql',
  },
  ...overrides,
});

const createTerm = (overrides?: Partial<BusinessTerm>): BusinessTerm => ({
  id: 'term-1',
  name: '用户标识',
  category: '主数据',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

const createMapping = (overrides?: Partial<ColumnMapping>): ColumnMapping => ({
  id: 'map-1',
  source_column_id: 'col-1',
  target_column_id: 'col-2',
  mapping_type: 'alias',
  confidence: 0.9,
  created_at: '2024-01-02T00:00:00Z',
  target_column: {
    id: 'col-2',
    name: 'already_mapped',
    data_type: 'varchar',
    object_name: 'dim_users',
    source_name: 'dw_mysql',
  },
  ...overrides,
});

const createSearchResult = (
  overrides?: Partial<ColumnSearchResult>,
): ColumnSearchResult => ({
  id: 'col-3',
  name: 'available_target',
  data_type: 'varchar',
  object_name: 'ads_users',
  source_id: 'src-2',
  source_name: 'ads_mysql',
  source_type: 'mysql',
  confidence: 0.88,
  ...overrides,
});

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={['/columns/col-1']}>
      <Routes>
        <Route path="/columns/:id" element={<ColumnDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );

type RefetchFn = () => Promise<void>;
type TagMutationFn = (tagId: string) => Promise<void>;
type MappingMutationPayload = {
  source_column_id: string;
  target_column_id: string;
  mapping_type: 'alias' | 'transform' | 'derived' | 'synonym';
  confidence?: number;
};
type CreateMappingFn = (data: MappingMutationPayload) => Promise<void>;
type DeleteMappingFn = (mappingId: string) => Promise<void>;
type SearchColumnsFn = (query: string, limit?: number) => Promise<void>;
type GenerateDDLFn = (
  objectId: string,
  targetType: 'mysql' | 'postgres',
) => Promise<DDLGenerateResponse>;

describe('ColumnDetailPage', () => {
  let refetchColumn: Mock<RefetchFn>;
  let addTag: Mock<TagMutationFn>;
  let removeTag: Mock<TagMutationFn>;
  let createColumnMapping: Mock<CreateMappingFn>;
  let deleteColumnMapping: Mock<DeleteMappingFn>;
  let searchColumns: Mock<SearchColumnsFn>;
  let generateDDL: Mock<GenerateDDLFn>;

  beforeEach(() => {
    refetchColumn = vi.fn<RefetchFn>().mockResolvedValue(undefined);
    addTag = vi.fn<TagMutationFn>().mockResolvedValue(undefined);
    removeTag = vi.fn<TagMutationFn>().mockResolvedValue(undefined);
    createColumnMapping = vi.fn<CreateMappingFn>().mockResolvedValue(undefined);
    deleteColumnMapping = vi.fn<DeleteMappingFn>().mockResolvedValue(undefined);
    searchColumns = vi.fn<SearchColumnsFn>().mockResolvedValue(undefined);
    generateDDL = vi
      .fn<GenerateDDLFn>()
      .mockResolvedValue({ object_id: 'obj-1', sql: 'CREATE TABLE users();' });

    vi.mocked(useColumnDetail).mockReturnValue({
      column: createColumn(),
      loading: false,
      error: null,
      refetch: refetchColumn,
    });
    vi.mocked(useLineage).mockReturnValue({
      lineage: null,
      loading: false,
      error: null,
    });
    vi.mocked(useImpactAnalysis).mockReturnValue({
      impact: null,
      loading: false,
      error: null,
    });
    vi.mocked(useDDLGeneration).mockReturnValue({
      generateDDL,
      generating: false,
      error: null,
    });
    vi.mocked(useColumnTags).mockReturnValue({
      tags: [],
      loading: false,
      error: null,
      addTag,
      removeTag,
      refetch: vi.fn(),
    });
    vi.mocked(useTags).mockReturnValue({
      tags: [],
      loading: false,
      error: null,
      refetch: vi.fn(),
      createTag: vi.fn(),
      updateTag: vi.fn(),
      deleteTag: vi.fn(),
    });
    vi.mocked(useTerms).mockReturnValue({
      terms: [createTerm()],
      loading: false,
      error: null,
      refetch: vi.fn(),
      createTerm: vi.fn(),
      updateTerm: vi.fn(),
      deleteTerm: vi.fn(),
    });
    vi.mocked(useColumnMappings).mockReturnValue({
      mappings: [],
      loading: false,
      error: null,
      refetch: vi.fn(),
      createMapping: createColumnMapping,
      deleteMapping: deleteColumnMapping,
    });
    vi.mocked(useColumnSearch).mockReturnValue({
      results: [],
      loading: false,
      error: null,
      search: searchColumns,
    });
    vi.mocked(columnService.assignTerm).mockResolvedValue(undefined);
  });

  it('renders governance empty states and disables unchanged actions', () => {
    renderPage();

    expect(screen.getByText('当前字段尚未绑定业务术语')).toBeInTheDocument();
    expect(screen.getByText('当前字段还没有映射关系')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '保存术语绑定' }),
    ).toBeDisabled();
    expect(screen.getByRole('button', { name: '解除绑定' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '新增映射' })).toBeDisabled();
  });

  it('renders load error state when column detail fails', () => {
    vi.mocked(useColumnDetail).mockReturnValue({
      column: null,
      loading: false,
      error: 'boom',
      refetch: refetchColumn,
    });

    renderPage();

    expect(screen.getByText('加载失败: boom')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '返回搜索' })).toBeInTheDocument();
  });

  it('assigns a term and refreshes the detail', async () => {
    const user = userEvent.setup();

    renderPage();

    await user.selectOptions(screen.getByLabelText('字段术语选择'), 'term-1');
    await user.click(screen.getByRole('button', { name: '保存术语绑定' }));

    await waitFor(() => {
      expect(columnService.assignTerm).toHaveBeenCalledWith('col-1', {
        term_id: 'term-1',
      });
    });
    expect(refetchColumn).toHaveBeenCalledTimes(1);
  });

  it('supports unassigning the current term', async () => {
    const user = userEvent.setup();

    vi.mocked(useColumnDetail).mockReturnValue({
      column: createColumn({
        term: {
          id: 'term-2',
          name: '旧术语',
        },
      }),
      loading: false,
      error: null,
      refetch: refetchColumn,
    });

    renderPage();

    await user.click(screen.getByRole('button', { name: '解除绑定' }));

    await waitFor(() => {
      expect(columnService.assignTerm).toHaveBeenCalledWith('col-1', {
        term_id: null,
      });
    });
    expect(refetchColumn).toHaveBeenCalledTimes(1);
  });

  it('filters out the current column and existing mappings from search results', () => {
    vi.mocked(useColumnMappings).mockReturnValue({
      mappings: [createMapping()],
      loading: false,
      error: null,
      refetch: vi.fn(),
      createMapping: createColumnMapping,
      deleteMapping: deleteColumnMapping,
    });
    vi.mocked(useColumnSearch).mockReturnValue({
      results: [
        createSearchResult({
          id: 'col-1',
          name: 'source_column',
          object_name: 'users',
          source_id: 'src-1',
          source_name: 'mysql-prod',
        }),
        createSearchResult({
          id: 'col-2',
          name: 'already_mapped',
          object_name: 'dim_users',
          source_id: 'src-2',
          source_name: 'dw_mysql',
        }),
        createSearchResult(),
      ],
      loading: false,
      error: null,
      search: searchColumns,
    });

    renderPage();

    fireEvent.change(screen.getByLabelText('目标字段搜索'), {
      target: { value: 'user' },
    });

    expect(searchColumns).toHaveBeenCalledWith('user', 8);
    expect(
      screen.queryByRole('button', { name: /source_column/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /already_mapped/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /available_target/i }),
    ).toBeInTheDocument();
  });

  it('creates a mapping with blank confidence as undefined and resets the form', async () => {
    const user = userEvent.setup();

    vi.mocked(useColumnSearch).mockReturnValue({
      results: [createSearchResult()],
      loading: false,
      error: null,
      search: searchColumns,
    });

    renderPage();

    fireEvent.change(screen.getByLabelText('目标字段搜索'), {
      target: { value: 'available' },
    });
    await user.click(
      screen.getByRole('button', { name: /available_target/i }),
    );
    fireEvent.change(screen.getByLabelText('映射置信度'), {
      target: { value: '' },
    });
    await user.click(screen.getByRole('button', { name: '新增映射' }));

    await waitFor(() => {
      expect(createColumnMapping).toHaveBeenCalledWith({
        source_column_id: 'col-1',
        target_column_id: 'col-3',
        mapping_type: 'alias',
        confidence: undefined,
      });
    });
    expect(refetchColumn).toHaveBeenCalledTimes(1);
    expect(screen.getByLabelText('目标字段搜索')).toHaveValue('');
    expect(screen.getByLabelText('映射置信度')).toHaveValue(1);
  });

  it('blocks mapping creation when confidence is out of range', async () => {
    const user = userEvent.setup();

    vi.mocked(useColumnSearch).mockReturnValue({
      results: [createSearchResult()],
      loading: false,
      error: null,
      search: searchColumns,
    });

    renderPage();

    fireEvent.change(screen.getByLabelText('目标字段搜索'), {
      target: { value: 'available' },
    });
    await user.click(
      screen.getByRole('button', { name: /available_target/i }),
    );
    fireEvent.change(screen.getByLabelText('映射置信度'), {
      target: { value: '1.1' },
    });
    await user.click(screen.getByRole('button', { name: '新增映射' }));

    expect(screen.getByText('置信度必须在 0 到 1 之间')).toBeInTheDocument();
    expect(createColumnMapping).not.toHaveBeenCalled();
  });

  it('shows mapping deletion errors without hiding the current list', async () => {
    const user = userEvent.setup();

    deleteColumnMapping.mockRejectedValue(new Error('delete failed'));
    vi.mocked(useColumnMappings).mockReturnValue({
      mappings: [createMapping()],
      loading: false,
      error: null,
      refetch: vi.fn(),
      createMapping: createColumnMapping,
      deleteMapping: deleteColumnMapping,
    });

    renderPage();

    await user.click(screen.getByRole('button', { name: '删除' }));

    await waitFor(() => {
      expect(deleteColumnMapping).toHaveBeenCalledWith('map-1');
    });
    expect(screen.getByText('delete failed')).toBeInTheDocument();
    expect(screen.getByText('already_mapped')).toBeInTheDocument();
    expect(refetchColumn).not.toHaveBeenCalled();
  });
});
