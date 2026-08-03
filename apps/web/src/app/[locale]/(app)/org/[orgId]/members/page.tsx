/** SCR-16 成员与邀请。 */

import { Avatar, Button, Card, CardBody, CardHeader, PageHeader, Tint } from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function OrgMembersPage({ params }: { params: Promise<{ locale: string; orgId: string }> }): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';
  const members = [
    { name: zh ? '机构管理员' : 'Org admin', role: 'admin', email: 'admin@example.org' },
    { name: zh ? '指导老师 A' : 'Coach A', role: 'coach', email: 'coach-a@example.org' },
    { name: zh ? '候选学员' : 'Candidate', role: 'candidate', email: 'candidate@example.com' },
  ];
  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navMembers')} actions={<Button variant="primary">{t('org.invite')}</Button>} />
      <p className="mb-4 text-sm text-neutral-600">{t('org.inviteMethods')}</p>
      <Card>
        <CardHeader title={zh ? '成员列表' : 'Members'} />
        <CardBody className="space-y-3">
          {members.map((m) => (
            <div key={m.email} className="flex items-center gap-4 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
              <Avatar name={m.name} />
              <div className="min-w-0 flex-1">
                <p className="mb-0.5 font-medium text-neutral-900">{m.name}</p>
                <p className="mb-0 text-xs text-neutral-500">{m.email}</p>
              </div>
              <Tint tone={m.role === 'admin' ? 'brand' : m.role === 'coach' ? 'info' : 'neutral'}>{m.role}</Tint>
            </div>
          ))}
        </CardBody>
      </Card>
      <p className="mt-4 text-xs text-neutral-500">{t('org.joinNotice')}</p>
    </>
  );
}
