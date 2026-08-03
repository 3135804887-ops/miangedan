/** SCR-13 资产与历史：简历库/岗位库/面试记录/训练进度四分区。 */

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  EmptyState,
  IconClock,
  IconEdit,
  IconFile,
  IconPlay,
  IconTrash,
  PageHeader,
  Tabs,
  Tint,
} from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function LibraryPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';

  const resumes = [
    { id: 'r1', name: zh ? '合成候选人简历 v1' : 'Synthetic resume v1', company: '合成科技', role: zh ? '后端工程师' : 'Backend engineer', version: 3, date: '2026-07-28' },
  ] as const;
  const jobs = [
    { id: 'j1', name: zh ? '后端工程师 JD' : 'Backend engineer JD', company: '合成科技', role: zh ? '后端工程师' : 'Backend engineer', version: 2, date: '2026-07-28' },
  ] as const;
  const interviews = [
    { id: 'p-0001', name: zh ? '后端工程师面试训练' : 'Backend interview training', status: zh ? '活动中' : 'Active', date: '2026-08-01' },
    { id: 'p-0006', name: zh ? '全流程训练（已完成）' : 'Full training (completed)', status: zh ? '已完成' : 'Completed', date: '2026-07-24' },
  ] as const;

  const listCard = (title: string, items: readonly { id: string; name: string; company: string; role: string; version: number; date: string }[], emptyTitle: string) => (
    <Card>
      <CardHeader title={title} />
      <CardBody className="space-y-3">
        {items.length === 0 ? (
          <EmptyState icon={<IconFile size={24} />} title={emptyTitle} />
        ) : (
          items.map((item) => (
            <div key={item.id} className="flex items-center gap-4 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
              <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-surface text-primary shadow-[var(--mgd-app-shadow-sm)]">
                <IconFile size={18} />
              </span>
              <div className="min-w-0 flex-1">
                <p className="mb-0.5 truncate font-medium text-neutral-900">{item.name}</p>
                <p className="mb-0 text-xs text-neutral-500">
                  {item.company} · {item.role} · {t('library.version', { v: String(item.version) })} · {item.date}
                </p>
              </div>
              <Button variant="secondary" targetSize="min" aria-label={t('action.edit')}><IconEdit size={15} /></Button>
              <Button variant="secondary" targetSize="min" aria-label={t('action.delete')}><IconTrash size={15} /></Button>
            </div>
          ))
        )}
      </CardBody>
    </Card>
  );

  return (
    <>
      <PageHeader kicker={t('library.kicker')} title={t('library.title')} />
      <Card>
        <CardBody>
          <Tabs
            initialId="resumes"
            items={[
              { id: 'resumes', label: t('library.tabResumes'), content: listCard(t('library.tabResumes'), resumes, t('library.emptyResumes')) },
              { id: 'jobs', label: t('library.tabJobs'), content: listCard(t('library.tabJobs'), jobs, t('library.emptyJobs')) },
              {
                id: 'interviews',
                label: t('library.tabInterviews'),
                content: (
                  <Card>
                    <CardBody className="space-y-3">
                      {interviews.map((it) => (
                        <div key={it.id} className="flex items-center gap-4 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                          <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-surface text-primary shadow-[var(--mgd-app-shadow-sm)]">
                            <IconPlay size={18} />
                          </span>
                          <div className="min-w-0 flex-1">
                            <p className="mb-0.5 truncate font-medium text-neutral-900">{it.name}</p>
                            <p className="mb-0 flex items-center gap-1 text-xs text-neutral-500">
                              <IconClock size={12} /> {it.date}
                            </p>
                          </div>
                          <Tint tone="brand">{it.status}</Tint>
                        </div>
                      ))}
                    </CardBody>
                  </Card>
                ),
              },
              {
                id: 'training',
                label: t('library.tabTraining'),
                content: (
                  <Card>
                    <CardBody>
                      <h3 className="mb-3 text-base font-semibold">{t('library.weaknesses')}</h3>
                      <div className="mb-5 flex flex-wrap gap-2">
                        <Tint tone="warning">{zh ? '行为协作' : 'Collaboration'}</Tint>
                        <Tint tone="warning">{zh ? '经历证据' : 'Experience evidence'}</Tint>
                      </div>
                      <h3 className="mb-3 text-base font-semibold">{t('library.retryTrail')}</h3>
                      <div className="space-y-2">
                        {[
                          { attempt: 1, score: 52, status: 'FAIL' },
                          { attempt: 2, score: 71, status: 'PASS' },
                        ].map((r) => (
                          <div key={r.attempt} className="flex items-center justify-between rounded-xl border border-neutral-100 px-4 py-3 text-sm">
                            <span className="text-neutral-600">{zh ? `第 ${r.attempt} 次尝试` : `Attempt ${r.attempt}`}</span>
                            <span className="flex items-center gap-3">
                              <span className="font-mono font-semibold text-neutral-900">{r.score}</span>
                              <Tint tone={r.status === 'PASS' ? 'success' : 'danger'}>{r.status}</Tint>
                            </span>
                          </div>
                        ))}
                      </div>
                    </CardBody>
                  </Card>
                ),
              },
            ]}
          />
        </CardBody>
      </Card>
    </>
  );
}
