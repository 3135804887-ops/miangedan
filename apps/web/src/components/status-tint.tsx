/** 项目状态徽标：状态机枚举 → 双语文案与语义色（SCREEN-SPEC 4.1：禁止自创状态名）。 */

import type { ReactNode } from 'react';

const STATUS_META: Readonly<Record<string, { tone: 'neutral' | 'info' | 'warning' | 'success' | 'danger' | 'brand'; zh: string; en: string }>> = {
  DRAFT: { tone: 'neutral', zh: '草稿', en: 'Draft' },
  PARSING: { tone: 'info', zh: '解析中', en: 'Parsing' },
  MATERIAL_REVIEW: { tone: 'info', zh: '材料校对', en: 'Material review' },
  PARSE_FAILED: { tone: 'danger', zh: '解析失败', en: 'Parse failed' },
  PLAN_GENERATING: { tone: 'info', zh: '计划生成中', en: 'Generating plan' },
  PLAN_REVIEW: { tone: 'warning', zh: '计划待确认', en: 'Plan review' },
  PLAN_FAILED: { tone: 'danger', zh: '计划生成失败', en: 'Plan failed' },
  READY: { tone: 'success', zh: '就绪', en: 'Ready' },
  IN_SESSION: { tone: 'brand', zh: '面试中', en: 'In session' },
  SCORING: { tone: 'info', zh: '评分中', en: 'Scoring' },
  ROUND_PASSED: { tone: 'success', zh: '本轮通过', en: 'Round passed' },
  ROUND_FAILED: { tone: 'danger', zh: '本轮未通过', en: 'Round failed' },
  PRACTICING: { tone: 'warning', zh: '练习中', en: 'Practicing' },
  EVALUATION_INCOMPLETE: { tone: 'info', zh: '评估未完成', en: 'Evaluation incomplete' },
  COMPLETED: { tone: 'success', zh: '已完成', en: 'Completed' },
};

export function StatusTint({
  status,
  locale,
}: {
  readonly status: string;
  readonly locale: 'zh-CN' | 'en-US';
}): ReactNode {
  const meta = STATUS_META[status] ?? { tone: 'neutral' as const, zh: status, en: status };
  const label = locale === 'zh-CN' ? meta.zh : meta.en;
  return <span className={`mgd-tint mgd-tint--${meta.tone}`}>{label}</span>;
}

export function Tint({
  tone,
  children,
}: {
  readonly tone: 'neutral' | 'info' | 'warning' | 'success' | 'danger' | 'brand';
  readonly children: ReactNode;
}): ReactNode {
  return <span className={`mgd-tint mgd-tint--${tone}`}>{children}</span>;
}
