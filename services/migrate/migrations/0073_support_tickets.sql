-- TASK-085 客服工单（FR-039；SCREEN-SPEC SCR-17 客服工单）
-- 约束：默认最小可见（工单表不含逐字稿/媒体正文）；逐字稿访问需用户针对会话
--       授权（范围+有效期）；媒体访问需用户申请+双人审批（审批人去重）。
CREATE TABLE tickets (
    ticket_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    subject text NOT NULL,
    category text NOT NULL
        CHECK (category IN ('account', 'order', 'entitlement', 'fault',
                            'transcript', 'media', 'other')),
    status text NOT NULL CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    visibility text NOT NULL CHECK (visibility IN ('minimal', 'authorized')),
    created_by uuid NOT NULL,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz,
    CONSTRAINT tickets_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX tickets_user_status_idx ON tickets (user_id, status);

CREATE TABLE ticket_transcript_auths (
    auth_id uuid PRIMARY KEY,
    ticket_id uuid NOT NULL REFERENCES tickets (ticket_id),
    user_id uuid NOT NULL,
    session_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('active', 'expired', 'revoked')),
    expires_at timestamptz NOT NULL,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    granted_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ticket_transcript_auths_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX ticket_transcript_auths_session_idx
    ON ticket_transcript_auths (ticket_id, session_id);

CREATE TABLE ticket_media_requests (
    access_request_id uuid PRIMARY KEY,
    ticket_id uuid NOT NULL REFERENCES tickets (ticket_id),
    user_id uuid NOT NULL,
    session_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('requested', 'approved', 'rejected')),
    approver_pair_json jsonb NOT NULL DEFAULT '[]',
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    requested_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz,
    CONSTRAINT ticket_media_requests_idempotency_key_unique UNIQUE (idempotency_key)
);

GRANT SELECT, INSERT, UPDATE ON tickets, ticket_transcript_auths, ticket_media_requests
    TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON tickets, ticket_transcript_auths, ticket_media_requests
    TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON tickets, ticket_transcript_auths, ticket_media_requests
    TO mgd_deletion_orchestrator;
