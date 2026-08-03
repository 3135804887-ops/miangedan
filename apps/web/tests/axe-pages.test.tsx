/**
 * TASK-094 页面级 axe-core 扫描（WCAG 2.2 AA 自动化层）。
 *
 * 覆盖 SCR-01 ~ SCR-17 全部 17 个页面路由的中文（zh-CN）与英文（en-US）渲染，
 * 断言 axe-core 无违规；jsdom 无法真实计算颜色，color-contrast 规则关闭并由
 * `pnpm tokens:check-contrast`（design-tokens 语义色板门禁）独立覆盖。
 * 页面为 Next.js 服务端组件，测试直接调用页面函数并 mock `next-intl/server`，
 * 使用与生产一致的 `@mgd/i18n` 消息资源生成标签。
 */

import { render, waitFor } from '@testing-library/react';
import { ToastProvider } from '@mgd/ui';
import { NextIntlClientProvider } from 'next-intl';
import axe from 'axe-core';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { loadMessages } from '../src/i18n/messages.ts';
import AdminPage from '../src/app/[locale]/(app)/admin/page.tsx';
import BillingPage from '../src/app/[locale]/(app)/billing/page.tsx';
import DashboardPage from '../src/app/[locale]/(app)/dashboard/page.tsx';
import LibraryPage from '../src/app/[locale]/(app)/library/page.tsx';
import OrgAggregatesPage from '../src/app/[locale]/(app)/org/[orgId]/aggregates/page.tsx';
import OrgAssignmentDetailPage from '../src/app/[locale]/(app)/org/[orgId]/assignments/[assignmentId]/page.tsx';
import OrgAssignmentsPage from '../src/app/[locale]/(app)/org/[orgId]/assignments/page.tsx';
import OrgCompletionPage from '../src/app/[locale]/(app)/org/[orgId]/completion/page.tsx';
import OrgMembersPage from '../src/app/[locale]/(app)/org/[orgId]/members/page.tsx';
import OrgPermissionsPage from '../src/app/[locale]/(app)/org/[orgId]/permissions/page.tsx';
import OrgSharesPage from '../src/app/[locale]/(app)/org/[orgId]/shares/page.tsx';
import NewProjectPage from '../src/app/[locale]/(app)/projects/new/page.tsx';
import PlanPage from '../src/app/[locale]/(app)/projects/[id]/plan/page.tsx';
import PracticePage from '../src/app/[locale]/(app)/projects/[id]/practice/[pid]/page.tsx';
import PrecheckPage from '../src/app/[locale]/(app)/projects/[id]/precheck/page.tsx';
import ReportPage from '../src/app/[locale]/(app)/projects/[id]/report/page.tsx';
import ReviewPage from '../src/app/[locale]/(app)/projects/[id]/review/page.tsx';
import ResultPage from '../src/app/[locale]/(app)/projects/[id]/rounds/[n]/result/page.tsx';
import SettingsPage from '../src/app/[locale]/(app)/settings/page.tsx';
import AuthPage from '../src/app/[locale]/(public)/auth/page.tsx';
import DemoPage from '../src/app/[locale]/(public)/demo/page.tsx';
import LandingPage from '../src/app/[locale]/(public)/page.tsx';
import SessionPage from '../src/app/[locale]/(room)/sessions/[id]/page.tsx';

vi.mock('next-intl/server', () => {
  const messageImports: Record<
    string,
    () => Promise<{ default: Record<string, unknown> }>
  > = {
    'zh-CN': () =>
      import('@mgd/i18n/messages/zh-CN/common.json', { with: { type: 'json' } }),
    'en-US': () =>
      import('@mgd/i18n/messages/en-US/common.json', { with: { type: 'json' } }),
  };
  return {
    setRequestLocale: () => undefined,
    getTranslations: async ({
      locale,
      namespace,
    }: {
      locale: string;
      namespace: string;
    }) => {
      const loader = messageImports[locale] ?? messageImports['zh-CN']!;
      const mod = await loader();
      const ns = (mod.default?.[namespace] ?? {}) as Record<string, unknown>;
      return (key: string, vars?: Record<string, string | number>) => {
        let value: unknown = ns;
        for (const part of key.split('.')) {
          if (value !== null && typeof value === 'object') {
            value = (value as Record<string, unknown>)[part];
          } else {
            value = undefined;
            break;
          }
        }
        let text = typeof value === 'string' ? value : key;
        if (vars !== undefined) {
          for (const [k, v] of Object.entries(vars)) {
            text = text.replaceAll(`{${k}}`, String(v));
          }
        }
        return text;
      };
    },
  };
});

type PageComponent = (props: {
  params: Promise<Record<string, string>>;
}) => Promise<ReactNode>;

const PAGES: ReadonlyArray<{
  readonly id: string;
  readonly scr: string;
  readonly page: PageComponent;
  readonly params: Record<string, string>;
  readonly locales: readonly ['zh-CN', 'en-US'];
}> = [
  { id: 'landing', scr: 'SCR-01', page: LandingPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
  { id: 'demo', scr: 'SCR-01', page: DemoPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
  { id: 'auth', scr: 'SCR-02', page: AuthPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
  { id: 'dashboard', scr: 'SCR-03', page: DashboardPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
  { id: 'new-project', scr: 'SCR-04', page: NewProjectPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
  { id: 'review', scr: 'SCR-05', page: ReviewPage as PageComponent, params: { locale: 'zh-CN', id: 'p-0003' }, locales: ['zh-CN', 'en-US'] },
  { id: 'plan', scr: 'SCR-06', page: PlanPage as PageComponent, params: { locale: 'zh-CN', id: 'p-0003' }, locales: ['zh-CN', 'en-US'] },
  { id: 'precheck', scr: 'SCR-07', page: PrecheckPage as PageComponent, params: { locale: 'zh-CN', id: 'p-0003' }, locales: ['zh-CN', 'en-US'] },
  { id: 'session', scr: 'SCR-08', page: SessionPage as PageComponent, params: { locale: 'zh-CN', id: 's-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'result', scr: 'SCR-10', page: ResultPage as PageComponent, params: { locale: 'zh-CN', id: 'p-0003', n: '1' }, locales: ['zh-CN', 'en-US'] },
  { id: 'report', scr: 'SCR-11', page: ReportPage as PageComponent, params: { locale: 'zh-CN', id: 'p-0003' }, locales: ['zh-CN', 'en-US'] },
  { id: 'practice', scr: 'SCR-12', page: PracticePage as PageComponent, params: { locale: 'zh-CN', id: 'p-0003', pid: 'pr-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'library', scr: 'SCR-13', page: LibraryPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
  { id: 'settings', scr: 'SCR-14', page: SettingsPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
  { id: 'billing', scr: 'SCR-15', page: BillingPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
  { id: 'org-aggregates', scr: 'SCR-16', page: OrgAggregatesPage as PageComponent, params: { locale: 'zh-CN', orgId: 'org-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'org-assignments', scr: 'SCR-16', page: OrgAssignmentsPage as PageComponent, params: { locale: 'zh-CN', orgId: 'org-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'org-assignment-detail', scr: 'SCR-16', page: OrgAssignmentDetailPage as PageComponent, params: { locale: 'zh-CN', orgId: 'org-0001', assignmentId: 'as-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'org-completion', scr: 'SCR-16', page: OrgCompletionPage as PageComponent, params: { locale: 'zh-CN', orgId: 'org-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'org-members', scr: 'SCR-16', page: OrgMembersPage as PageComponent, params: { locale: 'zh-CN', orgId: 'org-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'org-permissions', scr: 'SCR-16', page: OrgPermissionsPage as PageComponent, params: { locale: 'zh-CN', orgId: 'org-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'org-shares', scr: 'SCR-16', page: OrgSharesPage as PageComponent, params: { locale: 'zh-CN', orgId: 'org-0001' }, locales: ['zh-CN', 'en-US'] },
  { id: 'admin', scr: 'SCR-17', page: AdminPage as PageComponent, params: { locale: 'zh-CN' }, locales: ['zh-CN', 'en-US'] },
];

async function renderPage(
  page: PageComponent,
  params: Record<string, string>,
  locale: string,
): Promise<{ container: HTMLElement; unmount: () => void }> {
  const element = await page({ params: Promise.resolve({ ...params, locale }) } as never);
  const messages = await loadMessages(locale as 'zh-CN' | 'en-US');
  const view = render(
    <NextIntlClientProvider locale={locale} messages={messages}>
      <ToastProvider>{element}</ToastProvider>
    </NextIntlClientProvider>,
  );
  const container = view.container;
  await waitFor(() => {
    expect(container.textContent?.trim().length ?? 0).toBeGreaterThan(0);
  });
  // 等待客户端 effect（apiFetch/useState）完成一轮刷新后再扫描。
  await new Promise((resolve) => setTimeout(resolve, 30));
  return { container, unmount: view.unmount };
}

async function expectNoAxeViolations(container: HTMLElement, label: string) {
  const results = await axe.run(container, {
    rules: {
      // jsdom 无法渲染真实像素，颜色对比度由 tokens:check-contrast 门禁覆盖。
      'color-contrast': { enabled: false },
    },
  });
  const summary = results.violations.map(
    (v) => `${v.id}（${v.impact ?? 'unknown'}）: ${v.nodes.length} 节点`,
  );
  expect(summary, `${label} axe 违规`).toEqual([]);
}

describe('TASK-094 页面级 axe 扫描（WCAG 2.2 AA）', () => {
  for (const entry of PAGES) {
    for (const locale of entry.locales) {
      it(`${entry.scr} ${entry.id}（${locale}）无 axe 违规`, async () => {
        const { container, unmount } = await renderPage(entry.page, entry.params, locale);
        await expectNoAxeViolations(container, `${entry.scr} ${entry.id} ${locale}`);
        unmount();
      });
    }
  }

  it('SCR-08 实时房间具备 WCAG 2.2 健壮性语义（aria-live/字幕与故障提示）', async () => {
    const { container, unmount } = await renderPage(
      SessionPage as PageComponent,
      { locale: 'zh-CN', id: 's-0001' },
      'zh-CN',
    );
    const liveRegions = container.querySelectorAll('[aria-live]');
    expect(liveRegions.length).toBeGreaterThanOrEqual(1);
    expect(container.querySelector('svg[role="img"]') ?? container.querySelector('[role="img"]')).not.toBeNull();
    // 打开故障覆盖层后必须出现 alertdialog 与 role=alert 描述。
    const trigger = Array.from(container.querySelectorAll('button')).find((b) =>
      b.textContent?.includes('系统故障暂停'),
    );
    expect(trigger).toBeDefined();
    trigger?.click();
    await waitFor(() => {
      expect(container.querySelector('[role="alertdialog"]')).not.toBeNull();
    });
    expect(container.querySelectorAll('[role="alert"]').length).toBeGreaterThanOrEqual(1);
    unmount();
  });

  it('SCR-11 报告页雷达图有文字等价与图表标签', async () => {
    const { container, unmount } = await renderPage(
      ReportPage as PageComponent,
      { locale: 'zh-CN', id: 'p-0003' },
      'zh-CN',
    );
    const svg = container.querySelector('svg[role="img"]');
    expect(svg).not.toBeNull();
    expect(svg?.getAttribute('aria-label')?.length ?? 0).toBeGreaterThan(0);
    const tableEquivalent = Array.from(container.querySelectorAll('table, ul, ol')).some(
      (el) => el.textContent?.includes('专业能力') || el.textContent?.includes('professional'),
    );
    expect(tableEquivalent).toBe(true);
    unmount();
  });
});
