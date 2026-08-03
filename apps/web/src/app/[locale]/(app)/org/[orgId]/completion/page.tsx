/** SCR-16 完成情况：默认最小可见，失败详情不展示。 */

'use client';

import { Card, CardBody, CardHeader, PageHeader, Tint } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { useApiGet } from '../../../../../../lib/api-hooks.ts';

interface SharesPayload {
  readonly items: readonly { assignment_id?: string; scope?: string; expires_at?: string; status?: string }[];
}

export default function OrgCompletionPage(): ReactNode {
  const t = useTranslations('common');
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; orgId: string }>();
  const zh = params.locale === 'zh-CN';
  const assignmentId = params.orgId;
  const state = useApiGet<SharesPayload>('/v1/assignments/{assignmentId}/shares', { pathParams: { assignmentId } });
  const shares = state.data?.items ?? [];

  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.completion')} />
      <p className="mb-5 rounded-xl bg-info/10 px-4 py-3 text-sm text-info">{t('org.completedNotShared')}</p>
      <Card>
        <CardHeader title={zh ? '完成情况' : 'Completion'} />
        <CardBody className="space-y-3">
          {state.loading ? <p className="mb-0 text-sm text-neutral-500" role="status">{zh ? '加载中…' : 'Loading…'}</p> : null}
          {shares.length === 0 && !state.loading ? (
            <p className="mb-0 text-sm text-neutral-500">{zh ? '暂无授权分享' : 'No authorized shares'}</p>
          ) : (
            shares.map((s, i) => (
              <div key={`${s.assignment_id}-${i}`} className="flex items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4 text-sm">
                <span className="text-neutral-700">{s.scope ?? '—'} · {s.expires_at ?? '—'}</span>
                <Tint tone={s.status === 'active' ? 'success' : 'neutral'}>{s.status ?? '—'}</Tint>
              </div>
            ))
          )}
          <p className="mb-0 text-xs text-neutral-500">{t('org.neverShowFailure')}</p>
        </CardBody>
      </Card>
    </>
  );
}
