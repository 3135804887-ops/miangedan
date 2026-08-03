# 数据模型（DATA-MODEL）

| 字段 | 内容 |
|---|---|
| 文档编号 | DATA-001 |
| 版本 | 0.1.0（已批准 2026-08-01 规范评审） |
| 追踪 | PRD-001 "Key Data Entities"、"Recommended Technology Baseline"；US-01 ~ US-08；FR-001 ~ FR-040；NFR-005、NFR-006 |
| 一致性锚点 | `docs/domain/DOMAIN-MODEL.md`（实体语义事实源）、`ai/schemas/*.json`（内容结构）、`docs/domain/BILLING-STATE-MACHINE.md`（账本） |

## 1. 目的

给出面个蛋 PostgreSQL 物理数据模型：表清单、关键字段、不可变/追加式约束、索引与分区建议，以及 Redis、对象存储、事件流的用途边界，作为数据库迁移与服务实现的依据。

## 2. 范围

- 40 张核心业务表（含 TASK-010 的 4 张身份支撑表、TASK-012 上传/扫描、TASK-013 简历解析及 TASK-014 岗位解析/材料降级记录）、追加式账本约束、索引与分区策略。
- 对象存储桶划分、Redis 用途边界、区域事件流主题清单。

## 3. 非目标

- 不给出完整 DDL 脚本（实现阶段由迁移工具生成，本文为准入审查基线）。
- 不定义分析型数仓（首版聚合分析直接在区域内只读副本上执行，<10 人隐藏规则由服务层强制）。
- 不重复字段级业务语义（见 DOMAIN-MODEL 与 ai/schemas）。

## 4. 全局物理规则

1. **区域归属**：所有业务表含 `data_region char(4) NOT NULL CHECK (data_region IN ('cn','eu','intl'))`；每个数据区独立数据库集群，跨区无外键、无逻辑复制通道。
2. **追加式账本**：`evidence_items`、`score_versions`、`usage_ledger`、`access_audits` 四张表对应用角色 `REVOKE UPDATE, DELETE`；删除仅由删除编排使用专用受控角色执行并写审计。
3. **版本冻结表**：`resume_versions`、`job_versions`、`plan_versions` 对应用角色 REVOKE UPDATE（允许的唯一写路径是插入新版本行）。
4. **主键**：UUIDv7（时间有序，利于分区与索引局部性）。
5. **分区**：大表按 `(data_region, 月份)` 二级分区：`evidence_items`、`usage_ledger`、`access_audits`、`sessions`、`turns`。
6. **外键**：同区内启用；所有外键列与父表 `data_region` 一致性由 CHECK + 触发器保证。

## 5. 表清单

### 5.1 账户与授权族

| 表 | 用途 | 关键字段 | 主要索引 |
|---|---|---|---|
| `users` | 用户主体 | user_id PK、data_region、ui_language、interview_language_preference、age_status、status、条款/隐私/数据处理说明版本、registration_evidence_json、created_at | users(data_region, status) |
| `identities` | 登录身份（不保存邮箱/第三方 subject 明文） | identity_id PK、user_id FK、provider、provider_subject_hash、verified_at、data_region | identities(data_region, provider, provider_subject_hash) UNIQUE；identities(user_id) |
| `identity_verifications` | 邮箱/OAuth 短期验证与单次证明状态 | verification_id PK、provider、provider_subject_hash、code_hash、proof_hash、status、到期/消费时间、request_key、data_region | identity_verifications(data_region, provider, request_key) UNIQUE；主体限流与到期索引 |
| `identity_sessions` | 业务会话与刷新令牌轮换 | session_id PK、user_id FK、refresh_token_hash、status、access/refresh_expires_at、rotated_to_session_id、data_region | identity_sessions(refresh_token_hash) UNIQUE；identity_sessions(user_id, status) |
| `identity_conflicts` | 身份冲突恢复案件（绝不自动合并） | recovery_case_id PK、requesting/conflicting_user_id FK、provider、provider_subject_hash、status、data_region、created_at | identity_conflicts(requesting_user_id, created_at) |
| `identity_idempotency` | 身份写操作幂等结果引用 | idempotency_id PK、operation、idempotency_key、request_hash、result_ref、status、data_region | identity_idempotency(data_region, operation, idempotency_key) UNIQUE |
| `consent_grants` | 授权记录（版本化追加） | grant_id PK、user_id FK、consent_type、封闭 scope_json/scope_hash、status、granted/expires/withdrawn_at、证据 JSON/hash、version/supersedes_grant_id、request operation/key/hash、audit_id FK、data_region、recorded_at | consent_grants(user_id, consent_type, scope_hash, version) UNIQUE；consent_grants(data_region,user_id,request_operation,request_key) UNIQUE；consent_grants(status, expires_at) |

约束：`identities` 同一 `(data_region, provider, provider_subject_hash)` 唯一（防误合并）；验证/会话/幂等表禁止保存邮箱、验证码、OAuth 授权码、证明和令牌明文；`identity_conflicts` 对业务角色无删除能力；`consent_grants` 追加式（撤回插入新版本行），每个版本以 `(audit_id, data_region)` 绑定同事务写入的追加式 `access_audits`；scope/evidence 的键、枚举、版本格式与大小均为数据库闭集，不保存正文或敏感字段值。

### 5.2 材料族

| 表 | 用途 | 关键字段 | 主要索引 |
|---|---|---|---|
| `resume_uploads` | 简历隔离上传与当前扫描状态 | upload_id PK、user_id、data_region、idempotency_key、content_fingerprint、filename、size_bytes、status、object_bucket/key、rejection_reason、sandbox_attestation、created_at/updated_at | resume_uploads(data_region,user_id,idempotency_key) UNIQUE；resume_uploads(user_id,created_at) |
| `upload_scan_attempts` | 首次扫描与重试幂等执行记录 | attempt_id PK、upload_id FK、attempt_number、idempotency_key、status、failure_code、started_at/completed_at | upload_scan_attempts(upload_id,attempt_number) UNIQUE；upload_scan_attempts(upload_id,idempotency_key) UNIQUE |
| `resumes` | 简历主记录 | resume_id PK、upload_id FK UNIQUE、user_id、data_region、language、status、current_version、created_at/updated_at | resumes(user_id, updated_at) |
| `resume_parse_attempts` | 初次解析与步骤级重试记录 | task_id PK、resume_id FK、idempotency_key、input_fingerprint、status、provider/prompt_version、input_retained、retryable、failure_code、started_at/completed_at | resume_parse_attempts(resume_id,idempotency_key) UNIQUE |
| `resume_versions` | 简历版本（追加式冻结） | (resume_id, resume_version) PK、base_version、idempotency_key、operation_fingerprint、profile_json（符合 resume-profile schema）、excluded_sensitive_fields（仅类别）、low/reviewed paths、confirmed_by_user、created_at | resume_versions(resume_id,idempotency_key) UNIQUE；应用角色仅 SELECT/INSERT |
| `job_profiles` | 岗位主记录与原始输入引用 | job_id PK、user_id、data_region、language、source_kind、source_resume_id/version 或 uploads 区域桶 raw_text_ref（二选一）、create_idempotency_key、input_fingerprint、status、current_version、created_at/updated_at/deleted_at | job_profiles(data_region,user_id,create_idempotency_key) UNIQUE；job_profiles(user_id,updated_at) |
| `job_parse_attempts` | JD/岗位推导初次解析与步骤级重试 | task_id PK、job_id FK、idempotency_key、input_fingerprint、status、provider/prompt_version、input_retained、retryable、failure_code、started_at/completed_at | job_parse_attempts(job_id,idempotency_key) UNIQUE |
| `job_versions` | 岗位版本（追加式冻结） | (job_id,job_version) PK、base_version、idempotency_key、operation_fingerprint、profile_json（符合 job-profile schema）、excluded_from_scoring（仅类别）、confirmed_by_user、created_at | job_versions(job_id,idempotency_key) UNIQUE；应用角色仅 SELECT/INSERT |
| `user_resume_library` | 简历库（TASK-018） | (user_id, data_region, resume_id, resume_version) PK、company、job_title、saved_at | user_resume_library(user_id, data_region) |
| `user_job_library` | 岗位库（TASK-018） | (user_id, data_region, job_id, job_version) PK、company、job_title、saved_at | user_job_library(user_id, data_region) |
| `material_readiness_assessments` | 四种材料模式及用户可见影响快照（追加式） | assessment_id PK、user_id、data_region、resume_id/version、job_id/version、mode、consent_required、impact_snapshot_json、input_fingerprint、idempotency_key、created_at | material_readiness_assessments(data_region,user_id,idempotency_key) UNIQUE |
| `material_degradation_consents` | 非 full 模式明确同意（追加式功能确认，不替代六类隐私授权） | consent_grant_id PK、assessment_id FK UNIQUE、user_id、data_region、mode、accepted=true、impact_snapshot_json、operation_fingerprint、idempotency_key、granted_at | material_degradation_consents(data_region,user_id,idempotency_key) UNIQUE；应用角色仅 SELECT/INSERT |
| `process_sources` | 企业流程来源 | source_id PK、url（通用模板为空）、source_type（CHECK 白名单）、retrieved_at、credibility（CHECK）、expires_at、region（CHECK）、job_family、company/role/level（检索维度）、is_unofficial_experience、status（active/under_review/taken_down）、idempotency_key UNIQUE、data_region（CHECK 且与 region 相等） | process_sources(region, job_family, status)；process_sources(expires_at)（失效任务）；process_sources(data_region, url) UNIQUE WHERE url IS NOT NULL |

> `process_sources` 约束与迁移对应 `services/migrate/migrations/0002_process_sources.sql`（TASK-015）：
> 非追加式账本，允许状态流转（active → under_review → taken_down，版权投诉与下架）；
> 仅存结构化元数据（不含网页正文），来源内容不得进入评分证据（FR-008）；
> 同幂等键/同 (data_region, url) 唯一去重（NFR-006）。

### 5.3 项目与会话族

| 表 | 用途 | 关键字段 | 主要索引 |
|---|---|---|---|
| `interview_projects` | 面试项目聚合根 | project_id PK、user_id FK、data_region、interview_language、degraded_mode、status、current_round_sequence、assignment_id NULL、active_device_id NULL、created_at | interview_projects(user_id, status)；interview_projects(assignment_id) |
| `plan_versions` | 计划版本（冻结） | (project_id, plan_version) PK、plan_json（符合 interview-plan schema）、rubric_version、frozen、created_at | — |
| `sessions` | 实时会话 | session_id PK、project_id FK、round_sequence、attempt_id、kind（formal/formal_retry/practice）、status、room_provider_ref、active_device_id、billable_seconds、paused_at、paused_seconds、downgrade_status（none/prompted/accepted/rejected）、downgrade_prompt_id、text_degraded_at、end_reason、ended_at、created_at、updated_at | sessions(project_id, round_sequence, kind)；sessions(status, updated_at)（运营监控，仅匿名指标）；sessions(downgrade_status) |
| `turns` | 回合 | (session_id, turn_index) PK、project_id FK、status、question_id、frozen、created_at | turns(project_id, session_id) |
| `session_transcripts` | 双向字幕/转写（TASK-023） | transcript_id PK、session_id FK、turn_index、utterance_id、kind（partial/final）、text、language、confidence、revised_text、revision_id、revision_state（none/submitted/accepted/rejected）、revision_rejected_reason、frozen、created_at、updated_at | session_transcripts(session_id, utterance_id) UNIQUE；session_transcripts(session_id, turn_index, created_at) |
| `session_turns` | 回合冻结边界（TASK-023） | (session_id, turn_index) PK、frozen、frozen_at | — |
| `session_tool_events` | 岗位工具事件（TASK-024） | tool_event_id PK、session_id FK、tool_key（code_editor/whiteboard/case_materials/portfolio）、event_type（edit/run/annotate/submit）、content_ref、created_at | session_tool_events(session_id, tool_event_id) UNIQUE；session_tool_events(session_id, created_at) |
| `session_prechecks` | 会前检查冻结（TASK-027） | session_id PK、input_modes jsonb、accommodations jsonb、device_report jsonb、frozen、frozen_at | — |
| `evidence_items` | **追加式证据账本** | evidence_id PK、session_id FK、turn_index、project_id FK、round_sequence、attempt_id、evidence_json（符合 turn-evidence schema）、content_hash、recorded_at | evidence_items(session_id, turn_index) UNIQUE；evidence_items(project_id, attempt_id)；分区(data_region, 月份) |
| `evidence_events` | **追加式证据事件流水（TASK-026）** | evidence_id PK、session_id FK、turn_index、project_id FK、round_sequence、attempt_id、kind（question_played/answer/revision/tool_event）、event_id、payload_json、content_hash（SHA-256 64 位）、recorded_at | evidence_events(event_id) UNIQUE；evidence_events(session_id, turn_index, recorded_at)；evidence_events(project_id, attempt_id) |
| `score_versions` | **追加式评分版本** | score_id PK、project_id FK、round_sequence、attempt_id、score_version、result_json（符合 scoring-result schema）、evidence_snapshot_hash、supersedes_score_id NULL、computed_at | score_versions(project_id, attempt_id, score_version)；score_versions(idempotency_key) UNIQUE |
| `handoff_packages` | 跨轮交接（追加） | package_id PK、project_id FK、from_round_sequence、to_round_sequence、package_json（符合 handoff-package schema）、created_at | handoff_packages(project_id, to_round_sequence) |
| `practices` | 非评分练习 | practice_id PK、project_id FK、user_id FK、kind、content_json、started_at、ended_at | practices(project_id, ended_at)；**与正式证据链无外键关联** |
| `retry_attempts` | 正式重试 | attempt_id PK、project_id FK、round_sequence、source_attempt_id、status、created_at | retry_attempts(project_id, round_sequence) |

### 5.4 商业族

| 表 | 用途 | 关键字段 | 主要索引 |
|---|---|---|---|
| `entitlements` | 权益 | entitlement_id PK、user_id FK、kind（free_credit/project_pack/pro_subscription/topup_pack）、scope_json（project_id 等）、total_seconds、consumed_seconds、status、valid_from、valid_to | entitlements(user_id, kind, status) |
| `usage_ledger` | **追加式秒级账本** | entry_id PK、entitlement_id FK、user_id FK、project_id、round_sequence、entry_type（reserve/consume/release/refund/reversal）、seconds、reason、balance_after、idempotency_key、created_at | usage_ledger(entitlement_id, created_at)；usage_ledger(idempotency_key) UNIQUE；分区(data_region, 月份) |
| `quotes` | 报价 | quote_id PK、project_id FK、plan_version、amount、currency、tax_json、status、created_at | quotes(project_id, plan_version) |
| `billing_freezes` | 计费版本冻结（TASK-060） | project_id PK、quote_id FK、plan_version、frozen、frozen_at | — |
| `orders` | 订单 | order_id PK、user_id FK、quote_id、idempotency_key UNIQUE、status、amount、currency、provider、provider_txn_id NULL、created_at | orders(user_id, status)；orders(provider, provider_txn_id) |
| `payment_events` | 支付回调去重 | payment_event_id PK、provider、order_id FK、payload_hash、processed_at | payment_events(provider, payment_event_id) UNIQUE |
| `refunds` | 退款 | refund_id PK、order_id FK、amount、reason、status、approver_pair_json NULL、created_at | refunds(order_id)；refunds(status, created_at)（审批队列） |
| `subscriptions` | 订阅 | subscription_id PK、user_id FK、plan_code、status、period_start、period_end、auto_renew、carryover_seconds | subscriptions(user_id, status)；subscriptions(period_end, status)（到期任务） |

### 5.5 机构族

| 表 | 用途 | 关键字段 | 主要索引 |
|---|---|---|---|
| `organizations` | 机构租户 | org_id PK、data_region、name、status、seat_limit、created_at | organizations(data_region, status) |
| `org_members` | 机构成员与角色 | (org_id, user_id) PK、roles（owner/admin/instructor/privacy_auditor/finance/member 分离）、joined_at、left_at | org_members(user_id) |
| `assignments` | 训练任务 | assignment_id PK、org_id FK、template_json（禁改项由 CHECK/服务层强制）、deadline、status、created_at | assignments(org_id, status) |
| `assignment_members` | 任务-成员状态 | (assignment_id, user_id) PK、status（invited/accepted/not_started/in_progress/completed/exited）、completed_at、fault_flag | assignment_members(assignment_id, status) |
| `assignment_shares` | 按任务细粒度授权 | share_id PK、assignment_id FK、user_id FK、scope（radar/total_score/round_results/full_report/transcript/media 子集）、expires_at、withdrawn_at、grant_id FK | assignment_shares(assignment_id, user_id)；assignment_shares(expires_at)（到期失效任务） |

### 5.6 治理族

| 表 | 用途 | 关键字段 | 主要索引 |
|---|---|---|---|
| `access_audits` | **追加式访问审计** | audit_id PK、subject_type（user/org/staff/system）、subject_id、actor_id、actor_role、action、resource_type、resource_id、legal_basis（consent/break_glass/system）、created_at | access_audits(subject_id, created_at)；access_audits(actor_id, created_at)；分区(data_region, 月份) |
| `incidents` | 事故与破窗 | incident_id PK、kind（fault/break_glass/release/rollback/compensation）、severity、region、summary、timeline_json、postmortem_ref NULL、created_at | incidents(kind, severity, created_at) |
| `deletion_tasks` | 删除编排 | task_id PK、user_id FK、scope（account/project/resume/job）、status、progress_json（每存储层状态：database/cache/index/object_storage/backup/third_party）、created_at、completed_at | deletion_tasks(user_id, status)；deletion_tasks(status, created_at)（重试队列） |
| `export_tasks` | 导出编排（TASK-055） | task_id PK、user_id FK、scope（account/project）、project_id、status、progress_note、export_content_ref、training_marker（恒 true）、created_at | export_tasks(user_id, created_at)；export_tasks(idempotency_key) UNIQUE |

## 6. 对象存储桶划分（每区独立）

| 桶 | 内容 | 分类 | 规则 |
|---|---|---|---|
| `{region}-uploads` | 原始简历/JD 文件 | restricted | `quarantine/` 隔离前缀只供无网络一次性沙箱扫描；通过后移入 `accepted/`，解析器只读 accepted；安全拒绝删除隔离副本，扫描超时/暂时不可用保留以供重试；不直接对外暴露，仅签名 URL |
| `{region}-exports` | 用户导出物 | restricted | 短时效签名 URL；生成任务审计 |
| `{region}-media` | 明确授权的原始音视频 | restricted | **默认为空**；仅在 ConsentGrant 有效时写入；30 天生命周期自动过期；未成年用户禁止写入 |

## 7. Redis 用途边界

允许：短期会话状态、限流计数、分布式锁（单活动设备锁、预留原子化）、在线状态、验证码散列（带 pepper）。**禁止**：作为证据、分数、账本、授权的唯一存储；Redis 数据丢失不得导致业务证据丢失（NFR-005 证据 RPO=0 由 PostgreSQL/事件流保证）。

## 8. 区域事件流主题清单

| 主题 | 载荷 | 消费者 |
|---|---|---|
| `parse.jobs` | 解析任务（文件引用，不含正文快照） | 解析服务 |
| `scoring.requests` | 评分请求（scoring_request_id + idempotency_key） | 评分服务 |
| `report.jobs` | 报告模块生成/重试 | 报告服务 |
| `notification.outbox` | 邮件/站内通知 | 通知服务 |
| `deletion.tasks` | 删除编排步骤 | 删除执行器 |
| `compensation.jobs` | 故障返还/退款补偿 | 计费服务 |

所有主题按 `session_id`/`project_id` 分区保序；载荷不含简历正文、完整回答、原始媒体。

## 9. 关键规则（红线）

1. 四张追加式表数据库层 REVOKE UPDATE/DELETE；改分、改证据、改账目在物理层即不可能。
2. `data_region` 全表强制；跨区外键与跨区复制通道不存在。
3. 敏感字段（电话/邮箱/证件/地址/照片/保护属性）只存在于 `restricted` 隔离存储，不进入 `resume_versions.profile_json`、`evidence_items`、`handoff_packages`、`score_versions` 的内容字段。
4. `payment_events`、`usage_ledger.idempotency_key`、`score_versions.idempotency_key` 唯一约束是幂等的最后防线（NFR-006）。
5. `resume_uploads` 的 CHECK 强制对象桶等于 `{data_region}-uploads`；`upload_scan_attempts` 唯一键保证首次扫描与重试无重复副作用（TASK-012）。
6. `resume_versions` 根敏感键 CHECK 为服务层递归 SEC-040 门槛的数据库二次防线；已确认版本的 `low_confidence_paths` 必须为空，应用角色无 UPDATE/DELETE。解析读取器同时核对 upload/user/data_region，禁止跨区原件读取（TASK-013）。
7. `consent_grants` 只追加版本且业务角色无 UPDATE/DELETE；原始音视频最长 30 天、机构分享必须到期；撤回版本与 `access_audits` 同事务提交，在线判定只读取最新版本（TASK-011）。

## 10. 异常处理

| 异常 | 处理 |
|---|---|
| 并发编辑同一材料版本 | 乐观锁（current_version 条件更新）冲突返回 `conflict` |
| 分区维护失败 | 分区创建任务告警；写入自动落到默认分区并告警，不丢数据 |
| 追加式表误写尝试 | 数据库拒绝 + 安全告警（疑似绕过服务层） |
| 授权审计写入失败 | 整个授予/撤回事务回滚，在线权限保持原状态；同一幂等键可安全重试且不得重复版本或审计 |
| 删除编排部分失败 | deletion_tasks 记录逐层状态，可重试，用户可见真实进度 |

## 11. 验证方式

1. 迁移脚本通过 CI：约束存在性检查（REVOKE、UNIQUE、CHECK、分区）。基线迁移位于
   `services/migrate/migrations/`（`schema_migrations` + SHA-256 校验和，TASK-003）；
   `0010_identity_accounts.sql` 落地 TASK-010 用户、身份验证、会话、防误合并与幂等约束；
   `0011_consent_grants.sql` 落地 TASK-011 六类授权、追加式版本、同事务审计与区域内幂等约束；
   `0012_resume_uploads.sql` 落地 TASK-012 上传与扫描幂等状态表；
   `0013_resume_parsing.sql` 落地 TASK-013 解析尝试、追加式版本、幂等与敏感根键二次门槛；
   `0014_job_parsing.sql` 落地 TASK-014 岗位解析、追加式版本及材料降级同意门槛。
2. 服务层测试：尝试 UPDATE/DELETE 追加式表被拒；幂等键重复写入返回冲突或去重成功。
3. 与 `docs/domain/DOMAIN-MODEL.md` 实体覆盖核对（40 表 ↔ 全部实体）；与 `ai/schemas/` 内容字段命名抽样比对。
