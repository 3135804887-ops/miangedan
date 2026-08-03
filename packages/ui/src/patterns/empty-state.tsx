/**
 * EmptyState：空状态（说明为什么为空 + 下一步引导；不得展示误导性占位）。
 */

import type { ReactNode } from 'react';

export function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  readonly icon?: ReactNode;
  readonly title: ReactNode;
  readonly description?: ReactNode;
  readonly action?: ReactNode;
}): ReactNode {
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-6 py-14 text-center">
      {icon !== undefined ? (
        <div className="grid size-14 place-items-center rounded-2xl bg-[var(--mgd-app-brand-soft)] text-[var(--mgd-app-brand-ink)]">
          {icon}
        </div>
      ) : null}
      <div>
        <h3 className="m-0 text-lg font-semibold text-neutral-900">{title}</h3>
        {description !== undefined ? (
          <p className="mx-auto mt-1 max-w-md text-sm text-neutral-600">{description}</p>
        ) : null}
      </div>
      {action !== undefined ? <div className="mt-1">{action}</div> : null}
    </div>
  );
}
