/**
 * 语言前缀路由（偏离 1：SCREEN-SPEC 第 5 节路由建议的扩展）。
 * 全部页面在 /{locale} 前缀下提供；无前缀路径由 proxy 重定向到带前缀等价路径。
 */

import { DEFAULT_LOCALE, SUPPORTED_LOCALES } from '@mgd/i18n';
import { defineRouting } from 'next-intl/routing';
import { createNavigation } from 'next-intl/navigation';

export const routing = defineRouting({
  locales: [...SUPPORTED_LOCALES],
  defaultLocale: DEFAULT_LOCALE,
  // always：所有路径都带语言前缀，使界面语言在 URL 层可分享、可缓存、可直达
  localePrefix: 'always',
});

export const { Link, redirect, usePathname, useRouter, getPathname } = createNavigation(routing);
