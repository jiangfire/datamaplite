import { useState, useCallback, useRef } from 'react';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
  id: string;
  message: string;
  type: ToastType;
}

let globalId = 0;

export const useToast = () => {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  const remove = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
    const timer = timersRef.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timersRef.current.delete(id);
    }
  }, []);

  const add = useCallback(
    (message: string, type: ToastType = 'info', duration = 4000) => {
      const id = `toast-${++globalId}`;
      setToasts((prev) => [...prev, { id, message, type }]);
      const timer = setTimeout(() => remove(id), duration);
      timersRef.current.set(id, timer);
      return id;
    },
    [remove],
  );

  return { toasts, add, remove };
};
