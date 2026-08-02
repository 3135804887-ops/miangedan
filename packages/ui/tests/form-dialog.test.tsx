import { fireEvent, render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { AlertDialog, Button, DisclosureNote, Field, IconButton, Switch } from '../src/index.ts';

describe('UI 基础组件无障碍状态', () => {
  it('IconButton 具备名称和主要目标尺寸', () => {
    render(<IconButton controlId="close-panel" label="关闭" icon={<span>×</span>} />);
    const button = screen.getByRole('button', { name: '关闭' });
    expect(button.className).toContain('mgd-target-primary');
  });

  it('Switch 独立切换，disabled 与 loading 均不可聚焦', () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <Switch controlId="consent-media" checked={false} label="保存媒体" onCheckedChange={onChange} />,
    );
    fireEvent.click(screen.getByRole('switch', { name: '保存媒体' }));
    expect(onChange).toHaveBeenCalledWith(true);

    rerender(
      <Switch
        controlId="consent-media"
        checked={false}
        label="保存媒体"
        disabled
        disabledReason="监护人验证尚未完成"
      />,
    );
    expect(screen.getByRole('switch').getAttribute('tabindex')).toBe('-1');
    expect(screen.getByText('监护人验证尚未完成')).toBeTruthy();

    rerender(
      <Switch controlId="consent-media" checked={false} label="保存媒体" loading busyLabel="保存中" />,
    );
    expect(screen.getByRole('switch').getAttribute('aria-busy')).toBe('true');
    expect(screen.getByRole('switch').getAttribute('tabindex')).toBe('-1');
  });

  it('Field 把说明与错误关联到输入控件', () => {
    render(
      <Field fieldId="email" label="邮箱" description="用于接收验证码" errorMessage="验证码已过期">
        <input />
      </Field>,
    );
    const input = screen.getByLabelText('邮箱');
    expect(input.getAttribute('aria-invalid')).toBe('true');
    expect(input.getAttribute('aria-describedby')).toBe('email-description email-error');
    expect(screen.getByRole('alert').textContent).toContain('验证码已过期');
  });

  it('AlertDialog 移入焦点、支持 Escape 并返回触发元素', () => {
    const trigger = document.createElement('button');
    trigger.textContent = '打开';
    document.body.append(trigger);
    trigger.focus();
    const returnFocusTo = createRef<HTMLElement>();
    returnFocusTo.current = trigger;
    const onClose = vi.fn();

    const { unmount } = render(
      <AlertDialog
        open
        title="确认退出"
        description="将标记为评估未完成"
        onClose={onClose}
        returnFocusTo={returnFocusTo}
      >
        <Button controlId="cancel-exit">取消</Button>
        <Button controlId="confirm-exit">确认</Button>
      </AlertDialog>,
    );
    expect(document.activeElement).toBe(screen.getByRole('button', { name: '取消' }));
    fireEvent.keyDown(screen.getByRole('alertdialog'), { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
    unmount();
    expect(document.activeElement).toBe(trigger);
    trigger.remove();
  });

  it('DisclosureNote 是只读说明而不是编辑控件', () => {
    render(<DisclosureNote title="评分规则">60 分门槛在开始后冻结。</DisclosureNote>);
    expect(screen.getByRole('region', { name: '评分规则' }).getAttribute('aria-readonly')).toBe('true');
    expect(screen.queryByRole('button')).toBeNull();
  });
});
