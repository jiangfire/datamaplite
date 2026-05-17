import React, { createContext, useContext } from 'react';
import { useToast, type ToastType } from '../hooks/useToast';

interface ToastContextValue {
  toast: (message: string, type?: ToastType) => string;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export const useToastContext = () => {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToastContext must be used within ToastProvider');
  return ctx;
};

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const { toasts, add, remove } = useToast();

  return (
    <ToastContext.Provider value={{ toast: add }}>
      {children}
      <div className="fixed bottom-4 right-4 z-[100] flex flex-col gap-2">
        {toasts.map((t) => (
          <div
            key={t.id}
            role="alert"
            className={`px-4 py-3 rounded-lg shadow-lg text-sm font-medium max-w-sm transition-all ${
              t.type === 'error'
                ? 'bg-red-50 text-red-700 border border-red-200'
                : t.type === 'success'
                  ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                  : t.type === 'warning'
                    ? 'bg-amber-50 text-amber-700 border border-amber-200'
                    : 'bg-slate-50 text-slate-700 border border-slate-200'
            }`}
          >
            <div className="flex items-center gap-2">
              <span>{t.message}</span>
              <button
                onClick={() => remove(t.id)}
                className="ml-2 text-current opacity-60 hover:opacity-100"
                aria-label="关闭"
              >
                ×
              </button>
            </div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
};
