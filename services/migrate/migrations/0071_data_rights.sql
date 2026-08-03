-- TASK-083 数据权利请求与删除编排（FR-040；RETENTION-MATRIX 6.3）
-- 约束：六层真实进度（database/cache/search_index/object_storage/backups/
--       third_party）逐项 pending/in_progress/done/failed；级联删除可追踪
--       （cascade_json 逐项状态）；法定财务记录保留但解除内容关联；
--       失败可重试；幂等键唯一。
CREATE TABLE data_right_requests (
    request_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    request_type text NOT NULL
        CHECK (request_type IN ('delete', 'export', 'correct', 'withdraw')),
    target_type text NOT NULL CHECK (target_type IN ('project', 'resume', 'job', 'account')),
    target_id uuid NOT NULL,
    status text NOT NULL
        CHECK (status IN ('REQUESTED', 'VERIFYING', 'IN_PROGRESS', 'COMPLETED', 'FAILED')),
    progress_json jsonb NOT NULL DEFAULT '{}',
    cascade_json jsonb NOT NULL DEFAULT '{}',
    legal_retention_note text,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT data_right_requests_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX data_right_requests_user_status_idx ON data_right_requests (user_id, status);
CREATE INDEX data_right_requests_status_created_idx ON data_right_requests (status, created_at);

GRANT SELECT, INSERT, UPDATE ON data_right_requests TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON data_right_requests TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON data_right_requests TO mgd_deletion_orchestrator;
