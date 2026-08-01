-- TASK-018：用户材料库（简历库/岗位库）、语言偏好列与单活动设备锁（FR-028 ~ FR-030）。

CREATE TABLE user_resume_library (
    user_id uuid NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    resume_id uuid NOT NULL,
    resume_version integer NOT NULL CHECK (resume_version > 0),
    company text,
    job_title text,
    saved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, data_region, resume_id, resume_version),
    CONSTRAINT user_resume_library_user_fk
        FOREIGN KEY (user_id, data_region) REFERENCES users (user_id, data_region)
);

CREATE TABLE user_job_library (
    user_id uuid NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    job_id uuid NOT NULL,
    job_version integer NOT NULL CHECK (job_version > 0),
    company text,
    job_title text,
    saved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, data_region, job_id, job_version),
    CONSTRAINT user_job_library_user_fk
        FOREIGN KEY (user_id, data_region) REFERENCES users (user_id, data_region)
);

ALTER TABLE users
    ADD COLUMN interview_language_preference text
    CHECK (interview_language_preference IN ('zh-CN', 'en-US'));

-- 单活动设备锁由 interview_projects.active_device_id 承载（TASK-016 迁移已建列），
-- 无独立表；跨设备历史即项目列表/详情（interview_projects 已有 user_id/status 索引）。
