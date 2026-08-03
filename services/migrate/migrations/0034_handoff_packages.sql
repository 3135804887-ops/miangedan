-- TASK-034 跨轮交接包（HANDOFF-SPEC；docs/data/DATA-MODEL.md 5.3；ADR-0004 追加式）
-- 约束：package_json 必须符合 ai/schemas/handoff-package.schema.json（应用层 fail-closed 校验）；
--       to_round_sequence 2-5（第 1 轮无交接包）；每 (project, to_round) 只允许一份有效交接包；
--       data_region 强制；业务角色无 UPDATE/DELETE（前序 ScoreVersion 被复核更新时生成新行）。

CREATE TABLE handoff_packages (
    package_id uuid PRIMARY KEY,
    project_id uuid NOT NULL,
    from_round_sequence integer NOT NULL CHECK (from_round_sequence BETWEEN 1 AND 4),
    to_round_sequence integer NOT NULL CHECK (to_round_sequence BETWEEN 2 AND 5),
    package_json jsonb NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT handoff_packages_project_to_round_unique
        UNIQUE (project_id, to_round_sequence),
    CONSTRAINT handoff_packages_round_order_check
        CHECK (to_round_sequence = from_round_sequence + 1)
);

CREATE INDEX handoff_packages_project_created_idx
    ON handoff_packages (project_id, created_at);

REVOKE UPDATE, DELETE ON handoff_packages FROM PUBLIC;
GRANT SELECT, INSERT ON handoff_packages TO mgd_app_runtime;
GRANT SELECT, INSERT ON handoff_packages TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON handoff_packages TO mgd_deletion_orchestrator;
