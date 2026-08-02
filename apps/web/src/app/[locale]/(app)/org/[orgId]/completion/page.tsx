/** SCR-16 完成情况：默认最小可见，未授权显示"已完成未共享"。 */

import { Card, CardBody, PageHeader, Tint } from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function OrgCompletionPage({ params }: { params: Promise<{ locale: string; orgId: string }> }): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';
  const rows = [
    { name: zh ? '学员 A（匿名编号 001）' : 'Candidate A (anon 001)', status: zh ? '已完成' : 'Completed', shared: true },
    { name: zh ? '学员 B（匿名编号 002）' : 'Candidate B (anon 002)', status: zh ? '已完成' : 'Completed', shared: false },
    { name: zh ? '学员 C（匿名编号 003）' : 'Candidate C (anon 003)', status: zh ? '进行中' : 'In progress', shared: false },
  ];
  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.completion')} />
      <p className="mb-4 text-sm text-neutral-600">{t('org.neverShowFailure')}</p>
      <Card>
        <CardBody className="space-y-2">
          {rows.map((r) => (
            <div key={r.name} className="flex items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3.5 text-sm">
              <span className="font-medium text-neutral-800">{r.name}</span>
              <div className="flex items-center gap-3">
                <Tint tone={r.shared ? 'success' : 'neutral'}>{r.status}</Tint>
                {!r.shared ? <Tint tone="warning">{t('org.completedNotShared')}</Tint> : null}
              </div>
            </div>
          ))}
        </CardBody>
      </Card>
    </>
  );
}
