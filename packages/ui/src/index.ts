/**
 * @mgd/ui 公共入口：实现 DESIGN-SYSTEM 第 8 节组件状态清单与 ACCESSIBILITY 的无障碍约定。
 */

export {
  ErrorIcon,
  FailIcon,
  IncompleteIcon,
  NeutralIcon,
  PassIcon,
} from './a11y/status-icons.tsx';

export { Button, type ButtonProps, type ButtonVariant } from './primitives/button.tsx';
export { Skeleton, type SkeletonProps } from './primitives/skeleton.tsx';
export {
  assertDisabledReason,
  INTERACTIVE_VISUAL_STATES,
  TARGET_SIZE_CLASS,
  type InteractiveBaseProps,
  type InteractiveVisualState,
  type TargetSize,
} from './primitives/state-contract.ts';

export {
  ErrorPanel,
  type ErrorPanelLabels,
  type ErrorPanelProps,
  type UserFacingErrorView,
} from './patterns/error-panel.tsx';
export { StatusBadge, type StatusBadgeProps } from './patterns/status-badge.tsx';
export {
  StateView,
  type AcquirePath,
  type PageState,
  type StateViewLabels,
  type StateViewProps,
} from './patterns/state-view.tsx';

export {
  collectControls,
  findForbiddenControls,
  FORBIDDEN_CONTROL_PATTERNS,
  type ControlEntry,
  type ForbiddenControlHit,
} from './testing/control-registry.ts';
