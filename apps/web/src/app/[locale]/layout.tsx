/**
 * 根布局（需求 B0-2 第 2 条）：输出 <html lang>、skip-link、全局导航与页脚。
 * 导航项在全部页面顺序一致（ACCESSIBILITY 第 4.3 节「一致导航」）。
 */

import { isLocale, SUPPORTED_LOCALES, type Locale } from '@mgd/i18n';
import { NextIntlClientProvider } from 'next-intl';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { Metadata } from 'next';
import type { ReactNode } from 'react';

import { Link } from '../../i18n/navigation.ts';
import '../globals.css';

export function generateStaticParams(): Array<{ locale: Locale }> {
  return SUPPORTED_LOCALES.map((locale) => ({ locale }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'common' });

  return {
    title: t('brand.name'),
    description: t('brand.tagline'),
  };
}

interface LayoutProps {
  readonly children: ReactNode;
  readonly params: Promise<{ locale: string }>;
}

export default async function LocaleLayout({ children, params }: LayoutProps): Promise<ReactNode> {
  const { locale } = await params;
  if (!isLocale(locale)) notFound();

  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });

  return (
    <html lang={locale}>
      <body>
        <NextIntlClientProvider>
          <a className="mgd-skip-link" href="#main-content">
            {t('nav.skipToContent')}
          </a>

          <header className="mgd-site-header">
            <nav className="mgd-global-nav mgd-shell" aria-label={t('nav.primaryLabel')}>
              <Link className="mgd-brand" href="/" aria-label={t('brand.name')}>
                <span className="mgd-brand__copy">
                  <strong className="mgd-brand__name">{t('brand.name')}</strong>
                  <small className="mgd-brand__tagline">{t('brand.tagline')}</small>
                </span>
              </Link>
              <ul className="mgd-primary-links">
                <li>
                  <Link href="/"><span aria-hidden="true">01</span>{t('nav.landing')}</Link>
                </li>
                <li>
                  <Link href="/dashboard"><span aria-hidden="true">02</span>{t('nav.dashboard')}</Link>
                </li>
                <li>
                  <Link href="/library"><span aria-hidden="true">03</span>{t('nav.library')}</Link>
                </li>
                <li>
                  <Link href="/billing"><span aria-hidden="true">04</span>{t('nav.billing')}</Link>
                </li>
                <li>
                  <Link href="/settings"><span aria-hidden="true">05</span>{t('nav.settings')}</Link>
                </li>
              </ul>
            </nav>
          </header>

          {/* 宽度约束交给 (app) / (room) 路由组布局，房间页需要全宽 */}
          <main id="main-content" className="mgd-main">{children}</main>

          <footer className="mgd-site-footer" aria-label={t('nav.footerLabel')}>
            <div className="mgd-shell mgd-site-footer__inner">
              <p><strong>{t('brand.name')}</strong></p>
              <p>{t('brand.tagline')}</p>
            </div>
          </footer>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
