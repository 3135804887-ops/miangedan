import createNextIntlPlugin from 'next-intl/plugin';
import type { NextConfig } from 'next';

/**
 * apps/web 构建配置。
 * 红线（需求 G8 第 3 条）：不在此暴露任何供应商密钥；
 * 仅 NEXT_PUBLIC_MGD_MOCKS 与 NEXT_PUBLIC_MGD_APP_ENV 两个非敏感键允许进入客户端产物。
 */
const nextConfig: NextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  typedRoutes: true,
  experimental: {
    // 共享包以 TypeScript 源码形式发布，需由应用侧转译
    externalDir: true,
  },
  transpilePackages: ['@mgd/ui', '@mgd/i18n', '@mgd/domain-states', '@mgd/design-tokens'],
};

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

export default withNextIntl(nextConfig);
