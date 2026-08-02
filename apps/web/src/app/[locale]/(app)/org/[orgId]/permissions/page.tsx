/** SCR-16 权限与审计。 */

import { Card, CardBody, CardHeader, PageHeader, Tint } from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function OrgPermissionsPage({ params }: { params: Promise<{ locale: string; orgId: string }> }): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';
  const audit = [
    { actor: zh ? '指导老师 A' : 'Coach A', action: zh ? '查看维度聚合（匿名）' : 'Viewed aggregate (anonymous)', at: '2026-08-01 08:00' },
    { actor: zh ? '机构管理员' : 'Org admin', action: zh ? '邀请成员' : 'Invited member', at: '2026-08-01 07:00' },
  ];
  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navPermissions')} />
      <div className="mgd-grid mgd-grid--sidebar">
        <Card>
          <CardHeader title={t('org.roles')} />
          <CardBody>
            <Tint tone="brand">{t('org.roleOwners')}</Tint>
            <p className="mb-0 mt-3 text-sm text-neutral-600">{t('org.auditHint')}</p>
          </CardBody>
        </Card>
        <Card>
          <CardHeader title={t('org.audit')} />
          <CardBody className="space-y-2">
            {audit.map((a) => (
              <div key={`${a.actor}-${a.at}`} className="rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3 text-sm">
                <p className="mb-0.5 font-medium text-neutral-800">{a.actor} · {a.action}</p>
                <p className="mb-0 font-mono text-xs text-neutral-500">{a.at}</p>
              </div>
            ))}
          </CardBody>
        </Card>
      </div>
    </>
  );
}
