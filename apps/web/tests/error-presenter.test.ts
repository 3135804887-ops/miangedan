import { describe, expect, it } from 'vitest';

import { ERROR_CODES, ERROR_FACETS, parseErrorEnvelope, presentError } from '../src/lib/error-presenter.ts';

describe('错误五要素映射', () => {
  const t = (key: string): string => `译文:${key}`;

  it.each(ERROR_CODES)('覆盖 ErrorCode %s 的五要素', (code) => {
    const view = presentError({
      error: { code, trace_id: 'synthetic-trace', data_region: 'cn' },
    }, t);
    expect(view.code).toBe(code);
    expect(ERROR_FACETS.map((facet) => view[facet])).toHaveLength(5);
    for (const facet of ERROR_FACETS) expect(view[facet]).toContain(`error.${code}.${facet}`);
  });

  it('未知或畸形错误安全回退 internal', () => {
    expect(presentError(undefined, t).code).toBe('internal');
    expect(parseErrorEnvelope({ error: { code: 123 } })).toBeUndefined();
  });

  it('页面覆写存在时优先使用且保留重试安全', () => {
    const translator = (key: string): string =>
      key === 'error.internal.SCR-01.impact' ? '演示暂不可用，不涉及个人数据。' : key;
    expect(
      presentError({ error: { code: 'internal', trace_id: 'synthetic', data_region: 'intl' } }, translator, { scrId: 'SCR-01' }).impact,
    ).toContain('不涉及个人数据');
  });
});
