'use client';

import { Button, Field } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { PageStateBoundary, type PreviewState } from '../../components/page-state-boundary.tsx';
import { RouteShell } from '../../components/route-shell.tsx';

export function AuthExperience({ mode }: { readonly mode: PreviewState }): ReactNode {
  const t = useTranslations('batch1');
  const [registering, setRegistering] = useState(false);
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [codeSent, setCodeSent] = useState(false);
  const [invalidCode, setInvalidCode] = useState(false);
  const [providerFailed, setProviderFailed] = useState(false);
  const [agreements, setAgreements] = useState({ terms: false, privacy: false, processing: false, age: false });

  const registrationReady = Object.values(agreements).every(Boolean);

  return (
    <RouteShell scrId="SCR-02" title={t('auth.title')} notice={t('auth.lead')}>
      <PageStateBoundary
        mode={mode}
        regionLabel="SCR-02-auth"
        emptyReason={t('auth.empty')}
        forbiddenPermission={t('auth.forbidden')}
        recoveryPoint={t('auth.recovering')}
      >
        <div className="mgd-auth-layout">
          <section className="mgd-auth-card" aria-labelledby="auth-form-title">
            <div className="mgd-segmented" role="group" aria-label={t('auth.title')}>
              <button type="button" aria-pressed={!registering} onClick={() => setRegistering(false)}>
                {t('auth.loginTab')}
              </button>
              <button type="button" aria-pressed={registering} onClick={() => setRegistering(true)}>
                {t('auth.registerTab')}
              </button>
            </div>
            <h2 id="auth-form-title">{registering ? t('auth.registerTab') : t('auth.loginTab')}</h2>

            <div className="mgd-provider-grid">
              {[t('auth.google'), t('auth.apple'), t('auth.wechat')].map((label, index) => (
                <Button
                  key={label}
                  controlId={`auth-provider-${index}`}
                  onClick={() => setProviderFailed(true)}
                >
                  {label}
                </Button>
              ))}
            </div>

            {providerFailed ? (
              <div className="mgd-inline-notice" role="alert">
                <strong>{t('auth.providerFailed')}</strong>
                <Button controlId="auth-use-email" onClick={() => setProviderFailed(false)}>
                  {t('auth.useEmail')}
                </Button>
              </div>
            ) : null}

            <p className="mgd-divider"><span>{t('auth.divider')}</span></p>
            <form
              className="mgd-form-stack"
              onSubmit={(event) => {
                event.preventDefault();
                if (codeSent) setInvalidCode(true);
              }}
            >
              <Field fieldId="auth-email" label={t('auth.email')} required>
                <input
                  type="email"
                  value={email}
                  placeholder={t('auth.emailPlaceholder')}
                  autoComplete="email"
                  onChange={(event) => setEmail(event.currentTarget.value)}
                />
              </Field>
              {!codeSent ? (
                <Button
                  controlId="auth-send-code"
                  type="button"
                  variant="primary"
                  disabled={email.length === 0}
                  disabledReason={email.length === 0 ? t('auth.email') : undefined}
                  onClick={() => setCodeSent(true)}
                >
                  {t('auth.sendCode')}
                </Button>
              ) : (
                <>
                  <Field
                    fieldId="auth-code"
                    label={t('auth.code')}
                    description={t('auth.codeHint')}
                    errorMessage={invalidCode ? t('auth.invalidCode') : undefined}
                    required
                  >
                    <input
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      value={code}
                      onChange={(event) => setCode(event.currentTarget.value)}
                    />
                  </Field>
                  <Button controlId="auth-submit" type="submit" variant="primary">
                    {t('auth.submit')}
                  </Button>
                </>
              )}

              {registering ? (
                <fieldset className="mgd-check-list">
                  <legend>{t('auth.registerTab')}</legend>
                  {(['terms', 'privacy', 'processing', 'age'] as const).map((key) => (
                    <label key={key}>
                      <input
                        type="checkbox"
                        checked={agreements[key]}
                        onChange={(event) => setAgreements((current) => ({ ...current, [key]: event.currentTarget.checked }))}
                      />
                      {t(`auth.${key}`)}
                    </label>
                  ))}
                  <p>{t('auth.separateConsent')}</p>
                  <p>{t('auth.ageHint')}</p>
                  <Button
                    controlId="auth-register"
                    variant="primary"
                    disabled={!registrationReady}
                    disabledReason={!registrationReady ? t('auth.registerTab') : undefined}
                  >
                    {t('auth.registerTab')}
                  </Button>
                </fieldset>
              ) : null}
            </form>
          </section>

          <aside className="mgd-context-panel" aria-label={t('auth.returnHint')}>
            <span aria-hidden="true">↳</span>
            <p>{t('auth.returnHint')}</p>
            <small>{t('auth.lead')}</small>
          </aside>
        </div>
      </PageStateBoundary>
    </RouteShell>
  );
}
