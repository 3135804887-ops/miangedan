'use client';

import { Button, StateView, type PageState, type StateViewLabels } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import type { PreviewState } from '../lib/preview-state.ts';

export type { PreviewState } from '../lib/preview-state.ts';

interface PageStateBoundaryProps {
  readonly children: ReactNode;
  readonly mode: PreviewState;
  readonly regionLabel: string;
  readonly emptyReason: string;
  readonly loadingExpectation?: string;
  readonly forbiddenPermission: string;
  readonly recoveryPoint: string;
}

export function PageStateBoundary({
  children,
  mode,
  regionLabel,
  emptyReason,
  loadingExpectation,
  forbiddenPermission,
  recoveryPoint,
}: PageStateBoundaryProps): ReactNode {
  const common = useTranslations('common');
  const error = useTranslations('error');

  const retry = (
    <Button controlId={`${regionLabel.toLowerCase()}-retry`} onClick={() => window.location.reload()}>
      {common('state.retry')}
    </Button>
  );

  const state: PageState = (() => {
    switch (mode) {
      case 'empty':
        return { kind: 'empty', reason: emptyReason, nextAction: retry };
      case 'loading':
        return {
          kind: 'loading',
          busyLabel: common('state.loadingLabel'),
          expectation: loadingExpectation,
        };
      case 'error':
        return {
          kind: 'error',
          error: {
            code: 'internal',
            impact: error('internal.impact'),
            dataRetained: error('internal.dataRetained'),
            retryAction: error('internal.retryAction'),
            billing: error('internal.billing'),
            scoring: error('internal.scoring'),
            traceId: 'synthetic-trace',
          },
          retryAction: retry,
        };
      case 'forbidden':
        return {
          kind: 'forbidden',
          requiredPermission: forbiddenPermission,
          acquirePath: 'signIn',
          acquireHint: common('acquire.signIn'),
          acquireAction: retry,
        };
      case 'recovering':
        return { kind: 'recovering', resumeAt: recoveryPoint, resumeAction: retry };
      default:
        return { kind: 'ready' };
    }
  })();

  const labels: StateViewLabels = {
    emptyHeading: common('state.emptyHeading'),
    forbiddenHeading: common('state.forbiddenHeading'),
    recoveringHeading: common('state.recoveringHeading'),
    error: {
      heading: common('state.errorHeading'),
      impact: common('error.sectionImpact'),
      dataRetained: common('error.sectionDataRetained'),
      retryAction: common('error.sectionRetryAction'),
      billing: common('error.sectionBilling'),
      scoring: common('error.sectionScoring'),
      traceLabel: common('error.traceLabel'),
    },
  };

  return (
    <StateView state={state} labels={labels} regionLabel={regionLabel}>
      {children}
    </StateView>
  );
}
