-- TASK-024 岗位工具事件（FR-019；realtime-events 7.5；NFR-005）
-- 约束：追加式；工具类型/事件类型 CHECK；content_ref 对象存储引用；data_region 强制。

CREATE TABLE session_tool_events (
    tool_event_id uuid PRIMARY KEY,
    session_id uuid NOT NULL,
    tool_key text NOT NULL
        CHECK (tool_key IN ('code_editor', 'whiteboard', 'case_materials', 'portfolio')),
    event_type text NOT NULL CHECK (event_type IN ('edit', 'run', 'annotate', 'submit')),
    content_ref text NOT NULL DEFAULT '',
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT session_tool_events_session_tool_idx UNIQUE (session_id, tool_event_id)
);

CREATE INDEX session_tool_events_session_created_idx
    ON session_tool_events (session_id, created_at);

REVOKE UPDATE, DELETE ON session_tool_events FROM PUBLIC;
GRANT SELECT, INSERT ON session_tool_events TO mgd_app_runtime;
GRANT SELECT, INSERT ON session_tool_events TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON session_tool_events TO mgd_deletion_orchestrator;
