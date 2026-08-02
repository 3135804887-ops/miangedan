/** SCR-16 授权结果视图：范围与有效期，到期自动失效。 */

import { Card, CardBody, PageHeader, Tint } from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function OrgSharesPage({ params }: { params: Promise<{ locale: string; orgId: string }> }): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';
  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navShares')} description={t('org.shareScopeHint')} />
      <Card>
        <CardBody className="space-y-3">
          {[
            { assignment: zh ? '后端岗位模拟面试训练' : 'Backend mock interview training', scope: zh ? '雷达图与维度得分' : 'Radar chart and dimension scores', expiry: '2026-09-30' },
          ].map((s) => (
            <div key={s.assignment} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-4">
              <div>
                <p className="mb-0.5 font-medium text-neutral-900">{s.assignment}</p>
                <p className="mb-0 text-sm text-neutral-600">{t('org.shareScope')}: {s.scope}</p>
              </div>
              <div className="flex items-center gap-3 text-sm">
                <span className="font-mono text-neutral-500">{s.expiry}</span>
                <Tint tone="success">{zh ? '有效' : 'Active'}</Tint>
              </div>
            </div>
          ))}
        </CardBody>
      </Card>
    </>
  );
}
