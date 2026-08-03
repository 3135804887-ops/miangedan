'use client';

import {
  Button,
  Card,
  EmptyState,
  IconArrowRight,
  IconDashboard,
  IconEdit,
  IconPlay,
  IconSearch,
  IconTrash,
  PageHeader,
  StatCard,
  Tint,
  useToast,
} from '@mgd/ui';
import { useEffect, useMemo, useState } from 'react';
import { PROJECT_STATUSES } from '@mgd/domain-states';

import { apiFetch } from '../lib/api-fetch.ts';
import { useMockReady } from '../lib/use-mock-ready.ts';
import type { MockProject } from '../mocks/data.ts';
import { StatusTint } from './status-tint.tsx';

interface Labels {
  readonly kicker: string;
  readonly title: string;
  readonly desc: string;
  readonly newProject: string;
  readonly searchPlaceholder: string;
  readonly filterCompany: string;
  readonly filterStatus: string;
  readonly filterLanguage: string;
  readonly allStatuses: string;
  readonly allLanguages: string;
  readonly clearFilters: string;
  readonly emptyTitle: string;
  readonly emptyDesc: string;
  readonly emptyAction: string;
  readonly noResultsTitle: string;
  readonly noResultsDesc: string;
  readonly status: string;
  readonly nextAction: string;
  readonly round: string;
  readonly actions: string;
  readonly resume: string;
  readonly viewReport: string;
  readonly practice: string;
  readonly retry: string;
  readonly duplicate: string;
  readonly rename: string;
  readonly deleteProject: string;
  readonly statsProjects: string;
  readonly statsInProgress: string;
  readonly statsPassed: string;
  readonly statsStreak: string;
  readonly loadFailedTitle: string;
  readonly loadFailedDesc: string;
}

const STATUS_OPTIONS = [...PROJECT_STATUSES];

const [
  _S_DRAFT,
  _S_PARSING,
  S_MATERIAL_REVIEW,
  S_PARSE_FAILED,
  _S_PLAN_GENERATING,
  S_PLAN_REVIEW,
  S_PLAN_FAILED,
  S_READY,
  S_IN_SESSION,
  S_SCORING,
  S_ROUND_PASSED,
  S_ROUND_FAILED,
  _S_PRACTICING,
  S_EVALUATION_INCOMPLETE,
  S_COMPLETED,
] = PROJECT_STATUSES;
const ACTIVE_STATUSES: ReadonlySet<string> = new Set([S_IN_SESSION, S_READY, S_SCORING]);
const PASSED_STATUSES: ReadonlySet<string> = new Set([S_ROUND_PASSED, S_COMPLETED]);

export function DashboardView({
  locale,
  labels,
}: {
  readonly locale: 'zh-CN' | 'en-US';
  readonly labels: Labels;
}): React.ReactNode {
  const toast = useToast();
  const mocksReady = useMockReady();
  const [projects, setProjects] = useState<readonly MockProject[]>([]);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const [query, setQuery] = useState('');
  const [company, setCompany] = useState('');
  const [status, setStatus] = useState('');
  const [language, setLanguage] = useState('');

  const load = async () => {
    setLoading(true);
    setFailed(false);
    try {
      const result = await apiFetch<{ items: readonly MockProject[] }>('/v1/projects', { method: 'get' });
      if (!result.ok) throw new Error('load failed');
      setProjects(result.data.items);
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!mocksReady) return;
    void load();
  }, [mocksReady]);

  const companies = useMemo(
    () => [...new Set(projects.map((p) => p.company).filter(Boolean))].sort(),
    [projects],
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return projects.filter((p) => {
      if (q !== '' && !`${p.name} ${p.job_title} ${p.company}`.toLowerCase().includes(q)) return false;
      if (company !== '' && p.company !== company) return false;
      if (status !== '' && p.status !== status) return false;
      if (language !== '' && p.interview_language !== language) return false;
      return true;
    });
  }, [projects, query, company, status, language]);

  const stats = useMemo(
    () => ({
      total: projects.length,
      inProgress: projects.filter((p) => ACTIVE_STATUSES.has(p.status)).length,
      passed: projects.filter((p) => PASSED_STATUSES.has(p.status)).length,
      streak: 3,
    }),
    [projects],
  );

  const nextHref = (p: MockProject): string => {
    const base = `/${locale}`;
    switch (p.status) {
      case S_MATERIAL_REVIEW:
      case S_PARSE_FAILED:
        return `${base}/projects/${p.project_id}/review`;
      case S_PLAN_REVIEW:
      case S_PLAN_FAILED:
        return `${base}/projects/${p.project_id}/plan`;
      case S_READY:
        return `${base}/projects/${p.project_id}/precheck`;
      case S_IN_SESSION:
      case S_SCORING:
        return `${base}/sessions/s-0001`;
      case S_ROUND_PASSED:
        return `${base}/projects/${p.project_id}/rounds/${p.current_round_sequence}/result`;
      case S_ROUND_FAILED:
        return `${base}/projects/${p.project_id}/rounds/${p.current_round_sequence}/result`;
      case S_EVALUATION_INCOMPLETE:
        return `${base}/projects/${p.project_id}/precheck`;
      case S_COMPLETED:
        return `${base}/projects/${p.project_id}/report`;
      default:
        return `${base}/projects/new`;
    }
  };

  return (
    <>
      <PageHeader
        kicker={labels.kicker}
        title={labels.title}
        description={labels.desc}
        actions={
          <a
            href={`/${locale}/projects/new`}
            className="mgd-target-primary inline-flex items-center gap-2 rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-5 font-semibold text-white shadow-[var(--mgd-app-shadow-brand)] transition hover:brightness-105"
          >
            <IconPlay size={17} />
            {labels.newProject}
          </a>
        }
      />

      <div className="mgd-grid mgd-grid--4 mb-6">
        <StatCard label={labels.statsProjects} value={stats.total} tone="brand" />
        <StatCard label={labels.statsInProgress} value={stats.inProgress} tone="info" />
        <StatCard label={labels.statsPassed} value={stats.passed} tone="success" />
        <StatCard label={labels.statsStreak} value={stats.streak} tone="warning" />
      </div>

      <Card className="mb-5 p-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <label className="relative lg:col-span-2">
            <span className="sr-only">{labels.searchPlaceholder}</span>
            <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400">
              <IconSearch size={18} />
            </span>
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={labels.searchPlaceholder}
              className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface pl-10 pr-3 text-sm focus:border-primary"
            />
          </label>
          <label>
            <span className="sr-only">{labels.filterCompany}</span>
            <select value={company} onChange={(e) => setCompany(e.target.value)} className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-3 text-sm">
              <option value="">{labels.filterCompany} · {locale === 'zh-CN' ? '全部' : 'All'}</option>
              {companies.map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </label>
          <label>
            <span className="sr-only">{labels.filterStatus}</span>
            <select value={status} onChange={(e) => setStatus(e.target.value)} className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-3 text-sm">
              <option value="">{labels.allStatuses}</option>
              {STATUS_OPTIONS.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </label>
          <label>
            <span className="sr-only">{labels.filterLanguage}</span>
            <select value={language} onChange={(e) => setLanguage(e.target.value)} className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-3 text-sm">
              <option value="">{labels.allLanguages}</option>
              <option value="zh-CN">中文</option>
              <option value="en-US">English</option>
            </select>
          </label>
        </div>
        {(query !== '' || company !== '' || status !== '' || language !== '') ? (
          <button type="button" className="mt-3 cursor-pointer text-sm text-primary hover:underline" onClick={() => { setQuery(''); setCompany(''); setStatus(''); setLanguage(''); }}>
            {labels.clearFilters}
          </button>
        ) : null}
      </Card>

      {loading ? (
        <div className="grid gap-4 lg:grid-cols-2">
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="mgd-card p-5">
              <div className="mgd-skeleton mb-3 h-5 w-2/3" />
              <div className="mgd-skeleton mb-2 h-4 w-1/2" />
              <div className="mgd-skeleton h-4 w-1/3" />
            </div>
          ))}
        </div>
      ) : failed ? (
        <Card>
          <EmptyState
            icon={<IconDashboard size={26} />}
            title={labels.loadFailedTitle}
            description={labels.loadFailedDesc}
            action={<Button variant="primary" onClick={() => void load()}>{labels.actions}</Button>}
          />
        </Card>
      ) : filtered.length === 0 ? (
        <Card>
          <EmptyState
            icon={<IconDashboard size={26} />}
            title={projects.length === 0 ? labels.emptyTitle : labels.noResultsTitle}
            description={projects.length === 0 ? labels.emptyDesc : labels.noResultsDesc}
            action={
              projects.length === 0 ? (
                <a href={`/${locale}/projects/new`} className="mgd-target-primary inline-flex items-center gap-2 rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-5 font-semibold text-white shadow-[var(--mgd-app-shadow-brand)]">
                  {labels.emptyAction}
                </a>
              ) : undefined
            }
          />
        </Card>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {filtered.map((p) => (
            <Card key={p.project_id} hoverable className="flex flex-col p-5">
              <div className="mb-3 flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0">
                  <h2 className="mb-1 truncate text-base font-semibold text-neutral-900">{p.name}</h2>
                  <p className="mb-0 text-sm text-neutral-600">
                    {p.company !== '' ? `${p.company} · ` : ''}{p.job_title}
                  </p>
                </div>
                <StatusTint status={p.status} locale={locale} />
              </div>
              <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-neutral-500">
                <span>
                  {labels.round.replace('{n}', String(p.current_round_sequence))} / {p.total_rounds}
                </span>
                <span>{p.interview_language}</span>
                <span>{p.updated_at.slice(0, 10)}</span>
              </div>
              <div className="mb-4 flex items-center gap-2 rounded-xl bg-[var(--mgd-app-surface-muted)] px-3 py-2.5 text-sm">
                <Tint tone="brand">{labels.nextAction}</Tint>
                <span className="text-neutral-700">{p.next_action}</span>
              </div>
              <div className="mt-auto flex flex-wrap items-center gap-2">
                <a
                  href={nextHref(p)}
                  className="mgd-target-primary inline-flex flex-1 items-center justify-center gap-2 rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-4 text-sm font-semibold text-white shadow-[var(--mgd-app-shadow-brand)] transition hover:brightness-105"
                >
                  {labels.resume}
                  <IconArrowRight size={15} />
                </a>
                <Button
                  variant="secondary"
                  targetSize="min"
                  aria-label={labels.rename}
                  onClick={() => toast.push({ title: labels.rename, tone: 'info' })}
                >
                  <IconEdit size={16} />
                </Button>
                <Button
                  variant="secondary"
                  targetSize="min"
                  aria-label={labels.deleteProject}
                  onClick={() => toast.push({ title: labels.deleteProject, tone: 'danger' })}
                >
                  <IconTrash size={16} />
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  );
}
