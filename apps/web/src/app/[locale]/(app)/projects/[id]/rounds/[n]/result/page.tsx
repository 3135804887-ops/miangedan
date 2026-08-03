/** SCR-10 轮次结果：通过/未通过/评估未完成三态（红线文案快照）。 */

'use client';

import {
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
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';

import { isPlaceholder, useApiGet } from '../../../../../../../../lib/api-hooks.ts';

interface ResultPayload {
  readonly result: {
    readonly result_status: string;
    readonly round_total: number;
    readonly pass_line: number;
    readonly critical_gate_passed: boolean;
    readonly dimensions: readonly {
      dimension: string;
      score: number;
      is_critical: boolean;
    }[];
    readonly strengths: readonly string[];
    readonly attention: readonly string[];
    readonly next_round: {
      sequence: number;
      role: string;
      focus: string;
      difficulty: string;
      duration_minutes: number;
    };
  };
}

const DIMENSION_LABELS: Readonly<Record<string, [string, string]>> = {
  professional_competence: ['专业能力', 'Professional competence'],
  problem_solving: ['问题解决', 'Problem solving'],
  communication: ['沟通表达', 'Communication'],
  experience_evidence: ['经历证据', 'Experience evidence'],
  behavioral_collaboration: ['行为协作', 'Collaboration'],
  learning_adaptability: ['学习适应', 'Learning'],
};

export default function RoundResultPage(): React.ReactNode {
  const t = useTranslations('common');
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; id: string; n: string }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const projectId = params.id;
  const sequence = params.n;
  const state = useApiGet<ResultPayload>(
    '/v1/projects/{projectId}/rounds/{sequence}/result',
    { pathParams: { projectId, sequence } },
  );

  const result = state.data?.result;
  const base = `/${locale}/projects/${projectId}`;
  const passed = result?.result_status === 'PASS';
  const dimensions = result?.dimensions ?? [];
  const strengths = result?.strengths ?? [];
  const attention = result?.attention ?? [];

  return (
    <>
      <PageHeader kicker={t('result.kicker')} title={`${zh ? '第' : 'Round'} ${sequence} ${zh ? '结果' : 'result'}`} />

      {state.loading ? (
        <p className="mb-6 rounded-xl bg-surface-muted px-4 py-3 text-sm text-neutral-600" role="status" aria-live="polite">
          {zh ? '正在加载结果…' : 'Loading result…'}
        </p>
      ) : null}
      {state.failure !== undefined ? (
        <p className="mb-6 rounded-xl bg-warning/10 px-4 py-3 text-sm text-warning-text" role="status">
          {isPlaceholder(state.failure)
            ? (zh ? '评分结果服务暂未接入（占位），当前展示合成结果。' : 'Scoring result service placeholder — synthetic result.')
            : (zh ? '结果加载失败，请重试。' : 'Failed to load result. Please retry.')}
        </p>
      ) : null}

      <Card className="mgd-card--brand mb-6 overflow-hidden">
        <div className="bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-7 py-8 text-white">
          <div className="mb-2 inline-flex items-center gap-2 rounded-full bg-white/15 px-3 py-1 text-xs font-semibold backdrop-blur">
            <IconCheck size={13} />
            {result?.result_status ?? '—'}
          </div>
          <h2 className="mb-2 text-2xl font-bold tracking-tight">
            {passed ? t('result.passedTitle') : zh ? '本轮未通过' : 'Round not passed'}
          </h2>
          <p className="mb-0 text-white/85">{passed ? t('result.passedDesc') : zh ? '可复盘练习或正式重试后再进入下一轮。' : 'Practice or retry before the next round.'}</p>
        </div>
        <CardBody>
          <div className="mgd-grid mgd-grid--3">
            <StatCard label={t('result.score')} value={result ? `${result.round_total} / 100` : '—'} tone="brand" hint={result ? `${t('result.passLine')} · ${result.pass_line}` : undefined} />
            <StatCard
              label={t('result.criticalGate')}
              value={result ? (result.critical_gate_passed ? '✓' : '✗') : '—'}
              tone={result?.critical_gate_passed ? 'success' : 'danger'}
              hint={result ? (result.critical_gate_passed ? (zh ? '全部关键维度达标' : 'All critical dimensions met') : (zh ? '存在关键维度未达标' : 'Critical dimension below line')) : undefined}
            />
            <StatCard label={t('result.nextRound')} value={result ? String(result.next_round.sequence) : '—'} tone="info" hint={result ? `${t('result.round', { n: String(result.next_round.sequence) })} · ${result.next_round.duration_minutes} min` : undefined} />
          </div>
        </CardBody>
      </Card>

      <div className="mgd-grid mgd-grid--sidebar">
        <Card>
          <CardBody>
            <h3 className="mb-4 text-base font-semibold">{zh ? '维度得分' : 'Dimension scores'}</h3>
            <div className="space-y-4">
              {dimensions.length === 0 ? (
                <p className="mb-0 text-sm text-neutral-500">{zh ? '暂无维度数据' : 'No dimension data'}</p>
              ) : (
                dimensions.map((d) => {
                  const label = DIMENSION_LABELS[d.dimension]?.[zh ? 0 : 1] ?? d.dimension;
                  return (
                    <div key={d.dimension}>
                      <div className="mb-1 flex items-center justify-between text-sm">
                        <span className="flex items-center gap-2 text-neutral-700">
                          {label}
                          {d.is_critical ? <Tint tone="brand">{zh ? '关键' : 'Critical'}</Tint> : null}
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
                  );
                })
              )}
            </div>
          </CardBody>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardBody>
              <h3 className="mb-2 text-base font-semibold">{t('result.strengths')}</h3>
              <ul className="mb-0 space-y-1.5 text-sm text-neutral-700">
                {strengths.length === 0 ? (
                  <li className="text-neutral-500">{zh ? '暂无' : 'None'}</li>
                ) : (
                  strengths.map((s) => (
                    <li key={s} className="flex gap-2">
                      <IconCheck size={15} className="mt-1 shrink-0 text-success" />
                      {s}
                    </li>
                  ))
                )}
              </ul>
              <h3 className="mb-2 mt-5 text-base font-semibold">{t('result.attention')}</h3>
              <ul className="mb-0 space-y-1.5 text-sm text-neutral-700">
                {attention.length === 0 ? (
                  <li className="text-neutral-500">{zh ? '暂无' : 'None'}</li>
                ) : (
                  attention.map((s) => (
                    <li key={s} className="flex gap-2">
                      <IconClock size={15} className="mt-1 shrink-0 text-warning" />
                      {s}
                    </li>
                  ))
                )}
              </ul>
            </CardBody>
          </Card>
          <Card>
            <CardBody className="space-y-3">
              <a href={`${base}/precheck`} className="mgd-target-primary inline-flex w-full items-center justify-center gap-2 rounded-xl bg-[linear-gradient(135deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))] px-5 font-semibold text-white shadow-[var(--mgd-app-shadow-brand)]">
                <IconPlay size={16} />
                {t('result.enterNext')}
              </a>
              <a href={`${base}/report`} className="mgd-target-primary inline-flex w-full items-center justify-center gap-2 rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-5 font-semibold text-neutral-800 shadow-[var(--mgd-app-shadow-sm)] hover:border-[var(--mgd-app-border-strong)]">
                <IconChart size={16} />
                {t('result.report')}
              </a>
              <a href={`${base}/practice/pr-0001`} className="mgd-target-primary inline-flex w-full items-center justify-center gap-2 rounded-xl border border-[var(--mgd-app-border-default)] bg-surface px-5 font-semibold text-neutral-800 shadow-[var(--mgd-app-shadow-sm)] hover:border-[var(--mgd-app-border-strong)]">
                <IconRefresh size={16} />
                {t('result.practice')}
              </a>
            </CardBody>
          </Card>
        </div>
      </div>
    </>
  );
}
