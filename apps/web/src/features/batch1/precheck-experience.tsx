'use client';

import { ACCOMMODATION_KEYS, defaultAccommodations, type AccommodationKey } from '@mgd/domain-states';
import { Button, Switch } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState, type ReactNode } from 'react';

import { PageStateBoundary, type PreviewState } from '../../components/page-state-boundary.tsx';
import { RouteShell } from '../../components/route-shell.tsx';
import { Link } from '../../i18n/navigation.ts';

const CHECK_KEYS = ['camera', 'microphone', 'network', 'audio', 'avatar'] as const;
type CheckKey = (typeof CHECK_KEYS)[number];
type CheckStatus = 'pending' | 'running' | 'passed' | 'failed';

const INITIAL_CHECKS: Readonly<Record<CheckKey, CheckStatus>> = {
  camera: 'passed',
  microphone: 'failed',
  network: 'passed',
  audio: 'passed',
  avatar: 'running',
};

export function PrecheckExperience({
  mode,
  insufficientQuota,
  otherDeviceActive,
}: {
  readonly mode: PreviewState;
  readonly insufficientQuota: boolean;
  readonly otherDeviceActive: boolean;
}): ReactNode {
  const t = useTranslations('batch1');
  const [checks, setChecks] = useState(INITIAL_CHECKS);
  const [requestCounts, setRequestCounts] = useState<Record<CheckKey, number>>({ camera: 0, microphone: 0, network: 0, audio: 0, avatar: 1 });
  const [accommodations, setAccommodations] = useState(() => ({ ...defaultAccommodations(), text_only: true }));
  const [settingsFrozen, setSettingsFrozen] = useState(false);
  const inFlight = useRef<Set<CheckKey>>(new Set(['avatar']));

  useEffect(() => {
    const handle = window.setTimeout(() => {
      setChecks((current) => ({ ...current, avatar: 'passed' }));
      inFlight.current.delete('avatar');
    }, 800);
    return () => window.clearTimeout(handle);
  }, []);

  function retryCheck(key: CheckKey): void {
    if (inFlight.current.has(key)) return;
    inFlight.current.add(key);
    setRequestCounts((current) => ({ ...current, [key]: current[key] + 1 }));
    setChecks((current) => ({ ...current, [key]: 'running' }));
    queueMicrotask(() => {
      setChecks((current) => ({ ...current, [key]: 'passed' }));
      inFlight.current.delete(key);
    });
  }

  function changeAccommodation(key: AccommodationKey, checked: boolean): void {
    setAccommodations((current) => ({ ...current, [key]: checked }));
  }

  if (otherDeviceActive) {
    return (
      <RouteShell scrId="SCR-07" title={t('precheck.title')} notice={t('precheck.transfer')}>
        <section className="mgd-device-transfer" aria-labelledby="precheck-transfer-title">
          <h2 id="precheck-transfer-title">{t('precheck.transfer')}</h2>
          <p>{t('shared.notScored')}</p>
          <Button controlId="precheck-transfer-device" variant="primary">{t('precheck.transferAction')}</Button>
        </section>
      </RouteShell>
    );
  }

  const allPassed = CHECK_KEYS.every((key) => checks[key] === 'passed');
  const canStart = allPassed && !insufficientQuota && settingsFrozen;

  return (
    <RouteShell scrId="SCR-07" title={t('precheck.title')} notice={t('precheck.lead')}>
      <PageStateBoundary
        mode={mode}
        regionLabel="SCR-07-precheck"
        emptyReason={t('precheck.empty')}
        loadingExpectation={t('precheck.avatarExpectation')}
        forbiddenPermission={t('precheck.forbidden')}
        recoveryPoint={t('precheck.recovering')}
      >
        <div className="mgd-precheck-layout">
          <section aria-labelledby="precheck-checks-title">
            <div className="mgd-section-heading">
              <div><p className="mgd-kicker">DEVICE / NETWORK / PROVIDER</p><h2 id="precheck-checks-title">{t('precheck.checks')}</h2></div>
              <p>{t('precheck.mediaHint')}</p>
            </div>
            <div className="mgd-check-grid">
              {CHECK_KEYS.map((key, index) => {
                const status = checks[key];
                const failure = key === 'network' ? t('precheck.networkFailure') : t('precheck.deviceFailure');
                return (
                  <article key={key} className="mgd-check-card" data-check-status={status}>
                    <div className="mgd-check-card__heading">
                      <span aria-hidden="true">0{index + 1}</span>
                      <h3>{t(`precheck.${key}`)}</h3>
                      <strong className={`mgd-check-status mgd-check-status--${status}`}>● {t(`precheck.${status}`)}</strong>
                    </div>
                    {status === 'failed' ? <p className="mgd-warning-copy">⚠ {failure}</p> : null}
                    {key === 'avatar' ? <p>{t('precheck.avatarExpectation')}</p> : null}
                    <p className="mgd-muted" data-request-count={key}>{t('precheck.requestCount', { count: requestCounts[key] })}</p>
                    <div className="mgd-action-row">
                      <Button controlId={`precheck-retry-${key}`} loading={status === 'running'} busyLabel={t('precheck.running')} onClick={() => retryCheck(key)}>
                        {t('precheck.retryOne')}
                      </Button>
                      {(key === 'camera' || key === 'microphone') && status === 'failed' ? (
                        <Button controlId={`precheck-disable-${key}`} onClick={() => setChecks((current) => ({ ...current, [key]: 'passed' }))}>
                          {t('precheck.closeContinue')}
                        </Button>
                      ) : null}
                    </div>
                    {key === 'avatar' && status === 'failed' ? <a href="mailto:support@example.invalid">{t('precheck.support')}</a> : null}
                  </article>
                );
              })}
            </div>
          </section>

          <aside className="mgd-precheck-settings" aria-labelledby="precheck-settings-title">
            <p className="mgd-kicker">{t('precheck.inherited')}</p>
            <h2 id="precheck-settings-title">{t('precheck.settings')}</h2>
            <p>{t('precheck.settingsHint')}</p>
            {ACCOMMODATION_KEYS.map((key) => (
              <Switch
                key={key}
                controlId={`precheck-accommodation-${key}`}
                label={t(`plan.accommodationLabels.${key}`)}
                checked={accommodations[key]}
                onCheckedChange={(checked) => changeAccommodation(key, checked)}
              />
            ))}
            <Button controlId="precheck-freeze-settings" variant="primary" onClick={() => setSettingsFrozen(true)}>
              {t('precheck.freeze')}
            </Button>
            {settingsFrozen ? <p className="mgd-inline-notice" role="status">{t('precheck.frozen')}</p> : null}
          </aside>
        </div>

        <section className="mgd-quota-strip" aria-labelledby="precheck-quota-title">
          <div><p className="mgd-kicker">ENTITLEMENT</p><h2 id="precheck-quota-title">{t('precheck.quota')}</h2></div>
          {insufficientQuota ? (
            <div><p className="mgd-warning-copy">⚠ {t('precheck.quotaInsufficient')}</p><Link className="mgd-link-button" href="/billing">{t('precheck.purchase')}</Link></div>
          ) : null}
          <Button
            controlId="precheck-start-round"
            variant="primary"
            disabled={!canStart}
            disabledReason={!canStart ? t('precheck.startDisabled') : undefined}
          >
            {t('precheck.start')}
          </Button>
        </section>
      </PageStateBoundary>
    </RouteShell>
  );
}
