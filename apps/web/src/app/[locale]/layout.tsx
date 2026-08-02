/**
 * 根布局：html lang、全局 Provider（next-intl + Toast）。
 * 页面级导航由 (public) / (app) / (room) 路由组布局各自提供。
 */

import { isLocale, SUPPORTED_LOCALES, type Locale } from '@mgd/i18n';
import { ToastProvider } from '@mgd/ui';
import { NextIntlClientProvider } from 'next-intl';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { notFound } from 'next/navigation';
import type { Metadata } from 'next';
import type { ReactNode } from 'react';

import { loadMessages } from '../../i18n/messages.ts';
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
    title: { default: t('brand.name'), template: `%s · ${t('brand.name')}` },
    description: t('brand.tagline'),
    applicationName: t('brand.name'),
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
  const messages = await loadMessages(locale);

  return (
    <html lang={locale}>
      <body>
        <NextIntlClientProvider messages={messages} locale={locale}>
          <ToastProvider>
            <a
              href="#main-content"
              className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-lg focus:bg-surface focus:px-4 focus:py-2 focus:shadow-[var(--mgd-app-shadow-lg)]"
            >
              {locale === 'zh-CN' ? '跳到主内容' : 'Skip to main content'}
            </a>
            {children}
          </ToastProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
