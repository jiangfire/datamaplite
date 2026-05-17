import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Modal } from '../Modal';

describe('Modal', () => {
  it('renders when open with correct ARIA attributes', () => {
    render(
      <Modal isOpen={true} onClose={vi.fn()} title="Test Modal">
        <button>Action</button>
      </Modal>,
    );

    const dialog = screen.getByRole('dialog');
    expect(dialog).toBeInTheDocument();
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(screen.getByText('Test Modal')).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    render(
      <Modal isOpen={false} onClose={vi.fn()} title="Hidden">
        <div>Content</div>
      </Modal>,
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('close button has accessible label', () => {
    render(
      <Modal isOpen={true} onClose={vi.fn()} title="Accessible">
        <div>Content</div>
      </Modal>,
    );

    const closeBtn = screen.getByLabelText('关闭');
    expect(closeBtn).toBeInTheDocument();
  });
});
