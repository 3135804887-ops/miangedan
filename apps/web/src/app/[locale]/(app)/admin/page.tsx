/** SCR-17 运营后台：匿名监控、供应商、审计与占位分区。 */

'use client';

import { Card, CardBody, CardHeader, PageHeader, Tint } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { useApiGet } from '../../../../lib/api-hooks.ts';

interface RegionsPayload {
  readonly regions: readonly { region: string; online_rooms: number; queue: number; slo: string; error_budget?: string }[];
}

interface ProvidersPayload {
  readonly providers: readonly { provider_id: string; capability: string; status: string; latency_ms?: number; error_rate?: string }[];
}

interface AuditsPayload {
  readonly items: readonly { actor?: string; action?: string; resource?: string; at?: string }[];
}

export default function AdminPage(): ReactNode {
  const t = useTranslations('common');
  const params = useParams<{ locale: 'zh-CN' | 'en-US' }>();
  const zh = params.locale === 'zh-CN';
  const regions = useApiGet<RegionsPayload>('/v1/admin/regions', {});
  const providers = useApiGet<ProvidersPayload>('/v1/admin/providers', {});
  const audits = useApiGet<AuditsPayload>('/v1/admin/audit-logs', {});

  return (
    <>
      <PageHeader kicker={t('admin.kicker')} title={zh ? '运营后台' : 'Admin console'} />
      <div className="mgd-grid mgd-grid--3 mb-5">
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('admin.navMonitor')}</div>
          <p className="mb-0 mt-2 text-xs text-neutral-500">{t('admin.anonymized')} · {t('admin.noPersonalData')} · {t('admin.noEavesdrop')}</p>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('admin.navProviders')}</div>
          <p className="mb-0 mt-2 text-xs text-neutral-500">{t('admin.providerNote')}</p>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('admin.navAudit')}</div>
          <p className="mb-0 mt-2 text-xs text-neutral-500">{t('admin.transcriptAccess')} · {t('admin.ticketDefault')}</p>
        </Card>
      </div>

      <div className="mgd-grid mgd-grid--sidebar">
        <div className="space-y-4">
          <Card>
            <CardHeader title={t('admin.navMonitor')} />
            <CardBody className="space-y-2">
              {(regions.data?.regions ?? []).map((r) => (
                <div key={r.region} className="flex items-center justify-between gap-3 rounded-xl border border-neutral-100 px-4 py-3 text-sm">
                  <span className="font-medium text-neutral-800">{r.region}</span>
                  <span className="text-neutral-600">{t('admin.onlineRooms')}: {r.online_rooms} · {t('admin.queue')}: {r.queue}</span>
                  <Tint tone="brand">{r.slo}</Tint>
                </div>
              ))}
            </CardBody>
          </Card>
          <Card>
            <CardHeader title={t('admin.navProviders')} />
            <CardBody className="space-y-2">
              {(providers.data?.providers ?? []).map((p) => (
                <div key={p.provider_id} className="flex items-center justify-between gap-3 rounded-xl border border-neutral-100 px-4 py-3 text-sm">
                  <span className="font-medium text-neutral-800">{p.provider_id}</span>
                  <span className="text-neutral-600">{p.capability} · {p.latency_ms ?? '—'}ms · {p.error_rate ?? '—'}</span>
                  <Tint tone={p.status === 'open' ? 'success' : p.status === 'half_open' ? 'warning' : 'danger'}>{p.status}</Tint>
                </div>
              ))}
            </CardBody>
          </Card>
        </div>
        <Card>
          <CardHeader title={t('admin.navAudit')} />
          <CardBody className="space-y-2">
            {(audits.data?.items ?? []).map((a, i) => (
              <div key={`${a.actor}-${i}`} className="rounded-xl border border-neutral-100 px-4 py-3 text-sm">
                <span className="font-medium text-neutral-800">{a.actor ?? '—'}</span>
                <p className="mb-0 mt-1 text-neutral-600">{a.action ?? '—'} · {a.resource ?? '—'} · {a.at ?? ''}</p>
              </div>
            ))}
          </CardBody>
        </Card>
      </div>

      <div className="mt-5 grid gap-4 sm:grid-cols-3">
        <Card className="p-4"><div className="text-sm font-medium">{t('admin.navVersions')}</div><p className="mb-0 mt-1 text-xs text-neutral-500">{t('admin.contentNote')}</p></Card>
        <Card className="p-4"><div className="text-sm font-medium">{t('admin.navContent')}</div><p className="mb-0 mt-1 text-xs text-neutral-500">{t('admin.contentSafety')}</p></Card>
        <Card className="p-4"><div className="text-sm font-medium">{t('admin.navFinance')}</div><p className="mb-0 mt-1 text-xs text-neutral-500">{t('admin.providerRoutes')}</p></Card>
      </div>
    </>
  );
}
