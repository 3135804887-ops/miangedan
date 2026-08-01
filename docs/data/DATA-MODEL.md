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

- 30 张核心业务表、追加式账本约束、索引与分区策略。
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
| `users` | 用户主体 | user_id PK、data_region、ui_language、age_status、status、created_at | users(data_region, status) |
| `identities` | 登录身份 | identity_id PK、user_id FK、provider、provider_subject、verified_at | identities(provider, provider_subject) UNIQUE；identities(user_id) |
| `consent_grants` | 授权记录（版本化追加） | grant_id PK、user_id FK、consent_type、scope_json、status、granted_at、expires_at、withdrawn_at、evidence_json、version | consent_grants(user_id, consent_type, scope_hash, version)；consent_grants(status, expires_at)（到期任务） |

约束：`identities` 同一 (provider, provider_subject) 全区唯一（防误合并）；`consent_grants` 追加式（撤回插入新版本行）。

### 5.2 材料族

| 表 | 用途 | 关键字段 | 主要索引 |
|---|---|---|---|
| `resumes` | 简历主记录 | resume_id PK、user_id FK、data_region、current_version、created_at、deleted_at | resumes(user_id, deleted_at) |
| `resume_versions` | 简历版本（冻结） | (resume_id, resume_version) PK、original_file_ref（对象存储键）、profile_json（符合 resume-profile schema）、parse_meta_json、confirmed_by_user、created_at | resume_versions(user_id 经 resumes  join) |
| `job_profiles` | 岗位主记录 | job_id PK、user_id FK、data_region、current_version、created_at、deleted_at | job_profiles(user_id, deleted_at) |
| `job_versions` | JD 版本（冻结） | (job_id, job_version) PK、raw_text_ref、profile_json（符合 job-profile schema）、parse_meta_json、confirmed_by_user、created_at | — |
| `process_sources` | 企业流程来源 | source_id PK、url、source_type、retrieved_at、credibility、expires_at、region、job_family、status | process_sources(region, job_family, status)；process_sources(expires_at)（失效任务） |

### 5.3 项目与会话族

| 表 | 用途 | 关键字段 | 主要索引 |
|---|---|---|---|
| `interview_projects` | 面试项目聚合根 | project_id PK、user_id FK、data_region、interview_language、degraded_mode、status、current_round_sequence、assignment_id NULL、active_device_id NULL、created_at | interview_projects(user_id, status)；interview_projects(assignment_id) |
| `plan_versions` | 计划版本（冻结） | (project_id, plan_version) PK、plan_json（符合 interview-plan schema）、rubric_version、frozen、created_at | — |
| `sessions` | 实时会话 | session_id PK、project_id FK、round_sequence、attempt_id、kind（formal/formal_retry/practice）、status、room_provider_ref、active_device_id、billable_seconds、created_at | sessions(project_id, round_sequence, kind)；sessions(status, updated_at)（运营监控，仅匿名指标） |
| `turns` | 回合 | (session_id, turn_index) PK、project_id FK、status、question_id、frozen、created_at | turns(project_id, session_id) |
| `evidence_items` | **追加式证据账本** | evidence_id PK、session_id FK、turn_index、project_id FK、round_sequence、attempt_id、evidence_json（符合 turn-evidence schema）、content_hash、recorded_at | evidence_items(session_id, turn_index) UNIQUE；evidence_items(project_id, attempt_id)；分区(data_region, 月份) |
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

## 6. 对象存储桶划分（每区独立）

| 桶 | 内容 | 分类 | 规则 |
|---|---|---|---|
| `{region}-uploads` | 原始简历/JD 文件 | restricted | 隔离桶；解析在沙箱完成；不直接对外暴露，仅签名 URL |
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

## 10. 异常处理

| 异常 | 处理 |
|---|---|
| 并发编辑同一材料版本 | 乐观锁（current_version 条件更新）冲突返回 `conflict` |
| 分区维护失败 | 分区创建任务告警；写入自动落到默认分区并告警，不丢数据 |
| 追加式表误写尝试 | 数据库拒绝 + 安全告警（疑似绕过服务层） |
| 删除编排部分失败 | deletion_tasks 记录逐层状态，可重试，用户可见真实进度 |

## 11. 验证方式

1. 迁移脚本通过 CI：约束存在性检查（REVOKE、UNIQUE、CHECK、分区）。基线迁移位于
   `services/migrate/migrations/`（`schema_migrations` + SHA-256 校验和，TASK-003）。
2. 服务层测试：尝试 UPDATE/DELETE 追加式表被拒；幂等键重复写入返回冲突或去重成功。
3. 与 `docs/domain/DOMAIN-MODEL.md` 实体覆盖核对（30 表 ↔ 全部实体）；与 `ai/schemas/` 内容字段命名抽样比对。
