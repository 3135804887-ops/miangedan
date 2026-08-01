-- TASK-010：用户、Identity、多登录方式与双侧验证绑定（US-05、FR-027）
-- 约束：不保存邮箱明文、验证码明文、OAuth 授权码或业务/刷新令牌明文；
--       data_region 全表强制；(provider, provider_subject_hash) 区域内唯一，冲突绝不自动合并。

CREATE TABLE users (
    user_id uuid PRIMARY KEY,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    ui_language text NOT NULL CHECK (ui_language IN ('zh-CN', 'en-US')),
    age_status text NOT NULL
        CHECK (age_status IN ('adult', 'minor_guardian_verified', 'minor_pending')),
    status text NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'deletion_pending', 'deleted_anonymized')),
    display_name text,
    terms_version text NOT NULL,
    privacy_version text NOT NULL,
    data_processing_version text NOT NULL,
    registration_evidence_json jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_id_region_unique UNIQUE (user_id, data_region)
);

CREATE INDEX users_region_status_idx ON users (data_region, status);

CREATE TABLE identities (
    identity_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    provider text NOT NULL
        CHECK (provider IN ('email_otp', 'google', 'apple', 'wechat')),
    provider_subject_hash text NOT NULL,
    verified_at timestamptz NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT identities_user_region_fk
        FOREIGN KEY (user_id, data_region) REFERENCES users (user_id, data_region),
    CONSTRAINT identities_provider_subject_region_unique
        UNIQUE (data_region, provider, provider_subject_hash)
);

CREATE INDEX identities_user_idx ON identities (user_id);

CREATE TABLE identity_verifications (
    verification_id uuid PRIMARY KEY,
    provider text NOT NULL
        CHECK (provider IN ('email_otp', 'google', 'apple', 'wechat')),
    provider_subject_hash text NOT NULL,
    code_hash text,
    proof_hash text UNIQUE,
    status text NOT NULL
        CHECK (status IN ('pending', 'verified', 'consumed', 'expired', 'locked')),
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts >= 0),
    max_attempts integer NOT NULL CHECK (max_attempts > 0),
    requested_at timestamptz NOT NULL,
    verified_at timestamptz,
    expires_at timestamptz NOT NULL,
    proof_expires_at timestamptz,
    consumed_at timestamptz,
    notification_delivered_at timestamptz,
    request_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT identity_verifications_email_code_shape CHECK (
        (provider = 'email_otp' AND code_hash IS NOT NULL)
        OR (provider <> 'email_otp' AND code_hash IS NULL)
    ),
    CONSTRAINT identity_verifications_region_provider_request_unique
        UNIQUE (data_region, provider, request_key)
);

CREATE INDEX identity_verifications_subject_rate_idx
    ON identity_verifications (data_region, provider, provider_subject_hash, requested_at);
CREATE INDEX identity_verifications_expiry_idx
    ON identity_verifications (status, expires_at);

CREATE TABLE identity_sessions (
    session_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    refresh_token_hash text NOT NULL UNIQUE,
    status text NOT NULL
        CHECK (status IN ('active', 'rotated', 'revoked', 'expired')),
    access_expires_at timestamptz NOT NULL,
    refresh_expires_at timestamptz NOT NULL,
    rotated_to_session_id uuid,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    rotated_at timestamptz,
    CONSTRAINT identity_sessions_user_region_fk
        FOREIGN KEY (user_id, data_region) REFERENCES users (user_id, data_region),
    CONSTRAINT identity_sessions_rotated_to_fk
        FOREIGN KEY (rotated_to_session_id) REFERENCES identity_sessions (session_id)
);

CREATE INDEX identity_sessions_user_status_idx
    ON identity_sessions (user_id, status);

CREATE TABLE identity_conflicts (
    recovery_case_id uuid PRIMARY KEY,
    requesting_user_id uuid NOT NULL,
    conflicting_user_id uuid NOT NULL,
    provider text NOT NULL
        CHECK (provider IN ('email_otp', 'google', 'apple', 'wechat')),
    provider_subject_hash text NOT NULL,
    status text NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'resolved_no_merge', 'closed')),
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT identity_conflicts_requesting_region_fk
        FOREIGN KEY (requesting_user_id, data_region) REFERENCES users (user_id, data_region),
    CONSTRAINT identity_conflicts_conflicting_region_fk
        FOREIGN KEY (conflicting_user_id, data_region) REFERENCES users (user_id, data_region)
);

CREATE INDEX identity_conflicts_requesting_created_idx
    ON identity_conflicts (requesting_user_id, created_at);

CREATE TABLE identity_idempotency (
    idempotency_id uuid PRIMARY KEY,
    operation text NOT NULL,
    idempotency_key text NOT NULL,
    request_hash text NOT NULL,
    result_ref_type text,
    result_ref_id uuid,
    status text NOT NULL CHECK (status IN ('in_progress', 'completed', 'failed')),
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT identity_idempotency_region_operation_key_unique
        UNIQUE (data_region, operation, idempotency_key)
);

-- 幂等记录只保存结果引用，不保存 proof/access/refresh token 明文。
REVOKE UPDATE, DELETE ON identity_conflicts FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON users, identities, identity_verifications,
    identity_sessions, identity_idempotency TO mgd_app_runtime;
GRANT SELECT, INSERT ON identity_conflicts TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON users, identities, identity_verifications,
    identity_sessions, identity_conflicts, identity_idempotency TO mgd_deletion_orchestrator;
