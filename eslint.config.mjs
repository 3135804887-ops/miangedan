import js from '@eslint/js';
import mgd from '@mgd/eslint-plugin-mgd';
import globals from 'globals';
import tseslint from 'typescript-eslint';

/**
 * 工作区统一 ESLint 扁平配置（需求 B0-1 第 5 条、G1 第 3 条、G5 第 2 条、G8 第 1 条）。
 * 单次 `pnpm lint` 覆盖 apps/* 与 packages/*。
 */
export default tseslint.config(
  {
    ignores: [
      '**/node_modules/**',
      '**/.next/**',
      '**/dist/**',
      '**/coverage/**',
      // 生成物不参与 lint（contracts/README.md 规则 1、令牌生成物同理）
      'contracts/ts/**',
      'packages/design-tokens/generated/**',
    ],
  },

  js.configs.recommended,
  ...tseslint.configs.recommended,

  {
    languageOptions: {
      ecmaVersion: 2023,
      sourceType: 'module',
      globals: { ...globals.node, ...globals.browser },
    },
    plugins: { mgd },
    rules: {
      // 需求 G1 第 3 条：页面层不得书写领域状态字面量
      'mgd/no-domain-state-literal': 'error',
      '@typescript-eslint/consistent-type-imports': 'error',
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      'no-console': 'error',
    },
  },

  // 领域枚举包本身与规则实现需要书写状态字面量
  {
    files: ['packages/domain-states/**/*.ts', 'packages/eslint-plugin-mgd/**/*.mjs'],
    rules: { 'mgd/no-domain-state-literal': 'off' },
  },

  // 测试可以书写状态字面量作为被测输入
  {
    files: ['**/tests/**/*.ts', '**/tests/**/*.tsx'],
    rules: { 'mgd/no-domain-state-literal': 'off' },
  },

  // 构建脚本与门禁脚本允许写标准输出
  {
    files: ['tools/**/*.mjs', 'packages/**/scripts/**/*.mjs', 'packages/design-tokens/src/*.ts'],
    rules: { 'no-console': 'off' },
  },

  // 需求 G5 第 2 条：页面样式禁止硬编码色值与字号
  {
    files: ['apps/**/*.tsx', 'packages/ui/src/**/*.tsx'],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: "Literal[value=/#[0-9a-fA-F]{3,8}\\b/]",
          message: '禁止硬编码色值，请使用 @mgd/design-tokens 的语义令牌（需求 G5 第 2 条）。',
        },
        {
          selector: "Literal[value=/\\b(?:rgb|hsl)a?\\(/]",
          message: '禁止硬编码色值函数，请使用设计令牌 CSS 变量（需求 G5 第 2 条）。',
        },
        {
          selector: "Literal[value=/\\boutline-none\\b/]",
          message: '禁止移除焦点环（ACCESSIBILITY 第 4.2 节：focus-visible 必须可见）。',
        },
      ],
    },
  },
);
