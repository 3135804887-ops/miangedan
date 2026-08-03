/** SCR-15 购买与额度：报价、余额、流水、订单与自动续费（独立勾选）。 */

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  IconBill,
  IconCheck,
  IconClock,
  IconDownload,
  PageHeader,
  StatCard,
  Tint,
} from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function BillingPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';

  const plans = [
    { id: 'free', name: t('billing.freePlan'), desc: t('billing.freePlanDesc'), minutes: 60, price: '¥0', current: true },
    { id: 'pack', name: t('billing.projectPack'), desc: t('billing.projectPackDesc'), minutes: 180, price: '¥39', current: false },
    { id: 'pro', name: t('billing.proPlan'), desc: t('billing.proPlanDesc'), minutes: 600, price: '¥99', current: false },
    { id: 'topup', name: t('billing.topup'), desc: t('billing.perMinute'), minutes: 120, price: '¥29', current: false },
  ] as const;

  const ledger = [
    { type: 'reserve', seconds: 1800, reason: zh ? '第 1 轮预留' : 'Round 1 reserve', date: '2026-08-01 09:30' },
    { type: 'consume', seconds: 482, reason: zh ? '第 1 轮实际使用' : 'Round 1 usage', date: '2026-08-01 09:38' },
    { type: 'release', seconds: 1318, reason: zh ? '第 1 轮释放' : 'Round 1 release', date: '2026-08-01 09:38' },
  ] as const;

  const typeTone = (type: string) => (type === 'consume' ? 'danger' : type === 'refund' ? 'success' : 'info') as 'danger' | 'success' | 'info';

  return (
    <>
      <PageHeader kicker={t('billing.kicker')} title={t('billing.title')} description={t('billing.desc')} />

      <div className="mgd-grid mgd-grid--3 mb-6">
        <StatCard label={t('billing.balance')} value={t('billing.balanceMinutes', { minutes: '46' })} tone="brand" hint={zh ? '免费额度方案' : 'Free plan'} />
        <StatCard label={zh ? '已用额度' : 'Used credits'} value="14 min" tone="warning" />
        <StatCard label={zh ? '本轮预留' : 'Reserved this round'} value="30 min" tone="info" />
      </div>

      <div className="mgd-grid mgd-grid--4 mb-6">
        {plans.map((p) => (
          <Card key={p.id} className={`relative p-5 ${p.current ? 'mgd-card--brand ring-2 ring-[var(--mgd-app-brand-from)]' : ''}`}>
            {p.current ? (
              <span className="absolute -top-2.5 left-4 rounded-full bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-2.5 py-0.5 text-xs font-semibold text-white shadow-[var(--mgd-app-shadow-sm)]">
                {t('billing.currentPlan')}
              </span>
            ) : null}
            <div className="mb-1 flex items-center gap-2 text-base font-semibold text-neutral-900">
              <IconBill size={17} className="text-primary" />
              {p.name}
            </div>
            <p className="mb-3 text-sm text-neutral-600">{p.desc}</p>
            <div className="mb-3 font-mono text-2xl font-bold text-neutral-900">{p.price}</div>
            <p className="mb-4 text-xs text-neutral-500">{p.minutes} min</p>
            <Button variant={p.current ? 'secondary' : 'primary'} className="w-full" disabled={p.current} disabledReason={p.current ? t('billing.currentPlan') : undefined}>
              {p.current ? t('billing.currentPlan') : t('billing.choose')}
            </Button>
          </Card>
        ))}
      </div>

      <label className="mb-6 flex cursor-pointer items-center gap-3 rounded-xl border border-neutral-100 bg-surface px-4 py-3.5 shadow-[var(--mgd-app-shadow-sm)]">
        <input type="checkbox" defaultChecked className="size-4 accent-[var(--mgd-app-brand-from)]" />
        <span className="text-sm text-neutral-700">{t('billing.autoRenew')}</span>
        <span className="ml-auto">
          <Tint tone="neutral">{t('billing.cancelRenew')}</Tint>
        </span>
      </label>

      <div className="mgd-grid mgd-grid--sidebar">
        <Card>
          <CardHeader title={t('billing.usage')} />
          <CardBody>
            <div className="space-y-2">
              {ledger.map((l) => (
                <div key={`${l.type}-${l.date}`} className="flex items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3 text-sm">
                  <div className="min-w-0">
                    <p className="mb-0.5 font-medium text-neutral-800">{l.reason}</p>
                    <p className="mb-0 flex items-center gap-1 text-xs text-neutral-500">
                      <IconClock size={11} /> {l.date}
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="font-mono text-neutral-700">{l.seconds}s</span>
                    <Tint tone={typeTone(l.type)}>{t(`billing.usageTypes.${l.type}`)}</Tint>
                  </div>
                </div>
              ))}
            </div>
          </CardBody>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader title={t('billing.orders')} />
            <CardBody>
              <div className="space-y-2">
                {[1].map((i) => (
                  <div key={i} className="flex items-center justify-between gap-3 rounded-xl border border-neutral-100 px-4 py-3 text-sm">
                    <div>
                      <p className="mb-0.5 font-medium text-neutral-800">{zh ? '单项目包' : 'Project pack'}</p>
                      <p className="mb-0 text-xs text-neutral-500">o-{i} · 2026-07-28</p>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="font-mono font-semibold">¥39</span>
                      <Tint tone="success">{t('billing.paid')}</Tint>
                    </div>
                  </div>
                ))}
              </div>
              <Button variant="secondary" className="mt-4 w-full">
                <IconDownload size={15} />
                {t('billing.invoice')}
              </Button>
            </CardBody>
          </Card>
          <Card>
            <CardBody>
              <div className="flex items-center gap-2 text-sm font-medium text-neutral-800">
                <IconCheck size={16} className="text-success" />
                {t('billing.refund')}
              </div>
              <p className="mb-0 mt-1 text-xs text-neutral-500">{zh ? '系统责任故障自动全额返还；大额退款双人审批。' : 'System faults auto-refund fully; large refunds need two-person approval.'}</p>
            </CardBody>
          </Card>
        </div>
      </div>
    </>
  );
}
