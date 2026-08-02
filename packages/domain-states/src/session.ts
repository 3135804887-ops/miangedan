/**
 * 实时房间会话状态枚举。唯一事实源：docs/domain/INTERVIEW-STATE-MACHINE.md 第 6.2 节，
 * 并与 docs/api/openapi.yaml 的 components.schemas.Session.room_status 做编译期等价断言。
 */

export const SESSION_STATUSES = [
  'ROOM_CREATED',
  'PRE_CHECK',
  'AVATAR_CONNECTING',
  'LIVE',
  'PAUSED_SYSTEM',
  'RECONNECTING',
  'DOWNGRADE_PROMPTED',
  'TEXT_DEGRADED',
  'AUTH_PAUSED',
  'ENDED',
] as const;

export type SessionStatus = (typeof SESSION_STATUSES)[number];

const SESSION_STATUS_SET: ReadonlySet<string> = new Set(SESSION_STATUSES);

export function isSessionStatus(value: string): value is SessionStatus {
  return SESSION_STATUS_SET.has(value);
}

/**
 * 计时是否进行中（INTERVIEW-STATE-MACHINE 第 6.2 节表格「计时」列）。
 * 仅 LIVE 与 TEXT_DEGRADED 计时；其余状态暂停或未开始。
 */
export function isTimerRunning(status: SessionStatus): boolean {
  return status === 'LIVE' || status === 'TEXT_DEGRADED';
}

/**
 * 是否消耗数字人计费额度（同表「计费」列）。
 * TEXT_DEGRADED 自故障点起不再消耗数字人额度，因此为 false。
 */
export function isAvatarBillable(status: SessionStatus): boolean {
  return status === 'LIVE';
}

/** 系统责任导致的暂停：不判失败、不计时、不计费。 */
export function isSystemPause(status: SessionStatus): boolean {
  return status === 'PAUSED_SYSTEM' || status === 'RECONNECTING' || status === 'AUTH_PAUSED';
}
