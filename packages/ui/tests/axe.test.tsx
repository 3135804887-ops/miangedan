import { render } from '@testing-library/react';
import axe from 'axe-core';
import { describe, expect, it } from 'vitest';

import { Button, ErrorPanel, StateView } from '../src/index.ts';

describe('UI 基础层 axe 门禁', () => {
  it('serious 与 critical 违规为 0', async () => {
    const { container } = render(
      <main>
        <h1>状态示例</h1>
        <StateView
          state={{
            kind: 'empty',
            reason: '尚未创建项目',
            nextAction: <Button controlId="create-project">创建项目</Button>,
          }}
          labels={{
            emptyHeading: '暂无内容',
            forbiddenHeading: '无法访问',
            recoveringHeading: '可以继续',
            error: {
              heading: '操作未完成', impact: '影响', dataRetained: '数据保留', retryAction: '重试',
              billing: '计费', scoring: '评分', traceLabel: '请求标识',
            },
          }}
          regionLabel="示例状态"
        >
          <Button controlId="ready-action">继续</Button>
        </StateView>
        <ErrorPanel
          error={{
            code: 'internal', traceId: 'synthetic-trace', impact: '页面暂不可用', dataRetained: '数据已保留',
            retryAction: '可重试', billing: '不计费', scoring: '不影响评分',
          }}
          labels={{
            heading: '操作未完成', impact: '影响', dataRetained: '数据保留', retryAction: '重试',
            billing: '计费', scoring: '评分', traceLabel: '请求标识',
          }}
        />
      </main>,
    );
    const result = await axe.run(container, { rules: { 'color-contrast': { enabled: false } } });
    const blocking = result.violations.filter(({ impact }) => impact === 'serious' || impact === 'critical');
    expect(blocking).toEqual([]);
  });
});
