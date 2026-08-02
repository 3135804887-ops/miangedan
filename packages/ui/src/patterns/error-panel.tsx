/**
 * ErrorPanel：错误五要素面板（需求 G2 第 4 条、SCREEN-SPEC 第 4 节第 2 条）。
 *
 * 五要素必须同时出现：影响、数据是否保留、可重试动作、是否计费、是否影响评分。
 * 容器为 role="alert"，错误图标与文字同时渲染（不允许只变红）。
 */

import type { ReactNode } from 'react';

import { ErrorIcon } from '../a11y/status-icons.tsx';

export interface UserFacingErrorView {
  /** openapi components.schemas.ErrorCode */
  readonly code: string;
  readonly traceId: string;
  /** ①影响 */
  readonly impact: string;
  /** ②数据是否保留 */
  readonly dataRetained: string;
  /** ③可重试动作 */
  readonly retryAction: string;
  /** ④是否计费 */
  readonly billing: string;
  /** ⑤是否影响评分 */
  readonly scoring: string;
}

export interface ErrorPanelLabels {
  readonly heading: string;
  readonly impact: string;
  readonly dataRetained: string;
  readonly retryAction: string;
  readonly billing: string;
  readonly scoring: string;
  readonly traceLabel: string;
}

export interface ErrorPanelProps {
  readonly error: UserFacingErrorView;
  readonly labels: ErrorPanelLabels;
  /** 重试控件；由调用方传入以便绑定页面自身的重试动作。 */
  readonly action?: ReactNode;
}

interface FacetRow {
  readonly key: keyof ErrorPanelLabels;
  readonly label: string;
  readonly value: string;
}

export function ErrorPanel({ error, labels, action }: ErrorPanelProps): ReactNode {
  const facets: readonly FacetRow[] = [
    { key: 'impact', label: labels.impact, value: error.impact },
    { key: 'dataRetained', label: labels.dataRetained, value: error.dataRetained },
    { key: 'retryAction', label: labels.retryAction, value: error.retryAction },
    { key: 'billing', label: labels.billing, value: error.billing },
    { key: 'scoring', label: labels.scoring, value: error.scoring },
  ];

  return (
    <div
      role="alert"
      data-mgd-error-panel={error.code}
      className="mgd-error-panel border-danger rounded border p-4"
    >
      <p className="mgd-error-panel__heading flex items-center gap-2 font-medium">
        <ErrorIcon />
        <span>{labels.heading}</span>
      </p>

      <dl className="mgd-error-panel__facets">
        {facets.map((facet) => (
          <div key={facet.key} data-mgd-error-facet={facet.key}>
            <dt>{facet.label}</dt>
            <dd>{facet.value}</dd>
          </div>
        ))}
      </dl>

      <p className="mgd-error-panel__trace text-caption text-neutral-600">
        {labels.traceLabel}: <code>{error.traceId}</code>
      </p>

      {action === undefined ? null : <div className="mgd-error-panel__action">{action}</div>}
    </div>
  );
}
