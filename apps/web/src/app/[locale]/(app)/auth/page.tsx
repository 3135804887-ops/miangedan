/** SCR-02 登录与注册。 */

import { setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { AuthExperience } from '../../../../features/batch1/auth-experience.tsx';
import { normalizePreviewState } from '../../../../lib/preview-state.ts';
import { type PageSearchParams, readSearchParam } from '../../../../lib/search-params.ts';

export default async function Scr02AuthPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: PageSearchParams;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const query = await searchParams;
  return <AuthExperience mode={normalizePreviewState(readSearchParam(query, 'state'))} />;
}
