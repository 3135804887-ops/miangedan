'use client';

import { PROJECT_STATUSES, type ProjectStatus } from '@mgd/domain-states';
import { Button, Field } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { PageStateBoundary, type PreviewState } from '../../components/page-state-boundary.tsx';
import { RouteShell } from '../../components/route-shell.tsx';

export function ReviewExperience({
  mode,
  projectStatus,
}: {
  readonly mode: PreviewState;
  readonly projectStatus: ProjectStatus;
}): ReactNode {
  const t = useTranslations('batch1');
  const [experience, setExperience] = useState('Synthetic: led a cross-functional launch review.');
  const [skills, setSkills] = useState(['SQL', 'Experiment design']);
  const [missingAccepted, setMissingAccepted] = useState(false);
  const [materialsConfirmed, setMaterialsConfirmed] = useState(false);

  let content: ReactNode;
  if (projectStatus === PROJECT_STATUSES[1]) {
    content = (
      <section className="mgd-progress-panel" aria-busy="true" aria-labelledby="review-parsing-title">
        <p className="mgd-kicker">P95 ≤ 60s</p>
        <h2 id="review-parsing-title">{t('review.parsing')}</h2>
        <p>{t('review.parsingHint')}</p>
        <progress max="3" value="2">2 / 3</progress>
        <ol className="mgd-progress-steps">
          <li>✓ {t('create.resume')}</li>
          <li>● {t('review.fieldExperience')}</li>
          <li>○ {t('review.lowConfidence')}</li>
        </ol>
      </section>
    );
  } else if (projectStatus === PROJECT_STATUSES[3]) {
    content = (
      <section className="mgd-failure-panel" aria-labelledby="review-failed-title">
        <p className="mgd-kicker">PARSE / RETRYABLE</p>
        <h2 id="review-failed-title">{t('review.failed')}</h2>
        <dl className="mgd-retention-list">
          <div><dt>{t('shared.notScored')}</dt><dd>{t('review.failed')}</dd></div>
          <div><dt>{t('create.jd')}</dt><dd>{t('create.draftRestored')}</dd></div>
        </dl>
        <div className="mgd-action-row">
          <Button controlId="review-retry-step" variant="primary">{t('review.retryStep')}</Button>
          <Button controlId="review-open-manual">{t('review.manual')}</Button>
        </div>
      </section>
    );
  } else {
    const canGenerate = missingAccepted && materialsConfirmed;
    content = (
      <div className="mgd-review-layout">
        <div>
          <section className="mgd-section" aria-labelledby="review-fields-title">
            <p className="mgd-kicker">MATERIAL / REVIEW</p>
            <h2 id="review-fields-title">{t('review.reviewTitle')}</h2>
            <div className="mgd-review-field mgd-review-field--attention">
              <span className="mgd-warning-label">⚠ {t('review.lowConfidence')}</span>
              <Field fieldId="review-experience" label={t('review.fieldExperience')}>
                <textarea rows={4} value={experience} onChange={(event) => setExperience(event.currentTarget.value)} />
              </Field>
              <div className="mgd-action-row">
                <Button controlId="review-edit-experience">{t('review.edit')}</Button>
                <Button controlId="review-remove-experience" variant="danger" onClick={() => setExperience('')}>
                  {t('review.remove')}
                </Button>
              </div>
            </div>
            <div className="mgd-review-field">
              <h3>{t('review.fieldSkill')}</h3>
              <ul>{skills.map((skill) => <li key={skill}>{skill}</li>)}</ul>
              <Button controlId="review-add-skill" onClick={() => setSkills((current) => [...current, `Synthetic skill ${current.length + 1}`])}>
                {t('review.add')}
              </Button>
            </div>
          </section>

          <section className="mgd-sensitive-panel" aria-labelledby="review-sensitive-title">
            <h2 id="review-sensitive-title">{t('review.sensitiveTitle')}</h2>
            <p>{t('review.sensitiveHint')}</p>
            <p className="mgd-category-list">{t('review.categories')}</p>
          </section>
        </div>

        <aside className="mgd-consent-panel" aria-labelledby="review-missing-title">
          <p className="mgd-kicker">CONSENT / REQUIRED</p>
          <h2 id="review-missing-title">{t('review.missingTitle')}</h2>
          <p>{t('review.missingBody')}</p>
          <label>
            <input type="checkbox" checked={missingAccepted} onChange={(event) => setMissingAccepted(event.currentTarget.checked)} />
            {t('review.missingConsent')}
          </label>
          <label>
            <input type="checkbox" checked={materialsConfirmed} onChange={(event) => setMaterialsConfirmed(event.currentTarget.checked)} />
            {t('review.materialsConfirmed')}
          </label>
          <Button
            controlId="review-generate-plan"
            variant="primary"
            disabled={!canGenerate}
            disabledReason={!canGenerate ? t('review.generateDisabled') : undefined}
          >
            {t('review.generate')}
          </Button>
        </aside>
      </div>
    );
  }

  return (
    <RouteShell scrId="SCR-05" title={t('review.title')} notice={t('shared.synthetic')}>
      <PageStateBoundary
        mode={mode}
        regionLabel="SCR-05-review"
        emptyReason={t('review.empty')}
        loadingExpectation={t('review.parsingHint')}
        forbiddenPermission={t('review.forbidden')}
        recoveryPoint={t('review.recovering')}
      >
        {content}
      </PageStateBoundary>
    </RouteShell>
  );
}
