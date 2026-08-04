/** SCR-11 完整报告：总体、轨迹、岗位匹配、雷达图+表格等价、逐题证据、训练计划。 */

'use client';

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  IconChart,
  IconCheck,
  IconDownload,
  IconRadar,
  IconRefresh,
  IconSparkle,
  IconTrash,
  PageHeader,
  Tint,
  useToast,
} from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { apiFetch } from '../../../../../../lib/api-fetch.ts';
import { isPlaceholder, useApiGet, useApiWrite } from '../../../../../../lib/api-hooks.ts';

const DIMS = ['professional_competence', 'problem_solving', 'communication', 'experience_evidence', 'behavioral_collaboration', 'learning_adaptability'] as const;

interface ReportPayload {
  readonly report: {
    readonly report_kind: string;
    readonly project_status: string;
    readonly overall?: { status: string; total: number; passed_rounds: number };
    readonly round_trajectory?: readonly { round: number; role: string; total: number; status: string }[];
    readonly job_match?: { available: boolean; match_percent: number; must_have: readonly string[]; plus: readonly string[] };
    readonly dimensions?: readonly { dimension: string; score: number }[];
    readonly evidence?: readonly {
      question: string;
      answer_summary: string;
      score: number;
      strengths: readonly string[];
      gaps: readonly string[];
      contradictions: readonly string[];
      suggestions: string;
    }[];
    readonly training_plan?: readonly string[];
    readonly communication?: { mode: string; note: string };
    readonly tools?: readonly { tool: string; events: number; note: string }[];
    readonly export_disclaimer: string;
  };
}

const DIM_LABELS: Readonly<Record<string, [string, string]>> = {
  professional_competence: ['专业能力', 'Professional'],
  problem_solving: ['问题解决', 'Problem solving'],
  communication: ['沟通表达', 'Communication'],
  experience_evidence: ['经历证据', 'Experience'],
  behavioral_collaboration: ['行为协作', 'Collaboration'],
  learning_adaptability: ['学习适应', 'Learning'],
};

function RadarChart({ dims, label }: { dims: readonly { dimension: string; score: number }[]; label: string }): ReactNode {
  const scores = DIMS.map((d) => dims.find((x) => x.dimension === d)?.score ?? 0);
  const size = 280;
  const cx = size / 2;
  const cy = size / 2;
  const r = size / 2 - 34;
  const angle = (i: number) => (Math.PI * 2 * i) / DIMS.length - Math.PI / 2;
  const point = (i: number, value: number) => {
    const rad = angle(i);
    return [cx + r * (value / 100) * Math.cos(rad), cy + r * (value / 100) * Math.sin(rad)];
  };
  const poly = scores.map((v, i) => point(i, v).join(',')).join(' ');
  const rings = [25, 50, 75, 100].map((v) => DIMS.map((_, i) => point(i, v).join(',')).join(' '));
  return (
    <svg viewBox={`0 0 ${size} ${size}`} className="mx-auto" role="img" aria-label={label}>
      {rings.map((ring) => (
        <polygon key={ring} points={ring} fill="none" stroke="var(--mgd-app-border-default)" strokeWidth={1} />
      ))}
      {DIMS.map((d, i) => {
        const [x, y] = point(i, 100);
        return <line key={d} x1={cx} y1={cy} x2={x} y2={y} stroke="var(--mgd-app-border-default)" strokeWidth={1} />;
      })}
      <polygon points={poly} fill="color-mix(in srgb, var(--mgd-app-brand-from) 18%, transparent)" stroke="var(--mgd-app-brand-from)" strokeWidth={2} strokeLinejoin="round" />
      {DIMS.map((d, i) => {
        const [x, y] = point(i, scores[i] ?? 0);
        return <circle key={d} cx={x} cy={y} r={3.5} fill="var(--mgd-app-brand-from)" />;
      })}
      {DIMS.map((d, i) => {
        const [x, y] = point(i, 118);
        return (
          <text key={d} x={x} y={y} textAnchor="middle" dominantBaseline="middle" fontSize={11} fill="var(--mgd-color-neutral-600)">
            {d.split('_')[0]}
          </text>
        );
      })}
    </svg>
  );
}

export default function ReportPage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; id: string }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const projectId = params.id;
  const state = useApiGet<ReportPayload>('/v1/projects/{projectId}/report', { pathParams: { projectId } });
  const { run: runExport, pending: exporting } = useApiWrite<{ task_id: string }>();

  const report = state.data?.report;
  const dimLabels = (d: string) => DIM_LABELS[d]?.[zh ? 0 : 1] ?? d;
  const scores = DIMS.map((d) => report?.dimensions?.find((x) => x.dimension === d)?.score ?? 0);
  const evidence = report?.evidence ?? [];
  const trainingPlan = report?.training_plan ?? [];
  const trajectory = report?.round_trajectory ?? [];
  const jobMatch = report?.job_match;

  const exportReport = async () => {
    const res = await runExport('/v1/projects/{projectId}/report/export', {
      method: 'post',
      idempotencyKey: `report-export-${projectId}-${Date.now()}`,
      pathParams: { projectId },
    });
    toast.push({
      title: res.ok ? (zh ? '导出任务已创建（含训练用途标记）' : 'Export task created (with training disclaimer)') : (zh ? '导出暂未接入（占位）' : 'Export placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const requestReview = async () => {
    const res = await apiFetch('/v1/projects/{projectId}/rounds/{sequence}/review', {
      method: 'post',
      idempotencyKey: `review-request-${projectId}-${Date.now()}`,
      pathParams: { projectId, sequence: 1 },
    });
    toast.push({
      title: res.ok ? (zh ? '复核已提交（仅一次）' : 'Review requested (once per attempt)') : (zh ? '复核暂未接入（占位）' : 'Review placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const deleteReport = async () => {
    const res = await apiFetch('/v1/deletion-tasks', {
      method: 'post',
      idempotencyKey: `delete-project-${projectId}-${Date.now()}`,
      body: { target_type: 'project', target_id: projectId },
    });
    toast.push({
      title: res.ok ? (zh ? '删除任务已创建' : 'Deletion task created') : (zh ? '删除暂未接入（占位）' : 'Delete placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  return (
    <>
      <PageHeader
        kicker={t('report.kicker')}
        title={t('report.title')}
        description={t('report.desc')}
        actions={
          <>
            <Button variant="secondary" onClick={exportReport} disabled={exporting} disabledReason={exporting ? t('report.export') : undefined}>
              <IconDownload size={16} />
              {t('report.export')}
            </Button>
            <Button variant="danger" onClick={deleteReport}>
              <IconTrash size={16} />
              {t('report.delete')}
            </Button>
          </>
        }
      />
      <p className="mb-5 inline-flex items-center gap-2 rounded-full bg-warning/10 px-4 py-1.5 text-sm text-warning-text">
        <IconSparkle size={14} />
        {report?.export_disclaimer ?? t('report.exportDisclaimer')}
      </p>

      {state.loading ? (
        <p className="mb-5 rounded-xl bg-surface-muted px-4 py-3 text-sm text-neutral-600" role="status" aria-live="polite">
          {zh ? '正在生成报告…' : 'Generating report…'}
        </p>
      ) : null}
      {state.failure !== undefined ? (
        <p className="mb-5 rounded-xl bg-warning/10 px-4 py-3 text-sm text-warning-text" role="status">
          {isPlaceholder(state.failure)
            ? (zh ? '报告服务暂未接入（占位），当前展示合成报告。' : 'Report service placeholder — synthetic report.')
            : (zh ? '报告加载失败，请重试。' : 'Failed to load report. Please retry.')}
        </p>
      ) : null}

      <div className="mgd-grid mgd-grid--3 mb-5">
        <Card className="mgd-card--brand p-5">
          <div className="text-sm text-neutral-600">{t('report.overall')}</div>
          <div className="mgd-stat-value mt-1 text-[var(--mgd-app-brand-ink)]">{report?.overall?.total ?? '—'} / 100</div>
          <div className="mt-2"><Tint tone="success">{report?.overall?.status === 'PASS' ? (zh ? '全部轮次通过' : 'All rounds passed') : (report?.project_status ?? '—')}</Tint></div>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('report.jobMatch')}</div>
          <div className="mgd-stat-value mt-1 text-success-text">{jobMatch?.available ? `${jobMatch.match_percent}%` : '—'}</div>
          <div className="mt-2 text-xs text-neutral-500">
            {jobMatch?.available ? (zh ? `必备 ${jobMatch.must_have.length} 项 · 加分 ${jobMatch.plus.length} 项` : `Must-have ${jobMatch.must_have.length} · Plus ${jobMatch.plus.length}`) : (zh ? '无 JD，不展示岗位匹配' : 'No JD — match hidden')}
          </div>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('report.roundTrajectory')}</div>
          <div className="mt-2 space-y-1 font-mono text-sm">
            {trajectory.length === 0 ? (
              <div className="text-neutral-500">—</div>
            ) : (
              trajectory.map((r) => (
                <div key={r.round} className="flex justify-between">
                  <span>R{r.round}</span>
                  <span className="text-success-text">{r.total}</span>
                </div>
              ))
            )}
          </div>
        </Card>
      </div>

      <div className="mgd-grid mgd-grid--sidebar mb-5">
        <Card>
          <CardHeader title={<span className="flex items-center gap-2"><IconRadar size={18} className="text-primary" />{t('report.radar')}</span>} description={t('report.tableEquivalent')} />
          <CardBody>
            <RadarChart dims={report?.dimensions ?? []} label={zh ? '六维雷达图（含文字/表格等价）' : 'Six-dimension radar chart (text/table equivalent provided)'} />
            <table className="mt-4 w-full border-collapse text-left text-sm">
              <caption className="sr-only">{t('report.tableEquivalent')}</caption>
              <thead>
                <tr className="border-b border-neutral-100 text-xs uppercase tracking-wide text-neutral-500">
                  <th className="py-2 pr-3 font-semibold">{zh ? '维度' : 'Dimension'}</th>
                  <th className="py-2 pr-3 font-semibold">{zh ? '得分' : 'Score'}</th>
                  <th className="py-2 font-semibold">{zh ? '达标' : 'Met'}</th>
                </tr>
              </thead>
              <tbody>
                {DIMS.map((d, i) => (
                  <tr key={d} className="border-b border-neutral-100 last:border-0">
                    <td className="py-2.5 pr-3 text-neutral-700">{dimLabels(d)}</td>
                    <td className="py-2.5 pr-3 font-mono text-neutral-900">{scores[i] ?? 0}</td>
                    <td className="py-2.5">{(scores[i] ?? 0) >= 60 ? <IconCheck size={15} className="text-success" /> : <span className="text-danger">✗</span>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardBody>
        </Card>

        <Card>
          <CardHeader title={t('report.trainingPlan')} />
          <CardBody>
            <ol className="mb-0 space-y-2.5 text-sm text-neutral-700">
              {trainingPlan.length === 0 ? (
                <li className="text-neutral-500">{zh ? '暂无训练计划' : 'No training plan'}</li>
              ) : (
                trainingPlan.map((item) => (
                  <li key={item} className="flex gap-2 rounded-xl bg-[var(--mgd-app-surface-muted)] px-3 py-2.5">
                    <IconChart size={15} className="mt-0.5 shrink-0 text-primary" />
                    {item}
                  </li>
                ))
              )}
            </ol>
            <a href={`/${locale}/projects/${projectId}/practice/pr-0001`} className="mgd-target-primary mt-4 inline-flex w-full items-center justify-center gap-2 rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-5 font-semibold text-neutral-800 shadow-[var(--mgd-app-shadow-sm)] hover:border-[var(--mgd-app-border-strong)]">
              {zh ? '开始练习' : 'Start practice'}
            </a>
            <Button variant="primary" className="mt-2 w-full" onClick={requestReview}>
              <IconRefresh size={15} />
              {t('report.reviewRequest')}
            </Button>
            <p className="mb-0 mt-2 text-center text-xs text-neutral-500">{t('report.reviewOnce')}</p>
          </CardBody>
        </Card>
      </div>

      <Card>
        <CardHeader title={t('report.evidence')} />
        <CardBody className="space-y-4">
          {evidence.length === 0 ? (
            <p className="mb-0 text-sm text-neutral-500">{zh ? '暂无逐题证据' : 'No per-question evidence'}</p>
          ) : (
            evidence.map((e, i) => (
              <article key={i} className="rounded-xl border border-neutral-100 p-5">
                <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                  <h3 className="m-0 text-sm font-semibold text-neutral-900">{e.question}</h3>
                  <span className="font-mono text-sm font-semibold text-neutral-900">{e.score}</span>
                </div>
                <p className="mb-3 text-sm text-neutral-600">{e.answer_summary}</p>
                <dl className="mb-0 grid gap-2 text-sm sm:grid-cols-2">
                  <div className="rounded-lg bg-success/5 px-3 py-2">
                    <dt className="text-xs font-semibold text-success-text">{t('report.strengths')}</dt>
                    <dd className="mb-0 mt-0.5 text-neutral-700">{e.strengths.join('；') || '—'}</dd>
                  </div>
                  <div className="rounded-lg bg-warning/5 px-3 py-2">
                    <dt className="text-xs font-semibold text-warning-text">{t('report.gaps')}</dt>
                    <dd className="mb-0 mt-0.5 text-neutral-700">{e.gaps.join('；') || '—'}</dd>
                  </div>
                  <div className="rounded-lg bg-info/5 px-3 py-2 sm:col-span-2">
                    <dt className="text-xs font-semibold text-info">{t('report.suggestions')}</dt>
                    <dd className="mb-0 mt-0.5 text-neutral-700">{e.suggestions || '—'}</dd>
                  </div>
                </dl>
              </article>
            ))
          )}
        </CardBody>
      </Card>
    </>
  );
}
