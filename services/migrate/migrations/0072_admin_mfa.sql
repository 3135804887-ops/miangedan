-- TASK-084 追加式审计日志与抗钓鱼 MFA（FR-037/FR-040）
-- 约束：审计沿用 access_audits 追加式（管理员不可删除；无 UPDATE/DELETE 授权）；
--       MFA 挑战 5 分钟有效、一次使用（抗重放）；验证 15 分钟高风险窗口；
--       设备公钥绑定员工（WebAuthn 适配点）。
CREATE TABLE mfa_devices (
    device_id uuid PRIMARY KEY,
    staff_id uuid NOT NULL,
    name text NOT NULL,
    public_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    registered_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz
);

CREATE INDEX mfa_devices_staff_idx ON mfa_devices (staff_id);

CREATE TABLE mfa_challenges (
    challenge_id uuid PRIMARY KEY,
    staff_id uuid NOT NULL,
    nonce text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    expires_at timestamptz NOT NULL,
    used_at timestamptz
);

CREATE TABLE mfa_verifications (
    verification_id uuid PRIMARY KEY,
    staff_id uuid NOT NULL,
    challenge_id uuid NOT NULL REFERENCES mfa_challenges (challenge_id),
    device_id uuid NOT NULL REFERENCES mfa_devices (device_id),
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    verified_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX mfa_verifications_staff_created_idx ON mfa_verifications (staff_id, verified_at);

GRANT SELECT, INSERT, UPDATE ON mfa_devices, mfa_challenges, mfa_verifications
    TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON mfa_devices, mfa_challenges, mfa_verifications
    TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON mfa_devices, mfa_challenges, mfa_verifications
    TO mgd_deletion_orchestrator;

-- 审计沿用 access_audits（基线 0001 已仅授权 SELECT/INSERT 给业务角色）。
