/**
 * 状态图标（ACCESSIBILITY 第 4.1 节「非颜色唯一提示」）。
 *
 * 通过 / 未通过 / 评估未完成三态必须各配不同图标与文字标签；
 * 图标本身 aria-hidden，语义由同级文字承担，避免屏幕阅读器重复朗读。
 * 强制颜色模式下依靠形状与边框区分，不依赖填充色。
 */

import type { ReactNode } from 'react';

interface IconProps {
  readonly className?: string;
}

function iconClass(extra: string | undefined): string {
  return ['mgd-icon', extra].filter((value) => value !== undefined).join(' ');
}

/** 通过态：对勾 */
export function PassIcon({ className }: IconProps): ReactNode {
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      viewBox="0 0 16 16"
      width="16"
      height="16"
      className={iconClass(className)}
      data-mgd-icon="pass"
    >
      <path d="M2 8.5 6 12.5 14 4" fill="none" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}

/** 未通过态：叉号 */
export function FailIcon({ className }: IconProps): ReactNode {
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      viewBox="0 0 16 16"
      width="16"
      height="16"
      className={iconClass(className)}
      data-mgd-icon="fail"
    >
      <path d="M3 3 13 13 M13 3 3 13" fill="none" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}

/** 评估未完成：暂停双竖线（区别于失败，语义为「不是失败」） */
export function IncompleteIcon({ className }: IconProps): ReactNode {
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      viewBox="0 0 16 16"
      width="16"
      height="16"
      className={iconClass(className)}
      data-mgd-icon="incomplete"
    >
      <path d="M5 3 5 13 M11 3 11 13" fill="none" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}

/** 中性提示：信息圆点 */
export function NeutralIcon({ className }: IconProps): ReactNode {
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      viewBox="0 0 16 16"
      width="16"
      height="16"
      className={iconClass(className)}
      data-mgd-icon="neutral"
    >
      <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" strokeWidth="2" />
      <path d="M8 7 8 11" fill="none" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}

/** 错误图标：三角警示，供 ErrorPanel 与 error 态控件使用 */
export function ErrorIcon({ className }: IconProps): ReactNode {
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      viewBox="0 0 16 16"
      width="16"
      height="16"
      className={iconClass(className)}
      data-mgd-icon="error"
    >
      <path d="M8 2 15 14 1 14Z" fill="none" stroke="currentColor" strokeWidth="2" />
      <path d="M8 6 8 10" fill="none" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}
