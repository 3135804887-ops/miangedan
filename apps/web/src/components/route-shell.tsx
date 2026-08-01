/**
 * 路由壳（需求 B0-2 第 1 条）。
 *
 * 批次 0 只渲染页面标题与占位说明，页面内容在批次 1~4 落地。
 * 使用统一组件而不是各页手写，保证标题层级、区域语义与说明文案一致。
 */

import type { ReactNode } from 'react';

export interface RouteShellProps {
  /** SCREEN-SPEC 页面编号，如 SCR-03 */
  readonly scrId: string;
  readonly title: string;
  readonly notice: string;
  /** 房间页等需要全宽布局的页面传 true */
  readonly fullWidth?: boolean;
  readonly children?: ReactNode;
}

export function RouteShell({
  scrId,
  title,
  notice,
  fullWidth = false,
  children,
}: RouteShellProps): ReactNode {
  return (
    <section
      data-mgd-scr={scrId}
      data-mgd-route-shell="true"
      className={fullWidth ? 'mgd-shell--full' : undefined}
    >
      <h1>{title}</h1>
      <p className="text-caption text-neutral-600">
        <span data-mgd-scr-label>{scrId}</span> · {notice}
      </p>
      {children}
    </section>
  );
}
