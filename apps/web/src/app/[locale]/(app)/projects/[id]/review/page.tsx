/** SCR-05 解析校对页：低置信度高亮、逐字段编辑、敏感字段说明、缺失影响弹窗。 */

import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { ReviewView } from '../../../../../../components/review-view.tsx';

export default async function ReviewPage({
  params,
}: {
  params: Promise<{ locale: string; id: string }>;
}): Promise<ReactNode> {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  return (
    <ReviewView
      locale={locale as 'zh-CN' | 'en-US'}
      projectId={id}
      labels={{
        kicker: t('review.kicker'),
        title: t('review.title'),
        desc: t('review.desc'),
        tabResume: t('review.tabResume'),
        tabJob: t('review.tabJob'),
        lowConfidence: t('review.lowConfidence'),
        excludedFields: t('review.excludedFields'),
        excludedList: t('review.excludedList'),
        addField: t('review.addField'),
        removeField: t('review.removeField'),
        confirmMaterials: t('review.confirmMaterials'),
        confirmHint: t('review.confirmHint'),
        missingImpactTitle: t('review.missingImpactTitle'),
        missingImpactDesc: t('review.missingImpactDesc'),
        missingJdOnly: t('review.missingJdOnly'),
        agreeDegraded: t('review.agreeDegraded'),
        confirm: t('review.confirm'),
      }}
    />
  );
}
