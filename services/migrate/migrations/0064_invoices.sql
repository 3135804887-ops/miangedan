-- TASK-065 发票与税费（FR-033；BILLING-STATE-MACHINE §7）
-- 约束：中国区合规发票（kind=invoice，含发票号码与增值税行）；
--       国际区税费明示收据（kind=receipt）；同一订单最多一份有效票据；
--       发票作废后不得再次开票（红冲流程另行处理）。
CREATE TABLE invoices (
    invoice_id uuid PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders (order_id),
    user_id uuid NOT NULL,
    kind text NOT NULL CHECK (kind IN ('invoice', 'receipt')),
    number text NOT NULL,
    currency text NOT NULL,
    subtotal_cents integer NOT NULL CHECK (subtotal_cents >= 0),
    tax_json jsonb NOT NULL DEFAULT '[]',
    total_cents integer NOT NULL CHECK (total_cents >= 0),
    status text NOT NULL CHECK (status IN ('issued', 'cancelled')),
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    cancelled_at timestamptz,
    CONSTRAINT invoices_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT invoices_order_unique UNIQUE (order_id)
);

CREATE INDEX invoices_user_created_idx ON invoices (user_id, created_at);

GRANT SELECT, INSERT, UPDATE ON invoices TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON invoices TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON invoices TO mgd_deletion_orchestrator;
