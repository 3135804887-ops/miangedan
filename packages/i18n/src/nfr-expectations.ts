/**
 * 长加载预期时长与 NFR 指标的唯一绑定处（需求 G2 第 3 条、design.md 第 6.3 节）。
 * 数值来源：docs/prd/PRD-001-面个蛋-V1.0.md 的 NFR 表。
 * 页面加载文案只能引用本表，避免各页文案漂移。
 */

export type NfrId = 'NFR-007' | 'NFR-013' | 'NFR-014' | 'NFR-015' | 'NFR-016';

export interface LoadingExpectation {
  readonly nfrId: NfrId;
  readonly seconds: number;
  /** p95 = P95 分位；pct95 = 95% 请求 */
  readonly statistic: 'p95' | 'pct95';
  /** 关联页面（SCR 编号），仅作追踪说明 */
  readonly scrId: string;
}

export const LOADING_EXPECTATIONS: Readonly<Record<NfrId, LoadingExpectation>> = {
  // 数字人音视频建立 95% ≤8 秒
  'NFR-007': { nfrId: 'NFR-007', seconds: 8, statistic: 'pct95', scrId: 'SCR-07' },
  // 单轮评分生成 P95 ≤60 秒
  'NFR-013': { nfrId: 'NFR-013', seconds: 60, statistic: 'p95', scrId: 'SCR-10' },
  // 完整报告生成 P95 ≤120 秒
  'NFR-014': { nfrId: 'NFR-014', seconds: 120, statistic: 'p95', scrId: 'SCR-11' },
  // 10 MB 内简历解析 P95 ≤60 秒
  'NFR-015': { nfrId: 'NFR-015', seconds: 60, statistic: 'p95', scrId: 'SCR-05' },
  // 面试计划生成 P95 ≤120 秒
  'NFR-016': { nfrId: 'NFR-016', seconds: 120, statistic: 'p95', scrId: 'SCR-06' },
};

export function loadingExpectation(nfrId: NfrId): LoadingExpectation {
  return LOADING_EXPECTATIONS[nfrId];
}
