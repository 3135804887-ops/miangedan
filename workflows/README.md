# workflows — Temporal 工作流定义

| 字段 | 内容 |
|---|---|
| 追踪 | TASK-001（目录骨架）；TASK-004（集群落地）；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节 |
| 职责边界 | Temporal 管业务工作流；AI 决策图（LangGraph）位于 `ai/services/orchestrator`（ADR-0001） |

## 规则

1. 任务队列按域划分：ingestion / plan / interview / scoring / report / billing / deletion。
2. Go 与 Python 两侧共用的 payload 结构以 `ai/schemas/` 与本目录契约为准，禁止双份手写漂移。
3. 工作流必须跨可用区故障可恢复；幂等键与重试策略遵循 AGENTS.md 第 3 节。
4. 首个业务工作流随 TASK-017（项目状态机）实现，并与 `docs/domain/INTERVIEW-STATE-MACHINE.md` 逐条一致。

## 命名空间与任务队列（TASK-004）

- 每区每环境命名空间：`mgd-{region}-{env}-temporal`（如 `mgd-cn-prod-temporal`），
  由 `services/temporal` 与 `infra/modules/temporal` 契约校验；区域间无共享命名空间（ADR-0005）。
- 集群跨 3 AZ 持久化，单 AZ 故障由 Temporal 恢复机制接管；工作流故障不判失败、不丢证据。

| 任务队列 | 工作流域 | 拥有服务 |
|---|---|---|
| `ingestion` | 材料解析 | ingestion |
| `plan` | 计划生成 | project |
| `interview` | 面试状态机 | project |
| `scoring` | 评分与复核 | scoring |
| `report` | 报告生成 | report |
| `billing` | 计费与补偿 | billing |
| `deletion` | 删除编排 | deletion |

队列集合校验：`temporal.ValidateTaskQueues`；新增队列必须先登记到
`infra/modules/temporal/module.yaml`，禁止业务代码私自造队列。
