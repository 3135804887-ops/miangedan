'use client';

import { Button, Field, IconLock, IconMessage, useToast } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { apiFetch } from '../lib/api-fetch.ts';

function GoogleIcon(): React.ReactNode {
  return (
    <svg aria-hidden="true" width="18" height="18" viewBox="0 0 24 24">
      <path fill="currentColor" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.27-4.74 3.27-8.1z" />
      <path fill="currentColor" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84A11 11 0 0 0 12 23z" />
      <path fill="currentColor" d="M5.84 14.1a6.6 6.6 0 0 1 0-4.2V7.06H2.18a11 11 0 0 0 0 9.88z" />
      <path fill="currentColor" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15A11 11 0 0 0 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z" />
    </svg>
  );
}

function AppleIcon(): React.ReactNode {
  return (
    <svg aria-hidden="true" width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
      <path d="M17.05 20.28c-.98.95-2.05.86-3.08.38-1.09-.5-2.08-.53-3.2 0-1.44.62-2.2.44-3.06-.38C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.53 4.08zM12.03 7.25c-.15-2.23 1.66-4.07 3.74-4.25.29 2.58-2.34 4.5-3.74 4.25z" />
    </svg>
  );
}

function WechatIcon(): React.ReactNode {
  return (
    <svg aria-hidden="true" width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
      <path d="M9.5 4C5.36 4 2 6.8 2 10.25c0 1.95 1.05 3.7 2.68 4.86l-.66 2.03 2.3-1.2c.65.18 1.33.31 2.03.31.17 0 .34-.01.5-.03a5.2 5.2 0 0 1-.3-1.72c0-3.3 3.25-5.97 7.25-5.97.32 0 .63.02.94.06C16.32 6.2 13.2 4 9.5 4zm-3.1 4.6a.85.85 0 1 1 0-1.7.85.85 0 0 1 0 1.7zm5.2 0a.85.85 0 1 1 0-1.7.85.85 0 0 1 0 1.7zM15.5 11c-3.03 0-5.5 2.13-5.5 4.75S12.47 20.5 15.5 20.5c.57 0 1.12-.08 1.64-.22l1.86.97-.54-1.66A4.62 4.62 0 0 0 21 15.75C21 13.13 18.53 11 15.5 11zm-1.6 2.1a.7.7 0 1 1 0-1.4.7.7 0 0 1 0 1.4zm3.2 0a.7.7 0 1 1 0-1.4.7.7 0 0 1 0 1.4z" />
    </svg>
  );
}

export function AuthForm({ locale }: { locale: string }): React.ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [codeSent, setCodeSent] = useState(false);
  const [sending, setSending] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (countdown <= 0) return;
    const id = window.setTimeout(() => setCountdown((c) => c - 1), 1000);
    return () => window.clearTimeout(id);
  }, [countdown]);

  const sendCode = async () => {
    setError(null);
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setError(locale === 'zh-CN' ? '请输入有效邮箱' : 'Enter a valid email');
      return;
    }
    setSending(true);
    try {
      await apiFetch('/v1/identity/email/challenges', {
        method: 'post',
        idempotencyKey: 'auth-challenge-demo',
        body: JSON.stringify({ email }),
      });
      setCodeSent(true);
      setCountdown(60);
      toast.push({ title: t('auth.codeSent'), tone: 'success' });
    } catch {
      setError(t('auth.signInError'));
    } finally {
      setSending(false);
    }
  };

  const verify = async () => {
    setError(null);
    setVerifying(true);
    try {
      await apiFetch('/v1/identity/email/challenges/{challengeId}/verify', {
        method: 'post',
        idempotencyKey: 'auth-verify-demo',
        pathParams: { challengeId: 'ch-1' },
        body: JSON.stringify({ code }),
      });
      toast.push({ title: t('auth.signInSuccess'), tone: 'success' });
      window.location.href = `/${locale}/dashboard`;
    } catch {
      setError(t('auth.signInError'));
    } finally {
      setVerifying(false);
    }
  };

  return (
    <div className="p-8">
      <Field label={t('auth.emailLabel')} fieldId="auth-email" required>
        <div className="relative">
          <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400">
            <IconMessage size={18} />
          </span>
          <input
            id="auth-email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={t('auth.emailPlaceholder')}
            className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface pl-10 pr-3 text-neutral-900 placeholder:text-neutral-400 focus:border-primary"
          />
        </div>
      </Field>

      {codeSent ? (
        <Field label={t('auth.codeLabel')} fieldId="auth-code" required>
          <div className="flex gap-2">
            <input
              id="auth-code"
              inputMode="numeric"
              maxLength={6}
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
              placeholder={t('auth.codePlaceholder')}
              className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-3 font-mono tracking-[0.3em] text-neutral-900 placeholder:tracking-normal placeholder:text-neutral-400"
            />
            <Button variant="secondary" disabled={countdown > 0} disabledReason={countdown > 0 ? t('auth.resendIn', { seconds: String(countdown) }) : undefined} onClick={sendCode} loading={sending} busyLabel={t('state.loadingLabel')}>
              {countdown > 0 ? `${countdown}s` : t('auth.resend')}
            </Button>
          </div>
        </Field>
      ) : null}

      {error !== null ? (
        <p role="alert" className="mb-4 rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">
          {error}
        </p>
      ) : null}

      <Button variant="primary" className="mt-2 w-full" onClick={codeSent ? verify : sendCode} loading={sending || verifying} busyLabel={t('state.loadingLabel')}>
        {codeSent ? t('auth.verifyAndSignIn') : t('auth.sendCode')}
      </Button>

      <div className="my-6 flex items-center gap-3 text-xs text-neutral-400">
        <span className="h-px flex-1 bg-neutral-100" />
        {t('auth.or')}
        <span className="h-px flex-1 bg-neutral-100" />
      </div>

      <div className="grid gap-2">
        <Button variant="secondary" className="w-full" onClick={() => toast.push({ title: t('auth.thirdPartyFallback'), tone: 'info' })}>
          <GoogleIcon />
          {t('auth.google')}
        </Button>
        <Button variant="secondary" className="w-full" onClick={() => toast.push({ title: t('auth.thirdPartyFallback'), tone: 'info' })}>
          <AppleIcon />
          {t('auth.apple')}
        </Button>
        <Button variant="secondary" className="w-full" onClick={() => toast.push({ title: t('auth.thirdPartyFallback'), tone: 'info' })}>
          <WechatIcon />
          {t('auth.wechat')}
        </Button>
      </div>

      <p className="mt-6 mb-0 flex items-start gap-2 text-xs leading-5 text-neutral-500">
        <IconLock size={14} className="mt-0.5 shrink-0" />
        {t('auth.agreement')}
      </p>
    </div>
  );
}
