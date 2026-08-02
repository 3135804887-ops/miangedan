/** SCR-12 练习页：原题/变体、提示/框架/示例、逐步反馈；固定"不改分"标识。 */

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
} from '@mgd/ui';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

export default async function PracticePage({
  params,
}: {
  params: Promise<{ locale: string; id: string; pid: string }>;
}): Promise<ReactNode> {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  const zh = locale === 'zh-CN';

  const framework = zh
    ? ['情境（S）：时间、背景、干系人', '任务（T）：你的职责与目标', '行动（A）：具体动作与沟通方式', '结果（R）：量化结果与复盘']
    : ['Situation (S): timing, context, stakeholders', 'Task (T): your responsibility and goal', 'Action (A): concrete steps and communication', 'Result (R): quantified outcome and review'];

  return (
    <>
      <PageHeader kicker={t('practice.kicker')} title={t('practice.title')} description={t('practice.desc')} />
      <p className="mb-5 inline-flex items-center gap-2 rounded-full bg-info/10 px-4 py-1.5 text-sm font-medium text-info">
        <IconAlert size={14} />
        {t('practice.noScoreChange')}
      </p>

      <div className="mgd-grid mgd-grid--sidebar">
        <div className="space-y-4">
          <Card>
            <CardHeader
              title={<span className="flex items-center gap-2"><IconMessage size={18} className="text-primary" />{zh ? '练习题目' : 'Practice question'}</span>}
              actions={<Tint tone="brand">{zh ? '原题练习' : 'Original'}</Tint>}
            />
            <CardBody>
              <p className="mb-0 text-[15px] leading-7 text-neutral-800">
                {zh ? '请用一个具体案例说明你如何推动跨团队协作。' : 'Give a concrete example of how you drove cross-team collaboration.'}
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
                    content: <p className="mb-0 text-sm text-neutral-700">{zh ? '使用 STAR 结构组织回答，行动部分给出 2-3 个具体动作。' : 'Use the STAR structure; give 2-3 concrete actions.'}</p>,
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
                    content: <p className="mb-0 text-sm text-neutral-700">{zh ? '在订单中心重构中，与算法团队就降级策略达成一致：先定义优先级矩阵，再按流量比例灰度……' : 'During the order-centre refactor, alignment with the algorithm team on degradation: define a priority matrix, then roll out by traffic share…'}</p>,
                  },
                ]}
              />
            </CardBody>
          </Card>
          <div className="space-y-2">
            <Button variant="primary" className="w-full">
              {t('practice.getFeedback')}
              <IconArrowRight size={15} />
            </Button>
            <Button variant="secondary" className="w-full">{t('practice.variant')}</Button>
            <Button variant="danger" className="w-full">{t('practice.end')}</Button>
          </div>
        </div>
      </div>
    </>
  );
}
