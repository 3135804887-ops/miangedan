/** SCR-13 资产与历史：简历库/岗位库/面试记录/训练进度四分区。 */

'use client';

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
  useToast,
} from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { apiFetch } from '../../../../lib/api-fetch.ts';
import { useApiGet } from '../../../../lib/api-hooks.ts';

interface LibraryPayload {
  readonly items: readonly {
    material_id: string;
    name?: string;
    company?: string;
    job_title?: string;
    version?: number;
    confirmed_at?: string;
  }[];
}

export default function LibraryPage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US' }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const resumesState = useApiGet<LibraryPayload>('/v1/library/resumes', {});
  const jobsState = useApiGet<LibraryPayload>('/v1/library/jobs', {});

  const resumes = (resumesState.data?.items ?? []).map((x, i) => ({
    id: x.material_id,
    name: x.name ?? (zh ? `简历 ${i + 1}` : `Resume ${i + 1}`),
    company: x.company ?? '—',
    role: x.job_title ?? '—',
    version: x.version ?? 1,
    date: (x.confirmed_at ?? '').slice(0, 10),
  }));
  const jobs = (jobsState.data?.items ?? []).map((x, i) => ({
    id: x.material_id,
    name: x.name ?? (zh ? `JD ${i + 1}` : `JD ${i + 1}`),
    company: x.company ?? '—',
    role: x.job_title ?? '—',
    version: x.version ?? 1,
    date: (x.confirmed_at ?? '').slice(0, 10),
  }));

  const removeResume = async (id: string) => {
    const res = await apiFetch('/v1/library/resumes/{resumeId}', {
      method: 'delete',
      idempotencyKey: `delete-resume-${id}-${Date.now()}`,
      pathParams: { resumeId: id },
    });
    toast.push({
      title: res.ok ? (zh ? '删除任务已创建' : 'Deletion task created') : (zh ? '删除暂未接入（占位）' : 'Delete placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const removeJob = async (id: string) => {
    const res = await apiFetch('/v1/library/jobs/{jobId}', {
      method: 'delete',
      idempotencyKey: `delete-job-${id}-${Date.now()}`,
      pathParams: { jobId: id },
    });
    toast.push({
      title: res.ok ? (zh ? '删除任务已创建' : 'Deletion task created') : (zh ? '删除暂未接入（占位）' : 'Delete placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const listCard = (
    title: string,
    items: readonly { id: string; name: string; company: string; role: string; version: number; date: string }[],
    emptyTitle: string,
    onDelete: (id: string) => void,
    loading: boolean,
  ) => (
    <Card>
      <CardHeader title={title} />
      <CardBody className="space-y-3">
        {loading ? (
          <p className="mb-0 text-sm text-neutral-500" role="status">{zh ? '加载中…' : 'Loading…'}</p>
        ) : items.length === 0 ? (
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
              <Button variant="secondary" targetSize="min" aria-label={t('action.delete')} onClick={() => onDelete(item.id)}><IconTrash size={15} /></Button>
            </div>
          ))
        )}
      </CardBody>
    </Card>
  );

  const interviews = [
    { id: 'p-0001', name: zh ? '后端工程师面试训练' : 'Backend interview training', status: zh ? '活动中' : 'Active', date: '2026-08-01' },
    { id: 'p-0006', name: zh ? '全流程训练（已完成）' : 'Full training (completed)', status: zh ? '已完成' : 'Completed', date: '2026-07-24' },
  ] as const;

  return (
    <>
      <PageHeader kicker={t('library.kicker')} title={t('library.title')} />
      <Card>
        <CardBody>
          <Tabs
            initialId="resumes"
            items={[
              { id: 'resumes', label: t('library.tabResumes'), content: listCard(t('library.tabResumes'), resumes, t('library.emptyResumes'), removeResume, resumesState.loading) },
              { id: 'jobs', label: t('library.tabJobs'), content: listCard(t('library.tabJobs'), jobs, t('library.emptyJobs'), removeJob, jobsState.loading) },
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
