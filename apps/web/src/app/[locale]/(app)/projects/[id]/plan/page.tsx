/** SCR-06 面试计划。 */

import { PROJECT_STATUSES } from '@mgd/domain-states';
import { setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { PlanExperience } from '../../../../../../features/batch1/plan-experience.tsx';
import { normalizePreviewState } from '../../../../../../lib/preview-state.ts';
import { type PageSearchParams, readSearchParam } from '../../../../../../lib/search-params.ts';

export default async function Scr06PlanPage({
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
  const projectStatus = phase === 'generating' ? PROJECT_STATUSES[4] : phase === 'failed' ? PROJECT_STATUSES[6] : phase === 'confirmed' ? PROJECT_STATUSES[7] : PROJECT_STATUSES[5];
  return (
    <PlanExperience
      mode={normalizePreviewState(readSearchParam(query, 'state'))}
      projectStatus={projectStatus}
      hasUnreadyRound={readSearchParam(query, 'round') === 'unready'}
    />
  );
}
