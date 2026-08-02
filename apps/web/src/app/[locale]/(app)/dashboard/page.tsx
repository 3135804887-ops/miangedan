/** SCR-03 工作台。 */

import { setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { DashboardExperience } from '../../../../features/batch1/dashboard-experience.tsx';
import { normalizePreviewState } from '../../../../lib/preview-state.ts';
import { type PageSearchParams, readSearchParam } from '../../../../lib/search-params.ts';

export default async function Scr03DashboardPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: PageSearchParams;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const query = await searchParams;
  return (
    <DashboardExperience
      mode={normalizePreviewState(readSearchParam(query, 'state'))}
      initialFilters={{
        company: readSearchParam(query, 'company') ?? '',
        role: readSearchParam(query, 'role') ?? '',
        date: readSearchParam(query, 'date') ?? '',
        language: readSearchParam(query, 'language') ?? '',
        status: readSearchParam(query, 'status') ?? '',
      }}
    />
  );
}
