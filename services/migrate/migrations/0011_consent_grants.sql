-- TASK-011：六类独立授权、版本化证据、即时撤回与同事务追加式审计（FR-040）。
-- 约束：授权版本只追加；scope/evidence 仅保存封闭结构和版本元数据；
--       每个版本必须引用一条 AccessAudit，业务角色无 UPDATE/DELETE。

CREATE UNIQUE INDEX access_audits_id_region_unique
    ON access_audits (audit_id, data_region);

CREATE TABLE consent_grants (
    grant_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    consent_type text NOT NULL
        CHECK (consent_type IN (
            'core_service', 'raw_av_recording', 'org_sharing',
            'product_analytics', 'model_training', 'marketing'
        )),
    scope_json jsonb NOT NULL,
    scope_hash text NOT NULL CHECK (scope_hash ~ '^[0-9a-f]{64}$'),
    status text NOT NULL CHECK (status IN ('granted', 'withdrawn', 'expired')),
    granted_at timestamptz NOT NULL,
    expires_at timestamptz,
    withdrawn_at timestamptz,
    supersedes_grant_id uuid,
    evidence_json jsonb NOT NULL,
    evidence_hash text NOT NULL CHECK (evidence_hash ~ '^[0-9a-f]{64}$'),
    version integer NOT NULL CHECK (version > 0),
    request_operation text NOT NULL CHECK (request_operation IN ('grant', 'withdraw')),
    request_key text NOT NULL
        CHECK (request_key ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$'),
    request_hash text NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    audit_id uuid NOT NULL UNIQUE,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT consent_grants_user_region_fk
        FOREIGN KEY (user_id, data_region) REFERENCES users (user_id, data_region),
    CONSTRAINT consent_grants_supersedes_fk
        FOREIGN KEY (supersedes_grant_id) REFERENCES consent_grants (grant_id),
    CONSTRAINT consent_grants_audit_fk
        FOREIGN KEY (audit_id, data_region) REFERENCES access_audits (audit_id, data_region),
    CONSTRAINT consent_grants_scope_version_unique
        UNIQUE (user_id, consent_type, scope_hash, version),
    CONSTRAINT consent_grants_request_unique
        UNIQUE (data_region, user_id, request_operation, request_key),
    CONSTRAINT consent_grants_state_shape CHECK (
        (status = 'granted' AND withdrawn_at IS NULL)
        OR (status = 'withdrawn' AND withdrawn_at IS NOT NULL)
        OR (status = 'expired' AND expires_at IS NOT NULL)
    ),
    CONSTRAINT consent_grants_expiry_after_grant CHECK (
        expires_at IS NULL OR expires_at > granted_at
    ),
    CONSTRAINT consent_grants_raw_av_max_30_days CHECK (
        consent_type <> 'raw_av_recording'
        OR (expires_at IS NOT NULL AND expires_at <= granted_at + interval '30 days')
    ),
    CONSTRAINT consent_grants_org_share_has_expiry CHECK (
        consent_type <> 'org_sharing' OR expires_at IS NOT NULL
    ),
    CONSTRAINT consent_grants_scope_closed_shape CHECK (
        jsonb_typeof(scope_json) = 'object'
        AND scope_json - 'assignment_id' - 'data_categories' - 'media_categories' - 'channels' = '{}'::jsonb
    ),
    CONSTRAINT consent_grants_scope_values_closed CHECK (
        CASE WHEN scope_json ? 'assignment_id'
            THEN jsonb_typeof(scope_json -> 'assignment_id') = 'string'
                AND (scope_json ->> 'assignment_id') ~
                    '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$'
            ELSE true END
        AND CASE WHEN scope_json ? 'data_categories'
            THEN jsonb_typeof(scope_json -> 'data_categories') = 'array'
                AND jsonb_array_length(scope_json -> 'data_categories') BETWEEN 1 AND 6
                AND scope_json -> 'data_categories' <@
                    '["total_score", "radar", "round_results", "full_report", "transcript", "media"]'::jsonb
            ELSE true END
        AND CASE WHEN scope_json ? 'media_categories'
            THEN jsonb_typeof(scope_json -> 'media_categories') = 'array'
                AND jsonb_array_length(scope_json -> 'media_categories') BETWEEN 1 AND 2
                AND scope_json -> 'media_categories' <@ '["audio", "video"]'::jsonb
            ELSE true END
        AND CASE WHEN scope_json ? 'channels'
            THEN jsonb_typeof(scope_json -> 'channels') = 'array'
                AND jsonb_array_length(scope_json -> 'channels') BETWEEN 1 AND 3
                AND scope_json -> 'channels' <@ '["email", "in_app", "push"]'::jsonb
            ELSE true END
    ),
    CONSTRAINT consent_grants_scope_type_shape CHECK (
        CASE consent_type
            WHEN 'core_service' THEN scope_json = '{}'::jsonb
            WHEN 'product_analytics' THEN scope_json = '{}'::jsonb
            WHEN 'model_training' THEN scope_json = '{}'::jsonb
            WHEN 'raw_av_recording' THEN scope_json ? 'media_categories'
                AND scope_json - 'media_categories' = '{}'::jsonb
            WHEN 'org_sharing' THEN scope_json ?& ARRAY['assignment_id', 'data_categories']
                AND scope_json - 'assignment_id' - 'data_categories' = '{}'::jsonb
            WHEN 'marketing' THEN scope_json ? 'channels'
                AND scope_json - 'channels' = '{}'::jsonb
            ELSE false
        END
    ),
    CONSTRAINT consent_grants_evidence_closed_shape CHECK (
        jsonb_typeof(evidence_json) = 'object'
        AND evidence_json ?& ARRAY[
            'copy_version', 'privacy_policy_version', 'presented_at', 'ui_context',
            'action', 'recorded_at', 'evidence_hash'
        ]
        AND evidence_json - 'copy_version' - 'privacy_policy_version' - 'presented_at'
            - 'ui_context' - 'action' - 'recorded_at' - 'evidence_hash' = '{}'::jsonb
        AND jsonb_typeof(evidence_json -> 'copy_version') = 'string'
        AND jsonb_typeof(evidence_json -> 'privacy_policy_version') = 'string'
        AND jsonb_typeof(evidence_json -> 'presented_at') = 'string'
        AND jsonb_typeof(evidence_json -> 'action') = 'string'
        AND jsonb_typeof(evidence_json -> 'recorded_at') = 'string'
        AND jsonb_typeof(evidence_json -> 'evidence_hash') = 'string'
        AND (evidence_json ->> 'copy_version') ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND (evidence_json ->> 'privacy_policy_version') ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
        AND (evidence_json ->> 'action') = request_operation
        AND (evidence_json ->> 'evidence_hash') = evidence_hash
        AND (evidence_json ->> 'presented_at')::timestamptz <= recorded_at + interval '5 minutes'
        AND (evidence_json ->> 'recorded_at')::timestamptz = recorded_at
        AND jsonb_typeof(evidence_json -> 'ui_context') = 'object'
        AND (evidence_json -> 'ui_context') ?& ARRAY['surface', 'flow', 'ui_language']
        AND (evidence_json -> 'ui_context') - 'surface' - 'flow' - 'ui_language' = '{}'::jsonb
        AND jsonb_typeof(evidence_json -> 'ui_context' -> 'surface') = 'string'
        AND jsonb_typeof(evidence_json -> 'ui_context' -> 'flow') = 'string'
        AND jsonb_typeof(evidence_json -> 'ui_context' -> 'ui_language') = 'string'
        AND (evidence_json -> 'ui_context' ->> 'surface') IN ('web', 'ios', 'android')
        AND (evidence_json -> 'ui_context' ->> 'flow') IN (
            'registration', 'consent_center', 'interview_room', 'assignment_share'
        )
        AND (evidence_json -> 'ui_context' ->> 'ui_language') IN ('zh-CN', 'en-US')
    ),
    CONSTRAINT consent_grants_scope_size CHECK (octet_length(scope_json::text) <= 2048),
    CONSTRAINT consent_grants_evidence_size CHECK (octet_length(evidence_json::text) <= 4096)
);

CREATE INDEX consent_grants_current_idx
    ON consent_grants (user_id, consent_type, scope_hash, version DESC);
CREATE INDEX consent_grants_expiry_idx
    ON consent_grants (data_region, status, expires_at);

REVOKE UPDATE, DELETE ON consent_grants FROM PUBLIC;
GRANT SELECT, INSERT ON consent_grants TO mgd_app_runtime;
GRANT SELECT, INSERT ON consent_grants TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON consent_grants TO mgd_deletion_orchestrator;
