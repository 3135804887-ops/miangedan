/**
 * 令牌生成器（需求 G5 第 1、2 条）。
 * 输入：tokens/*.json（唯一事实源）
 * 输出：generated/tokens.css（CSS 变量）、generated/theme.css（Tailwind v4 @theme inline）、
 *       generated/tokens.ts（TS 常量，供组件与测试在 JS 侧读取令牌值）
 * 生成物提交入库，由 `pnpm tokens:check` 校验 diff。
 */

import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

import { GENERATED_DIR, kebab, loadTokens, semanticTokenNames, type TokenSet } from './load.ts';

const HEADER = [
  '/* 本文件由 packages/design-tokens/src/build-css.ts 生成，禁止手工编辑。 */',
  '/* 令牌事实源：packages/design-tokens/tokens/*.json */',
  '/* 重新生成：pnpm tokens:build　校验：pnpm tokens:check */',
].join('\n');

function buildTokensCss(tokens: TokenSet): string {
  const lines: string[] = [HEADER, '', ':root {'];

  lines.push('  /* 语义颜色（DESIGN-SYSTEM 第 5 节；色值为中性占位，品牌视觉见 OD-04） */');
  for (const [key, leaf] of Object.entries(tokens.color)) {
    lines.push(`  --mgd-color-${kebab(key)}: ${leaf.value};`);
  }

  lines.push('', '  /* 字体层级（DESIGN-SYSTEM 第 6 节） */');
  for (const [key, leaf] of Object.entries(tokens.font)) {
    lines.push(`  --mgd-font-${kebab(key)}-size: ${leaf.size};`);
    lines.push(`  --mgd-font-${kebab(key)}-line-height: ${leaf.lineHeight};`);
  }

  lines.push('', '  /* 字族栈 */');
  for (const [key, leaf] of Object.entries(tokens.fontFamily)) {
    lines.push(`  --mgd-font-family-${kebab(key)}: ${leaf.value};`);
  }

  lines.push('', '  /* 间距刻度（4px 基线） */');
  for (const [key, leaf] of Object.entries(tokens.space)) {
    lines.push(`  --mgd-space-${kebab(key)}: ${leaf.value};`);
  }

  lines.push('', '  /* 目标尺寸与焦点环（ACCESSIBILITY 第 4.2 节） */');
  for (const [key, leaf] of Object.entries(tokens.targetSize)) {
    lines.push(`  --mgd-target-size-${kebab(key)}: ${leaf.value};`);
  }
  for (const [key, leaf] of Object.entries(tokens.focusRing)) {
    lines.push(`  --mgd-focus-ring-${kebab(key)}: ${leaf.value};`);
  }

  lines.push('', '  /* 断点与布局 */');
  for (const [key, leaf] of Object.entries(tokens.breakpoint)) {
    lines.push(`  --mgd-breakpoint-${kebab(key)}: ${leaf.value};`);
  }
  for (const [key, leaf] of Object.entries(tokens.layout)) {
    lines.push(`  --mgd-layout-${kebab(key)}: ${leaf.value};`);
  }

  lines.push('}', '');
  return lines.join('\n');
}

function buildThemeCss(tokens: TokenSet): string {
  const lines: string[] = [
    HEADER,
    '',
    '/* Tailwind v4 CSS-first 主题：只做「Tailwind 命名空间 → 令牌变量」的转发， */',
    '/* 因此换肤时只需改 tokens/*.json 的 value，主题映射保持不变。 */',
    '@theme inline {',
  ];

  for (const key of Object.keys(tokens.color)) {
    lines.push(`  --color-${kebab(key)}: var(--mgd-color-${kebab(key)});`);
  }

  lines.push('');
  for (const key of Object.keys(tokens.font)) {
    lines.push(`  --text-${kebab(key)}: var(--mgd-font-${kebab(key)}-size);`);
    lines.push(
      `  --text-${kebab(key)}--line-height: var(--mgd-font-${kebab(key)}-line-height);`,
    );
  }

  lines.push('');
  for (const key of Object.keys(tokens.fontFamily)) {
    lines.push(`  --font-${kebab(key)}: var(--mgd-font-family-${kebab(key)});`);
  }

  lines.push('');
  for (const key of Object.keys(tokens.space)) {
    lines.push(`  --spacing-${kebab(key)}: var(--mgd-space-${kebab(key)});`);
  }

  lines.push('');
  // 断点必须输出字面值：Tailwind v4 会把 @theme 变量内联进媒体查询，
  // 媒体查询条件不支持 var()（Turbopack 解析会失败）。
  for (const [key, leaf] of Object.entries(tokens.breakpoint)) {
    lines.push(`  --breakpoint-${kebab(key)}: ${leaf.value};`);
  }

  lines.push('}', '');
  return lines.join('\n');
}

function buildTokensTs(tokens: TokenSet): string {
  const names = semanticTokenNames(tokens);
  return [
    '// 本文件由 packages/design-tokens/src/build-css.ts 生成，禁止手工编辑。',
    '// 令牌事实源：packages/design-tokens/tokens/*.json',
    '',
    'export const COLOR_TOKENS = ' + JSON.stringify(mapValues(tokens.color), null, 2) + ' as const;',
    '',
    'export const FONT_TOKENS = ' + JSON.stringify(tokens.font, null, 2) + ' as const;',
    '',
    'export const SPACE_TOKENS = ' + JSON.stringify(mapValues(tokens.space), null, 2) + ' as const;',
    '',
    'export const TARGET_SIZE_TOKENS = ' +
      JSON.stringify(mapValues(tokens.targetSize), null, 2) +
      ' as const;',
    '',
    'export const FOCUS_RING_TOKENS = ' +
      JSON.stringify(mapValues(tokens.focusRing), null, 2) +
      ' as const;',
    '',
    'export const BREAKPOINT_TOKENS = ' +
      JSON.stringify(mapValues(tokens.breakpoint), null, 2) +
      ' as const;',
    '',
    'export const LAYOUT_TOKENS = ' +
      JSON.stringify(mapValues(tokens.layout), null, 2) +
      ' as const;',
    '',
    '/** 语义令牌名称集合：稳定契约，与 tokens/NAMES.lock 一致。 */',
    'export const SEMANTIC_TOKEN_NAMES = ' + JSON.stringify(names, null, 2) + ' as const;',
    '',
  ].join('\n');
}

function mapValues(group: Record<string, { value: string }>): Record<string, string> {
  return Object.fromEntries(Object.entries(group).map(([key, leaf]) => [key, leaf.value]));
}

export interface GeneratedArtifacts {
  readonly 'tokens.css': string;
  readonly 'theme.css': string;
  readonly 'tokens.ts': string;
}

export function renderArtifacts(): GeneratedArtifacts {
  const tokens = loadTokens();
  return {
    'tokens.css': buildTokensCss(tokens),
    'theme.css': buildThemeCss(tokens),
    'tokens.ts': buildTokensTs(tokens),
  };
}

export function writeArtifacts(): readonly string[] {
  const artifacts = renderArtifacts();
  mkdirSync(GENERATED_DIR, { recursive: true });
  const written: string[] = [];
  for (const [fileName, content] of Object.entries(artifacts)) {
    const target = join(GENERATED_DIR, fileName);
    writeFileSync(target, content, 'utf8');
    written.push(target);
  }
  return written;
}

const isDirectRun = process.argv[1] !== undefined && process.argv[1].includes('build-css');
if (isDirectRun) {
  for (const file of writeArtifacts()) {
    process.stdout.write(`[令牌生成] ${file}\n`);
  }
}
