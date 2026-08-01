-- 面个蛋企业公开流程来源表（TASK-015）
-- 追踪：docs/data/DATA-MODEL.md 第 5.2 节；docs/domain/DOMAIN-MODEL.md 第 6.6 节；
-- PRD FR-007、FR-008；NFR-006（幂等）。
-- 说明：来源元数据仅含结构化字段（链接/日期/类型/可信度/失效状态），不含网页正文；
--       来源内容不得进入评分证据（FR-008）。本表非追加式账本，状态支持
--       active → under_review → taken_down（版权投诉与下架，US-08 规则 8）。
-- 幂等：同幂等键唯一；同 (data_region, url) 唯一（通用模板 url 为空不参与）。

CREATE TABLE process_sources (
    source_id uuid PRIMARY KEY,
    url text,
    source_type text NOT NULL
        CHECK (source_type IN ('official_careers_page', 'official_recruiting_content',
                               'credible_public_material', 'candidate_experience', 'generic_template')),
    retrieved_at timestamptz NOT NULL DEFAULT now(),
    credibility text NOT NULL CHECK (credibility IN ('high', 'medium', 'low')),
    expires_at timestamptz,
    region char(4) NOT NULL CHECK (region IN ('cn', 'eu', 'intl')),
    job_family text NOT NULL,
    company text,
    role text,
    level text,
    is_unofficial_experience boolean NOT NULL DEFAULT false,
    summary text,
    status text NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'under_review', 'taken_down')),
    idempotency_key text NOT NULL,
    data_region char(4) NOT NULL CHECK (data_region IN ('cn', 'eu', 'intl')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT process_sources_region_matches_data_region
        CHECK (region = data_region),
    CONSTRAINT process_sources_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX process_sources_region_job_family_status_idx
    ON process_sources (region, job_family, status);

CREATE INDEX process_sources_expires_at_idx
    ON process_sources (expires_at);

-- 同来源链接（同区域）不重复入库；通用模板（url 为空）不参与唯一约束。
CREATE UNIQUE INDEX process_sources_region_url_unique
    ON process_sources (data_region, url) WHERE url IS NOT NULL;

GRANT SELECT, INSERT, UPDATE ON process_sources TO mgd_app_runtime;
