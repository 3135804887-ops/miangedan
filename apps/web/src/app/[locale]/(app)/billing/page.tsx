/** SCR-15 购买与额度：报价、流水、订单、自动续费与退款入口。 */

'use client';

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  PageHeader,
  Tint,
  useToast,
} from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { apiFetch } from '../../../../lib/api-fetch.ts';
import { useApiGet, useApiWrite } from '../../../../lib/api-hooks.ts';

interface EntitlementPayload {
  readonly entitlement: { balance_minutes: number; plan: string };
}

interface LedgerPayload {
  readonly items: readonly { entry_id: string; entry_type: string; seconds: number; reason: string; created_at: string }[];
}

interface SubscriptionPayload {
  readonly subscription: { plan: string; status: string; auto_renew: boolean; balance_minutes?: number };
}

interface PricingPayload {
  readonly pricing: { region: string; currency: string; plans: readonly { id: string; name: string; minutes: number; price: number; currency: string; per_minute: boolean }[] };
}

export default function BillingPage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US' }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const entitlements = useApiGet<EntitlementPayload>('/v1/entitlements', {});
  const ledger = useApiGet<LedgerPayload>('/v1/usage-ledger', {});
  const subscription = useApiGet<SubscriptionPayload>('/v1/subscription', {});
  const pricing = useApiGet<PricingPayload>('/v1/pricing/{region}', { pathParams: { region: 'cn' } });
  const { run: runOrder } = useApiWrite<{ order: { order_id: string; status: string } }>();

  const planLabel = (id: string) =>
    ({ free: t('billing.freePlan'), pack: t('billing.projectPack'), pro: t('billing.proPlan'), topup: t('billing.topup') })[id] ?? id;
  const planDesc = (id: string) =>
    ({ free: t('billing.freePlanDesc'), pack: t('billing.projectPackDesc'), pro: t('billing.proPlanDesc'), topup: t('billing.perMinute') })[id] ?? '';

  const buy = async (planId: string) => {
    const res = await runOrder('/v1/orders', {
      method: 'post',
      idempotencyKey: `order-${planId}-${Date.now()}`,
      body: { plan_id: planId, region: 'cn' },
    });
    toast.push({
      title: res.ok ? (zh ? `订单已创建：${planLabel(planId)}` : `Order created: ${planLabel(planId)}`) : (zh ? '下单暂未接入（占位）' : 'Order placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const setAutoRenew = async (enabled: boolean) => {
    const res = await apiFetch('/v1/subscription/auto-renew', {
      method: 'put',
      idempotencyKey: `auto-renew-${Date.now()}`,
      body: { enabled },
    });
    toast.push({
      title: res.ok ? (zh ? '自动续费设置已保存' : 'Auto-renewal updated') : (zh ? '设置暂未接入（占位）' : 'Auto-renewal placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const cancelRenew = async () => {
    const res = await apiFetch('/v1/subscription/cancel', {
      method: 'post',
      idempotencyKey: `cancel-renew-${Date.now()}`,
      body: {},
    });
    toast.push({
      title: res.ok ? (zh ? '已取消自动续费（权益保留至账期结束）' : 'Auto-renewal cancelled (entitlement kept until period end)') : (zh ? '取消暂未接入（占位）' : 'Cancel placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const plans = pricing.data?.pricing.plans ?? [];
  const sub = subscription.data?.subscription;
  const items = ledger.data?.items ?? [];

  return (
    <>
      <PageHeader kicker={t('billing.kicker')} title={t('billing.title')} description={t('billing.desc')} />

      <div className="mgd-grid mgd-grid--3 mb-6">
        <Card className="mgd-card--brand p-5">
          <div className="text-sm text-neutral-600">{t('billing.balance')}</div>
          <div className="mgd-stat-value mt-1 text-[var(--mgd-app-brand-ink)]">{entitlements.data?.entitlement.balance_minutes ?? '—'} {t('billing.balanceMinutes')}</div>
          <div className="mt-2 text-xs text-neutral-500">{t('billing.currentPlan')}: {planLabel(entitlements.data?.entitlement.plan ?? 'free')}</div>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('billing.autoRenew')}</div>
          <div className="mt-2"><Tint tone={sub?.auto_renew ? 'success' : 'neutral'}>{sub?.auto_renew ? (zh ? '开启' : 'On') : (zh ? '关闭' : 'Off')}</Tint></div>
          <div className="mt-3 flex flex-wrap gap-2">
            <Button variant="secondary" targetSize="min" onClick={() => setAutoRenew(true)}>{t('billing.autoRenew')}</Button>
            {sub?.auto_renew ? <Button variant="secondary" targetSize="min" onClick={cancelRenew}>{t('billing.cancelRenew')}</Button> : null}
          </div>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('billing.refund')}</div>
          <p className="mb-0 mt-2 text-xs text-neutral-500">{zh ? '系统责任故障自动全额返还，账本记录原因。' : 'System-caused faults auto-refund in full with ledger reasons.'}</p>
        </Card>
      </div>

      <Card className="mb-6">
        <CardHeader title={zh ? '套餐' : 'Plans'} />
        <CardBody className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {plans.length === 0 ? (
            <p className="text-sm text-neutral-500">{zh ? '定价暂未接入（占位）' : 'Pricing placeholder'}</p>
          ) : (
            plans.map((p) => (
              <div key={p.id} className="rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-5">
                <p className="mb-1 font-semibold text-neutral-900">{planLabel(p.id)}</p>
                <p className="mb-3 text-xs text-neutral-500">{planDesc(p.id)}</p>
                <p className="mb-3 font-mono text-lg font-semibold text-neutral-900">{p.price} {p.currency}</p>
                <Button variant="primary" className="w-full" onClick={() => buy(p.id)}>{t('billing.choose')}</Button>
              </div>
            ))
          )}
        </CardBody>
      </Card>

      <div className="mgd-grid mgd-grid--sidebar">
        <Card>
          <CardHeader title={t('billing.usage')} />
          <CardBody className="space-y-3">
            {items.length === 0 ? (
              <p className="mb-0 text-sm text-neutral-500">{zh ? '暂无流水' : 'No ledger entries'}</p>
            ) : (
              items.map((l) => (
                <div key={l.entry_id} className="flex items-center justify-between gap-3 rounded-xl border border-neutral-100 px-4 py-3 text-sm">
                  <div className="min-w-0">
                    <Tint tone="brand">{t(`billing.usageTypes.${l.entry_type}`)}</Tint>
                    <p className="mb-0 mt-1 truncate text-xs text-neutral-500">{l.reason}</p>
                  </div>
                  <span className="shrink-0 font-mono text-neutral-900">{l.seconds}s</span>
                </div>
              ))
            )}
          </CardBody>
        </Card>
        <Card>
          <CardHeader title={t('billing.orders')} />
          <CardBody>
            <p className="mb-0 text-sm text-neutral-500">
              {zh ? '订单列表接口尚未提供（占位）；下单、发票与退款经契约端点处理。' : 'Order list endpoint placeholder; create/invoice/refund go through contract endpoints.'}
            </p>
          </CardBody>
        </Card>
      </div>
    </>
  );
}
