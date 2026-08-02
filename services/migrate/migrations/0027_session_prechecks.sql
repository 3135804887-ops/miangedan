-- TASK-027 输入模式与便利设置会前冻结（FR-015/FR-016；SCR-07）
-- 约束：每会话一行；冻结后不可修改（会前冻结）；数字人音视频始终开启无关闭选项；
-- 输入模式与便利设置为封闭枚举（应用层校验 + JSONB 数组）。

CREATE TABLE session_prechecks (
    session_id uuid PRIMARY KEY,
    input_modes jsonb NOT NULL,
    accommodations jsonb NOT NULL DEFAULT '[]'::jsonb,
    device_report jsonb NOT NULL,
    frozen boolean NOT NULL DEFAULT true,
    frozen_at timestamptz NOT NULL DEFAULT now(),
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT session_prechecks_modes_json CHECK (jsonb_typeof(input_modes) = 'array'),
    CONSTRAINT session_prechecks_accommodations_json CHECK (jsonb_typeof(accommodations) = 'array'),
    CONSTRAINT session_prechecks_device_json CHECK (jsonb_typeof(device_report) = 'object')
);

REVOKE UPDATE, DELETE ON session_prechecks FROM PUBLIC;
GRANT SELECT, INSERT ON session_prechecks TO mgd_app_runtime;
GRANT SELECT, INSERT ON session_prechecks TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON session_prechecks TO mgd_deletion_orchestrator;
