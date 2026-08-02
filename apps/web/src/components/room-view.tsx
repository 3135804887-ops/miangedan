'use client';

import {
  AlertDialog,
  Button,
  IconAlert,
  IconCamera,
  IconCameraOff,
  IconCheck,
  IconClock,
  IconCode,
  IconKeyboard,
  IconMic,
  IconMicOff,
  IconStop,
  IconWifi,
  Tint,
  useToast,
} from '@mgd/ui';
import { useEffect, useRef, useState } from 'react';

interface Labels {
  readonly title: string;
  readonly round: string;
  readonly live: string;
  readonly paused: string;
  readonly reconnecting: string;
  readonly timer: string;
  readonly network: string;
  readonly subtitles: string;
  readonly on: string;
  readonly off: string;
  readonly exit: string;
  readonly exitDialogTitle: string;
  readonly exitDialogBody: string;
  readonly question: string;
  readonly questionHint: string;
  readonly candidateTranscript: string;
  readonly revisionHint: string;
  readonly revise: string;
  readonly revised: string;
  readonly frozen: string;
  readonly dynamicFollowup: string;
  readonly tools: string;
  readonly noToolsHint: string;
  readonly controls: string;
  readonly stopAvatar: string;
  readonly textAnswer: string;
  readonly submitAnswer: string;
  readonly cameraLabel: string;
  readonly micLabel: string;
  readonly turnProgress: string;
  readonly systemPaused: string;
  readonly systemPausedHint: string;
  readonly reconnectCountdown: string;
  readonly reconnectHint: string;
  readonly downgradeTitle: string;
  readonly downgradeBody: string;
  readonly downgradeAccept: string;
  readonly downgradeDecline: string;
  readonly downgradeDeclineHint: string;
  readonly authPaused: string;
  readonly authPausedHint: string;
  readonly reauthenticate: string;
}

type Overlay = 'none' | 'paused' | 'reconnect' | 'downgrade' | 'auth' | 'exit';

export function RoomView({
  locale,
  labels,
}: {
  readonly locale: 'zh-CN' | 'en-US';
  readonly labels: Labels;
}): React.ReactNode {
  const toast = useToast();
  const [elapsed, setElapsed] = useState(482);
  const [overlay, setOverlay] = useState<Overlay>('none');
  const [countdown, setCountdown] = useState(180);
  const [cameraOn, setCameraOn] = useState(true);
  const [micOn, setMicOn] = useState(true);
  const [subtitles, setSubtitles] = useState(true);
  const [answer, setAnswer] = useState('');
  const [frozen, setFrozen] = useState(false);
  const [status, setStatus] = useState<'live' | 'paused' | 'reconnecting' | 'text'>('live');
  const lastTick = useRef(Date.now());

  useEffect(() => {
    const id = window.setInterval(() => {
      if (status === 'live') {
        const now = Date.now();
        setElapsed((e) => e + Math.floor((now - lastTick.current) / 1000));
        lastTick.current = now;
      }
    }, 1000);
    return () => window.clearInterval(id);
  }, [status]);

  useEffect(() => {
    if (overlay !== 'reconnect') return;
    const id = window.setInterval(() => {
      setCountdown((c) => {
        if (c <= 1) {
          window.clearInterval(id);
          return 0;
        }
        return c - 1;
      });
    }, 1000);
    return () => window.clearInterval(id);
  }, [overlay]);

  const mm = String(Math.floor(elapsed / 60)).padStart(2, '0');
  const ss = String(elapsed % 60).padStart(2, '0');

  const startOverlay = (o: Overlay) => {
    setOverlay(o);
    if (o === 'paused') setStatus('paused');
    if (o === 'reconnect') {
      setStatus('reconnecting');
      setCountdown(180);
    }
    if (o === 'auth') setStatus('paused');
  };

  const closeOverlay = (next: Overlay = 'none') => {
    setOverlay(next);
    setStatus('live');
    lastTick.current = Date.now();
  };

  const acceptDowngrade = () => {
    setOverlay('none');
    setStatus('text');
    toast.push({ title: locale === 'zh-CN' ? '已切换文字面试' : 'Switched to text interview', tone: 'info' });
  };

  const declineDowngrade = () => {
    setOverlay('none');
    toast.push({ title: locale === 'zh-CN' ? '已结束：评估未完成（不是失败），额度将返还' : 'Ended: evaluation incomplete (not a failure). Credits will be refunded.', tone: 'success' });
    window.location.href = `/${locale}/projects/p-0001/rounds/2/result`;
  };

  const submit = () => {
    if (answer.trim() === '') return;
    setFrozen(true);
    toast.push({ title: labels.revised, tone: 'success' });
  };

  return (
    <div className="mgd-room">
      <div className="mgd-room__stage">
        {/* 顶栏 */}
        <header className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-neutral-100 bg-surface px-4 py-3 shadow-[var(--mgd-app-shadow-sm)]">
          <div className="flex items-center gap-3">
            <span className="grid size-9 place-items-center rounded-xl bg-[var(--mgd-app-brand-soft)] font-bold text-[var(--mgd-app-brand-ink)]">
              2
            </span>
            <div>
              <div className="text-sm font-semibold text-neutral-900">{labels.round.replace('{n}', '2')}</div>
              <div className="text-xs text-neutral-500">{labels.title}</div>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Tint tone={status === 'live' ? 'success' : status === 'text' ? 'info' : 'warning'}>
              {status === 'live' ? labels.live : status === 'text' ? labels.reconnecting : labels.paused}
            </Tint>
            <span className="inline-flex items-center gap-1.5 font-mono text-sm text-neutral-700">
              <IconClock size={14} />
              {mm}:{ss}
            </span>
            <span className="inline-flex items-center gap-1.5 text-sm text-neutral-600">
              <IconWifi size={15} className="text-success" />
              {labels.network}
            </span>
            <button
              type="button"
              onClick={() => setSubtitles((s) => !s)}
              className="mgd-target-min cursor-pointer rounded-lg border-0 bg-surface-muted px-3 text-sm font-medium text-neutral-700 hover:bg-neutral-100"
            >
              {labels.subtitles}: {subtitles ? labels.on : labels.off}
            </button>
            <Button variant="secondary" targetSize="min" onClick={() => setOverlay('exit')}>
              {labels.exit}
            </Button>
          </div>
        </header>

        <div className="grid min-h-0 grid-rows-[minmax(0,1fr)_auto] gap-4">
          {/* 左列：问题/字幕/工具 */}
          <div className="grid min-h-0 grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-4 max-md:grid-cols-1">
            <section aria-labelledby="room-question" className="flex min-h-0 flex-col rounded-2xl border border-neutral-100 bg-surface shadow-[var(--mgd-app-shadow-sm)]">
              <div className="flex items-center justify-between gap-2 border-b border-neutral-100 px-5 py-3">
                <h2 id="room-question" className="m-0 text-sm font-semibold text-neutral-900">
                  {labels.question}
                </h2>
                <span className="text-xs text-neutral-500">{labels.questionHint}</span>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto p-5">
                <div className="mb-4 rounded-xl bg-[var(--mgd-app-surface-muted)] p-4 text-[15px] leading-7 text-neutral-800">
                  请以你在「订单中心重构」项目中的经历为例，说明你如何评估容量风险并做出架构取舍。
                </div>
                {subtitles ? (
                  <div className="mb-4">
                    <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">Avatar</div>
                    <p className="mb-0 text-sm leading-6 text-neutral-700">这个项目的峰值 QPS 预估是多少？你当时做了哪些压测验证？</p>
                  </div>
                ) : null}
                <div>
                  <div className="mb-1 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-neutral-500">
                    {labels.candidateTranscript}
                    {frozen ? <Tint tone="info">{labels.frozen}</Tint> : <Tint tone="warning">{labels.revise}</Tint>}
                  </div>
                  <p className="mb-0 rounded-xl border border-neutral-100 bg-surface-muted/60 p-4 text-sm leading-6 text-neutral-800">
                    峰值 QPS 我们预估约 8k，通过全链路压测发现数据库连接池是瓶颈，随后引入了读写分离与本地缓存……
                  </p>
                  <p className="mt-2 mb-0 text-xs text-neutral-500">{labels.revisionHint}</p>
                </div>
              </div>
            </section>

            <section aria-labelledby="room-tools" className="flex min-h-0 flex-col rounded-2xl border border-neutral-100 bg-surface shadow-[var(--mgd-app-shadow-sm)]">
              <div className="flex items-center justify-between gap-2 border-b border-neutral-100 px-5 py-3">
                <h2 id="room-tools" className="m-0 text-sm font-semibold text-neutral-900">{labels.tools}</h2>
                <span className="text-xs text-neutral-500">{labels.controls}</span>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto p-5">
                <div className="mb-3 flex items-center gap-2 rounded-xl bg-[var(--mgd-app-brand-soft)] px-4 py-3 text-sm font-medium text-[var(--mgd-app-brand-ink)]">
                  <IconCode size={16} />
                  code_editor
                </div>
                <pre className="mgd-mono mb-0 overflow-x-auto rounded-xl bg-neutral-900 p-4 text-xs leading-5 text-neutral-100">
{`func Reserve(ctx context.Context, u *User) error {
    // 幂等键去重（NFR-006）
    key := "reserve:" + u.ID
    if idem.Exists(ctx, key) { return nil }
    return ledger.Append(ctx, key, u)
}`}
                </pre>
                <p className="mb-0 mt-3 text-xs text-neutral-500">{labels.noToolsHint}</p>
              </div>
            </section>
          </div>

          {/* 底部控制栏 */}
          <div className="flex flex-wrap items-center gap-2 rounded-2xl border border-neutral-100 bg-surface px-4 py-3 shadow-[var(--mgd-app-shadow-md)]">
            <Button variant="secondary" targetSize="min" aria-label={labels.micLabel} aria-pressed={micOn} onClick={() => setMicOn((v) => !v)}>
              {micOn ? <IconMic size={18} /> : <IconMicOff size={18} />}
            </Button>
            <Button variant="secondary" targetSize="min" aria-label={labels.cameraLabel} aria-pressed={cameraOn} onClick={() => setCameraOn((v) => !v)}>
              {cameraOn ? <IconCamera size={18} /> : <IconCameraOff size={18} />}
            </Button>
            <Button variant="danger" targetSize="min" onClick={() => toast.push({ title: labels.stopAvatar, tone: 'info' })}>
              <IconStop size={16} />
              {labels.stopAvatar}
            </Button>
            <div className="relative min-w-0 flex-1">
              <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-neutral-400">
                <IconKeyboard size={17} />
              </span>
              <input
                type="text"
                value={answer}
                onChange={(e) => setAnswer(e.target.value)}
                placeholder={labels.textAnswer}
                aria-label={labels.textAnswer}
                className="mgd-target-primary w-full rounded-xl border border-[var(--mgd-app-border-default)] bg-surface pl-10 pr-3 text-sm"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') submit();
                }}
              />
            </div>
            <Button variant="primary" onClick={submit} disabled={answer.trim() === ''} disabledReason={answer.trim() === '' ? labels.textAnswer : undefined}>
              <IconCheck size={16} />
              {labels.submitAnswer}
            </Button>
          </div>
        </div>
      </div>

      {/* 右列：数字人 + 候选人 + 进度 */}
      <aside className="flex min-h-0 flex-col gap-4">
        <div className="relative min-h-56 flex-1 overflow-hidden rounded-2xl bg-[linear-gradient(160deg,var(--mgd-app-brand-ink),var(--mgd-app-brand-from)_55%,var(--mgd-app-brand-to))] shadow-[var(--mgd-app-shadow-lg)]">
          <div className="absolute inset-0 grid place-items-center">
            <div className="text-center text-white/90">
              <div className="mx-auto mb-3 grid size-20 place-items-center rounded-full bg-white/10 text-3xl font-bold backdrop-blur">
                {locale === 'zh-CN' ? 'AI' : 'AI'}
              </div>
              <p className="mb-0 text-sm font-medium">MianGeDan Avatar · 合成演示</p>
              <p className="mb-0 mt-1 text-xs text-white/60">实时数字人音视频（始终开启）</p>
            </div>
          </div>
          <span className="absolute left-3 top-3 inline-flex items-center gap-1.5 rounded-full bg-black/30 px-2.5 py-1 text-xs text-white backdrop-blur">
            <span className="inline-block size-1.5 animate-pulse rounded-full bg-red-400" />
            REC
          </span>
        </div>
        <div className="relative min-h-32 overflow-hidden rounded-2xl border border-neutral-100 bg-neutral-900 shadow-[var(--mgd-app-shadow-md)]">
          {cameraOn ? (
            <div className="absolute inset-0 grid place-items-center text-white/60">
              <p className="mb-0 text-sm">候选人摄像头 · 合成画面</p>
            </div>
          ) : (
            <div className="absolute inset-0 grid place-items-center text-white/40">
              <IconCameraOff size={28} />
            </div>
          )}
          <span className="absolute left-3 top-3 rounded-full bg-black/30 px-2.5 py-1 text-xs text-white backdrop-blur">
            {labels.cameraLabel}: {cameraOn ? labels.on : labels.off}
          </span>
        </div>
        <div className="rounded-2xl border border-neutral-100 bg-surface p-4 shadow-[var(--mgd-app-shadow-sm)]">
          <div className="mb-2 flex items-center justify-between text-sm">
            <span className="font-semibold text-neutral-900">{labels.turnProgress}</span>
            <span className="font-mono text-xs text-neutral-500">1 / 3</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-neutral-100">
            <div className="h-full w-1/3 rounded-full bg-[linear-gradient(90deg,var(--mgd-app-brand-from),var(--mgd-app-brand-to))]" />
          </div>
          <div className="mt-3 flex items-start gap-2 rounded-lg bg-[color-mix(in_srgb,var(--mgd-color-warning)_8%,transparent)] px-3 py-2 text-xs text-warning-text">
            <IconAlert size={13} className="mt-0.5 shrink-0" />
            <span>{labels.revisionHint}</span>
          </div>
          <button
            type="button"
            className="mt-3 w-full cursor-pointer rounded-lg border border-dashed border-[var(--mgd-app-border-strong)] px-3 py-2 text-xs text-neutral-500 hover:border-warning hover:text-warning-text"
            onClick={() => startOverlay('paused')}
          >
            {locale === 'zh-CN' ? '演示：触发系统故障暂停' : 'Demo: trigger system pause'}
          </button>
          <div className="mt-2 grid grid-cols-2 gap-2">
            <button type="button" className="cursor-pointer rounded-lg border border-dashed border-[var(--mgd-app-border-strong)] px-2 py-2 text-xs text-neutral-500 hover:border-info hover:text-info" onClick={() => startOverlay('reconnect')}>
              {locale === 'zh-CN' ? '演示：断线重连' : 'Demo: reconnect'}
            </button>
            <button type="button" className="cursor-pointer rounded-lg border border-dashed border-[var(--mgd-app-border-strong)] px-2 py-2 text-xs text-neutral-500 hover:border-danger hover:text-danger" onClick={() => startOverlay('downgrade')}>
              {locale === 'zh-CN' ? '演示：降级询问' : 'Demo: downgrade'}
            </button>
          </div>
        </div>
      </aside>

      {/* SCR-09 覆盖层 */}
      <AlertDialog open={overlay === 'paused'} title={labels.systemPaused} description={labels.systemPausedHint} onClose={() => closeOverlay()}>
        <Button variant="secondary" onClick={() => closeOverlay()}>{locale === 'zh-CN' ? '等待自动恢复' : 'Wait for recovery'}</Button>
        <Button variant="primary" onClick={() => closeOverlay()}>{locale === 'zh-CN' ? '我已恢复' : 'I am back'}</Button>
      </AlertDialog>

      <AlertDialog open={overlay === 'reconnect'} title={labels.reconnecting} description={`${labels.reconnectCountdown.replace('{seconds}', String(countdown))}。${labels.reconnectHint}`} onClose={() => closeOverlay()}>
        <Button variant="primary" onClick={() => closeOverlay()}>{locale === 'zh-CN' ? '重新连接' : 'Reconnect'}</Button>
      </AlertDialog>

      <AlertDialog open={overlay === 'downgrade'} title={labels.downgradeTitle} description={`${labels.downgradeBody} ${labels.downgradeDeclineHint}`} onClose={() => setOverlay('none')}>
        <Button variant="secondary" onClick={declineDowngrade}>{labels.downgradeDecline}</Button>
        <Button variant="primary" onClick={acceptDowngrade}>{labels.downgradeAccept}</Button>
      </AlertDialog>

      <AlertDialog open={overlay === 'auth'} title={labels.authPaused} description={labels.authPausedHint} onClose={() => closeOverlay()}>
        <Button variant="primary" onClick={() => closeOverlay()}>{labels.reauthenticate}</Button>
      </AlertDialog>

      <AlertDialog open={overlay === 'exit'} title={labels.exitDialogTitle} description={labels.exitDialogBody} onClose={() => setOverlay('none')}>
        <Button variant="secondary" onClick={() => setOverlay('none')}>{locale === 'zh-CN' ? '取消' : 'Cancel'}</Button>
        <Button variant="danger" onClick={() => { setOverlay('none'); toast.push({ title: locale === 'zh-CN' ? '本轮已标记评估未完成' : 'Round marked evaluation incomplete', tone: 'info' }); window.location.href = `/${locale}/projects/p-0001/rounds/2/result`; }}>
          {labels.exit}
        </Button>
      </AlertDialog>
    </div>
  );
}
