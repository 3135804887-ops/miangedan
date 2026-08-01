-- TASK-012 简历隔离上传与安全扫描状态。
-- 追踪：FR-001、FR-006、TM-02、SEC-020、NFR-006；docs/data/DATA-MODEL.md 第 5.2 节。

CREATE TABLE resume_uploads (
    upload_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    idempotency_key text NOT NULL,
    content_fingerprint char(64) NOT NULL,
    filename text NOT NULL,
    size_bytes bigint NOT NULL CHECK (size_bytes >= 0 AND size_bytes <= 10485760),
    detected_media_type text,
    status text NOT NULL CHECK (status IN (
        'QUARANTINED', 'SCANNING', 'ACCEPTED', 'REJECTED', 'RETRYABLE_FAILURE'
    )),
    object_bucket text,
    object_key text,
    rejection_reason text CHECK (rejection_reason IS NULL OR rejection_reason IN (
        'unsupported_type', 'oversized', 'type_spoofed', 'corrupted', 'encrypted',
        'virus_detected', 'macros_detected', 'archive_bomb_detected',
        'sandbox_policy_violation'
    )),
    user_message text,
    sandbox_attestation jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT resume_uploads_idempotency_unique
        UNIQUE (data_region, user_id, idempotency_key),
    CONSTRAINT resume_uploads_upload_bucket_only CHECK (
        object_bucket IS NULL OR object_bucket = btrim(data_region) || '-uploads'
    ),
    CONSTRAINT resume_uploads_object_state CHECK (
        (status = 'REJECTED' AND object_bucket IS NULL AND object_key IS NULL)
        OR (status = 'QUARANTINED' AND (
            (object_bucket IS NULL AND object_key IS NULL)
            OR (object_bucket IS NOT NULL AND object_key IS NOT NULL)
        ))
        OR (status NOT IN ('REJECTED', 'QUARANTINED')
            AND object_bucket IS NOT NULL AND object_key IS NOT NULL)
    )
);

CREATE INDEX resume_uploads_user_created_idx
    ON resume_uploads (user_id, created_at DESC);

CREATE TABLE upload_scan_attempts (
    attempt_id uuid PRIMARY KEY,
    upload_id uuid NOT NULL REFERENCES resume_uploads(upload_id),
    attempt_number integer NOT NULL CHECK (attempt_number >= 1),
    idempotency_key text NOT NULL,
    status text NOT NULL CHECK (status IN ('RUNNING', 'ACCEPTED', 'REJECTED', 'RETRYABLE_FAILURE')),
    failure_code text,
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT upload_scan_attempts_sequence_unique UNIQUE (upload_id, attempt_number),
    CONSTRAINT upload_scan_attempts_idempotency_unique UNIQUE (upload_id, idempotency_key)
);

GRANT SELECT, INSERT, UPDATE ON resume_uploads, upload_scan_attempts TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON resume_uploads, upload_scan_attempts TO mgd_deletion_orchestrator;
