import type { ReactNode } from 'react';

import { Button, type ButtonProps } from './button.tsx';

export interface IconButtonProps extends Omit<ButtonProps, 'children'> {
  readonly label: string;
  readonly icon: ReactNode;
}

/** 只有图标的控制必须提供可访问名称，命中区沿用主要 Button 的 44px 基线。 */
export function IconButton({ label, icon, ...props }: IconButtonProps): ReactNode {
  return (
    <Button {...props} aria-label={label}>
      <span aria-hidden="true">{icon}</span>
      <span className="mgd-visually-hidden">{label}</span>
    </Button>
  );
}
