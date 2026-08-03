-- TASK-026 追加式证据事件（NFR-005；realtime-events 7.3/7.5；ADR-0004）
-- 约束：证据类型封闭枚举；event_id 幂等唯一；content_hash 非空；data_region 强制。
-- 与 0001 evidence_items 的关系：evidence_items 为按 (session, turn) 的评分快照，
-- evidence_events 为问题/回答/修订/工具事件的细粒度追加式流水，两者均只增不改。

CREATE TABLE evidence_events (
    evidence_id uuid PRIMARY KEY,
    session_id uuid NOT NULL,
    turn_index integer NOT NULL CHECK (turn_index >= 1),
    project_id uuid NOT NULL,
    round_sequence integer NOT NULL CHECK (round_sequence >= 1),
    attempt_id uuid,
    kind text NOT NULL
        CHECK (kind IN ('question_played', 'answer', 'revision', 'tool_event')),
    event_id uuid NOT NULL,
    payload_json jsonb NOT NULL,
    content_hash text NOT NULL CHECK (char_length(content_hash) = 64),
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT evidence_events_event_id_unique UNIQUE (event_id)
);

CREATE INDEX evidence_events_session_turn_idx
    ON evidence_events (session_id, turn_index, recorded_at);

CREATE INDEX evidence_events_project_attempt_idx
    ON evidence_events (project_id, attempt_id);

REVOKE UPDATE, DELETE ON evidence_events FROM PUBLIC;
GRANT SELECT, INSERT ON evidence_events TO mgd_app_runtime;
GRANT SELECT, INSERT ON evidence_events TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON evidence_events TO mgd_deletion_orchestrator;
