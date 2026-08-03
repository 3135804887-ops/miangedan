'use client';

import { useEffect } from 'react';

import { startMocksIfEnabled } from '../mocks/browser.ts';
import { isMocksEnabled, MOCK_READY_EVENT } from '../mocks/flag.ts';

/**
 * Mock_Layer 客户端引导（开发/演示模式 NEXT_PUBLIC_MGD_MOCKS=on 时启用；
 * 生产默认关闭，页面代码直连真实 API）。
 */
export function MockBootstrap(): null {
  useEffect(() => {
    void startMocksIfEnabled().then(() => {
      if (isMocksEnabled()) {
        window.dispatchEvent(new Event(MOCK_READY_EVENT));
      }
    });
  }, []);
  return null;
}
