/** SCR-07 会前检查：设备/网络/数字人检测、便利设置冻结、额度预留说明。 */

import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { PrecheckView } from '../../../../../../components/precheck-view.tsx';

export default async function PrecheckPage({
  params,
}: {
  params: Promise<{ locale: string; id: string }>;
}): Promise<ReactNode> {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  return (
    <PrecheckView
      locale={locale as 'zh-CN' | 'en-US'}
      projectId={id}
      labels={{
        kicker: t('precheck.kicker'),
        title: t('precheck.title'),
        desc: t('precheck.desc'),
        camera: t('precheck.camera'),
        mic: t('precheck.mic'),
        network: t('precheck.network'),
        speaker: t('precheck.speaker'),
        avatar: t('precheck.avatar'),
        checking: t('precheck.checking'),
        passed: t('precheck.passed'),
        failed: t('precheck.failed'),
        retryItem: t('precheck.retryItem'),
        cameraOff: t('precheck.cameraOff'),
        micOff: t('precheck.micOff'),
        networkPoor: t('precheck.networkPoor'),
        accommodations: t('precheck.accommodations'),
        accommodationsHint: t('precheck.accommodationsHint'),
        freeze: t('precheck.freeze'),
        entitlement: t('precheck.entitlement'),
        entitlementHint: t('precheck.entitlementHint'),
        entitlementInsufficient: t('precheck.entitlementInsufficient'),
      }}
    />
  );
}
