/**
 * 机构端可见性投影（docs/design/SCREEN-SPEC.md 第 7 节「完成情况」）。
 *
 * 重要：这些值**不是**面试状态机状态，而是机构默认最小可见视图的投影值。
 * 机构默认不可见个人内容，且「不能显示失败」——因此本枚举内不存在任何表示失败或分数的值。
 * 这些值登记在 no-domain-state-literal 规则的白名单中，与需求 G1 不冲突。
 */

export const ORG_ASSIGNMENT_PROGRESSES = [
  'NOT_STARTED',
  'IN_PROGRESS',
  'COMPLETED_OR_EXITED',
] as const;

export type OrgAssignmentProgress = (typeof ORG_ASSIGNMENT_PROGRESSES)[number];

/** 机构聚合趋势的小样本保护下限（FR-036：细分群体 <10 人不展示）。 */
export const ORG_MIN_COHORT_SIZE = 10;
