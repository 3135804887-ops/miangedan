/**
 * Tabs：可访问选项卡（键盘左右切换、aria 关联面板）。
 */

'use client';

import { useId, useState, type ReactNode } from 'react';

export interface TabItem {
  readonly id: string;
  readonly label: ReactNode;
  readonly content: ReactNode;
  readonly disabled?: boolean;
  readonly disabledReason?: string;
}

export function Tabs({
  items,
  initialId,
}: {
  readonly items: readonly TabItem[];
  readonly initialId?: string;
}): ReactNode {
  const baseId = useId().replaceAll(':', '');
  const [activeId, setActiveId] = useState(initialId ?? items[0]?.id ?? '');
  const active = items.find((i) => i.id === activeId) ?? items[0];

  const onKeyDown = (e: React.KeyboardEvent) => {
    const idx = items.findIndex((i) => i.id === active?.id);
    let next = idx;
    if (e.key === 'ArrowRight') next = (idx + 1) % items.length;
    if (e.key === 'ArrowLeft') next = (idx - 1 + items.length) % items.length;
    if (e.key === 'Home') next = 0;
    if (e.key === 'End') next = items.length - 1;
    if (next !== idx) {
      e.preventDefault();
      const candidate = items[next];
      if (candidate !== undefined && !candidate.disabled) setActiveId(candidate.id);
    }
  };

  return (
    <div>
      <div
        role="tablist"
        aria-label="页面分区"
        onKeyDown={onKeyDown}
        className="flex flex-wrap gap-1 border-b border-neutral-100"
      >
        {items.map((item) => {
          const selected = item.id === active?.id;
          return (
            <button
              key={item.id}
              type="button"
              role="tab"
              id={`${baseId}-tab-${item.id}`}
              aria-selected={selected}
              aria-controls={`${baseId}-panel-${item.id}`}
              aria-disabled={item.disabled === true}
              title={item.disabled === true ? item.disabledReason : undefined}
              tabIndex={selected ? 0 : -1}
              onClick={() => {
                if (item.disabled !== true) setActiveId(item.id);
              }}
              className={`mgd-target-min cursor-pointer border-b-2 px-4 py-3 text-sm font-semibold transition-colors ${
                selected
                  ? 'border-[var(--mgd-app-brand-from)] text-[var(--mgd-app-brand-ink)]'
                  : 'border-transparent text-neutral-600 hover:text-neutral-900'
              } ${item.disabled === true ? 'cursor-not-allowed opacity-50' : ''}`}
            >
              {item.label}
            </button>
          );
        })}
      </div>
      <div
        role="tabpanel"
        id={`${baseId}-panel-${active?.id ?? ''}`}
        aria-labelledby={`${baseId}-tab-${active?.id ?? ''}`}
        className="pt-5"
      >
        {active?.content}
      </div>
    </div>
  );
}
