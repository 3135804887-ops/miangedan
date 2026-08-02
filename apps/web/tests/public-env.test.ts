import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

describe('公开环境变量白名单', () => {
  it('源码仅允许两个非敏感 NEXT_PUBLIC 键', () => {
    const files = [
      join(import.meta.dirname, '..', 'src', 'mocks', 'browser.ts'),
      join(import.meta.dirname, '..', 'next.config.ts'),
    ];
    const keys = new Set<string>();
    for (const file of files) {
      const text = readFileSync(file, 'utf8');
      for (const match of text.matchAll(/NEXT_PUBLIC_[A-Z0-9_]+/g)) keys.add(match[0]);
    }
    expect([...keys].sort()).toEqual(['NEXT_PUBLIC_MGD_APP_ENV', 'NEXT_PUBLIC_MGD_MOCKS']);
  });
});
