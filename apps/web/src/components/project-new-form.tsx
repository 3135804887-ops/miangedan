'use client';

import {
  Button,
  Card,
  CardBody,
  CardHeader,
  IconCheck,
  IconFile,
  IconTrash,
  IconUpload,
  PageHeader,
  useToast,
} from '@mgd/ui';
import { useEffect, useRef, useState } from 'react';

import { apiFetch } from '../lib/api-fetch.ts';
import { SyntheticNote } from './synthetic-note.tsx';

interface Labels {
  readonly kicker: string;
  readonly title: string;
  readonly desc: string;
  readonly resumeSection: string;
  readonly resumeHint: string;
  readonly dropHint: string;
  readonly browse: string;
  readonly replace: string;
  readonly remove: string;
  readonly scanning: string;
  readonly scanRejected: string;
  readonly jdSection: string;
  readonly jdHint: string;
  readonly jdPlaceholder: string;
  readonly jdCharCount: string;
  readonly sampleFill: string;
  readonly sampleBadge: string;
  readonly draftSaved: string;
  readonly continue: string;
  readonly noResumeOrJd: string;
  readonly resumeUploading: string;
  readonly jdTooShort: string;
}

const SAMPLE_JD = `职位：后端工程师（Go）

岗位职责：
1. 负责核心交易服务的架构设计与开发（Go/Python）；
2. 参与数据库建模与高可用方案设计（PostgreSQL/Redis）；
3. 主导线上故障排查与容量评估，保障 SLO 99.95%；
4. 与算法、前端团队协作完成跨团队项目交付。

任职要求：
- 3 年以上服务端开发经验，熟悉分布式系统基础；
- 熟悉至少一种消息队列与对象存储；
- 具备良好的系统设计文档能力；
- 有 K8s 或云原生实践经验者优先。`;

export function ProjectNewForm({
  locale,
  labels,
}: {
  readonly locale: 'zh-CN' | 'en-US';
  readonly labels: Labels;
}): React.ReactNode {
  const toast = useToast();
  const inputRef = useRef<HTMLInputElement>(null);
  const [fileName, setFileName] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);
  const [scanError, setScanError] = useState<string | null>(null);
  const [jd, setJd] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (fileName === null && jd === '') return;
    const id = window.setTimeout(() => toast.push({ title: labels.draftSaved, tone: 'success' }), 800);
    return () => window.clearTimeout(id);
  }, [fileName, jd, labels.draftSaved, toast]);

  const pickFile = (file: File | undefined) => {
    if (file === undefined) return;
    setScanError(null);
    setFileName(file.name);
    setScanning(true);
    // 合成扫描：1.2s 后接受；>10MB 拒绝。
    window.setTimeout(() => {
      setScanning(false);
      if (file.size > 10 * 1024 * 1024) {
        setScanError(labels.scanRejected.replace('{reason}', locale === 'zh-CN' ? '文件超过 10MB' : 'file exceeds 10MB'));
        setFileName(null);
      }
    }, 1200);
  };

  const fillSample = () => {
    setJd(SAMPLE_JD);
    setFileName('synthetic-resume.pdf');
    toast.push({ title: labels.sampleFill, tone: 'info' });
  };

  const submit = async () => {
    setError(null);
    if (fileName === null && jd.trim().length === 0) {
      setError(labels.noResumeOrJd);
      return;
    }
    if (scanning) {
      setError(labels.resumeUploading);
      return;
    }
    if (jd.trim().length > 0 && jd.trim().length < 50) {
      setError(labels.jdTooShort);
      return;
    }
    const res = await apiFetch<{ project: { project_id: string } }>('/v1/projects', {
      method: 'post',
      idempotencyKey: `create-project-${Date.now()}`,
      body: {
        interview_language: locale === 'zh-CN' ? 'zh-CN' : 'en-US',
        degraded_mode: fileName !== null ? 'resume_only' : 'jd_only',
      },
    });
    if (res.ok) {
      toast.push({ title: locale === 'zh-CN' ? '项目已创建，进入解析与校对' : 'Project created. Moving to parsing and review', tone: 'success' });
      window.location.href = `/${locale}/projects/${res.data.project.project_id}/review`;
    } else {
      setError(locale === 'zh-CN' ? '创建项目失败，请重试。' : 'Failed to create project. Please retry.');
    }
  };

  return (
    <>
      <PageHeader
        kicker={labels.kicker}
        title={labels.title}
        description={labels.desc}
        actions={<SyntheticNote />}
      />

      <div className="mgd-grid mgd-grid--2">
        <Card>
          <CardHeader title={labels.resumeSection} description={labels.resumeHint} />
          <CardBody>
            <input
              ref={inputRef}
              type="file"
              accept=".pdf,.doc,.docx"
              className="sr-only"
              aria-label={labels.browse}
              onChange={(e) => pickFile(e.target.files?.[0])}
            />
            {fileName === null ? (
              <button
                type="button"
                onClick={() => inputRef.current?.click()}
                className="grid min-h-44 w-full cursor-pointer place-items-center rounded-2xl border-2 border-dashed border-[var(--mgd-app-border-strong)] bg-[var(--mgd-app-surface-muted)] p-6 text-center transition hover:border-primary hover:bg-[var(--mgd-app-brand-soft)]"
              >
                <span className="flex flex-col items-center gap-3">
                  <span className="grid size-12 place-items-center rounded-2xl bg-surface text-primary shadow-[var(--mgd-app-shadow-sm)]">
                    <IconUpload size={22} />
                  </span>
                  <span className="font-semibold text-neutral-800">{labels.dropHint}</span>
                  <span className="inline-flex items-center gap-2 rounded-lg bg-surface px-4 py-2 text-sm font-medium text-primary shadow-[var(--mgd-app-shadow-sm)]">
                    <IconFile size={16} />
                    {labels.browse}
                  </span>
                </span>
              </button>
            ) : (
              <div className="flex items-center gap-4 rounded-2xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
                <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-[var(--mgd-app-brand-soft)] text-[var(--mgd-app-brand-ink)]">
                  <IconFile size={20} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="mb-0.5 truncate font-medium text-neutral-900">{fileName}</p>
                  <p className="mb-0 text-sm text-neutral-600">
                    {scanning ? labels.scanning : locale === 'zh-CN' ? '扫描通过' : 'Scan passed'}
                  </p>
                </div>
                <Button variant="secondary" targetSize="min" onClick={() => inputRef.current?.click()}>
                  {labels.replace}
                </Button>
                <Button variant="secondary" targetSize="min" aria-label={labels.remove} onClick={() => setFileName(null)}>
                  <IconTrash size={16} />
                </Button>
              </div>
            )}
            {scanError !== null ? (
              <p role="alert" className="mt-3 rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">
                {scanError}
              </p>
            ) : null}
          </CardBody>
        </Card>

        <Card>
          <CardHeader
            title={labels.jdSection}
            description={labels.jdHint}
            actions={
              <button type="button" className="cursor-pointer text-sm font-medium text-primary hover:underline" onClick={fillSample}>
                {labels.sampleFill}
              </button>
            }
          />
          <CardBody>
            <div className="relative">
              <textarea
                value={jd}
                onChange={(e) => setJd(e.target.value)}
                placeholder={labels.jdPlaceholder}
                rows={12}
                className="w-full resize-y rounded-2xl border border-[var(--mgd-app-border-default)] bg-surface p-4 text-sm leading-6 focus:border-primary"
              />
              <span className="absolute bottom-3 right-3 rounded-md bg-surface/90 px-2 py-0.5 font-mono text-xs text-neutral-500">
                {labels.jdCharCount.replace('{count}', String(jd.length))}
              </span>
            </div>
          </CardBody>
        </Card>
      </div>

      {error !== null ? (
        <p role="alert" className="mt-4 rounded-lg bg-danger/10 px-4 py-3 text-sm text-danger">
          {error}
        </p>
      ) : null}

      <div className="mt-6 flex items-center justify-end gap-3">
        <Button variant="primary" onClick={submit} disabled={scanning} disabledReason={scanning ? labels.resumeUploading : undefined} loading={scanning} busyLabel={labels.scanning}>
          <IconCheck size={17} />
          {labels.continue}
        </Button>
      </div>
    </>
  );
}
