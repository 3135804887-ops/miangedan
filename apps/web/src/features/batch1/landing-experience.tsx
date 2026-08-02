'use client';

import { DisclosureNote } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { PageStateBoundary, type PreviewState } from '../../components/page-state-boundary.tsx';
import { RouteShell } from '../../components/route-shell.tsx';
import { Link } from '../../i18n/navigation.ts';

export function LandingExperience({ mode }: { readonly mode: PreviewState }): ReactNode {
  const t = useTranslations('batch1');

  return (
    <RouteShell scrId="SCR-01" title={t('landing.pageTitle')} notice={t('landing.eyebrow')}>
      <PageStateBoundary
        mode={mode}
        regionLabel="SCR-01-landing"
        emptyReason={t('landing.empty')}
        forbiddenPermission={t('landing.forbidden')}
        recoveryPoint={t('landing.recovering')}
      >
        <div className="mgd-landing-hero">
          <div className="mgd-landing-hero__copy">
            <p className="mgd-kicker">{t('landing.eyebrow')}</p>
            <h2>{t('landing.title')}</h2>
            <p className="mgd-lead">{t('landing.lead')}</p>
            <div className="mgd-action-row">
              <Link className="mgd-link-button mgd-link-button--primary" href="/demo">
                {t('landing.demoAction')}
              </Link>
              <Link
                className="mgd-link-button"
                href={{ pathname: '/auth', query: { returnTo: '/projects/new' } }}
              >
                {t('landing.uploadAction')}
              </Link>
            </div>
          </div>
          <aside className="mgd-signal-card" aria-labelledby="landing-privacy-title">
            <span className="mgd-signal-card__index" aria-hidden="true">01</span>
            <h3 id="landing-privacy-title">{t('landing.privacyTitle')}</h3>
            <p>{t('landing.privacyBody')}</p>
          </aside>
        </div>

        <section className="mgd-section" aria-labelledby="landing-workflow-title">
          <div className="mgd-section-heading">
            <p className="mgd-kicker">01—03</p>
            <h2 id="landing-workflow-title">{t('landing.workflowTitle')}</h2>
          </div>
          <ol className="mgd-step-grid">
            {[t('landing.stepMaterial'), t('landing.stepPlan'), t('landing.stepInterview')].map(
              (step, index) => (
                <li key={step}>
                  <span aria-hidden="true">0{index + 1}</span>
                  <strong>{step}</strong>
                </li>
              ),
            )}
          </ol>
        </section>

        <div className="mgd-two-column">
          <DisclosureNote title={t('landing.ageTitle')}>{t('landing.ageBody')}</DisclosureNote>
          <DisclosureNote title={t('landing.boundaryTitle')}>
            {t('landing.boundaryBody')}
          </DisclosureNote>
        </div>
      </PageStateBoundary>
    </RouteShell>
  );
}

export function DemoExperience({ mode }: { readonly mode: PreviewState }): ReactNode {
  const t = useTranslations('batch1');

  return (
    <RouteShell scrId="SCR-01" title={t('demo.title')} notice={t('shared.synthetic')}>
      <PageStateBoundary
        mode={mode}
        regionLabel="SCR-01-demo"
        emptyReason={t('demo.empty')}
        forbiddenPermission={t('demo.forbidden')}
        recoveryPoint={t('demo.recovering')}
      >
        <p className="mgd-lead">{t('demo.lead')}</p>
        <div className="mgd-demo-board">
          <section className="mgd-demo-board__profile" aria-labelledby="demo-profile-title">
            <span className="mgd-synthetic-label">{t('shared.synthetic')}</span>
            <h2 id="demo-profile-title">{t('demo.candidate')}</h2>
            <p>{t('demo.role')}</p>
            <div className="mgd-avatar-stub" aria-label={t('shared.synthetic')}>
              <span aria-hidden="true">MGD / DEMO</span>
              <small>{t('shared.mockNotice')}</small>
            </div>
          </section>
          <section className="mgd-demo-board__signals" aria-labelledby="demo-result-title">
            <p className="mgd-kicker">SYNTHETIC REVIEW</p>
            <h2 id="demo-result-title">{t('demo.result')}</h2>
            <ol className="mgd-signal-list">
              <li>{t('demo.signalOne')}</li>
              <li>{t('demo.signalTwo')}</li>
              <li>{t('demo.signalThree')}</li>
            </ol>
            <p className="mgd-muted">{t('demo.dataNotice')}</p>
            <Link className="mgd-link-button" href="/demo">
              {t('demo.restart')}
            </Link>
          </section>
        </div>
      </PageStateBoundary>
    </RouteShell>
  );
}
