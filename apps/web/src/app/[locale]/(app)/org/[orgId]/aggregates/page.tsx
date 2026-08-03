/** SCR-16 聚合趋势：匿名维度聚合；细分 <10 人隐藏；无个人排名。 */

import { Card, CardBody, CardHeader, PageHeader, StatCard, Tint } from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function OrgAggregatesPage({ params }: { params: Promise<{ locale: string; orgId: string }> }): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';
  const trends = [
    { dim: zh ? '专业能力' : 'Professional competence', before: 61, after: 74 },
    { dim: zh ? '沟通表达' : 'Communication', before: 64, after: 71 },
    { dim: zh ? '问题解决' : 'Problem solving', before: 59, after: 69 },
  ];
  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navAggregates')} />
      <div className="mgd-grid mgd-grid--3 mb-5">
        <StatCard label={zh ? '完成率' : 'Completion rate'} value="68%" tone="brand" />
        <StatCard label={zh ? '平均提升' : 'Avg improvement'} value="+9.3" tone="success" />
        <StatCard label={zh ? '覆盖学员' : 'Candidates'} value="24" tone="info" />
      </div>
      <Card>
        <CardHeader title={zh ? '维度趋势（训练前后）' : 'Dimension trends (before/after)'} />
        <CardBody className="space-y-5">
          {trends.map((tr) => (
            <div key={tr.dim}>
              <div className="mb-1.5 flex justify-between text-sm">
                <span className="font-medium text-neutral-800">{tr.dim}</span>
                <span className="font-mono text-neutral-600">{tr.before} → <span className="font-semibold text-success-text">{tr.after}</span></span>
              </div>
              <div className="flex gap-1">
                <div className="h-2.5 flex-1 overflow-hidden rounded-full bg-neutral-100">
                  <div className="h-full rounded-full bg-neutral-400" style={{ width: `${tr.before}%` }} />
                </div>
                <div className="h-2.5 flex-1 overflow-hidden rounded-full bg-neutral-100">
                  <div className="h-full rounded-full bg-[linear-gradient(90deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))]" style={{ width: `${tr.after}%` }} />
                </div>
              </div>
            </div>
          ))}
          <div className="flex flex-wrap gap-2">
            <Tint tone="warning">{t('org.smallGroupHidden')}</Tint>
            <Tint tone="neutral">{t('org.noRanking')}</Tint>
          </div>
        </CardBody>
      </Card>
    </>
  );
}
