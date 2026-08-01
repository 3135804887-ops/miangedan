/**
 * Mock_Layer handler 汇总（需求 G6 第 4、7 条）。
 *
 * 组织约定：一个页面组一个 handler 文件，每个页面组导出 ok() / failure(code) / empty()
 * 三类场景工厂，供五态测试切换（design.md 第 8.3 节）。
 * 批次 0 只提供健康检查与通用错误场景骨架，页面级 handler 随批次 1~4 增补。
 */

import { http, HttpResponse } from 'msw';
import type { RequestHandler } from 'msw';

import { API_BASE_PATH } from '../../lib/api-fetch.ts';

/** 合成错误信封，形状与 openapi components.schemas.Error 一致。 */
export function errorEnvelope(code: string, traceId = 'synthetic-trace-0001') {
  return {
    error: {
      code,
      message: '合成数据：用于驱动五态测试，不含任何真实个人信息。',
      trace_id: traceId,
      data_region: 'cn',
    },
  };
}

/** 默认 handler：未被具体页面 handler 覆盖的请求返回 501，避免测试静默通过。 */
export const fallbackHandlers: RequestHandler[] = [
  http.all(`${API_BASE_PATH}/*`, () =>
    HttpResponse.json(errorEnvelope('internal', 'synthetic-unhandled'), { status: 501 }),
  ),
];

export const handlers: RequestHandler[] = [...fallbackHandlers];
