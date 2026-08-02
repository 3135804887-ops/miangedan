/**
 * Avatar：字母头像（合成环境用；真实用户头像不采集，见隐私红线）。
 */

import type { ReactNode } from 'react';

const PALETTES = [
  'bg-[color-mix(in_srgb,var(--mgd-color-primary)_14%,transparent)] text-[var(--mgd-app-brand-ink)]',
  'bg-[color-mix(in_srgb,var(--mgd-color-info)_14%,transparent)] text-[var(--mgd-color-info)]',
  'bg-[color-mix(in_srgb,var(--mgd-color-success)_14%,transparent)] text-[var(--mgd-color-success-text)]',
  'bg-[color-mix(in_srgb,var(--mgd-color-warning)_14%,transparent)] text-[var(--mgd-color-warning-text)]',
  'bg-[color-mix(in_srgb,var(--mgd-color-danger)_14%,transparent)] text-[var(--mgd-color-danger)]',
];

export function Avatar({
  name,
  size = 40,
}: {
  readonly name: string;
  readonly size?: number;
}): ReactNode {
  const initial = name.trim().charAt(0).toUpperCase() || '?';
  const palette = PALETTES[name.length % PALETTES.length] ?? PALETTES[0]!;
  return (
    <span
      aria-hidden="true"
      className={`inline-grid shrink-0 place-items-center rounded-full font-semibold ${palette}`}
      style={{ inlineSize: size, blockSize: size, fontSize: size * 0.42 }}
    >
      {initial}
    </span>
  );
}
