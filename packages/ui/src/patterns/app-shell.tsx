/**
 * AppShell：登录后应用壳（侧边导航 + 顶栏 + 内容区；移动端抽屉）。
 * 导航项由调用方注入（i18n 文案与链接由应用层提供）。
 */

'use client';

import { useEffect, useState, type ReactNode } from 'react';

import { IconMenu, IconX } from '../primitives/icons.tsx';

export interface NavItem {
  readonly href: string;
  readonly label: ReactNode;
  readonly icon?: ReactNode;
  readonly active?: boolean;
  readonly badge?: ReactNode;
}

export interface NavGroup {
  readonly label?: ReactNode;
  readonly items: readonly NavItem[];
}

export function AppShell({
  brand,
  brandMark,
  groups,
  topbar,
  children,
}: {
  readonly brand: ReactNode;
  readonly brandMark?: ReactNode;
  readonly groups: readonly NavGroup[];
  readonly topbar?: ReactNode;
  readonly children: ReactNode;
}): ReactNode {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open]);

  return (
    <div className="mgd-shell">
      {open ? (
        <button
          type="button"
          aria-label="关闭导航"
          className="mgd-shell__scrim cursor-pointer border-0"
          onClick={() => setOpen(false)}
        />
      ) : null}
      <aside className={`mgd-shell__sidebar ${open ? 'mgd-shell__sidebar--open' : ''}`}>
        <div className="mgd-shell__brand">
          <span className="mgd-shell__brand-mark" aria-hidden="true">
            {brandMark}
          </span>
          <span>{brand}</span>
          <button
            type="button"
            aria-label="关闭导航"
            className="ml-auto grid size-10 cursor-pointer place-items-center rounded-lg border-0 bg-transparent text-neutral-600 hover:bg-surface-muted lg:hidden"
            onClick={() => setOpen(false)}
          >
            <IconX size={18} />
          </button>
        </div>
        <nav className="mgd-shell__nav" aria-label="主导航">
          {groups.map((group, gi) => (
            <div key={gi}>
              {group.label !== undefined ? (
                <div className="mgd-shell__nav-label">{group.label}</div>
              ) : null}
              {group.items.map((item) => (
                <a
                  key={item.href}
                  href={item.href}
                  className="mgd-nav-item"
                  aria-current={item.active ? 'page' : undefined}
                >
                  {item.icon !== undefined ? <span aria-hidden="true">{item.icon}</span> : null}
                  <span className="flex-1">{item.label}</span>
                  {item.badge !== undefined ? <span>{item.badge}</span> : null}
                </a>
              ))}
            </div>
          ))}
        </nav>
      </aside>
      <div className="mgd-shell__main">
        <header className="mgd-shell__topbar">
          <button
            type="button"
            aria-label="打开导航"
            className="grid size-10 cursor-pointer place-items-center rounded-lg border-0 bg-transparent text-neutral-700 hover:bg-surface-muted lg:hidden"
            onClick={() => setOpen(true)}
          >
            <IconMenu size={20} />
          </button>
          {topbar}
        </header>
        <main id="main-content" className="mgd-safe">
          {children}
        </main>
      </div>
    </div>
  );
}
