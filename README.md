# 面个蛋（MianGeDan）

> **面个蛋——多面几轮，少慌一点。**
> **MianGeDan — Real-time AI Mock Interviews.**

面个蛋是一款由用户简历和目标岗位 JD 驱动、使用实时虚拟数字人完成多轮模拟面试、逐轮审核、证据化评分、复盘训练与正式重试的求职能力训练平台。

本仓库当前处于**研发规范阶段**：包含完整的产品需求、架构决策、领域模型、API/事件契约、AI 与评分规范、数据安全规范、测试与发布规范、配置契约和合成测试材料。**尚未包含正式产品源代码。**

## 文档导航

| 类别 | 位置 | 内容 |
|---|---|---|
| 产品需求（事实源） | [docs/prd/PRD-001-面个蛋-V1.0.md](docs/prd/PRD-001-面个蛋-V1.0.md) | US-01~08、FR-001~040、NFR-001~016、评分模型、发布阶段 |
| 工作规则 | [AGENTS.md](AGENTS.md) | AI 开发代理规则、禁令、DoD |
| 实施计划 | [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) | EPIC-01~10、任务分解、需求追踪、未决事项 |
| 变更记录 | [CHANGELOG.md](CHANGELOG.md) | 版本历史与变更格式 |
| 架构 | [docs/architecture/](docs/architecture/) | 系统架构、部署、ADR |
| 领域设计 | [docs/domain/](docs/domain/) | 领域模型、面试状态机、计费状态机 |
| API 与事件 | [docs/api/](docs/api/) | OpenAPI 3.1、实时事件契约 |
| AI 与评分 | [docs/ai/](docs/ai/) | 编排、评分规范、跨轮交接、提示词政策、供应商适配 |
| 数据与安全 | [docs/data/](docs/data/)、[docs/security/](docs/security/) | 数据模型、保留矩阵、分类、威胁模型、隐私地图、安全需求 |
| 设计与无障碍 | [docs/design/](docs/design/) | 页面规范、设计系统、WCAG 2.2 AA |
| 测试与发布 | [docs/testing/](docs/testing/) | 验收矩阵、测试策略、发布检查单 |
| 配置契约 | [config/](config/) | 评分量表、面试流程、安全政策、功能开关 |
| AI 资产 | [ai/](ai/) | 提示词契约、JSON Schema、评测数据集与预期结果 |
| 合成测试材料 | [fixtures/synthetic/](fixtures/synthetic/) | 虚构简历、JD、逐字稿、来源样例、故障事件 |

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

## 本地开发预期入口

当前仓库为规范与契约阶段，**尚无应用启动命令**。本地工具链的预期用途：

- 文档与契约校验（CI 同源）：`python tools/validate_docs.py` —— 必交文件、YAML/JSON 解析、OpenAPI 校验、JSON Schema 元校验、Mermaid 代码块、占位符、需求覆盖（US/FR/NFR）、跨文件一致性、配置语义与密钥扫描。
- AI 评测材料位于 `ai/evals/`，供后续评分与提示词回归使用。
- 当首个服务骨架（EPIC-01 / TASK-001）落地后，本小节将补充真实启动命令；在此之前不要编造命令。

## 当前状态与后续步骤

- [x] PRD V1.0 需求基线确认（2026-08-01）
- [x] 研发规范、契约与合成测试材料（v0.1.0）
- [ ] 各规范文档正式评审（工程、AI/评分、安全/隐私、法务、设计、无障碍）
- [ ] 未决事项 OD-01 ~ OD-06 关闭（OD-07 ~ OD-10 已于 2026-08-01 确认，见 IMPLEMENTATION_PLAN.md 第 7 节）
- [ ] EPIC-01 基础设施与数据区落地（建议首个实施的 Epic）

## 贡献

任何开发工作前请阅读 [AGENTS.md](AGENTS.md)。所有变更必须维护与 PRD 的需求追踪关系；评分证据、审计与账本为追加式记录，禁止直接修改历史。
