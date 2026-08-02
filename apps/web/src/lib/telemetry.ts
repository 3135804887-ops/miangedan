/**
 * 遥测出口（需求 G8 第 1、2 条）。
 *
 * 红线：前端日志与错误上报载荷字段限于路由名、状态枚举值、错误码、请求标识与时间戳。
 * 简历正文、完整回答、字幕正文、令牌与媒体引用一律不得进入载荷。
 * 本模块是唯一出口，应用代码禁止直接调用 console.error 或 window.onerror 上报。
 */

import type { ProjectStatus, SessionStatus } from '@mgd/domain-states';

/** 允许出站的字段白名单，顺序即序列化顺序，便于测试稳定断言。 */
export const TELEMETRY_ALLOWED_KEYS = [
  'route',
  'scrId',
  'projectStatus',
  'sessionStatus',
  'errorCode',
  'traceId',
  'at',
] as const;

export type TelemetryKey = (typeof TELEMETRY_ALLOWED_KEYS)[number];

export interface TelemetryEvent {
  /** 归一化路由模板，如 /[locale]/projects/[id]/plan；不得携带真实 id */
  readonly route: string;
  readonly scrId: string;
  readonly projectStatus?: ProjectStatus;
  readonly sessionStatus?: SessionStatus;
  readonly errorCode?: string;
  readonly traceId?: string;
  readonly at?: string;
}

type TelemetrySink = (payload: Readonly<Record<string, string>>) => void;

let sink: TelemetrySink = () => {
  // 批次 0 默认丢弃：真实上报通道在接入可观测服务时注入
};

/** 供测试与后续接入替换出站通道。 */
export function setTelemetrySink(next: TelemetrySink): void {
  sink = next;
}

/** 归一化路由：把动态段替换为占位，避免 id 等标识进入载荷。 */
export function normalizeRoute(pathname: string): string {
  return pathname
    .split('/')
    .map((segment) => {
      if (segment === '') return segment;
      // UUID 或纯数字段一律脱敏为占位
      if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(segment)) {
        return '[id]';
      }
      if (/^\d+$/.test(segment)) return '[n]';
      return segment;
    })
    .join('/');
}

/**
 * 白名单过滤：只保留允许的键，且值一律转为字符串。
 * 传入多余键在开发期抛错（尽早暴露实现缺陷），生产期静默丢弃。
 */
export function reportEvent(event: TelemetryEvent): void {
  const extraKeys = Object.keys(event).filter(
    (key) => !(TELEMETRY_ALLOWED_KEYS as readonly string[]).includes(key),
  );

  if (extraKeys.length > 0) {
    const message =
      `遥测载荷包含白名单外字段：${extraKeys.join('、')}；` +
      '前端上报不得包含正文、令牌或媒体引用（AGENTS.md 第 2 节第 9 条）。';
    if (process.env.NODE_ENV !== 'production') {
      throw new Error(message);
    }
  }

  const payload: Record<string, string> = {};
  for (const key of TELEMETRY_ALLOWED_KEYS) {
    const value = event[key];
    if (value === undefined) continue;
    payload[key] = String(value);
  }
  payload.at = payload.at ?? new Date().toISOString();

  sink(payload);
}
