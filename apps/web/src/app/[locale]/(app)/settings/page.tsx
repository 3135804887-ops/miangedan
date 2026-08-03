/** SCR-14 账户与隐私：身份、语言偏好、六类授权中心、导出与删除。 */

'use client';

import {
  Button,
  Card,
  CardBody,
  IconGlobe,
  IconUser,
  PageHeader,
  Tabs,
  Tint,
  useToast,
} from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import type { ReactNode } from 'react';

import { apiFetch } from '../../../../lib/api-fetch.ts';
import { useApiGet } from '../../../../lib/api-hooks.ts';

interface PreferencesPayload {
  readonly preferences: {
    readonly ui_language: string;
    readonly interview_language: string;
    readonly identity?: { email?: string; providers?: readonly string[] };
  };
}

interface ConsentsPayload {
  readonly items: readonly {
    consent_type: string;
    granted: boolean;
    required?: boolean;
    updated_at?: string;
  }[];
}

const CONSENT_TYPES = ['core_service', 'raw_av_recording', 'org_sharing', 'product_analytics', 'model_training', 'marketing'] as const;

export default function SettingsPage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US' }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const prefsState = useApiGet<PreferencesPayload>('/v1/me/preferences', {});
  const consentsState = useApiGet<ConsentsPayload>('/v1/consent/grants', {});

  const preferences = prefsState.data?.preferences;
  const consents = CONSENT_TYPES.map((type) => {
    const found = consentsState.data?.items.find((c) => c.consent_type === type);
    return { type, granted: found?.granted ?? false, required: found?.required ?? type === 'core_service' };
  });

  const saveLanguage = async (interviewLanguage: string) => {
    const res = await apiFetch('/v1/me/preferences', {
      method: 'put',
      idempotencyKey: `prefs-${Date.now()}`,
      body: { ui_language: locale, interview_language: interviewLanguage },
    });
    toast.push({
      title: res.ok ? (zh ? '偏好已保存' : 'Preferences saved') : (zh ? '保存失败，请重试' : 'Failed to save, please retry'),
      tone: res.ok ? 'success' : 'danger',
    });
  };

  const withdraw = async (type: string) => {
    const res = await apiFetch('/v1/consent/grants/{consentType}/withdrawals', {
      method: 'post',
      idempotencyKey: `withdraw-${type}-${Date.now()}`,
      pathParams: { consentType: type },
      body: { reason: 'user_requested' },
    });
    toast.push({
      title: res.ok ? (zh ? '授权已撤回并即时生效' : 'Consent withdrawn (effective immediately)') : (zh ? '撤回暂未接入（占位）' : 'Withdrawal placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const startExport = async () => {
    const res = await apiFetch('/v1/me/export', { method: 'get' });
    toast.push({
      title: res.ok ? (zh ? '导出任务已创建（含训练用途标记）' : 'Export task created (with training disclaimer)') : (zh ? '导出暂未接入（占位）' : 'Export placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  const deleteAccount = async () => {
    const res = await apiFetch('/v1/me/deletion', {
      method: 'post',
      idempotencyKey: `delete-account-${Date.now()}`,
      body: { reauth_proof: 'synthetic-proof', confirmation_text: 'delete' },
    });
    toast.push({
      title: res.ok ? (zh ? '删除任务已创建（真实进度可查）' : 'Deletion task created') : (zh ? '删除暂未接入（占位）' : 'Delete placeholder'),
      tone: res.ok ? 'success' : 'info',
    });
  };

  return (
    <>
      <PageHeader kicker={t('settings.kicker')} title={t('settings.title')} />
      <Card>
        <CardBody>
          <Tabs
            initialId="identity"
            items={[
              {
                id: 'identity',
                label: t('settings.tabIdentity'),
                content: (
                  <Card>
                    <CardBody className="space-y-4">
                      <div className="flex items-center gap-4 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                        <span className="grid size-12 place-items-center rounded-2xl bg-[var(--mgd-app-brand-soft)] text-[var(--mgd-app-brand-ink)]">
                          <IconUser size={22} />
                        </span>
                        <div className="flex-1">
                          <p className="mb-0.5 font-medium text-neutral-900">{t('settings.email')}</p>
                          <p className="mb-0 text-sm text-neutral-600">{preferences?.identity?.email ?? 'candidate@example.com'}</p>
                        </div>
                        <Tint tone="success">{zh ? '已验证' : 'Verified'}</Tint>
                      </div>
                      <div>
                        <h2 className="mb-2 text-sm font-semibold text-neutral-900">{t('settings.providers')}</h2>
                        <div className="flex flex-wrap gap-2">
                          <Tint tone="brand">email</Tint>
                          <Button variant="secondary" targetSize="min">{t('settings.bindProvider')}</Button>
                        </div>
                        <p className="mb-0 mt-3 text-xs text-neutral-500">{t('settings.unbindWarning')}</p>
                      </div>
                    </CardBody>
                  </Card>
                ),
              },
              {
                id: 'language',
                label: t('settings.tabLanguage'),
                content: (
                  <Card>
                    <CardBody className="grid gap-5 sm:grid-cols-2">
                      <div>
                        <label htmlFor="ui-lang" className="mb-1.5 block text-sm font-medium text-neutral-800">
                          <IconGlobe size={14} className="mr-1 inline" />
                          {t('settings.uiLanguage')}
                        </label>
                        <select id="ui-lang" defaultValue={locale} className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-3 text-sm">
                          <option value="zh-CN">中文</option>
                          <option value="en-US">English</option>
                        </select>
                      </div>
                      <div>
                        <label htmlFor="interview-lang" className="mb-1.5 block text-sm font-medium text-neutral-800">
                          <IconGlobe size={14} className="mr-1 inline" />
                          {t('settings.interviewLanguage')}
                        </label>
                        <select
                          id="interview-lang"
                          defaultValue={preferences?.interview_language ?? 'zh-CN'}
                          onChange={(e) => saveLanguage(e.target.value)}
                          className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-3 text-sm"
                        >
                          <option value="zh-CN">中文</option>
                          <option value="en-US">English</option>
                        </select>
                        <p className="mb-0 mt-2 text-xs text-neutral-500">{t('settings.languagesNote')}</p>
                      </div>
                    </CardBody>
                  </Card>
                ),
              },
              {
                id: 'consents',
                label: t('settings.tabConsents'),
                content: (
                  <Card>
                    <CardBody>
                      <p className="mb-4 text-sm text-neutral-600">{t('settings.consentDesc')}</p>
                      <div className="space-y-3">
                        {consents.map((c) => (
                          <div key={c.type} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                            <div className="min-w-0">
                              <p className="mb-0.5 font-medium text-neutral-900">{t(`settings.consentTypes.${c.type}`)}</p>
                              <p className="mb-0 text-xs text-neutral-500">{c.type}{c.required ? ` · ${zh ? '必要' : 'required'}` : ''}</p>
                            </div>
                            <div className="flex items-center gap-2">
                              <Tint tone={c.granted ? 'success' : 'neutral'}>{c.granted ? (zh ? '已授权' : 'Granted') : (zh ? '未授权' : 'Not granted')}</Tint>
                              {c.granted && !c.required ? (
                                <Button variant="secondary" targetSize="min" onClick={() => withdraw(c.type)}>{t('settings.withdraw')}</Button>
                              ) : null}
                              {!c.granted ? (
                                <Button variant="secondary" targetSize="min">{t('settings.grant')}</Button>
                              ) : null}
                            </div>
                          </div>
                        ))}
                      </div>
                    </CardBody>
                  </Card>
                ),
              },
              {
                id: 'data',
                label: t('settings.tabData'),
                content: (
                  <Card>
                    <CardBody className="space-y-4">
                      <div className="rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                        <p className="mb-1 font-medium text-neutral-900">{t('settings.export')}</p>
                        <p className="mb-3 text-sm text-neutral-600">{t('settings.exportHint')}</p>
                        <Button variant="secondary" onClick={startExport}>{t('settings.export')}</Button>
                      </div>
                      <div className="rounded-xl border border-danger/20 bg-danger/5 p-4">
                        <p className="mb-1 font-medium text-danger-text">{t('settings.deleteAccount')}</p>
                        <p className="mb-3 text-sm text-neutral-600">{t('settings.deleteAccountHint')}</p>
                        <Button variant="danger" onClick={deleteAccount}>{t('settings.deleteAccount')}</Button>
                      </div>
                    </CardBody>
                  </Card>
                ),
              },
            ]}
          />
        </CardBody>
      </Card>
    </>
  );
}
