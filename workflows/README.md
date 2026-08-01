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
