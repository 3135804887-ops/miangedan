/** SCR-05 解析校对。 */

import { PROJECT_STATUSES } from '@mgd/domain-states';
import { setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { ReviewExperience } from '../../../../../../features/batch1/review-experience.tsx';
import { normalizePreviewState } from '../../../../../../lib/preview-state.ts';
import { type PageSearchParams, readSearchParam } from '../../../../../../lib/search-params.ts';

export default async function Scr05ReviewPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; id: string }>;
  searchParams: PageSearchParams;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const query = await searchParams;
  const phase = readSearchParam(query, 'phase');
  const projectStatus = phase === 'parsing' ? PROJECT_STATUSES[1] : phase === 'failed' ? PROJECT_STATUSES[3] : PROJECT_STATUSES[2];
  return <ReviewExperience mode={normalizePreviewState(readSearchParam(query, 'state'))} projectStatus={projectStatus} />;
}
