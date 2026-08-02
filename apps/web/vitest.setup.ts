/**
 * apps/web 测试环境准备。
 * Mock_Layer 在测试期始终启用，页面代码发出的 fetch 全部被拦截并返回 synthetic 数据。
 */

import { afterAll, afterEach, beforeAll } from 'vitest';

import { mockServer } from './src/mocks/server.ts';

beforeAll(() => {
  // onUnhandledRequest: 'error' 保证遗漏的 handler 立即暴露，而不是静默通过
  mockServer.listen({ onUnhandledRequest: 'error' });
});

afterEach(() => {
  mockServer.resetHandlers();
});

afterAll(() => {
  mockServer.close();
});
