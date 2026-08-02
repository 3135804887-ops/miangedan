'use client';

import { Button, Field } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useState, type ChangeEvent, type ReactNode } from 'react';

import { PageStateBoundary, type PreviewState } from '../../components/page-state-boundary.tsx';
import { RouteShell } from '../../components/route-shell.tsx';

const MAX_FILE_BYTES = 10 * 1024 * 1024;
const ALLOWED_FILE_EXTENSIONS = ['.pdf', '.doc', '.docx'] as const;

export function CreateProjectExperience({ mode }: { readonly mode: PreviewState }): ReactNode {
  const t = useTranslations('batch1');
  const [jd, setJd] = useState('');
  const [fileName, setFileName] = useState<string>();
  const [fileError, setFileError] = useState<string>();
  const [synthetic, setSynthetic] = useState(false);
  const [draftSaved, setDraftSaved] = useState(false);

  function selectFile(event: ChangeEvent<HTMLInputElement>): void {
    const file = event.currentTarget.files?.[0];
    if (file === undefined) return;
    const lowerName = file.name.toLowerCase();
    const allowed = ALLOWED_FILE_EXTENSIONS.some((extension) => lowerName.endsWith(extension));
    if (!allowed || file.size > MAX_FILE_BYTES) {
      setFileName(undefined);
      setFileError(t('create.rejected'));
      return;
    }
    setFileError(undefined);
    setFileName(file.name);
  }

  return (
    <RouteShell scrId="SCR-04" title={t('create.title')} notice={t('create.lead')}>
      <PageStateBoundary
        mode={mode}
        regionLabel="SCR-04-create"
        emptyReason={t('create.empty')}
        forbiddenPermission={t('create.forbidden')}
        recoveryPoint={t('create.recovering')}
      >
        <div className="mgd-page-toolbar">
          <p>{t('shared.mockNotice')}</p>
          <Button
            controlId="project-fill-sample"
            onClick={() => {
              setSynthetic(true);
              setJd(`${t('shared.synthetic')}\n\n${t('create.lead')}`);
              setFileName('synthetic-resume.docx');
              setFileError(undefined);
            }}
          >
            {t('create.sample')}
          </Button>
        </div>

        {synthetic ? <p className="mgd-synthetic-label">{t('create.sampleMark')}</p> : null}
        <div className="mgd-input-columns">
          <section className="mgd-upload-panel" aria-labelledby="resume-upload-title">
            <p className="mgd-kicker">01 / FILE</p>
            <h2 id="resume-upload-title">{t('create.resume')}</h2>
            <p>{t('create.uploadHint')}</p>
            <label className="mgd-file-picker">
              <span>{t('create.replace')}</span>
              <input
                type="file"
                accept=".pdf,.doc,.docx,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
                onChange={selectFile}
              />
            </label>
            {fileName === undefined ? null : (
              <div className="mgd-file-status" role="status">
                <strong>{fileName}</strong>
                <span>{t('create.processing')}</span>
                <Button controlId="project-remove-file" variant="danger" onClick={() => setFileName(undefined)}>
                  {t('create.remove')}
                </Button>
              </div>
            )}
            {fileError === undefined ? null : <p className="mgd-inline-error" role="alert">⚠ {fileError}</p>}
          </section>

          <section className="mgd-jd-panel" aria-labelledby="jd-input-title">
            <p className="mgd-kicker">02 / TEXT</p>
            <h2 id="jd-input-title">{t('create.jd')}</h2>
            <Field fieldId="project-jd" label={t('create.jd')} description={t('create.lead')}>
              <textarea
                rows={14}
                value={jd}
                placeholder={t('create.jdPlaceholder')}
                onChange={(event) => {
                  setJd(event.currentTarget.value);
                  setDraftSaved(false);
                }}
              />
            </Field>
          </section>
        </div>

        <div className="mgd-action-row mgd-action-row--end">
          <Button controlId="project-save-draft" onClick={() => setDraftSaved(true)}>
            {t('create.saveDraft')}
          </Button>
          <Button
            controlId="project-start-parse"
            variant="primary"
            disabled={jd.trim().length === 0}
            disabledReason={jd.trim().length === 0 ? t('create.jd') : undefined}
          >
            {t('create.parse')}
          </Button>
        </div>
        {draftSaved ? <p className="mgd-inline-notice" role="status">{t('create.draftRestored')}</p> : null}
      </PageStateBoundary>
    </RouteShell>
  );
}
