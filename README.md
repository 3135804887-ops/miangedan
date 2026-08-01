# 面个蛋（MianGeDan）

> **面个蛋——多面几轮，少慌一点。**
> **MianGeDan — Real-time AI Mock Interviews.**

面个蛋是一款由用户简历和目标岗位 JD 驱动、使用实时虚拟数字人完成多轮模拟面试、逐轮审核、证据化评分、复盘训练与正式重试的求职能力训练平台。

本仓库包含完整的产品需求、架构决策、领域模型、API/事件契约、AI 与评分规范、数据安全规范、测试与发布规范、配置契约和合成测试材料；EPIC-01 首批工程产物（TASK-001 单仓骨架与 CI 门禁）已落地，各服务业务实现随 Epic 任务推进。

## 文档导航

| 类别 | 位置 | 内容 |
|---|---|---|
| 产品需求（事实源） | [docs/prd/PRD-001-面个蛋-V1.0.md](docs/prd/PRD-001-面个蛋-V1.0.md) | US-01~08、FR-001~040、NFR-001~016、评分模型、发布阶段 |
| 工作规则 | [AGENTS.md](AGENTS.md) | AI 开发代理规则、禁令、DoD |
| 实施计划 | [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) | EPIC-01~10、任务分解、需求追踪、未决事项 |
| 变更记录 | [CHANGELOG.md](CHANGELOG.md) | 版本历史与变更格式 |
| 架构 | [docs/architecture/](docs/architecture/) | 系统架构、部署、EPIC-01 详细设计、ADR |
| 观测与运营 | [docs/observability/](docs/observability/) | 日志与观测脱敏政策、公开状态页骨架（中英双语） |
| 领域设计 | [docs/domain/](docs/domain/) | 领域模型、面试状态机、计费状态机 |
| API 与事件 | [docs/api/](docs/api/) | OpenAPI 3.1、实时事件契约 |
| AI 与评分 | [docs/ai/](docs/ai/) | 编排、评分规范、跨轮交接、提示词政策、供应商适配 |
| 数据与安全 | [docs/data/](docs/data/)、[docs/security/](docs/security/) | 数据模型、保留矩阵、分类、威胁模型、隐私地图、安全需求 |
| 运维手册 | [docs/operations/](docs/operations/) | 密钥轮换运行手册、恢复运行手册（季度演练模板见 tools/backup/） |
| 设计与无障碍 | [docs/design/](docs/design/) | 页面规范、设计系统、WCAG 2.2 AA |
| 测试与发布 | [docs/testing/](docs/testing/) | 验收矩阵、测试策略、发布检查单、Phase 0 供应商评测、规范评审检查单 |
| 配置契约 | [config/](config/) | 评分量表、面试流程、安全政策、功能开关 |
| AI 资产 | [ai/](ai/) | 提示词契约、JSON Schema、评测数据集与预期结果 |
| 合成测试材料 | [fixtures/synthetic/](fixtures/synthetic/) | 虚构简历、JD、逐字稿、来源样例、故障事件、日志脱敏敏感样本 |

## 技术栈概览（PRD 基线）

| 层 | 基线 |
|---|---|
| Web/PWA | Next.js、React、TypeScript（桌面优先响应式） |
| 实时媒体 | WebRTC/SFU（LiveKit 为技术基线，可自托管或云部署） |
| 核心后端 | Go（账户、项目、计费、权限、控制面） |
| AI 服务 | Python（解析、LangGraph、模型网关、评分、报告、评测） |
| 持久工作流 | Temporal（业务长流程） |
| AI 决策图 | LangGraph（面试官内部决策） |
| 存储 | PostgreSQL / Redis（非证据）/ S3 兼容对象存储 / 区域事件流 |
| 观测 | OpenTelemetry（内容默认脱敏） |

最终商业供应商不锁定，须经 Phase 0 评测（见 `IMPLEMENTATION_PLAN.md` 未决事项 OD-01）。

## 本地开发入口

- 一键本地检查（与 CI 阶段 1~4、6 同源）：`bash tools/ci/run_local_checks.sh`；Python 工具链先经 `pip install -r tools/requirements-dev.txt` 安装（说明见 `tools/ci/README.md`）。
- 文档与契约校验（CI 同源）：`python tools/validate_docs.py` —— 必交文件、YAML/JSON 解析、OpenAPI 校验、JSON Schema 元校验、Mermaid 代码块、占位符、需求覆盖（US/FR/NFR）、跨文件一致性、配置语义、三数据区拓扑（`regions`）、数据平台契约（`data-platform`）与密钥扫描；`--suites` 支持按套件运行。
- Go 服务骨架运行示例（fail-closed 区域自检）：`cd services/identity && DATA_REGION=cn INFRA_REGION=cn SERVICE_ENV=dev go run ./cmd/identity`；`DATA_REGION` 与所连基础设施区域 `INFRA_REGION` 不一致、缺失或不在 `cn/eu/intl` 内必须拒绝启动（ADR-0005、OD-09，TASK-002）。
- 数据平台迁移工具：`cd services/migrate && DATA_REGION=cn INFRA_REGION=cn SERVICE_ENV=dev DATABASE_URL=postgres://... go run ./cmd/migrate up|status`；幂等执行并由 `schema_migrations` + SHA-256 校验和保证可重复（TASK-003）。
- Python AI 服务单测示例：`cd ai/services/scoring && pytest`（src 布局，可 `pip install -e .` 本地安装）。
- AI 评测材料位于 `ai/evals/`，供后续评分与提示词回归使用。
- 前端 `apps/web`、`apps/admin` 目前为目录占位，首个前端任务落地后补充真实启动命令；在此之前不要编造命令。

## 当前状态与后续步骤

- [x] PRD V1.0 需求基线确认（2026-08-01）
- [x] 研发规范、契约与合成测试材料（v0.1.0）
- [x] git 基线提交（2026-08-01，main 分支）
- [x] 项目执行计划批准（2026-08-01，OD-06，按阶段退出条件推进、不设主观日期）
- [x] 各规范文档正式评审（2026-08-01，十角色检查单全部通过，由需求发起人/产品决策人代表签署；规范状态已从 draft_for_review 提升为 approved）——记录见 `docs/testing/SPEC-REVIEW-CHECKLIST.md` 第 6 节
- [x] TASK-001 单仓工程骨架与 CI 门禁（2026-08-01，`task/TASK-001-monorepo-skeleton-ci` 分支；本地检查全绿，见 `tools/ci/README.md`）
- [x] TASK-002 三数据区环境拓扑与区域路由（2026-08-01，`task/TASK-002-three-region-topology-routing` 分支；9 份 `infra/regions/*/envs/*.yaml`、`regions` 校验套件、`services/region` 共享包）
- [x] TASK-003 数据平台基线部署与迁移工具（2026-08-01，`task/TASK-003-data-platform-migrations` 分支；`services/migrate` 幂等迁移、四张追加式账本表基线、`data-platform` 校验套件与数据平台模块契约）
- [x] TASK-004 Temporal 集群与每区命名空间、任务队列划分（2026-08-01，`task/TASK-004-temporal-cluster-namespaces` 分支；`services/temporal` 契约包、`infra/modules/temporal`、`temporal` 校验套件）
- [ ] 未决事项 OD-01 ~ OD-05 关闭（OD-06 ~ OD-10 已于 2026-08-01 确认/批准，见 IMPLEMENTATION_PLAN.md 第 7 节）
- [ ] EPIC-01 基础设施与数据区落地（TASK-001~004 已完成；下一任务 TASK-005 观测，实施设计见 `docs/architecture/EPIC-01-INFRA-DESIGN.md`）
- [ ] Phase 0 供应商评测（OD-01，方案见 `docs/testing/PHASE0-PROVIDER-EVALUATION.md`）

## 贡献

任何开发工作前请阅读 [AGENTS.md](AGENTS.md)。所有变更必须维护与 PRD 的需求追踪关系；评分证据、审计与账本为追加式记录，禁止直接修改历史。
