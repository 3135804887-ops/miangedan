/** SCR-17 运营后台：区域监控/供应商/版本治理/内容安全/工单/财务/审计（默认脱敏、无改分控件）。 */

import {
  Card,
  CardBody,
  CardHeader,
  IconLock,
  PageHeader,
  StatCard,
  Tabs,
  Tint,
} from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function AdminPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';

  const regions = [
    { region: 'cn', rooms: 12, queue: 0, slo: '99.95%', budget: '72%' },
    { region: 'eu', rooms: 4, queue: 0, slo: '99.95%', budget: '81%' },
    { region: 'intl', rooms: 7, queue: 1, slo: '99.9%', budget: '64%' },
  ] as const;
  const providers = [
    { id: 'llm_cn_primary', capability: 'LLM', status: 'open', latency: 420 },
    { id: 'asr_cn_primary', capability: 'ASR', status: 'open', latency: 180 },
    { id: 'avatar_cn_primary', capability: 'Avatar', status: 'half_open', latency: 320 },
  ] as const;

  const monitorTab = (
    <div className="space-y-5">
      <div className="mgd-grid mgd-grid--3">
        <StatCard label={t('admin.onlineRooms')} value="23" tone="brand" />
        <StatCard label={t('admin.queue')} value="1" tone="warning" />
        <StatCard label={t('admin.slo')} value="99.9%" tone="success" />
      </div>
      <Card>
        <CardBody className="space-y-3">
          {regions.map((r) => (
            <div key={r.region} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3.5">
              <span className="font-mono font-semibold text-neutral-900">{r.region}</span>
              <span className="text-sm text-neutral-600">{t('admin.onlineRooms')}: {r.rooms} · {t('admin.queue')}: {r.queue}</span>
              <span className="font-mono text-sm">{r.slo} · {r.budget}</span>
            </div>
          ))}
        </CardBody>
      </Card>
      <p className="mb-0 text-xs text-neutral-500">{t('admin.anonymized')}</p>
    </div>
  );

  const providerTab = (
    <Card>
      <CardHeader title={t('admin.providerRoutes')} description={t('admin.providerNote')} />
      <CardBody className="space-y-3">
        {providers.map((p) => (
          <div key={p.id} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3.5">
            <div>
              <p className="mb-0.5 font-mono text-sm font-semibold text-neutral-900">{p.id}</p>
              <p className="mb-0 text-xs text-neutral-500">{p.capability} · {p.latency}ms</p>
            </div>
            <Tint tone={p.status === 'open' ? 'success' : 'warning'}>{p.status}</Tint>
          </div>
        ))}
      </CardBody>
    </Card>
  );

  const versionsTab = (
    <Card>
      <CardHeader title={t('admin.navVersions')} />
      <CardBody className="space-y-3">
        {[
          ['prompt-plan-generation', 'v0.1 → v0.2', zh ? '灰度中' : 'Canary'],
          ['rubrics/v1/default', '冻结', zh ? '活跃正式面试固定版本' : 'Pinned for live formal sessions'],
          ['workflows/project', 'v1', zh ? '全部区域' : 'All regions'],
        ].map(([name, version, note]) => (
          <div key={name} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3.5 text-sm">
            <span className="font-mono font-semibold text-neutral-900">{name}</span>
            <span className="text-neutral-600">{version}</span>
            <Tint tone="info">{note}</Tint>
          </div>
        ))}
      </CardBody>
    </Card>
  );

  const contentTab = (
    <Card>
      <CardHeader title={t('admin.contentSafety')} description={t('admin.contentNote')} />
      <CardBody>
        <div className="space-y-2">
          {[
            { label: zh ? '来源可信度' : 'Source credibility', value: 'high', tone: 'success' as const },
            { label: zh ? '失效与下架' : 'Expiry and takedown', value: 'warning', tone: 'warning' as const },
            { label: zh ? '版权投诉' : 'Copyright complaints', value: 'info', tone: 'info' as const },
          ].map((item) => (
            <div key={item.label} className="flex items-center justify-between rounded-xl border border-neutral-100 px-4 py-3 text-sm">
              <span className="text-neutral-800">{item.label}</span>
              <Tint tone={item.tone}>{item.value}</Tint>
            </div>
          ))}
        </div>
      </CardBody>
    </Card>
  );

  const ticketTab = (
    <Card>
      <CardHeader title={t('admin.navTickets')} />
      <CardBody>
        <p className="mb-4 rounded-xl bg-[var(--mgd-app-surface-muted)] p-4 text-sm text-neutral-600">{t('admin.ticketDefault')}</p>
        <p className="mb-0 text-sm text-neutral-600">{t('admin.transcriptAccess')}</p>
      </CardBody>
    </Card>
  );

  const financeTab = (
    <Card>
      <CardHeader title={t('admin.navFinance')} />
      <CardBody className="space-y-2">
        {[
          [zh ? '故障返还（待审批）' : 'Fault refund (pending)', '¥12.00'],
          [zh ? '大额退款（双人审批）' : 'Large refund (two-person)', '¥499.00'],
        ].map(([label, amount]) => (
          <div key={label} className="flex items-center justify-between rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3.5 text-sm">
            <span className="text-neutral-800">{label}</span>
            <span className="font-mono font-semibold">{amount}</span>
          </div>
        ))}
      </CardBody>
    </Card>
  );

  const auditTab = (
    <Card>
      <CardHeader title={t('admin.navAudit')} />
      <CardBody className="space-y-2">
        {[
          [zh ? '破窗访问' : 'Break-glass access', 'system', '2026-08-01 03:12'],
          [zh ? '版本回滚' : 'Version rollback', 'admin', '2026-07-31 22:00'],
        ].map(([label, actor, at]) => (
          <div key={at} className="rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3 text-sm">
            <p className="mb-0.5 font-medium text-neutral-800">{label} · {actor}</p>
            <p className="mb-0 font-mono text-xs text-neutral-500">{at}</p>
          </div>
        ))}
      </CardBody>
    </Card>
  );

  return (
    <>
      <PageHeader kicker={t('admin.kicker')} title={t('admin.kicker')} />
      <p className="mb-5 inline-flex items-center gap-2 rounded-full bg-danger/10 px-4 py-1.5 text-sm text-danger">
        <IconLock size={14} />
        {t('admin.noPersonalData')} · {t('admin.noEavesdrop')}
      </p>
      <Card>
        <CardBody>
          <Tabs
            initialId="monitor"
            items={[
              { id: 'monitor', label: t('admin.navMonitor'), content: monitorTab },
              { id: 'providers', label: t('admin.navProviders'), content: providerTab },
              { id: 'versions', label: t('admin.navVersions'), content: versionsTab },
              { id: 'content', label: t('admin.navContent'), content: contentTab },
              { id: 'tickets', label: t('admin.navTickets'), content: ticketTab },
              { id: 'finance', label: t('admin.navFinance'), content: financeTab },
              { id: 'audit', label: t('admin.navAudit'), content: auditTab },
            ]}
          />
        </CardBody>
      </Card>
    </>
  );
}
