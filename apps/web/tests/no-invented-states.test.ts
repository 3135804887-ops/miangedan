import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

import { PROJECT_STATUSES, SESSION_STATUSES } from '@mgd/domain-states';
import { describe, expect, it } from 'vitest';

const ROOT = join(import.meta.dirname, '..', '..', '..');
const ALLOWED = new Set([
  ...PROJECT_STATUSES,
  ...SESSION_STATUSES,
  'NOT_STARTED',
  'IN_PROGRESS',
  'COMPLETED_OR_EXITED',
  'PASS',
  'FAIL',
]);

function sourceFiles(dir: string, sink: string[]): void {
  for (const entry of readdirSync(dir)) {
    if (['node_modules', '.next', 'tests'].includes(entry)) continue;
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) sourceFiles(full, sink);
    else if (/\.(?:ts|tsx)$/.test(entry)) sink.push(full);
  }
}

describe('页面状态名字面量守门', () => {
  it('应用与 UI 源码不自创大写蛇形状态', () => {
    const files: string[] = [];
    sourceFiles(join(ROOT, 'apps'), files);
    sourceFiles(join(ROOT, 'packages', 'ui', 'src'), files);
    const hits: string[] = [];
    const pattern = /(['"`])([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)\1/g;
    for (const file of files) {
      const text = readFileSync(file, 'utf8');
      for (const match of text.matchAll(pattern)) {
        const value = match[2];
        if (value === undefined || ALLOWED.has(value)) continue;
        const line = text.slice(0, match.index).split('\n').length;
        hits.push(`${relative(ROOT, file)}:${line} ${value}`);
      }
    }
    expect(hits, hits.join('\n')).toEqual([]);
  });
});
