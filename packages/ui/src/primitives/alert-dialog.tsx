'use client';

import { useRef, type ReactNode, type RefObject } from 'react';

import { useFocusTrap } from '../a11y/focus-trap.ts';

export interface AlertDialogProps {
  readonly open: boolean;
  readonly title: string;
  readonly description: string;
  readonly children: ReactNode;
  readonly onClose?: () => void;
  readonly returnFocusTo?: RefObject<HTMLElement | null>;
}

/** 故障覆盖层与破坏性确认的共享模态容器。 */
export function AlertDialog({
  open,
  title,
  description,
  children,
  onClose,
  returnFocusTo,
}: AlertDialogProps): ReactNode {
  const dialogRef = useRef<HTMLDivElement>(null);
  useFocusTrap(dialogRef, open, returnFocusTo);
  if (!open) return null;

  return (
    <div className="mgd-dialog-backdrop">
      <div
        ref={dialogRef}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="mgd-alert-dialog-title"
        aria-describedby="mgd-alert-dialog-description"
        className="mgd-alert-dialog"
        tabIndex={-1}
        onKeyDown={(event) => {
          if (event.key === 'Escape') onClose?.();
        }}
      >
        <h2 id="mgd-alert-dialog-title">{title}</h2>
        <p id="mgd-alert-dialog-description" role="alert">{description}</p>
        <div className="mgd-alert-dialog__actions">{children}</div>
      </div>
    </div>
  );
}
