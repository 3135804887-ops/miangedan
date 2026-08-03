/** SCR-16 机构任务列表：创建入口 + 占位（契约无列表 GET，创建走 POST）。 */

'use client';

import { Button, Card, CardBody, CardHeader, PageHeader, useToast } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { apiFetch } from '../../../../../../lib/api-fetch.ts';

export default function OrgAssignmentsPage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; orgId: string }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const orgId = params.orgId;

  const create = async () => {
    const res = await apiFetch('/v1/orgs/{orgId}/assignments', {
      method: 'post',
      idempotencyKey: `assignment-create-${orgId}-${Date.now()}`,
      pathParams: { orgId },
      body: { title: zh ? '新任务' : 'New assignment', deadline: '2026-09-30', quota_minutes: 3000 },
    });
    toast.push({
      title: res.ok ? (zh ? '任务已创建' : 'Assignment created') : (zh ? '创建暂未接入（占位）' : 'Create placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navAssignments')} actions={<Button variant="primary" onClick={create}>{zh ? '新建任务' : 'New assignment'}</Button>} />
      <Card>
        <CardHeader title={t('org.navAssignments')} />
        <CardBody>
          <p className="mb-0 text-sm text-neutral-500">
            {zh
              ? `任务列表端点尚未提供（占位）；创建/发布/关闭走契约端点。${t('org.deadline')} · ${t('org.quota')}`
              : 'Assignment list endpoint placeholder; create/publish/close use contract endpoints.'}
          </p>
        </CardBody>
      </Card>
    </>
  );
}
