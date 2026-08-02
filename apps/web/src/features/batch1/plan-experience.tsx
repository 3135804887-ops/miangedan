'use client';

import {
  ACCOMMODATION_KEYS,
  defaultAccommodations,
  PROJECT_STATUSES,
  type AccommodationKey,
  type ProjectStatus,
} from '@mgd/domain-states';
import { Button, DisclosureNote, Switch } from '@mgd/ui';
import { useTranslations } from 'next-intl';
import { useState, type DragEvent, type ReactNode } from 'react';

import { PageStateBoundary, type PreviewState } from '../../components/page-state-boundary.tsx';
import { RouteShell } from '../../components/route-shell.tsx';

const TOOL_KEYS = ['code', 'whiteboard', 'case', 'portfolio'] as const;

export function PlanExperience({
  mode,
  projectStatus,
  hasUnreadyRound,
}: {
  readonly mode: PreviewState;
  readonly projectStatus: ProjectStatus;
  readonly hasUnreadyRound: boolean;
}): ReactNode {
  const t = useTranslations('batch1');
  const confirmed = projectStatus === PROJECT_STATUSES[7];
  const [rounds, setRounds] = useState([t('plan.roundOne'), t('plan.roundTwo')]);
  const [accommodations, setAccommodations] = useState(defaultAccommodations());
  const [tools, setTools] = useState<Record<(typeof TOOL_KEYS)[number], boolean>>({ code: false, whiteboard: true, case: true, portfolio: false });
  const [draggedRound, setDraggedRound] = useState<number>();

  function changeAccommodation(key: AccommodationKey, checked: boolean): void {
    setAccommodations((current) => ({ ...current, [key]: checked }));
  }

  function dropRound(targetIndex: number, event: DragEvent<HTMLElement>): void {
    event.preventDefault();
    if (draggedRound === undefined || draggedRound === targetIndex) return;
    setRounds((current) => {
      const next = [...current];
      const [moved] = next.splice(draggedRound, 1);
      if (moved !== undefined) next.splice(targetIndex, 0, moved);
      return next;
    });
    setDraggedRound(undefined);
  }

  let mainContent: ReactNode;
  if (projectStatus === PROJECT_STATUSES[4]) {
    mainContent = (
      <section className="mgd-progress-panel" aria-busy="true" aria-labelledby="plan-generating-title">
        <p className="mgd-kicker">P95 ≤ 120s</p>
        <h2 id="plan-generating-title">{t('plan.generating')}</h2>
        <p>{t('plan.generatingHint')}</p>
        <div className="mgd-module-progress">
          <span>✓ {t('plan.sourceTitle')}</span>
          <span>● {t('plan.rounds')}</span>
          <span>○ {t('plan.tools')}</span>
        </div>
      </section>
    );
  } else {
    mainContent = (
      <>
        {projectStatus === PROJECT_STATUSES[6] ? (
          <section className="mgd-inline-notice" role="alert">
            <strong>{t('plan.partialFailed')}</strong>
            <Button controlId="plan-retry-module">{t('plan.retryModule')}</Button>
          </section>
        ) : null}

        {confirmed ? (
          <section className="mgd-frozen-banner" aria-label={t('plan.frozen')}>
            <strong>✓ {t('plan.frozen')}</strong>
            <p>{t('plan.quote')}</p>
          </section>
        ) : null}

        <div className="mgd-plan-layout">
          <div>
            <section className="mgd-source-card" aria-labelledby="plan-source-title">
              <p className="mgd-kicker">SOURCE / 01</p>
              <h2 id="plan-source-title">{t('plan.sourceTitle')}</h2>
              <p><a href="https://example.com/synthetic-source">{t('plan.source')}</a></p>
              <div className="mgd-action-row"><span className="mgd-warning-label">{t('plan.unofficial')}</span><span>{t('plan.confidence')}</span></div>
            </section>

            <section className="mgd-section" aria-labelledby="plan-rounds-title">
              <div className="mgd-section-heading">
                <h2 id="plan-rounds-title">{t('plan.rounds')}</h2>
                <div className="mgd-action-row">
                  <Button
                    controlId="plan-add-round"
                    disabled={confirmed || rounds.length >= 5}
                    disabledReason={confirmed ? t('plan.frozen') : rounds.length >= 5 ? '1–5' : undefined}
                    onClick={() => setRounds((current) => [...current, `${t('plan.roundTwo')} ${current.length + 1}`])}
                  >
                    {t('plan.addRound')}
                  </Button>
                  <Button
                    controlId="plan-remove-round"
                    variant="danger"
                    disabled={confirmed || rounds.length <= 1}
                    disabledReason={confirmed ? t('plan.frozen') : rounds.length <= 1 ? '1–5' : undefined}
                    onClick={() => setRounds((current) => current.slice(0, -1))}
                  >
                    {t('plan.removeRound')}
                  </Button>
                </div>
              </div>
              <div className="mgd-round-list">
                {rounds.map((round, index) => (
                  <article
                    key={`${round}-${index}`}
                    className="mgd-round-card"
                    draggable={!confirmed}
                    onDragStart={() => setDraggedRound(index)}
                    onDragOver={(event) => event.preventDefault()}
                    onDrop={(event) => dropRound(index, event)}
                  >
                    <div className="mgd-round-card__heading">
                      <span aria-hidden="true">0{index + 1}</span>
                      <h3>{round}</h3>
                      <span className={hasUnreadyRound && index === rounds.length - 1 ? 'mgd-warning-label' : 'mgd-pass-label'}>
                        {hasUnreadyRound && index === rounds.length - 1 ? t('plan.roundNotReady') : t('plan.roundReady')}
                      </span>
                    </div>
                    <div className="mgd-round-fields">
                      <label>{t('plan.role')}<input defaultValue={t('plan.authorizedRole')} disabled={confirmed} /></label>
                      <label>{t('plan.focus')}<input defaultValue={t('plan.focusValue')} disabled={confirmed} /></label>
                      <label>{t('plan.difficulty')}<select defaultValue="standard" disabled={confirmed}><option value="basic">{t('plan.basic')}</option><option value="standard">{t('plan.standard')}</option><option value="challenge">{t('plan.challenge')}</option></select></label>
                      <label>{t('plan.duration')}<select defaultValue="35" disabled={confirmed}>{[10, 20, 35, 45, 60].map((minutes) => <option key={minutes} value={minutes}>{t('plan.minutes', { minutes })}</option>)}</select></label>
                      <label>{t('plan.style')}<input defaultValue={t('plan.interviewerStyle')} disabled={confirmed} /></label>
                      <label>{t('plan.avatarVoice')}<select disabled={confirmed}><option>{t('plan.authorizedRole')}</option></select></label>
                    </div>
                    <Button
                      controlId={`plan-reorder-${index}`}
                      disabled={confirmed || index === 0}
                      disabledReason={confirmed ? t('plan.frozen') : index === 0 ? t('plan.reorder') : undefined}
                      onClick={() => setRounds((current) => index === 0 ? current : [current[index] ?? '', ...current.filter((_, currentIndex) => currentIndex !== index)])}
                    >
                      {t('plan.reorder')}
                    </Button>
                  </article>
                ))}
              </div>
            </section>
          </div>

          <aside className="mgd-plan-controls">
            <section aria-labelledby="plan-tools-title">
              <h2 id="plan-tools-title">{t('plan.tools')}</h2>
              {TOOL_KEYS.map((key) => (
                <Switch
                  key={key}
                  controlId={`plan-tool-${key}`}
                  label={t(`plan.${key}`)}
                  checked={tools[key]}
                  disabled={confirmed}
                  disabledReason={confirmed ? t('plan.frozen') : undefined}
                  onCheckedChange={(checked) => setTools((current) => ({ ...current, [key]: checked }))}
                />
              ))}
            </section>
            <section aria-labelledby="plan-accommodations-title">
              <h2 id="plan-accommodations-title">{t('plan.accommodations')}</h2>
              <p>{t('plan.accommodationHint')}</p>
              {ACCOMMODATION_KEYS.map((key) => (
                <Switch
                  key={key}
                  controlId={`plan-accommodation-${key}`}
                  label={t(`plan.accommodationLabels.${key}`)}
                  checked={accommodations[key]}
                  disabled={confirmed}
                  disabledReason={confirmed ? t('plan.frozen') : undefined}
                  onCheckedChange={(checked) => changeAccommodation(key, checked)}
                />
              ))}
            </section>
          </aside>
        </div>

        <DisclosureNote title={t('plan.readonlyTitle')}>{t('plan.readonlyBody')}</DisclosureNote>
        <div className="mgd-action-row mgd-action-row--end">
          <Button
            controlId="plan-restore-recommendation"
            disabled={confirmed}
            disabledReason={confirmed ? t('plan.frozen') : undefined}
          >
            {t('plan.restore')}
          </Button>
          <Button
            controlId="plan-confirm"
            variant="primary"
            disabled={confirmed || hasUnreadyRound}
            disabledReason={confirmed ? t('plan.frozen') : hasUnreadyRound ? t('plan.confirmDisabled') : undefined}
          >
            {t('plan.confirm')}
          </Button>
        </div>
      </>
    );
  }

  return (
    <RouteShell scrId="SCR-06" title={t('plan.title')} notice={t('shared.synthetic')}>
      <PageStateBoundary
        mode={mode}
        regionLabel="SCR-06-plan"
        emptyReason={t('plan.empty')}
        loadingExpectation={t('plan.generatingHint')}
        forbiddenPermission={t('plan.forbidden')}
        recoveryPoint={t('plan.recovering')}
      >
        {mainContent}
      </PageStateBoundary>
    </RouteShell>
  );
}
