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
export { IconButton, type IconButtonProps } from './primitives/icon-button.tsx';
export { Switch, type SwitchProps } from './primitives/switch.tsx';
export { Field, type FieldProps } from './primitives/field.tsx';
export { AlertDialog, type AlertDialogProps } from './primitives/alert-dialog.tsx';
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
export { DisclosureNote, type DisclosureNoteProps } from './patterns/disclosure-note.tsx';
export {
  StateView,
  type AcquirePath,
  type PageState,
  type StateViewLabels,
  type StateViewProps,
} from './patterns/state-view.tsx';

export {
  IconAlert,
  IconArrowLeft,
  IconArrowRight,
  IconBill,
  IconCamera,
  IconCameraOff,
  IconCertificate,
  IconChart,
  IconCheck,
  IconClipboard,
  IconClock,
  IconCode,
  IconCopy,
  IconDashboard,
  IconDownload,
  IconEdit,
  IconExternal,
  IconFile,
  IconGlobe,
  IconHome,
  IconKeyboard,
  IconLock,
  IconLogout,
  IconMenu,
  IconMessage,
  IconMic,
  IconMicOff,
  IconOrg,
  IconPlan,
  IconPlay,
  IconPlus,
  IconRadar,
  IconRefresh,
  IconSearch,
  IconSettings,
  IconShield,
  IconSparkle,
  IconStop,
  IconTools,
  IconTrash,
  IconUpload,
  IconUser,
  IconUsers,
  IconWhiteboard,
  IconWifi,
  IconX,
  type IconProps,
} from './primitives/icons.tsx';

export { Card, CardBody, CardHeader, StatCard, type CardProps } from './patterns/card.tsx';
export { PageHeader } from './patterns/page-header.tsx';
export { EmptyState } from './patterns/empty-state.tsx';
export {
  AppShell,
  type NavGroup,
  type NavItem,
} from './patterns/app-shell.tsx';
export { Tabs, type TabItem } from './patterns/tabs.tsx';
export { Indeterminate, Progress } from './patterns/progress.tsx';
export { ToastProvider, useToast, type ToastInput } from './patterns/toast.tsx';
export { DataTable, type Column } from './patterns/data-table.tsx';
export { Avatar } from './patterns/avatar.tsx';
export { Tint, type TintTone } from './patterns/tint.tsx';

export {
  collectControls,
  findForbiddenControls,
  FORBIDDEN_CONTROL_PATTERNS,
  type ControlEntry,
  type ForbiddenControlHit,
} from './testing/control-registry.ts';
