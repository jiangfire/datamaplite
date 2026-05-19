import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useLineage, useImpactAnalysis } from '../useColumns';
import * as services from '../../services';

vi.mock('../../services', () => ({
  columnService: {
    getLineage: vi.fn(),
    getImpactAnalysis: vi.fn(),
  },
}));

describe('useColumns hooks - race protection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('useLineage sets loading then resolves', async () => {
    vi.mocked(services.columnService.getLineage).mockResolvedValue({
      column_id: 'col-1',
      upward: [],
      downward: [],
    });

    const { result } = renderHook(() => useLineage('col-1'));

    expect(result.current.loading).toBe(true);

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.lineage).toEqual({
      column_id: 'col-1',
      upward: [],
      downward: [],
    });
  });

  it('useImpactAnalysis handles errors gracefully', async () => {
    vi.mocked(services.columnService.getImpactAnalysis).mockRejectedValue(
      new Error('network error'),
    );

    const { result } = renderHook(() => useImpactAnalysis('col-1'));

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.error).toBe('network error');
    expect(result.current.impact).toBeNull();
  });
});
