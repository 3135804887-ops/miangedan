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

import { apiFetch } from '../lib/api-fetch.ts';

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
  projectId,
}: {
  readonly locale: 'zh-CN' | 'en-US';
  readonly labels: Labels;
  readonly projectId: string;
}): React.ReactNode {
  const toast = useToast();
  const [states, setStates] = useState<Readonly<Record<string, CheckState>>>(
    Object.fromEntries(CHECKS.map((c) => [c, 'checking' as CheckState])),
  );
  const [cameraOff, setCameraOff] = useState(false);
  const [micOff, setMicOff] = useState(false);
  const [accom, setAccom] = useState<ReadonlySet<string>>(new Set(['reduced_motion']));
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [apiUnavailable, setApiUnavailable] = useState(false);

  const runChecks = () => {
    setStates(Object.fromEntries(CHECKS.map((c) => [c, 'checking' as CheckState])));
    CHECKS.forEach((c, i) => {
      window.setTimeout(() => {
        setStates((prev) => ({ ...prev, [c]: 'passed' }));
      }, 600 + i * 500);
    });
  };

  useEffect(() => {
    let alive = true;
    void (async () => {
      const created = await apiFetch<{ session_id: string }>(
        '/v1/projects/{projectId}/rounds/{sequence}/session',
        {
          method: 'post',
          idempotencyKey: `precheck-session-${projectId}-1`,
          pathParams: { projectId, sequence: 1 },
        },
      );
      if (!alive) return;
      if (!created.ok) {
        setApiUnavailable(true);
        runChecks();
        return;
      }
      const sid = created.data.session_id;
      setSessionId(sid);
      const pre = await apiFetch<{
        precheck: {
          frozen: boolean;
          input_modes: readonly string[];
          accommodations: readonly string[];
          device_report: { camera_ok: boolean; mic_ok: boolean; network_rated: string };
        };
      }>('/v1/sessions/{sessionId}/precheck', { method: 'get', pathParams: { sessionId: sid } });
      if (!alive) return;
      if (pre.ok) {
        const d = pre.data.precheck;
        setStates({
          camera: d.device_report.camera_ok ? 'passed' : 'failed',
          mic: d.device_report.mic_ok ? 'passed' : 'failed',
          network: d.device_report.network_rated === 'good' ? 'passed' : 'failed',
          speaker: 'passed',
          avatar: 'passed',
        });
        setAccom(new Set(d.accommodations));
        setCameraOff(!d.input_modes.includes('camera'));
        setMicOff(!d.input_modes.includes('mic'));
      } else {
        setApiUnavailable(true);
        runChecks();
      }
    })();
    return () => {
      alive = false;
    };
  }, [projectId]);

  const toggleAccom = (key: string) => {
    setAccom((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const freeze = async () => {
    if (sessionId !== null) {
      const res = await apiFetch<{ frozen: boolean }>(
        '/v1/sessions/{sessionId}/precheck/freeze',
        {
          method: 'post',
          idempotencyKey: `precheck-freeze-${sessionId}-${Date.now()}`,
          pathParams: { sessionId },
        },
      );
      if (!res.ok) {
        toast.push({ title: locale === 'zh-CN' ? '冻结失败，请重试' : 'Freeze failed, please retry', tone: 'danger' });
        return;
      }
      window.location.href = `/${locale}/sessions/${sessionId}`;
      return;
    }
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

      {apiUnavailable ? (
        <p className="mb-4 rounded-xl bg-warning/10 px-4 py-3 text-sm text-warning-text" role="status">
          {locale === 'zh-CN' ? '会前检查服务暂未接入（占位），当前为合成检测结果。' : 'Pre-session service placeholder — showing synthetic checks.'}
        </p>
      ) : null}

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
