/** SCR-16 权限与审计：角色分离 + 追加式审计（最小可见）。 */

'use client';

import { Card, CardBody, CardHeader, PageHeader, Tint } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { useApiGet } from '../../../../../../lib/api-hooks.ts';

interface AuditsPayload {
  readonly items: readonly { actor?: string; action?: string; resource?: string; at?: string }[];
}

export default function OrgPermissionsPage(): ReactNode {
  const t = useTranslations('common');
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; orgId: string }>();
  const zh = params.locale === 'zh-CN';
  const state = useApiGet<AuditsPayload>('/v1/orgs/{orgId}/audits', { pathParams: { orgId: params.orgId } });
  const audits = state.data?.items ?? [];

  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navPermissions')} />
      <Card className="mb-5">
        <CardHeader title={t('org.roles')} />
        <CardBody>
          <p className="mb-0 text-sm text-neutral-600">{t('org.roleOwners')}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            <Tint tone="brand">admin</Tint>
            <Tint tone="info">coach</Tint>
            <Tint tone="neutral">candidate</Tint>
          </div>
        </CardBody>
      </Card>
      <Card>
        <CardHeader title={t('org.audit')} description={t('org.auditHint')} />
        <CardBody className="space-y-2">
          {state.loading ? <p className="mb-0 text-sm text-neutral-500" role="status">{zh ? '加载中…' : 'Loading…'}</p> : null}
          {audits.map((a, i) => (
            <div key={`${a.actor}-${i}`} className="rounded-xl border border-neutral-100 px-4 py-3 text-sm">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="font-medium text-neutral-800">{a.actor ?? '—'}</span>
                <span className="text-xs text-neutral-500">{a.at ?? ''}</span>
              </div>
              <p className="mb-0 mt-1 text-neutral-600">{a.action ?? '—'} · {a.resource ?? '—'}</p>
            </div>
          ))}
        </CardBody>
      </Card>
    </>
  );
}
