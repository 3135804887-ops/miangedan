/**
 * 错误五要素映射（需求 G2 第 4 条）。
 *
 * 输入：openapi components.schemas.Error 信封
 * 输出：UserFacingErrorView（影响 / 数据是否保留 / 可重试动作 / 是否计费 / 是否影响评分）
 *
 * 文案来自 @mgd/i18n 的 error.<code>.<facet> 键；缺失映射由翻译键门禁在 CI 拦截，
 * 因此运行期不会回落到原始键名。
 */

import type { components } from '@mgd/api-types';
import type { UserFacingErrorView } from '@mgd/ui';

export type ErrorCode = components['schemas']['ErrorCode'];

/** openapi ErrorCode 的 21 项，用于兜底判定与测试遍历。 */
export const ERROR_CODES: readonly ErrorCode[] = [
  'unauthorized',
  'forbidden',
  'not_found',
  'conflict',
  'validation_failed',
  'upload_rejected',
  'scan_retryable',
  'parse_retryable',
  'sensitive_content_rejected',
  'low_confidence_unresolved',
  'idempotency_conflict',
  'rate_limited',
  'risk_verification_required',
  'verification_invalid',
  'verification_expired',
  'identity_conflict',
  'provider_unavailable',
  'region_mismatch',
  'insufficient_entitlement',
  'state_conflict',
  'internal',
] as const;

const FALLBACK_CODE: ErrorCode = 'internal';

export const ERROR_FACETS = [
  'impact',
  'dataRetained',
  'retryAction',
  'billing',
  'scoring',
] as const;

export type ErrorFacet = (typeof ERROR_FACETS)[number];

export interface ErrorEnvelope {
  readonly error: {
    readonly code: string;
    readonly message?: string;
    readonly trace_id: string;
    readonly data_region: string;
  };
}

/** 翻译函数签名，与 next-intl 的 useTranslations() 返回值兼容。 */
export type Translate = (key: string) => string;

export function isErrorCode(value: string): value is ErrorCode {
  return (ERROR_CODES as readonly string[]).includes(value);
}

export function parseErrorEnvelope(body: unknown): ErrorEnvelope | undefined {
  if (body === null || typeof body !== 'object') return undefined;
  const candidate = (body as { error?: unknown }).error;
  if (candidate === null || typeof candidate !== 'object') return undefined;

  const { code, trace_id: traceId, data_region: dataRegion } = candidate as Record<string, unknown>;
  if (typeof code !== 'string' || typeof traceId !== 'string' || typeof dataRegion !== 'string') {
    return undefined;
  }

  return { error: { code, trace_id: traceId, data_region: dataRegion } };
}

/**
 * 把错误信封转为五要素视图。
 * scrId 用于页面级文案覆写：存在 error.<code>.<scrId>.<facet> 时优先使用。
 */
export function presentError(
  envelope: ErrorEnvelope | undefined,
  t: Translate,
  options: { readonly scrId?: string } = {},
): UserFacingErrorView {
  const rawCode = envelope?.error.code ?? FALLBACK_CODE;
  const code: ErrorCode = isErrorCode(rawCode) ? rawCode : FALLBACK_CODE;
  const traceId = envelope?.error.trace_id ?? 'unknown';

  const read = (facet: ErrorFacet): string => {
    if (options.scrId !== undefined) {
      const override = `error.${code}.${options.scrId}.${facet}`;
      const value = t(override);
      // 页面级覆写存在时 t() 返回实际文案；不存在时 next-intl 返回键名本身
      if (value !== override) return value;
    }
    return t(`error.${code}.${facet}`);
  };

  return {
    code,
    traceId,
    impact: read('impact'),
    dataRetained: read('dataRetained'),
    retryAction: read('retryAction'),
    billing: read('billing'),
    scoring: read('scoring'),
  };
}
