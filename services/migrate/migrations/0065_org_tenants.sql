-- TASK-070 机构租户与六类角色（FR-034；DOMAIN-MODEL §6.16）
-- 约束：用户以个人账户加入（org_members.user_id 引用个人账户，无影子账户）；
--       六类角色封闭枚举；机构停用/注销后成员访问失效（应用层强制）；
--       邀请幂等键唯一；审计复用追加式 access_audits（SELECT/INSERT only）。
CREATE TABLE organizations (
    org_id uuid PRIMARY KEY,
    name text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    status text NOT NULL CHECK (status IN ('active', 'suspended', 'deactivated')),
    created_by uuid NOT NULL,
    idempotency_key text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE TABLE org_members (
    org_id uuid NOT NULL REFERENCES organizations (org_id),
    user_id uuid NOT NULL,
    role text NOT NULL
        CHECK (role IN ('owner', 'admin', 'instructor', 'privacy_auditor', 'finance', 'candidate')),
    invite_method text NOT NULL
        CHECK (invite_method IN ('link', 'org_email', 'bulk_list', 'sso', 'scim')),
    joined_at timestamptz NOT NULL DEFAULT now(),
    left_at timestamptz,
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX org_members_user_idx ON org_members (user_id);

CREATE TABLE org_invitations (
    invitation_id uuid PRIMARY KEY,
    org_id uuid NOT NULL REFERENCES organizations (org_id),
    email text,
    role text NOT NULL
        CHECK (role IN ('owner', 'admin', 'instructor', 'privacy_auditor', 'finance', 'candidate')),
    invite_method text NOT NULL
        CHECK (invite_method IN ('link', 'org_email', 'bulk_list', 'sso', 'scim')),
    status text NOT NULL CHECK (status IN ('pending', 'accepted', 'revoked')),
    expires_at timestamptz NOT NULL,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT org_invitations_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX org_invitations_org_status_idx ON org_invitations (org_id, status);

GRANT SELECT, INSERT, UPDATE ON organizations, org_members, org_invitations TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON organizations, org_members, org_invitations TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON organizations, org_members, org_invitations
    TO mgd_deletion_orchestrator;
