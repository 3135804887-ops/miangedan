'use client';

import { useId, type ReactNode } from 'react';

import { ErrorIcon } from '../a11y/status-icons.tsx';
import {
  assertDisabledReason,
  TARGET_SIZE_CLASS,
  type InteractiveBaseProps,
} from './state-contract.ts';

export interface SwitchProps extends InteractiveBaseProps {
  readonly checked: boolean;
  readonly label: string;
  readonly disabled?: boolean;
  readonly onCheckedChange?: (checked: boolean) => void;
}

/** 独立开关：不实现群组联动，授权与便利设置必须逐项操作。 */
export function Switch({
  checked,
  label,
  controlId,
  disabled = false,
  disabledReason,
  loading = false,
  busyLabel,
  errorMessage,
  onCheckedChange,
}: SwitchProps): ReactNode {
  const generatedId = useId().replaceAll(':', '');
  const cid = controlId ?? `sw-${generatedId}`;
  assertDisabledReason(cid, disabled, disabledReason);
  const inert = disabled || loading;
  const describedBy = [
    disabled && disabledReason !== undefined ? `${cid}-disabled-reason` : undefined,
    errorMessage !== undefined ? `${cid}-error` : undefined,
  ].filter((id): id is string => id !== undefined);

  return (
    <span className="mgd-switch-wrapper">
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        aria-disabled={inert ? true : undefined}
        aria-busy={loading ? true : undefined}
        aria-describedby={describedBy.length > 0 ? describedBy.join(' ') : undefined}
        tabIndex={inert ? -1 : undefined}
        data-mgd-control={cid}
        data-mgd-state={inert ? (loading ? 'loading' : 'disabled') : errorMessage === undefined ? 'default' : 'error'}
        className={['mgd-switch', TARGET_SIZE_CLASS.min].join(' ')}
        onClick={inert ? undefined : () => onCheckedChange?.(!checked)}
      >
        <span className="mgd-switch__track" aria-hidden="true">
          <span className="mgd-switch__thumb" />
        </span>
        <span>{label}</span>
        {loading && busyLabel !== undefined ? <span>{busyLabel}</span> : null}
      </button>
      {disabled && disabledReason !== undefined ? (
        <span id={`${cid}-disabled-reason`} className="text-caption text-neutral-600">
          {disabledReason}
        </span>
      ) : null}
      {errorMessage !== undefined ? (
        <span id={`${cid}-error`} className="mgd-inline-error" data-mgd-error={cid}>
          <ErrorIcon />
          <span>{errorMessage}</span>
        </span>
      ) : null}
    </span>
  );
}
