/**
 * StateView 五态断言（需求 G2、SCREEN-SPEC 第 9 节）。
 */

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { Button } from '../src/primitives/button.tsx';
import { StateView, type PageState, type StateViewLabels } from '../src/patterns/state-view.tsx';

const labels: StateViewLabels = {
  emptyHeading: '暂无内容',
  forbiddenHeading: '当前无法访问',
  recoveringHeading: '可从上次位置继续',
  error: {
    heading: '操作未完成',
    impact: '影响',
    dataRetained: '数据是否保留',
    retryAction: '可重试动作',
    billing: '是否计费',
    scoring: '是否影响评分',
    traceLabel: '请求标识',
  },
};

function renderState(state: PageState) {
  return render(
    <StateView state={state} labels={labels} regionLabel="test-region">
      <p>业务内容</p>
    </StateView>,
  );
}

describe('StateView 五态', () => {
  it('ready：渲染 children', () => {
    const { container } = renderState({ kind: 'ready' });

    expect(screen.getByText('业务内容')).toBeTruthy();
    expect(container.querySelector('[data-mgd-state-view="ready"]')).not.toBeNull();
  });

  it('empty：说明为什么为空并给出下一步，且不渲染业务数值', () => {
    const { container } = renderState({
      kind: 'empty',
      reason: '你还没有面试项目',
      nextAction: <Button controlId="create-project">新建面试</Button>,
    });

    const empty = container.querySelector('[data-mgd-state-view="empty"]');
    expect(empty).not.toBeNull();
    expect(screen.getByText('你还没有面试项目')).toBeTruthy();
    expect(screen.getByRole('button', { name: '新建面试' })).toBeTruthy();
    expect(screen.queryByText('业务内容')).toBeNull();
    // 空态不得展示误导性占位数据（G2 第 2 条）
    expect(/\d/.test(empty?.textContent ?? '')).toBe(false);
  });

  it('loading：输出 aria-busy 与预期时长文案', () => {
    renderState({
      kind: 'loading',
      busyLabel: '正在解析',
      expectation: '通常在 60 秒内完成（P95）',
    });

    const status = screen.getByRole('status');
    expect(status.getAttribute('aria-busy')).toBe('true');
    expect(screen.getByText('通常在 60 秒内完成（P95）')).toBeTruthy();
  });

  it('error：五要素全部渲染且容器为 role=alert', () => {
    const { container } = renderState({
      kind: 'error',
      error: {
        code: 'internal',
        traceId: 'trace-0001',
        impact: '服务出现内部错误，本次操作没有完成。',
        dataRetained: '已提交的数据与评分证据完整保留。',
        retryAction: '稍后重试。',
        billing: '系统责任故障不计费。',
        scoring: '不判失败，不影响已有评分。',
      },
      retryAction: <Button controlId="retry-report">重试</Button>,
    });

    expect(screen.getByRole('alert')).toBeTruthy();
    for (const facet of ['impact', 'dataRetained', 'retryAction', 'billing', 'scoring']) {
      expect(container.querySelector(`[data-mgd-error-facet="${facet}"]`)).not.toBeNull();
    }
    expect(screen.getByText('trace-0001')).toBeTruthy();
    expect(screen.getByRole('button', { name: '重试' })).toBeTruthy();
  });

  it('forbidden：说明所需权限与获取路径，不暴露他人数据', () => {
    const { container } = renderState({
      kind: 'forbidden',
      requiredPermission: '需要项目所有人权限',
      acquirePath: 'signIn',
      acquireHint: '登录后可继续',
    });

    const forbidden = container.querySelector('[data-mgd-state-view="forbidden"]');
    expect(forbidden?.getAttribute('data-mgd-acquire-path')).toBe('signIn');
    expect(screen.getByText('需要项目所有人权限')).toBeTruthy();
    expect(screen.getByText('登录后可继续')).toBeTruthy();
  });

  it('recovering：说明可继续的位置并提供继续入口', () => {
    renderState({
      kind: 'recovering',
      resumeAt: '可从最后已确认回合继续',
      resumeAction: <Button controlId="resume-session">继续</Button>,
    });

    expect(screen.getByText('可从最后已确认回合继续')).toBeTruthy();
    expect(screen.getByRole('button', { name: '继续' })).toBeTruthy();
  });
});
