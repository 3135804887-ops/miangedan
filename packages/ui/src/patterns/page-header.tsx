/**
 * PageHeader：页面标题区（kicker/标题/描述/操作）。
 */

import type { ReactNode } from 'react';

export function PageHeader({
  kicker,
  title,
  description,
  actions,
}: {
  readonly kicker?: ReactNode;
  readonly title: ReactNode;
  readonly description?: ReactNode;
  readonly actions?: ReactNode;
}): ReactNode {
  return (
    <header className="mgd-page-header">
      <div>
        {kicker !== undefined ? <div className="mgd-page-header__kicker">{kicker}</div> : null}
        <h1 className="mgd-page-header__title">{title}</h1>
        {description !== undefined ? <p className="mgd-page-header__desc">{description}</p> : null}
      </div>
      {actions !== undefined ? (
        <div className="flex flex-wrap items-center gap-3">{actions}</div>
      ) : null}
    </header>
  );
}
