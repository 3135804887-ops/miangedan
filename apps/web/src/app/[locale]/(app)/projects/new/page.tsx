/** SCR-04 创建面试项目：简历上传 + JD 粘贴（草稿保留、样例填充、具体拒绝原因）。 */

import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { ProjectNewForm } from '../../../../../components/project-new-form.tsx';

export default async function ProjectNewPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  return (
    <ProjectNewForm
      locale={locale as 'zh-CN' | 'en-US'}
      labels={{
        kicker: t('projectNew.kicker'),
        title: t('projectNew.title'),
        desc: t('projectNew.desc'),
        resumeSection: t('projectNew.resumeSection'),
        resumeHint: t('projectNew.resumeHint'),
        dropHint: t('projectNew.dropHint'),
        browse: t('projectNew.browse'),
        replace: t('projectNew.replace'),
        remove: t('projectNew.remove'),
        scanning: t('projectNew.scanning'),
        scanRejected: t('projectNew.scanRejected', { reason: '{reason}' }),
        jdSection: t('projectNew.jdSection'),
        jdHint: t('projectNew.jdHint'),
        jdPlaceholder: t('projectNew.jdPlaceholder'),
        jdCharCount: t('projectNew.jdCharCount', { count: '{count}' }),
        sampleFill: t('projectNew.sampleFill'),
        sampleBadge: t('projectNew.sampleBadge'),
        draftSaved: t('projectNew.draftSaved'),
        continue: t('action.continue'),
        noResumeOrJd: t('projectNew.validation.noResumeOrJd'),
        resumeUploading: t('projectNew.validation.resumeUploading'),
        jdTooShort: t('projectNew.validation.jdTooShort'),
      }}
    />
  );
}
