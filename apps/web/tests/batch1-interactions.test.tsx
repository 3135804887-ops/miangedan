import { PROJECT_STATUSES } from '@mgd/domain-states';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';

import { AuthExperience } from '../src/features/batch1/auth-experience.tsx';
import { CreateProjectExperience } from '../src/features/batch1/create-project-experience.tsx';
import { PlanExperience } from '../src/features/batch1/plan-experience.tsx';
import { PrecheckExperience } from '../src/features/batch1/precheck-experience.tsx';
import { ReviewExperience } from '../src/features/batch1/review-experience.tsx';
import { renderWithIntl } from './test-intl.tsx';

describe('批次 1 异常与幂等交互', () => {
  it('SCR-02 验证码错误保留邮箱，第三方失败提供邮箱路径且没有手机号字段', async () => {
    const user = userEvent.setup();
    renderWithIntl(<AuthExperience mode="ready" />);
    await user.click(screen.getByRole('button', { name: /使用 Google 继续/ }));
    expect(screen.getByRole('button', { name: /改用邮箱验证码/ })).toBeTruthy();
    const email = screen.getByLabelText(/邮箱地址/);
    await user.type(email, 'synthetic@example.com');
    await user.click(screen.getByRole('button', { name: /发送验证码/ }));
    await user.type(screen.getByLabelText(/6 位验证码/), '000000');
    await user.click(screen.getByRole('button', { name: /验证并继续/ }));
    expect(screen.getByText(/还可尝试 4 次/)).toBeTruthy();
    expect(email.getAttribute('value')).toBe('synthetic@example.com');
    expect(screen.queryByLabelText(/手机号/)).toBeNull();
  });

  it('SCR-04 文件拒绝后完整保留 JD', async () => {
    const user = userEvent.setup();
    renderWithIntl(<CreateProjectExperience mode="ready" />);
    const jd = screen.getByRole('textbox', { name: '完整岗位描述（JD）' });
    await user.type(jd, 'synthetic JD body retained');
    const fileInput = document.querySelector('input[type="file"]');
    const oversized = new File([new Uint8Array(10 * 1024 * 1024 + 1)], 'resume.pdf', { type: 'application/pdf' });
    if (fileInput instanceof HTMLInputElement) fireEvent.change(fileInput, { target: { files: [oversized] } });
    expect(screen.getByRole('alert').textContent).toContain('JD 文本已完整保留');
    expect((jd as HTMLTextAreaElement).value).toBe('synthetic JD body retained');
  });

  it('SCR-05 缺失影响与材料确认必须分别同意', async () => {
    const user = userEvent.setup();
    renderWithIntl(<ReviewExperience mode="ready" projectStatus={PROJECT_STATUSES[2]} />);
    const generate = screen.getByRole('button', { name: /生成面试计划/ });
    await user.click(screen.getByLabelText(/理解该降级模式/));
    expect(generate.getAttribute('aria-disabled')).toBe('true');
    await user.click(screen.getByLabelText(/逐字段确认材料/));
    expect(generate.getAttribute('aria-disabled')).toBeNull();
  });

  it('SCR-06 九项便利设置默认关闭并彼此独立', async () => {
    const user = userEvent.setup();
    renderWithIntl(<PlanExperience mode="ready" projectStatus={PROJECT_STATUSES[5]} hasUnreadyRound={false} />);
    const switches = screen.getAllByRole('switch').filter((control) => control.getAttribute('data-mgd-control')?.startsWith('plan-accommodation-'));
    expect(switches).toHaveLength(9);
    expect(switches.every((control) => control.getAttribute('aria-checked') === 'false')).toBe(true);
    await user.click(switches[0] as HTMLElement);
    expect(switches[0]?.getAttribute('aria-checked')).toBe('true');
    expect(switches.slice(1).every((control) => control.getAttribute('aria-checked') === 'false')).toBe(true);
  });

  it('SCR-07 同一检查项重复触发只产生一次请求', async () => {
    const { container } = renderWithIntl(
      <PrecheckExperience mode="ready" insufficientQuota otherDeviceActive={false} />,
    );
    const retry = screen.getAllByRole('button', { name: /单项重试/ })[0];
    if (retry !== undefined) {
      fireEvent.click(retry);
      fireEvent.click(retry);
    }
    await waitFor(() => {
      expect(container.querySelector('[data-request-count="camera"]')?.textContent).toContain('1');
    });
    expect(screen.getByRole('button', { name: /开始本轮/ }).getAttribute('aria-disabled')).toBe('true');
    expect(screen.getByRole('link', { name: /购买额度/ })).toBeTruthy();
  });
});
