/** SCR-03 工作台：项目卡片、统计、筛选与操作（状态机枚举徽标）。 */

import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { DashboardView } from '../../../../components/dashboard-view.tsx';

export default async function DashboardPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  return (
    <DashboardView
      locale={locale as 'zh-CN' | 'en-US'}
      labels={{
        kicker: t('dashboard.kicker'),
        title: t('dashboard.title'),
        desc: t('dashboard.desc'),
        newProject: t('dashboard.newProject'),
        searchPlaceholder: t('dashboard.searchPlaceholder'),
        filterCompany: t('dashboard.filterCompany'),
        filterStatus: t('dashboard.filterStatus'),
        filterLanguage: t('dashboard.filterLanguage'),
        allStatuses: t('dashboard.allStatuses'),
        allLanguages: t('dashboard.allLanguages'),
        clearFilters: t('dashboard.clearFilters'),
        emptyTitle: t('dashboard.emptyTitle'),
        emptyDesc: t('dashboard.emptyDesc'),
        emptyAction: t('dashboard.emptyAction'),
        noResultsTitle: t('dashboard.noResultsTitle'),
        noResultsDesc: t('dashboard.noResultsDesc'),
        status: t('dashboard.status'),
        nextAction: t('dashboard.nextAction'),
        round: t('dashboard.round', { n: '{n}' }),
        actions: t('dashboard.actions'),
        resume: t('dashboard.resume'),
        viewReport: t('dashboard.viewReport'),
        practice: t('dashboard.practice'),
        retry: t('dashboard.retry'),
        duplicate: t('dashboard.duplicate'),
        rename: t('dashboard.rename'),
        deleteProject: t('dashboard.deleteProject'),
        statsProjects: t('dashboard.statsProjects'),
        statsInProgress: t('dashboard.statsInProgress'),
        statsPassed: t('dashboard.statsPassed'),
        statsStreak: t('dashboard.statsStreak'),
        loadFailedTitle: t('dashboard.loadFailedTitle'),
        loadFailedDesc: t('dashboard.loadFailedDesc'),
      }}
    />
  );
}
