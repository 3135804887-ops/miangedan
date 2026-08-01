/**
 * StatusBadge：项目态 / 会话态徽标（需求 G1 第 4、5 条）。
 *
 * 强制双通道：图标 + 文字标签，并可附带该状态下的可用动作说明。
 * 三态（通过 / 未通过 / 评估未完成）图标互不相同，移除全部颜色后仍可区分。
 */

import { projectStatusTone, type ProjectStatus, type ResultTone } from '@mgd/domain-states';
import type { ReactNode } from 'react';

import {
  FailIcon,
  IncompleteIcon,
  NeutralIcon,
  PassIcon,
} from '../a11y/status-icons.tsx';

const TONE_ICON: Readonly<Record<ResultTone, () => ReactNode>> = {
  pass: PassIcon,
  fail: FailIcon,
  incomplete: IncompleteIcon,
  neutral: NeutralIcon,
};

const TONE_CLASS: Readonly<Record<ResultTone, string>> = {
  pass: 'mgd-badge--pass',
  fail: 'mgd-badge--fail',
  incomplete: 'mgd-badge--incomplete',
  neutral: 'mgd-badge--neutral',
};

export interface StatusBadgeProps {
  /** 状态标识必须来自 @mgd/domain-states，页面不得传入字面量 */
  readonly status: ProjectStatus;
  /** 用户可见状态文案（由 i18n 提供固定翻译键） */
  readonly label: string;
  /** 该状态下的可用动作说明（G1 第 4 条要求同时渲染） */
  readonly actionHint?: string;
}

export function StatusBadge({ status, label, actionHint }: StatusBadgeProps): ReactNode {
  const tone = projectStatusTone(status);
  const Icon = TONE_ICON[tone];

  return (
    <span
      data-mgd-status={status}
      data-mgd-tone={tone}
      className={['mgd-badge', TONE_CLASS[tone]].join(' ')}
    >
      <Icon />
      <span className="mgd-badge__label">{label}</span>
      {actionHint === undefined ? null : (
        <span className="mgd-badge__hint text-caption">{actionHint}</span>
      )}
    </span>
  );
}
