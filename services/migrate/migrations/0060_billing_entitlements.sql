-- TASK-060 报价引擎与权益模型（FR-031；BILLING-STATE-MACHINE §4/§5.1/§5.5）
-- 约束：权益四类封闭枚举；开始后计费版本冻结（billing_freezes）不可修改；
--       余额扣减由 TASK-061 usage_ledger 账本驱动（本表 consumed_seconds 只增）。

CREATE TABLE entitlements (
    entitlement_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    kind text NOT NULL
        CHECK (kind IN ('free_credit', 'project_pack', 'pro_subscription', 'topup_pack')),
    scope_json jsonb NOT NULL DEFAULT '{}',
    total_seconds integer NOT NULL CHECK (total_seconds >= 0),
    consumed_seconds integer NOT NULL DEFAULT 0 CHECK (consumed_seconds >= 0),
    status text NOT NULL CHECK (status IN ('active', 'consumed', 'expired', 'revoked')),
    valid_from timestamptz NOT NULL DEFAULT now(),
    valid_to timestamptz,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entitlements_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX entitlements_user_kind_status_idx
    ON entitlements (user_id, kind, status);

CREATE TABLE quotes (
    quote_id uuid PRIMARY KEY,
    project_id uuid NOT NULL,
    plan_version integer NOT NULL CHECK (plan_version >= 1),
    status text NOT NULL
        CHECK (status IN ('QUOTE_DRAFT', 'QUOTE_PRESENTED', 'QUOTE_ACCEPTED', 'QUOTE_RECALCULATED')),
    total_minutes integer NOT NULL CHECK (total_minutes > 0),
    free_retries integer NOT NULL DEFAULT 0,
    amount_cents integer NOT NULL CHECK (amount_cents >= 0),
    currency text NOT NULL,
    tax_description text,
    valid_until timestamptz NOT NULL,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT quotes_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT quotes_project_version_unique UNIQUE (project_id, plan_version)
);

CREATE TABLE billing_freezes (
    project_id uuid PRIMARY KEY,
    quote_id uuid NOT NULL,
    plan_version integer NOT NULL,
    frozen boolean NOT NULL DEFAULT true,
    frozen_at timestamptz NOT NULL DEFAULT now(),
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl'))
);

CREATE TABLE subscriptions (
    subscription_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('SUB_ACTIVE', 'SUB_CANCELLED', 'SUB_EXPIRED')),
    monthly_seconds integer NOT NULL CHECK (monthly_seconds > 0),
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    carryover_seconds integer NOT NULL DEFAULT 0,
    auto_renew boolean NOT NULL DEFAULT false,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT subscriptions_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT subscriptions_period_order_check CHECK (period_end > period_start)
);

CREATE INDEX subscriptions_user_status_idx
    ON subscriptions (user_id, status);

CREATE INDEX subscriptions_period_end_status_idx
    ON subscriptions (period_end, status);

GRANT SELECT, INSERT, UPDATE ON entitlements, quotes, billing_freezes, subscriptions
    TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON entitlements, quotes, billing_freezes, subscriptions
    TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON entitlements, quotes, billing_freezes, subscriptions
    TO mgd_deletion_orchestrator;
