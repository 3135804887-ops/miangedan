/**
 * StatusBadge 断言（需求 G1 第 4、5 条）：
 * 三态图标互不相同，且移除全部颜色样式后仍可通过图标与文字区分。
 */

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { StatusBadge } from '../src/patterns/status-badge.tsx';

describe('StatusBadge 状态双通道', () => {
  it('渲染图标 + 文字标签 + 可用动作说明', () => {
    const { container } = render(
      <StatusBadge status="ROUND_PASSED" label="当前轮通过" actionHint="可进入下一轮" />,
    );

    expect(screen.getByText('当前轮通过')).toBeTruthy();
    expect(screen.getByText('可进入下一轮')).toBeTruthy();
    expect(container.querySelector('[data-mgd-icon="pass"]')).not.toBeNull();
    expect(container.querySelector('[data-mgd-status="ROUND_PASSED"]')).not.toBeNull();
  });

  it('通过 / 未通过 / 评估未完成三态图标互不相同', () => {
    const passed = render(<StatusBadge status="ROUND_PASSED" label="通过" />);
    const failed = render(<StatusBadge status="ROUND_FAILED" label="未通过" />);
    const incomplete = render(<StatusBadge status="EVALUATION_INCOMPLETE" label="评估未完成" />);

    const icons = [passed, failed, incomplete].map((result) =>
      result.container.querySelector('[data-mgd-icon]')?.getAttribute('data-mgd-icon'),
    );

    expect(icons).toEqual(['pass', 'fail', 'incomplete']);
    expect(new Set(icons).size).toBe(3);
  });

  it('语义色调随状态映射，评估未完成不归入失败', () => {
    const incomplete = render(<StatusBadge status="EVALUATION_INCOMPLETE" label="评估未完成" />);
    const tone = incomplete.container
      .querySelector('[data-mgd-status="EVALUATION_INCOMPLETE"]')
      ?.getAttribute('data-mgd-tone');

    expect(tone).toBe('incomplete');
    expect(tone).not.toBe('fail');
  });

  it('图标对屏幕阅读器隐藏，语义由文字承担', () => {
    const { container } = render(<StatusBadge status="READY" label="已就绪" />);
    const icon = container.querySelector('[data-mgd-icon]');

    expect(icon?.getAttribute('aria-hidden')).toBe('true');
    expect(screen.getByText('已就绪')).toBeTruthy();
  });
});
