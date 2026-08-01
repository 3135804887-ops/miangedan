-- TASK-013 简历解析任务与追加式结构化版本。
-- 追踪：FR-002、FR-003、SEC-040、NFR-006、NFR-015。

CREATE TABLE resumes (
    resume_id uuid PRIMARY KEY,
    upload_id uuid NOT NULL UNIQUE REFERENCES resume_uploads(upload_id),
    user_id uuid NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    language text NOT NULL CHECK (language IN ('zh-CN', 'en-US')),
    status text NOT NULL CHECK (status IN (
        'PARSING', 'AWAITING_CONFIRMATION', 'CONFIRMED', 'RETRYABLE_FAILURE', 'FAILED'
    )),
    current_version integer CHECK (current_version IS NULL OR current_version >= 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX resumes_user_updated_idx ON resumes (user_id, updated_at DESC);

CREATE TABLE resume_parse_attempts (
    task_id uuid PRIMARY KEY,
    resume_id uuid NOT NULL REFERENCES resumes(resume_id),
    idempotency_key text NOT NULL,
    input_fingerprint char(64) NOT NULL,
    status text NOT NULL CHECK (status IN (
        'PENDING', 'PARSING', 'AWAITING_CONFIRMATION', 'RETRYABLE_FAILURE', 'FAILED'
    )),
    provider_version text,
    prompt_version text,
    input_retained boolean NOT NULL DEFAULT true,
    retryable boolean NOT NULL DEFAULT false,
    failure_code text,
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT resume_parse_attempts_idempotency_unique
        UNIQUE (resume_id, idempotency_key)
);

CREATE TABLE resume_versions (
    resume_id uuid NOT NULL REFERENCES resumes(resume_id),
    resume_version integer NOT NULL CHECK (resume_version >= 1),
    base_version integer CHECK (base_version IS NULL OR base_version >= 1),
    idempotency_key text NOT NULL,
    operation_fingerprint char(64) NOT NULL,
    profile_json jsonb NOT NULL,
    excluded_sensitive_fields text[] NOT NULL DEFAULT '{}',
    low_confidence_paths text[] NOT NULL DEFAULT '{}',
    reviewed_low_confidence_paths text[] NOT NULL DEFAULT '{}',
    confirmed_by_user boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (resume_id, resume_version),
    CONSTRAINT resume_versions_idempotency_unique UNIQUE (resume_id, idempotency_key),
    CONSTRAINT resume_versions_no_sensitive_root_keys CHECK (
        NOT profile_json ?| ARRAY[
            'phone', 'mobile', 'email', 'id_number', 'identity_number', 'address',
            'photo', 'avatar', 'gender', 'sex', 'age', 'race', 'ethnicity',
            'nationality', 'disability', 'marital_status', 'religion', 'appearance',
            'emotion', 'micro_expression', 'personality'
        ]
    ),
    CONSTRAINT resume_versions_confirmed_has_no_low_confidence CHECK (
        NOT confirmed_by_user OR cardinality(low_confidence_paths) = 0
    )
);

GRANT SELECT, INSERT, UPDATE ON resumes, resume_parse_attempts TO mgd_app_runtime;
GRANT SELECT, INSERT ON resume_versions TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON resumes, resume_parse_attempts, resume_versions
    TO mgd_deletion_orchestrator;
