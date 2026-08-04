'use client';

import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { BrandMark } from './app-nav.tsx';

/** 公开页品牌页眉（落地页/登录页）。文案走 i18n，链接带界面语言前缀。 */
export function PublicHeader(): ReactNode {
  const t = useTranslations('common');
  const locale = useLocale();
  const href = (path: string) => `/${locale}${path}`;

  return (
    <header className="sticky top-0 z-40 border-b border-neutral-100 bg-surface/85 backdrop-blur">
      <div className="mx-auto flex h-16 w-[min(100%-2rem,1200px)] items-center justify-between">
        <a href={href('/')} className="flex items-center gap-2.5 font-bold text-neutral-900">
          <span className="grid size-9 place-items-center rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] text-white shadow-[var(--mgd-app-shadow-brand)]">
            <BrandMark />
          </span>
          <span>
            面个蛋 <span className="font-medium text-neutral-500">MianGeDan</span>
          </span>
        </a>
        <nav className="flex items-center gap-2" aria-label={t('nav.primaryLabel')}>
          <a href={href('/demo')} className="rounded-lg px-3 py-2 text-sm font-medium text-neutral-600 hover:bg-surface-muted hover:text-neutral-900">
            {t('nav.demo')}
          </a>
          <a href={href('/auth')} className="mgd-target-primary inline-flex items-center rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-4 font-semibold text-white shadow-[var(--mgd-app-shadow-brand)] hover:brightness-105">
            {t('nav.signIn')}
          </a>
        </nav>
      </div>
    </header>
  );
}
