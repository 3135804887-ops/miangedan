/**
 * API 调用封装（需求 G6 第 3~5 条）。
 *
 * 页面写标准 fetch 语义、用生成类型标注请求与响应；开发与测试期由 Mock_Layer（MSW）拦截，
 * 关闭 Mock_Layer 后同一份页面代码直连真实 API，无需改写。
 *
 * 写操作必须携带 Idempotency-Key（openapi components.parameters.IdempotencyKey），
 * 由类型约束强制（缺失即类型错误）。
 */

import type { paths } from '@mgd/api-types';

import { parseErrorEnvelope, type ErrorEnvelope } from './error-presenter.ts';
import { readDataRegion, type DataRegion } from './region-context.ts';

export type ApiPath = keyof paths;

export type WriteMethod = 'post' | 'patch' | 'delete' | 'put';

export type ApiSuccess<T> = {
  readonly ok: true;
  readonly status: number;
  readonly data: T;
  readonly dataRegion: DataRegion | undefined;
};

export type ApiFailure = {
  readonly ok: false;
  readonly status: number;
  readonly envelope: ErrorEnvelope | undefined;
};

export type ApiResult<T> = ApiSuccess<T> | ApiFailure;

interface BaseInit {
  readonly pathParams?: Readonly<Record<string, string | number>>;
  readonly query?: Readonly<Record<string, string | number | boolean | undefined>>;
  readonly body?: unknown;
  readonly signal?: AbortSignal;
}

/** 读操作：无需幂等键。 */
export interface ReadInit extends BaseInit {
  readonly method: 'get';
}

/** 写操作：幂等键必填，重复提交返回首个结果、不产生重复副作用（NFR-006）。 */
export interface WriteInit extends BaseInit {
  readonly method: WriteMethod;
  readonly idempotencyKey: string;
}

export type ApiFetchInit = ReadInit | WriteInit;

export const API_BASE_PATH = '/api';

function interpolate(path: string, pathParams: BaseInit['pathParams']): string {
  if (pathParams === undefined) return path;

  return path.replace(/\{([A-Za-z0-9_]+)\}/g, (_match, key: string) => {
    const value = pathParams[key];
    if (value === undefined) {
      throw new Error(`路径参数缺失：${key}（路径 ${path}）`);
    }
    return encodeURIComponent(String(value));
  });
}

function buildQuery(query: BaseInit['query']): string {
  if (query === undefined) return '';
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined) continue;
    params.set(key, String(value));
  }
  const serialized = params.toString();
  return serialized === '' ? '' : `?${serialized}`;
}

/**
 * 生成幂等键。同一用户动作在其生命周期内复用同一个键，
 * 使重复点击与网络重试不产生重复副作用。
 */
export function newIdempotencyKey(): string {
  return globalThis.crypto.randomUUID();
}

export async function apiFetch<TData>(
  path: ApiPath,
  init: ApiFetchInit,
): Promise<ApiResult<TData>> {
  const url = `${API_BASE_PATH}${interpolate(path, init.pathParams)}${buildQuery(init.query)}`;

  const headers: Record<string, string> = { Accept: 'application/json' };
  if (init.body !== undefined) headers['Content-Type'] = 'application/json';
  if (init.method !== 'get') headers['Idempotency-Key'] = init.idempotencyKey;

  const response = await fetch(url, {
    method: init.method.toUpperCase(),
    headers,
    body: init.body === undefined ? undefined : JSON.stringify(init.body),
    signal: init.signal,
  });

  const text = await response.text();
  const parsed: unknown = text === '' ? undefined : JSON.parse(text);

  if (!response.ok) {
    return { ok: false, status: response.status, envelope: parseErrorEnvelope(parsed) };
  }

  return {
    ok: true,
    status: response.status,
    data: parsed as TData,
    dataRegion: readDataRegion(parsed),
  };
}
