-- TASK-064 Pro 订阅生命周期（FR-033；BILLING-STATE-MACHINE §5.5）
-- 约束：自动续费必须单独勾选并记录同意条款（价格/权益变化须重新同意）；
--       扣款前提醒事件（renewal_events）先于扣款；同一订阅同一账期唯一；
--       到期不删除历史（订阅/权益状态迁移为 SUB_EXPIRED/expired）。
ALTER TABLE subscriptions ADD COLUMN consent_price_cents integer;
ALTER TABLE subscriptions ADD COLUMN consent_monthly_seconds integer;
ALTER TABLE subscriptions ADD COLUMN idempotency_key_renew text;
CREATE TABLE renewal_events (
    renewal_id uuid PRIMARY KEY,
    subscription_id uuid NOT NULL REFERENCES subscriptions (subscription_id),
    user_id uuid NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    monthly_seconds integer NOT NULL CHECK (monthly_seconds > 0),
    price_cents integer NOT NULL CHECK (price_cents >= 0),
    status text NOT NULL CHECK (status IN ('reminded', 'charged', 'failed')),
    reminded_at timestamptz,
    charged_at timestamptz,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT renewal_events_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT renewal_events_period_unique UNIQUE (subscription_id, period_start)
);
CREATE INDEX renewal_events_subscription_idx ON renewal_events (subscription_id);
CREATE INDEX renewal_events_status_created_idx ON renewal_events (status, created_at);
GRANT SELECT, INSERT, UPDATE ON renewal_events TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON renewal_events TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON renewal_events TO mgd_deletion_orchestrator;
