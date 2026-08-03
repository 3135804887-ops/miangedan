-- TASK-072 按任务细粒度结果授权（FR-035；SCREEN-SPEC SCR-16）
-- 约束：范围（六类封闭枚举）+ 有效期 + 可撤回；到期自动失效；
--       withdrawn 后在线访问立即拒绝；幂等键唯一。
CREATE TABLE assignment_shares (
    share_id uuid PRIMARY KEY,
    assignment_id uuid NOT NULL REFERENCES assignments (assignment_id),
    user_id uuid NOT NULL,
    data_categories jsonb NOT NULL,
    expires_at timestamptz NOT NULL,
    status text NOT NULL CHECK (status IN ('active', 'withdrawn')),
    withdrawn_at timestamptz,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT assignment_shares_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX assignment_shares_assignment_user_idx
    ON assignment_shares (assignment_id, user_id);
CREATE INDEX assignment_shares_expires_idx ON assignment_shares (expires_at);

GRANT SELECT, INSERT, UPDATE ON assignment_shares TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON assignment_shares TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON assignment_shares TO mgd_deletion_orchestrator;
