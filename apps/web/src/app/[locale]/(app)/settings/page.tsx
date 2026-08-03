/** SCR-14 账户与隐私：身份、语言、六类授权中心、数据导出与删除。 */

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  IconCheck,
  IconDownload,
  IconGlobe,
  IconShield,
  IconTrash,
  IconUser,
  PageHeader,
  Tabs,
  Tint,
} from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function SettingsPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';

  const consents = [
    { type: 'core_service', granted: true, required: true },
    { type: 'raw_av_recording', granted: false, required: false },
    { type: 'org_sharing', granted: false, required: false },
    { type: 'non_essential_analytics', granted: true, required: false },
    { type: 'model_training', granted: false, required: false },
    { type: 'marketing', granted: false, required: false },
  ] as const;

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
                          <p className="mb-0 text-sm text-neutral-600">candidate@example.com</p>
                        </div>
                        <Tint tone="success">{zh ? '已验证' : 'Verified'}</Tint>
                      </div>
                      <div>
                        <h3 className="mb-2 text-sm font-semibold text-neutral-900">{t('settings.providers')}</h3>
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
                          {t('settings.interviewLanguage')}
                        </label>
                        <select id="interview-lang" defaultValue="zh-CN" className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-3 text-sm">
                          <option value="zh-CN">中文</option>
                          <option value="en-US">English</option>
                        </select>
                      </div>
                      <p className="mb-0 text-sm text-neutral-500 sm:col-span-2">{t('settings.languagesNote')}</p>
                    </CardBody>
                  </Card>
                ),
              },
              {
                id: 'consents',
                label: t('settings.tabConsents'),
                content: (
                  <Card>
                    <CardHeader title={<span className="flex items-center gap-2"><IconShield size={17} className="text-primary" />{t('settings.consentTitle')}</span>} description={t('settings.consentDesc')} />
                    <CardBody className="space-y-3">
                      {consents.map((c) => (
                        <div key={c.type} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] px-4 py-3.5">
                          <div>
                            <p className="mb-0.5 font-medium text-neutral-900">
                              {t(`settings.consentTypes.${c.type}`)}
                              {c.required ? <span className="ml-2 text-xs text-neutral-500">({zh ? '必要' : 'required'})</span> : null}
                            </p>
                            <p className="mb-0 text-xs text-neutral-500">{c.type}</p>
                          </div>
                          {c.granted ? (
                            <div className="flex items-center gap-2">
                              <Tint tone="success"><IconCheck size={12} />{zh ? '已授予' : 'Granted'}</Tint>
                              {!c.required ? <Button variant="secondary" targetSize="min">{t('settings.withdraw')}</Button> : null}
                            </div>
                          ) : (
                            <Button variant="secondary" targetSize="min">{t('settings.grant')}</Button>
                          )}
                        </div>
                      ))}
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
                      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                        <div>
                          <p className="mb-0.5 font-medium text-neutral-900">{t('settings.export')}</p>
                          <p className="mb-0 text-sm text-neutral-600">{t('settings.exportHint')}</p>
                        </div>
                        <Button variant="primary"><IconDownload size={16} />{t('settings.export')}</Button>
                      </div>
                      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-danger/30 bg-danger/5 p-4">
                        <div>
                          <p className="mb-0.5 font-medium text-danger">{t('settings.deleteAccount')}</p>
                          <p className="mb-0 text-sm text-neutral-600">{t('settings.deleteAccountHint')}</p>
                        </div>
                        <Button variant="danger"><IconTrash size={16} />{t('settings.deleteAccount')}</Button>
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
