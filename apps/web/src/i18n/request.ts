/**
 * next-intl 请求级配置：加载当前界面语言的消息资源。
 * 不支持的 locale 回退到 en-US（需求 G3 第 3 条）。
 */

import { resolveLocale, type Locale } from '@mgd/i18n';
import { getRequestConfig } from 'next-intl/server';

import { loadMessages } from './messages.ts';

export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale: Locale = resolveLocale(requested);

  return {
    locale,
    messages: await loadMessages(locale),
  };
});
