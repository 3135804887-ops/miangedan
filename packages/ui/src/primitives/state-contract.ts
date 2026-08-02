/**
 * 组件状态契约（DESIGN-SYSTEM 第 8 节、需求 G5 第 4~8 条）。
 *
 * 所有交互组件必须实现七态：default / hover / active / disabled / loading / error / focus-visible。
 * 其中 disabled 必须给出原因、loading 必须输出 aria-busy、error 必须图标 + 文字。
 */

export const INTERACTIVE_VISUAL_STATES = [
  'default',
  'hover',
  'active',
  'disabled',
  'loading',
  'error',
  'focusVisible',
] as const;

export type InteractiveVisualState = (typeof INTERACTIVE_VISUAL_STATES)[number];

export interface InteractiveBaseProps {
  /**
   * 控件标识，渲染为 data-mgd-control。
   * 用途：控件清册扫描测试（「0 个改分控件」红线，需求 G9 第 1 条）。
   */
  readonly controlId?: string;
  /** disabled 必须同时给出原因（DESIGN-SYSTEM 第 8 节 disabled 行）。 */
  readonly disabledReason?: string;
  readonly loading?: boolean;
  /** loading 时的可访问忙碌文案。 */
  readonly busyLabel?: string;
  /** error 态文字说明；与错误图标同时渲染，不允许只变色。 */
  readonly errorMessage?: string;
}

/** 主要控制（≥44px）与普通指针目标（≥24px）。 */
export type TargetSize = 'primary' | 'min';

export const TARGET_SIZE_CLASS: Readonly<Record<TargetSize, string>> = {
  primary: 'mgd-target-primary',
  min: 'mgd-target-min',
};

/**
 * 开发期断言：disabled 却没有原因是实现缺陷，直接抛错而不是静默渲染。
 * 生产构建下降级为无操作，避免影响用户。
 */
export function assertDisabledReason(
  controlId: string,
  disabled: boolean,
  disabledReason: string | undefined,
): void {
  if (!disabled) return;
  if (disabledReason !== undefined && disabledReason.trim().length > 0) return;
  if (process.env.NODE_ENV === 'production') return;

  throw new Error(
    `控件 "${controlId}" 处于 disabled 状态但未提供 disabledReason；` +
      'DESIGN-SYSTEM 第 8 节要求禁用态必须说明原因。',
  );
}
