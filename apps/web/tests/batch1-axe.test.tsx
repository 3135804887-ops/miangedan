import { PROJECT_STATUSES } from '@mgd/domain-states';
import axe from 'axe-core';
import { describe, expect, it } from 'vitest';

import { AuthExperience } from '../src/features/batch1/auth-experience.tsx';
import { CreateProjectExperience } from '../src/features/batch1/create-project-experience.tsx';
import { DashboardExperience } from '../src/features/batch1/dashboard-experience.tsx';
import { LandingExperience } from '../src/features/batch1/landing-experience.tsx';
import { PlanExperience } from '../src/features/batch1/plan-experience.tsx';
import { PrecheckExperience } from '../src/features/batch1/precheck-experience.tsx';
import { ReviewExperience } from '../src/features/batch1/review-experience.tsx';
import { renderWithIntl, type TestLocale } from './test-intl.tsx';

const CASES = [
  ['SCR-01', <LandingExperience key="landing" mode="ready" />],
  ['SCR-02', <AuthExperience key="auth" mode="ready" />],
  ['SCR-03', <DashboardExperience key="dashboard" mode="ready" initialFilters={{ company: '', role: '', date: '', language: '', status: '' }} />],
  ['SCR-04', <CreateProjectExperience key="create" mode="ready" />],
  ['SCR-05', <ReviewExperience key="review" mode="ready" projectStatus={PROJECT_STATUSES[2]} />],
  ['SCR-06', <PlanExperience key="plan" mode="ready" projectStatus={PROJECT_STATUSES[5]} hasUnreadyRound={false} />],
  ['SCR-07', <PrecheckExperience key="precheck" mode="ready" insufficientQuota={false} otherDeviceActive={false} />],
] as const;

describe('批次 1 页面 axe serious/critical 门禁', () => {
  it.each(['zh-CN', 'en-US'] as const)('%s 下 SCR-01~07 为 0', async (locale: TestLocale) => {
    for (const [scrId, page] of CASES) {
      const { container, unmount } = renderWithIntl(page, locale);
      const result = await axe.run(container, { rules: { 'color-contrast': { enabled: false } } });
      const blocking = result.violations.filter(({ impact }) => impact === 'serious' || impact === 'critical');
      expect(blocking, `${scrId}: ${blocking.map(({ id }) => id).join(', ')}`).toEqual([]);
      unmount();
    }
  }, 30000);
});
