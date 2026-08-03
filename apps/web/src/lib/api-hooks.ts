/**
 * apiFetch 的 React 封装（任务 6：全部页面数据源统一走真实契约端点）。
 *
 * 语义与 api-fetch.ts 完全一致：读操作 GET；写操作必须携带 Idempotency-Key；
 * MSW 开启时由 Mock_Layer 拦截，关闭时直连真实后端 /api/v1/*。
 */

'use client';

import { useEffect, useState } from 'react';

import {
  apiFetch,
  type ApiFailure,
  type ApiPath,
  type ApiResult,
  type ReadInit,
  type WriteInit,
} from './api-fetch.ts';

export interface ResourceState<T> {
  readonly loading: boolean;
  readonly data: T | undefined;
  readonly failure: ApiFailure | undefined;
}

/** 读资源：mount 后拉取一次；pathParams 变化时重新拉取。 */
export function useApiGet<TData>(
  path: ApiPath,
  init: Omit<ReadInit, 'method'>,
): ResourceState<TData> {
  const [state, setState] = useState<ResourceState<TData>>({ loading: true, data: undefined, failure: undefined });
  const key = JSON.stringify(init.pathParams ?? null) + JSON.stringify(init.query ?? null);
  useEffect(() => {
    let alive = true;
    setState({ loading: true, data: undefined, failure: undefined });
    void apiFetch<TData>(path, { ...init, method: 'get' }).then((result) => {
      if (!alive) return;
      setState(result.ok
        ? { loading: false, data: result.data, failure: undefined }
        : { loading: false, data: undefined, failure: result });
    });
    return () => {
      alive = false;
    };
    // init 为每次渲染新建对象，仅以 path 与序列化参数作为依赖（api-fetch 语义不变）。
  }, [path, key]);
  return state;
}

/** 写操作：返回 [触发函数, 进行中, 最近结果]；失败信封原样返回给调用方展示。 */
export function useApiWrite<TData>() {
  const [pending, setPending] = useState(false);
  const [result, setResult] = useState<ApiResult<TData> | undefined>(undefined);
  const run = async (path: ApiPath, init: WriteInit): Promise<ApiResult<TData>> => {
    setPending(true);
    try {
      const res = await apiFetch<TData>(path, init);
      setResult(res);
      return res;
    } finally {
      setPending(false);
    }
  };
  return { run, pending, result } as const;
}

/** 501/未实现端点占位文案（保持占位语义，不伪装真实数据）。 */
export function isPlaceholder(result: { readonly status: number } | undefined): boolean {
  return result?.status === 501;
}
