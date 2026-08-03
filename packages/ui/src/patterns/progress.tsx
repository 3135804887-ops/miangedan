/**
 * Progress：进度条（aria 可见；不确定进度用 indeterminate）。
 */

import type { ReactNode } from 'react';

export function Progress({
  value,
  label,
  max = 100,
  tone = 'brand',
}: {
  readonly value: number;
  readonly label?: ReactNode;
  readonly max?: number;
  readonly tone?: 'brand' | 'success' | 'warning' | 'danger' | 'info';
}): ReactNode {
  const pct = Math.max(0, Math.min(100, (value / max) * 100));
  const fill = {
    brand: 'bg-[var(--mgd-app-brand-from)]',
    success: 'bg-success',
    warning: 'bg-warning',
    danger: 'bg-danger',
    info: 'bg-info',
  }[tone];
  return (
    <div>
      {label !== undefined ? (
        <div className="mb-1.5 flex items-center justify-between gap-3 text-sm">
          <span className="text-neutral-600">{label}</span>
          <span className="font-mono text-xs text-neutral-600">{Math.round(pct)}%</span>
        </div>
      ) : null}
      <div
        role="progressbar"
        aria-valuenow={Math.round(pct)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={typeof label === 'string' ? label : '进度'}
        className="h-2 overflow-hidden rounded-full bg-neutral-100"
      >
        <div className={`h-full rounded-full transition-[width] duration-300 ${fill}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

export function Indeterminate({ label }: { readonly label?: ReactNode }): ReactNode {
  return (
    <div>
      {label !== undefined ? <div className="mb-1.5 text-sm text-neutral-600">{label}</div> : null}
      <div
        role="progressbar"
        aria-label={typeof label === 'string' ? label : '处理中'}
        className="h-2 overflow-hidden rounded-full bg-neutral-100"
      >
        <div className="mgd-skeleton h-full w-2/5 rounded-full" />
      </div>
    </div>
  );
}
