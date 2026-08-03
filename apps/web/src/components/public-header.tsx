import type { ReactNode } from 'react';

import { BrandMark } from './app-nav.tsx';

/** 公开页品牌页眉（落地页/登录页）。 */
export function PublicHeader(): ReactNode {
  return (
    <header className="sticky top-0 z-40 border-b border-neutral-100 bg-surface/85 backdrop-blur">
      <div className="mx-auto flex h-16 w-[min(100%-2rem,1200px)] items-center justify-between">
        <a href="/" className="flex items-center gap-2.5 font-bold text-neutral-900">
          <span className="grid size-9 place-items-center rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] text-white shadow-[var(--mgd-app-shadow-brand)]">
            <BrandMark />
          </span>
          <span>
            面个蛋 <span className="font-medium text-neutral-500">MianGeDan</span>
          </span>
        </a>
        <nav className="flex items-center gap-2">
          <a href="/demo" className="rounded-lg px-3 py-2 text-sm font-medium text-neutral-600 hover:bg-surface-muted hover:text-neutral-900">
            样例演示
          </a>
          <a href="/auth" className="mgd-target-primary inline-flex items-center rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-4 font-semibold text-white shadow-[var(--mgd-app-shadow-brand)] hover:brightness-105">
            登录 / 注册
          </a>
        </nav>
      </div>
    </header>
  );
}
