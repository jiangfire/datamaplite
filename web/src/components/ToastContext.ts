import { createContext } from 'react';
import type { ToastType } from '../hooks/useToast';

export interface ToastContextValue {
  toast: (message: string, type?: ToastType) => string;
}

export const ToastContext = createContext<ToastContextValue | null>(null);
