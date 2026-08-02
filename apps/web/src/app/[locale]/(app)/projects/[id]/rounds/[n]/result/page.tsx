/** SCR-10 轮次结果：通过/未通过/评估未完成三态（红线文案快照）。 */

import {
  Button,
  Card,
  CardBody,
  IconChart,
  IconCheck,
  IconClock,
  IconPlay,
  IconRefresh,
  PageHeader,
  StatCard,
  Tint,
} from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function RoundResultPage({
  params,
}: {
  params: Promise<{ locale: string; id: string; n: string }>;
}): Promise<ReactNode> {
  const { locale, n } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';

  const dimensions = [
    { key: 'professional_competence', label: zh ? '专业能力' : 'Professional competence', score: 78, critical: true },
    { key: 'problem_solving', label: zh ? '问题解决' : 'Problem solving', score: 72, critical: true },
    { key: 'communication', label: zh ? '沟通表达' : 'Communication', score: 81, critical: false },
    { key: 'experience_evidence', label: zh ? '经历证据' : 'Experience evidence', score: 76, critical: true },
    { key: 'behavioral_collaboration', label: zh ? '行为协作' : 'Collaboration', score: 68, critical: false },
    { key: 'learning_adaptability', label: zh ? '学习适应' : 'Learning', score: 70, critical: false },
  ] as const;

  const base = `/${locale}/projects/p-0001`;

  return (
    <>
      <PageHeader kicker={t('result.kicker')} title={`${zh ? '第' : 'Round'} ${n} ${zh ? '结果' : 'result'}`} />

      <Card className="mgd-card--brand mb-6 overflow-hidden">
        <div className="bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-7 py-8 text-white">
          <div className="mb-2 inline-flex items-center gap-2 rounded-full bg-white/15 px-3 py-1 text-xs font-semibold backdrop-blur">
            <IconCheck size={13} />
            PASS
          </div>
          <h2 className="mb-2 text-2xl font-bold tracking-tight">{t('result.passedTitle')}</h2>
          <p className="mb-0 text-white/85">{t('result.passedDesc')}</p>
        </div>
        <CardBody>
          <div className="mgd-grid mgd-grid--3">
            <StatCard label={t('result.score')} value="74 / 100" tone="brand" hint={`${t('result.passLine')} · 60`} />
            <StatCard label={t('result.criticalGate')} value="4 / 4" tone="success" hint={zh ? '全部关键维度达标' : 'All critical dimensions met'} />
            <StatCard label={t('result.nextRound')} value="2" tone="info" hint={`${t('result.round', { n: '2' })} · 30 min`} />
          </div>
        </CardBody>
      </Card>

      <div className="mgd-grid mgd-grid--sidebar">
        <Card>
          <CardBody>
            <h3 className="mb-4 text-base font-semibold">{zh ? '维度得分' : 'Dimension scores'}</h3>
            <div className="space-y-4">
              {dimensions.map((d) => (
                <div key={d.key}>
                  <div className="mb-1 flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2 text-neutral-700">
                      {d.label}
                      {d.critical ? <Tint tone="brand">{zh ? '关键' : 'Critical'}</Tint> : null}
                    </span>
                    <span className={`font-mono font-semibold ${d.score >= 60 ? 'text-success-text' : 'text-danger'}`}>{d.score}</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-neutral-100">
                    <div
                      className={`h-full rounded-full ${d.score >= 60 ? 'bg-success' : 'bg-danger'}`}
                      style={{ width: `${d.score}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </CardBody>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardBody>
              <h3 className="mb-2 text-base font-semibold">{t('result.strengths')}</h3>
              <ul className="mb-0 space-y-1.5 text-sm text-neutral-700">
                {[zh ? '结构化表达清晰' : 'Clear structured answers', zh ? '项目复杂度把握准确' : 'Accurate complexity assessment'].map((s) => (
                  <li key={s} className="flex gap-2">
                    <IconCheck size={15} className="mt-1 shrink-0 text-success" />
                    {s}
                  </li>
                ))}
              </ul>
              <h3 className="mb-2 mt-5 text-base font-semibold">{t('result.attention')}</h3>
              <ul className="mb-0 space-y-1.5 text-sm text-neutral-700">
                <li className="flex gap-2">
                  <IconClock size={15} className="mt-1 shrink-0 text-warning" />
                  {zh ? '行为类回答偏短，建议补充团队协作实例' : 'Behavioral answers are brief; add team-collaboration examples'}
                </li>
              </ul>
            </CardBody>
          </Card>
          <Card>
            <CardBody className="space-y-3">
              <Button variant="primary" className="w-full" onClick={() => { window.location.href = `/${locale}/sessions/s-0001`; }}>
                <IconPlay size={16} />
                {t('result.enterNext')}
              </Button>
              <Button variant="secondary" className="w-full" onClick={() => { window.location.href = `${base}/report`; }}>
                <IconChart size={16} />
                {t('result.report')}
              </Button>
              <Button variant="secondary" className="w-full" onClick={() => { window.location.href = `${base}/practice/pr-0001`; }}>
                <IconRefresh size={16} />
                {t('result.practice')}
              </Button>
            </CardBody>
          </Card>
        </div>
      </div>
    </>
  );
}
