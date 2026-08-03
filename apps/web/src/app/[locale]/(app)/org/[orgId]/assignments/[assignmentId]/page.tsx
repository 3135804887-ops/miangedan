/** SCR-16 任务配置：可配置项与禁止项（红线不在界面出现）。 */

import { Button, Card, CardBody, CardHeader, IconLock, PageHeader } from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function OrgAssignmentDetailPage({ params }: { params: Promise<{ locale: string; orgId: string; assignmentId: string }> }): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';
  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.configTitle')} description={t('org.configHint')} actions={<Button variant="primary">{t('action.save')}</Button>} />
      <div className="mgd-grid mgd-grid--sidebar">
        <Card>
          <CardHeader title={zh ? '任务设置' : 'Assignment settings'} />
          <CardBody className="space-y-4">
            {[
              [zh ? '岗位或岗位类别' : 'Role or category', zh ? '后端工程师' : 'Backend engineer'],
              [zh ? '轮次与时长' : 'Rounds and duration', zh ? '3 轮 · 25-30-20 分钟' : '3 rounds · 25-30-20 min'],
              [zh ? '难度与语言' : 'Difficulty and language', zh ? '标准 · 中文' : 'Standard · Chinese'],
              [zh ? '截止时间' : 'Deadline', '2026-09-01'],
            ].map(([label, value]) => (
              <div key={label} className="rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">{label}</div>
                <div className="text-sm text-neutral-800">{value}</div>
              </div>
            ))}
          </CardBody>
        </Card>
        <Card>
          <CardHeader title={<span className="flex items-center gap-2"><IconLock size={16} className="text-warning" />{zh ? '禁止配置项' : 'Locked items'}</span>} />
          <CardBody>
            <p className="mb-0 rounded-xl bg-[color-mix(in_srgb,var(--mgd-color-warning)_8%,transparent)] p-4 text-sm leading-6 text-warning-text">{t('org.notConfigurable')}</p>
          </CardBody>
        </Card>
      </div>
    </>
  );
}
