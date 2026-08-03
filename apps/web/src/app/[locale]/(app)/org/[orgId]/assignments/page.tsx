/** SCR-16 机构任务列表。 */

import { Button, Card, CardBody, IconArrowRight, IconClock, IconUsers, PageHeader, Tint } from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function OrgAssignmentsPage({ params }: { params: Promise<{ locale: string; orgId: string }> }): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';
  const tasks = [
    { id: 'a-1', title: zh ? '后端岗位模拟面试训练' : 'Backend mock interview training', deadline: '2026-09-01', quota: 3000, members: 24, status: zh ? '进行中' : 'In progress', tone: 'info' as const },
    { id: 'a-2', title: zh ? '产品经理结构化表达训练' : 'PM structured communication training', deadline: '2026-10-01', quota: 1500, members: 12, status: zh ? '已完成未共享' : 'Completed, not shared', tone: 'warning' as const },
  ];
  return (
    <>
      <PageHeader kicker={t('org.kicker')} title={t('org.navAssignments')} actions={<Button variant="primary">{zh ? '新建任务' : 'New assignment'}</Button>} />
      <div className="space-y-4">
        {tasks.map((task) => (
          <Card key={task.id} hoverable>
            <CardBody className="flex flex-wrap items-center justify-between gap-4">
              <div className="min-w-0 flex-1">
                <h2 className="mb-1 text-base font-semibold text-neutral-900">{task.title}</h2>
                <div className="flex flex-wrap gap-x-5 gap-y-1 text-sm text-neutral-600">
                  <span className="flex items-center gap-1.5"><IconClock size={14} />{t('org.deadline')}: {task.deadline}</span>
                  <span className="flex items-center gap-1.5"><IconUsers size={14} />{task.members} {zh ? '人' : 'members'}</span>
                  <span>{t('org.quota')}: {task.quota} min</span>
                </div>
              </div>
              <Tint tone={task.tone}>{task.status}</Tint>
              <a href={`/${locale}/org/demo/assignments/${task.id}`} className="mgd-target-primary inline-flex items-center gap-2 rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-4 text-sm font-semibold text-white shadow-[var(--mgd-app-shadow-brand)]">
                {zh ? '查看' : 'View'}
                <IconArrowRight size={14} />
              </a>
            </CardBody>
          </Card>
        ))}
      </div>
    </>
  );
}
