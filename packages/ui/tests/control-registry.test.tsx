/**
 * 控件清册与禁用词表断言（需求 G9 第 1 条）。
 * 该机制在批次 4 用于校验 apps/admin 与机构端「0 个改分控件」红线，
 * 批次 0 先建立并自测其判定能力。
 */

import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { Button } from '../src/primitives/button.tsx';
import { collectControls, findForbiddenControls } from '../src/testing/control-registry.ts';

describe('控件清册', () => {
  it('收集全部带 data-mgd-control 的控件并按标识排序', () => {
    const { container } = render(
      <>
        <Button controlId="retry-report">重试</Button>
        <Button controlId="download-report">下载</Button>
      </>,
    );

    const controls = collectControls(container);

    expect(controls.map((c) => c.controlId)).toEqual(['download-report', 'retry-report']);
    expect(controls[0]?.accessibleName).toBe('下载');
    expect(controls[0]?.tagName).toBe('button');
  });

  it('记录禁用态，便于骨架页断言操作按钮已禁用', () => {
    const { container } = render(
      <Button controlId="approve-refund" disabled disabledReason="服务端能力未开放">
        批准
      </Button>,
    );

    expect(collectControls(container)[0]?.disabled).toBe(true);
  });

  it('合规控件集合的禁用词表命中数为 0', () => {
    const { container } = render(
      <>
        <Button controlId="view-audit-log">查看审计日志</Button>
        <Button controlId="export-region-metrics">导出区域指标</Button>
      </>,
    );

    expect(findForbiddenControls(collectControls(container))).toEqual([]);
  });

  it('改分类控件被判定为红线违规（控件标识命中）', () => {
    const { container } = render(<Button controlId="edit-score">编辑</Button>);

    const hits = findForbiddenControls(collectControls(container));

    expect(hits.length).toBe(1);
    expect(hits[0]?.controlId).toBe('edit-score');
  });

  it('改分类控件被判定为红线违规（可访问名命中中文）', () => {
    const { container } = render(<Button controlId="admin-action-01">修改解锁状态</Button>);

    const hits = findForbiddenControls(collectControls(container));

    expect(hits.length).toBe(1);
    expect(hits[0]?.accessibleName).toBe('修改解锁状态');
  });

  it('证据正文编辑控件被判定为红线违规', () => {
    const { container } = render(<Button controlId="evidence-text-editor">编辑证据</Button>);

    expect(findForbiddenControls(collectControls(container)).length).toBe(1);
  });
});
