-- TASK-016：InterviewProject 聚合根与 PlanVersion 冻结（FR-009 ~ FR-011）。
-- 约束：plan_versions 只追加（应用角色仅 SELECT/INSERT）；计划确认后 frozen=true 不再可改。

CREATE TABLE interview_projects (
    project_id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    name text,
    interview_language text NOT NULL CHECK (interview_language IN ('zh-CN', 'en-US')),
    degraded_mode text NOT NULL CHECK (degraded_mode IN ('full', 'jd_only', 'resume_only', 'neither')),
    degraded_mode_consent_id uuid,
    resume_id uuid,
    resume_version integer CHECK (resume_version > 0),
    job_id uuid,
    job_version integer CHECK (job_version > 0),
    status text NOT NULL CHECK (status IN (
        'DRAFT', 'PARSING', 'MATERIAL_REVIEW', 'PARSE_FAILED',
        'PLAN_GENERATING', 'PLAN_REVIEW', 'PLAN_FAILED', 'READY',
        'IN_SESSION', 'SCORING', 'ROUND_PASSED', 'ROUND_FAILED',
        'PRACTICING', 'EVALUATION_INCOMPLETE', 'COMPLETED'
    )),
    current_round_sequence integer NOT NULL DEFAULT 0 CHECK (current_round_sequence BETWEEN 0 AND 5),
    plan_version integer CHECK (plan_version > 0),
    active_device_id text,
    assignment_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT interview_projects_user_region_fk
        FOREIGN KEY (user_id, data_region) REFERENCES users (user_id, data_region),
    CONSTRAINT interview_projects_resume_refs CHECK (
        (resume_id IS NULL AND resume_version IS NULL)
        OR (resume_id IS NOT NULL AND resume_version IS NOT NULL)
    ),
    CONSTRAINT interview_projects_job_refs CHECK (
        (job_id IS NULL AND job_version IS NULL)
        OR (job_id IS NOT NULL AND job_version IS NOT NULL)
    )
);

CREATE INDEX interview_projects_user_status_idx ON interview_projects (user_id, status);
CREATE INDEX interview_projects_assignment_idx ON interview_projects (assignment_id);

CREATE TABLE plan_versions (
    project_id uuid NOT NULL,
    plan_version integer NOT NULL CHECK (plan_version > 0),
    plan_json jsonb NOT NULL CHECK (plan_json ? 'schema_version'),
    rubric_version text NOT NULL,
    frozen boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, plan_version),
    CONSTRAINT plan_versions_project_fk
        FOREIGN KEY (project_id) REFERENCES interview_projects (project_id)
);

REVOKE UPDATE, DELETE ON plan_versions FROM PUBLIC;
GRANT SELECT, INSERT ON plan_versions TO mgd_app_runtime;
GRANT SELECT, INSERT ON plan_versions TO mgd_ledger_writer;
GRANT SELECT, INSERT, UPDATE, DELETE ON plan_versions TO mgd_deletion_orchestrator;
