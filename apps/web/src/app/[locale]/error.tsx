'use client';

/**
 * 路由级错误边界（需求 B0-2 第 3、5 条）。
 *
 * 渲染错误五要素与重试控件；展示内容限于错误码、请求标识与可读说明，
 * 不含堆栈、令牌或用户内容（需求 G8 第 1、2 条）。
 */

import { Button, ErrorPanel } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useEffect, type ReactNode } from 'react';

import { presentError } from '../../lib/error-presenter.ts';
import { normalizeRoute, reportEvent } from '../../lib/telemetry.ts';

interface ErrorViewProps {
  readonly error: Error & { readonly digest?: string };
  readonly reset: () => void;
}

export default function ErrorView({ error, reset }: ErrorViewProps): ReactNode {
  const t = useTranslations();

  // 只上报路由与摘要标识，不上报 message 与 stack
  useEffect(() => {
    reportEvent({
      route: normalizeRoute(globalThis.location?.pathname ?? ''),
      scrId: 'GLOBAL',
      errorCode: 'internal',
      traceId: error.digest ?? 'unknown',
    });
  }, [error.digest]);

  const view = presentError(
    {
      error: {
        code: 'internal',
        trace_id: error.digest ?? 'unknown',
        data_region: 'cn',
      },
    },
    t,
  );

  return (
    <div data-mgd-view="error">
      <ErrorPanel
        error={view}
        labels={{
          heading: t('common.error.globalHeading'),
          impact: t('common.error.sectionImpact'),
          dataRetained: t('common.error.sectionDataRetained'),
          retryAction: t('common.error.sectionRetryAction'),
          billing: t('common.error.sectionBilling'),
          scoring: t('common.error.sectionScoring'),
          traceLabel: t('common.error.traceLabel'),
        }}
        action={
          <Button controlId="retry-route" variant="primary" onClick={reset}>
            {t('common.state.retry')}
          </Button>
        }
      />
    </div>
  );
}
