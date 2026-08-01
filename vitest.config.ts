import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

/**
 * 根级 Vitest 配置（需求 G11 第 1、2 条）。
 * 用 projects 统一驱动各包测试：一次 `pnpm test` 覆盖单元、组件、契约一致性与隐私扫描。
 */
export default defineConfig({
  test: {
    projects: [
      {
        test: {
          name: 'design-tokens',
          root: './packages/design-tokens',
          environment: 'node',
          include: ['tests/**/*.test.ts'],
        },
      },
      {
        test: {
          name: 'domain-states',
          root: './packages/domain-states',
          environment: 'node',
          include: ['tests/**/*.test.ts'],
        },
      },
      {
        test: {
          name: 'i18n',
          root: './packages/i18n',
          environment: 'node',
          include: ['tests/**/*.test.ts'],
        },
      },
      {
        test: {
          name: 'eslint-plugin-mgd',
          root: './packages/eslint-plugin-mgd',
          environment: 'node',
          include: ['tests/**/*.test.ts'],
        },
      },
      {
        plugins: [react()],
        test: {
          name: 'ui',
          root: './packages/ui',
          environment: 'jsdom',
          globals: true,
          include: ['tests/**/*.test.tsx'],
        },
      },
      {
        plugins: [react()],
        test: {
          name: 'web',
          root: './apps/web',
          environment: 'jsdom',
          globals: true,
          setupFiles: ['./vitest.setup.ts'],
          include: ['tests/**/*.test.ts', 'tests/**/*.test.tsx'],
        },
      },
    ],
  },
});
