/**
 * Skeleton：加载骨架（需求 G2 第 3 条）。
 * 容器输出 aria-busy 与可读忙碌文案；尊重 prefers-reduced-motion（样式侧关闭闪烁动效）。
 */

import type { ReactNode } from 'react';

export interface SkeletonProps {
  readonly busyLabel: string;
  /** 骨架行数，用于近似真实内容结构，但不得渲染形如业务数据的数值。 */
  readonly lines?: number;
  /** 长加载的预期时长说明（由页面按 NFR 绑定表传入）。 */
  readonly expectation?: string;
}

export function Skeleton({ busyLabel, lines = 3, expectation }: SkeletonProps): ReactNode {
  return (
    <div
      role="status"
      aria-busy={true}
      aria-live="polite"
      data-mgd-skeleton="true"
      className="mgd-skeleton"
    >
      <span className="mgd-skeleton__label">{busyLabel}</span>
      {expectation === undefined ? null : (
        <span className="mgd-skeleton__expectation text-caption text-neutral-600">
          {expectation}
        </span>
      )}
      <span aria-hidden="true" className="mgd-skeleton__bars">
        {Array.from({ length: lines }, (_unused, index) => (
          <span key={index} className="mgd-skeleton__bar" />
        ))}
      </span>
    </div>
  );
}
