/**
 * apps/web 测试环境准备。
 * Mock_Layer 在测试期始终启用，页面代码发出的 fetch 全部被拦截并返回 synthetic 数据。
 */

import { createElement, type AnchorHTMLAttributes, type ReactNode } from 'react';
import { afterAll, afterEach, beforeAll, vi } from 'vitest';

import { mockServer } from './src/mocks/server.ts';

vi.mock('./src/i18n/navigation.ts', () => ({
  Link: ({
    href,
    children,
    ...props
  }: Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'href'> & {
    readonly href: string | { readonly pathname: string; readonly query?: Record<string, string> };
    readonly children: ReactNode;
  }) => {
    const target = typeof href === 'string'
      ? href
      : `${href.pathname}${href.query === undefined ? '' : `?${new URLSearchParams(href.query)}`}`;
    return createElement('a', { ...props, href: target }, children);
  },
}));

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
