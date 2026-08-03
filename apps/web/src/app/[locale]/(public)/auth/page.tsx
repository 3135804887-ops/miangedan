/** SCR-02 登录/注册：邮箱验证码 + 第三方（Google/Apple/微信）+ 协议与年龄声明。 */

import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { AuthForm } from '../../../../components/auth-form.tsx';

export default async function AuthPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });

  return (
    <div className="mx-auto flex w-[min(100%-2rem,460px)] flex-col py-14">
      <div className="mgd-card overflow-hidden">
        <div className="bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-8 py-8 text-white">
          <h1 className="mb-1 text-2xl font-bold tracking-tight">{t('auth.welcome')}</h1>
          <p className="mb-0 text-sm text-white/85">{t('auth.subtitle')}</p>
        </div>
        <AuthForm locale={locale} />
      </div>
      <p className="mt-5 text-xs leading-5 text-neutral-500">{t('auth.agreement')}</p>
      <p className="mt-2 text-xs text-neutral-500">{t('auth.ageNote')}</p>
    </div>
  );
}
