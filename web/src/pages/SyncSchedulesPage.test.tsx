import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { SyncSchedulesPage } from './SyncSchedulesPage';

vi.mock('../hooks/useSyncSchedules', () => ({
  useSyncSchedules: () => ({
    schedules: [
      {
        id: 'sched-1',
        source_id: 'src-1',
        name: 'Daily Sync',
        description: 'Sync every day',
        cron_expression: '0 0 2 * * *',
        is_active: true,
        last_run_at: '2026-05-27T02:00:00Z',
        last_run_status: 'success',
        next_run_at: '2026-05-28T02:00:00Z',
      },
    ],
    loading: false,
    createSchedule: vi.fn().mockResolvedValue(undefined),
    updateSchedule: vi.fn().mockResolvedValue(undefined),
    deleteSchedule: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock('../hooks/useSources', () => ({
  useSources: () => ({
    sources: [{ id: 'src-1', name: 'Test MySQL' }],
  }),
}));

vi.mock('../components/Layout', () => ({
  Layout: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

describe('SyncSchedulesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders schedule list', () => {
    render(<SyncSchedulesPage />);
    expect(screen.getByText('Daily Sync')).toBeInTheDocument();
    expect(screen.getByText('Test MySQL')).toBeInTheDocument();
    expect(screen.getByText('0 0 2 * * *')).toBeInTheDocument();
  });

  it('opens create modal on button click', () => {
    render(<SyncSchedulesPage />);
    fireEvent.click(screen.getByText('+ New Schedule'));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('New Schedule')).toBeInTheDocument();
  });

  it('opens edit modal with existing data', () => {
    render(<SyncSchedulesPage />);
    fireEvent.click(screen.getByText('Edit'));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Edit Schedule')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Daily Sync')).toBeInTheDocument();
  });

  it('opens confirm dialog on delete click', () => {
    render(<SyncSchedulesPage />);
    fireEvent.click(screen.getByText('Delete'));
    expect(screen.getByText('Delete Schedule')).toBeInTheDocument();
    expect(screen.getByText(/Are you sure/)).toBeInTheDocument();
  });

  it('closes modal on cancel', () => {
    render(<SyncSchedulesPage />);
    fireEvent.click(screen.getByText('+ New Schedule'));
    expect(screen.getByRole('dialog')).toBeInTheDocument();

    const cancelButtons = screen.getAllByText('Cancel');
    fireEvent.click(cancelButtons[0]);

    waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });
});
