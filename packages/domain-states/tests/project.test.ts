/**
 * 项目状态派生规则单测（三态语义分类与终态判定）。
 */

import { describe, expect, it } from 'vitest';

import {
  isProjectStatus,
  isTerminalProjectStatus,
  PROJECT_STATUSES,
  projectStatusTone,
} from '../src/project.ts';

describe('项目状态派生规则', () => {
  it('15 项状态枚举无重复', () => {
    expect(PROJECT_STATUSES.length).toBe(15);
    expect(new Set(PROJECT_STATUSES).size).toBe(15);
  });

  it('三态语义分类互不相同：通过 / 未通过 / 评估未完成', () => {
    expect(projectStatusTone('ROUND_PASSED')).toBe('pass');
    expect(projectStatusTone('COMPLETED')).toBe('pass');
    expect(projectStatusTone('ROUND_FAILED')).toBe('fail');
    expect(projectStatusTone('EVALUATION_INCOMPLETE')).toBe('incomplete');
    expect(projectStatusTone('READY')).toBe('neutral');
  });

  it('评估未完成不是失败也不是终态（允许重试）', () => {
    expect(projectStatusTone('EVALUATION_INCOMPLETE')).not.toBe('fail');
    expect(isTerminalProjectStatus('EVALUATION_INCOMPLETE')).toBe(false);
    expect(isTerminalProjectStatus('COMPLETED')).toBe(true);
  });

  it('未知状态标识被拒绝', () => {
    expect(isProjectStatus('READY')).toBe(true);
    expect(isProjectStatus('ROUND_PASSD')).toBe(false);
    expect(isProjectStatus('LIVE')).toBe(false);
  });
});
