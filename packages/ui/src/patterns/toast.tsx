/**
 * Toast：轻量通知（自关闭、可手动关闭、aria-live）。
 */

'use client';

import {
  createContext,
  useCallback,
  useContext,
  useId,
  useState,
  type ReactNode,
} from 'react';

import { IconAlert, IconCheck, IconX } from '../primitives/icons.tsx';

export interface ToastInput {
  readonly title: string;
  readonly description?: string;
  readonly tone?: 'success' | 'danger' | 'info';
}

interface ToastItem extends ToastInput {
  readonly id: string;
}

interface ToastContextValue {
  readonly push: (t: ToastInput) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { readonly children: ReactNode }): ReactNode {
  const [toasts, setToasts] = useState<readonly ToastItem[]>([]);
  const uid = useId();

  const dismiss = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const push = useCallback(
    (input: ToastInput) => {
      const id = `${uid}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
      setToasts((prev) => [...prev, { ...input, id }]);
      window.setTimeout(() => dismiss(id), 5000);
    },
    [dismiss, uid],
  );

  return (
    <ToastContext.Provider value={{ push }}>
      {children}
      <div
        aria-live="polite"
        className="pointer-events-none fixed bottom-4 right-4 z-[100] flex w-[min(92vw,380px)] flex-col gap-2"
      >
        {toasts.map((t) => (
          <div
            key={t.id}
            className="pointer-events-auto flex items-start gap-3 rounded-xl border border-neutral-100 bg-surface p-4 shadow-[var(--mgd-app-shadow-lg)]"
          >
            <span aria-hidden="true" className="mt-0.5">
              {t.tone === 'danger' ? (
                <span className="text-danger">
                  <IconAlert size={18} />
                </span>
              ) : t.tone === 'success' ? (
                <span className="text-success">
                  <IconCheck size={18} />
                </span>
              ) : (
                <span className="text-info">
                  <IconAlert size={18} />
                </span>
              )}
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-sm font-semibold text-neutral-900">{t.title}</div>
              {t.description !== undefined ? (
                <div className="mt-0.5 text-sm text-neutral-600">{t.description}</div>
              ) : null}
            </div>
            <button
              type="button"
              aria-label="关闭通知"
              className="grid size-8 cursor-pointer place-items-center rounded-lg border-0 bg-transparent text-neutral-500 hover:bg-neutral-100"
              onClick={() => dismiss(t.id)}
            >
              <IconX size={14} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (ctx === null) {
    throw new Error('useToast 必须在 ToastProvider 内使用');
  }
  return ctx;
}
