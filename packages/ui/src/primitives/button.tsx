/**
 * Button：实现 DESIGN-SYSTEM 第 8 节七态（需求 G5 第 4~8 条）。
 *
 * - disabled：aria-disabled + 移出 Tab 序 + 渲染原因
 * - loading：aria-busy + 忙碌文案
 * - error：错误图标 + 文字
 * - focus-visible：由共享样式提供 ≥2px 焦点环，组件不得设置 outline: none
 * - 目标尺寸：primary ≥44px，min ≥24px
 */

import type { ButtonHTMLAttributes, ReactNode } from 'react';

import { ErrorIcon } from '../a11y/status-icons.tsx';
import {
  assertDisabledReason,
  TARGET_SIZE_CLASS,
  type InteractiveBaseProps,
  type TargetSize,
} from './state-contract.ts';

export type ButtonVariant = 'primary' | 'secondary' | 'danger';

export interface ButtonProps
  extends InteractiveBaseProps,
    Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'disabled' | 'aria-disabled' | 'className'> {
  readonly children: ReactNode;
  readonly variant?: ButtonVariant;
  readonly targetSize?: TargetSize;
  readonly disabled?: boolean;
}

const VARIANT_CLASS: Readonly<Record<ButtonVariant, string>> = {
  primary: 'mgd-button--primary',
  secondary: 'mgd-button--secondary',
  danger: 'mgd-button--danger',
};

export function Button({
  children,
  controlId,
  variant = 'secondary',
  targetSize = 'primary',
  disabled = false,
  disabledReason,
  loading = false,
  busyLabel,
  errorMessage,
  ...rest
}: ButtonProps): ReactNode {
  assertDisabledReason(controlId, disabled, disabledReason);

  const inert = disabled || loading;
  const describedByIds = [
    disabled && disabledReason !== undefined ? `${controlId}-disabled-reason` : undefined,
    errorMessage !== undefined ? `${controlId}-error` : undefined,
  ].filter((id): id is string => id !== undefined);

  return (
    <span className="mgd-button-wrapper">
      <button
        {...rest}
        type={rest.type ?? 'button'}
        data-mgd-control={controlId}
        data-mgd-state={inert ? (loading ? 'loading' : 'disabled') : 'default'}
        className={['mgd-button', VARIANT_CLASS[variant], TARGET_SIZE_CLASS[targetSize]].join(' ')}
        disabled={inert}
        aria-disabled={inert ? true : undefined}
        aria-busy={loading ? true : undefined}
        aria-describedby={describedByIds.length > 0 ? describedByIds.join(' ') : undefined}
        // disabled 态移出 Tab 序（DESIGN-SYSTEM 第 8 节：不可聚焦）
        tabIndex={inert ? -1 : rest.tabIndex}
      >
        {children}
        {loading && busyLabel !== undefined ? (
          <span className="mgd-button__busy">{busyLabel}</span>
        ) : null}
      </button>

      {disabled && disabledReason !== undefined ? (
        <span
          id={`${controlId}-disabled-reason`}
          data-mgd-disabled-reason={controlId}
          className="mgd-button__reason text-caption text-neutral-600"
        >
          {disabledReason}
        </span>
      ) : null}

      {errorMessage !== undefined ? (
        <span
          id={`${controlId}-error`}
          data-mgd-error={controlId}
          className="mgd-button__error text-caption text-danger flex items-center gap-1"
        >
          <ErrorIcon />
          <span>{errorMessage}</span>
        </span>
      ) : null}
    </span>
  );
}
