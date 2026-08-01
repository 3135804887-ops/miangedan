-- TASK-014 JD 原文、解析尝试、追加式岗位版本与材料降级同意。
-- 追踪：US-01、FR-004、FR-005、NFR-006、NFR-015。

CREATE TABLE job_profiles (
    job_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    language text NOT NULL CHECK (language IN ('zh-CN', 'en-US')),
    source_kind text NOT NULL CHECK (source_kind IN ('jd_text', 'resume_inference')),
    source_resume_id uuid,
    source_resume_version integer CHECK (
        source_resume_version IS NULL OR source_resume_version >= 1
    ),
    raw_text_bucket text,
    raw_text_ref text,
    create_idempotency_key text NOT NULL,
    input_fingerprint char(64) NOT NULL,
    status text NOT NULL CHECK (status IN (
        'CREATED', 'PARSING', 'AWAITING_CONFIRMATION', 'CONFIRMED',
        'RETRYABLE_FAILURE', 'FAILED'
    )),
    current_version integer CHECK (current_version IS NULL OR current_version >= 1),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT job_profiles_create_idempotency_unique
        UNIQUE (data_region, user_id, create_idempotency_key),
    CONSTRAINT job_profiles_source_shape CHECK (
        (
            source_kind = 'jd_text'
            AND source_resume_id IS NULL
            AND source_resume_version IS NULL
            AND raw_text_bucket = btrim(data_region) || '-uploads'
            AND raw_text_ref IS NOT NULL
        ) OR (
            source_kind = 'resume_inference'
            AND source_resume_id IS NOT NULL
            AND source_resume_version IS NOT NULL
            AND raw_text_bucket IS NULL
            AND raw_text_ref IS NULL
        )
    )
);

CREATE INDEX job_profiles_user_updated_idx
    ON job_profiles (user_id, updated_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE job_parse_attempts (
    task_id uuid PRIMARY KEY,
    job_id uuid NOT NULL REFERENCES job_profiles(job_id),
    idempotency_key text NOT NULL,
    input_fingerprint char(64) NOT NULL,
    status text NOT NULL CHECK (status IN (
        'PARSING', 'AWAITING_CONFIRMATION', 'RETRYABLE_FAILURE', 'FAILED'
    )),
    provider_version text,
    prompt_version text,
    input_retained boolean NOT NULL DEFAULT true,
    retryable boolean NOT NULL DEFAULT false,
    failure_code text,
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT job_parse_attempts_idempotency_unique UNIQUE (job_id, idempotency_key)
);

CREATE TABLE job_versions (
    job_id uuid NOT NULL REFERENCES job_profiles(job_id),
    job_version integer NOT NULL CHECK (job_version >= 1),
    base_version integer CHECK (base_version IS NULL OR base_version >= 1),
    idempotency_key text NOT NULL,
    operation_fingerprint char(64) NOT NULL,
    profile_json jsonb NOT NULL,
    excluded_from_scoring text[] NOT NULL DEFAULT '{}',
    confirmed_by_user boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (job_id, job_version),
    CONSTRAINT job_versions_idempotency_unique UNIQUE (job_id, idempotency_key),
    CONSTRAINT job_versions_excluded_categories CHECK (
        excluded_from_scoring <@ ARRAY[
            'salary_benefits', 'recruiter_contact', 'company_perks'
        ]::text[]
    ),
    CONSTRAINT job_versions_no_excluded_root_keys CHECK (
        NOT profile_json ?| ARRAY[
            'salary', 'compensation', 'benefits', 'perks', 'recruiter',
            'recruiter_contact', 'contact_email', 'contact_phone'
        ]
    ),
    CONSTRAINT job_versions_ai_focus_marker_present CHECK (
        jsonb_typeof(profile_json->'ai_inferred_interview_focus') = 'array'
        AND jsonb_array_length(profile_json->'ai_inferred_interview_focus')
            = jsonb_array_length(jsonb_path_query_array(
                profile_json,
                '$.ai_inferred_interview_focus[*] ? (@.ai_inferred == true && @.editable == true)'
            ))
    )
);

CREATE TABLE material_readiness_assessments (
    assessment_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    resume_id uuid,
    resume_version integer CHECK (resume_version IS NULL OR resume_version >= 1),
    job_id uuid,
    job_version integer CHECK (job_version IS NULL OR job_version >= 1),
    mode text NOT NULL CHECK (mode IN ('full', 'jd_only', 'resume_only', 'neither')),
    consent_required boolean NOT NULL,
    impact_snapshot_json jsonb NOT NULL,
    input_fingerprint char(64) NOT NULL,
    idempotency_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT material_assessments_idempotency_unique
        UNIQUE (data_region, user_id, idempotency_key),
    CONSTRAINT material_assessments_mode_shape CHECK (
        (mode = 'full' AND resume_id IS NOT NULL AND resume_version IS NOT NULL
            AND job_id IS NOT NULL AND job_version IS NOT NULL
            AND NOT consent_required)
        OR (mode = 'jd_only' AND resume_id IS NULL AND resume_version IS NULL
            AND job_id IS NOT NULL AND job_version IS NOT NULL
            AND consent_required)
        OR (mode = 'resume_only' AND resume_id IS NOT NULL AND resume_version IS NOT NULL
            AND job_id IS NULL AND job_version IS NULL
            AND consent_required)
        OR (mode = 'neither' AND resume_id IS NULL AND resume_version IS NULL
            AND job_id IS NULL AND job_version IS NULL
            AND consent_required)
    )
);

CREATE TABLE material_degradation_consents (
    consent_grant_id uuid PRIMARY KEY,
    assessment_id uuid NOT NULL REFERENCES material_readiness_assessments(assessment_id),
    user_id uuid NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    mode text NOT NULL CHECK (mode IN ('jd_only', 'resume_only', 'neither')),
    accepted boolean NOT NULL CHECK (accepted),
    impact_snapshot_json jsonb NOT NULL,
    operation_fingerprint char(64) NOT NULL,
    idempotency_key text NOT NULL,
    granted_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT material_consents_idempotency_unique
        UNIQUE (data_region, user_id, idempotency_key),
    CONSTRAINT material_consents_one_per_assessment UNIQUE (assessment_id)
);

GRANT SELECT, INSERT, UPDATE ON job_profiles, job_parse_attempts TO mgd_app_runtime;
GRANT SELECT, INSERT ON job_versions, material_readiness_assessments,
    material_degradation_consents TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON job_profiles, job_parse_attempts, job_versions,
    material_readiness_assessments, material_degradation_consents TO mgd_deletion_orchestrator;
