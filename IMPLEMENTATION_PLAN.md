# 面个蛋（MianGeDan）实施计划

| 字段 | 内容 |
|---|---|
| 文档编号 | IMPL-PLAN-001 |
| 版本 | 0.1.0（已批准 2026-08-01 规范评审） |
| 创建日期 | 2026-08-01 |
| 唯一需求事实源 | [docs/prd/PRD-001-面个蛋-V1.0.md](docs/prd/PRD-001-面个蛋-V1.0.md) |

## 1. 目的

将 PRD-001 拆分为可执行、可追踪、可验收的 Epic 与任务，为后续 AI 开发代理和工程团队提供唯一的工作分解结构（WBS），并建立需求 ID（US / FR / NFR）到任务、测试的双向追踪。

## 2. 范围

- 覆盖 PRD 全部 In Scope 能力：账户、材料摄取、计划、实时数字人面试、评分、报告、训练、商业、机构、治理、三数据区上线验证。
- 覆盖 PRD 的 Phase 0–4 发布阶段，将阶段退出条件映射到 EPIC-10 的验证任务。

## 3. 非目标

- 不包含工期估算、团队规模与排期（PRD 规定不得以工期或团队规模削减范围）。
- 不选定最终商业供应商（属未决事项 OD-01，见第 7 节）。
- 不重复 PRD 已写明的业务规则正文，仅做追踪引用；规则冲突时以 PRD 为准。

## 4. 与 PRD 的追踪关系

### 4.1 需求总账

- 用户故事：US-01 至 US-08（8 项）。
- 功能需求：FR-001 至 FR-040（40 项，全部 P0）。
- 非功能需求：NFR-001 至 NFR-016（16 项）。
- 其他规则锚点：Scoring & Decision Model、Privacy/Security/AI Governance、Timeline & Milestones、Risks & Mitigation。

### 4.2 US → Epic 追踪

| US | 主题 | 主要 Epic | 关联 FR |
|---|---|---|---|
| US-01 | 提供简历与 JD | EPIC-02 | FR-001 ~ FR-006 |
| US-02 | 生成多轮面试计划 | EPIC-02、EPIC-04 | FR-007 ~ FR-012 |
| US-03 | 实时数字人面试 | EPIC-03 | FR-013 ~ FR-020、NFR-007 ~ NFR-012 |
| US-04 | 评分、复盘、训练与重试 | EPIC-05、EPIC-06 | FR-021 ~ FR-026 |
| US-05 | 登录与个人资产 | EPIC-02 | FR-027 ~ FR-030 |
| US-06 | 购买与额度 | EPIC-07 | FR-031 ~ FR-033 |
| US-07 | 机构训练 | EPIC-08 | FR-034 ~ FR-036 |
| US-08 | 运营与治理 | EPIC-09 | FR-037 ~ FR-040 |

### 4.3 FR → Epic 追踪

| FR 区间 | 内容摘要 | Epic |
|---|---|---|
| FR-001 ~ FR-006 | 简历/JD 上传、解析、敏感字段排除、缺失降级、文件安全 | EPIC-02 |
| FR-007 ~ FR-008 | 企业公开流程来源与可信度 | EPIC-02（服务）、EPIC-04（检索链路与安全） |
| FR-009 ~ FR-012 | 多轮计划、冻结、混合问题策略 | EPIC-02（计划服务）、EPIC-04（生成与安全过滤） |
| FR-013 ~ FR-020 | 实时数字人、输入模式、打断、字幕、工具、故障降级 | EPIC-03 |
| FR-021 ~ FR-026 | 双门槛审核、报告、练习、重试、复核、部分报告 | EPIC-05、EPIC-06 |
| FR-027 ~ FR-030 | 登录、多身份、语言、资产、单活动设备 | EPIC-02 |
| FR-031 ~ FR-033 | 报价、预留、秒级计费、支付、退款、故障返还 | EPIC-07 |
| FR-034 ~ FR-036 | 机构租户、任务、细粒度授权、小样本保护 | EPIC-08 |
| FR-037 ~ FR-040 | 运营后台、版本治理、禁止改分、数据权利与审计 | EPIC-09 |

### 4.4 NFR → Epic 追踪

| NFR | 目标 | Epic |
|---|---|---|
| NFR-001、NFR-002、NFR-003 | 可用性与有效完成率 | EPIC-01、EPIC-10 |
| NFR-004 | 每数据区 ≥3 可用区 | EPIC-01 |
| NFR-005、NFR-006 | 证据持久化、幂等 | EPIC-03、EPIC-07 |
| NFR-007、NFR-008、NFR-009、NFR-010、NFR-011、NFR-012 | 实时性能（建连、响应、打断、ASR、口型、画质） | EPIC-03、EPIC-10 |
| NFR-013、NFR-014、NFR-015、NFR-016 | 评分/报告/解析/计划生成时延 | EPIC-02、EPIC-05、EPIC-06、EPIC-10 |

## 5. Epic 与任务分解

> 约定：
> - 任务 ID 全局稳定，禁止复用已废弃 ID；任务只能在后续版本中追加，不得改写历史 ID 语义。
> - “依赖”列写阻塞性任务；未列出即仅依赖 Epic 内公共基础设施（EPIC-01）。
> - 验收要点是最低可验证条件；完整验收场景见 `docs/testing/ACCEPTANCE-MATRIX.md`。
> - 所有任务共享第 6 节全局完成定义（DoD）。

### EPIC-01 基础设施与数据区

目标：三数据区（`cn` / `eu` / `intl`）各自完整、隔离、可观测、可恢复的技术底座。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-001 | 单仓工程骨架、分支策略、CI（lint、单测、契约校验、安全扫描） | 全部间接 | — | CI 对 YAML/JSON/OpenAPI/Schema 变更自动校验 |
| TASK-002 | 三数据区环境拓扑与区域路由（区域独立命名空间、区域间默认无数据通路） | NFR-004；数据区规则 | TASK-001 | 区域配置错误时部署失败而非静默跨区 |
| TASK-003 | PostgreSQL、Redis、对象存储、区域事件流的基线部署与迁移工具 | 技术基线 | TASK-002 | 迁移可重复执行；Redis 不作为唯一证据存储 |
| TASK-004 | Temporal 集群与每区命名空间、任务队列划分 | 技术基线 | TASK-002 | 工作流跨可用区故障可恢复 |
| TASK-005 | OpenTelemetry 指标/追踪/结构化日志，内容默认脱敏；状态页骨架 | NFR 观测性 | TASK-002 | 日志扫描不含简历正文、完整回答、令牌 |
| TASK-006 | 密钥管理系统接入、敏感字段独立密钥与轮换 | 安全控制 | TASK-002 | 后台任何界面不展示完整密钥 |
| TASK-007 | 区域化邮件/通知通道与身份提供商接入点 | FR-027 | TASK-002 | 某区通道故障不影响其他区 |
| TASK-008 | 备份、持续增量、时间点恢复与季度恢复演练脚本 | 容灾 RPO/RTO | TASK-003 | 演练报告可重现；RPO 指标达标 |

> **任务状态（2026-08-01 更新）**：TASK-001（单仓骨架与 CI）与 TASK-002（三数据区环境拓扑与区域路由）
> 已完成；TASK-003（数据平台基线部署与迁移工具）已实现（迁移工具 `services/migrate`、
> `data-platform` 校验套件、数据平台模块契约）；TASK-004（Temporal 集群与每区命名空间、任务队列划分）
> 已实现（`services/temporal` 契约包、`infra/modules/temporal`、`temporal` 校验套件）；TASK-005
> （OpenTelemetry 观测与日志脱敏）已实现（`services/observability` 共享包、`infra/modules/observability`、
> `observability` 校验套件、日志脱敏政策与中英双语状态页骨架）；TASK-006（密钥管理）已实现
> （`services/secretref` 引用契约包、`infra/modules/secret-ref`、`key-mgmt` 校验套件与轮换演练、
> `docs/operations/KEY-ROTATION-RUNBOOK.md`）；TASK-007（通知与身份通道）已实现
> （`services/notify` 区域路由契约、`services/identity/provider` 提供商开放矩阵、
> `infra/modules/notification`、`channels` 校验套件）；TASK-008（备份与恢复）已实现
> （`services/backup` 契约包与一键恢复 CLI、`infra/modules/backup`、`backup` 校验套件、
> 恢复运行手册与季度演练模板）。**EPIC-01（基础设施与数据区）8 个任务全部完成**，
> EPIC-02 开工条件满足；后续 Epic 按依赖继续推进
> （实施细节见 `docs/architecture/EPIC-01-INFRA-DESIGN.md`）。

### EPIC-02 领域核心（身份、资产、材料、项目与计划）

目标：账户、授权、简历/JD 摄取、企业流程来源、面试项目与计划版本的确定性业务核心。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-010 | 用户、Identity、多登录方式（邮箱验证码、Google、Apple、微信）与多身份绑定/防误合并 | US-05、FR-027 | EPIC-01 | US-05 场景 4 通过；身份绑定双侧验证 |
| TASK-011 | ConsentGrant 授权中心：六类独立授权、撤回即时生效、授权证据 | FR-040、隐私规则 | TASK-010 | 撤回后在线访问立即失效并写审计 |
| TASK-012 | 简历上传与恶意文件隔离检测（病毒、宏、压缩炸弹、伪装、超限） | FR-001、FR-006 | TASK-003 | US-01 场景 4 通过；拒绝原因具体 |
| TASK-013 | 简历结构化解析、低置信度标记、逐字段编辑确认、敏感字段排除 | FR-002、FR-003 | TASK-012 | 电话/邮箱/证件/地址/照片/保护属性不进入面试上下文 |
| TASK-014 | JD 粘贴解析、AI 推理标记、人工校对、缺失降级弹窗与同意 | FR-004、FR-005 | TASK-013 | US-01 场景 3 通过；推理内容可编辑且有标记 |
| TASK-015 | 企业公开流程来源服务：检索、来源元数据（链接/日期/类型/可信度/失效）、通用模板回退 | FR-007、FR-008 | TASK-003 | 无可靠来源自动回退通用模板并标记 AI 推导 |
| TASK-016 | InterviewProject、PlanVersion、RoundConfig 服务与冻结规则 | FR-009、FR-010、FR-011 | TASK-014、TASK-015 | 开始后量表/权重/轮次不可改；不完整计划禁止开始 |
| TASK-017 | 项目状态机 Temporal 工作流（草稿→…→全部完成，含异常分支） | US-01~US-05 状态 | TASK-016 | 与 `docs/domain/INTERVIEW-STATE-MACHINE.md` 逐条一致 |
| TASK-018 | 简历库、岗位库、项目筛选、跨设备历史、单活动设备锁/安全转移与中英文界面/面试语言独立配置 | FR-028、FR-029、FR-030 | TASK-016 | US-05 场景 2、3 通过 |

> **任务状态（2026-08-01 更新）**：TASK-010 已实现：`services/identity` 提供邮箱验证码、
> Google/Apple/微信供应商中立登录、短期单次验证证明、业务/刷新令牌与轮换、账户偏好、
> 双侧验证身份绑定及冲突恢复案件（绝不自动合并）；邮箱投递复用所属数据区 `services/notify`，
> OAuth 复用 TASK-007 提供商开放矩阵，所有身份密钥仅以 `*_REF` 解析。OpenAPI 使用
> `/v1/identity/*` 前缀，数据库迁移为 `0010_identity_accounts.sql`；正常、异常、幂等、并发、
> 跨区和 US-05 场景 4 均有自动化测试。TASK-011 可基于该账户与业务令牌契约开工。

> **任务状态（2026-08-01 更新）**：TASK-011 已实现：`services/consent` 复用 TASK-010 业务令牌，
> 提供核心服务、原始音视频、机构共享、非必要产品分析、模型训练/研究、营销通知六类独立授权；
> scope 为封闭结构，model_training 默认关闭，非必要授权拒绝/撤回不影响 core_service。授予与撤回
> 均追加版本并与 content-free AccessAudit 同事务提交，撤回成功返回后同步在线判定立即拒绝；审计
> 失败全事务回滚并可用同一幂等键安全重试。API 仅使用 `/v1/consent/*`，迁移为
> `0011_consent_grants.sql`；关键路径提供 content-free 低基数指标/追踪/结构化日志端口；正常、异常、
> 并发幂等、未成年原始音视频拒绝、观测故障隔离及撤回即时失效均有自动化测试。

> **任务状态（2026-08-01 更新）**：TASK-012 已实现：`services/ingestion` 提供区域 uploads
> quarantine/accepted 隔离、一次性无网络沙箱证明门槛、病毒/宏/压缩炸弹/伪装/超限/损坏/加密
> 逐项扫描、具体拒绝原因与上传/重试两级幂等；扫描超时或扫描器暂时不可用时保留隔离原件并可只重试
> 失败步骤。数据库迁移为 `0012_resume_uploads.sql`，API 使用 `/v1/uploads/*` 前缀。

> **任务状态（2026-08-01 更新）**：TASK-013 已实现：`ai/services/parsing` 提供
> `ResumeParsingProvider.parse_resume` 供应商中立契约与合成桩、L1/L2/L4 分层输入、逐字段置信度、
> add/replace/remove/confirm 追加式校对版本及低置信度未清零禁止最终确认；电话/邮箱/证件/地址/照片/
> 保护属性经解析前脱敏、模型输出后清洗、Schema/版本写入前扫描、上下文/评分材料组装前扫描四道
> SEC-040 门槛，合成评测泄露命中为 0。解析连续暂时失败保留 uploads/accepted 原件并只重试失败步骤。
> API 使用 `/v1/parsing/*`，迁移为 `0013_resume_parsing.sql`。

> **任务状态（2026-08-01 更新）**：TASK-014 已实现：完整 JD 粘贴与保留、供应商中立结构化解析、
> 薪资福利/公司福利/招聘联系人逐层排除、AI 推理不可移除来源标记与追加式人工校对确认；
> `full/jd_only/resume_only/neither` 返回冻结影响弹窗，非 full 必须有严格匹配的显式同意记录。
> 仅简历模式只读已确认安全画像并生成全部带 AI 标记的可编辑岗位画像；超时/暂时失败保留输入
> 可只重试解析。数据库迁移为 `0014_job_parsing.sql`，API 仅扩展 `/v1/jobs*`。

> **任务状态（2026-08-01 更新）**：TASK-015（企业公开流程来源服务）已实现：
> `services/source`（Go 控制面单模块，登记 go.work 与 CI golangci 矩阵）——来源领域模型与
> 可信度/优先级规则（官方 > 官方内容 > 可信公开 > 经验；候选人经验强制标记非官方）、
> `search` 供应商中立适配层契约 + 合成桩（TASK-030 未开工前不绑定厂商 SDK，PROVIDER-ADAPTERS §4.5）、
> 检索编排（可重试错误退避重试 ≤2 次、幂等键去重）、无公司/断网/无可信来源自动回退通用模板并标记
> AI 推导（`flow_uses_generic_template`/`ai_derived`）、`store` 幂等存储抽象；
> 新增追加式迁移 `services/migrate/migrations/0002_process_sources.sql`（幂等键/URL 唯一、data_region 强制）；
> 契约同步：openapi.yaml 新增 `/v1/sources/*`（search/list/get）、DOMAIN-MODEL §6.6、DATA-MODEL §5.2、
> ACCEPTANCE-MATRIX FR-007/FR-008、CHANGELOG；合成来源样例扩展
> （`fixtures/synthetic/process-sources/`，synthetic: true）；含正常/异常/幂等/重试/注入按数据处理的单测。

> **任务状态（2026-08-01 更新）**：TASK-016 已实现：`services/project` 提供 InterviewProject /
> PlanVersion / RoundConfig 应用服务（创建/查询/列表/重命名/删除/复制、计划编辑与确认冻结）；
> 冻结规则（FR-011）：确认后量表/维度权重/轮次权重/轮次列表/问题覆盖方案冻结，项目进入 READY，
> 开始后编辑返回 `state_conflict`；不完整计划（缺覆盖方案或量表）确认返回 `plan_incomplete`（422）。
> 轮次边界（1-5 轮、10-60 分钟）与类型注册按 `config/interview-flows/v1/default.yaml` 实时校验
> （FR-009/FR-010）。`services/project/httpapi` 按 `docs/api/openapi.yaml` 的 `/v1/projects`、
> `/v1/projects/{projectId}/plan` 契约暴露（`plan:generate` 由 TASK-033 落地，当前 501 占位；
> company/job_title 筛选随 TASK-018；计费版本冻结随 TASK-060/061）。迁移为
> `0016_interview_projects.sql`（interview_projects + plan_versions 追加式约束）；
> 正常/异常/幂等/冻结与 HTTP 层测试齐备。

> **任务状态（2026-08-01 更新）**：TASK-017 已实现：`workflows/` 独立 Go 模块
> （go.temporal.io/sdk v1.47.0，登记 go.work 与 CI 六阶段循环 + golangci 独立作业）。
> `statemachine` 为确定性项目状态机引擎，与 `INTERVIEW-STATE-MACHINE.md` 5.2 迁移表逐条一致
> （15 状态 × 22 事件 + `project.ended_by_user` 终态分支，重放安全）；
> `workflow.ProjectWorkflow` 消费 `project.command` 信号、暴露 `project.state` 查询，每次迁移
> 追加式写审计与状态快照活动（契约桩），全部必需轮次通过自动触发 `project.all_rounds_passed`；
> `cmd/worker` 以 `interview` 队列与 `mgd-{region}-{env}-temporal` 命名空间运行（ADR-0001）。
> 测试：状态机迁移表逐行 + 非法/终态；工作流 testsuite 全旅程、失败分支、非法迁移与重试，
> 断言每次迁移写审计。CI 循环扩展为 `services/*/ workflows/`，golangci 新增 workflows 作业。

> **任务状态（2026-08-01 更新）**：TASK-018 已实现：`services/project` 扩展用户材料库
> （简历库/岗位库引用 + company/job_title 筛选元数据，`/v1/library/*`）、界面语言与面试语言独立偏好
> （`/v1/me/preferences`，FR-028）、项目列表公司/岗位筛选（FR-029）、正式面试单活动设备锁
> （claim/transfer/release；第二台设备被拒 `device_active`，确认安全转移后原设备会话失效，FR-030、
> US-05 场景 3）。迁移为 `0018_user_library_preferences.sql`（材料库两表 + users
> `interview_language_preference` 列）；openapi 契约、DOMAIN-MODEL §6.18、DATA-MODEL 材料族同步；
> 服务/HTTP 层正常、异常、幂等、设备锁测试齐备。

> **任务状态（2026-08-02 更新）**：TASK-020 已实现：`services/room`（新 Go 模块，登记 go.work 与 CI 矩阵）
> 提供会话房间创建/查询/结束/重连/设备转移（openapi `/v1/sessions/*` 契约）：
> 前置校验项目 READY + 本轮量表/覆盖方案就绪（FR-011）+ 单活动设备（TASK-018 ClaimDevice）；
> 短期媒体令牌（HMAC-SHA256、分钟级 TTL、一次性 nonce、按 nonce 吊销，与业务令牌隔离，SEC-003）；
> 3 分钟重连窗口（超窗 `reconnect_expired` → ENDED）、设备安全转移（原令牌立即失效）；
> `Provider` 供应商中立房间适配桩（ADR-0003）；交接包（TASK-034）与额度预留（TASK-061）为后续挂接点。
> 迁移为 `0020_sessions.sql`；DOMAIN-MODEL §6.19（RoomToken）、DATA-MODEL sessions 同步；
> 服务/HTTP 正常、异常、幂等、令牌一次性/吊销、重连窗口、设备转移测试齐备。

> **任务状态（2026-08-02 更新）**：TASK-030 已实现：`services/provider` 共享 Go 包（仅依赖
> `services/region`，零外部依赖）落地 PROVIDER-ADAPTERS §5~§9 治理骨架——五类能力（LLM/ASR/TTS/
> Avatar/Search）枚举与 `Info` 注册条目（provider_id 形态 `{capability}_{region}_{role}`、版本固定）、
> 按数据区隔离注册表（重复拒绝、紧急停用）、低频合成探针健康检查、每（区×供应商×能力）熔断器
> （closed → open → half_open → closed，注入时钟可测）、新会话区域路由（主 open 切 secondary、
> 主备不可用拒绝新会话、跨区不回退）、活跃正式面试会话钉扎（`Pin`/`Resolve`，被停用/版本变化返回
> `ErrPinnedUnavailable`，不静默切换）。熔断/路由/钉扎/校验测试齐备；能力适配器实现随对应任务接入。

> **任务状态（2026-08-02 更新）**：TASK-021 已实现：`services/avatar` 落地数字人驱动接入骨架
> （FR-014）——固定授权写实 2D 角色库（`CharacterLibrary`，授权凭证引用；未知角色拒绝，
> 禁止每场生成新脸）、动态面试官人格（`Persona` style_parameters 封闭枚举，越界拒绝）、
> 驱动契约 `Driver.Start/Drive/Stop`（口型预算 200ms，NFR-011；默认 720p/24fps，NFR-012）、
> 合成桩驱动；`RegisterDriver` 注册 `avatar_{region}_{role}` 至 TASK-030 注册表（版本固定，
> 主备路由 + 熔断）。角色库/人格/口型预算/注册路由测试齐备；真实媒体驱动随供应商选型接入。

> **任务状态（2026-08-02 更新）**：TASK-022 已实现：`services/asr` 落地流式语音识别接入骨架
> （FR-017、NFR-008~NFR-010）——双向流式契约 `Provider.OpenStream`（音频帧 → partial/final，
> 合成桩；语言/静音断点 fail-closed）、回合检测 `TurnDetector`（静音窗口断点 → final，
> 断点→final 预算 1s）、单说话方闸门 `TurnGate`（避免重叠说话；语音/按钮打断，
> 打断→停止预算 500ms）；`RegisterProvider` 注册 `asr_{region}_{role}` 至 TASK-030 注册表
> （版本固定、主备路由 + 熔断）。回合/打断/预算/注册路由测试齐备；真实 ASR 随供应商选型接入。

> **前端交付追踪（frontend-global-pages / frontend-batch-0）**：已建立 `apps/web` 与共享前端
> 工作区基线，提供 SCR-01 ~ SCR-16 的 `/{locale}` 路由壳、SCR-17 后台路由契约占位、
> FR-028 双语 URL 路由、项目/会话状态枚举契约断言、合成 Mock_Layer、WCAG 2.2 AA 自动化
> 基线以及阶段 2/3/6 前端 CI 门禁。批次 0 不实现真实业务服务、媒体链路或评分逻辑；后续
> frontend-batch-1 ~ 4 分别落地 SCR-01 ~ 07、SCR-08/09、SCR-10 ~ 15、SCR-16/17。
> （frontend-batch-0、SCR-01 ~ SCR-17、FR-028、NFR-006）

> **前端交付追踪（frontend-batch-1~4 完全重构，2026-08-02）**：路由壳已全部替换为真实业务页面——
> 应用级设计系统升级（品牌靛蓝→紫渐变、分层表面、阴影层级、圆角、动效、打印样式），
> `@mgd/ui` 新增 34 个内联 SVG 图标与 Card/PageHeader/AppShell/Tabs/Toast/Progress/
> EmptyState/StatCard/DataTable/Avatar/Tint 组件；i18n 扩展至 626 键 × 2 语言；
> Mock_Layer 覆盖身份/工作台/计划/会话/结果/报告/练习/资产/设置/购买/机构/后台全部页面组。
> SCR-01~17 全部页面实现：落地页（品牌 hero/特性/样例演示）、邮箱验证码+第三方登录、
> 工作台（统计/筛选/状态机徽标/操作）、创建项目（双栏上传+JD+样例填充）、解析校对
> （低置信度/敏感字段排除/缺失降级同意）、计划（轮次编辑/覆盖方案/冻结/报价）、会前检查
> （设备检测/便利设置/输入开关）、实时房间（字幕修订/工具区/控制栏 + SCR-09 故障暂停/
> 重连/降级/退出覆盖层）、轮次结果三态、完整报告（SVG 雷达+表格等价+逐题证据）、练习
> （不改分标识）、资产四分区、账户与隐私（六类授权中心/导出删除）、购买额度（报价/流水/
> 订单/自动续费）、机构端 7 页、运营后台 7 分区（默认脱敏/无改分控件）。
> 页面级测试（工作台筛选/房间红线文案与覆盖层/状态徽标）新增，前端 100 测试全绿；
> ESLint（含领域状态字面量与令牌硬编码门禁）、TypeScript strict、i18n 键门禁、
> 令牌门禁、生产构建全部通过。（frontend-batch-1~4、SCR-01~17、FR-028、NFR-006）

### EPIC-03 实时链路（房间、媒体、数字人、证据管道）

目标：低延迟、可恢复、证据完整的实时面试链路；控制面与媒体面分离。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-020 | WebRTC/SFU 房间、短期房间令牌、会话单活动设备绑定 | FR-013、NFR-007 | TASK-016 | 95% 建连 ≤8s；媒体令牌与业务令牌隔离 |
| TASK-021 | 数字人驱动接入（固定授权 2D 角色库、动态人格、口型同步） | FR-013、FR-014、NFR-011、NFR-012 | TASK-020、TASK-030 | 口型偏差 ≤200ms；禁止每场生成新脸 |
| TASK-022 | 流式 ASR、回合检测、自然打断与停止按钮、避免重叠说话 | FR-017、NFR-008~NFR-010 | TASK-020、TASK-030 | 打断至停止发声 P95 ≤500ms |
| TASK-023 | 双向字幕、ASR 临时/最终文本、修订窗口与原始/修订双版本 | FR-018 | TASK-022 | 下一主问题后正式回答冻结不可改写 |
| TASK-024 | 岗位工具（代码、白板、案例、作品集）与工具事件证据 | FR-019 | TASK-020 | 未配置工具禁止临时加载；事件入证据账本 |
| TASK-025 | 故障暂停计时、3 分钟重连窗口、文字降级询问与额度联动 | FR-020 | TASK-020、TASK-061 | US-03 场景 4 通过；拒绝降级=评估未完成且返还 |
| TASK-026 | 追加式证据账本写入管道（问题实际播放内容、回答、修订、工具事件） | NFR-005 | TASK-003 | 下一主问题前完成上一有效回答持久化；无更新/删除路径 |
| TASK-027 | 输入模式（语音/文字/摄像头/工具）与便利设置会前冻结 | FR-015、FR-016 | TASK-020 | 摄像头/麦克风可关，数字人音视频始终开启 |

> **任务状态（2026-08-02 更新）**：TASK-023 已实现：`services/room` 扩展双向字幕与转写修订
> （FR-018）——ASR 临时/最终文本追加（partial 仅展示不入证据；final 为正式文本）、修订状态机
> （none → submitted → accepted/rejected）、回合冻结边界（`turn.completed` 后修订一律
> `rejected(window_closed)`，原始 ASR 仅诊断、修订文本为评分证据）、按 `revision_id` 幂等与
> 冻结前持久化顺序保证（NFR-005）。API 为 `/v1/sessions/{id}/transcripts`、
> `/v1/sessions/{id}/revisions`、`/v1/sessions/{id}/turns/{turnIndex}/freeze`；
> 迁移为 `0023_session_transcripts.sql`；DOMAIN-MODEL §6.20、DATA-MODEL、openapi、
> realtime-events 同步；服务/HTTP 正常、异常、窗口竞态、幂等测试齐备。（TASK-023、FR-018、NFR-005）

> **任务状态（2026-08-02 更新）**：TASK-024 已实现：`services/room` 扩展岗位工具
> （FR-019）——四类工具（code_editor/whiteboard/case_materials/portfolio）封闭枚举、
> 激活校验（仅计划中已配置工具，正式房间不临时加载）、工具事件（edit/run/annotate/submit）
> 以 `tool_event_id` 幂等入证据账本（content_ref 对象存储引用，非内联大对象）。
> API 为 `/v1/sessions/{id}/tools`、`/v1/sessions/{id}/tools/{toolKey}/activate`、
> `/v1/sessions/{id}/tools/{toolKey}/events`；迁移为 `0024_session_tool_events.sql`；
> DOMAIN-MODEL §6.21、DATA-MODEL、openapi、realtime-events 同步；
> 服务/HTTP 正常、异常、未配置拒绝、幂等测试齐备。（TASK-024、FR-019、NFR-005）

> **任务状态（2026-08-02 更新）**：TASK-025 已实现：`services/room` 扩展故障控制
> （FR-020）——暂停计时（`timer.paused/resumed`，LIVE → PAUSED_SYSTEM/AUTH_PAUSED/RECONNECTING，
> 暂停段不计费不判失败，`paused_seconds` 只增不减）、数字人持续故障降级询问
> （`avatar.downgrade_prompted` → prompt_id 幂等）、接受降级（TEXT_DEGRADED，故障点起不计
> 数字人额度——TASK-061 挂接点）、拒绝降级（ENDED + EVALUATION_INCOMPLETE 语义 +
> 系统责任全额返还挂接 + 设备释放）。3 分钟重连窗口由 TASK-020 提供。API 为
> `/v1/sessions/{id}/timer/*`、`/v1/sessions/{id}/downgrade/*`；迁移为
> `0025_session_fault_controls.sql`；DOMAIN-MODEL §6.22、DATA-MODEL、openapi、
> INTERVIEW-STATE-MACHINE 同步；服务/HTTP 正常、异常、幂等测试齐备。
> （TASK-025、FR-020、NFR-005）

> **任务状态（2026-08-02 更新）**：TASK-026 已实现：`services/evidence` 新 Go 模块
> （仅依赖 services/region，登记 go.work 与 CI golangci 矩阵）提供追加式证据账本写入管道
> （NFR-005）——问题实际播放内容/回答/修订/工具事件四类证据（kind 封闭枚举）、`event_id`
> 幂等去重（NFR-006）、`content_hash` 与载荷一致性校验（fail-closed）、无更新/删除路径
> （ADR-0004）、列表只读副本；`turn.completed` 前完成上一有效回答持久化的顺序保证由
> TASK-023 冻结边界消费。迁移为 `0026_evidence_events.sql`；DOMAIN-MODEL §6.23、
> DATA-MODEL、realtime-events 同步；正常/异常/幂等/只读红线测试齐备。
> （TASK-026、NFR-005、NFR-006）

> **任务状态（2026-08-02 更新）**：TASK-027 已实现：`services/room` 扩展会前检查
> （FR-015、FR-016）——输入模式（voice/text/camera/job_tool 封闭枚举）与便利设置
> （对齐 project.Accommodations 计划冻结枚举）会前冻结（`session.pre_check_passed` →
> AVATAR_CONNECTING）；摄像头/麦克风可关不扣分、数字人音视频始终开启（无关闭选项）、
> 冻结后不可修改（幂等键重放返回首次结果）；设备报告（camera/mic/网络评级）校验
> fail-closed。API 为 `/v1/sessions/{id}/precheck/freeze`、`/v1/sessions/{id}/precheck`；
> 迁移为 `0027_session_prechecks.sql`；DOMAIN-MODEL §6.24、DATA-MODEL、openapi、
> SCREEN-SPEC 同步；服务/HTTP 正常、异常、重复冻结、幂等测试齐备。
> （TASK-027、FR-015、FR-016）

### EPIC-04 AI 编排（供应商适配、提示词、面试官图、交接、安全）

目标：概率性 AI 决策与确定性业务状态严格分离；LLM 无权直接改变业务状态。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-030 | 供应商适配层：LLM/ASR/TTS/数字人/搜索统一能力接口、健康检查、熔断、区域路由、版本固定 | 技术基线、FR-037 部分 | EPIC-01 | 按 `docs/ai/PROVIDER-ADAPTERS.md` 契约；活跃正式会话不静默切换 |
| TASK-031 | 提示词注册表：分层、版本、输入隔离、结构化输出 Schema 校验 | FR-038 部分 | TASK-030 | 按 `ai/prompts/README.md`；提示词不可直接产出最终评分 |
| TASK-032 | LangGraph 面试官决策图：覆盖点推进、动态追问、打断策略、工具使用 | FR-012 | TASK-030、TASK-031 | 追问不越出已确认范围；检查点可恢复 |
| TASK-033 | 计划生成链路：来源融合、轮次建议、安全过滤与重新生成 | FR-009、FR-011 | TASK-015、TASK-031 | US-02 场景 5 通过；不安全内容不进入房间 |
| TASK-034 | 跨轮交接包生成、上下文压缩与事实完整性校验 | 跨轮交接规则 | TASK-016、TASK-031 | 按 `docs/ai/HANDOFF-SPEC.md`；禁止重复问题清单生效 |
| TASK-035 | 提示注入防护与内容安全管道（简历/JD/网页均视为不可信数据） | P0 风险（注入） | TASK-030 | 红队注入用例全部阻断；模型无密钥访问 |
| TASK-036 | AI 评测框架：黄金集、回归门槛、公平性切分 | 评分硬门槛 | TASK-031 | `ai/evals/` 数据集可重复运行并产出报告 |

> **任务状态（2026-08-02 更新）**：TASK-031 已实现：`ai/services/orchestrator` 新增
> `mgd_orchestrator.prompt_registry`（FR-038 部分）——从 `ai/prompts/*.md` 解析全部契约
> 元数据（prompt_id/version/layer/output_schema/safety_policy/status），四层组装
> （system/developer/session/data，data 层以 `<<<UNTRUSTED_DATA>>>` 边界包裹，
> 下层永不覆盖上层指令）、注入模式检测（中文/英文基线，命中即标记不执行）、输出 JSON Schema
> 校验（引用 `ai/schemas/*.json`，fail-closed，不通过不可进入房间）、版本固定
> （活跃正式会话固定开始版本，不匹配拒绝）。依赖 `jsonschema` 登记于 pyproject；
> pytest 9 用例（加载/版本/分层/注入/输出校验正常与拒绝）全绿，ruff/mypy(strict) 通过。
> （TASK-031、FR-038 部分、PROMPT-POLICY）

> **任务状态（2026-08-02 更新）**：TASK-032 已实现：`ai/services/orchestrator` 新增
> `mgd_orchestrator.interviewer_graph`（FR-012）——与 LangGraph StateGraph 语义对齐的
> 确定性迷你图引擎（add_node/add_conditional_edges/compile/invoke，零外部依赖，
> 生产可同构迁移 LangGraph）：覆盖点推进（按计划冻结范围顺序推进）、动态追问
> （weak/partial 回答在预算内追问，question_id 不越出已确认覆盖点，预算用尽后推进）、
> 打断策略（voice/button → avatar_stopped → 聆听，未播放内容不入证据）、工具使用
> （白名单外拒绝并终止请求，杜绝死循环）、检查点快照/恢复（重放安全，NFR-006）；
> 图节点只产出建议，不直接写业务状态（确定性状态由 Temporal 工作流控制）。
> pytest 15 用例（推进/追问边界/打断/工具白名单/恢复/非收敛保护）全绿，
> ruff/mypy(strict) 通过。（TASK-032、FR-012、NFR-006）

> **任务状态（2026-08-02 更新）**：TASK-033 已实现：计划生成链路（FR-009/FR-011，
> US-02 场景 5）——`ai/services/orchestrator` 新增 `mgd_orchestrator.plan_generator`
> （来源融合：可信来源引用/无来源回退通用模板并标记 AI 推导；轮次建议默认 3 轮、
> 1-5 轮与 10-60 分钟边界、六维权重和 100；PII/注入安全过滤与重生成 ≤2 次；
> 单轮失败只重试失败模块；输出对齐 interview-plan schema）；
> `services/project` 新增 `PlanGenerator` 接口与合成 `StubPlanGenerator`、`CheckPlanSafety`
> （PII/注入 fail-closed），`/v1/projects/{id}/plan:generate` 由 501 占位落地为
> 材料确认后返回 201 草稿（进入 PLAN_REVIEW，Frozen=false），RoundConfig 增加
> `question_coverage_plan` 结构。Go 服务/HTTP 正常、异常、幂等、安全过滤测试齐备；
> Python 22 用例全绿，ruff/mypy(strict) 通过。（TASK-033、FR-009、FR-011、PROMPT-POLICY）

> **任务状态（2026-08-03 更新）**：TASK-034 已实现（跨轮交接规则，US-02 规则 12、
> US-04 规则 8）——`ai/services/orchestrator` 新增 `mgd_orchestrator.handoff_generator`：
> 八类必备内容组装（简历/JD 快照引用、轮次纪要、评价、风险、已验证能力、未覆盖点、
> 禁止重复问题与允许重新验证例外）；上下文压缩按 HANDOFF-SPEC 第 6 节优先级
> （摘要 ≤120 字/≤80 词、追问链合并、强弱项去重、最新有效维度分），超预算不删除
> 简历/JD/未覆盖/禁止重复四类；事实完整性独立复核（no_new_facts/source_refs_complete，
> 生成器声明与复核不一致即拒绝）；敏感字段扫描（手机号/邮箱/证件/地址/保护属性）
> 命中即拒绝并告警；Schema 校验 fail-closed（handoff-package.schema.json，
> 补充 locked_carried 状态与 SCORING-SPEC 6.1 对齐）；语义去重执行层
> （repeats_previous_question / allowed_to_reverberify，例外仅
> direct_contradiction / new_job_scenario_transfer）。评测集 zh-core/en-core 新增
> handoff_compression / contradictory_evidence 用例并附预期结果；迁移
> `0034_handoff_packages.sql`（追加式、业务角色无 UPDATE/DELETE）；DOMAIN-MODEL、
> openapi（HandoffPackage schema）同步。pytest 13 用例全绿，ruff/mypy(strict) 通过。
> （TASK-034、HANDOFF-SPEC、FR-011）

> **任务状态（2026-08-03 更新）**：TASK-035 已实现（P0 注入风险；US-02 场景 5）——
> `ai/services/orchestrator` 新增 `mgd_orchestrator.safety_pipeline`：
> 以 `config/safety/policy.yaml` 为唯一事实源（policy_version safety/v1、八类
> prohibited_content 与 action、regeneration.max_attempts=3、injection_defense
> sanitize_and_log、audit.log_minimization）；简历/JD/网页/自由文本/工具输出一律视为
> 不可信数据；注入检测在 prompt_registry 基线之上补充编码混淆（%22/\\u0022/HTML 实体）、
> 工具诱导与中文“忽略……指令”宽模式，命中即中和指令并标记 injection_detected
> （内容仍按数据处理，不向用户暴露安全细节）；禁止内容分类动作与 policy.yaml 一一对应
> （歧视/侮辱/无关隐私/危险/骚扰/作弊协助/录用预测/PII 复述），阻断-重生成 ≤3 次、
> 危险/骚扰直接升级人工；评分证据保护属性零携带扫描（evidence_scan）；审计记录
> 最小化（不含敏感正文）。红队回归：zh-core/en-core prompt_injection /
> protected_attribute 用例 + fixtures/synthetic/jobs/jd-injection-zh.md 全部通过。
> pytest 23 用例全绿，ruff/mypy(strict) 通过。（TASK-035、PROMPT-POLICY、policy.yaml）

### EPIC-05 评分与复核

目标：独立、可重复、可解释、版本冻结的评分与正式复核。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-040 | 评分服务：六维锚点评分、双门槛、关键维度、取整、评估未完成 | FR-021 | TASK-026、TASK-031 | 与 `docs/ai/SCORING-SPEC.md` 伪代码逐条一致 |
| TASK-041 | 输入模式归一化：文字模式口语“未评估”不记零、混合模式合并 | 沟通维度规则 | TASK-040 | 文字模式报告标注证据限制 |
| TASK-042 | 岗位匹配度计算（必备/加分分列；无 JD 不展示；无简历不做经历一致性评分） | Job Match 规则 | TASK-040 | 匹配度不作为解锁隐藏因素 |
| TASK-043 | 正式复核服务：冻结证据+量表重算、新版本、前后对比 | FR-025 | TASK-040 | 每次正式尝试仅一次；禁止改写证据或权重 |
| TASK-044 | 量表/权重版本化与公平性监控（语言、口音、输入模式、便利设置切分） | FR-038 部分 | TASK-040 | 历史分数不因版本升级被修改 |
| TASK-045 | 评分稳定性回归：重复评分 95% 维度差 ≤3 分、及格结论一致率 ≥98% | 硬门槛 | TASK-040、TASK-036 | 回归报告达标才可进入 Beta |

> **任务状态（2026-08-03 更新）**：TASK-040 已实现（FR-021）——新 Go 模块
> `services/scoring`（仅依赖 region，登记 go.work 与 CI golangci 矩阵）：
> SCORING-SPEC 6.1-6.7 伪代码逐条实现。六维证据状态机（scored /
> insufficient_evidence / uncovered / not_applicable / locked_carried）；
> 锚点映射 1→20…5→100 与相邻锚点插值（必须引用锚点等级与证据 ID，缺失回退
> 下锚点，SC-EC-20）；覆盖率 ≥0.5 证据充分度；关键转写 unrecoverable →
> EVALUATION_INCOMPLETE(unrecoverable_transcript)；沟通维度 voice 公式
> （0.6×structure_clarity + 0.4×oral_delivery，half-up 取整）；双门槛
> round_total ≥60 且全部关键维度 ≥60（取整后比较，OD-07）；非关键弱项只记录、
> 非关键未覆盖按已评分维度归一化；正式重试锁定沿用/失败维度新分替换/矛盾解锁重评
> （6.7）；`idempotency_key` 幂等（NFR-006，重复提交不产生新版本）；
> 持久化故障/panic 降级 EVALUATION_INCOMPLETE(scoring_service_failure)
> 不判失败、不落库、恢复可重算（SC-EC-18）。HTTP：result/scores 分页落地，
> review 为 TASK-043 501 占位；scoring-input schema 增加 coverage_assessments
> 冻结判定结构。SC-EC-01~08/13/14/18/20/24 等边界案例回归 + 正常/异常/幂等/
> 故障恢复测试齐备（服务 20 用例 + HTTP 6 用例），gofmt/vet 通过。
> （TASK-040、FR-021、SCORING-SPEC、NFR-006）

> **任务状态（2026-08-03 更新）**：TASK-041 已实现（沟通维度规则，SCORING-SPEC 6.4）——
> `services/scoring` 输入模式归一化：voice 公式 0.6×structure_clarity + 0.4×oral_delivery；
> 文字模式 communication = structure_clarity，oral_delivery 标记 `not_evaluated`
> （不记 0、不扣分，SC-EC-09），报告标注输入模式与证据限制（input_mode_notes）；
> 混合模式按语音/文字有效证据占比合并（SC-EC-10：0.6×80 + 0.4×60 = 72），
> 语音占比 mixed_mode_voice_share 缺失即拒绝；摄像头开关与便利设置只记录、
> 不进入评分证据（SC-EC-11/12）。SC-EC-09/10/11/12 + 异常（缺语音占比）测试
> 全绿，gofmt/vet 通过。（TASK-041、SCORING-SPEC 6.4）

> **任务状态（2026-08-03 更新）**：TASK-042 已实现（Job Match 规则，SCORING-SPEC 6.8）——
> `services/scoring` 新增 `ComputeJobMatch`：必备/加分分列独立计算
> （match = Σ weight(已证明)/Σ weight(全部)，比例保留 4 位小数）；
> 已证明 = 简历证据（仅当存在简历）∪ 面试证据引用；无 JD →
> not_displayed_reason = no_jd 不展示百分比（SC-EC-22）；JD-only（无简历）→
> 只按面试证明计算、无简历证明（拒绝），且 experience_evidence 权重必须在
> 计划阶段重新分配为 0（评分侧 fail-closed，SC-EC-21）；匹配度与面试分数
> 相互独立、不作为单轮解锁隐藏因素。scoring-input schema 增加 job_match_input；
> openapi ScoreResult 增加 JobMatch/MatchBucket。正常（分列/加权/独立判定）、
> 异常（非法分列/重复/未注册证明/无简历证明/经历权重非 0）、幂等测试全绿。
> （TASK-042、SCORING-SPEC 6.8）

> **任务状态（2026-08-03 更新）**：TASK-043 已实现（FR-025）——`services/scoring`
> 扩展正式复核（SCORING-SPEC 6.10）：每次正式尝试仅一次自动复核（SC-EC-17，
> 二次请求 409 state_conflict）；复核使用与原始评分完全相同的冻结证据
> （evidence_snapshot_hash 校验，不一致拒绝并触发安全审计）、量表、权重与版本；
> 复核产出新 ScoreVersion（supersedes_score_id 指向原版本），返回原结果/新结果/
> 逐维前后对比与原因（SC-EC-16），全部版本保留、历史分数不可改写；冻结输入随
> ScoreVersion 追加保存（只增不改）；复核幂等键去重；`/v1/.../review` 由 501 落地为
> 202（ReviewAccepted 含 review_result），409/404/400 异常路径齐备。openapi 新增
> ReviewAccepted/ReviewResult 组件并重新生成 contracts/ts 产物；DOMAIN-MODEL §6.12
> 同步。服务/HTTP 正常、异常、幂等、二次拒绝、证据篡改测试齐备（新增 8 用例）。
> （TASK-043、FR-025、SCORING-SPEC 6.10）

> **任务状态（2026-08-03 更新）**：TASK-044 已实现（FR-038 部分）——
> `services/scoring` 新增 `RubricRegistry`（量表/权重版本化）：从
> `config/rubrics/v1/default.yaml` 加载（六维默认权重、锚点 1→20…5→100、
> evidence_sufficiency 阈值），版本唯一不可覆盖、未知版本 fail-closed；
> 冻结权重校验（总和 100、单维 ±5，0 权重重分配路径允许，SC-EC-19/21）；
> 活跃正式会话固定版本（PinnedCheck 不匹配即拒绝）；历史分数保留各自
> rubric_version，注册新版本后旧结果与幂等重放均原样返回（历史分数不因
> 版本升级被修改）。`FairnessMonitor` 公平性监控：按语言/口音/岗位族/
> 工作年限段/输入模式/便利设置切分聚合（计数、通过率、均分、维度均值），
> 快照确定性排序、标签最小化（不含用户内容）。评分服务注入默认注册表，
> 服务/注册表/监控测试齐备（新增 11 用例），gofmt/vet 通过。
> （TASK-044、FR-038、SCORING-SPEC 10.4）

> **任务状态（2026-08-03 更新）**：TASK-036 已实现——新 Python 包
> `ai/services/evals`（`mgd_evals`，ruff/mypy(strict)/pytest）：黄金集
> （ai/evals/datasets/*.jsonl）可重复运行并产出 JSON 报告
> （ai/evals/reports/*.eval.json）；数据集与预期结果对齐校验（缺条目即失败）；
> 内置评测器按场景自动分发（handoff：八类内容/压缩预算/事实完整性；
> safety：注入/保护属性对抗；generic：must_include/must_not_include/
> expected_flag），未支持场景如实标记 skipped；报告含 totals/metrics/
> thresholds/cases，确定性可复现；`validate_stability_report()` 提供
> TASK-045 稳定性门槛校验（95%/98%）握手。zh-core/en-core 实测 6/6 已支持
> 用例全过，报告已提交；pytest 7 用例全绿，ruff/mypy(strict) 通过。
> （TASK-036、PROMPT-POLICY 13、SCORING-SPEC 10）

> **任务状态（2026-08-03 更新）**：TASK-045 已实现（评分硬门槛）——
> `services/scoring` 新增 `RunStabilityRegression`：冻结输入基线 + 默认 200 次
> 受控微扰重复评分（锚点带内插值 ±1、4% 扰动率、固定种子可复现），输出六维
> 逐维维度差 ≤3 占比（取最差维度）与及格结论一致率；门槛
> 维度差 ≤3 比例 ≥95%、及格一致率 ≥98%；`cmd/stability` 生成
> `ai/evals/reports/stability.json`（report_kind=stability，含 config/metrics/
> thresholds/passed），由 TASK-036 `mgd_evals.validate_stability_report` 校验
> （Python 侧握手测试）。实测 200 次回归维度差 ≤3 比例 1.0、及格一致率 1.0，
> 报告提交入库；Go 测试 4 用例（门槛/确定性/异常/报告形状）+ evals 握手 2 用例
> 全绿，gofmt/vet 通过。（TASK-045、SCORING-SPEC 10、TASK-036）

> **任务状态（2026-08-03 更新）**：TASK-050 已实现（FR-023/FR-026，US-04 规则 3）——
> 新增 `docs/ai/REPORT-SPEC.md`（报告契约）；`mgd_orchestrator.report_generator`
> 确定性合成实现：由冻结 ScoreVersion/证据摘要/HandoffPackage/输入模式标记生成
> 七类报告模块（必备 overview/radar/rounds/training_plan），分数只读引用、
> 雷达图文字等价、逐题证据回溯、多轮轨迹、沟通/工具分析与训练计划；最终结果按
> SCORING-SPEC 6.9（任一 FAIL → 整体未通过；任一必需轮无有效分 →
> EVALUATION_INCOMPLETE + partial）；模块级失败重试（FR-026：失败模块
> status=failed、只重试失败模块 ≤2 次）；输出过 report.schema.json 校验
> （fail-closed）；训练用途声明与 deletion_entry 强制；保护属性零携带
> （safety evidence_scan 脱敏）。zh-core/en-core 新增 report_generation 评测用例，
> mgd_evals 增加 report_evaluator 并重新生成评测报告（zh 5/5、en 3/3 全过）。
> pytest 68 用例全绿（含报告 10 新用例），ruff/mypy(strict) 通过。
> （TASK-050、FR-023、FR-026、REPORT-SPEC）

> **任务状态（2026-08-03 更新）**：TASK-052 已实现（FR-024，US-04 场景 3）——
> `mgd_orchestrator.training_coach`（对齐 ai/prompts/training-coach.md 与
> SCREEN-SPEC SCR-12）：原题/变体/框架/示例练习项（变体只关联已考覆盖点，
> 未知覆盖点降级为维度级并标注非个性化；不泄露后续轮次考点）；逐步反馈
> 亮点 → 缺口 → 下一步（先优势后改进）；练习隔离红线 is_formal_evidence 恒 false、
> PracticeRecord 独立于正式证据链、永不产生 ScoreVersion/不改解锁；用户练习回答
> 注入按数据处理并说明隔离；录用预测/作弊/侮辱阻断重生成 ≤3、危险/骚扰升级人工；
> 简历/JD 原文不进入练习内容（保护属性零携带）。zh-core/en-core 新增
> practice_isolation 评测用例（en-practice-01 已有 + zh-practice-01 新增），
> mgd_evals 增加 coach_evaluator，评测报告重新生成（zh 6/6、en 4/4 全过）。
> pytest 78 用例全绿（含教练 10 新用例），ruff/mypy(strict) 通过。
> （TASK-052、FR-024、training-coach、SCR-12）

> **任务状态（2026-08-03 更新）**：TASK-053 已实现（FR-024，US-04 场景 4；
> SCORING-SPEC 6.7，DOMAIN-MODEL §6.14）——`services/scoring` 正式重试流：
> `BeginRetry`（仅 FAIL/EVALUATION_INCOMPLETE 可发起，PASS 拒绝；locked=上轮
> ≥60 维度、rescope=失败维度∪未覆盖点；状态机 RETRY_SCHEDULED→…→COMPLETED；
> 幂等）；`SelectRetryQuestions`（新题不重复已通过相同问题：规范化语义去重，
> direct_contradiction / new_job_scenario_transfer 例外允许主题重验但相同措辞
> 一律丢弃；全重复候选池拒绝）；`ScoreRetry`（新分替换失败维度旧分、锁定沿用、
> 矛盾解锁旧+新证据重评、重评维度必须携带新证据引用；历史版本保留不可改写；
> 未登记尝试拒绝=练习隔离）。HTTP `/v1/.../retry` 落地（201/409/404）；
> 迁移 `0053_retry_attempts.sql`。服务/HTTP 正常、异常、幂等、语义去重、
> 矛盾解锁端到端测试齐备（新增 7 用例），gofmt/vet 通过。
> （TASK-053、FR-024、SCORING-SPEC 6.7、DOMAIN-MODEL 6.14）

> **任务状态（2026-08-03 更新）**：TASK-054 已实现（FR-021/FR-022，SCR-10）——
> `services/scoring` 新增 `BuildRoundResultView` 轮次结果流：通过态固定祝贺文案
> （"恭喜你通过本轮面试，已进入下一轮"，与现有 result 前端语义一致）、总分/60
> 分线/关键维度通过情况/优势与注意点/进度/下一轮预告（角色/重点/难度/时长）；
> 失败态阻断下一轮、生成第 1 轮至当前轮累计纪要、提供复盘练习/正式重试/报告/
> 复核训练入口；评估未完成明确"这不是失败"，按原因（证据不足/系统故障/用户结束）
> 展示说明，系统责任（scoring_service_failure/system_fault）展示额度已返还；
> 红线：结果视图不携带后续轮次完整答案或考点内容（JSON 级断言）。测试 4 用例
> 全绿（通过/失败累计/未完成返还/不泄露），gofmt/vet 通过。
> （TASK-054、FR-021、FR-022、SCR-10）

> **任务状态（2026-08-03 更新）**：TASK-055 已实现（FR-040，US-05 场景 5；
> RETENTION-MATRIX）——新 Go 模块 `services/export`（登记 go.work 与 CI golangci
> 矩阵）：导出任务异步创建/执行/进度可查（queued→running→succeeded），导出物必带
> 训练用途标记"模拟训练结果，不代表真实企业录用结论"，account/project 范围、
> 项目导出必带 project_id、幂等；删除任务按 target_type（project/resume/job/
> account）级联编排，六层真实进度（database/cache/search_index/object_storage/
> backups/third_party_processors）逐项 pending/in_progress/done/failed，失败可
> 重试、不伪造完成；法定财务记录保留但解除内容关联；到期提醒 30/7 天窗口扫描。
> 迁移 `0055_export_deletion_tasks.sql`；DATA-MODEL 增加 export_tasks。服务/HTTP
> 正常、异常、幂等、失败重试、进度真实性测试齐备（新增 10 用例），gofmt/vet 通过。
> （TASK-055、FR-040、RETENTION-MATRIX）

### EPIC-06 报告与训练

目标：完整/部分报告、练习隔离、正式重试与维度锁定闭环。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-050 | 报告生成：雷达、岗位匹配、逐题证据、多轮轨迹、沟通与工具分析；模块级局部失败重试 | FR-023、FR-026、NFR-014 | TASK-040 | US-04 场景 6 通过 |
| TASK-051 | 报告文字等价版本、导出“模拟训练结果”标记 | 无障碍、AI 治理 | TASK-050 | 雷达图均有文字/表格等价；导出必带标记 |
| TASK-052 | AI 教练练习：原题/变体、提示、框架、逐步反馈；练习永不改分 | FR-024 | TASK-050 | US-04 场景 3 通过 |
| TASK-053 | 正式重试：新题、维度锁定、矛盾解锁重评、新旧证据替换 | FR-024 | TASK-040、TASK-052 | US-04 场景 4 通过；不重复已通过相同问题 |
| TASK-054 | 通过祝贺流、失败阻断与累计复盘、评估未完成展示 | FR-021、FR-022 | TASK-040 | US-04 场景 1、2 通过；不泄露后续完整答案 |
| TASK-055 | 数据导出与下载（到期提醒、删除入口） | 保留策略 | TASK-050 | 导出内容完整且带训练用途标记 |

### EPIC-07 商业（报价、额度、支付、退款）

目标：透明报价、会前预留、秒级账本、幂等支付与故障自动返还；付费永不影响评分。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-060 | 报价引擎与 Entitlement 模型（免费 60 分钟、单项目包、Pro、加油包） | FR-031 | TASK-016 | US-06 场景 1 通过；开始后计费版本冻结 |
| TASK-061 | 秒级 UsageLedger、每轮预留、故障/等待不计费、主动退出按实际扣减 | FR-032 | TASK-003 | 只计数字人已连接且正式进行中的秒数 |
| TASK-062 | 区域化支付集成、签名回调、重放防护、幂等订单 | FR-033 | TASK-060 | US-06 场景 4 通过；重复扣款自动识别退回 |
| TASK-063 | 退款、系统故障自动全额返还本轮预留、大额双人审批 | FR-033 | TASK-061 | US-06 场景 3 通过；账本记录原因 |
| TASK-064 | Pro 订阅：月度时长、最多结转一个账期、余额 ≤2 倍月额度、到期当前轮可完成 | FR-033 | TASK-062 | US-06 场景 5 通过 |
| TASK-065 | 发票/收据、税费展示、区域定价配置 | FR-033 | TASK-062 | 中国区合规发票；国际区税费明示 |

> **任务状态（2026-08-03 更新）**：TASK-060 已实现（FR-031，US-06 场景 1）——
> `services/billing` 首次业务实现：权益模型（免费 60 分钟幂等发放、单项目包、
> Pro 订阅月额度/结转 ≤1 账期/总余额 ≤2×月额度、时长加油包）与
> `Balance`/`CanReserve`（余额校验只在每轮开始前，已开始轮次不中断）；
> 报价引擎 Quote 状态机（DRAFT → PRESENTED → ACCEPTED，开始前计划修改 →
> RECALCULATED → PRESENTED 版本递增；接受后计费版本冻结 billing_freezes，
> 冻结后拒绝重新报价）；区域化合成定价（OD-02 未决前确定性可测）。
> 迁移 `0060_billing_entitlements.sql`；DATA-MODEL 增加 billing_freezes。
> 服务正常、异常、幂等、冻结、结转上限测试齐备（新增 7 用例），gofmt/vet 通过。
> （TASK-060、FR-031、BILLING-STATE-MACHINE）

> **任务状态（2026-08-03 更新）**：TASK-061 已实现（FR-032）——`services/billing`
> 秒级 UsageLedger：`Reserve`（每轮开始前预留，不足阻止开始，402；消费顺序
> 免费→项目包（限本项目）→Pro→加油包）、`StartMetering`/`StopMetering`（只计
> LIVE 秒；故障/等待/重连/认证暂停与降级后文字面试不计）、`Settle`（按实际扣减 +
> 冲正释放未使用预留；用户主动退出同规则）、`RefundFull`（系统责任自动全额返还
> 本轮预留）；账本追加式（reserve/consume/reversal），幂等键去重、逐笔可查。
> **room 挂接点闭环**：创建会话 → 预留+开始计量；EndSession → 按实际结算；
> 降级接受 → 故障点起停止计量；**降级拒绝 → 全额返还本轮预留**；暂停/恢复 →
> 停止/开始计量；余额不足 → ErrEntitlementMissing（HTTP 402 insufficient_
> entitlement）。服务（billing 6 用例 + room 5 用例）全绿，gofmt/vet 通过。
> （TASK-061、FR-032、BILLING-STATE-MACHINE §5.3/§6）

> **任务状态（2026-08-03 更新）**：TASK-062 已实现（FR-033，US-06 场景 4）——
> `services/billing` 区域化支付集成：Order 状态机（ORDER_CREATED → PAYMENT_PENDING →
> PAID；PAYMENT_FAILED 保留计划可重发起；PAYMENT_TIMEOUT 保持处理中），创建订单幂等
> 键去重；支付回调 HMAC-SHA256 验签 + ±5 分钟时间戳重放窗口 + payment_event_id 去重
> （同一事件重复回调无副作用）；支付成功只记一次权益与一次扣款；成功未到账保持
> PAYMENT_PENDING 由对账任务按 provider_txn_id 收敛（ReconcileOrder）；检测到重复扣款
> 自动原路退回（Refund REFUNDED + 账本冲正原因）+ 写 Incident；状态不明禁止重复发起
> 扣款（ErrPaymentPendingBlocked）。迁移 `0062_payment_orders.sql`（orders/payment_events/
> refunds/incidents）；openapi 回调补充 data_region，Order 增加 refunded_total。
> 服务正常、异常、幂等、重复回调、重复扣款、签名/重放、对账收敛测试齐备（新增 6 用例），
> gofmt/vet 通过。
> （TASK-062、FR-033、BILLING-STATE-MACHINE §5.2/§8/§9）

> **任务状态（2026-08-03 更新）**：TASK-063 已实现（FR-033，US-06 场景 3）——
> `services/billing` 退款与补偿：小额用户退款自动执行；大额（≥¥500 等值）与人工补偿
> 双人审批（同一审批人去重、不可自批、并发审批原子去重）；系统故障自动全额执行；
> 拒绝说明原因、用户可申诉；执行退款幂等（MarkRefundExecuted 原子），账本追加
> refund 冲正条目记录原因，订单累计 refunded_cents；退款不影响评分/复核/解锁。
> openapi 增加 Refund schema 与 approve/reject/appeal 端点；contracts 已重新生成。
> 服务正常、异常、幂等、并发双人审批、拒绝申诉测试齐备（新增 6 用例），gofmt/vet 通过。
> （TASK-063、FR-033、BILLING-STATE-MACHINE §5.4/§8）

> **任务状态（2026-08-03 更新）**：TASK-064 已实现（FR-033，US-06 场景 5）——
> `services/billing` Pro 订阅生命周期：自动续费单独勾选且必须明确同意（记录同意时
> 月额度与价格条款，条款变化须重新同意）；扣款前提醒（PrepareRenewal → reminded，
> 7 天窗口内 ChargeRenewal，超窗拒绝）；续费经 ActivatePro 保证结转 ≤1 账期、
> 总余额 ≤2×月额度；取消续费权益保留至账期结束；到期任务 ExpireDueSubscriptions
> → SUB_EXPIRED（历史保留、不影响导出/删除）；进行中的正式轮次到期仍可正常结束
> （余额校验只在轮次开始前）。迁移 `0063_pro_renewals.sql`（renewal_events +
> 订阅同意条款列）；openapi 增加 auto-renew/renewals/charge 端点与 RenewalRecord。
> 服务正常、异常、幂等、条款变化、提醒窗口、到期轮次完成测试齐备（新增 6 用例），
> gofmt/vet 通过。
> （TASK-064、FR-033、BILLING-STATE-MACHINE §5.5）

> **任务状态（2026-08-03 更新）**：TASK-065 已实现（FR-033，US-06 场景 11）——
> `services/billing` 发票与税费：区域定价配置（cn CNY/6% 增值税/合规发票；
> eu EUR/19% VAT/收据；intl USD/0% 明示税费/收据）；IssueInvoice 幂等开票
> （同一订单一份，价税合计与订单一致）；中国区发票含发票号码与增值税行、可作废
> （收据不可作废）；国际区税费明示收据。迁移 `0064_invoices.sql`；openapi 增加
> invoice/cancel/pricing 端点与 Invoice/TaxLine/RegionalPricing schema。
> 服务正常、异常、幂等、区域合规测试齐备（新增 5 用例），gofmt/vet 通过。
> （TASK-065、FR-033、BILLING-STATE-MACHINE §7）

> **任务状态（2026-08-03 更新）**：TASK-070 已实现（FR-034，US-07）——
> `services/org` 首次业务实现：机构租户（创建者以个人账户加入并成为所有者，
> 无影子账户）；六类角色权限分离矩阵（owner/admin/instructor/privacy_auditor/
> finance/candidate，财务/审计/教学/管理默认分离）；邀请适配点
> （link/org_email/bulk_list/sso/scim，14 天有效、幂等键去重、邮箱匹配校验）；
> 成员加入/列表/角色调整（owner 仅本人可变更）/退出（left_at 保留个人记录）/
> 机构停用/启用/注销；所有变更写追加式审计。迁移 `0065_org_tenants.sql`
> （organizations/org_members/org_invitations）；openapi 增加 accept/members/
> role/leave/status 端点与 OrgMember/Invitation schema。
> 服务正常、异常、幂等、角色分离、退出保留测试齐备（新增 6 用例），gofmt/vet 通过。
> （TASK-070、FR-034、DOMAIN-MODEL §6.16）

> **任务状态（2026-08-03 更新）**：TASK-071 已实现（FR-035，US-07 场景 5）——
> `services/org` 训练任务与模板：可配项（岗位/轮次/角色/时长/难度/语言/工具/
> 截止时间/练习次数/机构额度）白名单过滤；60 分线/统一评分算法/保护属性/证据标准/
> 跨轮解锁/正式复核为禁止项（大小写不敏感识别，拒绝并写审计）；
> 任务状态 draft → published → closed；完成情况默认最小可见（接受/未开始/进行中/
> 已完成/退出/系统故障/机构额度消耗，不暴露个人结果）；成员状态变更写审计。
> 迁移 `0066_assignments.sql`；openapi 增加 publish/close 端点。
> 服务正常、异常、幂等、禁止项审计、最小可见测试齐备（新增 4 用例），gofmt/vet 通过。
> （TASK-071、FR-035、DOMAIN-MODEL Assignment）

> **任务状态（2026-08-03 更新）**：TASK-072 已实现（FR-035，US-07 场景 1/2）——
> `services/org` 按任务细粒度结果授权：范围（total_score/radar/round_results/
> full_report/transcript/media 六类封闭枚举）+ 有效期 + 可撤回（幂等）；
> 机构侧 CheckShareEffective 校验（有效期内返回可见类别并写 AccessAudit，
> 撤回/到期立即失效）；ExpireShares 到期扫描；任务摘要"已完成未共享"仅计数
> （ResultShared/ResultNotShared，不显示失败）。迁移 `0067_assignment_shares.sql`；
> openapi 增加 shares 列表端点与 AssignmentShare schema。
> 服务正常、异常、幂等、撤回即时、到期失效、未共享计数测试齐备（新增 4 用例），
> gofmt/vet 通过。
> （TASK-072、FR-035、SCREEN-SPEC SCR-16）

> **任务状态（2026-08-03 更新）**：TASK-073 已实现（FR-036，US-07 场景 3）——
> `services/org` 聚合分析：按岗位类别/能力维度分组（overall 汇总），完成率、
> 维度均值、提升趋势（单人首末分数对比）；细分群体 <10 人 hidden 且不返回任何
> 指标；无个人排行榜/排名/候选人搜索（PersonalRankingAvailable 恒 false，
> 无个人粒度 API）；样本由评分服务注入（机构侧不持久化个人分数）。
> 服务正常、异常、小样本隐藏、类别过滤测试齐备（新增 4 用例），gofmt/vet 通过。
> （TASK-073、FR-036）

> **任务状态（2026-08-03 更新）**：TASK-074 已实现（FR-034/FR-035，US-07 场景 4）——
> `services/org` 机构访问审计与退出即时失效：ListAccessAudits（谁/何时/访问了什么，
> privacy_auditor/owner 可见，追加式 SELECT/INSERT）；退出/被移除 → left_at +
> 全部分享授权立即撤回（共享链接即时失效），个人记录保留、审计继续存在；
> 机构停用/注销 → InvalidateOrgAccess 撤回全部共享链接并写审计；
> IsMemberAccessValid 供令牌失效判定。openapi 增加 audits 列表与成员移除端点
> （OrgAuditEntry schema）。
> 服务正常、异常、退出即时、停用失效、审计可见性测试齐备（新增 4 用例），
> gofmt/vet 通过。
> （TASK-074、FR-034/FR-035、DOMAIN-MODEL §6.17）

### EPIC-08 机构（租户、任务、授权、聚合）

目标：机构可组织训练，默认不可见个人内容，永不演变为排名或筛选。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-070 | 机构租户、六类角色权限分离、邀请/机构邮箱/批量名单/SSO/SCIM | FR-034 | TASK-010 | 用户以个人账户加入，无影子账户 |
| TASK-071 | 训练任务与模板：可配项与禁止项（不得改 60 分线/量表/证据规则） | FR-035 | TASK-016 | US-07 场景 5 通过；违规操作写审计 |
| TASK-072 | 按任务细粒度结果授权（范围、有效期、可撤回）与“已完成未共享”展示 | FR-035 | TASK-070 | US-07 场景 1、2 通过 |
| TASK-073 | 聚合分析：维度/完成率/提升趋势；细分 <10 人隐藏；禁止个人排名 | FR-036 | TASK-072 | US-07 场景 3 通过 |
| TASK-074 | 机构访问审计（谁、何时、访问了什么）与退出即时失效 | FR-034、FR-035 | TASK-070 | US-07 场景 4 通过 |

### EPIC-09 治理（后台、版本、审计、数据权利）

目标：职责分离、默认脱敏、不可改分、全量审计的运营治理体系。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-080 | 运营后台：区域/房间/供应商/SLO 监控，默认匿名技术指标 | FR-037 | TASK-005 | US-08 场景 1 通过；运营不可旁听或代答 |
| TASK-081 | 模型/提示词/量表/工作流版本注册、离线-影子-灰度-放量、冻结与回滚 | FR-038 | TASK-030、TASK-031 | US-08 场景 2、6 通过 |
| TASK-082 | 禁止后台直接改分的系统级约束、正式复核唯一入口、破窗访问流程 | FR-039 | TASK-080 | US-08 场景 3 通过；破窗事后复核 |
| TASK-083 | 数据权利请求与删除编排（数据库/缓存/索引/对象存储/备份/第三方真实进度） | FR-040 | TASK-011 | US-05 场景 5 通过；级联删除可追踪 |
| TASK-084 | 追加式审计日志（管理员不可删除）、抗钓鱼 MFA、高风险再验证 | FR-037、FR-040 | TASK-080 | 审计写入无更新/删除路径 |
| TASK-085 | 客服工单：默认最小可见、用户授权逐字稿、双人审批媒体访问 | FR-039 | TASK-080 | US-08 场景 4 通过 |

> **任务状态（2026-08-03 更新）**：TASK-080 已实现（FR-037，US-08 场景 1）——
> `services/adminapi` 首次业务实现：运营后台区域/房间/供应商/SLO 监控
> （匿名会话编号 + 技术指标，无姓名/简历/回答/媒体）；供应商状态变更
> （active/ramping/disabled，停用必须记录原因并写审计）；运营不可加入/
> 旁听/代答（OperatorSessionGuard 一律拒绝并写审计）；后台角色/跨区访问
> 校验。迁移 `0068_admin_ops.sql`（ops_providers/ops_room_snapshots/
> ops_region_status）；openapi 增加 regions/rooms 端点与匿名 schema。
> 服务正常、异常、匿名、红线、角色跨区测试齐备（新增 4 用例），gofmt/vet 通过。
> （TASK-080、FR-037、SCREEN-SPEC SCR-17）

> **任务状态（2026-08-03 更新）**：TASK-081 已实现（FR-038，US-08 场景 2/6）——
> `services/adminapi` 版本治理：模型/提示词/量表/工作流版本注册
> （offline → shadow → canary → full 逐级推进）；灰度门槛=结构兼容+安全测试，
> 放量门槛=影子/灰度指标通过（未过门槛全量发布被阻止）；项目版本固定
> （活跃正式面试固定开始版本，不可中途改变）；回滚仅允许无进行中正式会话
> （新会话回稳定版）；量表停用需产品/面试专业/安全公平三方审批且不批量改写
> 历史分数。迁移 `0069_version_registry.sql`；openapi 增加 versions/promote/
> version-pin/rollback 端点。
> 服务正常、异常、门槛阻止、冻结、回滚门禁、三方审批测试齐备（新增 4 用例），
> gofmt/vet 通过。
> （TASK-081、FR-038、PROVIDER-ADAPTERS、TASK-031 pinned）

> **任务状态（2026-08-03 更新）**：TASK-082 已实现（FR-039，US-08 场景 3）——
> `services/adminapi` 禁止改分的系统级约束：AttemptScoreWrite 对编辑分数/解锁/
> 改证据一律拒绝并写审计（无任何分数修改存储路径，与前端 control-registry 红线
> 呼应）；正式复核为个体结果唯一入口（常量声明 + 无 score 写端点）；
> 破窗访问流程（限重大安全/法律事件、理由+时长 ≤8h、72h 内事后复核、开启者不可
> 自审、到期自动 expired、目标用户敏感访问记录可查、通知由上层触发）；
> break_glass 与审计存储仅 SELECT/INSERT（状态由追加式评审事件推导，
> 测试反射断言无 Update/Delete 修改路径）。迁移 `0070_break_glass.sql`；
> openapi 增加 break-glass 端点与 schema。
> 服务正常、异常、自审拒绝、超窗拒绝、到期、无修改路径测试齐备（新增 4 用例），
> gofmt/vet 通过。
> （TASK-082、FR-039、AGENTS.md §2）

> **任务状态（2026-08-03 更新）**：TASK-083 已实现（FR-040，US-05 场景 5）——
> `services/adminapi` 数据权利请求（复用 services/export 删除编排骨架）：
> delete/export/correct/withdraw 四类请求工单化（幂等键去重）；删除编排六层
> 真实进度（database/cache/search_index/object_storage/backups/third_party，
> 逐项 pending/in_progress/done/failed）；级联删除逐项可追踪（sessions/turns/
> evidence_items/score_versions/handoff_packages/reports/practices/
> retry_attempts/media）；法定财务记录保留但解除内容关联；失败如实 FAILED
> 且可重试。迁移 `0071_data_rights.sql`；openapi 增加 data-rights 端点与
> DataRightRequest/DataRightProgress schema。
> 服务正常、异常、幂等、失败重试、进度真实性测试齐备（新增 4 用例），gofmt/vet 通过。
> （TASK-083、FR-040、RETENTION-MATRIX）

### EPIC-10 上线验证（Phase 0–4 门槛）

目标：把 PRD 发布阶段退出条件转化为可执行验证任务；P0 风险未关闭禁止发布。

| 任务 ID | 任务 | 关联需求 | 依赖 | 验收要点 |
|---|---|---|---|---|
| TASK-090 | 验收矩阵全量映射与自动化分层落地 | 全部 US/FR/NFR | EPIC-02~09 | `docs/testing/ACCEPTANCE-MATRIX.md` 每项需求 ≥1 正常 +1 异常用例 |
| TASK-091 | WebRTC 混沌与故障演练（断网、抖动、供应商断连、重连、降级） | NFR-002、NFR-003、FR-020 | EPIC-03 | 故障不判失败、不丢证据、不计费 |
| TASK-092 | 2 倍预计峰值压测与单可用区/区域级容灾演练 | NFR-001~NFR-004 | EPIC-01、EPIC-03 | 单 AZ 故障 60s 接管；区域 RTO ≤30min |
| TASK-093 | 安全渗透与红队：注入、越权、跨租户、恶意文件、重放、重复扣费 | 安全控制 | EPIC-09 | 严重泄露/越权/重复扣费/证据丢失为 0 |
| TASK-094 | 中英文 WCAG 2.2 AA 自动化+人工+残障用户测试 | 无障碍 | EPIC-02~EPIC-06 | 全部核心流程通过 |
| TASK-095 | 评分硬门槛验证：稳定性、专家盲评一致率 ≥85%、维度 MAE ≤10、禁止属性进入证据为 0 | 硬门槛 | TASK-045 | 报告经 AI/评分负责人签字 |
| TASK-096 | 三数据区生产验证与 GA 放量守门（1%→5%→20%→50%→100%） | Phase 3/4 | TASK-090~095 | 单区故障不扩散；不违法跨境；放量可暂停回滚 |

## 6. 全局完成定义（Definition of Done）

任何任务标记完成前必须同时满足：

1. **追踪**：代码、配置、测试均引用本文件任务 ID 与 PRD 需求 ID；新增/变更行为已更新追踪关系。
2. **契约**：涉及 API / 事件 / Schema / 状态的变更与 `docs/api/`、`docs/domain/`、`ai/schemas/` 一致，且通过 CI 契约校验。
3. **测试**：对应验收矩阵用例通过；含至少一个异常路径；幂等与重试有测试。
4. **规则守门**：不改变 PRD 已确认规则（60 分门槛、评分算法、隐私边界、数据区隔离、禁止项）；不确定时停下来提问。
5. **安全隐私**：无密钥入库或入仓；日志不含正文/令牌/媒体；权限与数据区域经过检查；敏感操作写审计。
6. **文档同步**：用户可见行为变化同步更新相关规范文档与 CHANGELOG。
7. **可观测**：关键路径有指标、追踪与结构化日志（脱敏）。

## 7. 未决事项登记（Decision Register）

> 继承 PRD“Open Decisions”，并补充工程解释类未决项。每项包含负责人、所需证据与决策门槛；未关闭前相关实现按本文件与 ADR 的保守默认执行。
> **2026-08-01 更新：OD-06 经项目发起人批准（执行计划按阶段退出条件推进、不设主观日期）；OD-07 ~ OD-10 经需求发起人/产品决策人确认为最终规则；OD-01 ~ OD-05 维持 PRD 决策门槛继续未决。**

| 编号 | 事项 | 决策负责人 | 所需证据 | 决策门槛 | 状态 |
|---|---|---|---|---|---|
| OD-01 | WebRTC/数字人/ASR/TTS/LLM 主备供应商 | 技术负责人、AI 负责人、安全负责人 | 中英文质量、实时 SLO、数据区、授权、成本、故障切换对比 | Phase 0 退出前 | 未决 |
| OD-02 | 正式定价与各地区套餐 | 产品与商业负责人 | 单分钟完全成本、用户研究、税费、支付成本、价格实验 | 封闭 Beta 真实付费前 | 未决 |
| OD-03 | 三数据区最终法律实施方案 | 法务与隐私负责人 | 隐私/AI 影响评估、跨境机制、未成年人与消费者规则 | 各区域生产验证前 | 未决 |
| OD-04 | 商标、域名、社交账号与应用商店名称 | 品牌与法务负责人 | 中国与主要国际市场检索与申请策略 | 公开品牌发布前 | 未决 |
| OD-05 | 正式峰值容量基线 | SRE 与商业负责人 | 市场预测、Beta 并发、供应商配额与成本模型 | Phase 3 压测计划前 | 未决 |
| OD-06 | 团队、预算与日历排期 | 项目发起人与工程负责人 | 技术设计、工作分解、采购与合规计划 | 项目执行计划批准前 | **已批准 2026-08-01**（项目发起人/需求发起人）：批准按本计划（EPIC-01 ~ EPIC-10、阶段退出条件驱动、不设主观日期，符合 PRD Timeline 原则）推进；预算金额与采购细节按线下流程执行，不在仓内记录 |
| OD-07 | 评分取整规则（half-up 到整数后比较门槛） | AI/评分负责人 | 校准集上取整对一致率的影响分析 | EPIC-05 评审前 | **已确认 2026-08-01**（需求发起人/产品决策人）：按 half-up 取整后比较门槛执行 |
| OD-08 | 非关键维度未覆盖时的重新归一化计分口径 | AI/评分负责人、产品负责人 | 覆盖率统计与公平性切分影响 | EPIC-05 评审前 | **已确认 2026-08-01**（需求发起人/产品决策人）：关键维度证据不足=评估未完成；仅非关键维度未覆盖=按已评分维度归一化并标记进入重试范围；`min_coverage_ratio=0.5` 为规则下的可校准参数 |
| OD-09 | 区域代码枚举 `cn/eu/intl` 与区域路由细节 | 技术负责人、安全负责人 | 法律实施方案（OD-03）确认 | EPIC-01 评审前 | **已确认 2026-08-01**（需求发起人/产品决策人）：枚举 `cn/eu/intl` 生效；路由细节随 OD-03 落地 |
| OD-10 | 状态/事件英文命名规范 | 技术负责人 | 全部契约文档一致性检查 | 首个契约合并前 | **已确认 2026-08-01**（需求发起人/产品决策人）：项目状态大写蛇形、分析事件 snake_case、实时事件点分命名空间生效 |

## 8. 关键规则（不可协商摘录）

以下为 PRD 已确认规则的实现红线，任何任务不得违反：

1. 数字面试官必须始终实时输出视频与音频；用户摄像头/麦克风可关。
2. 通过条件：本轮加权总分 ≥60 且全部关键维度 ≥60；证据不足或系统故障 = 评估未完成，不判失败。
3. 文字模式口语标记“未评估”，不记 0 分。
4. 练习永不改变正式分数与解锁状态；正式重试用新题，维度 ≥60 锁定，矛盾则解锁重评。
5. 原始音视频默认不保存，须单独明确授权；禁止外貌/情绪/微表情/人格等保护属性评分。
6. 三数据区隔离，不为容灾擅自跨境；机构默认不可见个人内容。
7. 付费状态不改变评分、复核、隐私、无障碍与故障恢复规则。
8. 禁止实现 PRD Out of Scope 全部条目（抓取、聚合投递、ATS、作弊 Copilot、未授权克隆、PPT 课堂等）。

## 9. 异常处理总则

- 一切系统责任故障：不判失败、不丢证据、暂停计时、可恢复、故障额度自动返还。
- 所有对外写操作：幂等键 + 可重试 + 真实进度可查（删除、导出、退款、评分均为异步任务）。
- 所有第三方回调：签名验证 + 重放防护 + 幂等处理。
- 计划/解析/报告等部分失败：保留成功部分，只重试失败模块。

## 10. 验证方式

- 本计划的追踪完整性由 CI 脚本校验（US/FR/NFR 在本文件与验收矩阵中均有映射）。
- Epic 完成度按第 5 节验收要点 + 验收矩阵执行；EPIC-10 对应 PRD 阶段退出条件。
- 每次发布候选执行 `docs/testing/RELEASE-CHECKLIST.md`。
