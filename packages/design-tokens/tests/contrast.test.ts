import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

import { contrastRatio, meetsThreshold, parseHexColor, roundRatio } from '../src/contrast.ts';
import { loadTokens, semanticTokenNames, TOKENS_DIR } from '../src/load.ts';

describe('设计令牌门禁', () => {
  it('按 WCAG 算法计算已知色对', () => {
    expect(roundRatio(contrastRatio('#000000', '#ffffff'))).toBe(21);
    expect(roundRatio(contrastRatio('#777777', '#777777'))).toBe(1);
    expect(meetsThreshold(4.5, 'body')).toBe(true);
    expect(meetsThreshold(2.99, 'large')).toBe(false);
  });

  it('拒绝非法颜色而不静默回退', () => {
    expect(() => parseHexColor('not-a-color')).toThrow(/非法颜色值/);
  });

  it('语义名称集合与锁文件完全一致', () => {
    const expected = readFileSync(join(TOKENS_DIR, 'NAMES.lock'), 'utf8')
      .split('\n')
      .map((line) => line.trim())
      .filter((line) => line.length > 0 && !line.startsWith('#'));
    expect(semanticTokenNames(loadTokens())).toEqual(expected);
  });

  it('新增语义名会产生可检测差异', () => {
    const tokens = loadTokens();
    const changed = { ...tokens, color: { ...tokens.color, unexpectedAccent: { value: '#000000' } } };
    expect(semanticTokenNames(changed)).toContain('color.unexpectedAccent');
  });
});
