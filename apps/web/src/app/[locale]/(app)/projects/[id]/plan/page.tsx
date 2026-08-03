/** SCR-06 面试计划页：轮次编辑、覆盖方案、确认冻结、报价与来源。 */

import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { PlanView } from '../../../../../../components/plan-view.tsx';

export default async function PlanPage({
  params,
}: {
  params: Promise<{ locale: string; id: string }>;
}): Promise<ReactNode> {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  return (
    <PlanView
      locale={locale as 'zh-CN' | 'en-US'}
      projectId={id}
      labels={{
        kicker: t('plan.kicker'),
        title: t('plan.title'),
        desc: t('plan.desc'),
        roundCount: t('plan.roundCount', { count: '{count}' }),
        addRound: t('plan.addRound'),
        roundType: t('plan.roundType'),
        role: t('plan.role'),
        focus: t('plan.focus'),
        duration: t('plan.duration'),
        difficulty: t('plan.difficulty'),
        criticalDimensions: t('plan.criticalDimensions'),
        tools: t('plan.tools'),
        noTools: t('plan.noTools'),
        notReady: t('plan.notReady'),
        notReadyHint: t('plan.notReadyHint'),
        confirmPlan: t('plan.confirmPlan'),
        confirmDialogTitle: t('plan.confirmDialogTitle'),
        confirmDialogBody: t('plan.confirmDialogBody'),
        frozen: t('plan.frozen'),
        frozenHint: t('plan.frozenHint'),
        quoteTotal: t('plan.quoteTotal', { minutes: '{minutes}' }),
        quoteRetry: t('plan.quoteRetry'),
        sourceRef: t('plan.sourceRef'),
        genericTemplate: t('plan.genericTemplate'),
        difficultyBasic: t('plan.difficultyMap.basic'),
        difficultyStandard: t('plan.difficultyMap.standard'),
        difficultyChallenge: t('plan.difficultyMap.challenge'),
      }}
    />
  );
}
