-- TASK-055 数据导出与删除编排（RETENTION-MATRIX 6.3；FR-040，US-05 场景 5）
-- 约束：删除进度六层（database/cache/search_index/object_storage/backup/third_party）
--       逐项状态 pending/in_progress/done/failed，用户可见、失败可重试；
--       删除必须是真实删除或不可逆匿名化（禁止软删除冒充）；导出物带训练用途标记
--       （应用层强制，exports 桶为 restricted 隔离存储）。

CREATE TABLE deletion_tasks (
    task_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    target_type text NOT NULL CHECK (target_type IN ('project', 'resume', 'job', 'account')),
    target_id uuid NOT NULL,
    status text NOT NULL
        CHECK (status IN ('REQUESTED', 'VERIFYING', 'IN_PROGRESS', 'COMPLETED', 'FAILED')),
    progress_json jsonb NOT NULL DEFAULT '{}',
    legal_retention_note text,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT deletion_tasks_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX deletion_tasks_user_status_idx
    ON deletion_tasks (user_id, status);

CREATE INDEX deletion_tasks_status_created_idx
    ON deletion_tasks (status, created_at);

CREATE TABLE export_tasks (
    task_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    scope text NOT NULL CHECK (scope IN ('account', 'project')),
    project_id uuid,
    status text NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    progress_note text,
    export_content_ref text,
    training_marker boolean NOT NULL DEFAULT true,
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT export_tasks_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX export_tasks_user_created_idx
    ON export_tasks (user_id, created_at);

GRANT SELECT, INSERT, UPDATE ON deletion_tasks, export_tasks TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON deletion_tasks, export_tasks TO mgd_deletion_orchestrator;
