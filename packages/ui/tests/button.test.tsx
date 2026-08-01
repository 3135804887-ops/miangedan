/**
 * Button 七态断言（需求 G5 第 4~8 条、DESIGN-SYSTEM 第 8 节）。
 * 正常路径 + 异常路径（disabled 缺原因、error 态）齐备。
 */

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { Button } from '../src/primitives/button.tsx';

describe('Button 七态', () => {
  it('default：渲染文案并携带控件标识', () => {
    render(<Button controlId="submit-answer">提交</Button>);

    const button = screen.getByRole('button', { name: '提交' });
    expect(button.getAttribute('data-mgd-control')).toBe('submit-answer');
    expect(button.getAttribute('data-mgd-state')).toBe('default');
    expect(button.getAttribute('aria-disabled')).toBeNull();
  });

  it('disabled：不可聚焦、aria-disabled 为 true，并渲染禁用原因', () => {
    render(
      <Button controlId="start-round" disabled disabledReason="本轮量表未就绪">
        开始本轮
      </Button>,
    );

    const button = screen.getByRole('button', { name: /开始本轮/ });
    expect(button.getAttribute('aria-disabled')).toBe('true');
    expect(button.tabIndex).toBe(-1);
    expect(screen.getByText('本轮量表未就绪')).toBeTruthy();
    expect(button.getAttribute('aria-describedby')).toContain('start-round-disabled-reason');
  });

  it('disabled 缺少原因时开发期抛错，避免静默违反 DESIGN-SYSTEM 第 8 节', () => {
    expect(() => render(<Button controlId="no-reason" disabled>确认</Button>)).toThrow(
      /disabledReason/,
    );
  });

  it('loading：输出 aria-busy 与忙碌文案，并移出 Tab 序防重复提交', () => {
    render(
      <Button controlId="pay-order" loading busyLabel="正在提交">
        支付
      </Button>,
    );

    const button = screen.getByRole('button', { name: /支付/ });
    expect(button.getAttribute('aria-busy')).toBe('true');
    expect(button.getAttribute('data-mgd-state')).toBe('loading');
    expect(button.tabIndex).toBe(-1);
    expect(screen.getByText('正在提交')).toBeTruthy();
  });

  it('error：同时渲染错误图标与文字，不只变色', () => {
    const { container } = render(
      <Button controlId="resend-code" errorMessage="验证码发送失败">
        重新发送
      </Button>,
    );

    expect(screen.getByText('验证码发送失败')).toBeTruthy();
    expect(container.querySelector('[data-mgd-icon="error"]')).not.toBeNull();
    expect(
      screen.getByRole('button', { name: /重新发送/ }).getAttribute('aria-describedby'),
    ).toContain('resend-code-error');
  });

  it('目标尺寸：primary 应用 44px 类，min 应用 24px 类', () => {
    const { container } = render(
      <>
        <Button controlId="primary-target" targetSize="primary">
          主控制
        </Button>
        <Button controlId="min-target" targetSize="min">
          次控制
        </Button>
      </>,
    );

    expect(
      container.querySelector('[data-mgd-control="primary-target"]')?.className,
    ).toContain('mgd-target-primary');
    expect(container.querySelector('[data-mgd-control="min-target"]')?.className).toContain(
      'mgd-target-min',
    );
  });

  it('组件不得关闭焦点环：className 中不出现 outline-none', () => {
    const { container } = render(<Button controlId="focus-check">聚焦</Button>);

    expect(container.innerHTML).not.toContain('outline-none');
  });
});
