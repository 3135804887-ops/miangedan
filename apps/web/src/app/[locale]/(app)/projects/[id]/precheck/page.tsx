/** SCR-07 会前检查。 */

import { setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { PrecheckExperience } from '../../../../../../features/batch1/precheck-experience.tsx';
import { normalizePreviewState } from '../../../../../../lib/preview-state.ts';
import { type PageSearchParams, readSearchParam } from '../../../../../../lib/search-params.ts';

export default async function Scr07PrecheckPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string; id: string }>;
  searchParams: PageSearchParams;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const query = await searchParams;
  return (
    <PrecheckExperience
      mode={normalizePreviewState(readSearchParam(query, 'state'))}
      insufficientQuota={readSearchParam(query, 'quota') === 'insufficient'}
      otherDeviceActive={readSearchParam(query, 'device') === 'active'}
    />
  );
}
