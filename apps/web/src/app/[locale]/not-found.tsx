/**
 * 404 视图（需求 B0-2 第 4 条）：提供返回工作台与返回落地页两个入口。
 */

import { getTranslations } from 'next-intl/server';
import type { ReactNode } from 'react';

import { Link } from '../../i18n/navigation.ts';

export default async function NotFound(): Promise<ReactNode> {
  const t = await getTranslations('common');

  return (
    <section data-mgd-view="not-found">
      <h1>{t('error.notFoundHeading')}</h1>
      <p>{t('error.notFoundBody')}</p>
      <ul>
        <li>
          <Link href="/dashboard">{t('error.backToDashboard')}</Link>
        </li>
        <li>
          <Link href="/">{t('error.backToLanding')}</Link>
        </li>
      </ul>
    </section>
  );
}
