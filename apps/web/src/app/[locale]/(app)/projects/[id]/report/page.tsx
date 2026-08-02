/** SCR-11 完整报告：总体、轨迹、岗位匹配、雷达图+表格等价、逐题证据、训练计划。 */

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  IconChart,
  IconCheck,
  IconDownload,
  IconRadar,
  IconRefresh,
  IconSparkle,
  IconTrash,
  PageHeader,
  Tint,
} from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

const DIMS = ['professional_competence', 'problem_solving', 'communication', 'experience_evidence', 'behavioral_collaboration', 'learning_adaptability'] as const;
const SCORES = [80, 76, 82, 74, 72, 78] as const;

function RadarChart({ size = 280 }: { size?: number }): ReactNode {
  const cx = size / 2;
  const cy = size / 2;
  const r = size / 2 - 34;
  const angle = (i: number) => (Math.PI * 2 * i) / DIMS.length - Math.PI / 2;
  const point = (i: number, value: number) => {
    const rad = angle(i);
    return [cx + r * (value / 100) * Math.cos(rad), cy + r * (value / 100) * Math.sin(rad)];
  };
  const poly = SCORES.map((v, i) => point(i, v).join(',')).join(' ');
  const rings = [25, 50, 75, 100].map((v) => DIMS.map((_, i) => point(i, v).join(',')).join(' '));
  return (
    <svg viewBox={`0 0 ${size} ${size}`} className="mx-auto" role="img" aria-label="六维雷达图">
      {rings.map((ring) => (
        <polygon key={ring} points={ring} fill="none" stroke="var(--mgd-app-border-default)" strokeWidth={1} />
      ))}
      {DIMS.map((d, i) => {
        const [x, y] = point(i, 100);
        return <line key={d} x1={cx} y1={cy} x2={x} y2={y} stroke="var(--mgd-app-border-default)" strokeWidth={1} />;
      })}
      <polygon points={poly} fill="color-mix(in srgb, var(--mgd-app-brand-from) 18%, transparent)" stroke="var(--mgd-app-brand-from)" strokeWidth={2} strokeLinejoin="round" />
      {DIMS.map((d, i) => {
        const [x, y] = point(i, SCORES[i] ?? 0);
        return <circle key={d} cx={x} cy={y} r={3.5} fill="var(--mgd-app-brand-from)" />;
      })}
      {DIMS.map((d, i) => {
        const [x, y] = point(i, 118);
        return (
          <text key={d} x={x} y={y} textAnchor="middle" dominantBaseline="middle" fontSize={11} fill="var(--mgd-color-neutral-600)">
            {d.split('_')[0]}
          </text>
        );
      })}
    </svg>
  );
}

export default async function ReportPage({
  params,
}: {
  params: Promise<{ locale: string; id: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';
  const dimLabels: Record<string, string> = {
    professional_competence: zh ? '专业能力' : 'Professional',
    problem_solving: zh ? '问题解决' : 'Problem solving',
    communication: zh ? '沟通表达' : 'Communication',
    experience_evidence: zh ? '经历证据' : 'Experience',
    behavioral_collaboration: zh ? '行为协作' : 'Collaboration',
    learning_adaptability: zh ? '学习适应' : 'Learning',
  };

  const evidence = [
    { q: zh ? '请介绍一个你主导的复杂项目' : 'Present a complex project you led', summary: zh ? '订单中心重构：拆分策略与容量评估' : 'Order-centre refactor: split strategy and capacity planning', score: 78, good: zh ? '量化结果明确、权衡过程清晰' : 'Clear metrics and trade-offs', gap: zh ? '未提及跨团队协作细节' : 'Cross-team collaboration detail missing', suggestion: zh ? '补充决策失败的备选方案' : 'Add alternative options considered' },
    { q: zh ? '线上事故如何定位与恢复？' : 'How do you diagnose and recover from incidents?', summary: zh ? '监控→隔离→回滚→复盘四步' : 'Monitor, isolate, roll back, review', score: 82, good: zh ? '流程完整、RTO 意识强' : 'Complete process, strong RTO awareness', gap: '', suggestion: zh ? '统一压测结论口径' : 'Align load-test conclusions' },
  ] as const;

  return (
    <>
      <PageHeader
        kicker={t('report.kicker')}
        title={t('report.title')}
        description={t('report.desc')}
        actions={
          <>
            <Button variant="secondary">
              <IconDownload size={16} />
              {t('report.export')}
            </Button>
            <Button variant="danger">
              <IconTrash size={16} />
              {t('report.delete')}
            </Button>
          </>
        }
      />
      <p className="mb-5 inline-flex items-center gap-2 rounded-full bg-warning/10 px-4 py-1.5 text-sm text-warning-text">
        <IconSparkle size={14} />
        {t('report.exportDisclaimer')}
      </p>

      <div className="mgd-grid mgd-grid--3 mb-5">
        <Card className="mgd-card--brand p-5">
          <div className="text-sm text-neutral-600">{t('report.overall')}</div>
          <div className="mgd-stat-value mt-1 text-[var(--mgd-app-brand-ink)]">76 / 100</div>
          <div className="mt-2"><Tint tone="success">{zh ? '全部轮次通过' : 'All rounds passed'}</Tint></div>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('report.jobMatch')}</div>
          <div className="mgd-stat-value mt-1 text-success-text">82%</div>
          <div className="mt-2 text-xs text-neutral-500">{zh ? '必备项 5/6 · 加分项 2/3' : 'Must-have 5/6 · Plus 2/3'}</div>
        </Card>
        <Card className="p-5">
          <div className="text-sm text-neutral-600">{t('report.roundTrajectory')}</div>
          <div className="mt-2 space-y-1 font-mono text-sm">
            <div className="flex justify-between"><span>R1</span><span className="text-success-text">71</span></div>
            <div className="flex justify-between"><span>R2</span><span className="text-success-text">78</span></div>
            <div className="flex justify-between"><span>R3</span><span className="text-success-text">80</span></div>
          </div>
        </Card>
      </div>

      <div className="mgd-grid mgd-grid--sidebar mb-5">
        <Card>
          <CardHeader title={<span className="flex items-center gap-2"><IconRadar size={18} className="text-primary" />{t('report.radar')}</span>} description={t('report.tableEquivalent')} />
          <CardBody>
            <RadarChart />
            <table className="mt-4 w-full border-collapse text-left text-sm">
              <caption className="sr-only">{t('report.tableEquivalent')}</caption>
              <thead>
                <tr className="border-b border-neutral-100 text-xs uppercase tracking-wide text-neutral-500">
                  <th className="py-2 pr-3 font-semibold">{zh ? '维度' : 'Dimension'}</th>
                  <th className="py-2 pr-3 font-semibold">{zh ? '得分' : 'Score'}</th>
                  <th className="py-2 font-semibold">{zh ? '达标' : 'Met'}</th>
                </tr>
              </thead>
              <tbody>
                {DIMS.map((d, i) => (
                  <tr key={d} className="border-b border-neutral-100 last:border-0">
                    <td className="py-2.5 pr-3 text-neutral-700">{dimLabels[d]}</td>
                    <td className="py-2.5 pr-3 font-mono text-neutral-900">{SCORES[i] ?? 0}</td>
                    <td className="py-2.5">{(SCORES[i] ?? 0) >= 60 ? <IconCheck size={15} className="text-success" /> : <span className="text-danger">✕</span>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardBody>
        </Card>

        <Card>
          <CardHeader title={t('report.trainingPlan')} />
          <CardBody>
            <ol className="mb-0 space-y-2.5 text-sm text-neutral-700">
              {[zh ? '行为类 STAR 练习 ×3' : 'Behavioral STAR practice ×3', zh ? '系统设计表达框架练习' : 'System-design framing practice', zh ? '矛盾点一致性校准' : 'Contradiction calibration'].map((item) => (
                <li key={item} className="flex gap-2 rounded-xl bg-[var(--mgd-app-surface-muted)] px-3 py-2.5">
                  <IconChart size={15} className="mt-0.5 shrink-0 text-primary" />
                  {item}
                </li>
              ))}
            </ol>
            <Button variant="secondary" className="mt-4 w-full" onClick={() => { window.location.href = `/${locale}/projects/p-0006/practice/pr-0001`; }}>
              {zh ? '开始练习' : 'Start practice'}
            </Button>
            <Button variant="primary" className="mt-2 w-full">
              <IconRefresh size={15} />
              {t('report.reviewRequest')}
            </Button>
            <p className="mb-0 mt-2 text-center text-xs text-neutral-500">{t('report.reviewOnce')}</p>
          </CardBody>
        </Card>
      </div>

      <Card>
        <CardHeader title={t('report.evidence')} />
        <CardBody className="space-y-4">
          {evidence.map((e, i) => (
            <article key={i} className="rounded-xl border border-neutral-100 p-5">
              <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                <h3 className="m-0 text-sm font-semibold text-neutral-900">{e.q}</h3>
                <span className="font-mono text-sm font-semibold text-neutral-900">{e.score}</span>
              </div>
              <p className="mb-3 text-sm text-neutral-600">{e.summary}</p>
              <dl className="mb-0 grid gap-2 text-sm sm:grid-cols-2">
                <div className="rounded-lg bg-success/5 px-3 py-2">
                  <dt className="text-xs font-semibold text-success-text">{t('report.strengths')}</dt>
                  <dd className="mb-0 mt-0.5 text-neutral-700">{e.good}</dd>
                </div>
                {e.gap !== '' ? (
                  <div className="rounded-lg bg-warning/5 px-3 py-2">
                    <dt className="text-xs font-semibold text-warning-text">{t('report.gaps')}</dt>
                    <dd className="mb-0 mt-0.5 text-neutral-700">{e.gap}</dd>
                  </div>
                ) : null}
                <div className="rounded-lg bg-info/5 px-3 py-2 sm:col-span-2">
                  <dt className="text-xs font-semibold text-info">{t('report.suggestions')}</dt>
                  <dd className="mb-0 mt-0.5 text-neutral-700">{e.suggestion}</dd>
                </div>
              </dl>
            </article>
          ))}
        </CardBody>
      </Card>
    </>
  );
}
