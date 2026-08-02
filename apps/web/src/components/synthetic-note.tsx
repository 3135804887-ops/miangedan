import type { ReactNode } from 'react';

/** 合成数据标注（Mock_Layer：禁止把演示数据伪装成真实数据）。 */
export function SyntheticNote(): ReactNode {
  return (
    <p className="mb-0 inline-flex items-center gap-2 rounded-full bg-[var(--mgd-app-surface-sunken)] px-3 py-1 text-xs text-neutral-600">
      <span aria-hidden="true" className="inline-block size-1.5 rounded-full bg-warning" />
      合成演示数据 · Synthetic demo data
    </p>
  );
}
