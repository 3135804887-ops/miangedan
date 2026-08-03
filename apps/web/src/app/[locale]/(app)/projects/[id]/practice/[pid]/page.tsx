/** SCR-12 练习页：原题/变体、提示/框架/示例、逐步反馈；固定“不改分”标识。 */

'use client';

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  IconAlert,
  IconArrowRight,
  IconCheck,
  IconMessage,
  IconSparkle,
  PageHeader,
  Tabs,
  Tint,
  useToast,
} from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useParams } from 'next/navigation';
import { useEffect, useState, type ReactNode } from 'react';

import { apiFetch } from '../../../../../../../lib/api-fetch.ts';

interface PracticePayload {
  readonly practice: {
    readonly practice_id: string;
    readonly question: string;
    readonly variant?: string;
    readonly hint?: string;
    readonly framework?: readonly string[];
    readonly example?: string;
  };
}

export default function PracticePage(): ReactNode {
  const t = useTranslations('common');
  const toast = useToast();
  const params = useParams<{ locale: 'zh-CN' | 'en-US'; id: string; pid: string }>();
  const locale = params.locale;
  const zh = locale === 'zh-CN';
  const projectId = params.id;
  const [practice, setPractice] = useState<PracticePayload['practice'] | undefined>(undefined);
  const [unavailable, setUnavailable] = useState(false);

  useEffect(() => {
    let alive = true;
    void (async () => {
      const res = await apiFetch<PracticePayload>('/v1/projects/{projectId}/practice', {
        method: 'post',
        idempotencyKey: `practice-create-${projectId}-${Date.now()}`,
        pathParams: { projectId },
      });
      if (!alive) return;
      if (res.ok) setPractice(res.data.practice);
      else setUnavailable(true);
    })();
    return () => {
      alive = false;
    };
  }, [projectId]);

  const framework = practice?.framework?.length
    ? practice.framework
    : [
        zh ? '情境（S）：时间、背景、干系人' : 'Situation (S): timing, context, stakeholders',
        zh ? '任务（T）：你的职责与目标' : 'Task (T): your responsibility and goal',
        zh ? '行动（A）：具体动作与沟通方式' : 'Action (A): concrete steps and communication',
        zh ? '结果（R）：量化结果与复盘' : 'Result (R): quantified outcome and review',
      ];

  const endPractice = async () => {
    if (practice !== undefined) {
      await apiFetch('/v1/practice/{practiceId}/end', {
        method: 'post',
        idempotencyKey: `practice-end-${practice.practice_id}-${Date.now()}`,
        pathParams: { practiceId: practice.practice_id },
      });
    }
    toast.push({ title: zh ? '练习已结束（不影响正式分数）' : 'Practice ended (formal score unchanged)', tone: 'success' });
    window.location.href = `/${locale}/projects/${projectId}/report`;
  };

  return (
    <>
      <PageHeader kicker={t('practice.kicker')} title={t('practice.title')} description={t('practice.desc')} />
      <p className="mb-5 inline-flex items-center gap-2 rounded-full bg-info/10 px-4 py-1.5 text-sm font-medium text-info">
        <IconAlert size={14} />
        {t('practice.noScoreChange')}
      </p>

      {unavailable ? (
        <p className="mb-5 rounded-xl bg-warning/10 px-4 py-3 text-sm text-warning-text" role="status">
          {zh ? '练习服务暂未接入（占位），当前展示合成练习内容。' : 'Practice service placeholder — synthetic practice content.'}
        </p>
      ) : null}

      <div className="mgd-grid mgd-grid--sidebar">
        <div className="space-y-4">
          <Card>
            <CardHeader
              title={<span className="flex items-center gap-2"><IconMessage size={18} className="text-primary" />{zh ? '练习题目' : 'Practice question'}</span>}
              actions={<Tint tone="brand">{zh ? '原题练习' : 'Original'}</Tint>}
            />
            <CardBody>
              <p className="mb-0 text-[15px] leading-7 text-neutral-800">
                {practice?.question ?? (zh ? '请用一个具体案例说明你如何推动跨团队协作。' : 'Give a concrete example of how you drove cross-team collaboration.')}
              </p>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-neutral-900">
                <IconSparkle size={16} className="text-primary" />
                {t('practice.feedback')}
              </div>
              <div className="rounded-xl bg-[var(--mgd-app-surface-muted)] p-4 text-sm leading-6 text-neutral-700">
                {zh
                  ? '你的回答结构完整，但缺少量化结果。建议补充：协作前后效率对比、冲突解决的具体沟通方式，以及复盘后的改进。'
                  : 'Your answer is well structured but lacks quantified results. Add before/after efficiency metrics, the concrete communication used, and what changed after review.'}
              </div>
            </CardBody>
          </Card>
        </div>

        <div className="space-y-4">
          <Card>
            <CardHeader title={zh ? '提示与框架' : 'Hint and framework'} />
            <CardBody>
              <Tabs
                initialId="hint"
                items={[
                  {
                    id: 'hint',
                    label: t('practice.hint'),
                    content: <p className="mb-0 text-sm text-neutral-700">{practice?.hint ?? (zh ? '使用 STAR 结构组织回答，行动部分给出 2-3 个具体动作。' : 'Use the STAR structure; give 2-3 concrete actions.')}</p>,
                  },
                  {
                    id: 'framework',
                    label: t('practice.framework'),
                    content: (
                      <ol className="mb-0 space-y-2 text-sm text-neutral-700">
                        {framework.map((f) => (
                          <li key={f} className="flex gap-2"><IconCheck size={14} className="mt-1 shrink-0 text-success" />{f}</li>
                        ))}
                      </ol>
                    ),
                  },
                  {
                    id: 'example',
                    label: t('practice.example'),
                    content: <p className="mb-0 text-sm text-neutral-700">{practice?.example ?? (zh ? '在订单中心重构中，与算法团队就降级策略达成一致：先定义优先级矩阵，再按流量比例灰度……' : 'During the order-centre refactor, alignment with the algorithm team on degradation: define a priority matrix, then roll out by traffic share…')}</p>,
                  },
                ]}
              />
            </CardBody>
          </Card>
          <div className="space-y-2">
            <Button variant="primary" className="w-full" onClick={() => toast.push({ title: zh ? '已提交练习作答（隔离，不改分）' : 'Practice answer submitted (isolated, no score change)', tone: 'success' })}>
              {t('practice.getFeedback')}
              <IconArrowRight size={15} />
            </Button>
            <Button variant="secondary" className="w-full" onClick={() => toast.push({ title: zh ? '变体练习（仅关联已考覆盖点）' : 'Variant practice (linked to covered points)', tone: 'info' })}>{t('practice.variant')}</Button>
            <Button variant="danger" className="w-full" onClick={endPractice}>{t('practice.end')}</Button>
          </div>
        </div>
      </div>
    </>
  );
}
