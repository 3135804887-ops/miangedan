/**
 * Tint：语义色块标签（状态不只靠颜色表达：图标/文字并出）。
 */

import type { ReactNode } from 'react';

export type TintTone = 'neutral' | 'info' | 'warning' | 'success' | 'danger' | 'brand';

export function Tint({
  tone,
  children,
  className = '',
}: {
  readonly tone: TintTone;
  readonly children: ReactNode;
  readonly className?: string;
}): ReactNode {
  return <span className={`mgd-tint mgd-tint--${tone} ${className}`.trim()}>{children}</span>;
}
