'use client';

import { useEffect, useState } from 'react';

import { isMocksEnabled, MOCK_READY_EVENT } from '../mocks/flag.ts';

/**
 * 等待 Mock_Layer 就绪后再发起数据请求（避免与 SW 注册竞态；
 * 演示模式关闭时立即返回 true，行为与直连真实 API 完全一致）。
 */
export function useMockReady(timeoutMs = 8000): boolean {
  const [ready, setReady] = useState(!isMocksEnabled());
  useEffect(() => {
    if (!isMocksEnabled()) return;
    const onReady = () => setReady(true);
    window.addEventListener(MOCK_READY_EVENT, onReady);
    const timer = window.setTimeout(onReady, timeoutMs);
    return () => {
      window.removeEventListener(MOCK_READY_EVENT, onReady);
      window.clearTimeout(timer);
    };
  }, [timeoutMs]);
  return ready;
}
