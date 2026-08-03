/** SCR-16 聚合趋势：>=10 人展示，<10 人隐藏/合并，个人排名面不存在。 */

'use client';

import { Card, CardBody, CardHeader, PageHeader, StatCard, Tint } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { useApiGet } from '../../../../../../lib/api-hooks.ts';

interface AggregatesPayload {
  readonly aggregates: {
    completion_rate: number;
    dimension_trends: readonly { dimension: string; before: number; after: number }[];
    small_groups_hidden: boolean;
  };
}

export default function OrgAggregatesPage(): ReactNode {
  const t = useTranslations('common');
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; orgId: string }>();
  const zh = params.locale === 'zh-CN';
  const state = useApiGet<AggregatesPayload>('/v1/orgs/{orgId}/aggregates', { pathParams: { orgId: params.orgId } });
  const agg = state.data?.aggregates;
  const dimLabel = (d: string) =>
    ({ professional_competence: zh ? '专业能力' : 'Professional', communication: zh ? '沟通表达' : 'Communication', problem_solving: zh ? '问题解决' : 'Problem solving' })[d] ?? d;

  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navAggregates')} />
      <div className="mgd-grid mgd-grid--3 mb-5">
        <StatCard label={zh ? '完成率' : 'Completion rate'} value={agg ? `${agg.completion_rate}%` : '—'} tone="brand" />
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{zh ? '小样本保护' : 'Small-group protection'}</div>
          <div className="mt-2"><Tint tone="warning">{agg?.small_groups_hidden ? t('org.smallGroupHidden') : (zh ? '样本充足' : 'Sufficient sample')}</Tint></div>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('org.noRanking')}</div>
          <p className="mb-0 mt-2 text-xs text-neutral-500">{zh ? '不提供个人排名或搜索接口。' : 'No personal ranking or search surface.'}</p>
        </Card>
      </div>
      <Card>
        <CardHeader title={zh ? '维度趋势（训练前后）' : 'Dimension trends (before/after)'} />
        <CardBody className="space-y-4">
          {state.loading ? <p className="mb-0 text-sm text-neutral-500" role="status">{zh ? '加载中…' : 'Loading…'}</p> : null}
          {(agg?.dimension_trends ?? []).map((d) => (
            <div key={d.dimension}>
              <div className="mb-1 flex items-center justify-between text-sm">
                <span className="text-neutral-700">{dimLabel(d.dimension)}</span>
                <span className="font-mono text-neutral-900">{zh ? '训练前' : 'Before'} {d.before} → {zh ? '训练后' : 'After'} {d.after}</span>
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-neutral-100">
                <div className="h-full rounded-full bg-[linear-gradient(90deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))]" style={{ width: `${d.after}%` }} />
              </div>
            </div>
          ))}
        </CardBody>
      </Card>
    </>
  );
}
