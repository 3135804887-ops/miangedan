/** SCR-01 落地页。 */

import { setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { LandingExperience } from '../../../features/batch1/landing-experience.tsx';
import { normalizePreviewState } from '../../../lib/preview-state.ts';
import { type PageSearchParams, readSearchParam } from '../../../lib/search-params.ts';

export default async function Scr01LandingPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: PageSearchParams;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const query = await searchParams;
  return <LandingExperience mode={normalizePreviewState(readSearchParam(query, 'state'))} />;
}
