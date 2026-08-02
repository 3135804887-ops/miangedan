/**
 * Node 侧 Mock_Layer（测试用）。在 vitest.setup.ts 中始终启用。
 * 关闭 Mock_Layer 后页面代码不变即可直连真实 API（需求 G6 第 5 条）。
 */

import { setupServer } from 'msw/node';

import { handlers } from './handlers/index.ts';

export const mockServer = setupServer(...handlers);
