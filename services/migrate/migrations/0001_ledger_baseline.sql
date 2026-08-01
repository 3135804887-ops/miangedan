-- 面个蛋数据平台基线迁移：四张追加式账本表（TASK-003）
-- 追踪：docs/data/DATA-MODEL.md 第 5.3/5.4/5.6 节；ADR-0004；NFR-005、NFR-006
-- 约束：业务角色无 UPDATE/DELETE；幂等键唯一；data_region 强制。

CREATE TABLE evidence_items (
    evidence_id uuid PRIMARY KEY,
    session_id uuid NOT NULL,
    turn_index integer NOT NULL,
    project_id uuid NOT NULL,
    round_sequence integer NOT NULL,
    attempt_id uuid,
    evidence_json jsonb NOT NULL,
    content_hash text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evidence_items_session_turn_unique UNIQUE (session_id, turn_index)
);

CREATE INDEX evidence_items_project_attempt_idx
    ON evidence_items (project_id, attempt_id);

CREATE TABLE score_versions (
    score_id uuid PRIMARY KEY,
    project_id uuid NOT NULL,
    round_sequence integer NOT NULL,
    attempt_id uuid,
    score_version integer NOT NULL,
    result_json jsonb NOT NULL,
    evidence_snapshot_hash text NOT NULL,
    supersedes_score_id uuid,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    computed_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT score_versions_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX score_versions_project_attempt_version_idx
    ON score_versions (project_id, attempt_id, score_version);

CREATE TABLE usage_ledger (
    entry_id uuid PRIMARY KEY,
    entitlement_id uuid,
    user_id uuid NOT NULL,
    project_id uuid,
    round_sequence integer,
    entry_type text NOT NULL
        CHECK (entry_type IN ('reserve', 'consume', 'release', 'refund', 'reversal')),
    seconds integer NOT NULL,
    reason text NOT NULL,
    balance_after integer NOT NULL,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT usage_ledger_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX usage_ledger_entitlement_created_idx
    ON usage_ledger (entitlement_id, created_at);

CREATE TABLE access_audits (
    audit_id uuid PRIMARY KEY,
    subject_type text NOT NULL
        CHECK (subject_type IN ('user', 'org', 'staff', 'system')),
    subject_id uuid NOT NULL,
    actor_id uuid,
    actor_role text NOT NULL,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_id text NOT NULL,
    legal_basis text NOT NULL
        CHECK (legal_basis IN ('consent', 'break_glass', 'system')),
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX access_audits_subject_created_idx
    ON access_audits (subject_id, created_at);

CREATE INDEX access_audits_actor_created_idx
    ON access_audits (actor_id, created_at);

-- 数据库层追加式约束（ADR-0004）：普通应用/运营角色无 UPDATE/DELETE，
-- 物理删除仅删除编排专用角色在数据权利流程中执行。
CREATE ROLE mgd_app_runtime NOLOGIN;
CREATE ROLE mgd_ledger_writer NOLOGIN;
CREATE ROLE mgd_deletion_orchestrator NOLOGIN;

REVOKE UPDATE, DELETE ON evidence_items, score_versions, usage_ledger, access_audits FROM PUBLIC;
GRANT SELECT, INSERT ON evidence_items, score_versions, usage_ledger, access_audits TO mgd_app_runtime;
GRANT SELECT, INSERT ON evidence_items, score_versions, usage_ledger, access_audits TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON evidence_items, score_versions, usage_ledger, access_audits TO mgd_deletion_orchestrator;
