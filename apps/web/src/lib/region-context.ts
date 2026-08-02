/**
 * 数据区只读上下文（需求 G8 第 4 条）。
 *
 * 数据区由后端在响应体的 data_region 字段给出，前端只做展示。
 * 本模块不提供任何切换数据区或跨区取数的入口——三数据区相互隔离，
 * 不为容灾、成本或便利建立跨区通道（AGENTS.md 第 2 节第 6 条）。
 */

import type { components } from '@mgd/api-types';

export type DataRegion = components['schemas']['Region'];

export const DATA_REGIONS: readonly DataRegion[] = ['cn', 'eu', 'intl'];

export function isDataRegion(value: string): value is DataRegion {
  return (DATA_REGIONS as readonly string[]).includes(value);
}

/**
 * 从响应体读取数据区。无法识别时返回 undefined 而不是猜测，
 * 避免在界面上显示错误的区域归属。
 */
export function readDataRegion(body: unknown): DataRegion | undefined {
  if (body === null || typeof body !== 'object') return undefined;
  const candidate = (body as { data_region?: unknown }).data_region;
  if (typeof candidate !== 'string') return undefined;
  return isDataRegion(candidate) ? candidate : undefined;
}
