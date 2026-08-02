/** SCR-01 合成样例演示。 */

import { setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { DemoExperience } from '../../../../features/batch1/landing-experience.tsx';
import { normalizePreviewState } from '../../../../lib/preview-state.ts';
import { type PageSearchParams, readSearchParam } from '../../../../lib/search-params.ts';

export default async function Scr01DemoPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: PageSearchParams;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const query = await searchParams;
  return <DemoExperience mode={normalizePreviewState(readSearchParam(query, 'state'))} />;
}
