-- TASK-020：实时会话房间（FR-013、NFR-007、SEC-003）。
-- 媒体令牌为短期一次性状态（Redis），不入库；本迁移只建会话持久化表。

CREATE TABLE sessions (
    session_id uuid PRIMARY KEY,
    project_id uuid NOT NULL,
    round_sequence integer NOT NULL CHECK (round_sequence BETWEEN 1 AND 5),
    attempt_id uuid,
    kind text NOT NULL CHECK (kind IN ('formal', 'formal_retry', 'practice')),
    status text NOT NULL CHECK (status IN (
        'ROOM_CREATED', 'PRE_CHECK', 'AVATAR_CONNECTING', 'LIVE',
        'PAUSED_SYSTEM', 'RECONNECTING', 'DOWNGRADE_PROMPTED',
        'TEXT_DEGRADED', 'AUTH_PAUSED', 'ENDED'
    )),
    room_provider_ref text,
    active_device_id text,
    billable_seconds integer NOT NULL DEFAULT 0 CHECK (billable_seconds >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sessions_project_fk
        FOREIGN KEY (project_id) REFERENCES interview_projects (project_id)
);

CREATE INDEX sessions_project_round_kind_idx ON sessions (project_id, round_sequence, kind);
CREATE INDEX sessions_status_updated_idx ON sessions (status, updated_at);
