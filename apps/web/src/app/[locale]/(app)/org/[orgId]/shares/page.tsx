/** SCR-16 授权分享：范围+期限，撤回即时失效。 */

'use client';

import { Button, Card, CardBody, CardHeader, PageHeader, Tint, useToast } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { apiFetch } from '../../../../../../lib/api-fetch.ts';
import { useApiGet } from '../../../../../../lib/api-hooks.ts';

interface SharesPayload {
  readonly items: readonly { share_id?: string; assignment_id?: string; scope?: string; expires_at?: string; status?: string }[];
}

export default function OrgSharesPage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; orgId: string }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const assignmentId = params.orgId;
  const state = useApiGet<SharesPayload>('/v1/assignments/{assignmentId}/shares', { pathParams: { assignmentId } });
  const shares = state.data?.items ?? [];

  const grant = async () => {
    const res = await apiFetch('/v1/assignments/{assignmentId}/shares', {
      method: 'post',
      idempotencyKey: `share-${assignmentId}-${Date.now()}`,
      pathParams: { assignmentId },
      body: { scope: 'radar', expires_at: '2026-09-30T00:00:00Z' },
    });
    toast.push({
      title: res.ok ? (zh ? '分享已授权（范围+期限）' : 'Share granted (scope + expiry)') : (zh ? '授权暂未接入（占位）' : 'Grant placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const revoke = async (shareId: string) => {
    const res = await apiFetch('/v1/assignments/{assignmentId}/shares/{shareId}', {
      method: 'delete',
      idempotencyKey: `revoke-${shareId}-${Date.now()}`,
      pathParams: { assignmentId, shareId },
    });
    toast.push({
      title: res.ok ? (zh ? '已撤回，在线访问立即失效' : 'Revoked, access invalid immediately') : (zh ? '撤回暂未接入（占位）' : 'Revoke placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navShares')} actions={<Button variant="primary" onClick={grant}>{zh ? '授权分享' : 'Grant share'}</Button>} />
      <p className="mb-5 text-sm text-neutral-600">{t('org.shareScopeHint')}</p>
      <Card>
        <CardHeader title={t('org.shareScope')} />
        <CardBody className="space-y-3">
          {state.loading ? <p className="mb-0 text-sm text-neutral-500" role="status">{zh ? '加载中…' : 'Loading…'}</p> : null}
          {shares.length === 0 && !state.loading ? (
            <p className="mb-0 text-sm text-neutral-500">{zh ? '暂无分享' : 'No shares'}</p>
          ) : (
            shares.map((s) => (
              <div key={s.share_id ?? s.scope} className="flex items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4 text-sm">
                <div className="min-w-0">
                  <p className="mb-0.5 font-medium text-neutral-800">{s.scope ?? '—'}</p>
                  <p className="mb-0 text-xs text-neutral-500">{s.assignment_id ?? '—'} · {s.expires_at ?? '—'}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Tint tone={s.status === 'active' ? 'success' : 'neutral'}>{s.status ?? '—'}</Tint>
                  {s.share_id !== undefined ? <Button variant="secondary" targetSize="min" onClick={() => revoke(s.share_id as string)}>{zh ? '撤回' : 'Revoke'}</Button> : null}
                </div>
              </div>
            ))
          )}
        </CardBody>
      </Card>
    </>
  );
}
