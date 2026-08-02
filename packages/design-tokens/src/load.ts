/**
 * 令牌读取与扁平化。JSON 文件是唯一的机器可读事实源（需求 G5 第 1 条），
 * CSS 变量、Tailwind 主题与 TS 常量都由本模块的输出派生。
 */

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

export const TOKENS_DIR = join(import.meta.dirname, '..', 'tokens');
export const GENERATED_DIR = join(import.meta.dirname, '..', 'generated');

export interface TokenLeaf {
  readonly value: string;
  readonly usage?: string;
}

type TokenFile = Record<string, unknown>;

function readTokenFile(fileName: string): TokenFile {
  const raw = readFileSync(join(TOKENS_DIR, fileName), 'utf8');
  return JSON.parse(raw) as TokenFile;
}

export interface TokenSet {
  readonly color: Record<string, TokenLeaf>;
  readonly font: Record<string, { readonly size: string; readonly lineHeight: string }>;
  readonly fontFamily: Record<string, TokenLeaf>;
  readonly space: Record<string, TokenLeaf>;
  readonly targetSize: Record<string, TokenLeaf>;
  readonly focusRing: Record<string, TokenLeaf>;
  readonly breakpoint: Record<string, TokenLeaf>;
  readonly layout: Record<string, TokenLeaf>;
}

function pickGroup<T>(file: TokenFile, group: string): Record<string, T> {
  const value = file[group];
  if (value === undefined || value === null || typeof value !== 'object') {
    throw new Error(`令牌文件缺少分组 "${group}"`);
  }
  return value as Record<string, T>;
}

export function loadTokens(): TokenSet {
  const colorFile = readTokenFile('color.json');
  const typographyFile = readTokenFile('typography.json');
  const spaceFile = readTokenFile('space.json');
  const breakpointFile = readTokenFile('breakpoint.json');

  return {
    color: pickGroup<TokenLeaf>(colorFile, 'color'),
    font: pickGroup<{ size: string; lineHeight: string }>(typographyFile, 'font'),
    fontFamily: pickGroup<TokenLeaf>(typographyFile, 'fontFamily'),
    space: pickGroup<TokenLeaf>(spaceFile, 'space'),
    targetSize: pickGroup<TokenLeaf>(spaceFile, 'targetSize'),
    focusRing: pickGroup<TokenLeaf>(spaceFile, 'focusRing'),
    breakpoint: pickGroup<TokenLeaf>(breakpointFile, 'breakpoint'),
    layout: pickGroup<TokenLeaf>(breakpointFile, 'layout'),
  };
}

/** kebab-case 化令牌键（neutral900 → neutral-900、contentMaxWidth → content-max-width）。 */
export function kebab(key: string): string {
  return key
    .replace(/([a-z])([A-Z])/g, '$1-$2')
    .replace(/([a-zA-Z])(\d)/g, '$1-$2')
    .toLowerCase();
}

/**
 * 语义令牌名称集合：稳定契约（DESIGN-SYSTEM 第 10 节第 1 条）。
 * 名称集合变更必须显式更新 tokens/NAMES.lock，换肤只允许改 value。
 */
export function semanticTokenNames(tokens: TokenSet): readonly string[] {
  const names: string[] = [];
  for (const key of Object.keys(tokens.color)) names.push(`color.${key}`);
  for (const key of Object.keys(tokens.font)) {
    names.push(`font.${key}.size`, `font.${key}.lineHeight`);
  }
  for (const key of Object.keys(tokens.fontFamily)) names.push(`fontFamily.${key}`);
  for (const key of Object.keys(tokens.space)) names.push(`space.${key}`);
  for (const key of Object.keys(tokens.targetSize)) names.push(`targetSize.${key}`);
  for (const key of Object.keys(tokens.focusRing)) names.push(`focusRing.${key}`);
  for (const key of Object.keys(tokens.breakpoint)) names.push(`breakpoint.${key}`);
  for (const key of Object.keys(tokens.layout)) names.push(`layout.${key}`);
  return names.sort((a, b) => a.localeCompare(b));
}
