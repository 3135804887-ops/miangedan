-- TASK-053 正式重试尝试（DOMAIN-MODEL §6.14；SCORING-SPEC 6.7）
-- 约束：仅 FAIL/EVALUATION_INCOMPLETE 的轮次可重试（应用层）；状态推进
--       RETRY_SCHEDULED → RETRY_IN_PROGRESS → SCORING → COMPLETED；
--       retry_attempts 为流程登记表（非评分账本），允许状态 UPDATE，
--       但分数与证据仍在 score_versions / evidence_items 追加式记录。

CREATE TABLE retry_attempts (
    attempt_id uuid PRIMARY KEY,
    project_id uuid NOT NULL,
    round_sequence integer NOT NULL CHECK (round_sequence BETWEEN 1 AND 5),
    source_attempt_id uuid NOT NULL,
    status text NOT NULL
        CHECK (status IN ('RETRY_SCHEDULED', 'RETRY_IN_PROGRESS', 'SCORING', 'COMPLETED')),
    locked_dimensions jsonb NOT NULL DEFAULT '[]',
    rescope_dimensions jsonb NOT NULL DEFAULT '[]',
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT retry_attempts_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX retry_attempts_project_round_idx
    ON retry_attempts (project_id, round_sequence, created_at);

GRANT SELECT, INSERT, UPDATE ON retry_attempts TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON retry_attempts TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON retry_attempts TO mgd_deletion_orchestrator;
