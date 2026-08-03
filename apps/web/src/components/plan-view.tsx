'use client';

import {
  AlertDialog,
  Button,
  Card,
  CardBody,
  CardHeader,
  IconCheck,
  IconClock,
  IconCode,
  IconExternal,
  IconPlus,
  IconSparkle,
  PageHeader,
  Tint,
  useToast,
} from '@mgd/ui';
import { useState } from 'react';

import { mockPlan, type MockPlanRound } from '../mocks/data.ts';

interface Labels {
  readonly kicker: string;
  readonly title: string;
  readonly desc: string;
  readonly roundCount: string;
  readonly addRound: string;
  readonly roundType: string;
  readonly role: string;
  readonly focus: string;
  readonly duration: string;
  readonly difficulty: string;
  readonly criticalDimensions: string;
  readonly tools: string;
  readonly noTools: string;
  readonly notReady: string;
  readonly notReadyHint: string;
  readonly confirmPlan: string;
  readonly confirmDialogTitle: string;
  readonly confirmDialogBody: string;
  readonly frozen: string;
  readonly frozenHint: string;
  readonly quoteTotal: string;
  readonly quoteRetry: string;
  readonly sourceRef: string;
  readonly genericTemplate: string;
  readonly difficultyBasic: string;
  readonly difficultyStandard: string;
  readonly difficultyChallenge: string;
}

const DIFFICULTY_LABEL: Readonly<Record<string, string>> = {
  basic: 'difficultyBasic',
  standard: 'difficultyStandard',
  challenge: 'difficultyChallenge',
};

const DIMENSION_LABELS: Readonly<Record<string, string>> = {
  professional_competence: '专业能力',
  problem_solving: '问题解决',
  communication: '沟通表达',
  experience_evidence: '经历证据',
  behavioral_collaboration: '行为协作',
  learning_adaptability: '学习适应',
};

export function PlanView({
  locale,
  labels,
}: {
  readonly locale: 'zh-CN' | 'en-US';
  readonly labels: Labels;
}): React.ReactNode {
  const toast = useToast();
  const [rounds, setRounds] = useState<readonly MockPlanRound[]>(mockPlan.rounds);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [frozen, setFrozen] = useState<boolean>(mockPlan.frozen);
  const totalMinutes = rounds.reduce((s, r) => s + r.duration_minutes, 0);
  const allReady = rounds.every((r) => r.ready);

  const confirm = () => {
    setFrozen(true);
    setConfirmOpen(false);
    toast.push({ title: locale === 'zh-CN' ? '计划已冻结' : 'Plan frozen', tone: 'success' });
  };

  const toggleReady = (sequence: number) => {
    setRounds((prev) => prev.map((r) => (r.sequence === sequence ? { ...r, ready: !r.ready } : r)));
  };

  return (
    <>
      <PageHeader
        kicker={labels.kicker}
        title={labels.title}
        description={labels.desc}
        actions={frozen ? <Tint tone="success">{labels.frozen}</Tint> : undefined}
      />

      <div className="mgd-grid mgd-grid--sidebar">
        <div className="space-y-4">
          {rounds.map((r) => (
            <Card key={r.sequence} className={r.ready ? '' : 'border-warning/50'}>
              <CardHeader
                title={
                  <span className="flex items-center gap-3">
                    <span className="grid size-9 place-items-center rounded-xl bg-[var(--mgd-app-brand-soft)] text-sm font-bold text-[var(--mgd-app-brand-ink)]">
                      {r.sequence}
                    </span>
                    <span>{r.role}</span>
                  </span>
                }
                description={`${labels.roundType} · ${r.round_type}`}
                actions={
                  r.ready ? (
                    <Tint tone="success"><IconCheck size={13} />{locale === 'zh-CN' ? '就绪' : 'Ready'}</Tint>
                  ) : (
                    <Tint tone="warning">{labels.notReady}</Tint>
                  )
                }
              />
              <CardBody>
                <div className="grid gap-4 sm:grid-cols-3">
                  <div>
                    <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">{labels.focus}</div>
                    <p className="mb-0 text-sm text-neutral-700">{r.focus}</p>
                  </div>
                  <div>
                    <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">{labels.duration}</div>
                    <p className="mb-0 flex items-center gap-1.5 text-sm text-neutral-700">
                      <IconClock size={14} />
                      {r.duration_minutes} min
                    </p>
                  </div>
                  <div>
                    <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">{labels.difficulty}</div>
                    <p className="mb-0 text-sm text-neutral-700">{labels[DIFFICULTY_LABEL[r.difficulty] as keyof Labels]}</p>
                  </div>
                </div>
                <div className="mt-4 grid gap-4 sm:grid-cols-2">
                  <div>
                    <div className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-neutral-500">{labels.criticalDimensions}</div>
                    <div className="flex flex-wrap gap-1.5">
                      {r.critical_dimensions.map((d) => (
                        <Tint key={d} tone="info">{DIMENSION_LABELS[d] ?? d}</Tint>
                      ))}
                    </div>
                  </div>
                  <div>
                    <div className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-neutral-500">{labels.tools}</div>
                    <div className="flex flex-wrap gap-1.5">
                      {r.tools.length === 0 ? (
                        <span className="text-sm text-neutral-500">{labels.noTools}</span>
                      ) : (
                        r.tools.map((tool) => (
                          <Tint key={tool} tone="neutral">
                            <IconCode size={13} />
                            {tool}
                          </Tint>
                        ))
                      )}
                    </div>
                  </div>
                </div>
                {!r.ready ? (
                  <div className="mt-4 flex flex-wrap items-center justify-between gap-2 rounded-xl bg-[color-mix(in_srgb,var(--mgd-color-warning)_8%,transparent)] px-4 py-3">
                    <p className="mb-0 text-sm text-warning-text">{labels.notReadyHint}</p>
                    <Button variant="secondary" targetSize="min" onClick={() => toggleReady(r.sequence)}>
                      {locale === 'zh-CN' ? '标记覆盖方案就绪' : 'Mark coverage ready'}
                    </Button>
                  </div>
                ) : null}
              </CardBody>
            </Card>
          ))}
          <Button variant="secondary" onClick={() => toast.push({ title: labels.addRound, tone: 'info' })}>
            <IconPlus size={16} />
            {labels.addRound}
          </Button>
        </div>

        <div className="space-y-4">
          <Card className="mgd-card--brand">
            <CardHeader
              title={
                <span className="flex items-center gap-2">
                  <IconSparkle size={18} className="text-[var(--mgd-app-brand-ink)]" />
                  {locale === 'zh-CN' ? 'AI 建议' : 'AI suggestion'}
                </span>
              }
              description={labels.roundCount.replace('{count}', String(rounds.length))}
            />
            <CardBody>
              <dl className="mb-0 space-y-3 text-sm">
                <div className="flex justify-between gap-3">
                  <dt className="text-neutral-600">{labels.quoteTotal.replace('{minutes}', String(totalMinutes))}</dt>
                  <dd className="mb-0 font-mono font-semibold text-neutral-900">{totalMinutes} min</dd>
                </div>
                <div className="flex justify-between gap-3">
                  <dt className="text-neutral-600">{labels.quoteRetry}</dt>
                  <dd className="mb-0 text-success">✓</dd>
                </div>
                <div className="flex justify-between gap-3">
                  <dt className="text-neutral-600">{labels.sourceRef}</dt>
                  <dd className="mb-0">
                    <a href="#" className="inline-flex items-center gap-1 text-primary hover:underline">
                      官方招聘页 <IconExternal size={12} />
                    </a>
                  </dd>
                </div>
              </dl>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <div className="mb-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">{labels.duration}</div>
              <div className="flex h-2.5 overflow-hidden rounded-full bg-neutral-100">
                {rounds.map((r) => (
                  <div
                    key={r.sequence}
                    className={r.ready ? 'bg-[var(--mgd-app-brand-from)]' : 'bg-warning'}
                    style={{ width: `${(r.duration_minutes / totalMinutes) * 100}%` }}
                  />
                ))}
              </div>
              <p className="mb-0 mt-2 text-xs text-neutral-500">{locale === 'zh-CN' ? '各轮时长占比' : 'Duration share per round'}</p>
            </CardBody>
          </Card>

          <Button variant="primary" className="w-full" disabled={!allReady} disabledReason={allReady ? undefined : labels.notReadyHint} onClick={() => setConfirmOpen(true)}>
            {labels.confirmPlan}
          </Button>
        </div>
      </div>

      <AlertDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title={labels.confirmDialogTitle}
        description={labels.confirmDialogBody}
      >
        <Button variant="secondary" onClick={() => setConfirmOpen(false)}>
          {locale === 'zh-CN' ? '取消' : 'Cancel'}
        </Button>
        <Button variant="primary" onClick={confirm}>
          <IconCheck size={16} />
          {labels.confirmPlan}
        </Button>
      </AlertDialog>

      {frozen ? (
        <p className="mt-4 rounded-xl bg-success/10 px-4 py-3 text-sm text-success-text">{labels.frozenHint}</p>
      ) : null}
    </>
  );
}
