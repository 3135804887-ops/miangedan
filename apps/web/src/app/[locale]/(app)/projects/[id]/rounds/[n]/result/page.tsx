/** SCR-10 路由壳（批次 0 建立，页面内容在后续批次落地）。 */

import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { RouteShell } from '../../../../../../../../components/route-shell.tsx';

export default async function Scr10ResultPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('common');

  return (
    <RouteShell
      scrId="SCR-10"
      title={t('pages.scr10Result')}
      notice={t('pages.shellNotice')}
    />
  );
}
