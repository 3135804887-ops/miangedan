'use client';

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  IconCamera,
  IconCheck,
  IconClock,
  IconMic,
  IconRefresh,
  IconSparkle,
  IconWifi,
  PageHeader,
  Switch,
  Tint,
  useToast,
} from '@mgd/ui';
import { useEffect, useState } from 'react';

interface Labels {
  readonly kicker: string;
  readonly title: string;
  readonly desc: string;
  readonly camera: string;
  readonly mic: string;
  readonly network: string;
  readonly speaker: string;
  readonly avatar: string;
  readonly checking: string;
  readonly passed: string;
  readonly failed: string;
  readonly retryItem: string;
  readonly cameraOff: string;
  readonly micOff: string;
  readonly networkPoor: string;
  readonly accommodations: string;
  readonly accommodationsHint: string;
  readonly freeze: string;
  readonly entitlement: string;
  readonly entitlementHint: string;
  readonly entitlementInsufficient: string;
}

type CheckState = 'checking' | 'passed' | 'failed';

const CHECKS = ['camera', 'mic', 'network', 'speaker', 'avatar'] as const;

export function PrecheckView({
  locale,
  labels,
}: {
  readonly locale: 'zh-CN' | 'en-US';
  readonly labels: Labels;
}): React.ReactNode {
  const toast = useToast();
  const [states, setStates] = useState<Readonly<Record<string, CheckState>>>(
    Object.fromEntries(CHECKS.map((c) => [c, 'checking' as CheckState])),
  );
  const [cameraOff, setCameraOff] = useState(false);
  const [micOff, setMicOff] = useState(false);
  const [accom, setAccom] = useState<ReadonlySet<string>>(new Set(['reduced_motion']));

  const runChecks = () => {
    setStates(Object.fromEntries(CHECKS.map((c) => [c, 'checking' as CheckState])));
    CHECKS.forEach((c, i) => {
      window.setTimeout(() => {
        setStates((prev) => ({ ...prev, [c]: 'passed' }));
      }, 600 + i * 500);
    });
  };

  useEffect(() => {
    runChecks();
  }, []);

  const toggleAccom = (key: string) => {
    setAccom((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const freeze = () => {
    toast.push({ title: locale === 'zh-CN' ? '会前配置已冻结，正在连接数字人…' : 'Pre-session config frozen. Connecting the avatar…', tone: 'success' });
    window.location.href = `/${locale}/sessions/s-0001`;
  };

  const icons: Readonly<Record<string, React.ReactNode>> = {
    camera: <IconCamera size={19} />,
    mic: <IconMic size={19} />,
    network: <IconWifi size={19} />,
    speaker: <IconSparkle size={19} />,
    avatar: <IconSparkle size={19} />,
  };

  const nameOf = (key: string) =>
    ({ camera: labels.camera, mic: labels.mic, network: labels.network, speaker: labels.speaker, avatar: labels.avatar })[key] ?? key;

  const allPassed = CHECKS.every((c) => states[c] === 'passed');

  return (
    <>
      <PageHeader kicker={labels.kicker} title={labels.title} description={labels.desc} />

      <div className="mgd-grid mgd-grid--sidebar">
        <div className="space-y-4">
          <Card>
            <CardHeader
              title={locale === 'zh-CN' ? '设备与网络检测' : 'Device and network check'}
              actions={
                <Button variant="secondary" targetSize="min" onClick={runChecks}>
                  <IconRefresh size={15} />
                  {labels.retryItem}
                </Button>
              }
            />
            <CardBody className="space-y-3">
              {CHECKS.map((c) => (
                <div key={c} className="flex items-center gap-4 rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                  <span className="grid size-10 place-items-center rounded-xl bg-surface text-neutral-700 shadow-[var(--mgd-app-shadow-sm)]">
                    {icons[c]}
                  </span>
                  <span className="flex-1 font-medium text-neutral-800">{nameOf(c)}</span>
                  {states[c] === 'checking' ? (
                    <Tint tone="info">
                      <span className="mgd-skeleton inline-block size-2 rounded-full" aria-hidden="true" />
                      {labels.checking}
                    </Tint>
                  ) : states[c] === 'passed' ? (
                    <Tint tone="success">
                      <IconCheck size={13} />
                      {labels.passed}
                    </Tint>
                  ) : (
                    <Tint tone="danger">{labels.failed}</Tint>
                  )}
                </div>
              ))}
            </CardBody>
          </Card>

          <Card>
            <CardHeader title={labels.accommodations} description={labels.accommodationsHint} />
            <CardBody>
              <div className="grid gap-3 sm:grid-cols-2">
                {[
                  ['text_only', '纯文字模式'],
                  ['slower_avatar_speech', '数字人语速放慢'],
                  ['repeat_questions', '问题可重复'],
                  ['extended_time', '延长作答时间'],
                  ['reduced_motion', '减少动效'],
                  ['tool_keyboard_alternative', '工具键盘替代'],
                ].map(([key = '', zhLabel = '']) => (
                  <label key={key} className="flex cursor-pointer items-center justify-between gap-3 rounded-xl border border-neutral-100 px-4 py-3">
                    <span className="text-sm text-neutral-700">{locale === 'zh-CN' ? zhLabel : key}</span>
                    <input
                      type="checkbox"
                      checked={accom.has(key)}
                      onChange={() => toggleAccom(key)}
                      className="size-4 accent-[var(--mgd-app-brand-from)]"
                    />
                  </label>
                ))}
              </div>
            </CardBody>
          </Card>
        </div>

        <div className="space-y-4">
          <Card className="mgd-card--brand">
            <CardBody>
              <div className="mb-3 text-sm font-semibold text-neutral-900">{locale === 'zh-CN' ? '输入开关' : 'Input switches'}</div>
              <div className="space-y-4">
                <label className="flex items-center justify-between gap-3">
                  <span className="flex items-center gap-2 text-sm text-neutral-700">
                    <IconCamera size={16} />
                    {labels.cameraOff}
                  </span>
                  <Switch checked={cameraOff} onCheckedChange={setCameraOff} label={labels.camera} />
                </label>
                <label className="flex items-center justify-between gap-3">
                  <span className="flex items-center gap-2 text-sm text-neutral-700">
                    <IconMic size={16} />
                    {labels.micOff}
                  </span>
                  <Switch checked={micOff} onCheckedChange={setMicOff} label={labels.mic} />
                </label>
              </div>
              <p className="mb-0 mt-4 text-xs text-neutral-500">{labels.networkPoor}</p>
            </CardBody>
          </Card>

          <Card>
            <CardBody>
              <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-neutral-900">
                <IconClock size={16} />
                {labels.entitlement}
              </div>
              <p className="mb-3 text-sm text-neutral-600">{labels.entitlementHint}</p>
              <div className="mb-1 flex justify-between text-sm">
                <span className="text-neutral-600">{locale === 'zh-CN' ? '本轮预计' : 'Estimated this round'}</span>
                <span className="font-mono font-semibold text-neutral-900">30 min</span>
              </div>
              <div className="mb-4 flex justify-between text-sm">
                <span className="text-neutral-600">{locale === 'zh-CN' ? '账户余额' : 'Balance'}</span>
                <span className="font-mono font-semibold text-success-text">46 min</span>
              </div>
              <Button variant="primary" className="w-full" disabled={!allPassed} disabledReason={allPassed ? undefined : labels.checking} onClick={freeze}>
                {labels.freeze}
              </Button>
            </CardBody>
          </Card>
        </div>
      </div>
    </>
  );
}
