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
