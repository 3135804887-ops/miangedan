/** SCR-01 落地页：品牌 hero、特性、样例演示入口（游客可访问）。 */

import { IconArrowRight, IconCamera, IconChart, IconPlay, IconShield, IconSparkle } from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function LandingPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';

  const heroTitle = t('landing.heroTitle');
  const highlight = t('landing.heroHighlight');
  const titleParts = heroTitle.split(highlight);

  const stats = [
    { value: t('landing.stat1Value'), label: t('landing.stat1Label') },
    { value: t('landing.stat2Value'), label: t('landing.stat2Label') },
    { value: t('landing.stat3Value'), label: t('landing.stat3Label') },
    { value: t('landing.stat4Value'), label: t('landing.stat4Label') },
  ] as const;

  const features = [
    { icon: <IconSparkle size={22} />, title: t('landing.feature1Title'), desc: t('landing.feature1Desc') },
    { icon: <IconCamera size={22} />, title: t('landing.feature2Title'), desc: t('landing.feature2Desc') },
    { icon: <IconChart size={22} />, title: t('landing.feature3Title'), desc: t('landing.feature3Desc') },
    { icon: <IconShield size={22} />, title: t('landing.feature4Title'), desc: t('landing.feature4Desc') },
  ] as const;

  return (
    <>
      {/* Hero */}
      <section className="relative overflow-hidden">
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(900px_500px_at_70%_-20%,color-mix(in_srgb,var(--mgd-color-primary)_14%,transparent),transparent_60%),radial-gradient(700px_400px_at_15%_10%,color-mix(in_srgb,var(--mgd-app-brand-to)_10%,transparent),transparent_55%)]"
        />
        <div className="relative mx-auto w-[min(100%-2rem,1200px)] pb-20 pt-16 text-center sm:pt-24">
          <div className="mx-auto mb-6 inline-flex items-center gap-2 rounded-full border border-[var(--mgd-app-border-default)] bg-surface px-4 py-1.5 text-sm text-neutral-600 shadow-[var(--mgd-app-shadow-sm)]">
            <span className="inline-block size-2 rounded-full bg-success" aria-hidden="true" />
            {t('landing.heroKicker')}
          </div>
          <h1 className="mx-auto max-w-3xl text-[clamp(2rem,5vw,3.4rem)] font-extrabold leading-[1.15] tracking-[-0.03em] text-neutral-900">
            {titleParts[0]}
            <span className="mgd-brand-gradient">{highlight}</span>
            {titleParts[1]}
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg leading-8 text-neutral-600">{t('landing.heroDesc')}</p>
          <div className="mt-9 flex flex-wrap items-center justify-center gap-4">
            <a
              href={`/${locale}/auth`}
              className="mgd-target-primary inline-flex items-center gap-2 rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-6 text-base font-semibold text-white shadow-[var(--mgd-app-shadow-brand)] transition hover:brightness-105"
            >
              {t('landing.heroCtaStart')}
              <IconArrowRight size={18} />
            </a>
            <a
              href={`/${locale}/demo`}
              className="mgd-target-primary inline-flex items-center gap-2 rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-6 text-base font-semibold text-neutral-800 shadow-[var(--mgd-app-shadow-sm)] transition hover:border-[var(--mgd-app-border-strong)] hover:shadow-[var(--mgd-app-shadow-md)]"
            >
              <IconPlay size={18} className="text-primary" />
              {t('landing.heroCtaDemo')}
            </a>
          </div>
          <p className="mt-4 text-xs text-neutral-500">{t('landing.heroNote')}</p>

          <dl className="mx-auto mt-14 grid max-w-3xl grid-cols-2 gap-4 sm:grid-cols-4">
            {stats.map((s) => (
              <div key={s.label} className="rounded-2xl border border-neutral-100 bg-surface/80 px-4 py-5 shadow-[var(--mgd-app-shadow-sm)] backdrop-blur">
                <dt className="order-2 mt-1 text-sm text-neutral-600">{s.label}</dt>
                <dd className="mgd-stat-value text-[var(--mgd-app-brand-ink)]">{s.value}</dd>
              </div>
            ))}
          </dl>
        </div>
      </section>

      {/* 特性 */}
      <section className="mx-auto w-[min(100%-2rem,1200px)] py-14">
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {features.map((f, i) => (
            <article key={f.title} className="mgd-card mgd-card--hoverable p-6" style={{ animationDelay: `${i * 60}ms` }}>
              <div className="mb-4 grid size-11 place-items-center rounded-xl bg-[var(--mgd-app-brand-soft)] text-[var(--mgd-app-brand-ink)]">
                {f.icon}
              </div>
              <h2 className="mb-2 text-lg font-semibold text-neutral-900">{f.title}</h2>
              <p className="mb-0 text-sm leading-6 text-neutral-600">{f.desc}</p>
            </article>
          ))}
        </div>
      </section>

      {/* 样例演示 CTA */}
      <section className="mx-auto w-[min(100%-2rem,1200px)] pb-16">
        <div className="mgd-card mgd-card--brand relative overflow-hidden p-8 sm:p-12">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -right-20 -top-24 size-72 rounded-full bg-[radial-gradient(circle,color-mix(in_srgb,var(--mgd-color-primary)_18%,transparent),transparent_70%)]"
          />
          <div className="relative max-w-xl">
            <h2 className="mb-3 text-2xl font-bold text-neutral-900">{t('landing.demoSection')}</h2>
            <p className="mb-6 text-neutral-600">{t('landing.demoDesc')}</p>
            <a
              href={`/${locale}/demo`}
              className="mgd-target-primary inline-flex items-center gap-2 rounded-xl bg-neutral-900 px-5 font-semibold text-white transition hover:bg-neutral-800"
            >
              {zh ? '立即体验' : 'Try it now'}
              <IconArrowRight size={17} />
            </a>
          </div>
        </div>
      </section>

      <footer className="border-t border-neutral-100 bg-surface/70 py-8">
        <div className="mx-auto flex w-[min(100%-2rem,1200px)] flex-wrap items-center justify-between gap-3 text-sm text-neutral-500">
          <span>
            {t('brand.name')} · {t('landing.footerDesc')}
          </span>
          <span>
            © 2026 {t('brand.name')} · {t('common.allRightsReserved')}
          </span>
        </div>
      </footer>
    </>
  );
}
