/**
 * 浏览器侧 Mock_Layer（开发用）。
 * 仅当 NEXT_PUBLIC_MGD_MOCKS === 'on' 时启动；生产构建不启用。
 */

import { setupWorker } from 'msw/browser';

import { handlers } from './handlers/index.ts';

export const mockWorker = setupWorker(...handlers);

export async function startMocksIfEnabled(): Promise<void> {
  if (process.env.NEXT_PUBLIC_MGD_MOCKS !== 'on') return;
  await mockWorker.start({ onUnhandledRequest: 'bypass' });
}
