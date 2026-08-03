-- TASK-081 模型/提示词/量表/工作流版本治理（FR-038；PROVIDER-ADAPTERS；
-- TASK-031 pinned 机制）
-- 约束：阶段封闭枚举（offline/shadow/canary/full），只能逐级推进；
--       灰度门槛=结构兼容+安全测试，放量门槛=影子/灰度指标通过；
--       项目版本固定唯一（活跃正式面试固定开始版本，回滚需无进行中会话）。
CREATE TABLE artifact_versions (
    version_id uuid PRIMARY KEY,
    asset_type text NOT NULL CHECK (asset_type IN ('model', 'prompt', 'rubric', 'workflow')),
    asset_key text NOT NULL,
    version text NOT NULL,
    stage text NOT NULL CHECK (stage IN ('offline', 'shadow', 'canary', 'full')),
    compatible boolean NOT NULL DEFAULT false,
    safety_tested boolean NOT NULL DEFAULT false,
    metrics_ok boolean NOT NULL DEFAULT false,
    deprecated boolean NOT NULL DEFAULT false,
    note text,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT artifact_versions_asset_unique UNIQUE (asset_type, asset_key, version)
);

CREATE INDEX artifact_versions_region_asset_stage_idx
    ON artifact_versions (data_region, asset_type, stage);

CREATE TABLE version_pins (
    project_id uuid PRIMARY KEY,
    asset_type text NOT NULL,
    asset_key text NOT NULL,
    version_id uuid NOT NULL REFERENCES artifact_versions (version_id),
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    pinned_at timestamptz NOT NULL DEFAULT now()
);

GRANT SELECT, INSERT, UPDATE ON artifact_versions, version_pins TO mgd_app_runtime;
GRANT SELECT, INSERT, UPDATE ON artifact_versions, version_pins TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON artifact_versions, version_pins
    TO mgd_deletion_orchestrator;
