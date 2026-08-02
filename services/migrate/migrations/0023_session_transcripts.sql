-- TASK-023 双向字幕与转写修订（FR-018；realtime-events 7.3；NFR-005）
-- 约束：追加式；会话×utterance 唯一；回合冻结状态独立登记；data_region 强制。

CREATE TABLE session_transcripts (
    transcript_id uuid PRIMARY KEY,
    session_id uuid NOT NULL,
    turn_index integer NOT NULL CHECK (turn_index >= 1),
    utterance_id uuid NOT NULL,
    kind text NOT NULL CHECK (kind IN ('partial', 'final')),
    text text NOT NULL,
    language text NOT NULL DEFAULT '',
    confidence numeric(5,4) NOT NULL DEFAULT 0 CHECK (confidence BETWEEN 0 AND 1),
    revised_text text,
    revision_id uuid,
    revision_state text NOT NULL DEFAULT 'none'
        CHECK (revision_state IN ('none', 'submitted', 'accepted', 'rejected')),
    revision_rejected_reason text,
    frozen boolean NOT NULL DEFAULT false,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT session_transcripts_session_utterance_unique UNIQUE (session_id, utterance_id)
);

CREATE INDEX session_transcripts_session_turn_idx
    ON session_transcripts (session_id, turn_index, created_at);

CREATE TABLE session_turns (
    session_id uuid NOT NULL,
    turn_index integer NOT NULL CHECK (turn_index >= 1),
    frozen boolean NOT NULL DEFAULT false,
    frozen_at timestamptz,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, turn_index)
);

-- 业务角色无 UPDATE/DELETE（ADR-0004）：转录文本修订通过追加版本表达，禁止改写历史。
REVOKE UPDATE, DELETE ON session_transcripts, session_turns FROM PUBLIC;
GRANT SELECT, INSERT ON session_transcripts, session_turns TO mgd_app_runtime;
GRANT SELECT, INSERT ON session_transcripts, session_turns TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON session_transcripts, session_turns TO mgd_deletion_orchestrator;
