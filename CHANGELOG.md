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

- git 仓库初始化与基线提交（main 分支）；`.gitignore`（机密、构建产物、用户内容入库禁令）。
- `docs/testing/PHASE0-PROVIDER-EVALUATION.md`：Phase 0 供应商评测方案（准入门槛、评分卡、指标矩阵、故障演练、决策产出），驱动 OD-01 关闭。（OD-01、TASK-030、TASK-096）
- `docs/architecture/EPIC-01-INFRA-DESIGN.md`：EPIC-01 详细实施设计（单仓结构、CI 门禁、三数据区拓扑、TASK-003~008 实施要点）。（TASK-001 ~ TASK-008、NFR-001 ~ NFR-006）
- `docs/testing/SPEC-REVIEW-CHECKLIST.md`：十角色规范评审检查单与签字表，驱动规范评审闭环。

### Changed

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
