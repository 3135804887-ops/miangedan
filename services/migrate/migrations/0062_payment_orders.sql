-- TASK-062 区域化支付集成（FR-033；BILLING-STATE-MACHINE §5.2/§8/§9）
-- 约束：订单幂等键唯一；支付回调 provider+payment_event_id 去重；
--       provider_txn_id 唯一（对账按流水号收敛）；重复扣款自动原路退回写
--       refunds + incidents；退款/事故为追加式记录。

CREATE TABLE orders (
    order_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    quote_id uuid NOT NULL REFERENCES quotes (quote_id),
    status text NOT NULL
        CHECK (status IN ('ORDER_CREATED', 'PAYMENT_PENDING', 'PAID',
                          'PAYMENT_FAILED', 'PAYMENT_TIMEOUT', 'ORDER_CANCELLED')),
    amount_cents integer NOT NULL CHECK (amount_cents >= 0),
    currency text NOT NULL,
    payment_method text NOT NULL,
    provider text,
    provider_txn_id text,
    refunded_cents integer NOT NULL DEFAULT 0 CHECK (refunded_cents >= 0),
    progress_note text,
    auto_renew_consent boolean NOT NULL DEFAULT false,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    paid_at timestamptz,
    CONSTRAINT orders_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX orders_user_status_idx ON orders (user_id, status);
CREATE UNIQUE INDEX orders_provider_txn_unique
    ON orders (provider, provider_txn_id) WHERE provider_txn_id IS NOT NULL;

CREATE TABLE payment_events (
    payment_event_id text NOT NULL,
    provider text NOT NULL,
    order_id uuid NOT NULL REFERENCES orders (order_id),
    event_type text NOT NULL
        CHECK (event_type IN ('payment_succeeded', 'payment_failed', 'refund_succeeded')),
    payload_hash text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    processed_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payment_events_provider_event_unique UNIQUE (provider, payment_event_id)
);

CREATE TABLE refunds (
    refund_id uuid PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders (order_id),
    user_id uuid NOT NULL,
    amount_cents integer NOT NULL CHECK (amount_cents > 0),
    currency text NOT NULL,
    reason text NOT NULL,
    kind text NOT NULL
        CHECK (kind IN ('user_request', 'system_fault', 'compensation', 'duplicate_charge')),
    status text NOT NULL
        CHECK (status IN ('REFUND_REQUESTED', 'REFUND_REVIEWING', 'REFUND_APPROVED',
                          'REFUNDED', 'REFUND_REJECTED')),
    reject_reason text,
    approver_pair_json jsonb,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    refunded_at timestamptz,
    CONSTRAINT refunds_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX refunds_order_idx ON refunds (order_id);
CREATE INDEX refunds_status_created_idx ON refunds (status, created_at);

CREATE TABLE incidents (
    incident_id uuid PRIMARY KEY,
    kind text NOT NULL
        CHECK (kind IN ('duplicate_charge', 'payment_fault', 'fault', 'break_glass',
                        'release', 'rollback', 'compensation')),
    severity text NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    region char(4) NOT NULL CHECK (region IN ('cn', 'eu', 'intl')),
    summary text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX incidents_kind_severity_created_idx ON incidents (kind, severity, created_at);

GRANT SELECT, INSERT, UPDATE ON orders, payment_events, refunds, incidents TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON orders, payment_events, refunds, incidents TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON orders, payment_events, refunds, incidents
    TO mgd_deletion_orchestrator;
