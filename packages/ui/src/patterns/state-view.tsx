/**
 * StateView：五态容器（需求 G2、SCREEN-SPEC 第 9 节）。
 *
 * 页面把数据状态收敛为 PageState，由本容器统一渲染空 / 加载 / 失败 / 权限不足 / 恢复五态，
 * 只有 kind === 'ready' 时才渲染 children。页面不得自行拼装空态或错误态。
 *
 * 空态视图内不渲染任何形如业务数据的数值（G2 第 2 条），由组件测试断言。
 */

import type { ReactNode } from 'react';

import { NeutralIcon } from '../a11y/status-icons.tsx';
import { Skeleton } from '../primitives/skeleton.tsx';
import { ErrorPanel, type ErrorPanelLabels, type UserFacingErrorView } from './error-panel.tsx';

/** 权限获取路径（不暴露他人数据存在性，只说明如何获得权限）。 */
export type AcquirePath = 'signIn' | 'grantConsent' | 'purchase' | 'contactOrgOwner';

export type PageState =
  | { readonly kind: 'ready' }
  | { readonly kind: 'empty'; readonly reason: string; readonly nextAction: ReactNode }
  | { readonly kind: 'loading'; readonly busyLabel: string; readonly expectation?: string }
  | {
      readonly kind: 'error';
      readonly error: UserFacingErrorView;
      readonly retryAction: ReactNode;
    }
  | {
      readonly kind: 'forbidden';
      readonly requiredPermission: string;
      readonly acquirePath: AcquirePath;
      readonly acquireHint: string;
      readonly acquireAction?: ReactNode;
    }
  | {
      readonly kind: 'recovering';
      readonly resumeAt: string;
      readonly resumeAction: ReactNode;
    };

export interface StateViewLabels {
  readonly emptyHeading: string;
  readonly forbiddenHeading: string;
  readonly recoveringHeading: string;
  readonly error: ErrorPanelLabels;
}

export interface StateViewProps {
  readonly state: PageState;
  readonly labels: StateViewLabels;
  /** 区域标签，便于屏幕阅读器定位与测试选择器。 */
  readonly regionLabel: string;
  readonly children: ReactNode;
}

export function StateView({ state, labels, regionLabel, children }: StateViewProps): ReactNode {
  if (state.kind === 'ready') {
    return (
      <div data-mgd-state-view="ready" data-mgd-region={regionLabel}>
        {children}
      </div>
    );
  }

  if (state.kind === 'loading') {
    return (
      <div data-mgd-state-view="loading" data-mgd-region={regionLabel}>
        <Skeleton busyLabel={state.busyLabel} expectation={state.expectation} />
      </div>
    );
  }

  if (state.kind === 'empty') {
    return (
      <div data-mgd-state-view="empty" data-mgd-region={regionLabel} className="mgd-state-empty">
        <p className="mgd-state-empty__heading flex items-center gap-2">
          <NeutralIcon />
          <span>{labels.emptyHeading}</span>
        </p>
        <p className="mgd-state-empty__reason">{state.reason}</p>
        <div className="mgd-state-empty__action">{state.nextAction}</div>
      </div>
    );
  }

  if (state.kind === 'error') {
    return (
      <div data-mgd-state-view="error" data-mgd-region={regionLabel}>
        <ErrorPanel error={state.error} labels={labels.error} action={state.retryAction} />
      </div>
    );
  }

  if (state.kind === 'forbidden') {
    return (
      <div
        data-mgd-state-view="forbidden"
        data-mgd-region={regionLabel}
        data-mgd-acquire-path={state.acquirePath}
        className="mgd-state-forbidden"
      >
        <p className="mgd-state-forbidden__heading flex items-center gap-2">
          <NeutralIcon />
          <span>{labels.forbiddenHeading}</span>
        </p>
        <p className="mgd-state-forbidden__permission">{state.requiredPermission}</p>
        <p className="mgd-state-forbidden__hint">{state.acquireHint}</p>
        {state.acquireAction === undefined ? null : (
          <div className="mgd-state-forbidden__action">{state.acquireAction}</div>
        )}
      </div>
    );
  }

  return (
    <div
      data-mgd-state-view="recovering"
      data-mgd-region={regionLabel}
      className="mgd-state-recovering"
    >
      <p className="mgd-state-recovering__heading flex items-center gap-2">
        <NeutralIcon />
        <span>{labels.recoveringHeading}</span>
      </p>
      <p className="mgd-state-recovering__resume">{state.resumeAt}</p>
      <div className="mgd-state-recovering__action">{state.resumeAction}</div>
    </div>
  );
}
