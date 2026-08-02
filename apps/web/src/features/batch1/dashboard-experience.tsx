'use client';

import { PROJECT_STATUSES, type ProjectStatus } from '@mgd/domain-states';
import { Button, StatusBadge } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { PageStateBoundary, type PreviewState } from '../../components/page-state-boundary.tsx';
import { RouteShell } from '../../components/route-shell.tsx';
import { Link } from '../../i18n/navigation.ts';

export interface DashboardFilters {
  readonly company: string;
  readonly role: string;
  readonly date: string;
  readonly language: string;
  readonly status: string;
}

const PROJECTS: ReadonlyArray<{
  readonly id: string;
  readonly company: string;
  readonly role: string;
  readonly round: number;
  readonly status: ProjectStatus;
  readonly activeElsewhere?: boolean;
}> = [
  { id: 'synthetic-1', company: 'Northstar Labs', role: 'Product Operations', round: 2, status: PROJECT_STATUSES[5] },
  { id: 'synthetic-2', company: 'Atlas Studio', role: 'UX Researcher', round: 1, status: PROJECT_STATUSES[7], activeElsewhere: true },
  { id: 'synthetic-3', company: 'Cedar Systems', role: 'Frontend Engineer', round: 3, status: PROJECT_STATUSES[10] },
];

export function DashboardExperience({
  mode,
  initialFilters,
}: {
  readonly mode: PreviewState;
  readonly initialFilters: DashboardFilters;
}): ReactNode {
  const t = useTranslations('batch1');
  const [filters, setFilters] = useState(initialFilters);

  function updateFilter(key: keyof DashboardFilters, value: string): void {
    const next = { ...filters, [key]: value };
    setFilters(next);
    const query = new URLSearchParams();
    for (const [filterKey, filterValue] of Object.entries(next)) {
      if (filterValue.length > 0) query.set(filterKey, filterValue);
    }
    window.history.replaceState(null, '', `${window.location.pathname}${query.size > 0 ? `?${query}` : ''}`);
  }

  const visibleProjects = PROJECTS.filter((project) => {
    if (filters.company && !project.company.toLowerCase().includes(filters.company.toLowerCase())) return false;
    if (filters.role && !project.role.toLowerCase().includes(filters.role.toLowerCase())) return false;
    if (filters.status && project.status !== filters.status) return false;
    return true;
  });

  return (
    <RouteShell scrId="SCR-03" title={t('dashboard.title')} notice={t('dashboard.lead')}>
      <PageStateBoundary
        mode={mode}
        regionLabel="SCR-03-dashboard"
        emptyReason={t('dashboard.empty')}
        forbiddenPermission={t('dashboard.forbidden')}
        recoveryPoint={t('dashboard.recovering')}
      >
        <div className="mgd-page-toolbar">
          <p>{t('shared.mockNotice')}</p>
          <Link className="mgd-link-button mgd-link-button--primary" href="/projects/new">
            {t('dashboard.new')}
          </Link>
        </div>

        <fieldset className="mgd-filter-bar">
          <legend>{t('dashboard.filters')}</legend>
          <label>
            {t('dashboard.company')}
            <input value={filters.company} onChange={(event) => updateFilter('company', event.currentTarget.value)} />
          </label>
          <label>
            {t('dashboard.role')}
            <input value={filters.role} onChange={(event) => updateFilter('role', event.currentTarget.value)} />
          </label>
          <label>
            {t('dashboard.date')}
            <input type="date" value={filters.date} onChange={(event) => updateFilter('date', event.currentTarget.value)} />
          </label>
          <label>
            {t('dashboard.language')}
            <select value={filters.language} onChange={(event) => updateFilter('language', event.currentTarget.value)}>
              <option value="">{t('dashboard.all')}</option>
              <option value="zh-CN">中文</option>
              <option value="en-US">English</option>
            </select>
          </label>
          <label>
            {t('dashboard.status')}
            <select value={filters.status} onChange={(event) => updateFilter('status', event.currentTarget.value)}>
              <option value="">{t('dashboard.all')}</option>
              {PROJECT_STATUSES.map((status) => (
                <option key={status} value={status}>{t(`dashboard.statusLabels.${status}`)}</option>
              ))}
            </select>
          </label>
          <Button
            controlId="dashboard-clear-filters"
            onClick={() => {
              const cleared = { company: '', role: '', date: '', language: '', status: '' };
              setFilters(cleared);
              window.history.replaceState(null, '', window.location.pathname);
            }}
          >
            {t('dashboard.clear')}
          </Button>
        </fieldset>

        {visibleProjects.length === 0 ? (
          <section className="mgd-local-empty" aria-live="polite">
            <h2>{t('shared.empty')}</h2>
            <Button controlId="dashboard-empty-clear" onClick={() => window.history.replaceState(null, '', window.location.pathname)}>
              {t('dashboard.clear')}
            </Button>
          </section>
        ) : (
          <div className="mgd-project-grid">
            {visibleProjects.map((project) => (
              <article key={project.id} className="mgd-project-card">
                <div>
                  <p className="mgd-kicker">{project.company}</p>
                  <h2>{project.role}</h2>
                  <p>{t('dashboard.round', { round: project.round })}</p>
                </div>
                <StatusBadge
                  status={project.status}
                  label={t(`dashboard.statusLabels.${project.status}`)}
                  actionHint={t('dashboard.actionHint')}
                />
                {project.activeElsewhere ? (
                  <div className="mgd-device-alert">
                    <strong>{t('dashboard.activeDevice')}</strong>
                    <Button controlId={`dashboard-transfer-${project.id}`}>{t('dashboard.transfer')}</Button>
                  </div>
                ) : null}
                <Link className="mgd-link-button" href={`/projects/${project.id}/plan`}>
                  {t('dashboard.next')}
                </Link>
              </article>
            ))}
          </div>
        )}

        <details className="mgd-status-legend">
          <summary>{t('dashboard.legend')}</summary>
          <div className="mgd-status-legend__grid">
            {PROJECT_STATUSES.map((status) => (
              <StatusBadge
                key={status}
                status={status}
                label={t(`dashboard.statusLabels.${status}`)}
                actionHint={t('dashboard.actionHint')}
              />
            ))}
          </div>
        </details>
      </PageStateBoundary>
    </RouteShell>
  );
}
