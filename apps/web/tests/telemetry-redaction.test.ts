import { afterEach, describe, expect, it } from 'vitest';

import { normalizeRoute, reportEvent, setTelemetrySink } from '../src/lib/telemetry.ts';

describe('前端遥测白名单', () => {
  afterEach(() => setTelemetrySink(() => undefined));

  it('只发送无正文的允许字段并归一化动态路由', () => {
    let payload: Readonly<Record<string, string>> = {};
    setTelemetrySink((next) => { payload = next; });
    reportEvent({
      route: normalizeRoute('/zh-CN/projects/3f71d7bd-cdf2-4ad2-a726-ecab2b3ad35c/rounds/2'),
      scrId: 'SCR-10',
      errorCode: 'internal',
      traceId: 'synthetic-trace',
    });
    expect(payload.route).toBe('/zh-CN/projects/[id]/rounds/[n]');
    expect(Object.keys(payload).sort()).toEqual(['at', 'errorCode', 'route', 'scrId', 'traceId'].sort());
  });

  it('开发期拒绝字幕、回答或令牌等额外字段', () => {
    expect(() => reportEvent({ route: '/sessions/[id]', scrId: 'SCR-08', answer: '正文' } as never))
      .toThrow(/白名单外字段/);
  });
});
