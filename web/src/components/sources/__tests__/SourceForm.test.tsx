import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SourceForm } from '../SourceForm';
import type { DataSourceUpdate } from '../../../types';

vi.mock('../../../services', () => ({
  sourceService: {
    testConnection: vi.fn().mockResolvedValue(undefined),
  },
}));

describe('SourceForm', () => {
  it('edit mode strips empty credentials from payload', () => {
    const onSubmit = vi.fn();
    render(
      <SourceForm
        isOpen={true}
        onClose={vi.fn()}
        onSubmit={onSubmit}
        mode="edit"
        initialData={{
          name: 'MySQL Prod',
          type: 'mysql',
          host: 'localhost',
          port: 3306,
          database: 'prod',
          username: '',
          password: '',
          description: '',
        }}
      />,
    );

    fireEvent.click(screen.getByText('保存'));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'MySQL Prod',
        type: 'mysql',
        host: 'localhost',
        port: 3306,
        database: 'prod',
      }),
    );

    const payload = onSubmit.mock.calls[0][0] as DataSourceUpdate;
    expect(payload.username).toBeUndefined();
    expect(payload.password).toBeUndefined();
  });

  it('edit mode includes non-empty credentials in payload', () => {
    const onSubmit = vi.fn();
    render(
      <SourceForm
        isOpen={true}
        onClose={vi.fn()}
        onSubmit={onSubmit}
        mode="edit"
        initialData={{
          name: 'MySQL Prod',
          type: 'mysql',
          host: 'localhost',
          port: 3306,
          database: 'prod',
          username: 'admin',
          password: 'secret',
        }}
      />,
    );

    fireEvent.click(screen.getByText('保存'));

    const payload = onSubmit.mock.calls[0][0] as DataSourceUpdate;
    expect(payload.username).toBe('admin');
    expect(payload.password).toBe('secret');
  });

  it('create mode requires password', () => {
    const onSubmit = vi.fn();
    render(
      <SourceForm
        isOpen={true}
        onClose={vi.fn()}
        onSubmit={onSubmit}
        mode="create"
      />,
    );

    fireEvent.click(screen.getByText('创建'));

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText('密码不能为空')).toBeInTheDocument();
  });
});
