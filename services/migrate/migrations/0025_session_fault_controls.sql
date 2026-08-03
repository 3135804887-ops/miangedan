-- TASK-025 故障暂停计时与文字降级（FR-020；INTERVIEW-STATE-MACHINE 5.2）
-- 追加式：sessions 表新增故障控制列；暂停秒数只增不减；data_region 强制。

ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS paused_at timestamptz,
    ADD COLUMN IF NOT EXISTS paused_seconds integer NOT NULL DEFAULT 0
        CHECK (paused_seconds >= 0),
    ADD COLUMN IF NOT EXISTS downgrade_status text NOT NULL DEFAULT 'none'
        CHECK (downgrade_status IN ('none', 'prompted', 'accepted', 'rejected')),
    ADD COLUMN IF NOT EXISTS downgrade_prompt_id uuid,
    ADD COLUMN IF NOT EXISTS text_degraded_at timestamptz,
    ADD COLUMN IF NOT EXISTS end_reason text
        CHECK (end_reason IN ('completed', 'user_exit', 'unrecoverable', 'downgrade_rejected')),
    ADD COLUMN IF NOT EXISTS ended_at timestamptz;

CREATE INDEX IF NOT EXISTS sessions_downgrade_status_idx
    ON sessions (downgrade_status);
