# 变更记录（CHANGELOG）

本仓库全部显著变更记录于此文件。

## 格式约定

- 遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 结构与语义化版本。
- 每条条目归属以下类型之一：`Added`（新增）、`Changed`（变更）、`Deprecated`（弃用）、`Removed`（移除）、`Fixed`（修复）、`Security`（安全）。
- 每条涉及行为的条目必须引用追踪 ID：`PRD 需求 ID（US/FR/NFR）`、`TASK-ID`、`ADR-ID` 或 `OD-ID` 至少其一。
- 涉及 PRD 规则解释的变更，必须先更新需求追踪关系（IMPLEMENTATION_PLAN、验收矩阵、契约文档），再记录于此。
- 评分量表、面试流程、安全政策等配置变更必须注明配置版本号（如 `rubrics/v1`）。
- 日期使用 ISO 格式（YYYY-MM-DD）。最新版本在最上方。

## [Unreleased]

### Added

- TASK-010 用户、Identity 与多方式登录（`task/TASK-010-identity-login` 分支）：
  - 新增 `/v1/identity/*` OpenAPI 3.1 契约与 Go HTTP 适配层：邮箱验证码、Google/Apple/微信
    验证、首次注册、会话/刷新轮换、账户偏好和身份绑定；写操作强制幂等键，部署与令牌双重
    校验 `data_region`，错误响应不回显验证码、OAuth 授权码或令牌。（TASK-010、US-05、FR-027）
  - `services/identity` 新增供应商中立验证适配器、区域 `services/notify` 邮件投递、主体/验证码/
    证明摘要、业务 JWT、单次刷新令牌、串行事务存储与并发幂等执行；US-05 场景 4 要求当前侧与
    目标侧双重验证，身份冲突只追加恢复案件且绝不移动身份或合并账户。（TASK-010、SEC-002、SEC-012）
  - 新增 `0010_identity_accounts.sql`：用户、身份、验证、会话、冲突恢复与幂等表，区域/提供商/
    主体摘要唯一约束；邮箱、验证码、OAuth 授权码、证明和业务/刷新令牌均不落明文。同步领域、
    数据、安全、隐私、验收矩阵、实施计划与身份服务文档，并覆盖正常、异常、重试、并发幂等、
    跨区和零泄露测试。（TASK-010、US-05、FR-027、SEC-040）
- TASK-013 简历结构化解析与敏感字段硬隔离（`task/TASK-013-resume-parsing` 分支）：
  - 新增 `/v1/parsing/resumes*` OpenAPI 3.1 契约：accepted 原件解析/状态/步骤级重试、
    追加式版本查询、逐字段 add/replace/remove/confirm 与低置信度清零后最终确认。
  - `ai/services/parsing` 新增供应商中立 `ResumeParsingProvider`、L1/L2/L4 分层请求、确定性合成桩、
    Schema/暂时错误自动重试 ≤2 次及 NFR-015 原件保留重试；不绑定厂商 SDK。（TASK-013、FR-002）
  - 落实 SEC-040 四道门：解析前脱敏、模型输出后递归清洗、版本写入前 Schema/零命中断言、
    面试上下文与评分上游材料组装前再次 fail-closed；新增全类别合成样例与
    `resume-parsing-security` 评测集，敏感泄露命中为 0。（TASK-013、FR-003、SEC-040）
  - 新增 `0013_resume_parsing.sql` 的解析尝试/主记录/追加式版本、幂等、敏感根键与低置信度
    二次 CHECK；同步领域、数据、安全、隐私、验收、Provider/Prompt 契约与实施计划。
- TASK-012 简历隔离上传与恶意文件检测（`task/TASK-012-upload-scanning` 分支）：
  - 新增 `/v1/uploads/resumes`、`/v1/uploads/{uploadId}`、`/v1/uploads/{uploadId}:retry`
    OpenAPI 3.1 契约：具体拒绝码、沙箱证明、原件保留/可重试/不计费/不影响评分语义。
  - `services/ingestion` 新增上传服务、所属区域 uploads 桶专用对象存储接口、一次性无网络沙箱
    证明门槛、供应商中立恶意软件检测接口，以及 PDF/DOC/DOCX 病毒、宏、压缩炸弹、伪装、
    超限、损坏、加密矩阵；含正常、异常、并发幂等与超时重试测试。（TASK-012、FR-001、FR-006）
  - 新增 `0012_resume_uploads.sql`：上传/扫描尝试状态表、10 MiB 与区域 uploads 桶 CHECK、
    上传及扫描重试两级幂等唯一键；同步领域、数据、安全、威胁、验收与对象存储契约。
    （TASK-012、TM-02、SEC-020、NFR-006）
- TASK-015 企业公开流程来源服务（`task/TASK-015-enterprise-process-sources` 分支）：
  - 新增 `services/source` Go 控制面服务（单模块，登记 `go.work` 与 CI golangci 矩阵）：来源领域模型
    （链接/日期/类型/可信度/失效状态，候选人经验强制标记非官方）、官方优先排序与可靠性判定、
    供应商中立检索适配层契约 + 合成桩（TASK-030 未开工前不绑定厂商 SDK，PROVIDER-ADAPTERS §4.5）、
    可重试错误退避重试 ≤2 次、幂等键去重（NFR-006）；无公司/断网/无可信来源自动回退通用模板并
    标记 AI 推导（`flow_uses_generic_template`/`ai_derived`，可人工校对，不伪装企业流程）；
    内存幂等存储抽象；外部网页内容仅作为不可信数据进入结构化提取（SEC-024/SEC-025）。
    含正常/异常/幂等/并发/重试/注入按数据处理单测。（TASK-015、FR-007、FR-008、NFR-006）
  - 新增追加式迁移 `services/migrate/migrations/0002_process_sources.sql`（幂等键/URL 唯一、
    data_region 强制且与 region 一致、source_type/credibility/status CHECK；TASK-003 迁移工具）。
  - API 契约新增 `/v1/sources/*`（search/list/get）与 ProcessSource/SourceSearchResult 等 Schema，
    同步 DOMAIN-MODEL §6.6、DATA-MODEL §5.2、ACCEPTANCE-MATRIX FR-007/FR-008、
    IMPLEMENTATION_PLAN（TASK-015 状态）；`fixtures/synthetic/process-sources/` 扩展合成样例。
    （TASK-015、FR-007、FR-008、SEC-024、SEC-025）
- TASK-008 备份与恢复（`task/TASK-008-backup-recovery` 分支）：
  - 新增 `services/backup` 共享包与 CLI（yaml.v3）：备份契约 fail-closed（每日完整+WAL+PITR、
    证据 RPO=0、其他 RPO ≤5s、RTO ≤30 分钟、区域内备份桶、恢复前强制 tombstone 过滤）、
    一键恢复 dry-run 固定步骤序列；含正常/异常/幂等单测。（TASK-008、SEC-050、SEC-052）
  - 新增 `infra/modules/backup/` 模块契约与三区×三环境 `topology.backup` 配置，
    纳入 `regions` 套件校验（策略/PITR/RPO/RTO/tombstone）；`tools/validate_docs.py` 新增
    `backup` 套件（第 18 套件）并接入 CI 阶段 1、golangci 矩阵与本地检查。（TASK-008）
  - 新增 `docs/operations/RECOVERY-RUNBOOK.md`（一键恢复流程）与
    `tools/backup/quarterly-drill-template.md`（季度恢复演练模板，含 RPO/RTO 结果记录表）。
    **EPIC-01（TASK-001~008）全部完成。**（TASK-008）
- TASK-007 区域化通知与身份通道（`task/TASK-007-notification-identity-channels` 分支）：
  - 新增 `services/notify` 共享 Go 包：区域化邮件通道契约（`Config.Validate` fail-closed）、
    `Router` 按 `data_region` 路由（单区通道故障不影响他区，未配置区拒绝发送）、消息强制幂等键、
    模板变量禁止正文/令牌/媒体（PRIVACY-DATA-MAP）；含正常/异常/幂等/隔离单测。（TASK-007、FR-027）
  - 新增 `services/identity/provider` 身份提供商区域注册表（FR-027 开放矩阵：email 全区域、
    wechat 仅 cn、google/apple 仅 eu/intl），`ValidateProviders` fail-closed；含正常/异常/幂等单测。
  - 新增 `infra/modules/notification/` 模块契约与三区×三环境 `notification.channels`、
    `identity_providers` 配置，纳入 `regions` 套件校验；`tools/validate_docs.py` 新增
    `channels` 套件（第 17 套件）并接入 CI 阶段 1、golangci 矩阵与本地检查。（TASK-007）
- TASK-006 密钥管理系统接入与轮换契约（`task/TASK-006-secrets-key-mgmt` 分支）：
  - 新增 `services/secretref` 共享 Go 包（零外部依赖）：`*_REF` 引用命名契约（`ValidateRefName`）、
    区域 `secrets.refs` fail-closed 校验（值必须为合法环境变量名、不得内联真实密钥，SEC-012）、
    展示掩码 `MaskSecret`（只保留末 4 位）；含正常/异常/幂等单测。（TASK-006、SEC-012）
  - 新增 `infra/modules/secret-ref/` 模块契约（每区 KMS、ref_only、零明文）与三区×三环境 `kms_name`，
    纳入 `regions` 套件资源命名校验。（TASK-006）
  - `tools/validate_docs.py` 新增 `key-mgmt` 套件（第 16 套件：模块契约 + `tools/secret-rotation/rotation_drill.py`
    轮换演练）并接入 CI 阶段 1、golangci 矩阵与本地检查；新增
    `docs/operations/KEY-ROTATION-RUNBOOK.md` 轮换运行手册（五步流程 + 分类过渡方式）。（TASK-006、SEC-013）
- TASK-005 OpenTelemetry 观测与日志脱敏（`task/TASK-005-observability-otel` 分支）：
  - 新增 `services/observability` 共享 Go 包（OTel v1.44.0）：结构化 JSON 日志默认 strict 脱敏
    （敏感键整值替换 + JWT/Bearer/`sk-`/超长令牌值模式替换；生产强制 strict，SEC-032）、
    OTLP HTTP 指标/追踪 Provider 装配（资源含 `data_region`，导出器 none 时业务链路不受影响）、
    指标/追踪属性白名单校验（强制 `data_region`，白名单外/敏感键/疑似敏感值拒绝）；
    含正常/异常/幂等单测与合成敏感样本零泄露回归。（TASK-005、SEC-032）
  - 新增 `infra/modules/observability/` 模块契约与 README（每区 OTLP 采集端点、strict 脱敏、
    状态页骨架）；三区×三环境拓扑实例补充 `otel_collector`、`otlp_endpoint`、`redaction`、
    `status_page`，纳入 `regions` 套件校验。（TASK-005）
  - `tools/validate_docs.py` 新增 `observability` 套件（第 15 套件：模块契约、SDK 符号与测试、
    合成敏感样本、日志政策与状态页文档）并接入 CI 阶段 1、golangci 矩阵与本地检查。
    （TASK-005、SEC-032、SEC-033）
  - 新增 `docs/observability/LOGGING-POLICY.md`（日志与观测脱敏政策）与
    `docs/observability/STATUS-PAGE.md`（中英双语公开状态页骨架）；`.env.example` 增加
    SERVICE_NAME/SERVICE_VERSION/OTEL_* /REDACTION_MODE/STATUS_PAGE_BASE_URL；README、
    IMPLEMENTATION_PLAN、EPIC-01 设计、TEST-STRATEGY 同步。（TASK-005）
- TASK-004 Temporal 集群与每区命名空间、任务队列划分（`task/TASK-004-temporal-cluster-namespaces` 分支）：
  - 新增 `services/temporal` 共享 Go 包：`mgd-{region}-{env}-temporal` 命名空间生成与校验、
    七域任务队列（ingestion/plan/interview/scoring/report/billing/deletion）集合校验、
    `ValidateConfig` fail-closed；含正常/异常/幂等单测。（TASK-004、ADR-0001、ADR-0005）
  - 新增 `infra/modules/temporal/` 模块契约与 README：每区独立集群、跨 3 AZ、
    `history_retention_days=30`、命名空间模式与队列所有权；三区×三环境拓扑实例补充
    `cluster_name`、`cross_az: true`、`history_retention_days: 30`。（TASK-004）
  - `tools/validate_docs.py` 新增 `temporal` 套件（第 14 套件）并接入 CI 阶段 1 与本地检查；
    `regions` 套件扩展 Temporal 跨 AZ 与保留期校验；golangci 矩阵新增 `temporal`。
    （TASK-004）
  - `workflows/README.md` 登记命名空间模式与七域队列所有权表；README、
    IMPLEMENTATION_PLAN、EPIC-01 设计同步。（TASK-004）
- TASK-003 数据平台基线部署与迁移工具（`task/TASK-003-data-platform-migrations` 分支）：
  - 新增 `services/migrate` 迁移工具：`schema_migrations` + SHA-256 校验和，重复执行幂等；
    已应用迁移校验和变化即失败（fail-closed）；CLI 支持 `up` / `status`，并复用
    `DATA_REGION == INFRA_REGION` 启动自检。（TASK-003、NFR-005、NFR-006）
  - 基线迁移 `services/migrate/migrations/0001_ledger_baseline.sql`：四张追加式账本表
    （`evidence_items`、`score_versions`、`usage_ledger`、`access_audits`），含
    `data_region` CHECK、幂等键 UNIQUE、内容散列与数据库层 `REVOKE UPDATE, DELETE`
    （业务角色仅 SELECT/INSERT，删除编排专用角色保留受控 UPDATE/DELETE）。（TASK-003、ADR-0004）
  - 新增 `infra/modules/{database,object-storage,event-stream}/` 模块契约与 README；
    区域拓扑补充事件流六主题与媒体桶 30 天生命周期（`RETENTION-MATRIX`）。
    （TASK-003、NFR-005、RETENTION-MATRIX）
  - `tools/validate_docs.py` 新增 `data-platform` 套件（第 13 套件）：模块清单、
    迁移文件名、账本表/REVOKE/幂等键存在性、业务角色权限、迁移执行器与幂等测试，
    接入 CI 阶段 1 与本地检查；`regions` 套件新增事件流主题与媒体生命周期校验。
    （TASK-003、ADR-0004、NFR-005、NFR-006）
  - README、IMPLEMENTATION_PLAN、ACCEPTANCE-MATRIX（NFR-005/NFR-006 契约层级）、
    DATA-MODEL（迁移工具锚点）同步。（TASK-003）
- TASK-002 三数据区环境拓扑与区域路由（`task/TASK-002-three-region-topology-routing` 分支）：
  - `infra/regions/{cn,eu,intl}/envs/{dev,staging,production}.yaml`：9 个环境拓扑实例，覆盖网络（3 AZ）、
    PostgreSQL / Redis（非证据存储）、对象存储三桶、区域事件流、SFU、Temporal 命名空间与任务队列、
    密钥引用、区域化供应商白名单、邮件通道与区域路由配置；资源命名统一 `mgd-{region}-{env}-`，
    区域间无跨区引用。（TASK-002、NFR-004、ADR-0005、OD-09）
  - `tools/validate_docs.py` 新增 `regions` 套件（第 12 套件）：强制区域/环境代码、3 AZ、副本门槛、
    Redis 非证据、Temporal 队列、资源命名前缀与零跨区引用；CI 阶段 1 与本地检查接入。（TASK-002、NFR-004）
  - 新增 `services/region` 共享 Go 包：数据区枚举、fail-closed 启动自检
    （`DATA_REGION == INFRA_REGION`、`SERVICE_ENV` 合法）与区域路由决策（跨区拒绝并返回
    `region_mismatch`）；七个 Go 控制面服务与四个 Python AI 服务接入，全部含正常/异常/幂等单测。
    （TASK-002、SEC-051、ADR-0005）
  - CI 阶段 6 与本地脚本构建循环适配库模块：仅对含 main 包的模块写入临时产物，纯库模块
    （如 `services/region`）执行 `go build` 编译检查，避免 `go: no main packages to build`。
    （TASK-002）
  - `.env.example` 新增 `[REGION-SCOPED] INFRA_REGION`；README 与各服务 README 同步运行与自检说明。
    （TASK-002）
- TASK-001 单仓工程骨架（`task/TASK-001-monorepo-skeleton-ci` 分支）：
  - 目录结构按 `docs/architecture/EPIC-01-INFRA-DESIGN.md` 第 4.1 节落地：`apps/web`、`apps/admin`（占位）；`services/` 七个 Go 控制面服务模块（identity、consent、ingestion、project、billing、org、adminapi，经根 `go.work` 统一工作区）；`ai/services/` 四个 Python AI 服务包（parsing、orchestrator、scoring、report，src 布局 + pyproject）；`contracts/`、`workflows/`、`infra/modules/`、`infra/regions/{cn,eu,intl}/`（占位）。
  - 每个服务含 fail-closed `DATA_REGION` 启动自检最小形态（ADR-0005、OD-09）与正常 + 异常路径单测；与所连基础设施区域的一致性校验挂接 TASK-002。
  - 各目录 README 记录技术基线、拥有任务、追踪锚点与红线。（TASK-001、ADR-0005、OD-09）
- CI 门禁流水线 `.github/workflows/ci.yml`（PR 与 main 推送触发）：阶段1 规范校验（`tools/validate_docs.py`）、阶段2 静态检查（gofmt、go vet 逐模块、golangci-lint v2 逐模块矩阵含 gosec SAST 基线、ruff、mypy strict）、阶段3 单元测试（go test、pytest）、阶段4 契约校验（YAML/JSON/JSON Schema/OpenAPI）、阶段5 安全扫描（仓内密钥模式、gitleaks 全历史、govulncheck + pip-audit 依赖漏洞门禁、SBOM 工件；GitHub 原生 dependency-review-action 需 GHAS 许可，转公开或购 GHAS 后可换回）、阶段6 构建（go build、Python 包可安装性）；TS 检查、契约生成物 diff、Schema 样例、事件目录一致性、集成测试、评测回归与 dev 部署留有挂接注释，随对应任务接入。（TASK-001；NFR-005、NFR-007 间接）
- 协作基线：`.gitattributes`（仓内统一 LF）、`.golangci.yml`（lint + gosec 基线）、`.github/pull_request_template.md`（DoD 自检清单）、`tools/ci/run_local_checks.sh`（本地一键复现 CI 阶段 1~4、6 及阶段 5 本地等价项）、`tools/ci/README.md`、`tools/requirements-dev.txt`（统一 Python 工具链）。（TASK-001）
- `tools/validate_docs.py` 新增 `--suites` 参数：11 个校验套件可按逗号键选（默认全部），支撑 CI 分阶段门禁。（TASK-001）
- `docs/testing/SPEC-REVIEW-CHECKLIST.md`：十角色规范评审检查单与签字表，驱动规范评审闭环。
- `docs/architecture/EPIC-01-INFRA-DESIGN.md`：EPIC-01 详细实施设计（单仓结构、CI 门禁、三数据区拓扑、TASK-003~008 实施要点）。（TASK-001 ~ TASK-008、NFR-001 ~ NFR-006）
- `docs/testing/PHASE0-PROVIDER-EVALUATION.md`：Phase 0 供应商评测方案（准入门槛、评分卡、指标矩阵、故障演练、决策产出），驱动 OD-01 关闭。（OD-01、TASK-030、TASK-096）
- git 仓库初始化与基线提交（main 分支）；`.gitignore`（机密、构建产物、用户内容入库禁令）。

### Changed

- `README.md`：仓库状态由"研发规范阶段"更新为 TASK-001 骨架已落地；"本地开发预期入口"改写为真实可执行命令（本地一键检查、Go/Python 骨架运行示例）；当前状态新增 TASK-001 完成项。（TASK-001）
- ruff 配置（`ai/services/*/pyproject.toml`）：按 AGENTS.md 中文书写约定关闭 RUF001/RUF002/RUF003（全角中文标点非 Unicode 混淆字符，启用为系统性误报）；其余规则集（含 S 安全规则）保持全量启用。（TASK-001）
- 规范评审签字完成（2026-08-01，需求发起人/产品决策人）：十角色评审检查单全部结论"通过"，由需求发起人/产品决策人代表全部角色签署（如实记录于 `docs/testing/SPEC-REVIEW-CHECKLIST.md` 第 6 节，含代表签字说明与专职角色复核义务）。按检查单第 7 节规则，全部规范文档版本标记由"草案，待评审"提升为"已批准 2026-08-01 规范评审"，配置与提示词契约 `status: draft_for_review` → `approved`（涉及 `config/rubrics/v1/default.yaml`、`config/interview-flows/v1/default.yaml`、`config/safety/policy.yaml`、`config/feature-flags.yaml`、`ai/prompts/` 六份契约及 26 份规范文档；PRD、枚举示例与规则文本保持不变）。
- 决策批准（2026-08-01，项目发起人/需求发起人）：OD-06 排期预算——批准按 `IMPLEMENTATION_PLAN.md`（EPIC-01 ~ EPIC-10、阶段退出条件驱动、不设主观日期）推进；预算金额与采购细节按线下流程执行、不入仓。OD-01 ~ OD-05 维持未决。同步更新：`IMPLEMENTATION_PLAN.md` 第 7 节、`README.md`、`docs/architecture/adr/README.md`。（OD-06）
- 决策确认（2026-08-01，需求发起人/产品决策人）：OD-07 取整规则（half-up 取整后比较门槛）、OD-08 非关键维度未覆盖归一化口径（`min_coverage_ratio=0.5` 保留为可校准参数）、OD-09 区域代码 `cn/eu/intl`、OD-10 状态/事件英文命名规范，由"未决"转为"已确认"；OD-01 ~ OD-06 维持 PRD 决策门槛继续未决。同步更新：`IMPLEMENTATION_PLAN.md` 第 7 节（新增状态列）、`docs/ai/SCORING-SPEC.md`、`config/rubrics/v1/default.yaml`、`docs/architecture/adr/README.md`、`docs/architecture/adr/ADR-0005-three-data-regions.md`、`README.md`。（OD-07 ~ OD-10）

## [0.1.0] - 2026-08-01

### Added

- 初始研发规范套件，覆盖 PRD-001 V1.0 全部需求（US-01~US-08、FR-001~FR-040、NFR-001~NFR-016）：
  - 根文件：`AGENTS.md`、`README.md`、`IMPLEMENTATION_PLAN.md`、`.env.example`。
  - 架构：`docs/architecture/SYSTEM-ARCHITECTURE.md`、`DEPLOYMENT.md`、ADR-0001~ADR-0005。
  - 领域：`docs/domain/DOMAIN-MODEL.md`、`INTERVIEW-STATE-MACHINE.md`、`BILLING-STATE-MACHINE.md`。
  - API 与事件：`docs/api/openapi.yaml`、`docs/api/realtime-events.md`。
  - AI 与评分：`docs/ai/`（AI-ORCHESTRATION、SCORING-SPEC、HANDOFF-SPEC、PROMPT-POLICY、PROVIDER-ADAPTERS）。
  - 数据与安全：`docs/data/`（DATA-MODEL、RETENTION-MATRIX、DATA-CLASSIFICATION）、`docs/security/`（THREAT-MODEL、PRIVACY-DATA-MAP、SECURITY-REQUIREMENTS）。
  - 设计与无障碍：`docs/design/`（SCREEN-SPEC、DESIGN-SYSTEM、ACCESSIBILITY）。
  - 测试与发布：`docs/testing/`（ACCEPTANCE-MATRIX、TEST-STRATEGY、RELEASE-CHECKLIST）。
  - 配置契约：`config/rubrics/v1/default.yaml`、`config/interview-flows/v1/default.yaml`、`config/safety/policy.yaml`、`config/feature-flags.yaml`。
  - AI 资产：`ai/prompts/` 提示词契约、`ai/schemas/` 8 份 JSON Schema、`ai/evals/` 合成评测数据集与预期结果。
  - 合成测试材料：`fixtures/synthetic/`（虚构简历、JD、逐字稿、来源样例、故障事件）。
- 文档与契约校验工具 `tools/validate_docs.py`（开发工具，非产品源代码）：必交文件、YAML/JSON、OpenAPI、JSON Schema、Mermaid、占位符、需求覆盖、跨文件一致性与密钥扫描。
- 未决事项登记 OD-01 ~ OD-10（IMPLEMENTATION_PLAN.md 第 7 节）。
