/**
 * 状态机文档一致性断言（需求 G1 第 1 条）。
 * 直接解析已批准的 docs/domain/INTERVIEW-STATE-MACHINE.md，避免枚举随代码漂移。
 */

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

import { PROJECT_STATUSES } from '../src/project.ts';
import { SESSION_STATUSES } from '../src/session.ts';

const REPO_ROOT = join(import.meta.dirname, '..', '..', '..');
const DOC_PATH = join(REPO_ROOT, 'docs', 'domain', 'INTERVIEW-STATE-MACHINE.md');

function readDoc(): string {
  return readFileSync(DOC_PATH, 'utf8');
}

/** 提取指定小节标题之后、下一个同级标题之前的正文。 */
function sectionBody(text: string, headingPrefix: string): string {
  const lines = text.split('\n');
  const startIndex = lines.findIndex((line) => line.startsWith(headingPrefix));
  expect(startIndex, `未找到小节 ${headingPrefix}`).toBeGreaterThanOrEqual(0);

  const rest = lines.slice(startIndex + 1);
  const endOffset = rest.findIndex((line) => /^#{2,3} /.test(line));
  return (endOffset === -1 ? rest : rest.slice(0, endOffset)).join('\n');
}

/** 取 Markdown 表格首列中反引号包裹的大写蛇形标识。 */
function firstColumnCodeIdentifiers(body: string): string[] {
  const found: string[] = [];
  for (const line of body.split('\n')) {
    if (!line.startsWith('|')) continue;
    const firstCell = line.split('|')[1];
    if (firstCell === undefined) continue;
    const match = /`([A-Z][A-Z0-9_]*)`/.exec(firstCell);
    if (match?.[1] !== undefined) found.push(match[1]);
  }
  return found;
}

describe('INTERVIEW-STATE-MACHINE 与 @mgd/domain-states 枚举一致', () => {
  it('项目状态集合与第 5.1 节表格完全一致', () => {
    const identifiers = firstColumnCodeIdentifiers(sectionBody(readDoc(), '### 5.1'));

    expect(identifiers.length).toBe(15);
    expect([...identifiers].sort()).toEqual([...PROJECT_STATUSES].sort());
  });

  it('会话状态集合与第 6.2 节表格完全一致', () => {
    const body = sectionBody(readDoc(), '### 6.2');
    // 6.2 含两张表：状态表与迁移表；状态表的首列是反引号包裹的状态名，
    // 迁移表首列是「当前」列（同为状态名），因此取去重后的集合比较。
    const identifiers = [...new Set(firstColumnCodeIdentifiers(body))];

    expect(identifiers.length).toBe(10);
    expect([...identifiers].sort()).toEqual([...SESSION_STATUSES].sort());
  });

  it('项目状态枚举无重复且顺序与文档一致', () => {
    const identifiers = firstColumnCodeIdentifiers(sectionBody(readDoc(), '### 5.1'));

    expect(new Set(PROJECT_STATUSES).size).toBe(PROJECT_STATUSES.length);
    expect([...PROJECT_STATUSES]).toEqual(identifiers);
  });
});
