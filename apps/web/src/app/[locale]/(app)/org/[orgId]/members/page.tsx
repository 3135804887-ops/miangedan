/** SCR-16 机构成员：列表、邀请、移除与角色。 */

'use client';

import { Button, Card, CardBody, CardHeader, PageHeader, Tint, useToast } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { apiFetch } from '../../../../../../lib/api-fetch.ts';
import { useApiGet } from '../../../../../../lib/api-hooks.ts';

interface MembersPayload {
  readonly items: readonly { name?: string; role?: string; email?: string; user_id?: string; joined_at?: string }[];
}

export default function OrgMembersPage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; orgId: string }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const orgId = params.orgId;
  const state = useApiGet<MembersPayload>('/v1/orgs/{orgId}/members', { pathParams: { orgId } });

  const invite = async () => {
    const res = await apiFetch('/v1/orgs/{orgId}/members/invite', {
      method: 'post',
      idempotencyKey: `invite-${orgId}-${Date.now()}`,
      pathParams: { orgId },
      body: { email: 'member@example.org', role: 'candidate' },
    });
    toast.push({
      title: res.ok ? (zh ? '邀请已发送' : 'Invitation sent') : (zh ? '邀请暂未接入（占位）' : 'Invite placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const remove = async (userId: string) => {
    const res = await apiFetch('/v1/orgs/{orgId}/members/{userId}', {
      method: 'delete',
      idempotencyKey: `remove-${userId}-${Date.now()}`,
      pathParams: { orgId, userId },
    });
    toast.push({
      title: res.ok ? (zh ? '成员已移除，机构访问立即失效' : 'Member removed, access revoked') : (zh ? '移除暂未接入（占位）' : 'Remove placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const members = state.data?.items ?? [];
  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navMembers')} actions={<Button variant="primary" onClick={invite}>{t('org.invite')}</Button>} />
      <Card>
        <CardHeader title={zh ? '成员列表' : 'Members'} />
        <CardBody className="space-y-3">
          {state.loading ? <p className="mb-0 text-sm text-neutral-500" role="status">{zh ? '加载中…' : 'Loading…'}</p> : null}
          {members.length === 0 && !state.loading ? (
            <p className="mb-0 text-sm text-neutral-500">{zh ? '暂无成员' : 'No members'}</p>
          ) : (
            members.map((m) => (
              <div key={m.user_id ?? m.email ?? m.name} className="flex items-center gap-4 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                <div className="min-w-0 flex-1">
                  <p className="mb-0.5 truncate font-medium text-neutral-900">{m.name ?? m.email}</p>
                  <p className="mb-0 text-xs text-neutral-500">{m.email ?? ''}</p>
                </div>
                <Tint tone="brand">{m.role ?? 'candidate'}</Tint>
                <Button variant="secondary" targetSize="min" onClick={() => m.user_id !== undefined && remove(m.user_id)}>{zh ? '移除' : 'Remove'}</Button>
              </div>
            ))
          )}
        </CardBody>
      </Card>
      <p className="mt-4 text-xs text-neutral-500">{t('org.inviteMethods')} · {t('org.joinNotice')}</p>
    </>
  );
}
