/** SCR-04 创建项目。 */

import { setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { CreateProjectExperience } from '../../../../../features/batch1/create-project-experience.tsx';
import { normalizePreviewState } from '../../../../../lib/preview-state.ts';
import { type PageSearchParams, readSearchParam } from '../../../../../lib/search-params.ts';

export default async function Scr04ProjectNewPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: PageSearchParams;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const query = await searchParams;
  return <CreateProjectExperience mode={normalizePreviewState(readSearchParam(query, 'state'))} />;
}
