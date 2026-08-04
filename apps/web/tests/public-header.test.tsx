/**
 * 公开页页眉组件测试（SCR-01/02）。
 * 校验：导航文案随界面语言切换（不回落中文）、链接带 /{locale} 前缀（不丢失语言）。
 */

import { render, screen } from '@testing-library/react';
import { NextIntlClientProvider } from 'next-intl';
import { describe, expect, it } from 'vitest';

import { PublicHeader } from '../src/components/public-header.tsx';
import { loadMessages } from '../src/i18n/messages.ts';

async function renderHeader(locale: 'zh-CN' | 'en-US') {
  const messages = await loadMessages(locale);
  return render(
    <NextIntlClientProvider locale={locale} messages={messages}>
      <PublicHeader />
    </NextIntlClientProvider>,
  );
}

describe('PublicHeader（SCR-01/02 公开页页眉）', () => {
  it('中文：渲染中文导航并带 zh-CN 链接前缀', async () => {
    await renderHeader('zh-CN');
    const demo = screen.getByRole('link', { name: '样例演示' });
    const signIn = screen.getByRole('link', { name: '登录 / 注册' });
    expect(demo.getAttribute('href')).toBe('/zh-CN/demo');
    expect(signIn.getAttribute('href')).toBe('/zh-CN/auth');
  });

  it('英文：渲染英文导航、带 en-US 链接前缀，且不回落中文文案', async () => {
    await renderHeader('en-US');
    const demo = screen.getByRole('link', { name: 'Live demo' });
    const signIn = screen.getByRole('link', { name: 'Sign in' });
    expect(demo.getAttribute('href')).toBe('/en-US/demo');
    expect(signIn.getAttribute('href')).toBe('/en-US/auth');
    expect(screen.queryByText('样例演示')).toBeNull();
    expect(screen.queryByText('登录 / 注册')).toBeNull();
  });
});
