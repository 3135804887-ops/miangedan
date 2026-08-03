/**
 * Card 系列：卡片、区块头、统计卡（DESIGN-SYSTEM 分层表面）。
 */

import type { HTMLAttributes, ReactNode } from 'react';

export interface CardProps extends HTMLAttributes<HTMLElement> {
  readonly children: ReactNode;
  readonly hoverable?: boolean;
  readonly brand?: boolean;
  readonly as?: 'article' | 'section' | 'div';
}

export function Card({
  children,
  hoverable = false,
  brand = false,
  as: Tag = 'article',
  className = '',
  ...rest
}: CardProps): ReactNode {
  const classes = [
    'mgd-card',
    hoverable ? 'mgd-card--hoverable' : '',
    brand ? 'mgd-card--brand' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ');
  return (
    <Tag className={classes} {...rest}>
      {children}
    </Tag>
  );
}

export function CardHeader({
  title,
  description,
  actions,
}: {
  readonly title: ReactNode;
  readonly description?: ReactNode;
  readonly actions?: ReactNode;
}): ReactNode {
  return (
    <header className="flex flex-wrap items-start justify-between gap-3 border-b border-neutral-100 px-5 py-4">
      <div>
        <h3 className="m-0 font-semibold text-neutral-900">{title}</h3>
        {description !== undefined ? (
          <p className="mt-1 text-sm text-neutral-600">{description}</p>
        ) : null}
      </div>
      {actions !== undefined ? <div className="flex items-center gap-2">{actions}</div> : null}
    </header>
  );
}

export function CardBody({
  children,
  className = '',
}: {
  readonly children: ReactNode;
  readonly className?: string;
}): ReactNode {
  return <div className={`px-5 py-5 ${className}`}>{children}</div>;
}

export function StatCard({
  label,
  value,
  hint,
  tone = 'neutral',
}: {
  readonly label: ReactNode;
  readonly value: ReactNode;
  readonly hint?: ReactNode;
  readonly tone?: 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand';
}): ReactNode {
  const dot = {
    neutral: 'bg-neutral-400',
    success: 'bg-success',
    warning: 'bg-warning',
    danger: 'bg-danger',
    info: 'bg-info',
    brand: 'bg-primary',
  }[tone];
  return (
    <Card className="px-5 py-4">
      <div className="flex items-center gap-2 text-sm font-medium text-neutral-600">
        <span aria-hidden="true" className={`inline-block size-2 rounded-full ${dot}`} />
        {label}
      </div>
      <div className="mgd-stat-value mt-2 text-neutral-900">{value}</div>
      {hint !== undefined ? <div className="mt-1 text-sm text-neutral-600">{hint}</div> : null}
    </Card>
  );
}
