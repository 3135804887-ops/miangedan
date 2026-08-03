'use client';

import {
  Button,
  Card,
  CardBody,
  IconAlert,
  IconCheck,
  IconEdit,
  IconPlus,
  IconTrash,
  PageHeader,
  Tabs,
  Tint,
  useToast,
} from '@mgd/ui';
import { useState } from 'react';

interface Labels {
  readonly kicker: string;
  readonly title: string;
  readonly desc: string;
  readonly tabResume: string;
  readonly tabJob: string;
  readonly lowConfidence: string;
  readonly excludedFields: string;
  readonly excludedList: string;
  readonly addField: string;
  readonly removeField: string;
  readonly confirmMaterials: string;
  readonly confirmHint: string;
  readonly missingImpactTitle: string;
  readonly missingImpactDesc: string;
  readonly missingJdOnly: string;
  readonly agreeDegraded: string;
  readonly confirm: string;
}

interface ProfileField {
  readonly key: string;
  readonly label: string;
  readonly value: string;
  readonly lowConfidence?: boolean;
}

const RESUME_FIELDS: readonly ProfileField[] = [
  { key: 'name', label: '姓名', value: '合成候选人' },
  { key: 'experience', label: '工作经历', value: '5 年后端开发经验，最近任职于合成科技（Go/Python）', lowConfidence: true },
  { key: 'education', label: '教育背景', value: '计算机科学学士' },
  { key: 'skills', label: '技能', value: 'Go、PostgreSQL、Redis、K8s、分布式系统' },
];

const JOB_FIELDS: readonly ProfileField[] = [
  { key: 'title', label: '岗位名称', value: '后端工程师' },
  { key: 'responsibilities', label: '岗位职责', value: '核心交易服务架构与开发、数据库建模、线上故障排查', lowConfidence: true },
  { key: 'requirements', label: '任职要求', value: '3 年以上服务端经验、分布式系统基础、消息队列与对象存储' },
];

export function ReviewView({
  locale,
  labels,
}: {
  readonly locale: 'zh-CN' | 'en-US';
  readonly labels: Labels;
}): React.ReactNode {
  const toast = useToast();
  const [resume, setResume] = useState<readonly ProfileField[]>(RESUME_FIELDS);
  const [job, setJob] = useState<readonly ProfileField[]>(JOB_FIELDS);
  const [agree, setAgree] = useState(false);

  const confirm = () => {
    toast.push({ title: locale === 'zh-CN' ? '材料已确认，计划生成中' : 'Materials confirmed. Generating plan…', tone: 'success' });
    window.location.href = `/${locale}/projects/p-0003/plan`;
  };

  const renderFields = (
    fields: readonly ProfileField[],
    setFields: (f: readonly ProfileField[]) => void,
  ) => (
    <div className="space-y-3">
      {fields.map((f) => (
        <div key={f.key} className={`flex items-start gap-3 rounded-xl border p-4 ${f.lowConfidence ? 'border-warning/40 bg-[color-mix(in_srgb,var(--mgd-color-warning)_8%,transparent)]' : 'border-neutral-100 bg-[var(--mgd-app-surface-muted)]'}`}>
          <div className="min-w-0 flex-1">
            <div className="mb-1 flex items-center gap-2">
              <span className="text-sm font-semibold text-neutral-800">{f.label}</span>
              {f.lowConfidence ? <Tint tone="warning">{labels.lowConfidence}</Tint> : null}
            </div>
            <p className="mb-0 text-sm text-neutral-700">{f.value}</p>
          </div>
          <Button variant="secondary" targetSize="min" aria-label={labels.addField} onClick={() => toast.push({ title: labels.addField, tone: 'info' })}>
            <IconEdit size={15} />
          </Button>
          <Button variant="secondary" targetSize="min" aria-label={labels.removeField} onClick={() => setFields(fields.filter((x) => x.key !== f.key))}>
            <IconTrash size={15} />
          </Button>
        </div>
      ))}
      <Button variant="secondary" onClick={() => toast.push({ title: labels.addField, tone: 'info' })}>
        <IconPlus size={16} />
        {labels.addField}
      </Button>
    </div>
  );

  const resumeTab = (
    <div className="space-y-5">
      {renderFields(resume, setResume)}
      <div className="rounded-xl border border-neutral-100 bg-[var(--mgd-app-surface-muted)] p-4">
        <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-neutral-800">
          <IconAlert size={16} className="text-warning" />
          {labels.excludedFields}
        </div>
        <p className="mb-0 text-sm text-neutral-600">{labels.excludedList}</p>
      </div>
    </div>
  );

  const jobTab = (
    <div className="space-y-5">
      {renderFields(job, setJob)}
      <div className="rounded-xl border border-warning/30 bg-[color-mix(in_srgb,var(--mgd-color-warning)_8%,transparent)] p-4">
        <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-neutral-800">
          <IconAlert size={16} className="text-warning" />
          {labels.missingImpactTitle}
        </div>
        <p className="mb-3 text-sm text-neutral-600">{labels.missingImpactDesc}</p>
        <p className="mb-3 text-sm text-neutral-700">{labels.missingJdOnly}</p>
        <label className="flex items-start gap-2 text-sm">
          <input type="checkbox" checked={agree} onChange={(e) => setAgree(e.target.checked)} className="mt-1 size-4 accent-[var(--mgd-app-brand-from)]" />
          <span>{labels.agreeDegraded}</span>
        </label>
      </div>
    </div>
  );

  return (
    <>
      <PageHeader kicker={labels.kicker} title={labels.title} description={labels.desc} />
      <Card>
        <CardBody>
          <Tabs
            initialId="resume"
            items={[
              { id: 'resume', label: labels.tabResume, content: resumeTab },
              { id: 'job', label: labels.tabJob, content: jobTab },
            ]}
          />
        </CardBody>
      </Card>
      <div className="mt-6 flex flex-wrap items-center justify-between gap-3">
        <p className="mb-0 text-sm text-neutral-600">{labels.confirmHint}</p>
        <Button variant="primary" onClick={confirm} disabled={!agree} disabledReason={agree ? undefined : labels.agreeDegraded}>
          <IconCheck size={17} />
          {labels.confirmMaterials}
        </Button>
      </div>
    </>
  );
}
