import { PROJECT_STATUSES } from '@mgd/domain-states';
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { PageStateBoundary } from '../src/components/page-state-boundary.tsx';
import { DashboardExperience } from '../src/features/batch1/dashboard-experience.tsx';
import { LandingExperience } from '../src/features/batch1/landing-experience.tsx';
import { PlanExperience } from '../src/features/batch1/plan-experience.tsx';
import { ReviewExperience } from '../src/features/batch1/review-experience.tsx';
import { renderWithIntl } from './test-intl.tsx';

describe('批次 1 页面状态与契约', () => {
  it.each(['empty', 'loading', 'error', 'forbidden', 'recovering'] as const)(
    '统一边界渲染 %s 状态',
    (mode) => {
      const { container } = renderWithIntl(
        <PageStateBoundary
          mode={mode}
          regionLabel="SCR-TEST"
          emptyReason="synthetic empty"
          forbiddenPermission="synthetic permission"
          recoveryPoint="synthetic recovery"
        >
          <p>ready content</p>
        </PageStateBoundary>,
      );
      expect(container.querySelector(`[data-mgd-state-view="${mode}"]`)).not.toBeNull();
      if (mode === 'error') {
        expect(container.querySelectorAll('[data-mgd-error-facet]')).toHaveLength(5);
      }
    },
  );

  it('SCR-01 明示年龄和能力边界，不包含录用预测或投递承诺', () => {
    renderWithIntl(<LandingExperience mode="ready" />);
    expect(screen.getByText(/未满 16 岁/)).toBeTruthy();
    expect(screen.getByText(/不预测录用结果/)).toBeTruthy();
    expect(screen.getByRole('link', { name: /登录后上传材料/ }).getAttribute('href')).toContain('returnTo');
  });

  it('SCR-03 渲染 15 项领域状态并恢复 URL 初始筛选', () => {
    const { container } = renderWithIntl(
      <DashboardExperience
        mode="ready"
        initialFilters={{ company: 'Northstar', role: '', date: '', language: '', status: '' }}
      />,
    );
    for (const status of PROJECT_STATUSES) {
      expect(container.querySelector(`[data-mgd-status="${status}"]`)).not.toBeNull();
    }
    expect(screen.getByLabelText('公司').getAttribute('value')).toBe('Northstar');
  });

  it('SCR-05 三种领域状态视图与敏感字段类别脱敏', () => {
    const parsing = renderWithIntl(<ReviewExperience mode="ready" projectStatus={PROJECT_STATUSES[1]} />);
    expect(screen.getByText(/60 秒内完成/)).toBeTruthy();
    parsing.unmount();

    const failed = renderWithIntl(<ReviewExperience mode="ready" projectStatus={PROJECT_STATUSES[3]} />);
    expect(screen.getByRole('button', { name: /重试失败步骤/ })).toBeTruthy();
    failed.unmount();

    const review = renderWithIntl(<ReviewExperience mode="ready" projectStatus={PROJECT_STATUSES[2]} />);
    expect(screen.getByText(/电话、邮箱、证件号/)).toBeTruthy();
    expect(screen.queryByText(/138\d{8}/)).toBeNull();
    expect(screen.getByRole('button', { name: /生成面试计划/ }).getAttribute('aria-disabled')).toBe('true');
    review.unmount();
  });

  it('SCR-06 只读规则区没有编辑控件，未就绪轮阻止确认', () => {
    const { container } = renderWithIntl(
      <PlanExperience mode="ready" projectStatus={PROJECT_STATUSES[5]} hasUnreadyRound />,
    );
    const readonlyTitle = screen.getByText('统一规则（只读）');
    const readonlyRegion = readonlyTitle.closest('.mgd-disclosure-note');
    expect(readonlyRegion?.querySelector('button,input,select,textarea')).toBeNull();
    expect(container.querySelector('[data-mgd-control="plan-confirm"]')?.getAttribute('aria-disabled')).toBe('true');
    expect(container.textContent).not.toContain('标准答案：');
  });
});
