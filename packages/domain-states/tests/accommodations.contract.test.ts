/**
 * 便利设置枚举一致性断言（需求 G1 第 1 条、ACCESSIBILITY 第 5.1 节）。
 * 事实源为 ai/schemas/turn-evidence.schema.json 的 accommodations_in_effect。
 */

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

import { ACCOMMODATION_KEYS, defaultAccommodations } from '../src/accommodations.ts';

const REPO_ROOT = join(import.meta.dirname, '..', '..', '..');
const SCHEMA_PATH = join(REPO_ROOT, 'ai', 'schemas', 'turn-evidence.schema.json');

interface TurnEvidenceSchema {
  readonly properties: {
    readonly accommodations_in_effect: {
      readonly items: { readonly enum: readonly string[] };
    };
  };
}

function schemaEnum(): readonly string[] {
  const parsed = JSON.parse(readFileSync(SCHEMA_PATH, 'utf8')) as TurnEvidenceSchema;
  return parsed.properties.accommodations_in_effect.items.enum;
}

describe('turn-evidence.schema.json 与便利设置枚举一致', () => {
  it('9 项便利设置集合完全一致', () => {
    const fromSchema = schemaEnum();

    expect(fromSchema.length).toBe(9);
    expect([...ACCOMMODATION_KEYS].sort()).toEqual([...fromSchema].sort());
  });

  it('默认值全部关闭且逐项独立', () => {
    const defaults = defaultAccommodations();

    expect(Object.keys(defaults).length).toBe(9);
    expect(Object.values(defaults).every((enabled) => enabled === false)).toBe(true);
  });
});
