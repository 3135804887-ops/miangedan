/**
 * 会话状态派生规则单测（对齐 INTERVIEW-STATE-MACHINE 第 6.2 节表格的计时与计费列）。
 */

import { describe, expect, it } from 'vitest';

import {
  isAvatarBillable,
  isSessionStatus,
  isSystemPause,
  isTimerRunning,
  SESSION_STATUSES,
  type SessionStatus,
} from '../src/session.ts';

describe('会话状态派生规则', () => {
  it('仅 LIVE 与 TEXT_DEGRADED 计时', () => {
    const running = SESSION_STATUSES.filter((status) => isTimerRunning(status));

    expect([...running].sort()).toEqual(['LIVE', 'TEXT_DEGRADED']);
  });

  it('仅 LIVE 消耗数字人额度：TEXT_DEGRADED 自故障点起不再计费', () => {
    const billable = SESSION_STATUSES.filter((status) => isAvatarBillable(status));

    expect(billable).toEqual(['LIVE']);
    expect(isAvatarBillable('TEXT_DEGRADED')).toBe(false);
  });

  it('系统责任暂停不计时：PAUSED_SYSTEM、RECONNECTING、AUTH_PAUSED', () => {
    const paused = SESSION_STATUSES.filter((status) => isSystemPause(status));

    expect([...paused].sort()).toEqual(['AUTH_PAUSED', 'PAUSED_SYSTEM', 'RECONNECTING']);
    for (const status of paused) {
      expect(isTimerRunning(status)).toBe(false);
      expect(isAvatarBillable(status)).toBe(false);
    }
  });

  it('未知状态标识被拒绝', () => {
    expect(isSessionStatus('LIVE')).toBe(true);
    expect(isSessionStatus('PAUSED')).toBe(false);
    expect(isSessionStatus('ROUND_PASSED')).toBe(false);
  });

  it('每个会话状态都有确定的计时与计费判定', () => {
    for (const status of SESSION_STATUSES satisfies readonly SessionStatus[]) {
      expect(typeof isTimerRunning(status)).toBe('boolean');
      expect(typeof isAvatarBillable(status)).toBe('boolean');
    }
  });
});
