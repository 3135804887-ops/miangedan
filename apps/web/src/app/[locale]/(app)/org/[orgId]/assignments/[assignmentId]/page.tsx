/** SCR-16 任务详情：模板不可配置（红线）与发布/关闭。 */

'use client';

import { Button, Card, CardBody, CardHeader, PageHeader, Tint, useToast } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { apiFetch } from '../../../../../../../lib/api-fetch.ts';
import { useApiGet } from '../../../../../../../lib/api-hooks.ts';

interface AssignmentPayload {
  readonly assignment: { assignment_id: string; title?: string; status?: string; deadline?: string; quota_minutes?: number };
}

export default function OrgAssignmentDetailPage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; orgId: string; assignmentId: string }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const orgId = params.orgId;
  const assignmentId = params.assignmentId;
  const state = useApiGet<AssignmentPayload>('/v1/orgs/{orgId}/assignments/{assignmentId}', { pathParams: { orgId, assignmentId } });
  const assignment = state.data?.assignment;

  const publish = async () => {
    const res = await apiFetch('/v1/orgs/{orgId}/assignments/{assignmentId}/publish', {
      method: 'post',
      idempotencyKey: `publish-${assignmentId}-${Date.now()}`,
      pathParams: { orgId, assignmentId },
    });
    toast.push({
      title: res.ok ? (zh ? '任务已发布' : 'Assignment published') : (zh ? '发布暂未接入（占位）' : 'Publish placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const close = async () => {
    const res = await apiFetch('/v1/orgs/{orgId}/assignments/{assignmentId}/close', {
      method: 'post',
      idempotencyKey: `close-${assignmentId}-${Date.now()}`,
      pathParams: { orgId, assignmentId },
    });
    toast.push({
      title: res.ok ? (zh ? '任务已关闭' : 'Assignment closed') : (zh ? '关闭暂未接入（占位）' : 'Close placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={assignment?.title ?? assignmentId} />
      <Card>
        <CardHeader title={t('org.configTitle')} description={t('org.configHint')} />
        <CardBody>
          <Tint tone="warning">{t('org.notConfigurable')}</Tint>
          <p className="mb-4 mt-3 text-sm text-neutral-600">
            {zh ? `状态：${assignment?.status ?? '—'} · 截止：${assignment?.deadline ?? '—'} · 额度：${assignment?.quota_minutes ?? '—'} 分钟` : `Status: ${assignment?.status ?? '—'} · Deadline: ${assignment?.deadline ?? '—'} · Quota: ${assignment?.quota_minutes ?? '—'} min`}
          </p>
          <div className="flex gap-2">
            <Button variant="primary" onClick={publish}>{zh ? '发布' : 'Publish'}</Button>
            <Button variant="secondary" onClick={close}>{zh ? '关闭' : 'Close'}</Button>
            <Button variant="secondary">{t('action.save')}</Button>
          </div>
        </CardBody>
      </Card>
    </>
  );
}
