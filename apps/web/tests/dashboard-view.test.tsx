/** SCR-03 工作台组件测试：正常渲染、筛选、空态与状态徽标。 */

import { ToastProvider } from '@mgd/ui';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';

import { DashboardView } from '../src/components/dashboard-view.tsx';

const labels = {
  kicker: '工作台',
  title: '你的面试项目',
  desc: '从这里开始。',
  newProject: '新建面试',
  searchPlaceholder: '搜索项目或岗位',
  filterCompany: '公司',
  filterStatus: '状态',
  filterLanguage: '语言',
  allStatuses: '全部状态',
  allLanguages: '全部语言',
  clearFilters: '清除筛选',
  emptyTitle: '还没有面试项目',
  emptyDesc: '创建第一个项目。',
  emptyAction: '创建面试项目',
  noResultsTitle: '没有匹配的项目',
  noResultsDesc: '调整筛选条件。',
  status: '状态',
  nextAction: '下一步',
  round: '第 {n} 轮',
  actions: '操作',
  resume: '继续面试',
  viewReport: '查看报告',
  practice: '复盘练习',
  retry: '正式重试',
  duplicate: '复制项目',
  rename: '重命名',
  deleteProject: '删除项目',
  statsProjects: '项目总数',
  statsInProgress: '进行中',
  statsPassed: '已通过轮次',
  statsStreak: '连续训练天数',
} as const;

function renderDashboard() {
  return render(
    <ToastProvider>
      <DashboardView locale="zh-CN" labels={labels} />
    </ToastProvider>,
  );
}

describe('DashboardView（SCR-03）', () => {
  it('加载合成项目并渲染卡片与统计', async () => {
    renderDashboard();
    await waitFor(() => expect(screen.getByText('后端工程师面试训练（合成科技）')).toBeDefined());
    expect(screen.getByText('项目总数')).toBeDefined();
    expect(screen.getByText('本轮通过')).toBeDefined(); // 状态徽标
  });

  it('搜索过滤生效', async () => {
    const user = userEvent.setup();
    renderDashboard();
    await waitFor(() => expect(screen.getByText('后端工程师面试训练（合成科技）')).toBeDefined());
    await user.type(screen.getByPlaceholderText('搜索项目或岗位'), '产品经理');
    expect(screen.queryByText('后端工程师面试训练（合成科技）')).toBeNull();
    expect(screen.getByText('产品经理面试训练')).toBeDefined();
  });

  it('无结果时展示空态与清除筛选入口', async () => {
    const user = userEvent.setup();
    renderDashboard();
    await waitFor(() => expect(screen.getByText('后端工程师面试训练（合成科技）')).toBeDefined());
    await user.type(screen.getByPlaceholderText('搜索项目或岗位'), '不存在的项目名xyz');
    expect(screen.getByText('没有匹配的项目')).toBeDefined();
    await user.click(screen.getByText('清除筛选'));
    expect(screen.getByText('后端工程师面试训练（合成科技）')).toBeDefined();
  });
});
