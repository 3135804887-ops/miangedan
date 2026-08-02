/**
 * DataTable：语义表格（可排序表头、空态由调用方提供）。
 */

import type { ReactNode } from 'react';

export interface Column<T> {
  readonly key: string;
  readonly header: ReactNode;
  readonly render: (row: T) => ReactNode;
  readonly align?: 'start' | 'end';
}

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  empty,
  caption,
}: {
  readonly columns: readonly Column<T>[];
  readonly rows: readonly T[];
  readonly rowKey: (row: T) => string;
  readonly empty?: ReactNode;
  readonly caption?: ReactNode;
}): ReactNode {
  if (rows.length === 0 && empty !== undefined) {
    return <>{empty}</>;
  }
  return (
    <div className="overflow-x-auto rounded-xl border border-neutral-100 bg-surface shadow-[var(--mgd-app-shadow-sm)]">
      <table className="w-full border-collapse text-left text-sm">
        {caption !== undefined ? (
          <caption className="sr-only">{caption}</caption>
        ) : null}
        <thead>
          <tr className="border-b border-neutral-100 bg-surface-muted">
            {columns.map((c) => (
              <th
                key={c.key}
                scope="col"
                className={`px-5 py-3 text-xs font-semibold uppercase tracking-wide text-neutral-600 ${
                  c.align === 'end' ? 'text-right' : ''
                }`}
              >
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={rowKey(row)}
              className="border-b border-neutral-100 last:border-0 hover:bg-surface-muted/60"
            >
              {columns.map((c) => (
                <td
                  key={c.key}
                  className={`px-5 py-3.5 align-top text-neutral-700 ${
                    c.align === 'end' ? 'text-right' : ''
                  }`}
                >
                  {c.render(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
