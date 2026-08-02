/**
 * 项目状态枚举。唯一事实源：docs/domain/INTERVIEW-STATE-MACHINE.md 第 5.1 节，
 * 并与 docs/api/openapi.yaml 的 components.schemas.ProjectStatus 做编译期等价断言
 * （见 assert-contract.ts）。页面层禁止自创状态名（需求 G1）。
 */

export const PROJECT_STATUSES = [
  'DRAFT',
  'PARSING',
  'MATERIAL_REVIEW',
  'PARSE_FAILED',
  'PLAN_GENERATING',
  'PLAN_REVIEW',
  'PLAN_FAILED',
  'READY',
  'IN_SESSION',
  'SCORING',
  'ROUND_PASSED',
  'ROUND_FAILED',
  'PRACTICING',
  'EVALUATION_INCOMPLETE',
  'COMPLETED',
] as const;

export type ProjectStatus = (typeof PROJECT_STATUSES)[number];

const PROJECT_STATUS_SET: ReadonlySet<string> = new Set(PROJECT_STATUSES);

export function isProjectStatus(value: string): value is ProjectStatus {
  return PROJECT_STATUS_SET.has(value);
}

/**
 * 三态语义分类（DESIGN-SYSTEM 第 5 节）：通过 / 未通过 / 评估未完成必须各配不同图标与文字，
 * 不得仅靠颜色区分。此处只做分类，具体图标与文案由 packages/ui 与 i18n 提供。
 */
export type ResultTone = 'pass' | 'fail' | 'incomplete' | 'neutral';

export function projectStatusTone(status: ProjectStatus): ResultTone {
  switch (status) {
    case 'ROUND_PASSED':
    case 'COMPLETED':
      return 'pass';
    case 'ROUND_FAILED':
      return 'fail';
    case 'EVALUATION_INCOMPLETE':
      return 'incomplete';
    default:
      return 'neutral';
  }
}

/** 终态：COMPLETED 为正常终态；EVALUATION_INCOMPLETE 允许重试，不是终态。 */
export function isTerminalProjectStatus(status: ProjectStatus): boolean {
  return status === 'COMPLETED';
}
