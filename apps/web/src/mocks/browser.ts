/**
 * 浏览器侧 Mock_Layer（开发/演示用）。
 * 仅当 NEXT_PUBLIC_MGD_MOCKS === 'on' 时启动；生产构建不启用。
 * setupWorker 采用动态导入：本模块可能被服务端组件引用（'use client' 边界的
 * SSR 预渲染），顶层求值会触发 MSW 的 non-browser 断言导致页面 500。
 */

import { handlers } from './handlers/index.ts';

export async function startMocksIfEnabled(): Promise<void> {
  if (process.env.NEXT_PUBLIC_MGD_MOCKS !== 'on') return;
  const { setupWorker } = await import('msw/browser');
  const worker = setupWorker(...handlers);
  await worker.start({ onUnhandledRequest: 'bypass' });
}
