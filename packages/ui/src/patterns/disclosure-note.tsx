import type { ReactNode } from 'react';

export interface DisclosureNoteProps {
  readonly title: string;
  readonly children: ReactNode;
}

/** 评分门槛、冻结规则等不可编辑内容的只读说明区。 */
export function DisclosureNote({ title, children }: DisclosureNoteProps): ReactNode {
  return (
    <section className="mgd-disclosure-note" aria-readonly="true" aria-label={title}>
      <h2>{title}</h2>
      <div>{children}</div>
    </section>
  );
}
