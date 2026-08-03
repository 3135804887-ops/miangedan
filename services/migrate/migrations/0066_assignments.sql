-- TASK-071 训练任务与模板（FR-035；DOMAIN-MODEL Assignment）
-- 约束：模板禁止项（60 分线/统一评分算法/保护属性/证据标准/跨轮解锁/正式复核）
--       不落在表结构（无对应列），任何携带禁止键的写入由服务层拒绝并写审计；
--       任务状态 draft → published → closed；成员状态默认最小可见。
CREATE TABLE assignments (
    assignment_id uuid PRIMARY KEY,
    org_id uuid NOT NULL REFERENCES organizations (org_id),
    title text NOT NULL,
    job_category text,
    template_json jsonb NOT NULL DEFAULT '{}',
    deadline_at timestamptz NOT NULL,
    max_practice_count integer NOT NULL DEFAULT 0 CHECK (max_practice_count >= 0),
    org_credit_seconds integer NOT NULL DEFAULT 0 CHECK (org_credit_seconds >= 0),
    status text NOT NULL CHECK (status IN ('draft', 'published', 'closed')),
    created_by uuid NOT NULL,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT assignments_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX assignments_org_status_idx ON assignments (org_id, status);

CREATE TABLE assignment_members (
    assignment_id uuid NOT NULL REFERENCES assignments (assignment_id),
    user_id uuid NOT NULL,
    status text NOT NULL
        CHECK (status IN ('not_started', 'in_progress', 'completed', 'exited')),
    completed_at timestamptz,
    fault_flag boolean NOT NULL DEFAULT false,
    org_credit_used_seconds integer NOT NULL DEFAULT 0 CHECK (org_credit_used_seconds >= 0),
    PRIMARY KEY (assignment_id, user_id)
);

CREATE INDEX assignment_members_user_idx ON assignment_members (user_id);

GRANT SELECT, INSERT, UPDATE ON assignments, assignment_members TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON assignments, assignment_members TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON assignments, assignment_members
    TO mgd_deletion_orchestrator;
