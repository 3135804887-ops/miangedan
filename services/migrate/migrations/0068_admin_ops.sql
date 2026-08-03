-- TASK-080 运营后台（FR-037；SCREEN-SPEC SCR-17）
-- 约束：房间快照仅匿名会话编号与技术指标（无姓名/简历/回答/媒体）；
--       供应商停用必须记录原因；审计追加式（SELECT/INSERT only）。
CREATE TABLE ops_providers (
    provider_id text PRIMARY KEY,
    capability text NOT NULL
        CHECK (capability IN ('llm', 'asr', 'tts', 'avatar', 'search')),
    region char(4) NOT NULL CHECK (region IN ('cn', 'eu', 'intl')),
    status text NOT NULL CHECK (status IN ('active', 'ramping', 'disabled')),
    ramp_percent integer NOT NULL DEFAULT 0 CHECK (ramp_percent BETWEEN 0 AND 100),
    latency_p95_ms integer NOT NULL DEFAULT 0,
    error_rate numeric NOT NULL DEFAULT 0,
    circuit_breaker text NOT NULL CHECK (circuit_breaker IN ('closed', 'open', 'half_open')),
    note text,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ops_room_snapshots (
    snapshot_id uuid PRIMARY KEY,
    region char(4) NOT NULL CHECK (region IN ('cn', 'eu', 'intl')),
    anonymous_session_id text NOT NULL,
    state text NOT NULL,
    duration_seconds integer NOT NULL DEFAULT 0,
    fault_code text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ops_room_snapshots_region_created_idx
    ON ops_room_snapshots (region, created_at);

CREATE TABLE ops_region_status (
    region char(4) PRIMARY KEY CHECK (region IN ('cn', 'eu', 'intl')),
    online_rooms integer NOT NULL DEFAULT 0,
    queued_sessions integer NOT NULL DEFAULT 0,
    capacity integer NOT NULL DEFAULT 0,
    provider_health_json jsonb NOT NULL DEFAULT '[]',
    slo_json jsonb NOT NULL DEFAULT '{}',
    error_budget_burn numeric NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
);

GRANT SELECT, INSERT, UPDATE ON ops_providers, ops_room_snapshots, ops_region_status
    TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON ops_providers, ops_room_snapshots, ops_region_status
    TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON ops_providers, ops_room_snapshots, ops_region_status
    TO mgd_deletion_orchestrator;
