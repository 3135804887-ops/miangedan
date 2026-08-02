/**
 * 语言前缀代理（需求 G3 第 1~3 条）。
 * - 无前缀路径重定向到带前缀等价路径
 * - 不支持的 locale 段按 en-US 回退渲染
 *
 * Next.js 16 使用 proxy.ts 取代已弃用的 middleware.ts 文件约定。
 */

import createMiddleware from 'next-intl/middleware';

import { routing } from './i18n/navigation.ts';

export default createMiddleware(routing);

export const config = {
  // 跳过 Next 内部资源与静态文件；其余路径全部经过语言前缀处理
  matcher: ['/((?!_next|_vercel|.*\\..*).*)'],
};
