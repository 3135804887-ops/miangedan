/** SCR-08 实时面试房间（含 SCR-09 故障/重连/降级覆盖层）。 */

import { getTranslations, setRequestLocale } from 'next-intl/server';
import type { ReactNode } from 'react';

import { RoomView } from '../../../../../components/room-view.tsx';

export default async function SessionPage({
  params,
}: {
  params: Promise<{ locale: string; id: string }>;
}): Promise<ReactNode> {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  return (
    <RoomView
      locale={locale as 'zh-CN' | 'en-US'}
      sessionId={id}
      labels={{
        title: t('room.title'),
        round: t('room.round', { n: '{n}' }),
        live: t('room.live'),
        paused: t('room.paused'),
        reconnecting: t('room.reconnecting'),
        timer: t('room.timer'),
        network: t('room.network'),
        subtitles: t('room.subtitles'),
        on: t('room.on'),
        off: t('room.off'),
        exit: t('room.exit'),
        exitDialogTitle: t('room.exitDialogTitle'),
        exitDialogBody: t('room.exitDialogBody'),
        question: t('room.question'),
        questionHint: t('room.questionHint'),
        candidateTranscript: t('room.candidateTranscript'),
        revisionHint: t('room.revisionHint'),
        revise: t('room.revise'),
        revised: t('room.revised'),
        frozen: t('room.frozen'),
        dynamicFollowup: t('room.dynamicFollowup'),
        tools: t('room.tools'),
        noToolsHint: t('room.noToolsHint'),
        controls: t('room.controls'),
        stopAvatar: t('room.stopAvatar'),
        textAnswer: t('room.textAnswer'),
        submitAnswer: t('room.submitAnswer'),
        cameraLabel: t('room.cameraLabel'),
        micLabel: t('room.micLabel'),
        turnProgress: t('room.turnProgress'),
        systemPaused: t('room.systemPaused'),
        systemPausedHint: t('room.systemPausedHint'),
        reconnectCountdown: t('room.reconnectCountdown', { seconds: '{seconds}' }),
        reconnectHint: t('room.reconnectHint'),
        downgradeTitle: t('room.downgradeTitle'),
        downgradeBody: t('room.downgradeBody'),
        downgradeAccept: t('room.downgradeAccept'),
        downgradeDecline: t('room.downgradeDecline'),
        downgradeDeclineHint: t('room.downgradeDeclineHint'),
        authPaused: t('room.authPaused'),
        authPausedHint: t('room.authPausedHint'),
        reauthenticate: t('room.reauthenticate'),
      }}
    />
  );
}
