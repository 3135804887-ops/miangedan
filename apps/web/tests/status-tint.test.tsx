/** 状态徽标：状态机枚举 → 双语文案（SCREEN-SPEC：禁止自创状态名）。 */

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { StatusTint } from '../src/components/status-tint.tsx';

describe('StatusTint（状态机枚举映射）', () => {
  it('ROUND_PASSED 中文显示"本轮通过"', () => {
    render(<StatusTint status="ROUND_PASSED" locale="zh-CN" />);
    expect(screen.getByText('本轮通过')).toBeDefined();
  });

  it('EVALUATION_INCOMPLETE 英文显示 evaluation incomplete 且语义为 info', () => {
    const { container } = render(<StatusTint status="EVALUATION_INCOMPLETE" locale="en-US" />);
    expect(screen.getByText('Evaluation incomplete')).toBeDefined();
    expect(container.querySelector('.mgd-tint--info')).not.toBeNull();
  });

  it('未知状态不会伪造语义（回退原文）', () => {
    render(<StatusTint status="UNKNOWN_STATE" locale="zh-CN" />);
    expect(screen.getByText('UNKNOWN_STATE')).toBeDefined();
  });
});
