-- TASK-082 禁止改分的系统级约束与破窗访问（FR-039；AGENTS.md §2）
-- 约束：后台无编辑分数/解锁表（本迁移不含任何 score 写表）；
--       break_glass 与 break_glass_reviews 只 INSERT/SELECT（状态由评审事件推导，
--       无 UPDATE/DELETE 授权）；破窗限定理由与时长并事后复核。
CREATE TABLE break_glass (
    glass_id uuid PRIMARY KEY,
    target_user_id uuid NOT NULL,
    reason text NOT NULL,
    duration_minutes integer NOT NULL CHECK (duration_minutes BETWEEN 1 AND 480),
    target_ref text,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    opened_by uuid NOT NULL,
    opened_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX break_glass_target_idx ON break_glass (target_user_id, opened_at);

CREATE TABLE break_glass_reviews (
    review_id uuid PRIMARY KEY,
    glass_id uuid NOT NULL REFERENCES break_glass (glass_id),
    reviewer_id uuid NOT NULL,
    decision text NOT NULL CHECK (decision IN ('approved', 'rejected')),
    note text,
    reviewed_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT break_glass_reviews_glass_unique UNIQUE (glass_id)
);

GRANT SELECT, INSERT ON break_glass, break_glass_reviews TO mgd_app_runtime;
GRANT SELECT, INSERT ON break_glass, break_glass_reviews TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON break_glass, break_glass_reviews
    TO mgd_deletion_orchestrator;
