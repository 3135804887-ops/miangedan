/** SCR-01 样例演示：合成材料完整流程导览（游客可访问，不收集个人信息）。 */

import {
  Card,
  CardBody,
  CardHeader,
  IconArrowRight,
  IconCamera,
  IconChart,
  IconFile,
  IconPlay,
  IconSparkle,
  PageHeader,
  Tint,
} from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function DemoPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';

  const steps = [
    { icon: <IconFile size={20} />, title: zh ? '① 创建项目' : '① Create project', desc: zh ? '上传合成简历、粘贴合成 JD，隔离扫描后解析为结构化画像。' : 'Upload a synthetic resume, paste a synthetic JD, scan in isolation and parse into structured profiles.' },
    { icon: <IconSparkle size={20} />, title: zh ? '② 生成计划' : '② Generate plan', desc: zh ? 'AI 建议 3 轮计划：简历深挖 → 岗位专业 → 综合终面；确认后冻结。' : 'AI suggests 3 rounds: resume deep-dive → role professional → comprehensive final; frozen on confirmation.' },
    { icon: <IconCamera size={20} />, title: zh ? '③ 实时面试' : '③ Live interview', desc: zh ? '数字人实时问答，支持打断、字幕修订、岗位工具与文字降级。' : 'Live avatar Q&A with interruption, subtitle revision, job tools and text fallback.' },
    { icon: <IconChart size={20} />, title: zh ? '④ 报告复盘' : '④ Report and review', desc: zh ? '六维评分、逐题证据与岗位匹配；练习不改分，正式重试用新题。' : 'Six-dimension scores, per-question evidence and role match; practice never changes scores, formal retries use new questions.' },
  ] as const;

  return (
    <>
      <PageHeader kicker={t('landing.demoSection')} title={t('landing.demoSection')} description={t('landing.demoDesc')} />
      <p className="mb-6"><Tint tone="warning">{t('common.syntheticBadge')}</Tint></p>

      <div className="mgd-grid mgd-grid--2 mb-8">
        {steps.map((s) => (
          <Card key={s.title} className="mgd-card--hoverable p-6">
            <div className="mb-3 grid size-11 place-items-center rounded-xl bg-[var(--mgd-app-brand-soft)] text-[var(--mgd-app-brand-ink)]">
              {s.icon}
            </div>
            <h2 className="mb-2 text-lg font-semibold text-neutral-900">{s.title}</h2>
            <p className="mb-0 text-sm leading-6 text-neutral-600">{s.desc}</p>
          </Card>
        ))}
      </div>

      <Card className="mgd-card--brand">
        <CardHeader title={zh ? '下一步' : 'Next step'} />
        <CardBody className="flex flex-wrap items-center justify-between gap-4">
          <p className="mb-0 text-sm text-neutral-700">
            {zh ? '体验完整流程？进入工作台创建你的第一个项目（需登录，16 岁以上）。' : 'Want the full flow? Create your first project from the dashboard (sign-in required, ages 16+).'}
          </p>
          <div className="flex gap-3">
            <a href={`/${locale}/auth`} className="mgd-target-primary inline-flex items-center gap-2 rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-5 font-semibold text-white shadow-[var(--mgd-app-shadow-brand)]">
              {t('auth.welcome')}
              <IconArrowRight size={16} />
            </a>
            <a href={`/${locale}/dashboard`} className="mgd-target-primary inline-flex items-center gap-2 rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-5 font-semibold text-neutral-800">
              <IconPlay size={16} className="text-primary" />
              {t('nav.dashboard')}
            </a>
          </div>
        </CardBody>
      </Card>
    </>
  );
}
